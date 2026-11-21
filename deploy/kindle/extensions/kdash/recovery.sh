#!/bin/sh
# Emergency recovery: remove all kdash/USBNetwork boot jobs so the Kindle
# boots with factory startup behavior. Run once from KUAL or any shell,
# then reboot. Output is ASCII only (KUAL display safety).
KD=/mnt/us/kdash
LOG=$KD/emergency-recovery.log
mkdir -p "$KD" 2>/dev/null
{
echo "=== emergency recovery $(date) ==="
initctl stop kdash >/dev/null 2>&1 || true
pkill -f "/mnt/us/kdash/kdash" 2>/dev/null || true
touch "$KD/DISABLED"
mkdir -p "$KD/backups"
STAMP=$(date +%Y%m%d-%H%M%S)
mount -o remount,rw /
if [ -f /etc/upstart/usbnet.conf ]; then
    cp /etc/upstart/usbnet.conf "$KD/backups/usbnet.conf.$STAMP"
    rm -f /etc/upstart/usbnet.conf
    echo "removed /etc/upstart/usbnet.conf"
fi
if [ -f /etc/upstart/kdash.conf ]; then
    cp /etc/upstart/kdash.conf "$KD/backups/kdash.conf.$STAMP"
    rm -f /etc/upstart/kdash.conf
    echo "removed /etc/upstart/kdash.conf"
fi
mount -o remount,ro /
rm -f /mnt/us/usbnet/auto
initctl reload-configuration >/dev/null 2>&1 || true
echo "done - reboot now; boot config is back to factory behavior"
} > "$LOG" 2>&1
cat "$LOG"
