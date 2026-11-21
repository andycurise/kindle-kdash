# macOS Apple Silicon 安装指南

## 前置条件

- Apple Silicon Mac（arm64）。
- `curl`、`tar` 和 macOS 系统 CA 证书。
- 当前用户可登录并允许运行 LaunchAgent。

LaunchAgent 在当前用户登录后运行；未登录时不会运行。若需要无人登录的系统级服务，应自行评审 LaunchDaemon、专用账号和目录权限，不要把用户 LaunchAgent 直接改成 root LaunchDaemon。

## 一键安装（推荐）

Release 发布后执行：

```sh
curl -fsSL https://github.com/andycurise/kindle-kdash/releases/latest/download/install.sh | bash -s -- --repo andycurise/kindle-kdash --non-interactive
```

安装器会下载 `kdash-darwin-arm64.tar.gz`、核对 `SHA256SUMS`，并配置：

- 程序：`~/.local/bin/kdash`；
- 配置：`~/Library/Application Support/kdash/kdash.json`；
- 数据：`~/Library/Application Support/kdash/data`；
- 日志：`~/Library/Logs/kdash`；
- LaunchAgent：`~/Library/LaunchAgents/com.kdash.agent.plist`；
- 随机 `settings_token` 和 `/api/status` 健康检查。

### Agent / 固定版本

```sh
curl -fsSL https://github.com/andycurise/kindle-kdash/releases/download/v0.4.0/install.sh | bash -s -- --repo andycurise/kindle-kdash --version v0.4.0 --non-interactive
```

可选 `--prefix`、`--no-start`、`--release-base`。`--prefix` 只改变二进制和文档前缀；配置、日志和 LaunchAgent 路径由 `HOME` 决定。本地验收应使用临时 `HOME`、模拟 `launchctl` 和 `--no-start`，避免加载或卸载当前用户真实服务。

## 手动前台测试

```sh
tar -xzf kdash-darwin-arm64.tar.gz
cd kdash-darwin-arm64
cp kdash.json.example kdash.json
chmod 600 kdash.json
./kdash -config ./kdash.json
```

启动前编辑城市、监听地址和随机 `settings_token`，然后访问 `http://127.0.0.1:8088/`。

当前发布流程不承诺 Developer ID 签名或 notarization。若 Gatekeeper 拦截，请先验证下载来源和 SHA-256；不要对来源不明的文件使用 `xattr` 绕过检查。对安全要求较高的环境，建议按 `go.mod` 指定工具链从源码构建。

## 服务管理与日志

```sh
launchctl print "gui/$UID/com.kdash.agent"
tail -n 100 ~/Library/Logs/kdash/kdash.stdout.log ~/Library/Logs/kdash/kdash.stderr.log
```

卸载或停止 LaunchAgent 可使用安装器；手动操作前先确认 plist 标签和路径，避免误停其他服务。

## 升级与回滚

重新执行安装命令即可，既有 `kdash.json` 会保留，失败时恢复旧版本。手动升级前备份程序目录和配置，回滚后核对 `/api/status` 的版本。包含令牌的备份不得公开上传。

## 卸载

```sh
curl -fsSL https://github.com/andycurise/kindle-kdash/releases/latest/download/install.sh | bash -s -- --repo andycurise/kindle-kdash --uninstall --keep-config
```

`--keep-config` 保留配置。删除 `~/Library/Application Support/kdash` 和日志前请备份并确认。

## 故障排查

- 使用 `file kdash` 确认产物为 `Mach-O 64-bit executable arm64`。
- 使用 `launchctl print` 和日志判断退出原因，不要只看 plist 是否加载。
- 修改 `listen_addr` 后需要重启服务；安装脚本不会再用命令行端口覆盖配置。
- 远程保存返回 401 时，检查 WebUI 中的 `settings_token`；HTTP 令牌不能替代可信局域网或 TLS。
