#!/bin/sh
# shellcheck disable=SC2034  # codacy:Unused variables

# ANSI color codes (only used if stdout is a terminal)
CDBG='\033[0;90m'  # gray
CERR='\033[0;31m'  # red
CWARN='\033[0;33m' # yellow
CINFO='\033[0;36m' # cyan
CSUC='\033[0;32m'  # green
CLOG='\033[0;37m'  # white
CRESET='\033[0m'
LOG_FACILITY='user'

if [ -t 1 ]; then
    USE_COLOR=1
else
    USE_COLOR=0
fi

printlog() {
    level=$1
    shift
    msg=$*

    # Map LEVEL → syslog priority & console color
    case "$level" in
    ALERT | CRIT)
        priority='crit'
        color=$CERR
        ;;
    ERROR)
        priority='err'
        color=$CERR
        ;;
    WARN | WARNING)
        priority='warning'
        color=$CWARN
        ;;
    NOTICE)
        priority='notice'
        color=$CINFO
        ;;

    LOG)
        priority='info'
        color=$CLOG
        ;;
    INFO)
        priority='info'
        color=$CINFO
        ;;
    DEBUG)
        priority='debug'
        color=$CDBG
        ;;
    SUCCESS | OK)
        priority='info'
        color=$CSUC
        ;;
    *)
        priority='info'
        color=$CINFO
        ;;
    esac

    logger -t "IDEFIX" -p "${LOG_FACILITY}.${priority}" -- "$msg"

    if [ "$ADDON_DEBUG" = "false" ] && [ "$level" = "DEBUG" ]; then
        return
    fi

    if [ "$USE_COLOR" -eq 1 ]; then
        printf '%b%s%b\n' "$color" "$msg" "$CRESET"
    else
        printf '%s\n' "$msg"
    fi
}

log_alert() { printlog ALERT "$@"; }
log_crit() { printlog CRIT "$@"; }
log_error() { printlog ERROR "$@"; }
log_warn() { printlog WARN "$@"; }
log_notice() { printlog NOTICE "$@"; }
log_info() { printlog INFO "$@"; }
log_log() { printlog LOG "$@"; }
log_ok() { printlog OK "$@"; }
log_debug() { printlog DEBUG "$@"; }

log_info_box() {
    log_box "$1" "$CINFO"
}

log_warn_box() {
    log_box "$1" "$CWARN"
}

log_error_box() {
    log_box "$1" "$CERR"
}

log_box() {
    msg="$1"
    padding=2
    len=$(printf '%s' "$msg" | wc -c)
    width=$((len + padding * 2))
    color="${2:-$CSUC}"

    border=''
    i=0
    while [ "$i" -lt "$width" ]; do
        border="${border}═"
        i=$((i + 1))
    done

    printf '%b╔%s╗\n' "$color" "$border"
    printf '%b║%*s║\n' "$color" "$width" ''
    printf '%b║%*s%s%*s║\n' "$color" "$padding" '' "$msg" "$padding" ''
    printf '%b║%*s║\n' "$color" "$width" ''
    printf '%b╚%s╝%b\n' "$color" "$border" "$CRESET"
}

get_proc() {
    local proc_name="$1"
    echo $(/bin/pidof "$proc_name" 2>/dev/null)
}

get_proc_uptime() {
    local uptime_s=$(cut -d. -f1 /proc/uptime)
    local pid=$(pidof "$1")

    local localstart_time_jiffies=$(awk '{print $22}' /proc/$pid/stat)

    local jiffies_per_sec=100

    local process_start_s=$((localstart_time_jiffies / jiffies_per_sec))

    local proc_uptime=$((uptime_s - process_start_s))
    echo $proc_uptime
}

