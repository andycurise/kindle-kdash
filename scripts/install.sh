#!/usr/bin/env bash
# Install, upgrade, or remove the kdash daemon on Linux and macOS.
# No configuration contents (in particular settings_token) are ever logged.
set -Eeuo pipefail

REPO="andycurise/kindle-kdash"
VERSION="latest"
PREFIX=""
RELEASE_BASE="${KDASH_RELEASE_BASE:-}"
NON_INTERACTIVE=0
NO_START=0
UNINSTALL=0
KEEP_CONFIG=0

OS=""
ARCH=""
ASSET=""
ARCHIVE_TOP=""
LOCAL_RELEASE=0
RELEASE_URL=""
DOWNLOAD_TOOL=""
TMP_DIR=""
BACKUP_DIR=""
TRANSACTION_ACTIVE=0
INSTALL_SUCCEEDED=0

BIN_DIR=""
BIN_PATH=""
DOC_DIR=""
CONFIG_DIR=""
CONFIG_PATH=""
DATA_DIR=""
CACHE_PATH=""
LOG_DIR=""
SERVICE_PATH=""
SERVICE_NAME=""

HAD_BINARY=0
HAD_CONFIG=0
HAD_SERVICE=0
HAD_DOC_DIR=0
HAD_DATA_DIR=0
WAS_RUNNING=0
WAS_ENABLED=0

usage() {
    cat <<'EOF'
Usage: install.sh [options]

Options:
  --repo OWNER/REPO       GitHub repository (required unless --release-base is set)
  --version VERSION       latest (default) or vX.Y.Z
  --prefix PATH           Linux: /usr/local; macOS: $HOME/.local
  --release-base PATH|URL Directory/URL containing release assets directly;
                          also available through KDASH_RELEASE_BASE
  --non-interactive       Never prompt for input
  --no-start              Install/enable resident configuration without starting now
  --uninstall             Remove the program and resident configuration
  --keep-config           With --uninstall, preserve configuration and data
  -h, --help              Show this help

Linux uses /etc/kdash/kdash.json, /var/lib/kdash and a system-level
systemd service running as the unprivileged kdash user. Root is required.
macOS uses a per-user LaunchAgent, configuration under
~/Library/Application Support/kdash and logs under ~/Library/Logs/kdash.
EOF
}

log() {
    printf '%s\n' "$*"
}

die() {
    printf 'kdash installer: %s\n' "$*" >&2
    exit 1
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --repo)
            [ "$#" -ge 2 ] || die "--repo requires a value"
            REPO=$2
            shift 2
            ;;
        --version)
            [ "$#" -ge 2 ] || die "--version requires a value"
            VERSION=$2
            shift 2
            ;;
        --prefix)
            [ "$#" -ge 2 ] || die "--prefix requires a value"
            PREFIX=$2
            shift 2
            ;;
        --release-base)
            [ "$#" -ge 2 ] || die "--release-base requires a value"
            RELEASE_BASE=$2
            shift 2
            ;;
        --non-interactive)
            NON_INTERACTIVE=1
            shift
            ;;
        --no-start)
            NO_START=1
            shift
            ;;
        --uninstall)
            UNINSTALL=1
            shift
            ;;
        --keep-config)
            KEEP_CONFIG=1
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            die "unknown option: $1 (use --help)"
            ;;
    esac
done

[ "$VERSION" = "latest" ] || [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || \
    die "--version must be latest or vX.Y.Z"

case "$(uname -s)" in
    Linux) OS=linux ;;
    Darwin) OS=macos ;;
    *) die "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
    x86_64|amd64) ARCH=amd64 ;;
    arm64|aarch64) ARCH=arm64 ;;
    *) die "unsupported CPU architecture: $(uname -m)" ;;
esac

if [ "$OS" = "linux" ]; then
    [ "$ARCH" = "amd64" ] || [ "$ARCH" = "arm64" ] || die "unsupported Linux architecture: $ARCH"
    ASSET="kdash-linux-${ARCH}.tar.gz"
    PREFIX=${PREFIX:-/usr/local}
    CONFIG_DIR=/etc/kdash
    DATA_DIR=/var/lib/kdash
    CACHE_PATH="$DATA_DIR/cache_weather.json"
    LOG_DIR=""
    SERVICE_NAME=kdash.service
    SERVICE_PATH=/etc/systemd/system/kdash.service
