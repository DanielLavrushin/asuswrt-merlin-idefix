start() {
    log_info "Starting $ADDON_TITLE..."
    local pid=$(get_proc "idefix-server")

    if [ -n "$pid" ]; then
        log_error "$ADDON_TITLE is already running with PID: $pid"
        return 1
    fi

    $ADDON_SERVER &
    local pid=$!
    echo $pid >/var/run/$ADDON_TAG.pid

    log_ok "$ADDON_TITLE started with PID: $pid"
}

stop() {
    log_info "Stopping $ADDON_TITLE..."
    if ! get_proc "$ADDON_TAG" >/dev/null; then
        log_error "$ADDON_TITLE is not running."
        return 1
    fi

    # Stop the process
    killall idefix-server 2>/dev/null && log_ok "$ADDON_TITLE stopped." || log_error "$ADDON_TITLE failed to stop."
    rm -f /var/run/$ADDON_TAG.pid

}

restart() {
    log_info "Restarting $ADDON_TITLE..."
    stop
    sleep 1
    start
}

startup() {
    log_info "Starting $ADDON_TITLE on startup..."

    remount_ui

    log_ok "$ADDON_TITLE started with PID: $pid"
}
