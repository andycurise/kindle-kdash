#!/bin/sh
# kdash Kindle 客户端部署脚本
# 适用架构：Kindle 纯 Shell 极简客户端 + KUAL 菜单，不再复制任何 Go/ARM 二进制
#
# 用法:
#   ./scripts/deploy-kindle.sh usb [/Volumes/Kindle]
#   ./scripts/deploy-kindle.sh ssh [Kindle-IP] [端口，默认 2222]
#   ./scripts/deploy-kindle.sh root@192.168.1.x
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
KDASH_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC_KINDLE="$KDASH_ROOT/deploy/kindle"
SRC_EXT="$SRC_KINDLE/extensions/kdash"

MODE="${1:-}"
if [ -z "$MODE" ]; then
    echo "用法:"
    echo "  $0 usb [/Volumes/Kindle]                 -- 复制到 USB 存储挂载点"
    echo "  $0 ssh <Kindle-IP> [端口, 默认 2222]      -- 通过 SSH 自动化部署更新"
    echo "  $0 root@<Kindle-IP>                      -- 通过原生 SSH 目标部署"
    exit 1
fi

STAMP=$(date +%Y%m%d-%H%M%S)

if [ "$MODE" = "usb" ]; then
    USB_ROOT="${2:-/Volumes/Kindle}"
    [ -d "$USB_ROOT" ] || { echo "错误: 挂载点目录不存在: $USB_ROOT"; exit 1; }
    [ -d "$USB_ROOT/extensions" ] || { echo "警告: 未在 $USB_ROOT 发现 extensions 目录，尝试创建..."; mkdir -p "$USB_ROOT/extensions"; }

    echo "==> 部署 kdash 客户端到 USB 存储: $USB_ROOT"
    mkdir -p "$USB_ROOT/kdash" "$USB_ROOT/extensions/kdash"

    # 1. 复制客户端核心脚本（保留原有 client.conf）
    for f in sync-screen.sh run.sh install.sh restore.sh; do
        cp "$SRC_KINDLE/$f" "$USB_ROOT/kdash/$f"
        chmod 755 "$USB_ROOT/kdash/$f" 2>/dev/null || true
    done

    # 备份净化配置模板，绝不依赖开发机活动配置
    cp "$SRC_KINDLE/client.conf.example" "$USB_ROOT/kdash/client.conf.example"
    if [ ! -f "$USB_ROOT/kdash/client.conf" ]; then
        cp "$SRC_KINDLE/client.conf.example" "$USB_ROOT/kdash/client.conf"
        echo "已初始化 $USB_ROOT/kdash/client.conf"
    else
        echo "保留现有 $USB_ROOT/kdash/client.conf（未覆盖）"
    fi

    # 2. 复制 KUAL 扩展
    cp -r "$SRC_EXT/"* "$USB_ROOT/extensions/kdash/"
    chmod 755 "$USB_ROOT/extensions/kdash/"*.sh 2>/dev/null || true

    echo "==> USB 部署完成！"
    echo "后续步骤："
    echo "1. 检查并编辑 $USB_ROOT/kdash/client.conf 中的 SERVER_URL（填写服务端局域网 IP 与端口）"
    echo "2. 安全弹出 Kindle"
    echo "3. 在 Kindle 打开 KUAL -> 日历看板 -> 安装/修复开机自启（或直接启动日历看板）"
    exit 0
fi

# SSH 模式处理
SSH_TARGET=""
SSH_PORT="2222"
SSH_KEY=""
[ -f "$HOME/.ssh/kdash_key" ] && SSH_KEY="$HOME/.ssh/kdash_key"

case "$MODE" in
    ssh)
        IP="${2:?请输入 Kindle IP 地址，例如: $0 ssh 192.168.1.100}"
        SSH_PORT="${3:-2222}"
        SSH_TARGET="root@$IP"
        ;;
    *@*)
        SSH_TARGET="$MODE"
        ;;
    *)
        SSH_TARGET="root@$MODE"
        ;;
esac

echo "==> 通过 SSH 部署到 $SSH_TARGET (端口 $SSH_PORT)"

KNOWN_HOSTS="${KDASH_KNOWN_HOSTS:-$HOME/.ssh/kdash_known_hosts}"
if [ ! -f "$KNOWN_HOSTS" ]; then
    echo "错误: 未找到 known_hosts 文件: $KNOWN_HOSTS"
    echo "请先核对 Kindle 主机密钥并写入该文件；也可用 KDASH_KNOWN_HOSTS 指定路径。"
    exit 1
fi

SSH_OPTS="-p $SSH_PORT -o BatchMode=yes -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes -o UserKnownHostsFile=$KNOWN_HOSTS"
SCP_OPTS="-P $SSH_PORT -o BatchMode=yes -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes -o UserKnownHostsFile=$KNOWN_HOSTS"
if [ -n "$SSH_KEY" ]; then
    SSH_OPTS="$SSH_OPTS -i $SSH_KEY"
    SCP_OPTS="$SCP_OPTS -i $SSH_KEY"
fi

# 1. 在设备创建必要目录
ssh $SSH_OPTS "$SSH_TARGET" 'mkdir -p /mnt/us/kdash/backups /mnt/us/extensions/kdash'

# 2. 传输客户端核心脚本（保留原有 client.conf）
scp $SCP_OPTS \
    "$SRC_KINDLE/sync-screen.sh" \
    "$SRC_KINDLE/run.sh" \
    "$SRC_KINDLE/install.sh" \
    "$SRC_KINDLE/restore.sh" \
    "$SSH_TARGET:/mnt/us/kdash/"

scp $SCP_OPTS \
    "$SRC_KINDLE/client.conf.example" \
    "$SSH_TARGET:/mnt/us/kdash/client.conf.example"

# 3. 传输 KUAL 扩展
scp $SCP_OPTS -r \
    "$SRC_EXT/"* \
    "$SSH_TARGET:/mnt/us/extensions/kdash/"

# 4. 在设备上修正权限并执行安装
ssh $SSH_OPTS "$SSH_TARGET" '
    chmod 755 /mnt/us/kdash/*.sh /mnt/us/extensions/kdash/*.sh
    if [ ! -f /mnt/us/kdash/client.conf ]; then
        cp /mnt/us/kdash/client.conf.example /mnt/us/kdash/client.conf 2>/dev/null || true
    fi
    sh /mnt/us/kdash/install.sh
'

echo "==> SSH 部署与安装完成！"
echo "可执行以下命令查看日志："
echo "  ssh $SSH_OPTS $SSH_TARGET 'tail -n 30 /var/tmp/kdash-client.log'"
