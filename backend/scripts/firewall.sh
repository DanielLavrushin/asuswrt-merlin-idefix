#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

# The terminal port hands out a root shell. It must be reachable from the LAN
# and from the router's own VPN tunnels, and from nowhere else — above all not
# from the WAN.
#
# The ruleset this builds, top of INPUT downwards:
#
#   1. DROP  on every WAN interface          (explicit, in case a WAN interface
#                                             name would match an allow pattern)
#   2. ACCEPT on lo / LAN bridges / tunnels
#   3. DROP  everything else                 (catch-all: unknown or future
#                                             interfaces are refused, not allowed)
#
# The catch-all sits at the bottom of our own block, so the port is closed by
# default and opened only for the interfaces named below. `+` is the iptables
# wildcard, not a shell glob.
#
#   lo    loopback
#   br+   LAN bridge and guest-network bridges (br0, br1, …)
#   tun+  OpenVPN server and client tunnels
#   tap+  OpenVPN bridged tunnels
#   wg+   WireGuard server and client tunnels
#
# IDEFIX_ALLOW_IFACES overrides the list for anyone running the terminal over
# another private overlay (Tailscale, ZeroTier). The same variable is honoured
# by the server binary. Never add a WAN interface to it.
FIREWALL_DEFAULT_IFACES="lo br+ tun+ tap+ wg+"

firewall_allowed_ifaces() {
    local ifaces="${IDEFIX_ALLOW_IFACES:-$FIREWALL_DEFAULT_IFACES}"

    ifaces="$(printf '%s' "$ifaces" | tr ',' ' ')"
    local valid=1
    local tok

    for tok in $ifaces; do
        case "$tok" in
        '+' | *[!A-Za-z0-9._+-]*)
            valid=0
            break
            ;;
        esac
    done

    # An empty or all-whitespace value leaves the loop with nothing to reject.
    [ -n "$(printf '%s' "$ifaces" | tr -d ' ')" ] || valid=0

    if [ "$valid" -ne 1 ]; then
        log_warn "Ignoring malformed IDEFIX_ALLOW_IFACES, using defaults."
        ifaces="$FIREWALL_DEFAULT_IFACES"
    fi

    printf '%s' "$ifaces"
}

# Every interface that faces the Internet, as the router currently understands
# it. Dual-WAN and PPPoE links each carry their own name, and nvram can lag
# behind a redial, so ppp+ is always included.
firewall_wan_ifaces() {
    local lan
    lan="$(nvram get lan_ifname 2>/dev/null)"

    local seen=""
    local key val
    for key in wan_ifname wan0_ifname wan1_ifname wan0_pppoe_ifname wan1_pppoe_ifname wan0_gw_ifname wan1_gw_ifname; do
        val="$(nvram get "$key" 2>/dev/null)"

        case "$val" in
        '' | *[!A-Za-z0-9._-]*) continue ;;
        esac

        # In AP, repeater and media-bridge modes the WAN name can be the LAN
        # bridge itself. Dropping that would make the terminal unreachable on
        # the only interface it has, and there is no Internet-facing port to
        # protect in those modes anyway.
        [ "$val" = "$lan" ] && continue
        case "$val" in br*) continue ;; esac

        case " $seen " in *" $val "*) continue ;; esac
        seen="$seen $val"
    done

    printf '%s ppp+' "$seen"
}

# Deleting by rule number rather than by spec: it removes whatever is on the
# port regardless of how it was written, including the unrestricted
# `-I INPUT -p tcp --dport 8787 -j ACCEPT` shipped in 1.4.4/1.4.5. Deleting by
# spec would leave that rule in place on upgrade, which is the whole bug.
firewall_flush_table() {
    local ipt="$1"
    local port="$2"

    [ -n "$ipt" ] || return 0

    local nums n
    # Highest number first, so each delete leaves the remaining numbers valid.
    nums="$("$ipt" -L INPUT -n --line-numbers 2>/dev/null |
        awk -v p="$port" '$0 ~ ("dpt:" p "($|[^0-9])") { print $1 }' |
        sort -rn)"

    for n in $nums; do
        case "$n" in
        '' | *[!0-9]*) continue ;;
        esac
        "$ipt" -D INPUT "$n" 2>/dev/null
    done
}

