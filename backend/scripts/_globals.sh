#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

export PATH="/opt/bin:/opt/sbin:/sbin:/bin:/usr/sbin:/usr/bin"

source /usr/sbin/helper.sh

ADDON_TAG="idefix"
ADDON_TAG_UPPER="IDEFIX"
ADDON_TITLE="Idefix Terminal"

XRAYUI_VERSION="1.0.0"

ADDON_WEB_DIR="/www/user/$ADDON_TAG"

ADDON_SCRIPT="/jffs/scripts/$ADDON_TAG"
ADDON_SHARE_DIR="/opt/share/$ADDON_TAG"
ADDON_LOGS_DIR="$ADDON_SHARE_DIR/logs"

ADDON_DEBUG="false"
