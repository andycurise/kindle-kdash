#!/bin/sh
# USB 首装包存在时一并配置 USBNetwork；日常修复只处理 kdash。
if [ -f /mnt/us/kdash-bootstrap/bootstrap.sh ]; then
    exec sh /mnt/us/kdash-bootstrap/bootstrap.sh
fi
exec sh /mnt/us/kdash/install.sh
