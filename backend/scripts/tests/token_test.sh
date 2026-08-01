#!/bin/sh
# Exercises token.sh under busybox ash, the shell it actually runs under on the
# router. Stubs out the router-specific bits (logger, custom_settings).
#
#   busybox sh backend/scripts/tests/token_test.sh
#
# Requires jq and openssl, same as the addon itself.
set -u

SRC=${SRC:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

SYSLOG="$WORK/syslog"
: >"$SYSLOG"

ADDON_TITLE="Idefix"
ADDON_DEBUG="false"
ADDON_SECRET_FILE="$WORK/sec.key"
ADDON_TOKEN_FILE="$WORK/idefix.token"
ADDON_WEB_DIR="$WORK/www"
LOG_FACILITY='user'
USE_COLOR=0
mkdir -p "$ADDON_WEB_DIR"

# Stub logger so we can assert on what reaches syslog.
logger() {
    shift 4 2>/dev/null
    printf '%s\n' "$*" >>"$SYSLOG"
}

CDBG='' CERR='' CWARN='' CINFO='' CSUC='' CLOG='' CRESET=''

. "$SRC/_helper.sh"
. "$SRC/token.sh"

# reconstruct_payload reads router nvram; feed it directly instead.
reconstruct_payload() { printf '%s' "$PAYLOAD"; }

fail=0
check() {
    if [ "$2" = "$3" ]; then
        printf 'ok   %s\n' "$1"
    else
        printf 'FAIL %s\n       got:  %s\n       want: %s\n' "$1" "$2" "$3"
        fail=1
    fi
}

# --- generate_secret ---------------------------------------------------------
generate_secret >/dev/null 2>&1
check "secret file is 0600" "$(stat -c '%a' "$ADDON_SECRET_FILE")" "600"
check "secret is 64 hex chars" "$(wc -c <"$ADDON_SECRET_FILE" | tr -d ' ')" "65"

# --- generate_token, happy path ---------------------------------------------
PAYLOAD='{"client_token":"AbCdEf0123456789"}'
generate_token x >/dev/null 2>&1
check "token generated" "$?" "0"
check "token file is 0600" "$(stat -c '%a' "$ADDON_TOKEN_FILE")" "600"
check "token.json symlinked" "$([ -L "$ADDON_WEB_DIR/token.json" ] && echo yes)" "yes"

cl=$(jq -r '.cl' <"$ADDON_TOKEN_FILE")
sig=$(jq -r '.sig' <"$ADDON_TOKEN_FILE")
ts=$(jq -r '.ts' <"$ADDON_TOKEN_FILE")
check "token json is valid and carries the client id" "$cl" "AbCdEf0123456789"

# Signature must match what the Go server recomputes: HMAC-SHA256(cl|ts).
secret=$(cat "$ADDON_SECRET_FILE")
expect=$(printf '%s|%d' "$cl" "$ts" |
    openssl dgst -sha256 -mac HMAC -macopt "hexkey:$secret" -hex | awk '{print $2}')
check "signature matches HMAC(cl|ts)" "$sig" "$expect"

# --- the leak ----------------------------------------------------------------
check "secret never reaches syslog" "$(grep -c "$secret" "$SYSLOG")" "0"
check "signature never reaches syslog" "$(grep -c "$sig" "$SYSLOG")" "0"

# --- rejected client tokens --------------------------------------------------
for bad in '"a\"b; rm -rf /"' '"short"' '"has space"' '""' 'null'; do
    PAYLOAD="{\"client_token\":$bad}"
    if generate_token x >/dev/null 2>&1; then
        printf 'FAIL accepted malformed client token: %s\n' "$bad"
        fail=1
    else
        printf 'ok   rejected malformed client token: %s\n' "$bad"
    fi
done

# A rejected token must not have overwritten the good one.
check "good token survived the bad ones" "$(jq -r '.cl' <"$ADDON_TOKEN_FILE")" "AbCdEf0123456789"

exit "$fail"
