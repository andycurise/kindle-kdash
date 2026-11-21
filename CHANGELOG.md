# 更新记录

本项目采用语义化版本，Release tag 使用 `vX.Y.Z`。

## [0.4.0] - 2026-11-20

### 新增

- 正式构建 Windows amd64、macOS arm64、Linux amd64/arm64 服务端。
- 新增 Linux/macOS/Windows 安装器、SHA-256 校验、配置保留、健康检查和失败回滚。
- 新增 systemd、macOS LaunchAgent 模板；Windows 保留当前用户登录计划任务。
- 新增本机 `POST /api/shutdown` 优雅停止接口。
- 新增 CI/tag Release 工作流、平台安装文档、Release 文档和开源审查报告。
- 补齐 MIT、第三方授权、安全与贡献说明。

### 修复

- Windows 启动脚本不再用 `-serve :8088` 覆盖配置文件。
- 环境变量管理令牌不再随配置保存落盘；Unix 私密配置使用 `0600`。
- 前光保存失败不再返回成功；配置遵循“写盘成功后提交内存”。
- Kindle `REFRESH_SECONDS` 现在实际控制轮询间隔，60 秒保持分钟对齐。
- 修复 DST 时区节气自然日倒计时和测试固定 `/tmp` 路径。
- Kindle 部署使用净化配置模板和严格 SSH 主机密钥校验。

### 安全与兼容性变更

- 所有管理修改接口使用 POST/PUT 和统一鉴权；空令牌只允许本机回环首次设置，不再放行局域网写操作。
- `/api/set-frontlight` 不再接受 GET。Kindle KUAL 前光脚本改为 POST，并支持白名单 `SETTINGS_TOKEN`。
- 配置文件加载拒绝未知字段，旧架构字段需要清理，不再静默忽略。
- 默认不再优先请求明文 HTTP 国内空气质量镜像，使用 HTTPS Open-Meteo 空气质量源。

### 尚未承诺

- Windows 登录计划任务不是未登录运行的 SCM 服务。
- 尚未提供 Authenticode、macOS 公证、Sigstore、SBOM 或 provenance。
- GitHub 三系统 CI、tag Release 和真机安装结果需在对应环境验收。

[0.4.0]: https://github.com/andycurise/kindle-kdash/releases/tag/v0.4.0
