#!/bin/sh
# 彻底关闭 Kindle 前光/背光
CONF=/mnt/us/kdash/client.conf
if [ -f "$CONF" ]; then
    if grep -q "^[ \t]*FRONTLIGHT_MODE=" "$CONF"; then
        sed -i 's/^[ \t]*FRONTLIGHT_MODE=.*/FRONTLIGHT_MODE="off"/' "$CONF"
    else
        echo 'FRONTLIGHT_MODE="off"' >> "$CONF"
    fi
fi

# 1. 通知 powerd 框架
lipc-set-prop -i com.lab126.powerd flIntensity 0 2>/dev/null || true

# 2. 彻底关闭背光：在 Kindle PW2/PW3 等设备上强制向底层 sysfs 写 0 熄灭微光
for f in /sys/class/backlight/*/brightness /sys/devices/system/fl_tps6116x/fl_tps6116x0/fl_intensity; do
    if [ -w "$f" ]; then
        echo 0 > "$f" 2>/dev/null || true
    fi
done

# 3. 同步通知服务端（如果在线）
. /mnt/us/extensions/kdash/frontlight-api.sh
frontlight_notify off || true
