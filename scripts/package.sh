#!/bin/sh
# Build and package kdash without active configuration, diagnostics, or old
# build artifacts. A fresh staging tree replaces dist only after all checks pass.
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
KDASH_ROOT=$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)
DIST_DIR="$KDASH_ROOT/dist"
VERSION=$(cat "$KDASH_ROOT/VERSION")
KUAL_CONFIG="$KDASH_ROOT/deploy/kindle/extensions/kdash/config.xml"
WORK_DIR=
BACKUP_DIR=

cleanup() {
    status=$?
    trap - EXIT HUP INT TERM
    if [ -n "$WORK_DIR" ] && [ -d "$WORK_DIR" ]; then
        rm -rf "$WORK_DIR"
    fi
    if [ -n "$BACKUP_DIR" ] && { [ -d "$BACKUP_DIR" ] || [ -L "$BACKUP_DIR" ]; }; then
        if [ ! -e "$DIST_DIR" ] && [ ! -L "$DIST_DIR" ]; then
            mv "$BACKUP_DIR" "$DIST_DIR"
        else
            echo "Previous dist retained at $BACKUP_DIR" >&2
        fi
    fi
    exit "$status"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

fail() {
    echo "package.sh: $*" >&2
    exit 1
}

require_file() {
    [ -f "$1" ] || fail "required file is missing: $1"
}

require_tools() {
    command -v zip >/dev/null 2>&1 || fail "zip is required"
    command -v tar >/dev/null 2>&1 || fail "tar is required"
    if command -v sha256sum >/dev/null 2>&1; then
        SHA_TOOL=sha256sum
    elif command -v shasum >/dev/null 2>&1; then
        SHA_TOOL=shasum
    else
        fail "sha256sum or shasum is required"
    fi
}

write_checksums() {
    if [ "$SHA_TOOL" = sha256sum ]; then
        sha256sum "$@"
    else
        shasum -a 256 "$@"
    fi
}

verify_checksums() {
    if [ "$SHA_TOOL" = sha256sum ]; then
        sha256sum -c SHA256SUMS
    else
        shasum -a 256 -c SHA256SUMS
    fi
}

copy_license_files() {
    cp "$KDASH_ROOT/LICENSE" "$1/LICENSE"
    cp "$KDASH_ROOT/THIRD_PARTY_NOTICES.md" "$1/THIRD_PARTY_NOTICES.md"
    mkdir -p "$1/assets/fonts/licenses"
    for license_path in "$KDASH_ROOT"/assets/fonts/licenses/*.txt; do
        require_file "$license_path"
        cp "$license_path" "$1/assets/fonts/licenses/"
    done
}

make_zip() {
    archive_name=$1
    package_root=$2
    (cd "$PACKAGE_STAGE" && zip -q -r "$OUT_DIR/$archive_name" "$package_root")
    zip -T -q "$OUT_DIR/$archive_name"
}

make_tar() {
    archive_name=$1
    package_root=$2
    tar -czf "$OUT_DIR/$archive_name" -C "$PACKAGE_STAGE" "$package_root"
    tar -tzf "$OUT_DIR/$archive_name" >/dev/null
}

cd "$KDASH_ROOT"
require_tools

[ -n "$VERSION" ] || fail "VERSION is empty"
KUAL_VERSION=$(sed -n 's:.*<version>\([^<]*\)</version>.*:\1:p' "$KUAL_CONFIG" | head -n 1)
[ "$KUAL_VERSION" = "$VERSION" ] || \
    fail "VERSION ($VERSION) does not match KUAL config.xml ($KUAL_VERSION)"

# Validate every source asset before compiling; never substitute active config.
require_file "$KDASH_ROOT/configs/kdash.json.example"
require_file "$KDASH_ROOT/LICENSE"
require_file "$KDASH_ROOT/THIRD_PARTY_NOTICES.md"
require_file "$KDASH_ROOT/deploy/linux/kdash.service"
require_file "$KDASH_ROOT/deploy/macos/com.kdash.agent.plist"
require_file "$KDASH_ROOT/deploy/kindle/client.conf.example"
require_file "$KUAL_CONFIG"
for script_name in install.sh restore.sh run.sh sync-screen.sh; do
    require_file "$KDASH_ROOT/deploy/kindle/$script_name"
done
for script_name in start.bat stop.bat start_service.vbs install_task.bat install_task.ps1 uninstall_task.bat uninstall_task.ps1; do
    require_file "$KDASH_ROOT/deploy/win11/$script_name"
done

echo "==> Building tested release binaries"
"$KDASH_ROOT/scripts/build.sh"
for binary_name in kdash-windows-amd64.exe kdash-darwin-arm64 kdash-linux-amd64 kdash-linux-arm64; do
    require_file "$KDASH_ROOT/build/$binary_name"
done

WORK_DIR=$(mktemp -d "$KDASH_ROOT/.package.staging.XXXXXX")
PACKAGE_STAGE="$WORK_DIR/packages"
OUT_DIR="$WORK_DIR/dist"
mkdir -p "$PACKAGE_STAGE" "$OUT_DIR"

for package_name in kdash-windows-amd64 kdash-darwin-arm64 kdash-linux-amd64 kdash-linux-arm64; do
    mkdir -p "$PACKAGE_STAGE/$package_name"
    cp "$KDASH_ROOT/configs/kdash.json.example" "$PACKAGE_STAGE/$package_name/kdash.json.example"
    copy_license_files "$PACKAGE_STAGE/$package_name"
done
mkdir -p "$PACKAGE_STAGE/kdash-kindle-pw2"
copy_license_files "$PACKAGE_STAGE/kdash-kindle-pw2"

echo "==> Packaging Windows amd64"
cp "$KDASH_ROOT/build/kdash-windows-amd64.exe" "$PACKAGE_STAGE/kdash-windows-amd64/kdash.exe"
sed 's|](../releases.md)|](RELEASES.md)|g' "$KDASH_ROOT/docs/install/windows.md" > "$PACKAGE_STAGE/kdash-windows-amd64/README.md"
cp "$KDASH_ROOT/docs/releases.md" "$PACKAGE_STAGE/kdash-windows-amd64/RELEASES.md"
for source_path in "$KDASH_ROOT"/deploy/win11/*; do
    case "${source_path##*/}" in
        *.bat|*.ps1|*.vbs)
            [ ! -f "$source_path" ] || cp "$source_path" "$PACKAGE_STAGE/kdash-windows-amd64/"
            ;;
    esac
