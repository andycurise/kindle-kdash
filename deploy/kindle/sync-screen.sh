#!/bin/sh
# Kindle 极简日历看板客户端
# 纯 Shell 实现，内存开销 < 1MB，彻底消除 OOM 与崩溃重启隐患。
# 架构：从 Win11 服务端拉取渲染好的灰度图片，使用 Kindle 原生 eips 命令刷屏。

KD=/mnt/us/kdash
CONF="$KD/client.conf"
LOG="/var/tmp/kdash-client.log"
PIDFILE="/var/tmp/kdash.pid"
TOUCH_FILE="/var/tmp/kdash-touch-active"
CUR_PNG="/var/tmp/screen_current.png"
NEW_PNG="/var/tmp/screen_new.png"
FBDUMP="/var/tmp/kdash-fb.dump"

# 默认参数（如果配置文件不存在）
SERVER_URL="http://192.168.1.99:8088/screen.png"
REFRESH_SECONDS=60
GHOST_CLEAR_SECONDS=3600
TOUCH_PAUSE_SECONDS=180
FRONTLIGHT_MODE="schedule"
DAY_BRIGHTNESS=10
NIGHT_BRIGHTNESS=0
DAY_START_HOUR=7
NIGHT_START_HOUR=22

# 安全白名单加载配置，杜绝重复 source 与任意代码执行
load_config() {
    [ -f "$CONF" ] || return 0
    while IFS='=' read -r raw_key raw_val || [ -n "$raw_key" ]; do
        # 移除行首尾空格与换行符
        key=$(echo "$raw_key" | tr -d ' \t\r\n')
        case "$key" in
            \#*|"") continue ;;
        esac
        # 去除回车符、引号与两端空白（注意：BusyBox sed 不支持 \r 转义，必须用 tr -d '\r' 避免误删字母 r）
        val=$(echo "$raw_val" | tr -d '\r' | sed -e 's/^[ \t"'\''"]*//' -e 's/[ \t"'\''"]*$//')
        case "$key" in
            SERVER_URL)           [ -n "$val" ] && SERVER_URL="$val" ;;
            REFRESH_SECONDS)      [ -n "$val" ] && REFRESH_SECONDS="$val" ;;
            GHOST_CLEAR_SECONDS)  [ -n "$val" ] && GHOST_CLEAR_SECONDS="$val" ;;
            TOUCH_PAUSE_SECONDS)  [ -n "$val" ] && TOUCH_PAUSE_SECONDS="$val" ;;
            FRONTLIGHT_MODE)      [ -n "$val" ] && FRONTLIGHT_MODE="$val" ;;
            DAY_BRIGHTNESS)       [ -n "$val" ] && DAY_BRIGHTNESS="$val" ;;
            NIGHT_BRIGHTNESS)     [ -n "$val" ] && NIGHT_BRIGHTNESS="$val" ;;
            DAY_START_HOUR)       [ -n "$val" ] && DAY_START_HOUR="$val" ;;
            NIGHT_START_HOUR)     [ -n "$val" ] && NIGHT_START_HOUR="$val" ;;
        esac
    done < "$CONF"
}

# 初始加载配置
load_config

