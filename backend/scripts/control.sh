#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

start() {
    log_info "Starting $ADDON_TITLE..."
    local pid=$(get_proc "idefix-server")

    cleanup_stale_asdfiles

    if [ -n "$pid" ]; then
        log_error "$ADDON_TITLE is already running with PID: $pid"
        return 1
    fi

    # time_zone_x is what the firmware itself writes to /etc/TZ: the POSIX
    # timezone plus the daylight-saving rules. time_zone_dst is only a 0/1
    # flag, and time_zone an Asus-internal code — neither is a timezone.
    if [ -z "$TZ" ]; then
        if [ -r /etc/TZ ]; then
            TZ=$(cat /etc/TZ)
        elif command -v nvram >/dev/null 2>&1; then
            TZ=$(nvram get time_zone_x)
        fi
    fi
    export TZ

    # Before the listener exists, not after: starting first leaves the port
    # answering on every interface — the WAN included — until the rules land.
    if ! firewall_add_rules; then
        log_error "$ADDON_TITLE not started: the port could not be restricted to the LAN."
        return 1
    fi

    $ADDON_SERVER &
    local pid=$!
    echo $pid >/var/run/$ADDON_TAG.pid

    log_ok "$ADDON_TITLE started with PID: $pid"
}

stop() {
    log_info "Stopping $ADDON_TITLE..."

    if ! get_proc "idefix-server" >/dev/null; then
        log_info "$ADDON_TITLE is not running."
        rm -f /var/run/$ADDON_TAG.pid
        firewall_clear_rules
        return 0
    fi

    # Stop the process and wait for it to actually exit, otherwise a later
    # start() sees a live PID and refuses to launch
    killall idefix-server 2>/dev/null

    if ! wait_proc_gone "idefix-server" 10; then
        log_warn "$ADDON_TITLE did not exit, forcing..."
        killall -9 idefix-server 2>/dev/null
        if ! wait_proc_gone "idefix-server" 5; then
            log_error "$ADDON_TITLE could not be stopped."
            return 1
        fi
    fi

    log_ok "$ADDON_TITLE stopped."
    rm -f /var/run/$ADDON_TAG.pid

    firewall_clear_rules

    # explicit: firewall_clear_rules ends on an `iptables -D` that fails
    # whenever the rule is already absent, and callers test stop's status
    return 0
}

restart() {
    log_info "Restarting $ADDON_TITLE..."
    if ! stop; then
        log_error "Restart aborted: $ADDON_TITLE is still running."
        return 1
    fi
    start
}

startup() {
    log_info "Starting $ADDON_TITLE on startup..."
    generate_secret
    remount_ui
    start
}

# Entry point for /jffs/scripts/firewall-start. The router rebuilds its INPUT
# chain on every WAN reconnect, VPN state change and firewall restart, which
# drops our rules with it. Without this hook the terminal simply stops being
# reachable from the LAN until the next reboot — and any rule ordering we rely
# on is gone.
firewall() {
    log_info "Reapplying firewall rules for $ADDON_TITLE after a firewall restart..."

    if ! firewall_add_rules; then
        log_error "Could not reapply firewall rules; stopping $ADDON_TITLE to keep the port closed."
        stop
        return 1
    fi
}