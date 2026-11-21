# kdash 发布维护指南

本文面向维护者。普通用户只需阅读 `docs/releases.md` 和对应安装指南。

## 发布前必须完成的仓库配置

官方仓库为 `andycurise/kindle-kdash`，Go canonical module path 为 `github.com/andycurise/kindle-kdash`。

1. 检查 README 徽章、Release 链接、安全反馈渠道和安装器默认仓库。
2. 在干净克隆中确认活动配置、日志、缓存、诊断文件和旧构建不会被跟踪或打包。
3. 确认字体及 Go 依赖许可证完整，发布包包含 `LICENSE` 和 `THIRD_PARTY_NOTICES.md`。

一键下载命令只有在对应 Release 和资产发布后才可使用。

## 每次提交的检查

```sh
gofmt -l ./cmd ./internal
go mod verify
go mod tidy -diff
go test -count=1 ./...
go vet ./...
./scripts/build.sh
./scripts/package.sh
sh -n scripts/*.sh deploy/kindle/*.sh deploy/kindle/extensions/kdash/*.sh
```

在工具可用的平台还应运行 ShellCheck、PowerShell parser、`go test -race ./...`。CI 在 Ubuntu、ARM macOS 和 Windows 原生运行核心测试；交叉编译不能替代目标系统运行验收。

## tag 发布流程

1. 更新 `VERSION`、`CHANGELOG.md` 和必要的 KUAL 元数据，保持版本一致；同时编写 `docs/release-notes/vX.Y.Z.md`，Release 工作流会优先把它作为用户可读说明。
2. 完成全部测试和发布包白名单检查。
3. 创建带注释的 SemVer tag，例如 `v0.4.0`。
4. 推送 tag 后，`.github/workflows/release.yml` 应执行测试、构建、打包、版本/SHA 校验，并使用 `gh` 上传 Release。
5. 在 Release 页面确认五个归档、两个安装器和 `SHA256SUMS` 完整。
6. 从全新下载位置分别执行 Linux、macOS、Windows 安装烟测；不要只在打包机检查。

创建仓库、推送 tag 和公开发布属于外发操作，应单独获得维护者授权，不能由本地审查任务隐式执行。

## 发布包验收清单

- `kdash -version` 与 tag 一致，KUAL 版本一致。
- Windows EXE 为 PE32+ x86-64；macOS 为 Mach-O arm64；Linux 产物架构正确。
- ZIP/tar 结构正确，无路径穿越、绝对路径或符号链接逃逸。
- 所有归档只含白名单文件，包含 LICENSE/NOTICE，不含活动配置和诊断数据。
- `SHA256SUMS` 只含 basename，下载后校验通过。
- 安装器首次安装、重复安装、校验失败、保留配置、回滚和卸载均在临时环境验证。
- Windows 登录计划任务、Linux systemd、macOS LaunchAgent 分别在真实目标系统验证。
- Kindle 仅做经授权的单独设备验收，检查日志、停止恢复和实际 framebuffer。

## 签名与供应链后续工作

当前未配置代码签名、macOS 公证、Sigstore、SBOM 或 provenance。接入时应：

- 将第三方 Actions 固定到 commit SHA；
- 使用 GitHub OIDC/Sigstore 或受控证书签署 `SHA256SUMS`/二进制；
- 在 `docs/releases.md` 写出可复制的验证命令和预期身份；
- 将签名失败设为发布失败，而不是继续上传“已签名”资产。

## 紧急撤回

若发现密钥泄露、恶意归档或严重安全问题，先停止分发并标记 Release，保留可审计记录；随后撤销受影响凭据、发布修复版本和升级说明。不要仅删除文件后声称风险已消失。
