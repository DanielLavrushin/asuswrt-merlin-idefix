#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

update() {

    update_loading_progress "Updating $ADDON_TITLE..." true

    local pid=$(get_proc "idefix-server")

    local specific_version=${1:-"latest"}
    local temp_file="/tmp/asuswrt-merlin-idefix.tar.gz"
    local temp_dir="/tmp/idefix"

    local url="https://github.com/daniellavrushin/asuswrt-merlin-idefix/releases/latest/download/asuswrt-merlin-idefix.tar.gz"

    if [ ! $specific_version = "latest" ]; then

        local url="https://github.com/DanielLavrushin/asuswrt-merlin-idefix/releases/download/v$specific_version/asuswrt-merlin-idefix.tar.gz"
    fi

    update_loading_progress "Downloading the version:$specific_version..." true
    if wget -q --show-progress -O "$temp_file" "$url"; then
        log_ok "Download completed successfully."
    else
        log_error "Failed to download the $specific_version version. Exiting."
        return 1
    fi

    update_loading_progress "Extracting the package..." true
    if tar -xzf "$temp_file" -C "/tmp"; then
        log_ok "Extraction completed."
    else
        log_error "Failed to extract the package. Exiting."
        return 1
    fi

    rm -f "$temp_file"

    update_loading_progress "Setting up the script..." true
    if mv "$temp_dir/idefix" "$ADDON_SCRIPT" && chmod 0755 "$ADDON_SCRIPT"; then
        log_ok "Script set up successfully."
    else
        log_error "Failed to set up the script. Exiting."
        return 1
    fi

    update_server "$specific_version"

    update_loading_progress "Running the installation..." true
    if sh "$ADDON_SCRIPT" install; then
        log_ok "Installation completed successfully."
    else
        log_error "Installation failed. Exiting."
        return 1
    fi

}

update_server() {

    log_info "Installing $ADDON_TITLE server..."

    stop || true

    local specific_version=${1:-"latest"}
    local temp_file="/tmp/idefix-server.tar.gz"
    local temp_dir="/tmp/idefix"

    mkdir -p "$temp_dir"

    local arch=$(uname -m)
    local asset_name=""
    case "$arch" in
    x86_64)
        asset_name="idefix-server-amd64.tar.gz"
        ;;
    armv5* | armv6* | armv7*)
        asset_name="idefix-server-arm.tar.gz"
        ;;
    aarch64 | arm64)
        asset_name="idefix-server-arm64.tar.gz"
        ;;
    *)
        echo "Unsupported architecture: $arch"
        return 1
        ;;
    esac

    log_debug "Detected architecture: $arch"
    log_debug "Using asset name: $asset_name"
    log_debug "Downloading $ADDON_TITLE $specific_version server..."

    local asset_url="https://github.com/daniellavrushin/asuswrt-merlin-idefix/releases/latest/download/$asset_name"

    if [ ! $specific_version = "latest" ]; then

        local asset_url="https://github.com/DanielLavrushin/asuswrt-merlin-idefix/releases/download/v$specific_version/$asset_name"
    fi

    log_info "Downloading  release version $asset_url into $temp_file"
    if wget -q --show-progress -O "$temp_file" "$asset_url"; then
        log_info "Download completed successfully."
    else
        log_error "Failed to download the $specific_version version. Exiting."
        return 1
    fi

    log_info "Extracting the package..."
    if tar -xzf "$temp_file" -C "$temp_dir"; then
        log_info "Extraction completed."
    else
        log_error "Failed to extract the package. Exiting."
        return 1
    fi

    rm -f "$temp_file"
    log_info "$ADDON_TITLE server installed successfully."

}