done
make_zip kdash-windows-amd64.zip kdash-windows-amd64

echo "==> Packaging macOS arm64"
cp "$KDASH_ROOT/build/kdash-darwin-arm64" "$PACKAGE_STAGE/kdash-darwin-arm64/kdash"
cp "$KDASH_ROOT/deploy/macos/com.kdash.agent.plist" "$PACKAGE_STAGE/kdash-darwin-arm64/"
sed 's|](../releases.md)|](RELEASES.md)|g' "$KDASH_ROOT/docs/install/macos.md" > "$PACKAGE_STAGE/kdash-darwin-arm64/README.md"
cp "$KDASH_ROOT/docs/releases.md" "$PACKAGE_STAGE/kdash-darwin-arm64/RELEASES.md"
make_tar kdash-darwin-arm64.tar.gz kdash-darwin-arm64

echo "==> Packaging Linux amd64 and arm64"
for package_name in kdash-linux-amd64 kdash-linux-arm64; do
    cp "$KDASH_ROOT/build/$package_name" "$PACKAGE_STAGE/$package_name/kdash"
    cp "$KDASH_ROOT/deploy/linux/kdash.service" "$PACKAGE_STAGE/$package_name/"
    sed 's|](../releases.md)|](RELEASES.md)|g' "$KDASH_ROOT/docs/install/linux.md" > "$PACKAGE_STAGE/$package_name/README.md"
    cp "$KDASH_ROOT/docs/releases.md" "$PACKAGE_STAGE/$package_name/RELEASES.md"
    make_tar "$package_name.tar.gz" "$package_name"
done

echo "==> Packaging Kindle PW2 Shell client and KUAL extension"
KINDLE_STAGE="$PACKAGE_STAGE/kdash-kindle-pw2"
mkdir -p "$KINDLE_STAGE/kdash" "$KINDLE_STAGE/extensions"
for script_name in install.sh restore.sh run.sh sync-screen.sh; do
    cp "$KDASH_ROOT/deploy/kindle/$script_name" "$KINDLE_STAGE/kdash/"
done
cp "$KDASH_ROOT/deploy/kindle/client.conf.example" "$KINDLE_STAGE/kdash/client.conf"
cp -R "$KDASH_ROOT/deploy/kindle/extensions/kdash" "$KINDLE_STAGE/extensions/"
cp "$KDASH_ROOT/docs/install/kindle-pw2.md" "$KINDLE_STAGE/README.md"
cp "$KDASH_ROOT/docs/releases.md" "$KINDLE_STAGE/RELEASES.md"
make_zip kdash-kindle-pw2.zip kdash-kindle-pw2

cp "$KDASH_ROOT/scripts/install.sh" "$OUT_DIR/install.sh"
cp "$KDASH_ROOT/scripts/install.ps1" "$OUT_DIR/install.ps1"
chmod 755 "$OUT_DIR/install.sh"

echo "==> Generating and verifying SHA256SUMS"
(cd "$OUT_DIR" && write_checksums \
    kdash-windows-amd64.zip \
    kdash-darwin-arm64.tar.gz \
    kdash-linux-amd64.tar.gz \
    kdash-linux-arm64.tar.gz \
    kdash-kindle-pw2.zip \
    > SHA256SUMS)
(cd "$OUT_DIR" && verify_checksums)
set -- "$OUT_DIR"/*
[ "$#" = 8 ] || fail "unexpected number of release files"

# Publish on the same filesystem. The old dist remains untouched until every
# archive and checksum has been created and verified successfully.
if [ -e "$DIST_DIR" ] || [ -L "$DIST_DIR" ]; then
    BACKUP_DIR="$KDASH_ROOT/.dist.previous.$$"
    [ ! -e "$BACKUP_DIR" ] && [ ! -L "$BACKUP_DIR" ] || fail "backup path already exists: $BACKUP_DIR"
    mv "$DIST_DIR" "$BACKUP_DIR"
fi
if ! mv "$OUT_DIR" "$DIST_DIR"; then
    if [ -n "$BACKUP_DIR" ] && [ -d "$BACKUP_DIR" ] && [ ! -e "$DIST_DIR" ]; then
        mv "$BACKUP_DIR" "$DIST_DIR"
        BACKUP_DIR=
    fi
    fail "could not install the new dist directory"
fi
if [ -n "$BACKUP_DIR" ]; then
    rm -rf "$BACKUP_DIR"
    BACKUP_DIR=
fi

echo "==> Release packages are ready in $DIST_DIR"
ls -lh "$DIST_DIR"
cat "$DIST_DIR/SHA256SUMS"
