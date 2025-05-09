#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

generate_token() {
    log_info "Generating token for $ADDON_TITLE..."

    local payload=$(reconstruct_payload "$1")
    local client_token=$(echo "$payload" | jq -r '.client_token')

    local now=$(date +%s)

    local secret=$(cat $ADDON_SECRET_FILE)
    local sig=$(printf '%s|%d' "$client_token" "$now" |
        openssl dgst -sha256 -mac HMAC -macopt "hexkey:$secret" -hex |
        awk '{print $2}')

    log_debug "Client token: $client_token"
    log_debug "Secret: $secret"
    log_debug "Signature: $sig"
    log_debug "Now: $now"

    echo -n "{\"cl\":\"$client_token\", \"ts\":$now, \"sig\":\"$sig\"}" >"$ADDON_TOKEN_FILE"

    ln -s -f "$ADDON_TOKEN_FILE" "$ADDON_WEB_DIR/token.json" || log_error "Failed to create symlink for token.json."
}

generate_secret() {
    log_info "Generating secret for $ADDON_TITLE..."

    mkdir -p "$(dirname "$ADDON_SECRET_FILE")"

    rm -f "$ADDON_SECRET_FILE"

    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 32 >"$ADDON_SECRET_FILE"
    else
        hexdump -v -n 32 -e '1/1 "%02x"' /dev/urandom >"$ADDON_SECRET_FILE"
    fi
    chmod 600 "$ADDON_SECRET_FILE"
}
