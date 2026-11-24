#!/bin/sh
# 恢复定时背光模式 (白天亮/夜间灭)
CONF=/mnt/us/kdash/client.conf
if [ -f "$CONF" ]; then
    if grep -q "^[ \t]*FRONTLIGHT_MODE=" "$CONF"; then
        sed -i 's/^[ \t]*FRONTLIGHT_MODE=.*/FRONTLIGHT_MODE="schedule"/' "$CONF"
    else
        echo 'FRONTLIGHT_MODE="schedule"' >> "$CONF"
    fi
fi
rm -f /var/tmp/kdash-remote.conf

# 同步通知服务端（如果在线）
. /mnt/us/extensions/kdash/frontlight-api.sh
frontlight_notify schedule || true

# 立即根据当前定时策略执行一次刷新
sh /mnt/us/kdash/sync-screen.sh once >/dev/null 2>&1 || true
