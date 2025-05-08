#!/bin/sh

import ./_globals.sh
import ./_helper.sh

import ./mount.sh

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
esac

exit 0