# 从远端服务端获取 /client.conf，实现 Web 设置修改即时下发
fetch_remote_config() {
    base_url=$(echo "$SERVER_URL" | sed -e 's|/screen\.png.*||' -e 's|/*$||')
    conf_url="${base_url}/client.conf"
    tmp_conf="/var/tmp/client.conf.remote"

    if wget -q -T 5 -O "$tmp_conf" "$conf_url" 2>/dev/null; then
        while IFS='=' read -r raw_key raw_val || [ -n "$raw_key" ]; do
            key=$(echo "$raw_key" | tr -d ' \t\r\n')
            case "$key" in
                \#*|"") continue ;;
            esac
            val=$(echo "$raw_val" | tr -d '\r' | sed -e 's/^[ \t"'\''"]*//' -e 's/[ \t"'\''"]*$//')
            case "$key" in
                FRONTLIGHT_MODE)
                    case "$val" in
                        schedule|off|on) FRONTLIGHT_MODE="$val" ;;
                    esac
                    ;;
                DAY_BRIGHTNESS)
                    case "$val" in
                        ''|*[!0-9]*) ;;
                        *) [ "$val" -ge 0 ] && [ "$val" -le 24 ] && DAY_BRIGHTNESS="$val" ;;
                    esac
                    ;;
                NIGHT_BRIGHTNESS)
                    case "$val" in
                        ''|*[!0-9]*) ;;
                        *) [ "$val" -ge 0 ] && [ "$val" -le 24 ] && NIGHT_BRIGHTNESS="$val" ;;
                    esac
                    ;;
                DAY_START_HOUR)
                    case "$val" in
                        ''|*[!0-9]*) ;;
                        *) [ "$val" -ge 0 ] && [ "$val" -le 23 ] && DAY_START_HOUR="$val" ;;
                    esac
                    ;;
                NIGHT_START_HOUR)
                    case "$val" in
                        ''|*[!0-9]*) ;;
                        *) [ "$val" -ge 0 ] && [ "$val" -le 23 ] && NIGHT_START_HOUR="$val" ;;
                    esac
                    ;;
                GHOST_CLEAR_SECONDS)
                    case "$val" in
                        ''|*[!0-9]*) ;;
                        *) [ "$val" -ge 0 ] && GHOST_CLEAR_SECONDS="$val" ;;
                    esac
                    ;;
            esac
        done < "$tmp_conf"
        rm -f "$tmp_conf"
        return 0
    fi
    return 1
}

set_frontlight() {
    target_val="$1"
    [ -z "$target_val" ] && return 0

    cur_sysfs=""
    for f in /sys/class/backlight/*/brightness /sys/devices/system/fl_tps6116x/fl_tps6116x0/fl_intensity; do
        if [ -r "$f" ]; then
            cur_sysfs=$(cat "$f" 2>/dev/null | awk '{print $NF}')
            break
        fi
    done

    if [ "$target_val" -eq 0 ] 2>/dev/null; then
        if [ "$cur_sysfs" = "0" ]; then
            return 0
        fi
        lipc-set-prop -i com.lab126.powerd flIntensity 0 2>/dev/null || true
        for f in /sys/class/backlight/*/brightness /sys/devices/system/fl_tps6116x/fl_tps6116x0/fl_intensity; do
            if [ -w "$f" ]; then
                echo 0 > "$f" 2>/dev/null || true
            fi
        done
        echo "$(date '+%H:%M:%S') 背光彻底关闭 (0 亮度)"
    else
        cur_lipc=$(lipc-get-prop com.lab126.powerd flIntensity 2>/dev/null || echo "")
        if [ "$cur_lipc" = "$target_val" ] && [ -n "$cur_sysfs" ] && [ "$cur_sysfs" -gt 0 ] 2>/dev/null; then
            return 0
        fi
        lipc-set-prop -i com.lab126.powerd flIntensity 0 2>/dev/null || true
        lipc-set-prop -i com.lab126.powerd flIntensity "$target_val" 2>/dev/null || true
        echo "$(date '+%H:%M:%S') 背光调节至: $target_val"
    fi
}

apply_frontlight_policy() {
    mode="${FRONTLIGHT_MODE:-schedule}"
    case "$mode" in
        off|OFF|0)
            set_frontlight 0
            ;;
        on|ON)
            set_frontlight "${DAY_BRIGHTNESS:-10}"
            ;;
        schedule|SCHEDULE|auto|AUTO)
            cur_h=$(date +%H)
            cur_h=${cur_h#0}
            [ -z "$cur_h" ] && cur_h=0
            d_start=${DAY_START_HOUR:-7}
            n_start=${NIGHT_START_HOUR:-22}
            d_bri=${DAY_BRIGHTNESS:-10}
            n_bri=${NIGHT_BRIGHTNESS:-0}

            if [ "$d_start" -lt "$n_start" ]; then
                if [ "$cur_h" -ge "$d_start" ] && [ "$cur_h" -lt "$n_start" ]; then
                    set_frontlight "$d_bri"
                else
                    set_frontlight "$n_bri"
                fi
            else
                if [ "$cur_h" -ge "$d_start" ] || [ "$cur_h" -lt "$n_start" ]; then
                    set_frontlight "$d_bri"
                else
                    set_frontlight "$n_bri"
                fi
            fi
            ;;
    esac
}

