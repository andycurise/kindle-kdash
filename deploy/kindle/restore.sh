#!/bin/sh
# 可重复执行的恢复脚本；不依赖 KUAL 或看板进程仍在运行。
# 严格按照规范：先还原 Wi-Fi 电源、powerd、状态栏和主页，再回写 framebuffer，最后全刷；回写失败保留 dump。

LOG=/var/tmp/kdash-restore.log
FBDUMP=/var/tmp/kdash-fb.dump

{
    echo "=== kdash restore $(date) ==="
    
    # 1. 还原 Wi-Fi 电源与电源管理
    iwconfig wlan0 power on 2>/dev/null || true
    lipc-set-prop com.lab126.powerd preventScreenSaver 0 2>/dev/null || true

    # 2. 还原原生状态栏
    lipc-set-prop com.lab126.pillow interrogatePillow '{"pillowId": "default_status_bar", "function": "nativeBridge.showMe();"}' 2>/dev/null || true

    # 3. 切换回原生主页
    lipc-set-prop com.lab126.appmgrd start app://com.lab126.booklet.home 2>/dev/null || true
    sleep 2

    # 4. 回写启动前的完整画面；回写失败保留 dump
    if [ -f "$FBDUMP" ]; then
        if cat "$FBDUMP" > /dev/fb0 2>/dev/null; then
            rm -f "$FBDUMP"
        else
            echo "警告: 回写 framebuffer 失败，已保留 dump: $FBDUMP"
        fi
    fi

    # 5. 全屏刷新
    eips -s w=758,h=1024 -f 2>/dev/null || true
    echo "=== restore finished ==="
} >> "$LOG" 2>&1
exit 0
