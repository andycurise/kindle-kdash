#!/bin/sh
# 常亮背光 (亮度 10)
CONF=/mnt/us/kdash/client.conf
if [ -f "$CONF" ]; then
    if grep -q "^[ \t]*FRONTLIGHT_MODE=" "$CONF"; then
        sed -i 's/^[ \t]*FRONTLIGHT_MODE=.*/FRONTLIGHT_MODE="on"/' "$CONF"
    else
        echo 'FRONTLIGHT_MODE="on"' >> "$CONF"
    fi
fi

# 脉冲触发恢复背光，避免底层因之前写 0 而未被 powerd 刷新
lipc-set-prop -i com.lab126.powerd flIntensity 0 2>/dev/null || true
lipc-set-prop -i com.lab126.powerd flIntensity 10 2>/dev/null || true

# 同步通知服务端（如果在线）
. /mnt/us/extensions/kdash/frontlight-api.sh
frontlight_notify on || true
