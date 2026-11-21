#!/bin/sh
# 看板运行状态查询脚本
KD=/mnt/us/kdash
PIDFILE=/var/tmp/kdash.pid
LOG="$KD/status.log"

{
    echo "=========================================="
    echo "kdash 状态检查 $(date '+%Y-%m-%d %H:%M:%S')"
    echo "=========================================="

    # 1. 进程状态
    RUNNING=0
    if [ -f "$PIDFILE" ]; then
        PID=$(cat "$PIDFILE" 2>/dev/null)
        if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
            echo "[状态] 正在运行 (PID: $PID)"
            RUNNING=1
        fi
    fi
    if [ "$RUNNING" -eq 0 ]; then
        echo "[状态] 未运行"
        if [ -f "$KD/DISABLED" ]; then
            echo "[提示] 检测到 DISABLED 停用标记"
        fi
    fi

    # 2. Upstart 自启服务状态
    echo "------------------------------------------"
    echo "[Upstart 服务]"
    initctl status kdash 2>/dev/null || echo "无法获取 Upstart 状态"

    # 3. 客户端配置
    echo "------------------------------------------"
    echo "[配置参数]"
    if [ -f "$KD/client.conf" ]; then
        grep -v '^[ \t]*#' "$KD/client.conf" | grep -v '^[ \t]*$'
    else
        echo "未找到 $KD/client.conf"
    fi

    # 4. 电量与网络
    echo "------------------------------------------"
    echo "[设备状态]"
    BATT=$(lipc-get-prop com.lab126.powerd battLevel 2>/dev/null || echo "未知")
    CHARGING=$(lipc-get-prop com.lab126.powerd isCharging 2>/dev/null || echo "未知")
    echo "电池电量: $BATT% (充电状态: $CHARGING)"
    IP=$(ifconfig wlan0 2>/dev/null | awk '/inet addr:/ {print $2}' | cut -d: -f2)
    echo "Wi-Fi IP: ${IP:-未连接或未分配}"

    # 5. 最近日志
    echo "------------------------------------------"
    echo "[最近客户端日志 (末尾 10 行)]"
    if [ -f /var/tmp/kdash-client.log ]; then
        tail -n 10 /var/tmp/kdash-client.log
    else
        echo "无日志文件"
    fi
    echo "=========================================="
} > "$LOG" 2>&1

cat "$LOG"
exit 0