get_request_url() {
    batt=$(lipc-get-prop com.lab126.powerd battLevel 2>/dev/null || echo "")
    charging=$(lipc-get-prop com.lab126.powerd isCharging 2>/dev/null || echo "0")
    if [ -n "$batt" ]; then
        case "$SERVER_URL" in
            *\?*) echo "${SERVER_URL}&batt=${batt}&charging=${charging}" ;;
            *)    echo "${SERVER_URL}?batt=${batt}&charging=${charging}" ;;
        esac
    else
        echo "$SERVER_URL"
    fi
}

# 单次刷新模式
if [ "$1" = "-once" ] || [ "$1" = "once" ]; then
    load_config
    fetch_remote_config || true
    apply_frontlight_policy
    echo "$(date '+%Y-%m-%d %H:%M:%S') 单次刷新..."
    REQ_URL=$(get_request_url)
    if wget -q -T 15 -O "$NEW_PNG" "$REQ_URL"; then
        if eips -g "$NEW_PNG"; then
            cp "$NEW_PNG" "$CUR_PNG"
            echo "刷屏完成。"
            exit 0
        else
            echo "eips 刷屏失败，请检查屏幕驱动与图像格式"
            rm -f "$NEW_PNG"
            exit 1
        fi
    else
        echo "拉取图片失败，请检查网络或服务端：$REQ_URL"
        exit 1
    fi
fi

# 日志重定向
exec >> "$LOG" 2>&1

echo "=========================================="
echo "kdash client 启动 $(date '+%Y-%m-%d %H:%M:%S')"
echo "服务端地址: $SERVER_URL"
echo "刷新间隔: ${REFRESH_SECONDS}s, 清残影: ${GHOST_CLEAR_SECONDS}s, 触摸窗口: ${TOUCH_PAUSE_SECONDS}s"
echo "=========================================="

echo "$$" > "$PIDFILE"

# 恢复 Kindle 原生状态函数
cleanup_and_restore() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') 收到停止信号，恢复原生系统..."
    trap - TERM INT HUP EXIT
    if [ -n "$TOUCH_PID" ]; then
        kill "$TOUCH_PID" 2>/dev/null || true
    fi
    rm -f "$PIDFILE" "$TOUCH_FILE" "$CUR_PNG"
    sh "$KD/restore.sh"
    exit 0
}

trap cleanup_and_restore TERM INT HUP

# 1. 备份当前屏幕内容
cat /dev/fb0 > "$FBDUMP" 2>/dev/null || true

# 2. 唤醒并保持 Wi-Fi 与防休眠，隐藏原生状态栏
lipc-set-prop com.lab126.cmd wirelessEnable 1 2>/dev/null || true
lipc-set-prop com.lab126.powerd preventScreenSaver 1 2>/dev/null || true
iwconfig wlan0 power off 2>/dev/null || true
lipc-set-prop com.lab126.pillow interrogatePillow '{"pillowId": "default_status_bar", "function": "nativeBridge.hideMe();"}' 2>/dev/null || true

# 3. 启动后台触摸检测（如果开启）
TOUCH_PID=""
if [ "$TOUCH_PAUSE_SECONDS" -gt 0 ] && [ -r /dev/input/event0 ]; then
    (
        while true; do
            # 阻塞读取触摸屏输入事件（每个事件 16 字节）
            dd if=/dev/input/event0 bs=16 count=1 >/dev/null 2>&1
            date +%s > "$TOUCH_FILE"
            sleep 1
        done
    ) &
    TOUCH_PID=$!
