#!/bin/sh

import ./_globals.sh
import ./_helper.sh

import ./mount.sh
import ./control.sh
import ./install.sh
import ./lock.sh
import ./token.sh
import ./firewall.sh

case "$1" in
mount_ui)
    mount_ui
    ;;
unmount_ui)
    unmount_ui
    ;;
remount_ui)
    remount_ui $2
    ;;
start)
    start
    ;;
stop)
    stop
    ;;
restart)
    restart
    ;;
startup)
    startup
    ;;
install)
    install
    ;;
uninstall)
    uninstall
    ;;
update)
    update
    ;;
service_event)
    case "$2" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    generate)
        case "$3" in
        token)
            generate_token "$4"
            ;;
        esac
        ;;
    *)
        log_error "Unknown service event: $2"
        ;;
    esac
    ;;
esac

exit 0
