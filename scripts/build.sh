#!/bin/sh
# Test first, then build the four supported server targets plus a native
# development preview binary. Kindle PW2 intentionally has no Go target.
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
KDASH_ROOT=$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)
BUILD_DIR="$KDASH_ROOT/build"
GO_BIN=${GO_BIN:-go}
VERSION=$(cat "$KDASH_ROOT/VERSION")

case "$VERSION" in
    ""|*[!0-9.]*)
        echo "VERSION must contain a numeric dotted version: $VERSION" >&2
        exit 1
        ;;
esac

cd "$KDASH_ROOT"
mkdir -p "$BUILD_DIR"

build_target() {
    target_os=$1
    target_arch=$2
    target_output=$3
    target_ldflags="-X main.version=$VERSION"

    # Optional Windows symbol stripping; version injection stays identical.
    if [ "$target_os" = windows ] && [ "${KDASH_WINDOWS_STRIP:-0}" = 1 ]; then
        target_ldflags="-s -w $target_ldflags"
    fi

    echo "==> Building $target_os/$target_arch -> ${target_output#"$KDASH_ROOT"/}"
    GOOS="$target_os" GOARCH="$target_arch" CGO_ENABLED=0 \
        "$GO_BIN" build \
        -trimpath \
        -buildvcs=false \
        -ldflags "$target_ldflags" \
        -o "$target_output" \
        ./cmd/kdash
}

echo "==> [1/3] Running the complete Go test suite"
"$GO_BIN" test ./...

echo "==> [2/3] Building the four formal release targets"
build_target windows amd64 "$BUILD_DIR/kdash-windows-amd64.exe"
build_target darwin arm64 "$BUILD_DIR/kdash-darwin-arm64"
build_target linux amd64 "$BUILD_DIR/kdash-linux-amd64"
build_target linux arm64 "$BUILD_DIR/kdash-linux-arm64"

echo "==> [3/3] Building the native development preview binary"
host_os=$("$GO_BIN" env GOOS)
host_arch=$("$GO_BIN" env GOARCH)
build_target "$host_os" "$host_arch" "$BUILD_DIR/kdash-dev"

echo "==> Build complete (kdash $VERSION):"
ls -lh \
    "$BUILD_DIR/kdash-windows-amd64.exe" \
    "$BUILD_DIR/kdash-darwin-arm64" \
    "$BUILD_DIR/kdash-linux-amd64" \
    "$BUILD_DIR/kdash-linux-arm64" \
    "$BUILD_DIR/kdash-dev"