else
    [ "$ARCH" = "arm64" ] || die "macOS releases support only darwin/arm64"
    ASSET="kdash-darwin-arm64.tar.gz"
    PREFIX=${PREFIX:-"$HOME/.local"}
    CONFIG_DIR="$HOME/Library/Application Support/kdash"
    DATA_DIR="$CONFIG_DIR/data"
    CACHE_PATH="$DATA_DIR/cache_weather.json"
    LOG_DIR="$HOME/Library/Logs/kdash"
    SERVICE_NAME=com.kdash.agent.plist
    SERVICE_PATH="$HOME/Library/LaunchAgents/com.kdash.agent.plist"
fi

case "$PREFIX" in
    /*) ;;
    *) die "--prefix must be an absolute path" ;;
esac
case "$PREFIX$CONFIG_DIR$DATA_DIR" in
    *$'\n'*|*$'\r'*|*$'\t'*) die "paths must not contain control characters" ;;
esac
while [ "$PREFIX" != "/" ] && [[ "$PREFIX" == */ ]]; do PREFIX=${PREFIX%/}; done
[ "$PREFIX" != "/" ] || die "--prefix=/ is not supported"

BIN_DIR="$PREFIX/bin"
BIN_PATH="$BIN_DIR/kdash"
DOC_DIR="$PREFIX/share/doc/kdash"
CONFIG_PATH="$CONFIG_DIR/kdash.json"
ARCHIVE_TOP=${ASSET%.tar.gz}

require_linux_root() {
    if [ "$(id -u)" -ne 0 ]; then
        die "Linux system-level installation requires root; rerun this command with sudo. No files were changed."
    fi
}

stop_resident_service() {
    if [ "$OS" = "linux" ]; then
        if command -v systemctl >/dev/null 2>&1; then
            systemctl stop kdash.service >/dev/null 2>&1 || true
        fi
    else
        launchctl bootout "gui/$(id -u)/com.kdash.agent" >/dev/null 2>&1 || \
            launchctl unload "$SERVICE_PATH" >/dev/null 2>&1 || true
    fi
}

