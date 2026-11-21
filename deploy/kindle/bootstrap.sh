#!/bin/sh
# USB first-install bootstrap: prepare/keep USBNetwork, inject the SSH public
# key, then run the standard kdash installer. Run from KUAL on the Kindle.
# Output is ASCII only (KUAL display safety).
SRC=/mnt/us/kdash-bootstrap
KD=/mnt/us/kdash
LOG="$SRC/bootstrap.log"

{
echo "=== kdash bootstrap $(date) ==="

echo "--- [1/6] prepare kdash client files ---"
mkdir -p "$KD"
if [ ! -f "$KD/client.conf" ] && [ -f "$SRC/client.conf" ]; then
    cp "$SRC/client.conf" "$KD/client.conf"
fi
for FILE in sync-screen.sh install.sh run.sh restore.sh; do
    if [ -f "$SRC/$FILE" ]; then
        cp "$SRC/$FILE" "$KD/$FILE"
    fi
done
chmod 755 "$KD/sync-screen.sh" "$KD/install.sh" "$KD/run.sh" "$KD/restore.sh" 2>/dev/null || true

echo "--- [2/6] install USBNetwork runtime (keep existing) ---"
if [ ! -d /mnt/us/usbnet ] && [ -d "$SRC/usbnet" ]; then
    cp -r "$SRC/usbnet" /mnt/us/usbnet
fi
if [ -d /mnt/us/usbnet ]; then
    chmod 755 /mnt/us/usbnet/bin/usbnetwork /mnt/us/usbnet/bin/usbnetwork.sh /mnt/us/usbnet/bin/libkh5 2>/dev/null || true
    # 永不启用 USBNetwork 的开机 auto；它曾导致 PW2 卡在启动进度条。
    rm -f /mnt/us/usbnet/auto
    touch /mnt/us/usbnet/DISABLED_auto
    if [ -f /mnt/us/usbnet/etc/config ]; then
        sed -i 's/^USE_WIFI="false"/USE_WIFI="true"/' /mnt/us/usbnet/etc/config
        grep '^USE_WIFI' /mnt/us/usbnet/etc/config 2>/dev/null || true
    fi
fi

echo "--- [3/6] install KUAL extensions (never overwrite others) ---"
mkdir -p /mnt/us/extensions
if [ ! -d /mnt/us/extensions/usbnet ] && [ -d "$SRC/extensions/usbnet" ]; then
    cp -r "$SRC/extensions/usbnet" /mnt/us/extensions/usbnet
fi
mkdir -p /mnt/us/extensions/kdash
if [ -d "$SRC/extensions/kdash" ]; then
    cp "$SRC/extensions/kdash/"* /mnt/us/extensions/kdash/
fi
chmod 755 /mnt/us/extensions/usbnet/bin/usbnet.sh /mnt/us/extensions/kdash/*.sh 2>/dev/null || true

echo "--- [4/6] register SSH public key ---"
if [ -f "$SRC/pubkey.txt" ]; then
    mount -o remount,rw / 2>/dev/null || true
    mkdir -p /root/.ssh
    touch /root/.ssh/authorized_keys
    if ! grep -F -q -f "$SRC/pubkey.txt" /root/.ssh/authorized_keys; then
        cat "$SRC/pubkey.txt" >> /root/.ssh/authorized_keys
    fi
    chmod 700 /root/.ssh
    chmod 600 /root/.ssh/authorized_keys
    echo "SSH public key installed"
    mount -o remount,ro / 2>/dev/null || true
fi
initctl reload-configuration >/dev/null 2>&1 || true

echo "--- [5/6] install and start kdash client ---"
sh "$KD/install.sh"

echo "--- [6/6] NOTE: boot-time USBNetwork not enabled ---"
echo "Use the kdash maintenance menu to start Wi-Fi SSH on port 2222."

echo "=== bootstrap done; kdash logs live in /var/tmp/kdash-client.log ==="
} > "$LOG" 2>&1

cat "$LOG"
echo "(log saved to $LOG)"
