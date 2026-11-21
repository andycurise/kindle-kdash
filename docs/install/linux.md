# Linux 安装指南

## 前置条件

- Linux x86-64（amd64）或 ARM64。
- systemd、`curl`、`tar`、CA 根证书、DNS 和正确系统时间。
- 一键系统服务安装需要 `sudo`。
- 可选：OpenSSH 客户端，仅用于 Kindle 立即刷新；普通 HTTP 同步不需要。

精简系统或容器需自行提供 CA 证书和时区以外的网络基础配置；程序已内嵌 Go 时区数据。

## 一键安装（推荐）

Release 发布后执行：

```sh
curl -fsSL https://github.com/andycurise/kindle-kdash/releases/latest/download/install.sh | sudo bash -s -- --repo andycurise/kindle-kdash --non-interactive
```

安装器自动选择 `kdash-linux-amd64.tar.gz` 或 `kdash-linux-arm64.tar.gz`，核对 `SHA256SUMS`，并配置：

- 低权限 `kdash` 系统用户；
- `/etc/kdash/kdash.json`；
- `/var/lib/kdash` 天气缓存和运行数据；
- `/etc/systemd/system/kdash.service`；
- 随机 `settings_token` 和 `/api/status` 健康检查。

### Agent / 固定版本

```sh
curl -fsSL https://github.com/andycurise/kindle-kdash/releases/download/v0.4.0/install.sh | sudo bash -s -- --repo andycurise/kindle-kdash --version v0.4.0 --non-interactive
```

可选参数：`--prefix <目录>`、`--no-start`、`--release-base <镜像URL>`。注意：`--prefix` 只改变二进制和文档前缀，`/etc/kdash`、`/var/lib/kdash` 与 systemd 路径不变；`--no-start` 仍会安装并启用服务配置。安装测试应在允许修改系统的测试机或容器中进行，不要误以为临时前缀就是完全隔离模式。

## 手动前台测试

1. 下载匹配架构的归档和 `SHA256SUMS`，按 [Release 校验文档](../releases.md) 验证。
2. 解压并复制配置模板：

   ```sh
   tar -xzf kdash-linux-amd64.tar.gz
   cd kdash-linux-amd64
   cp kdash.json.example kdash.json
   chmod 600 kdash.json
   ```

3. 编辑城市、监听地址和随机 `settings_token`，然后运行：

   ```sh
   ./kdash -config ./kdash.json
   ```

4. 访问 `http://127.0.0.1:8088/` 和 `http://127.0.0.1:8088/api/status`。

正式常驻仍推荐使用安装器，不要直接以 root 长期运行二进制。

## 服务管理

```sh
sudo systemctl status kdash
sudo systemctl restart kdash
sudo systemctl stop kdash
sudo journalctl -u kdash -n 100 --no-pager
```

Kindle 图片地址为 `http://<主机局域网IP>:8088/screen.png`。防火墙只应对可信局域网开放所需端口，不要映射到公网。

## 升级与回滚

重新执行相同安装命令即可，安装器会保留配置、验证新资产并在失败时回滚。手动升级前备份：

```sh
sudo cp -a /etc/kdash /etc/kdash.backup
sudo cp -a /var/lib/kdash /var/lib/kdash.backup
```

回滚时停止服务、恢复旧二进制和兼容配置，再执行 `systemctl start kdash` 并核对版本。备份可能包含令牌，应按私密文件保护。

## 卸载

```sh
curl -fsSL https://github.com/andycurise/kindle-kdash/releases/latest/download/install.sh | sudo bash -s -- --repo andycurise/kindle-kdash --uninstall --keep-config
```

`--keep-config` 保留配置和数据。删除 `/etc/kdash`、`/var/lib/kdash` 或 `kdash` 用户前，应另行备份并确认不再需要。

## 故障排查

- `systemctl status kdash` 与 `journalctl -u kdash` 先判断是配置、端口、权限还是网络问题。
- HTTPS 天气失败时检查 CA、DNS、代理和系统时间，不要关闭 TLS 验证。
- 配置包含未知字段会启动失败；根据错误修正拼写，不要添加旧架构的 `output_png` 或 `stay_awake_cmds`。
- 无令牌且携带 `X-KDash-Client: kdash` 的本机回环请求可完成首次设置；来自局域网的管理写请求必须携带 `X-KDash-Token`。
