#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

update() {
    log_info "Updating $ADDON_TITLE..."
    local pid=$(get_proc "idefix-server")

    local specific_version=${1:-"latest"}
    local temp_file="/tmp/asuswrt-merlin-idefix.tar.gz"

    local url=$(github_proxy_url "https://github.com/daniellavrushin/asuswrt-merlin-idefix/releases/latest/download/asuswrt-merlin-xrayui.tar.gz")

    if [ ! $specific_version = "latest" ]; then

        local url=$(github_proxy_url "https://github.com/DanielLavrushin/asuswrt-merlin-idefix/releases/download/v$specific_version/asuswrt-merlin-xrayui.tar.gz")
    fi

    log_info "Downloading the version:$specific_version..."
    update_loading_progress "Downloading the version:$specific_version..."
    if wget -q --show-progress -O "$temp_file" "$url"; then
        log_info "Download completed successfully."
    else
        log_error "Failed to download the $specific_version version. Exiting."
        return 1
    fi

    log_info "Extracting the package..."
    update_loading_progress "Extracting the package..."
    if tar -xzf "$temp_file" -C "$jffs_addons_path"; then
        log_info "Extraction completed."
    else
        log_error "Failed to extract the package. Exiting."
        return 1
    fi

}