get_webui_page() {
    ADDON_USER_PAGE="none"
    local max_user_page=0
    local used_pages=""

    for page in /www/user/user*.asp; do
        if [ -f "$page" ]; then
            if grep -q "page:$ADDON_TAG" "$page"; then
                ADDON_USER_PAGE=$(basename "$page")
                log_ok "Found existing $ADDON_TAG_UPPER page: $ADDON_USER_PAGE"
                return
            fi

            user_number=$(echo "$page" | sed -E 's/.*user([0-9]+)\.asp$/\1/')
            used_pages="$used_pages $user_number"

            if [ "$user_number" -gt "$max_user_page" ]; then
                max_user_page="$user_number"
            fi
        fi
    done

    if [ "$ADDON_USER_PAGE" != "none" ]; then
        log_ok "Found existing $ADDON_TAG_UPPER page: $ADDON_USER_PAGE"
        return
    fi

    if [ "$1" = "true" ]; then
        i=1
        while true; do
            if ! echo "$used_pages" | grep -qw "$i"; then
                ADDON_USER_PAGE="user$i.asp"
                log_ok "Assigning new $ADDON_TAG_UPPER page: $ADDON_USER_PAGE"
                return
            fi
            i=$((i + 1))
        done
    fi
}

am_settings_del() {
    local key="$1"
    sed -i "/$key/d" /jffs/addons/custom_settings.txt
}

reconstruct_payload() {

    local idx=0
    local chunk
    local payload=""
    while :; do
        chunk=$(am_settings_get idefix_payload$idx)
        if [ -z "$chunk" ]; then
            break
        fi
        payload="$payload$chunk"
        idx=$((idx + 1))
    done

    cleanup_payload

    echo "$payload"

}

cleanup_payload() {
    # clean up all payload chunks from the custom settings
    sed -i '/^idefix_payload/d' /jffs/addons/custom_settings.txt
}

load_ui_response() {

    if [ ! -f "$ADDON_RESPONSE_FILE" ]; then
        log_ok "Creating $ADDON_TITLE response file: $ADDON_RESPONSE_FILE"
        echo '{}' >"$ADDON_RESPONSE_FILE"
        chmod 600 "$ADDON_RESPONSE_FILE"
    fi

    UI_RESPONSE=$(cat "$ADDON_RESPONSE_FILE")
    if [ "$UI_RESPONSE" = "" ]; then
        UI_RESPONSE="{}"
    fi
}

save_ui_response() {

    log_debug "Saving UI response to $ADDON_RESPONSE_FILE"

    if ! echo "$UI_RESPONSE" >"$ADDON_RESPONSE_FILE"; then
        log_error "Failed to save UI response to $ADDON_RESPONSE_FILE"
        clear_lock
        return 1
    fi

}

update_loading_progress() {
    local message=$1
    local loginfo=${2:-false}
    local progress=$3

    if [ "$loginfo" = "true" ]; then
        log_info "$message"
    fi

    if [ ! -d "$ADDON_WEB_DIR" ]; then
        return
    fi

    load_ui_response

    local json_content
    if [ -f "$ADDON_RESPONSE_FILE" ]; then
        json_content=$(cat "$ADDON_RESPONSE_FILE")
    else
        json_content="{}"
    fi

    if [ -n "$progress" ]; then
        json_content=$(echo "$json_content" | jq --argjson progress "$progress" --arg message "$message" '
            .loading.message = $message |
            .loading.progress = $progress
        ')
    else
        json_content=$(echo "$json_content" | jq --arg message "$message" '
            .loading.message = $message
        ')
    fi

    echo "$json_content" >"/tmp/idefix-response.tmp" && mv -f "/tmp/idefix-response.tmp" "$ADDON_RESPONSE_FILE"

    if [ "$progress" = "100" ]; then
        $ADDON_SCRIPT service_event loading clean >/dev/null 2>&1 &
    fi

}

remove_loading_progress() {

    log_info "Removing loading progress..."
    if [ ! -d "$ADDON_WEB_DIR" ]; then
        return
    fi

    sleep 1
    load_ui_response

    local json_content=$(cat "$ADDON_RESPONSE_FILE")

    json_content=$(echo "$json_content" | jq '
            del(.loading)
        ')

    echo "$json_content" >"/tmp/idefix-response.tmp" && mv -f "/tmp/idefix-response.tmp" "$ADDON_RESPONSE_FILE"
}
