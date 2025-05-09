#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

export PATH="/opt/bin:/opt/sbin:/sbin:/bin:/usr/sbin:/usr/bin"
export HOME="/tmp/home/root"
export USER="root"
export SHELL="/bin/sh"
export TERM="xterm-256color"

source /usr/sbin/helper.sh

ADDON_TAG="idefix"
ADDON_TAG_UPPER="IDEFIX"
ADDON_TITLE="Idefix Terminal"

ADDON_VERSION="1.0.0"

ADDON_WEB_DIR="/www/user/$ADDON_TAG"

ADDON_SCRIPT="/jffs/scripts/$ADDON_TAG"
ADDON_SHARE_DIR="/opt/share/$ADDON_TAG"
ADDON_LOGS_DIR="$ADDON_SHARE_DIR/logs"

ADDON_SERVER="$ADDON_SHARE_DIR/idefix-server"

ADDON_SERVER_PORT=8787

ADDON_TOKEN_FILE="/tmp/$ADDON_TAG.token"
ADDON_SECRET_FILE=/jffs/addons/idefix/sec.key

ADDON_DEBUG="true"
