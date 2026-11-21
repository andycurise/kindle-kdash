#!/bin/sh
KD=/mnt/us/kdash
PIDFILE=/var/tmp/kdash.pid

touch "$KD/DISABLED"

# 停止 Upstart 自启守护任务（如果存在）
initctl stop kdash >/dev/null 2>&1 || true

if [ -f "$PIDFILE" ]; then
    PID=$(cat "$PIDFILE" 2>/dev/null)
    if [ -n "$PID" ]; then
        kill "$PID" 2>/dev/null || true
        sleep 1
        kill -9 "$PID" 2>/dev/null || true
    fi
    rm -f "$PIDFILE"
fi

killall sync-screen.sh 2>/dev/null || true
pkill -f "sync-screen.sh" 2>/dev/null || true

# 清理运行时临时状态标记，避免下次启动被旧状态误导
rm -f /var/tmp/kdash-touch-active /var/tmp/screen_current.png /var/tmp/screen_new.png

sh "$KD/restore.sh"
exit 0
