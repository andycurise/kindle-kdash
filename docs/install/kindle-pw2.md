# Kindle Paperwhite 2 安装指南

## 支持范围与前置条件

- 已验证：Kindle Paperwhite 2（PW2）、758×1024、固件 5.6.1.1。
- 已越狱并安装 KUAL；设备具有 `eips`、`wget`、LIPC、`iwconfig` 和 BusyBox 常用命令。
- Windows/Linux/macOS 主机已运行 kdash 服务端，且与 Kindle 位于同一可信局域网。

“纯 Shell 客户端”表示不需要 Kindle Go 工具链或额外 Go ARM 程序，并不表示不需要上述系统命令。其他 Kindle 型号和固件尚未验证。

## USB 安装（推荐）

1. 下载并校验 `kdash-kindle-pw2.zip`。
2. 将包内 `kdash/` 和 `extensions/` 解压到 Kindle USB 存储根目录，最终应存在：
   - `/mnt/us/kdash/sync-screen.sh`；
   - `/mnt/us/kdash/client.conf`；
   - `/mnt/us/extensions/kdash/config.xml`。
3. 编辑 `kdash/client.conf`：

   ```sh
   SERVER_URL="http://192.168.1.100:8088/screen.png"
   REFRESH_SECONDS=60
   SETTINGS_TOKEN="与服务端 settings_token 相同"
   ```

   `REFRESH_SECONDS=60` 会等待下一分钟整点；其他正整数表示按该秒数轮询。令牌只用于 KUAL 远程前光切换等管理操作，不使用这些操作时可不保存到 Kindle。
4. 安全弹出并拔掉 USB，在 KUAL 的“日历看板”中选择：
   - “安装/修复开机自启”；或
   - “启动日历看板”立即测试。

已有 `client.conf` 是个人活动配置，升级时应先备份，不能用 ZIP 中的示例盲目覆盖。

## 日常使用

- “立即刷新一次”只通过 HTTP 拉图，不依赖 SSH。
- “停止看板并恢复原生界面”会恢复 Wi-Fi 电源策略、powerd、状态栏和主页，再回写 framebuffer 并全刷。
- wget 或 `eips` 失败时保留旧图并重试，不应把日志中的单一“成功”当作实际画面验收。
- 截图验收可使用 `/mnt/us/usbnet/bin/fbgrab /var/tmp/kdash-screen.png`，再取回电脑检查完整布局、方向、天气和裁切。

## SSH 开发者部署（可选）

先确认设备 Wi-Fi IP、私钥和 `known_hosts`，不要用 `StrictHostKeyChecking=no`：

```sh
./scripts/deploy-kindle.sh ssh <已确认的KindleIP> 2222
```

默认读取：

- `~/.ssh/kdash_key`；
- `~/.ssh/kdash_known_hosts`，也可用 `KDASH_KNOWN_HOSTS` 覆盖。

脚本使用严格主机密钥校验、保留设备 `client.conf`，但会复制脚本并执行安装/启动操作，不只是“同步文件”。它不会部署 Go ARM 二进制，也不会创建 `/mnt/us/usbnet/auto`。不要猜测密码或设备 IP。

## 升级与回滚

1. 备份 `/mnt/us/kdash/client.conf` 和 `/var/tmp/kdash-client.log`（如需诊断）。
2. 只替换发布包中的 Shell/KUAL 文件，保留活动配置。
3. 通过 KUAL 重启，检查日志和实际 framebuffer。
4. 回滚时恢复旧脚本，执行停止恢复流程后再启动；若 framebuffer 回写失败，保留 dump 供排查。

## 卸载与停用

优先使用 KUAL 的停止/移除自启菜单。不要通过创建 `/mnt/us/usbnet/auto` 修复网络或启动问题。项目不会恢复或创建该文件，也不让 USBNetwork 参与开机启动。

## 故障排查

- KUAL 只显示 `exception logged` 时，读取 `/var/tmp/KUAL.log`、客户端日志和进程状态。
- 图片横向拉伸时，确认服务端 `/screen.png` 是 758×1024 的 8-bit Gray，并用 `eips -g` 显示。
- 前光远程切换返回失败时，检查服务端与 Kindle 的 `SETTINGS_TOKEN` 是否一致；本地前光操作不应因服务端离线而被报告为同步成功。
- 开机测试前确认 `/mnt/us/kdash/DISABLED` 和 `/mnt/us/usbnet/auto` 的预期状态；用户分区空间不足时不要删除书籍或个人数据。
