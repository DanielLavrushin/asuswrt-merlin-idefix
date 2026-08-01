#!/bin/sh
# Exercises firewall.sh under busybox ash, the shell it runs under on the
# router, against a stub iptables that keeps a real ordered rule list.
#
#   busybox sh backend/scripts/tests/firewall_test.sh
#
# The point of these tests is the one property that matters: the terminal port
# must never end up reachable from the WAN.
set -u

SRC=${SRC:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

V4="$WORK/v4"
V6="$WORK/v6"
: >"$V4"
: >"$V6"

ADDON_TITLE="Idefix"
ADDON_DEBUG="false"
ADDON_SERVER_PORT=8787
LOG_FACILITY='user'
USE_COLOR=0
CDBG='' CERR='' CWARN='' CINFO='' CSUC='' CLOG='' CRESET=''

logger() { :; }

. "$SRC/_helper.sh"
. "$SRC/firewall.sh"

# --- stub router -------------------------------------------------------------

NV_LAN="br0"
NV_WAN="eth0"
NV_WAN0="eth0"

nvram() {
    case "$2" in
    lan_ifname) printf '%s' "$NV_LAN" ;;
    wan_ifname) printf '%s' "$NV_WAN" ;;
    wan0_ifname) printf '%s' "$NV_WAN0" ;;
    *) printf '' ;;
    esac
}

# A stub iptables that behaves like the real one for the three operations
# firewall.sh uses: numbered insert, numbered delete, and a numbered listing.
# Rules are stored one per line, in chain order.
_ipt() {
    store="$1"
    shift

    case "${1:-}" in
    -L)
        # iptables -L INPUT -n --line-numbers → "<num> <rule>", and the real
        # thing renders --dport N as dpt:N, which is what the parser matches.
        awk '{ line=$0; gsub(/--dport ([0-9]+)/, "", line); printf "%d %s dpt:%s\n", NR, line, dport($0) }
             function dport(s,   a) { if (match(s, /--dport [0-9]+/)) { a=substr(s, RSTART+8, RLENGTH-8); return a } return "none" }' "$store"
        return 0
        ;;
    -D)
        # iptables -D INPUT <num>
        n="$3"
        awk -v d="$n" 'NR != d' "$store" >"$store.tmp" && mv "$store.tmp" "$store"
        return 0
        ;;
    -I)
        # iptables -I INPUT <pos> <spec...>
        shift 2
        pos="$1"
        shift
        spec="$*"
        awk -v p="$pos" -v s="$spec" '
            { lines[NR] = $0 }
            END {
                n = NR
                if (p > n + 1) p = n + 1
                if (p < 1) p = 1
                for (i = 1; i < p; i++) print lines[i]
                print s
                for (i = p; i <= n; i++) print lines[i]
            }' "$store" >"$store.tmp" && mv "$store.tmp" "$store"
        return 0
        ;;
    esac
    return 1
}

# V4_OK/V6_OK model a table that cannot be opened at all — ip6tables on
# firmware with IPv6 disabled answers that way for every operation.
# V6_INSERT_OK models a table that lists fine but refuses to take a rule.
V4_OK=1
V6_OK=1
V6_INSERT_OK=1

iptables() {
    [ "$V4_OK" = 1 ] || return 1
    _ipt "$V4" "$@"
}

ip6tables() {
    [ "$V6_OK" = 1 ] || return 1
    if [ "$V6_INSERT_OK" != 1 ] && [ "${1:-}" = "-I" ]; then
        return 1
    fi
    _ipt "$V6" "$@"
}

# --- harness -----------------------------------------------------------------

fail=0
check() {
    if [ "$2" = "$3" ]; then
        printf 'ok   %s\n' "$1"
    else
        printf 'FAIL %s\n       got:  %s\n       want: %s\n' "$1" "$2" "$3"
        fail=1
    fi
}

# Line number of the first rule matching a pattern, or 0.
lineno() {
    n=$(grep -n -- "$2" "$1" 2>/dev/null | head -n1 | cut -d: -f1)
    printf '%s' "${n:-0}"
}

# grep -c prints its count and *exits non-zero* on no match, so the count has
# to be captured rather than defaulted with ||.
count() {
    n=$(grep -c -- "$2" "$1" 2>/dev/null)
    printf '%s' "${n:-0}"
}

# An ACCEPT with no -i is an ACCEPT from anywhere, WAN included. This is the
# exact shape of the 1.4.4 regression, and the assertion that must never break.
unrestricted_accepts() {
    awk '/-j ACCEPT/ && !/-i /' "$1" | wc -l | tr -d ' '
}

reset_store() {
    : >"$V4"
    : >"$V6"
}

# --- a clean install ---------------------------------------------------------

reset_store
firewall_add_rules >/dev/null 2>&1
check "add_rules succeeds" "$?" "0"

check "no ACCEPT without an interface (v4)" "$(unrestricted_accepts "$V4")" "0"
check "no ACCEPT without an interface (v6)" "$(unrestricted_accepts "$V6")" "0"

check "loopback accepted" "$(count "$V4" '\-i lo -p tcp --dport 8787 -j ACCEPT')" "1"
check "LAN bridges accepted" "$(count "$V4" '\-i br+ -p tcp --dport 8787 -j ACCEPT')" "1"
check "OpenVPN tunnels accepted" "$(count "$V4" '\-i tun+ -p tcp --dport 8787 -j ACCEPT')" "1"
check "WireGuard tunnels accepted" "$(count "$V4" '\-i wg+ -p tcp --dport 8787 -j ACCEPT')" "1"

