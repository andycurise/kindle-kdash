#!/bin/sh
# Shared, optional server notification for the KUAL frontlight actions.
# Configuration is parsed as data; it is never sourced as shell code.

frontlight_notify() {
    mode=$1
    conf=/mnt/us/kdash/client.conf
    [ -f "$conf" ] || return 0

    server_url=
    settings_token=
    while IFS='=' read -r raw_key raw_val || [ -n "$raw_key" ]; do
        key=$(echo "$raw_key" | tr -d ' \t\r\n')
        case "$key" in
            SERVER_URL|SETTINGS_TOKEN) ;;
            *) continue ;;
        esac
        val=$(echo "$raw_val" | tr -d '\r' | sed -e 's/^[ \t"'\''"]*//' -e 's/[ \t"'\''"]*$//')
        case "$key" in
            SERVER_URL) server_url=$val ;;
            SETTINGS_TOKEN) settings_token=$val ;;
        esac
    done < "$conf"

    [ -n "$server_url" ] || return 0
    case "$server_url" in
        http://*|https://*) ;;
        *) return 1 ;;
    esac
    base_url=$(echo "$server_url" | sed -e 's|/screen\.png.*||' -e 's|/*$||')
    if [ -n "$settings_token" ]; then
        wget -q -T 3 -O /dev/null --post-data='' --header="X-KDash-Client: kdash" --header="X-KDash-Token: $settings_token" "${base_url}/api/set-frontlight?mode=${mode}" 2>/dev/null
    else
        wget -q -T 3 -O /dev/null --post-data='' --header="X-KDash-Client: kdash" "${base_url}/api/set-frontlight?mode=${mode}" 2>/dev/null
    fi
}