fi

# 计算刷新延迟；默认 60 秒时对齐下一分钟整点。
calc_refresh_delay() {
    case "$REFRESH_SECONDS" in
        ''|*[!0-9]*) delay=60 ;;
        *) delay=$REFRESH_SECONDS ;;
    esac
    if [ "$delay" -lt 1 ] 2>/dev/null; then
        delay=60
    fi
    if [ "$delay" -ne 60 ]; then
        echo "$delay"
        return
    fi
    sec=$(date +%S)
    sec=${sec#0}
    [ -z "$sec" ] && sec=0
    delay=$((60 - sec))
    if [ "$delay" -le 0 ] || [ "$delay" -gt 60 ]; then
        delay=60
    fi
    echo "$delay"
}

LAST_GHOST=$(date +%s)
rm -f "$CUR_PNG" "$TOUCH_FILE"

# 启动时立即生效背光策略
fetch_remote_config || true
apply_frontlight_policy

FIRST_RUN=1

while true; do
    # 检查是否被禁用
    if [ -f "$KD/DISABLED" ]; then
        echo "检测到 DISABLED 标记，退出看板"
        cleanup_and_restore
    fi

    # 动态载入配置修改（无需重复全量 source）
    load_config
    fetch_remote_config || true

    NOW=$(date +%s)

    # 检查触摸维护窗口（首次启动必须无条件拉取并刷屏，绝不被启动时触屏误拦截）
    PAUSED=0
    if [ "$FIRST_RUN" -eq 1 ]; then
        PAUSED=0
        FIRST_RUN=0
    elif [ "$TOUCH_PAUSE_SECONDS" -gt 0 ] && [ -f "$TOUCH_FILE" ]; then
        LAST_TOUCH=$(cat "$TOUCH_FILE" 2>/dev/null || echo 0)
        ELAPSED=$((NOW - LAST_TOUCH))
        if [ "$ELAPSED" -lt "$TOUCH_PAUSE_SECONDS" ]; then
            PAUSED=1
        fi
    fi

    if [ "$PAUSED" -eq 1 ]; then
        # 维护暂停期间不刷新屏幕
        sleep 5
        continue
    fi

    # 执行背光定时策略
    apply_frontlight_policy

    # 拉取最新屏幕图片
    REQ_URL=$(get_request_url)
    if wget -q -T 15 -O "$NEW_PNG" "$REQ_URL"; then
        # 检查是否需要全刷清残影
        if [ "$GHOST_CLEAR_SECONDS" -gt 0 ] && [ $((NOW - LAST_GHOST)) -ge "$GHOST_CLEAR_SECONDS" ]; then
            echo "$(date '+%H:%M:%S') 定时全屏清残影..."
            eips -c || true
            LAST_GHOST=$NOW
            sleep 1
        fi

        # 图片有变化才调用 eips 刷屏，避免重复闪烁
        if ! cmp -s "$NEW_PNG" "$CUR_PNG" 2>/dev/null; then
            if eips -g "$NEW_PNG"; then
                cp "$NEW_PNG" "$CUR_PNG"
                echo "$(date '+%H:%M:%S') 看板刷屏成功"
            else
                echo "$(date '+%H:%M:%S') eips 刷屏失败，保留旧图并在下次重试"
                rm -f "$NEW_PNG"
            fi
        fi
    else
        echo "$(date '+%H:%M:%S') 拉取图片失败，保持当前画面"
    fi

    # 日志控制：超过 100KB 时截断保留最新
    if [ -f "$LOG" ]; then
        LOG_SIZE=$(ls -l "$LOG" | awk '{print $5}')
        if [ "$LOG_SIZE" -gt 102400 ] 2>/dev/null; then
            tail -c 51200 "$LOG" > "${LOG}.tmp" && mv "${LOG}.tmp" "$LOG"
        fi
    fi

    # 按配置间隔睡眠；60 秒保持分钟对齐
    DELAY=$(calc_refresh_delay)
    sleep "$DELAY"
done
