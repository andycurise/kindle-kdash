#!/bin/sh
# Offline process-level smoke test: real HTTP, no weather or Kindle access.
set -eu

ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
BINARY="$ROOT/build/kdash-dev"
[ -x "$BINARY" ] || {
    echo "run scripts/build.sh first" >&2
    exit 1
}
command -v curl >/dev/null 2>&1
command -v python3 >/dev/null 2>&1

TEST_TMP=$(mktemp -d "${TMPDIR:-/tmp}/kdash-server-test.XXXXXXXX")
PID=
cleanup() {
    status=$?
    trap - EXIT HUP INT TERM
    if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
        kill "$PID" 2>/dev/null || true
        wait "$PID" 2>/dev/null || true
    fi
    rm -rf "$TEST_TMP"
    exit "$status"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

write_config() {
    python3 -I -c 'import json,sys; p=sys.argv[1]; d=json.load(open(p)); d["listen_addr"]=sys.argv[2]; d["settings_token"]="integration-secret"; d["weather_cache_path"]=sys.argv[3]; json.dump(d,open(p,"w"),ensure_ascii=False,indent=2)' \
        "$1" "$2" "$TEST_TMP/weather.json"
}

cp "$ROOT/configs/kdash.json.example" "$TEST_TMP/kdash.json"
chmod 600 "$TEST_TMP/kdash.json"
write_config "$TEST_TMP/kdash.json" 127.0.0.1:18088

start_server() {
    "$BINARY" -config "$TEST_TMP/kdash.json" -offline > "$TEST_TMP/server.log" 2>&1 &
    PID=$!
    attempt=0
    while [ "$attempt" -lt 30 ]; do
        if curl -fsS "http://127.0.0.1:$1/api/status" -o "$TEST_TMP/status.json" 2>/dev/null; then
            grep -F '"version":"'"$(cat "$ROOT/VERSION")"'"' "$TEST_TMP/status.json" >/dev/null
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    printf '%s\n' "server did not become healthy on port $1" >&2
    exit 1
}

stop_server() {
    code=$(curl -sS -o "$TEST_TMP/shutdown.json" -w '%{http_code}' \
        -X POST -H 'X-KDash-Token: integration-secret' -H 'X-KDash-Client: kdash' \
        "http://127.0.0.1:$1/api/shutdown")
    [ "$code" = "200" ]
    wait "$PID"
    PID=
}

start_server 18088
curl -fsS "http://127.0.0.1:18088/screen.png" -o "$TEST_TMP/screen.png"
python3 -I -c 'import struct,sys; d=open(sys.argv[1],"rb").read(); assert d[:8]==b"\x89PNG\r\n\x1a\n"; assert struct.unpack(">II",d[16:24])==(758,1024); assert d[24]==8 and d[25]==0, (d[24],d[25])' "$TEST_TMP/screen.png"

code=$(curl -sS -o "$TEST_TMP/unauthorized.json" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' --data-binary @"$TEST_TMP/kdash.json" \
    "http://127.0.0.1:18088/api/config")
[ "$code" = "401" ]
code=$(curl -sS -o "$TEST_TMP/frontlight-get.json" -w '%{http_code}' \
    "http://127.0.0.1:18088/api/set-frontlight?mode=off")
[ "$code" = "405" ]
code=$(curl -sS -o "$TEST_TMP/frontlight.json" -w '%{http_code}' \
    -X POST -H 'X-KDash-Token: integration-secret' -H 'X-KDash-Client: kdash' \
    "http://127.0.0.1:18088/api/set-frontlight?mode=off")
[ "$code" = "200" ]

# Save a full config and a different listen port, then restart to prove it is used.
code=$(curl -sS -o "$TEST_TMP/save.json" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' -H 'X-KDash-Token: integration-secret' \
    -H 'X-KDash-Client: kdash' --data-binary @"$TEST_TMP/kdash.json" \
    "http://127.0.0.1:18088/api/config")
[ "$code" = "200" ]
write_config "$TEST_TMP/kdash.json" 127.0.0.1:18089
code=$(curl -sS -o "$TEST_TMP/save-port.json" -w '%{http_code}' \
    -X PUT -H 'Content-Type: application/json' -H 'X-KDash-Token: integration-secret' \
    -H 'X-KDash-Client: kdash' --data-binary @"$TEST_TMP/kdash.json" \
    "http://127.0.0.1:18088/api/config")
[ "$code" = "200" ]
grep -F '"restart_required":true' "$TEST_TMP/save-port.json" >/dev/null
stop_server 18088
start_server 18089
stop_server 18089

printf '%s\n' "offline HTTP process tests passed (ports 18088/18089, version, auth, Gray PNG, restart, shutdown)"
