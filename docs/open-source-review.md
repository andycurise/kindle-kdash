# kdash 开源发布审查报告

审查日期：2026-11-20。范围包括 Go 服务、Kindle Shell/KUAL、构建与安装脚本、发布资产和公开文档。

本报告区分“代码已整改”和“目标平台已验证”。通过本机测试或交叉编译，不等于 Windows 11、Linux、macOS 服务生命周期或 Kindle 真机已经验收。

## 结论

当前整改覆盖主要发布阻断项：三平台构建矩阵、净化打包、安装与常驻脚本、管理写接口鉴权、秘密保存、配置权限、许可证要求和公开文档。

在真实仓库地址、完整许可证文件和 Release 资产确认前，仍不应直接公开现有 `dist/` 旧包，也不应把占位安装命令当作可执行下载地址。

## P0：发布阻断项

| 问题 | 证据/影响 | 整改状态 |
| --- | --- | --- |
| 空令牌放行局域网管理写操作，前光接口允许 GET 且无鉴权 | `internal/server/api.go`；可造成未授权配置修改和 CSRF | **已整改**：修改接口统一 POST/PUT 和鉴权；空令牌仅允许回环首次设置 |
| 环境变量令牌可能保存为 `0644` 明文配置 | `cmd/kdash/main.go` → API 保存 → `config.Save` | **已整改**：新增仅运行时令牌字段，序列化和 API 脱敏；Unix 配置为 `0600` |
| 前光保存吞掉磁盘错误并返回成功 | 磁盘只读时内存与磁盘不一致 | **已整改**：保存失败返回 500，不提交内存 |
| 打包依赖被忽略的活动配置 | `scripts/package.sh`、Kindle 部署脚本 | **已整改**：使用受控 `*.example`，活动配置不进入发布包 |
| 无完整三平台构建与发布链路 | macOS 架构继承宿主、无 Linux 目标 | **已整改**：显式 Windows amd64、macOS arm64、Linux amd64/arm64 |
| MIT 与字体/依赖授权缺失 | README 链接失效，嵌入字体再分发条件不明 | **已整改**：补齐 LICENSE/NOTICE、9 个字体族 OFL 和 Go 依赖许可；全部归档携带完整授权文本 |
| 无可复现安装入口 | 无稳定仓库 URL、安装器、校验、升级/卸载说明 | **已整改框架**：新增安装器、SHA-256、文档；真实仓库替换后才能联网使用 |

## P1：高优先级质量与兼容问题

| 问题 | 整改状态 |
| --- | --- |
| Windows 启动脚本固定 `-serve :8088`，覆盖 `listen_addr` | **已整改**：只传 `-config` |
| Windows 任务以最高权限运行且文档混淆开机/登录 | **已整改**：当前用户 Limited 登录任务，文档明确不是 SCM 服务 |
| 停止脚本按映像名强杀所有 kdash 进程 | **已整改**：优先本机 `/api/shutdown`，回退限定任务和完整 EXE 路径 |
| 配置文件未知字段静默回退默认值 | **已整改**：加载和 API 均拒绝未知字段及多个 JSON 对象 |
| 默认优先明文 HTTP 空气质量镜像 | **已整改**：默认禁用该镜像，使用 HTTPS Open-Meteo |
| Kindle `REFRESH_SECONDS` 配置不生效 | **已整改**：自定义正整数控制间隔，60 秒保持分钟对齐 |
| Kindle SSH 部署关闭主机密钥校验 | **已整改**：`StrictHostKeyChecking=yes` + `UserKnownHostsFile` |
| DST 时区按 24 小时计算自然日 | **已整改**：按 UTC 年月日计算自然日差并补测试 |
| 测试写固定 `/tmp` 文件并忽略错误 | **已整改**：`t.TempDir()` 且检查全部错误 |
| README Go 1.21 与 `go.mod` 1.27.4 不一致 | **已整改文档**：以 `go.mod` 为准 |
| 版本来自多个硬编码位置 | **已整改构建**：`VERSION` + `-ldflags` 注入，KUAL 一致性检查 |
| 发布包缺少校验和验证、升级、回滚、卸载文档 | **已整改文档**：按平台补齐 |

## P2：后续改进与剩余风险

以下项目不应被当前“已整改”表述覆盖：

