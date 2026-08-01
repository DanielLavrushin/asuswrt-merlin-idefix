#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

generate_token() {
    log_info "Generating token for $ADDON_TITLE..."

    local payload
    payload=$(reconstruct_payload "$1")

    local client_token
    client_token=$(echo "$payload" | jq -r '.client_token')

    # The client id is signed and echoed back into token.json, so keep it to a
    # charset that can break out of neither.
    case "$client_token" in
    '' | null | *[!A-Za-z0-9_-]*)
        log_error "Refusing to sign a malformed client token."
        return 1
        ;;
    esac

    if [ "${#client_token}" -lt 8 ] || [ "${#client_token}" -gt 64 ]; then
        log_error "Refusing to sign a client token of invalid length."
        return 1
    fi

    if [ ! -s "$ADDON_SECRET_FILE" ]; then
        log_error "Missing signing key: $ADDON_SECRET_FILE"
        return 1
    fi

    local now
    now=$(date +%s)

    local secret
    secret=$(cat "$ADDON_SECRET_FILE")

    local sig
    sig=$(printf '%s|%d' "$client_token" "$now" |
        openssl dgst -sha256 -mac HMAC -macopt "hexkey:$secret" -hex |
        awk '{print $2}')

    if [ -z "$sig" ]; then
        log_error "Failed to sign the terminal token."
        return 1
    fi

    # Never log $secret or $sig. Everything logged here reaches the router's
    # system log, which people routinely paste into support threads — and the
    # signing key alone is enough to mint a token for a root shell.
    log_debug "Client token: $client_token (ts=$now)"

    local old_umask
    old_umask=$(umask)
    umask 077
    echo -n "{\"cl\":\"$client_token\", \"ts\":$now, \"sig\":\"$sig\"}" >"$ADDON_TOKEN_FILE"
    umask "$old_umask"
    chmod 600 "$ADDON_TOKEN_FILE"

    ln -s -f "$ADDON_TOKEN_FILE" "$ADDON_WEB_DIR/token.json" || log_error "Failed to create symlink for token.json."
}

generate_secret() {
    log_info "Generating secret for $ADDON_TITLE..."

    mkdir -p "$(dirname "$ADDON_SECRET_FILE")"

    rm -f "$ADDON_SECRET_FILE"

    # Create it unreadable rather than widening it after the fact — chmod after
    # the write leaves the key world-readable in between.
    local old_umask
    old_umask=$(umask)
    umask 077

    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 32 >"$ADDON_SECRET_FILE"
    else
        hexdump -v -n 32 -e '1/1 "%02x"' /dev/urandom >"$ADDON_SECRET_FILE"
    fi

    umask "$old_umask"
    chmod 600 "$ADDON_SECRET_FILE"

    if [ ! -s "$ADDON_SECRET_FILE" ]; then
        log_error "Failed to generate signing key: $ADDON_SECRET_FILE"
        return 1
    fi
}