check "WAN interface dropped" "$(count "$V4" '\-i eth0 -p tcp --dport 8787 -j DROP')" "1"
check "PPPoE WAN dropped" "$(count "$V4" '\-i ppp+ -p tcp --dport 8787 -j DROP')" "1"
check "catch-all DROP present" "$(count "$V4" '^-p tcp --dport 8787 -j DROP$')" "1"

# Ordering is the whole ruleset: a catch-all above its own ACCEPTs closes the
# port to the LAN, and ACCEPTs above the WAN DROP would open it to the world.
wan_drop=$(lineno "$V4" '\-i eth0 .* -j DROP')
first_accept=$(lineno "$V4" '\-j ACCEPT')
catch_all=$(lineno "$V4" '^-p tcp --dport 8787 -j DROP$')
check "WAN DROP sits above the ACCEPTs" "$([ "$wan_drop" -lt "$first_accept" ] && echo yes)" "yes"
check "catch-all DROP sits below the ACCEPTs" "$([ "$catch_all" -gt "$first_accept" ] && echo yes)" "yes"
check "our block starts at the top of INPUT" "$wan_drop" "1"

check "IPv6 is covered too" "$(count "$V6" '^-p tcp --dport 8787 -j DROP$')" "1"

# --- upgrading from the vulnerable 1.4.4/1.4.5 ruleset -----------------------

reset_store
printf '%s\n' '-p tcp --dport 8787 -j ACCEPT' >"$V4"
printf '%s\n' '-i br0 -p tcp --dport 80 -j ACCEPT' >>"$V4"
firewall_add_rules >/dev/null 2>&1
check "the 1.4.4 blanket ACCEPT is removed on upgrade" "$(unrestricted_accepts "$V4")" "0"
check "unrelated rules on other ports survive" "$(count "$V4" '\-\-dport 80 -j ACCEPT')" "1"

# --- reapplying (firewall-start fires on every firewall rebuild) -------------

reset_store
firewall_add_rules >/dev/null 2>&1
once=$(wc -l <"$V4" | tr -d ' ')
firewall_add_rules >/dev/null 2>&1
firewall_add_rules >/dev/null 2>&1
check "rules are not duplicated when reapplied" "$(wc -l <"$V4" | tr -d ' ')" "$once"

# --- clearing ----------------------------------------------------------------

reset_store
printf '%s\n' '-i br0 -p tcp --dport 80 -j ACCEPT' >"$V4"
firewall_add_rules >/dev/null 2>&1
firewall_clear_rules >/dev/null 2>&1
check "clear removes every rule on our port" "$(count "$V4" '\-\-dport 8787')" "0"
check "clear leaves other ports alone" "$(count "$V4" '\-\-dport 80')" "1"

# --- AP / media-bridge mode, where the WAN name is the LAN bridge ------------

reset_store
NV_WAN="br0"
NV_WAN0="br0"
firewall_add_rules >/dev/null 2>&1
check "the LAN bridge is never dropped as a WAN interface" "$(count "$V4" '\-i br0 .* -j DROP')" "0"
check "the LAN is still reachable in AP mode" "$(count "$V4" '\-i br+ -p tcp --dport 8787 -j ACCEPT')" "1"
NV_WAN="eth0"
NV_WAN0="eth0"

# --- a hostile IDEFIX_ALLOW_IFACES -------------------------------------------

reset_store
IDEFIX_ALLOW_IFACES='br0 -j ACCEPT; eth0'
firewall_add_rules >/dev/null 2>&1
check "a malformed interface override falls back to the defaults" \
    "$(count "$V4" '\-i eth0 -p tcp --dport 8787 -j ACCEPT')" "0"
check "the port is still closed by default after a bad override" \
    "$(count "$V4" '^-p tcp --dport 8787 -j DROP$')" "1"
unset IDEFIX_ALLOW_IFACES

# --- a legitimate override ---------------------------------------------------

reset_store
IDEFIX_ALLOW_IFACES='lo br+ tailscale0'
firewall_add_rules >/dev/null 2>&1
check "a valid override is honoured" "$(count "$V4" '\-i tailscale0 -p tcp --dport 8787 -j ACCEPT')" "1"
check "an override still cannot open the port to everything" "$(unrestricted_accepts "$V4")" "0"
unset IDEFIX_ALLOW_IFACES

# --- IPv6 disabled, which is the firmware default ----------------------------

reset_store
V6_OK=0
firewall_add_rules >/dev/null 2>&1
check "a router without IPv6 netfilter still starts" "$?" "0"
check "the IPv4 rules are applied anyway" "$(count "$V4" '^-p tcp --dport 8787 -j DROP$')" "1"
V6_OK=1

# --- a firewall that answers but refuses the rules ---------------------------
# Half-applied rules are worse than none: the caller must fail so the server is
# never started behind them.

reset_store
V6_INSERT_OK=0
firewall_add_rules >/dev/null 2>&1
check "a table that refuses rules is a hard failure" "$?" "1"
check "no half-applied IPv4 rules are left behind" "$(count "$V4" '\-\-dport 8787')" "0"
V6_INSERT_OK=1

reset_store
V4_OK=0
firewall_add_rules >/dev/null 2>&1
check "missing iptables is a hard failure" "$?" "1"
V4_OK=1

exit "$fail"
