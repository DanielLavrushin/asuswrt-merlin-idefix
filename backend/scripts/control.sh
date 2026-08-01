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

    if [ -z "$TZ" ]; then
        if [ -r /etc/TZ ]; then
            TZ=$(cat /etc/TZ)
        elif command -v nvram >/dev/null 2>&1; then
            TZ=$(nvram get time_zone_dst)
            [ -z "$TZ" ] && TZ=$(nvram get time_zone)
        fi
    fi
    export TZ
    $ADDON_SERVER &
    local pid=$!
    echo $pid >/var/run/$ADDON_TAG.pid

    firewall_add_rules
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
    firewall_add_rules
    generate_secret
    remount_ui
    start  
}