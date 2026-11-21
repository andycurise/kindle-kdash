# Release 资产、校验与升级

## 版本规则

- Git tag 使用 `vX.Y.Z`，例如 `v0.4.0`；程序内版本为 `0.4.0`。
- 归档名称不包含版本号，版本由 Release tag 确定，便于 `latest/download` 和固定版本 URL 使用同一资产名。
- 普通手动体验可使用 `releases/latest/download`；自动化和 agent 应固定 tag，避免“最新版本”在部署途中变化。

## 资产清单

| 文件 | 目标 |
| --- | --- |
| `kdash-windows-amd64.zip` | Windows 11 x86-64 |
| `kdash-linux-amd64.tar.gz` | Linux x86-64 |
| `kdash-linux-arm64.tar.gz` | Linux ARM64 |
| `kdash-darwin-arm64.tar.gz` | macOS Apple Silicon |
| `kdash-kindle-pw2.zip` | Kindle PW2 Shell/KUAL 客户端 |
| `install.sh` | Linux/macOS 安装器 |
| `install.ps1` | Windows 安装器 |
| `SHA256SUMS` | 五个归档的 SHA-256 清单 |

服务端归档包含二进制、净化配置模板、许可证和第三方 NOTICE；Unix 归档还包含对应服务模板。归档不得包含活动配置、私钥、真实令牌、天气缓存、日志、诊断截图或开发机绝对路径。

## 下载地址

官方仓库 `andycurise/kindle-kdash` 的下载地址：

```text
最新 Release：
https://github.com/andycurise/kindle-kdash/releases/latest/download/<资产名>

固定 v0.4.0：
https://github.com/andycurise/kindle-kdash/releases/download/v0.4.0/<资产名>
```

不要把相对 `../../releases` 当作面向外部用户的下载入口。

## SHA-256 校验

先下载目标归档和 `SHA256SUMS` 到同一目录。

### Linux

```sh
grep ' kdash-linux-amd64.tar.gz$' SHA256SUMS | sha256sum -c -
```

ARM64 将文件名改为 `kdash-linux-arm64.tar.gz`。

### macOS

```sh
grep ' kdash-darwin-arm64.tar.gz$' SHA256SUMS | shasum -a 256 -c -
```

### Windows PowerShell

```powershell
$expected = ((Select-String -Path .\SHA256SUMS -Pattern 'kdash-windows-amd64.zip$').Line -split '\s+')[0].ToLower()
$actual = (Get-FileHash .\kdash-windows-amd64.zip -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected) { throw 'SHA-256 不匹配，请勿安装' }
```

安装器会执行相应校验。校验失败时必须停止安装，不应跳过。

> SHA-256 清单能发现传输损坏或归档与清单不一致，但如果发布账号、Release 和清单同时被篡改，它不能独立证明发布者身份。

## 签名与公证状态

当前整理版本只承诺生成 SHA-256，**不承诺**：

- Windows Authenticode 签名；
- macOS Developer ID 签名、notarization 或 staple；
- Sigstore/cosign 签名、SLSA provenance 或 SBOM。

这些能力应在真实仓库、Release 身份和证书可用后单独接入并验证。完成后本文应给出精确的签名验证命令，而不是只写“已签名”。

## 升级、配置与回滚

- 重新执行相同安装命令即可升级；安装器应保留已有配置并采用暂存/失败回滚。
- 固定版本部署前记录当前版本和配置摘要；不要公开包含令牌的备份。
- 回滚前停止服务，恢复旧二进制及与其兼容的配置，再请求 `/api/status` 核对版本。
- Kindle 活动 `client.conf` 不属于 Release 模板，升级时单独备份和保留。

## 面向 Agent 的约束

1. 优先固定 Release tag 和资产名，不依赖 GitHub API 猜测“最新”。
2. 使用 `--non-interactive`（shell）或 `-NonInteractive`（PowerShell）。
3. 必须校验 SHA-256；失败即停止。
4. 默认保留配置，不执行 Kindle、真实天气或 SSH 部署。
5. 完成后请求 `/api/status`，报告版本、路径、服务状态和未验证项目。
