#!/bin/sh
# kdash 安装/修复脚本 —— 在 Kindle 上以 root 运行。
# 纯 Shell 极简架构：安装轻量级同步客户端与 upstart 开机自启服务。
set -e
KD=/mnt/us/kdash

[ -f "$KD/sync-screen.sh" ] || { echo "错误: 缺少 $KD/sync-screen.sh"; exit 1; }
[ -f "$KD/run.sh" ] || { echo "错误: 缺少 $KD/run.sh"; exit 1; }
[ -f "$KD/restore.sh" ] || { echo "错误: 缺少 $KD/restore.sh"; exit 1; }

# 如果 client.conf 不存在，尝试从备选或默认生成
if [ ! -f "$KD/client.conf" ]; then
    if [ -f "$KD/client.conf.example" ]; then
        cp "$KD/client.conf.example" "$KD/client.conf"
    else
        cat > "$KD/client.conf" <<'EOF'
# Kindle 日历看板客户端配置
SERVER_URL="http://192.168.1.99:8088/screen.png"
REFRESH_SECONDS=60
GHOST_CLEAR_SECONDS=3600
TOUCH_PAUSE_SECONDS=180
FRONTLIGHT_MODE="schedule"
DAY_BRIGHTNESS=10
NIGHT_BRIGHTNESS=0
DAY_START_HOUR=7
NIGHT_START_HOUR=22
EOF
    fi
    echo "已生成默认配置文件: $KD/client.conf"
fi

chmod 755 "$KD/sync-screen.sh" "$KD/run.sh" "$KD/restore.sh"
mkdir -p "$KD/backups"

# ---- USBNetwork SSH symlinks（为 dropbear 补齐常用软链接） ----
USBNET_BINDIR=/mnt/us/usbnet/bin
if [ -x "$USBNET_BINDIR/dropbearmulti" ]; then
    mount -o remount,rw / 2>/dev/null || true
    for var in /usr/sbin/dropbearmulti /usr/bin/dropbear /usr/bin/dbclient /usr/bin/dropbearkey /usr/bin/dropbearconvert /usr/bin/dbscp; do
        if [ ! -e "$var" ] && [ ! -L "$var" ]; then
            ln -fs "$USBNET_BINDIR/dropbearmulti" "$var"
            echo "创建符号链接: $var"
        fi
    done
    mount -o remount,ro / 2>/dev/null || true
fi

# ---- Upstart 守护进程配置 ----
STAMP=$(date +%Y%m%d-%H%M%S)
if [ -f /etc/upstart/kdash.conf ]; then
    cp /etc/upstart/kdash.conf "$KD/backups/kdash.conf.$STAMP"
    echo "已备份旧的 /etc/upstart/kdash.conf"
fi

cat > "$KD/kdash.conf.new" <<'EOF'
description "kdash calendar dashboard client"
author "kdash"
start on started framework
stop on shutdown
respawn
respawn limit 5 60
normal exit 0
script
    exec sh /mnt/us/kdash/run.sh >> /var/tmp/kdash-client.log 2>&1
end script
EOF

mount -o remount,rw / 2>/dev/null || true
cp "$KD/kdash.conf.new" /etc/upstart/kdash.conf
mount -o remount,ro / 2>/dev/null || true
rm -f "$KD/kdash.conf.new"
initctl reload-configuration >/dev/null 2>&1 || true

# 若已有运行中的任务，重启它
if initctl status kdash 2>/dev/null | grep -q '^kdash start/running'; then
    initctl stop kdash >/dev/null 2>&1 || true
    sleep 1
fi

rm -f "$KD/DISABLED"
initctl start kdash >/dev/null 2>&1 || nohup sh "$KD/run.sh" >/dev/null 2>&1 &

echo "kdash 客户端安装完成 (已配置开机自启与异常守护)"
