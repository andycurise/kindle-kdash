# Windows 11 x64 安装指南

## 前置条件

- Windows 11 x86-64（amd64）。
- PowerShell 5.1 或更新版本。
- 主机与 Kindle 位于同一可信局域网。
- 可选：Windows OpenSSH Client，仅在使用服务端 SSH 立即刷新时需要；普通 HTTP 图片同步不需要。

## 一键安装（推荐）

Release 发布后在 PowerShell 执行：

```powershell
& ([scriptblock]::Create((irm -Uri 'https://github.com/andycurise/kindle-kdash/releases/latest/download/install.ps1'))) -Repo 'andycurise/kindle-kdash'
```

安装器会：

1. 识别 Windows amd64，下载 `kdash-windows-amd64.zip` 和 `SHA256SUMS`。
2. 校验归档 SHA-256，安装到 `%LOCALAPPDATA%\kdash`。
3. 首次启动前生成随机 `settings_token`，升级时保留现有配置。
4. 注册 `KindleDashboard` 当前用户登录计划任务并请求 `/api/status`。

该任务使用当前用户权限，不以最高权限运行，**只在用户登录后启动**，不是未登录也可运行的 Windows SCM 服务。

### Agent / 固定版本

```powershell
& ([scriptblock]::Create((irm -Uri 'https://github.com/andycurise/kindle-kdash/releases/download/v0.4.0/install.ps1'))) -Repo 'andycurise/kindle-kdash' -Version 'v0.4.0' -NonInteractive
```

`NoStart` 可只安装不启动；`ReleaseBase` 或环境变量 `KDASH_RELEASE_BASE` 可指定测试镜像。

## 手动安装

1. 下载 `kdash-windows-amd64.zip` 和 `SHA256SUMS`，按 [Release 校验文档](../releases.md) 核对 ZIP。
2. 解压到例如 `D:\kdash`，复制 `kdash.json.example` 为 `kdash.json`。
3. 编辑 `kdash.json`，设置城市、经纬度、`listen_addr` 和高强度随机 `settings_token`。可用 PowerShell 生成令牌：

   ```powershell
   [guid]::NewGuid().ToString('N') + [guid]::NewGuid().ToString('N')
   ```

4. 双击 `start.bat` 前台测试，访问 `http://127.0.0.1:8088/`。
5. 确认后运行 `install_task.bat` 注册登录计划任务。无需为普通安装选择“最高权限运行”；配置 Windows 防火墙时按系统提示单独提权。

## 启动、停止与状态

```bat
start.bat
stop.bat
```

停止脚本只处理当前安装目录对应的任务/进程，优先通过本机 `POST /api/shutdown` 优雅停止，不会用 `taskkill /IM kdash.exe` 结束所有同名程序。

也可以在任务计划程序中管理 `KindleDashboard`。浏览器访问：

- `http://127.0.0.1:8088/`
- `http://127.0.0.1:8088/api/status`
- `http://<主机局域网IP>:8088/screen.png`

## 升级与回滚

重新执行同一安装命令即可。安装器会保留配置、暂存新文件，失败时恢复旧版本。固定版本升级最便于重复部署：

```powershell
& ([scriptblock]::Create((irm -Uri 'https://github.com/andycurise/kindle-kdash/releases/download/v0.4.0/install.ps1'))) -Repo 'andycurise/kindle-kdash' -Version 'v0.4.0' -NonInteractive
```

手动升级前先备份 `kdash.json` 和整个旧版本目录；不要直接覆盖正在运行的 EXE。回滚时停止任务，恢复旧目录和兼容配置，再启动并检查 `/api/status` 中的版本。

## 卸载

仅移除登录计划任务，可运行 `uninstall_task.bat`。完整使用安装器卸载：

```powershell
& ([scriptblock]::Create((irm -Uri 'https://github.com/andycurise/kindle-kdash/releases/latest/download/install.ps1'))) -Repo 'andycurise/kindle-kdash' -Uninstall -KeepConfig
```

`KeepConfig` 表示保留配置。删除配置或整个安装目录属于不可恢复操作，请先备份并人工确认。

## 故障排查

- **Kindle 无法拉图**：检查主机防火墙的 8088/TCP、AP 隔离、实际局域网 IP 和 `SERVER_URL`。
- **修改端口无效**：当前脚本只传 `-config`，请停止旧实例后再启动；检查 `kdash.json` 的 `listen_addr`。
- **保存返回 401**：在 WebUI 填入配置文件中的 `settings_token`；Kindle 远程前光操作还需在 `client.conf` 配置同一令牌。
- **SSH 刷新失败**：确认已安装 OpenSSH Client，私钥和 `known_hosts` 使用绝对路径，并严格核对 Kindle 主机密钥。
- **下载被 SmartScreen 提示**：当前发布不承诺 Authenticode 签名。先核对来源与 SHA-256，也可从源码构建；不要绕过来源不明的程序警告。
