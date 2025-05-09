#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

clear_script_entries() {

    log_info "Removing existing $ADDON_TITLE entries from scripts."
    sed -i '/#idefix/d' /jffs/scripts/services-start >/dev/null 2>&1 || log_debug "Failed to remove entry from /jffs/scripts/services-start."
    sed -i '/#idefix/d' /jffs/scripts/nat-start >/dev/null 2>&1 || log_debug "Failed to remove entry from /jffs/scripts/nat-start."
    sed -i '/#idefix/d' /jffs/scripts/post-mount >/dev/null 2>&1 || log_debug "Failed to remove entry from /jffs/scripts/post-mount."
    sed -i '/#idefix/d' /jffs/scripts/service-event >/dev/null 2>&1 || log_debug "Failed to remove entry from /jffs/scripts/service-event."
    sed -i '/#idefix/d' /jffs/scripts/dnsmasq.postconf >/dev/null 2>&1 || log_debug "Failed to remove entry from /jffs/scripts/dnsmasq.postconf."
    sed -i '/#idefix/d' /jffs/scripts/wan-start >/dev/null 2>&1 || log_debug "Failed to remove entry from /jffs/scripts/wan-start."
}

check_jffs_ready() {

    if [ "$(nvram get jffs2_on 2>/dev/null)" != "1" ]; then
        log_error "JFFS partition is DISABLED (nvram jffs2_on=0)."
        log_error "Enable it under Administration ▸ System, or run:"
        log_error "    nvram set jffs2_on=1 && nvram commit && reboot"
        exit 1
    else
        log_info "JFFS partition is ENABLED."
    fi

    if [ "$(nvram get jffs2_scripts 2>/dev/null)" != "1" ]; then
        log_error "JFFS custom scripts and configs are DISABLED."
        log_error "Enable it in the router UI (Administration ▸ System) or run:"
        log_error "    nvram set jffs2_scripts=1 && nvram commit && reboot"
        exit 1
    else
        log_info "JFFS custom scripts and configs are ENABLED."
    fi
}

setup_script_file() {

    local script_file="$1"
    local command_entry="$2"

    log_debug "Scritpt file: $script_file"
    log_debug "Command entry: $command_entry"

    if [ ! -f "$script_file" ]; then
        printf '#!/bin/sh\n\n' >"$script_file"
    else
        if ! head -n1 "$script_file" | grep -qx '#!/bin/sh'; then
            log_error "$script_file is missing the she-bang (#!/bin/sh)"
        fi
    fi

    if ! grep -Fqx -- "$command_entry" "$script_file"; then
        echo "$command_entry" >>"$script_file" || {
            log_error "Failed to add command entry to $script_file"
            return 1
        }
    fi

    chmod +x "$script_file"
    log_info "Updated $script_file with $ADDON_TITLE command entry."
}

install() {

    log_info "Installing $ADDON_TITLE $ADDON_VERSION..."

    # Add or update post-mount
    setup_script_file "/jffs/scripts/post-mount" "/jffs/scripts/idefix startup & #idefix"

    # Add or update service-event
    setup_script_file "/jffs/scripts/service-event" "echo \"\$2\" | grep -q \"^idefix\" && /jffs/scripts/idefix service_event \$(echo \"\$2\" | cut -d'_' -f2- | tr '_' ' ') & #idefix"

    generate_secret
    firewall_add_rules

    am_settings_set "idefix_version" "$ADDON_VERSION"

    local tmp_dir="/tmp/idefix"
    mkdir -p /jffs/share/idefix

    mv "$tmp_dir/app.js" /jffs/share/idefix
    mv "$tmp_dir/index.asp" /jffs/share/idefix

    mv "$tmp_dir/idefix-server" /jffs/scripts/idefix

    chmod 0755 /jffs/scripts/idefix-server

    remount_ui

    ln -s -f "$ADDON_SCRIPT" "/opt/bin/$ADDON_TAG" || log_error "Failed to create symlink for $ADDON_TAG."

    rm -rf "$tmp_dir"

    log_box "$ADDON_TITLE $ADDON_VERSION installed successfully."
}

uninstall() {
    clear_script_entries
    unmount_ui
    firewall_clear_rules

    am_settings_del "idefix"

    rm -rf /jffs/scripts/idefix
    rm -rf /opt/share/idefix

    rm -rf /www/user/idefix

    rm -rf "/opt/bin/$ADDON_TAG" || log_error "Failed to remove symlink for $ADDON_TAG."

    log_box "$ADDON_TITLE $ADDON_VERSION uninstalled successfully."
}