1. **签名与供应链**：SHA-256 已接入，但 Authenticode、macOS 公证、Sigstore、SBOM 和 provenance 尚未配置。
2. **Windows 未登录运行**：当前只支持登录计划任务；需要 SCM 服务时应另行设计、实现和 Win11 真机验收。
3. **直接运行的默认路径**：未传 `-config` 时仍使用当前目录 `kdash.json`；正式安装器和服务模板使用显式路径。
4. **天气循环取消粒度**：服务关闭会取消天气循环上下文，但底层天气 HTTP 请求尚未全部继承该上下文，极端网络条件下停止仍可能受客户端超时影响。
5. **实例与运维**：尚未提供完整单实例锁、跨平台统一日志轮转和 SCM/Event Log 集成。
6. **只读数据暴露**：图片、状态和配置公开视图可能包含城市与 SSH 主机信息，部署文档明确要求可信 LAN，不能将其视为公网匿名接口。
7. **真实平台覆盖**：Windows 11 已验收；Linux systemd 和 Kindle 真机结果必须单独记录，CI 不能替代目标平台验收。
8. **Git 历史**：初始审查工作区没有可用 `.git`；发布仓库已按公开文件白名单建立，后续提交仍需检查密钥、活动配置和被跟踪的忽略文件。

## 建议的发布验收矩阵

| 层级 | 必验内容 |
| --- | --- |
| Go 代码 | `go test ./...`、可行的 `-race`、`go vet`、gofmt、`go mod verify/tidy` |
| 构建 | 四平台架构、`-version`、CGO-free、无 Kindle Go ARM |
| 发布包 | ZIP/tar 结构、白名单、LICENSE/NOTICE、SHA-256、无活动配置/日志/密钥 |
| 安装器 | 首次安装、重复安装、校验失败、配置保留、失败回滚、卸载 |
| HTTP | 非 8088 端口、令牌、GET 修改拒绝、保存失败、优雅停止、Gray PNG |
| Windows 11 | 登录启动、停止、重启、路径含空格/中文、防火墙、最低权限 |
| Linux | systemd 低权限用户、目录权限、journald、CA/DNS/NTP、SIGTERM |
| macOS ARM | LaunchAgent、日志、Gatekeeper 状态、架构、配置保留 |
| Kindle PW2 | 日志、拉图失败保留旧图、前光、停止恢复、实际 framebuffer |

## 本次实际验证记录

- `go mod verify`、`go mod tidy -diff`、`go vet ./...`、`go test -count=1 ./...`、`go test -race -count=1 ./...`：通过。
- Windows amd64、macOS arm64、Linux amd64/arm64：构建成功；`file` 架构检查正确；可原生执行的 macOS 二进制报告 `0.4.0`。
- `dist`：生成五个归档、`install.sh`、`install.ps1` 和 `SHA256SUMS`，全部归档 SHA-256 验证通过。
- 归档白名单：README/RELEASES、LICENSE、NOTICE 和 9 份 OFL 齐全；无活动配置、诊断、密钥、缓存、旧本机路径或 `bootstrap.sh`。
- Shell：`sh -n`、`bash -n`、ShellCheck 通过；launchd plist、工作流 YAML 和 15 份文档的本地链接检查通过。
- macOS 安装器：使用临时 HOME、真实本地 Release 和 fake launchctl，首次安装、重复安装配置不变、坏校验和拒绝、保留配置卸载通过；未加载真实服务。
- 离线 HTTP 进程：18088→18089 保存重启、版本、鉴权、Gray PNG 和优雅停止通过。
- 浏览器：字体草稿、令牌保存、重载持久化、错误令牌可见提示和 Kindle 灰度预览通过。
- [GitHub CI 44886844593](https://github.com/andycurise/kindle-kdash/actions/runs/44886844593)：Ubuntu、Windows、macOS 15 原生测试，PowerShell 5.1/pwsh 双解析，ShellCheck，四平台归档和 macOS 安装器测试通过。
- [v0.4.0 Release 工作流](https://github.com/andycurise/kindle-kdash/actions/runs/44887554172)：测试、构建、归档检查和发布通过；八个实际发布资产完整，下载后五个归档 SHA-256 均通过。
- Windows 11 真机：PowerShell 5.1 下从 0.3.0 升级到 0.4.0、重复安装配置 SHA-256 不变、坏校验和拒绝且服务不中断、计划任务停止后进程退出、重启后健康检查及灰度 PNG 均通过；使用真实 GitHub Release 联网重复安装且配置不变。

## 审查边界

初始本地审查未执行真实天气、SSH、Kindle 部署、系统服务安装、计划任务修改、仓库推送或 Release 发布。后续已获授权完成 `main`/`v0.4.0` 推送、GitHub CI/Release 和 Win11 SSH/计划任务验收。Linux systemd、真实 LaunchAgent 和 Kindle framebuffer 仍未验证。
