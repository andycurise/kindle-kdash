# 安全说明

## 支持范围

当前安全维护对象为 `VERSION` 对应的主分支和最新 Release。旧版本是否修复应查看 `CHANGELOG.md`，不要假设所有历史版本都获得补丁。

## 报告漏洞

请通过 [GitHub Private vulnerability reporting](https://github.com/andycurise/kindle-kdash/security/advisories/new) 提交，或发送邮件至 [andycurise@163.com](mailto:andycurise@163.com)。不要附带真实私钥、管理令牌、家庭网络信息或个人数据。

GitHub 私密报告入口需要维护者在仓库 Security 设置中启用；未启用时请使用上述邮箱。

## 部署安全边界

- 默认服务用于**可信局域网**。不要将 8088 管理端口映射到公网，也不要在公共 Wi-Fi 开放。
- 当前 HTTP 不加密。`X-KDash-Token` 能防止未授权写操作，但不能防止同一网络中的窃听或重放；高安全环境应在反向代理后使用 HTTPS，并限制来源网络。
- 安装器应生成随机 `settings_token`。空令牌只允许携带 `X-KDash-Client: kdash` 的本机回环请求完成首次设置，防止恶意网页通过简单表单触发 CSRF；局域网管理写操作必须提供有效令牌。
- `KDASH_SETTINGS_TOKEN` 只存在于运行时，不应因 Web 保存而写入磁盘；文件中的令牌应按私密配置保护，Unix 权限为 `0600`。
- `/screen.png`、`/preview.png` 和状态/布局等只读接口仍可在局域网访问。它们可能包含城市、天气或设备状态，不属于公网匿名数据接口。
- `/api/shutdown` 仅允许本机回环，并需要符合当前鉴权策略。
- Kindle 的 `client.conf` 如果包含 `SETTINGS_TOKEN`，也应视为敏感配置。

## SSH 与安装器

- Kindle SSH 只使用公钥、`BatchMode` 和严格 `known_hosts` 校验；不要设置 `StrictHostKeyChecking=no`。
- SSH 立即刷新仅执行固定 `sh /mnt/us/kdash/sync-screen.sh once`，Web 参数不得拼接为 root shell 代码。
- 安装器下载归档后必须核对 `SHA256SUMS`，失败即停止。SHA-256 不替代发布者签名。
- `curl | sh` 和 PowerShell 远程脚本适合已信任仓库的快速部署；审计要求较高时，应先下载、阅读、校验安装器，再以固定版本运行。

## 测试约束

测试不得访问真实天气、真实 SSH 或修改活动配置。使用 `t.TempDir()`、`httptest`、fake refresher 和本地模拟 Release 源。任何真实 Kindle 启停、恢复或截图都应单独授权。