# The addon's PATH puts Entware (/opt/sbin) ahead of the firmware directories,
# so a bare `iptables` can resolve to an Entware build compiled against another
# kernel, which cannot drive the firmware's netfilter. Always prefer the
# firmware binary — it is the one the router's own rules are written with.
firewall_resolve_bin() {
    local name="$1"
    local p

    for p in "/usr/sbin/$name" "/sbin/$name" "/usr/bin/$name" "/bin/$name"; do
        if [ -x "$p" ]; then
            printf '%s' "$p"
            return 0
        fi
    done

    command -v "$name" 2>/dev/null || true
}

# 0 = usable, 2 = not available on this router. The binary being present says
# nothing: ip6tables ships on firmware with IPv6 turned off, where opening the
# filter table fails outright. That has to be told apart from a rule we failed
# to install, which is a hole and must stop the addon.
firewall_table_usable() {
    local ipt="$1"

    [ -n "$ipt" ] || return 2

    # Keep the reason: this decision can stop the addon from starting, and
    # "unavailable" with no explanation is not something a user can act on.
    local err
    if err="$("$ipt" -L INPUT -n 2>&1 >/dev/null)"; then
        return 0
    fi

    log_warn "$ipt cannot be used: ${err:-unknown error}"
    return 2
}

firewall_apply_table() {
    local ipt="$1"
    local port="$2"

    firewall_table_usable "$ipt" || return 2

    # Inserting at an explicit, increasing position keeps the block in the
    # order written above; a bare -I would reverse it and put the catch-all
    # DROP on top of its own ACCEPTs.
    local pos=1
    local ifn

    for ifn in $(firewall_wan_ifaces); do
        if ! "$ipt" -I INPUT "$pos" -i "$ifn" -p tcp --dport "$port" -j DROP 2>/dev/null; then
            log_error "Failed to install the WAN DROP for port $port on $ifn ($ipt)."
            return 1
        fi
        pos=$((pos + 1))
    done

    for ifn in $(firewall_allowed_ifaces); do
        if "$ipt" -I INPUT "$pos" -i "$ifn" -p tcp --dport "$port" -j ACCEPT 2>/dev/null; then
            pos=$((pos + 1))
        fi
    done

    if ! "$ipt" -I INPUT "$pos" -p tcp --dport "$port" -j DROP 2>/dev/null; then
        log_error "Failed to install the catch-all DROP for port $port ($ipt)."
        return 1
    fi
}

firewall_add_rules() {
    log_info "Adding firewall rules for $ADDON_TITLE..."

    local port="$ADDON_SERVER_PORT"
    local ipt ip6t
    ipt="$(firewall_resolve_bin iptables)"
    ip6t="$(firewall_resolve_bin ip6tables)"

    # Clear first: rules are rebuilt from scratch every time, so the block
    # cannot end up duplicated or half-ordered after a restart or an upgrade.
    firewall_clear_rules

    firewall_apply_table "$ipt" "$port"
    case "$?" in
    0) ;;
    2)
        log_error "iptables is unavailable (${ipt:-not found}); cannot restrict port $port."
        return 1
        ;;
    *)
        log_error "Failed to install the IPv4 rules for port $port."
        firewall_clear_rules
        return 1
        ;;
    esac

    # The server listens dual-stack, so an IPv6-enabled router would answer on
    # the WAN over IPv6 with the IPv4 rules alone.
    firewall_apply_table "$ip6t" "$port"
    case "$?" in
    0) ;;
    2)
        # No IPv6 netfilter means no IPv6 traffic to filter. The server's own
        # interface check still refuses anything arriving on a WAN address.
        log_info "IPv6 firewall unavailable; skipping the IPv6 rules for port $port."
        ;;
    *)
        log_error "Failed to install the IPv6 rules for port $port."
        firewall_clear_rules
        return 1
        ;;
    esac

    log_ok "Port $port restricted to: $(firewall_allowed_ifaces)"
    return 0
}

firewall_clear_rules() {
    log_info "Clearing firewall rules for $ADDON_TITLE…"

    local port="$ADDON_SERVER_PORT"

    firewall_flush_table "$(firewall_resolve_bin iptables)" "$port"
    firewall_flush_table "$(firewall_resolve_bin ip6tables)" "$port"

    return 0
}