uninstall_kdash() {
    if [ "$OS" = "linux" ]; then
        require_linux_root
        if command -v systemctl >/dev/null 2>&1; then
            systemctl disable kdash.service >/dev/null 2>&1 || true
        fi
    fi
    stop_resident_service
    rm -f -- "$SERVICE_PATH"
    if [ "$OS" = "linux" ] && command -v systemctl >/dev/null 2>&1; then
        systemctl daemon-reload >/dev/null 2>&1 || true
    fi
    rm -f -- "$BIN_PATH"
    rm -rf -- "$DOC_DIR"
    if [ "$KEEP_CONFIG" -eq 0 ]; then
        rm -rf -- "$CONFIG_DIR"
        if [ "$DATA_DIR" != "$CONFIG_DIR" ] && [[ "$DATA_DIR" != "$CONFIG_DIR"/* ]]; then
            rm -rf -- "$DATA_DIR"
        fi
        [ -z "$LOG_DIR" ] || rm -rf -- "$LOG_DIR"
    fi
    rmdir -- "$BIN_DIR" "$PREFIX/share/doc" "$PREFIX/share" "$PREFIX" 2>/dev/null || true
    log "kdash was uninstalled from $PREFIX"
    if [ "$KEEP_CONFIG" -eq 1 ]; then
        log "Configuration and data were preserved in $CONFIG_DIR and $DATA_DIR"
    elif [ "$OS" = "linux" ]; then
        log "The unprivileged kdash system account was left in place."
    fi
}

if [ "$UNINSTALL" -eq 1 ]; then
    uninstall_kdash
    exit 0
fi

if [ -n "$RELEASE_BASE" ]; then
    case "$RELEASE_BASE" in
        file://*)
            RELEASE_BASE=${RELEASE_BASE#file://}
            [ -d "$RELEASE_BASE" ] || die "local release directory does not exist: $RELEASE_BASE"
            LOCAL_RELEASE=1
            ;;
        http://*|https://*)
            RELEASE_URL=${RELEASE_BASE%/}
            ;;
        *)
            [ -d "$RELEASE_BASE" ] || die "local release directory does not exist: $RELEASE_BASE"
            LOCAL_RELEASE=1
            ;;
    esac
else
    if [ -z "$REPO" ]; then
        die "repository is empty; pass --repo OWNER/REPO or use --release-base for a local release"
    fi
    [[ "$REPO" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || \
        die "--repo must have the form OWNER/REPO"
    if [ "$VERSION" = "latest" ]; then
        RELEASE_URL="https://github.com/$REPO/releases/latest/download"
    else
        RELEASE_URL="https://github.com/$REPO/releases/download/$VERSION"
    fi
fi

if [ "$OS" = "linux" ]; then
    require_linux_root
    command -v systemctl >/dev/null 2>&1 || die "systemctl is required for the Linux system service"
else
    command -v launchctl >/dev/null 2>&1 || die "launchctl is required for the macOS LaunchAgent"
fi

if command -v curl >/dev/null 2>&1; then
    DOWNLOAD_TOOL=curl
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD_TOOL=wget
else
    die "curl or wget is required for downloads and health checks"
fi
command -v tar >/dev/null 2>&1 || die "tar is required"
command -v awk >/dev/null 2>&1 || die "awk is required"

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/kdash-install.XXXXXXXX") || die "could not create a temporary directory"
BACKUP_DIR="$TMP_DIR/backup"
mkdir -p "$BACKUP_DIR"

cleanup_and_exit() {
    local status=$1
    trap - EXIT
    if [ "$status" -ne 0 ] && [ "$TRANSACTION_ACTIVE" -eq 1 ] && [ "$INSTALL_SUCCEEDED" -eq 0 ]; then
        rollback_install
    fi
    [ -z "$TMP_DIR" ] || rm -rf -- "$TMP_DIR"
    exit "$status"
}
trap 'cleanup_and_exit $?' EXIT
trap 'exit 130' HUP INT TERM

fetch_file() {
    local name=$1
    local destination=$2
    local attempt
    local delay
    for attempt in 1 2 3 4; do
        rm -f -- "$destination"
        if [ "$LOCAL_RELEASE" -eq 1 ]; then
            if cp -- "$RELEASE_BASE/$name" "$destination" 2>/dev/null; then
                return 0
            fi
        elif [ "$DOWNLOAD_TOOL" = curl ]; then
            if curl -fL --silent --show-error --connect-timeout 15 --max-time 300 \
                -o "$destination" "$RELEASE_URL/$name"; then
                return 0
            fi
        else
            if wget -q --timeout=300 -O "$destination" "$RELEASE_URL/$name"; then
                return 0
            fi
        fi
        if [ "$attempt" -lt 4 ]; then
            delay=$((attempt * 2))
            log "Download of $name failed; retrying in ${delay}s ($attempt/4)"
            sleep "$delay"
        fi
    done
    rm -f -- "$destination"
    die "could not download $name after 4 attempts from ${RELEASE_BASE:-$RELEASE_URL}"
}

sha256_file() {
    local file=$1
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$file" | awk '{print $1}'
    elif command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 "$file" | awk '{print $NF}'
    else
        die "sha256sum, shasum, or openssl is required to verify SHA256SUMS"
    fi
}

verify_archive() {
    local expected
    local actual
    expected=$(awk -v asset="$ASSET" '
        $2 == asset || $2 == "*" asset { print tolower($1); exit }
    ' "$TMP_DIR/SHA256SUMS")
    [[ "$expected" =~ ^[0-9a-f]{64}$ ]] || die "SHA256SUMS has no valid entry for $ASSET"
    actual=$(sha256_file "$TMP_DIR/$ASSET" | tr 'A-F' 'a-f')
    [ "$actual" = "$expected" ] || die "SHA-256 mismatch for $ASSET; the download was discarded"
}

check_archive_paths() {
    tar -tzf "$TMP_DIR/$ASSET" | awk -v top="$ARCHIVE_TOP/" '
        BEGIN { bad = 0 }
        {
            if (index($0, top) != 1) { bad = 1; exit }
            count = split($0, part, "/")
            for (i = 1; i <= count; i++) {
                if (part[i] == "..") { bad = 1; exit }
            }
        }
        END { exit bad }
    ' || die "archive contains entries outside $ARCHIVE_TOP or parent traversal"
    tar -tvzf "$TMP_DIR/$ASSET" | awk '
        substr($0, 1, 1) != "-" && substr($0, 1, 1) != "d" { bad = 1 }
        END { exit bad }
    ' || die "archive contains links or other non-regular entries"
}

generate_token() {
    local token=""
    if command -v openssl >/dev/null 2>&1; then
        token=$(openssl rand -hex 32 2>/dev/null || true)
    fi
    if [[ ! "$token" =~ ^[0-9a-f]{64}$ ]]; then
        token=$(od -An -N32 -tx1 /dev/urandom 2>/dev/null | tr -d ' \n' || true)
    fi
    [[ "$token" =~ ^[0-9a-f]{64}$ ]] || die "could not generate a cryptographically random 256-bit settings_token"
    printf '%s\n' "$token"
}

create_initial_config() {
    local example=$1
    {
        generate_token
        printf '%s\n' "$CACHE_PATH"
    } > "$TMP_DIR/generated-values"
    chmod 600 "$TMP_DIR/generated-values"

    # Replace any example token/cache values, or insert them before the final
    # root brace. The token is read from a mode-0600 file, not command arguments.
    awk '
        function json_escape(value,    i, c, out) {
            out = ""
            for (i = 1; i <= length(value); i++) {
                c = substr(value, i, 1)
                if (c == "\\" || c == "\"") out = out "\\" c
                else out = out c
            }
            return out
        }
        function indent_of(line,    copy) {
            copy = line
            sub(/^[ \t]*/, "", copy)
            return substr(line, 1, length(line) - length(copy))
        }
        function replacement(line, key, value,    suffix) {
            suffix = (line ~ /,[ \t]*$/) ? "," : ""
            return indent_of(line) "\"" key "\": \"" json_escape(value) "\"" suffix
        }
        NR == FNR {
            if (FNR == 1) token = $0
            else if (FNR == 2) cache = $0
            next
        }
        {
            n++
            lines[n] = $0
            if ($0 ~ /^[ \t]*"settings_token"[ \t]*:/) {
                lines[n] = replacement($0, "settings_token", token)
                have_token = 1
            } else if ($0 ~ /^[ \t]*"weather_cache_path"[ \t]*:/) {
                lines[n] = replacement($0, "weather_cache_path", cache)
                have_cache = 1
            }
        }
        END {
            if (length(token) != 64 || length(cache) == 0) exit 1
            for (i = n; i >= 1; i--) {
                    if (lines[i] ~ /^[ \t]*}[ \t]*$/) { closing = i; break }
            }
            if (!closing) exit 1
            missing = (have_token ? 0 : 1) + (have_cache ? 0 : 1)
            if (missing) {
                previous = closing - 1
                while (previous > 0 && lines[previous] ~ /^[ \t]*$/) previous--
                if (previous < 1) exit 1
                if (lines[previous] !~ /,[ \t]*$/ && lines[previous] !~ /^[ \t]*{[ \t]*$/) {
                    lines[previous] = lines[previous] ","
                }
                indent = indent_of(lines[previous])
                closing_line = lines[closing]
                inserted = 0
                if (!have_token) {
                    inserted++
                    suffix = (inserted < missing) ? "," : ""
                    lines[closing + inserted - 1] = indent "\"settings_token\": \"" json_escape(token) "\"" suffix
                }
                if (!have_cache) {
                    inserted++
                    suffix = (inserted < missing) ? "," : ""
                    lines[closing + inserted - 1] = indent "\"weather_cache_path\": \"" json_escape(cache) "\"" suffix
                }
                lines[closing + inserted] = closing_line
                n += inserted
            }
            for (i = 1; i <= n; i++) print lines[i]
        }
    ' "$TMP_DIR/generated-values" "$example" > "$TMP_DIR/kdash.json" || \
        die "could not create configuration from kdash.json.example"
    chmod 600 "$TMP_DIR/kdash.json"
}

