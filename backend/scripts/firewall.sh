#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

firewall_add_rules() {
    log_info "Adding firewall rules for $ADDON_TITLE..."

    local ifn="$(nvram get lan_ifname)"
    local port="$ADDON_SERVER_PORT"

    iptables -C INPUT -p tcp --dport $port ! -i lo -j DROP 2>/dev/null || iptables -I INPUT -p tcp --dport $port ! -i lo -j DROP

    # Make sure LAN traffic is accepted
    iptables -C INPUT -i $ifn -p tcp --dport $port -j ACCEPT 2>/dev/null || iptables -I INPUT -i $ifn -p tcp --dport $port -j ACCEPT

    # Accept from loopback
    iptables -C INPUT -i lo -p tcp --dport $port -j ACCEPT 2>/dev/null || iptables -I INPUT -i lo -p tcp --dport $port -j ACCEPT

}

firewall_clear_rules() {
    log_info "Clearing firewall rules for $ADDON_TITLE…"

    local ifn="$(nvram get lan_ifname)"
    local port="$ADDON_SERVER_PORT"

    iptables -D INPUT -i $ifn -p tcp --dport $port -j ACCEPT 2>/dev/null
    iptables -D INPUT -i lo -p tcp --dport $port -j ACCEPT 2>/dev/null
    iptables -D INPUT ! -i lo -p tcp --dport $port -j DROP 2>/dev/null
}
