#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

mount_ui() {
    check_lock
    nvram get rc_support | grep -q am_addons
    if [ $? != 0 ]; then
        log_error "This firmware does not support addons!"
        exit 5
    fi

    get_webui_page true

    if [ "$ADDON_USER_PAGE" = "none" ]; then
        log_error "Unable to install $ADDON_TITLE"
        exit 5
    fi

    update_loading_progress "Mounting $ADDON_TITLE as $ADDON_USER_PAGE" true

    echo "$ADDON_TAG" >"/www/user/$(echo $ADDON_USER_PAGE | cut -f1 -d'.').title"

    if [ ! -f /tmp/menuTree.js ]; then
        cp /www/require/modules/menuTree.js /tmp/
        mount -o bind /tmp/menuTree.js /www/require/modules/menuTree.js
    fi

    sed -i '/index: "menu_Setting"/,/index:/ {
  /url:\s*"NULL",\s*tabName:\s*"__INHERIT__"/ i \
    { url: "'"$ADDON_USER_PAGE"'", tabName: "'"${ADDON_TITLE}"'" },
}' /tmp/menuTree.js

    umount /www/require/modules/menuTree.js && mount -o bind /tmp/menuTree.js /www/require/modules/menuTree.js

    if [ ! -d $ADDON_WEB_DIR ]; then
        mkdir -p "$ADDON_WEB_DIR"
    fi

    ln -s -f "$ADDON_SCRIPT" "/opt/bin/$ADDON_TAG" || log_error "Failed to create symlink for $ADDON_TAG."

    ln -s -f "$ADDON_SHARE_DIR/index.asp" "/www/user/$ADDON_USER_PAGE"
    ln -s -f "$ADDON_SHARE_DIR/app.js" $ADDON_WEB_DIR/app.js || log_error "Failed to create symlink for app.js."

    clear_lock
    log_ok "$ADDON_TITLE mounted successfully as $ADDON_USER_PAGE"
}

unmount_ui() {

    check_lock
    nvram get rc_support | grep -q am_addons
    if [ $? != 0 ]; then
        log_error "This firmware does not support addons!"
        exit 5
    fi

    get_webui_page

    local base_user_page="${ADDON_USER_PAGE%.asp}"

    if [ -z "$ADDON_USER_PAGE" ] || [ "$ADDON_USER_PAGE" = "none" ]; then
        log_warn "No $ADDON_TITLE page found to unmount. Continuing to clean up..."
    else
        log_info "Unmounting $ADDON_TITLE $ADDON_USER_PAGE"
        rm -fr /www/user/$ADDON_USER_PAGE
        rm -fr /www/user/$base_user_page.title
    fi

    if [ ! -f /tmp/menuTree.js ]; then
        log_warn "menuTree.js not found, skipping unmount."
    else
        log_info "Removing any $ADDON_TITLE menu entry from menuTree.js."

        grep -v "tabName: \"$ADDON_TITLE\"" /tmp/menuTree.js >/tmp/menuTree_temp.js
        mv /tmp/menuTree_temp.js /tmp/menuTree.js

        umount /www/require/modules/menuTree.js
        mount -o bind /tmp/menuTree.js /www/require/modules/menuTree.js
    fi

    rm -rf "/opt/bin/$ADDON_TAG" || log_error "Failed to remove symlink for $ADDON_TAG."

    clear_lock
    log_ok "Unmount completed."
}

remount_ui() {
    if [ "$1" != "skipwait" ]; then
        log_warn "sleeping for 10 seconds..."
        # sleep 10
    fi

    unmount_ui
    sleep 1
    mount_ui
}
