#!/bin/sh
# Isolated tests for Kindle sync-screen.sh remote config caching, offline
# frontlight preservation, input validation, and shell syntax.
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
KDASH_ROOT=$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)
SYNC_SCRIPT="$KDASH_ROOT/deploy/kindle/sync-screen.sh"

TMP_ROOT=$(mktemp -d)
trap 'rm -rf "$TMP_ROOT"' EXIT

echo "==> [1/5] POSIX shell syntax check on Kindle scripts"
for f in "$KDASH_ROOT"/deploy/kindle/*.sh "$KDASH_ROOT"/deploy/kindle/extensions/kdash/*.sh; do
    sh -n "$f"
done

BIN_DIR="$TMP_ROOT/bin"
KD_DIR="$TMP_ROOT/kdash"
VAR_DIR="$TMP_ROOT/tmp"
mkdir -p "$BIN_DIR" "$KD_DIR" "$VAR_DIR"

# Fake Kindle tools
cat > "$BIN_DIR/lipc-get-prop" <<'EOF'
#!/bin/sh
case "$2" in
    battLevel) echo "75" ;;
    isCharging) echo "0" ;;
    flIntensity) cat "${FAKE_STATE}/fl" 2>/dev/null || echo "12" ;;
    *) echo "" ;;
esac
EOF

cat > "$BIN_DIR/lipc-set-prop" <<'EOF'
#!/bin/sh
echo "$*" >> "${FAKE_STATE}/lipc.log"
case "$*" in
    *"flIntensity "*)
        val=$(echo "$*" | awk '{print $NF}')
        echo "$val" > "${FAKE_STATE}/fl"
        ;;
esac
EOF

cat > "$BIN_DIR/eips" <<'EOF'
#!/bin/sh
echo "$*" >> "${FAKE_STATE}/eips.log"
exit 0
EOF

cat > "$BIN_DIR/wget" <<'EOF'
#!/bin/sh
out=""
url=""
while [ $# -gt 0 ]; do
    case "$1" in
        -O) out="$2"; shift 2 ;;
        -T) shift 2 ;;
        -q) shift ;;
        *) url="$1"; shift ;;
    esac
done
mode=$(cat "${FAKE_STATE}/wget_mode" 2>/dev/null || echo "online")
if [ "$mode" = "offline" ]; then
    exit 1
fi
case "$url" in
    */client.conf)
        if [ "$mode" = "malicious" ]; then
            printf 'FRONTLIGHT_MODE=$(touch %s/pwned)\nBAD_KEY=123\n' "${FAKE_STATE}" > "$out"
            exit 0
        fi
        cat "${FAKE_STATE}/remote.conf" > "$out"
        exit 0
        ;;
    */screen.png*)
        printf 'FAKE_PNG_%s\n' "$mode" > "$out"
        exit 0
        ;;
esac
exit 1
EOF

chmod 755 "$BIN_DIR"/*

FAKE_STATE="$TMP_ROOT/state"
mkdir -p "$FAKE_STATE"
export FAKE_STATE
export KDASH_DIR="$KD_DIR"
export KDASH_TMP="$VAR_DIR"
export PATH="$BIN_DIR:$PATH"

cat > "$KD_DIR/client.conf" <<'EOF'
SERVER_URL="http://127.0.0.1:8088/screen.png"
REFRESH_SECONDS=60
GHOST_CLEAR_SECONDS=3600
TOUCH_PAUSE_SECONDS=180
FRONTLIGHT_MODE="schedule"
DAY_BRIGHTNESS=12
NIGHT_BRIGHTNESS=0
DAY_START_HOUR=0
NIGHT_START_HOUR=23
EOF

cat > "$FAKE_STATE/remote.conf" <<'EOF'
FRONTLIGHT_MODE=off
DAY_BRIGHTNESS=0
NIGHT_BRIGHTNESS=0
DAY_START_HOUR=7
NIGHT_START_HOUR=22
GHOST_CLEAR_SECONDS=3600
EOF

echo "==> [2/5] Online sync caches remote frontlight=off and applies brightness 0"
echo "online" > "$FAKE_STATE/wget_mode"
echo "12" > "$FAKE_STATE/fl"
sh "$SYNC_SCRIPT" once >/dev/null
[ -f "$VAR_DIR/kdash-remote.conf" ] || { echo "FAIL: remote cache missing"; exit 1; }
[ "$(cat "$FAKE_STATE/fl")" = "0" ] || { echo "FAIL: expected fl=0, got $(cat "$FAKE_STATE/fl")"; exit 1; }
[ -f "$VAR_DIR/screen_current.png" ] || { echo "FAIL: screen_current.png missing"; exit 1; }

echo "==> [3/5] Offline sync preserves cached frontlight=off instead of reverting to local schedule (12)"
echo "offline" > "$FAKE_STATE/wget_mode"
echo "12" > "$FAKE_STATE/fl"
if sh "$SYNC_SCRIPT" once >/dev/null 2>&1; then
    echo "FAIL: offline once should exit non-zero when wget fails"
    exit 1
fi
[ "$(cat "$FAKE_STATE/fl")" = "0" ] || { echo "FAIL: offline reverted frontlight to $(cat "$FAKE_STATE/fl")"; exit 1; }

echo "==> [4/5] Malicious remote config does not execute code and preserves previous valid cache"
echo "malicious" > "$FAKE_STATE/wget_mode"
echo "12" > "$FAKE_STATE/fl"
sh "$SYNC_SCRIPT" once >/dev/null
[ ! -f "$FAKE_STATE/pwned" ] || { echo "FAIL: command injection occurred!"; exit 1; }
[ "$(cat "$FAKE_STATE/fl")" = "0" ] || { echo "FAIL: malicious payload corrupted cached frontlight: $(cat "$FAKE_STATE/fl")"; exit 1; }

echo "==> [5/5] Newer local client.conf while offline overrides stale remote cache"
sleep 1
cat > "$KD_DIR/client.conf" <<'EOF'
SERVER_URL="http://127.0.0.1:8088/screen.png"
FRONTLIGHT_MODE="on"
DAY_BRIGHTNESS=10
NIGHT_BRIGHTNESS=0
DAY_START_HOUR=7
NIGHT_START_HOUR=22
EOF
echo "offline" > "$FAKE_STATE/wget_mode"
echo "0" > "$FAKE_STATE/fl"
sh "$SYNC_SCRIPT" once >/dev/null 2>&1 || true
[ "$(cat "$FAKE_STATE/fl")" = "10" ] || { echo "FAIL: newer local client.conf not respected, got $(cat "$FAKE_STATE/fl")"; exit 1; }

# Verify no PID-scoped temp files leaked
if ls "$VAR_DIR"/client.conf.remote.* "$VAR_DIR"/screen_new.*.png >/dev/null 2>&1; then
    echo "FAIL: leaked temp files in $VAR_DIR"
    exit 1
fi

echo "ALL KINDLE CLIENT TESTS PASSED"
