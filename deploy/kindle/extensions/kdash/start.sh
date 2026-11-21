#!/bin/sh
# 启动日历看板客户端
KD=/mnt/us/kdash
PIDFILE=/var/tmp/kdash.pid

# 1. 终止可能正在运行的旧客户端（不调用 restore.sh，防止把原生主页刷回屏幕造成闪烁与冲突）
if [ -f "$PIDFILE" ]; then
    PID=$(cat "$PIDFILE" 2>/dev/null)
    [ -n "$PID" ] && kill -9 "$PID" 2>/dev/null || true
fi
killall sync-screen.sh 2>/dev/null || true
pkill -f "sync-screen.sh" 2>/dev/null || true

# 2. 彻底清除停止与暂停标记，清除上一次渲染的缓存图，确保启动必刷屏
rm -f "$KD/DISABLED" "$PIDFILE" /var/tmp/kdash-touch-active /var/tmp/screen_current.png /var/tmp/screen_new.png

# 3. 唤醒并保持 Wi-Fi 与防休眠
lipc-set-prop com.lab126.cmd wirelessEnable 1 2>/dev/null || true
lipc-set-prop com.lab126.powerd preventScreenSaver 1 2>/dev/null || true
iwconfig wlan0 power off 2>/dev/null || true

# 4. 隐藏状态栏并让原生桌面退至后台，避免 KUAL 菜单遮挡
lipc-set-prop com.lab126.pillow interrogatePillow '{"pillowId": "default_status_bar", "function": "nativeBridge.hideMe();"}' 2>/dev/null || true
lipc-set-prop com.lab126.appmgrd start app://com.lab126.booklet.home >/dev/null 2>&1 || true

# 5. 立即同步拉取并强制刷屏一次（所见即所得，瞬间呈现看板，不再等待下一分钟整点）
sh "$KD/sync-screen.sh" once >/dev/null 2>&1 || true

# 6. 后台启动常驻循环同步进程
sh "$KD/sync-screen.sh" </dev/null >/dev/null 2>&1 &

sleep 1
sync
exit 0