escape_sed_replacement() {
    printf '%s' "$1" | sed 's/[\\&|]/\\&/g'
}

systemd_path() {
    # Percent signs introduce systemd specifiers; quotes and backslashes need
    # escaping inside a quoted unit-file command path.
    printf '%s' "$1" | sed -e 's/%/%%/g' -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

render_linux_service() {
    local template=$1
    local binary config data config_dir
    binary=$(escape_sed_replacement "\"$(systemd_path "$BIN_PATH")\"")
    config=$(escape_sed_replacement "\"$(systemd_path "$CONFIG_PATH")\"")
    data=$(escape_sed_replacement "\"$(systemd_path "$DATA_DIR")\"")
    config_dir=$(escape_sed_replacement "\"$(systemd_path "$CONFIG_DIR")\"")
    sed -e "s|@KDASH_BINARY@|$binary|g" \
        -e "s|@KDASH_CONFIG@|$config|g" \
        -e "s|@KDASH_DATA_DIR@|$data|g" \
        -e "s|@KDASH_CONFIG_DIR@|$config_dir|g" \
        "$template" > "$TMP_DIR/kdash.service"
}

xml_escape() {
    printf '%s' "$1" | sed \
        -e 's/&/\&amp;/g' \
        -e 's/</\&lt;/g' \
        -e 's/>/\&gt;/g' \
        -e 's/"/\&quot;/g' \
        -e "s/'/\&apos;/g"
}

render_macos_agent() {
    local template=$1
    local binary config data stdout stderr
    binary=$(escape_sed_replacement "$(xml_escape "$BIN_PATH")")
    config=$(escape_sed_replacement "$(xml_escape "$CONFIG_PATH")")
    data=$(escape_sed_replacement "$(xml_escape "$DATA_DIR")")
    stdout=$(escape_sed_replacement "$(xml_escape "$LOG_DIR/kdash.stdout.log")")
    stderr=$(escape_sed_replacement "$(xml_escape "$LOG_DIR/kdash.stderr.log")")
    sed -e "s|@KDASH_BINARY@|$binary|g" \
        -e "s|@KDASH_CONFIG@|$config|g" \
        -e "s|@KDASH_DATA_DIR@|$data|g" \
        -e "s|@KDASH_STDOUT@|$stdout|g" \
        -e "s|@KDASH_STDERR@|$stderr|g" \
        "$template" > "$TMP_DIR/com.kdash.agent.plist"
}

health_url() {
    local address
    local host_name
    local port
    address=$(awk -F'"' '
        {
            for (i = 1; i <= NF; i++) {
                if ($i == "listen_addr") { print $(i + 2); exit }
            }
        }
    ' "$CONFIG_PATH")
    address=${address:-:8088}
    if [[ "$address" =~ ^\[(.*)\]:([0-9]+)$ ]]; then
        host_name=${BASH_REMATCH[1]}
        port=${BASH_REMATCH[2]}
    elif [[ "$address" =~ ^(.*):([0-9]+)$ ]]; then
        host_name=${BASH_REMATCH[1]}
        port=${BASH_REMATCH[2]}
    else
        return 1
    fi
    case "$host_name" in
        ""|0.0.0.0|::) host_name=127.0.0.1 ;;
    esac
    if [[ "$host_name" == *:* ]] && [[ "$host_name" != \[*\] ]]; then
        host_name="[$host_name]"
    fi
    printf 'http://%s:%s/api/status\n' "$host_name" "$port"
}

health_request() {
    local url=$1
    local destination=$2
    if [ "$DOWNLOAD_TOOL" = curl ]; then
        curl -fL --silent --show-error --connect-timeout 2 --max-time 3 -o "$destination" "$url" >/dev/null 2>&1
    else
        wget -q -T 3 -O "$destination" "$url" >/dev/null 2>&1
    fi
}

wait_for_health() {
    local url
    local attempt
    local expected_version
    url=$(health_url) || die "could not derive a health-check URL from listen_addr"
    expected_version=$("$BIN_PATH" -version) || die "could not execute the installed kdash binary"
    for attempt in $(seq 1 45); do
        if health_request "$url" "$TMP_DIR/health.json"; then
            if grep -Eq "\"version\"[[:space:]]*:[[:space:]]*\"${expected_version}\"" "$TMP_DIR/health.json"; then
                log "Health check passed: $url (version $expected_version)"
                return 0
            fi
            die "health endpoint at $url returned a different kdash version"
        fi
        sleep 1
    done
    die "health check failed at $url after 45 seconds; inspect the service logs (configuration tokens are not shown)"
}

backup_existing() {
    if [ -f "$BIN_PATH" ]; then
        HAD_BINARY=1
        cp -p -- "$BIN_PATH" "$BACKUP_DIR/kdash"
    fi
    if [ -f "$CONFIG_PATH" ]; then
        HAD_CONFIG=1
        cp -p -- "$CONFIG_PATH" "$BACKUP_DIR/kdash.json"
    fi
    if [ -f "$SERVICE_PATH" ]; then
        HAD_SERVICE=1
        cp -p -- "$SERVICE_PATH" "$BACKUP_DIR/$SERVICE_NAME"
    fi
    if [ -d "$DOC_DIR" ]; then
        HAD_DOC_DIR=1
        cp -Rp -- "$DOC_DIR" "$BACKUP_DIR/doc"
    fi
    [ ! -d "$DATA_DIR" ] || HAD_DATA_DIR=1
    if [ "$OS" = "linux" ]; then
        if systemctl is-active --quiet kdash.service; then WAS_RUNNING=1; fi
        if systemctl is-enabled --quiet kdash.service; then WAS_ENABLED=1; fi
    else
        if launchctl print "gui/$(id -u)/com.kdash.agent" >/dev/null 2>&1; then WAS_RUNNING=1; fi
    fi
}

restore_backup_file() {
    local backup=$1
    local destination=$2
    local existed=$3
    if [ "$existed" -eq 1 ]; then
        mkdir -p "$(dirname "$destination")"
        cp -p -- "$backup" "$destination"
    else
        rm -f -- "$destination"
    fi
}

rollback_install() {
    printf '%s\n' "Installation failed; restoring the previous installation..." >&2
    set +e
    stop_resident_service
    restore_backup_file "$BACKUP_DIR/kdash" "$BIN_PATH" "$HAD_BINARY"
    restore_backup_file "$BACKUP_DIR/kdash.json" "$CONFIG_PATH" "$HAD_CONFIG"
    restore_backup_file "$BACKUP_DIR/$SERVICE_NAME" "$SERVICE_PATH" "$HAD_SERVICE"
    if [ "$HAD_DOC_DIR" -eq 1 ]; then
        rm -rf -- "$DOC_DIR"
        mkdir -p "$(dirname "$DOC_DIR")"
        cp -Rp -- "$BACKUP_DIR/doc" "$DOC_DIR"
    else
        rm -rf -- "$DOC_DIR"
    fi
    if [ "$HAD_DATA_DIR" -eq 0 ]; then
        rm -rf -- "$DATA_DIR"
    fi
    if [ "$OS" = "linux" ]; then
        systemctl daemon-reload >/dev/null 2>&1
        if [ "$WAS_ENABLED" -eq 0 ]; then systemctl disable kdash.service >/dev/null 2>&1; fi
        if [ "$WAS_RUNNING" -eq 1 ] && [ "$HAD_SERVICE" -eq 1 ]; then
            systemctl start kdash.service >/dev/null 2>&1
        fi
    elif [ "$WAS_RUNNING" -eq 1 ] && [ "$HAD_SERVICE" -eq 1 ]; then
        launchctl bootstrap "gui/$(id -u)" "$SERVICE_PATH" >/dev/null 2>&1 || \
            launchctl load -w "$SERVICE_PATH" >/dev/null 2>&1
    fi
    printf '%s\n' "Rollback finished. The installer did not print or log the settings token." >&2
}

prepare_linux_account() {
    local nologin
    if ! getent group kdash >/dev/null 2>&1; then
        command -v groupadd >/dev/null 2>&1 || die "groupadd is required to create the kdash group"
        groupadd --system kdash
    fi
    if ! id kdash >/dev/null 2>&1; then
        command -v useradd >/dev/null 2>&1 || die "useradd is required to create the kdash user"
        if [ -x /usr/sbin/nologin ]; then nologin=/usr/sbin/nologin; else nologin=/sbin/nologin; fi
        useradd --system --gid kdash --home-dir "$DATA_DIR" --no-create-home --shell "$nologin" kdash
    fi
}

install_files() {
    local extracted="$TMP_DIR/extracted/$ARCHIVE_TOP"
    mkdir -p "$BIN_DIR" "$DOC_DIR" "$CONFIG_DIR" "$DATA_DIR"
    cp -- "$extracted/kdash" "$BIN_DIR/.kdash.new.$$"
    chmod 755 "$BIN_DIR/.kdash.new.$$"
    mv -f -- "$BIN_DIR/.kdash.new.$$" "$BIN_PATH"

    cp -- "$extracted/kdash.json.example" "$DOC_DIR/kdash.json.example"
    cp -- "$extracted/LICENSE" "$DOC_DIR/LICENSE"
    cp -- "$extracted/THIRD_PARTY_NOTICES.md" "$DOC_DIR/THIRD_PARTY_NOTICES.md"
    cp -- "$extracted/README.md" "$DOC_DIR/README.md"
    cp -- "$extracted/RELEASES.md" "$DOC_DIR/RELEASES.md"
    cp -R -- "$extracted/assets" "$DOC_DIR/"
    chmod 644 "$DOC_DIR/kdash.json.example" "$DOC_DIR/LICENSE" "$DOC_DIR/THIRD_PARTY_NOTICES.md"

    if [ "$HAD_CONFIG" -eq 0 ]; then
        create_initial_config "$extracted/kdash.json.example"
        cp -- "$TMP_DIR/kdash.json" "$CONFIG_PATH.new"
        mv -f -- "$CONFIG_PATH.new" "$CONFIG_PATH"
    fi
    chmod 600 "$CONFIG_PATH"

    if [ "$OS" = "linux" ]; then
        prepare_linux_account
        chown root:kdash "$CONFIG_DIR"
        chmod 770 "$CONFIG_DIR"
        chown kdash:kdash "$CONFIG_PATH" "$DATA_DIR"
        chmod 750 "$DATA_DIR"
        render_linux_service "$extracted/kdash.service"
        cp -- "$TMP_DIR/kdash.service" "$SERVICE_PATH.new"
        chmod 644 "$SERVICE_PATH.new"
        mv -f -- "$SERVICE_PATH.new" "$SERVICE_PATH"
    else
        mkdir -p "$LOG_DIR" "$(dirname "$SERVICE_PATH")"
        chmod 700 "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
        touch "$LOG_DIR/kdash.stdout.log" "$LOG_DIR/kdash.stderr.log"
        chmod 600 "$LOG_DIR/kdash.stdout.log" "$LOG_DIR/kdash.stderr.log"
        render_macos_agent "$extracted/com.kdash.agent.plist"
        cp -- "$TMP_DIR/com.kdash.agent.plist" "$SERVICE_PATH.new"
        chmod 600 "$SERVICE_PATH.new"
        mv -f -- "$SERVICE_PATH.new" "$SERVICE_PATH"
    fi
}

start_installed_service() {
    if [ "$OS" = "linux" ]; then
        systemctl daemon-reload
        systemctl enable kdash.service
        if [ "$NO_START" -eq 0 ]; then systemctl start kdash.service; fi
    elif [ "$NO_START" -eq 0 ]; then
        launchctl bootstrap "gui/$(id -u)" "$SERVICE_PATH" 2>/dev/null || \
            launchctl load -w "$SERVICE_PATH"
    fi
}

main() {
    local extracted
    if [ "$NON_INTERACTIVE" -eq 1 ]; then
        log "Running non-interactively; the installer will not prompt for input."
    fi
    log "Downloading $ASSET ($VERSION)"
    fetch_file SHA256SUMS "$TMP_DIR/SHA256SUMS"
    fetch_file "$ASSET" "$TMP_DIR/$ASSET"
    verify_archive
    check_archive_paths
    mkdir -p "$TMP_DIR/extracted"
    tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR/extracted"
    extracted="$TMP_DIR/extracted/$ARCHIVE_TOP"
    [ -f "$extracted/kdash" ] || die "release archive is missing kdash"
    [ -f "$extracted/kdash.json.example" ] || die "release archive is missing kdash.json.example"
    [ -f "$extracted/LICENSE" ] || die "release archive is missing LICENSE"
    [ -f "$extracted/THIRD_PARTY_NOTICES.md" ] || die "release archive is missing THIRD_PARTY_NOTICES.md"
    [ -f "$extracted/README.md" ] || die "release archive is missing README.md"
    [ -f "$extracted/RELEASES.md" ] || die "release archive is missing RELEASES.md"
    [ -f "$extracted/assets/fonts/licenses/NotoSansSC-OFL.txt" ] || die "release archive is missing font licenses"
    [ -f "$extracted/$SERVICE_NAME" ] || die "release archive is missing $SERVICE_NAME"

    backup_existing
    TRANSACTION_ACTIVE=1
    stop_resident_service
    install_files
    start_installed_service
    if [ "$NO_START" -eq 0 ]; then wait_for_health; fi
    INSTALL_SUCCEEDED=1
    TRANSACTION_ACTIVE=0
    log "kdash was installed successfully at $BIN_PATH"
    log "Configuration: $CONFIG_PATH"
    log "Data: $DATA_DIR"
    if [ "$NO_START" -eq 1 ]; then
        if [ "$OS" = "linux" ]; then
            log "The systemd service is enabled but was not started (--no-start)."
        else
            log "The LaunchAgent was installed but not loaded (--no-start); it will run at the next login."
        fi
    fi
}

main "$@"
