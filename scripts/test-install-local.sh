#!/bin/sh
# Exercise the macOS installer with a local release and a fake launchctl.
# No real LaunchAgent, network download, or active configuration is touched.
set -eu

ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
[ "$(uname -s)" = "Darwin" ] || {
    echo "test-install-local.sh currently exercises the macOS installer only" >&2
    exit 1
}
[ -d "$ROOT/dist" ] || {
    echo "run scripts/package.sh first" >&2
    exit 1
}

TEST_TMP=$(mktemp -d "${TMPDIR:-/tmp}/kdash-install-test.XXXXXXXX")
trap 'rm -rf "$TEST_TMP"' EXIT HUP INT TERM
TEST_HOME="$TEST_TMP/home"
FAKE_BIN="$TEST_TMP/fake-bin"
PREFIX="$TEST_TMP/prefix"
mkdir -p "$TEST_HOME" "$FAKE_BIN"

cat > "$FAKE_BIN/launchctl" <<'EOF'
#!/bin/sh
if [ "${1:-}" = "print" ]; then
    exit 1
fi
exit 0
EOF
chmod 755 "$FAKE_BIN/launchctl"

run_installer() {
    HOME="$TEST_HOME" PATH="$FAKE_BIN:$PATH" \
        "$ROOT/scripts/install.sh" --release-base "$ROOT/dist" \
        --prefix "$PREFIX" --non-interactive "$@"
}

run_installer --no-start
CONFIG="$TEST_HOME/Library/Application Support/kdash/kdash.json"
AGENT="$TEST_HOME/Library/LaunchAgents/com.kdash.agent.plist"
[ -x "$PREFIX/bin/kdash" ]
[ -f "$CONFIG" ]
[ -f "$AGENT" ]
[ -f "$PREFIX/share/doc/kdash/README.md" ]
[ -f "$PREFIX/share/doc/kdash/assets/fonts/licenses/NotoSansSC-OFL.txt" ]
[ "$("$PREFIX/bin/kdash" -version)" = "$(cat "$ROOT/VERSION")" ]
[ "$(stat -f '%Lp' "$CONFIG")" = "600" ]
grep -Eq '"settings_token"[[:space:]]*:[[:space:]]*"[0-9a-f]{64}"' "$CONFIG" >/dev/null
grep -F '"weather_cache_path": "' "$CONFIG" >/dev/null
grep -F "$PREFIX/bin/kdash" "$AGENT" >/dev/null
BEFORE=$(cksum < "$CONFIG")

# Reinstallation must preserve the exact existing configuration.
run_installer --no-start
AFTER=$(cksum < "$CONFIG")
[ "$BEFORE" = "$AFTER" ]

# A bad checksum must fail before changing the installed binary or config.
cp -R "$ROOT/dist" "$TEST_TMP/bad-release"
printf '%s\n' 'not a valid checksum manifest' > "$TEST_TMP/bad-release/SHA256SUMS"
if HOME="$TEST_HOME" PATH="$FAKE_BIN:$PATH" \
    "$ROOT/scripts/install.sh" --release-base "$TEST_TMP/bad-release" \
    --prefix "$PREFIX" --non-interactive --no-start >/dev/null 2>&1; then
    echo "installer accepted an invalid SHA256SUMS" >&2
    exit 1
fi
[ "$("$PREFIX/bin/kdash" -version)" = "$(cat "$ROOT/VERSION")" ]
[ "$(cksum < "$CONFIG")" = "$BEFORE" ]

# Uninstall keeps only the requested configuration/data in the temporary HOME.
run_installer --uninstall --keep-config
[ -f "$CONFIG" ]
[ ! -e "$PREFIX/bin/kdash" ]
[ ! -e "$AGENT" ]

printf '%s\n' "macOS local installer tests passed (fake launchctl; no service was loaded)"
