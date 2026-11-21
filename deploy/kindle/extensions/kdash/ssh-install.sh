#!/bin/sh
# Start KOReader's dropbear directly on Wi-Fi and collect diagnostics.
# No reboot, USB gadget switch, or rootfs modification is required.
KD=/mnt/us/kdash
KO=/mnt/us/koreader
SRC=/mnt/us/kdash-bootstrap
LOG="$KD/wifi-ssh.log"
PIDFILE=/tmp/dropbear_koreader.pid
mkdir -p "$KD"

{
    echo "=== Wi-Fi SSH setup $(date) ==="

    # KOReader's patched dropbear reads this file relative to its working directory.
    mkdir -p "$KO/settings/SSH"
    touch "$KO/settings/SSH/authorized_keys"
    if [ -f "$SRC/pubkey.txt" ]; then
        if ! grep -F -q -f "$SRC/pubkey.txt" "$KO/settings/SSH/authorized_keys"; then
            cat "$SRC/pubkey.txt" >> "$KO/settings/SSH/authorized_keys"
        fi
        echo "KOReader SSH public key installed"
    else
        echo "ERROR: missing $SRC/pubkey.txt"
    fi

    RUNNING=0
    if [ -f "$PIDFILE" ]; then
        PID=$(cat "$PIDFILE" 2>/dev/null)
        case "$PID" in
            ''|*[!0-9]*) ;;
            *)
                if kill -0 "$PID" 2>/dev/null && grep -a -q dropbear "/proc/$PID/cmdline" 2>/dev/null; then
                    RUNNING=1
                    echo "dropbear already running: $PID"
                fi
                ;;
        esac
    fi

    if [ "$RUNNING" -ne 1 ]; then
        rm -f "$PIDFILE"
        cd "$KO" || exit 1
        ./dropbear -E -R -s -p 2222 -P "$PIDFILE" >> "$KD/dropbear.log" 2>&1
        echo "dropbear launch status: $?"
    fi

    # Match KOReader's own firewall rules for its SSH plugin.
    iptables -A INPUT -p tcp --dport 2222 -m conntrack --ctstate NEW,ESTABLISHED -j ACCEPT 2>/dev/null || true
    iptables -A OUTPUT -p tcp --sport 2222 -m conntrack --ctstate ESTABLISHED -j ACCEPT 2>/dev/null || true
    sleep 2

    cp /var/tmp/KUAL.log "$KD/KUAL.log" 2>/dev/null || true
    cp /var/tmp/kdash-restore.log "$KD/kdash-restore.log" 2>/dev/null || true
    ifconfig -a > "$KD/network.txt" 2>&1
    netstat -lnt > "$KD/listeners.txt" 2>&1
    ps > "$KD/processes.txt" 2>&1
    [ -f "$PIDFILE" ] && echo "dropbear pid: $(cat "$PIDFILE")"
    echo "Wi-Fi SSH setup finished; port 2222"
    sync
} > "$LOG" 2>&1

# KUAL appends its own wrapper error after this action returns; retain that too.
nohup sh -c 'sleep 3; cp /var/tmp/KUAL.log /mnt/us/kdash/KUAL-latest.log 2>/dev/null; sync' >/dev/null 2>&1 &
exit 0
