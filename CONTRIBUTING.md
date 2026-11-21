# 贡献指南

## 开发原则

- 正式架构是主机 Go 服务 + Kindle Shell 客户端；不要重新引入 Kindle 原生 Go ARM 构建。
- 正式渲染和 `/api/layout` 必须共用布局计算，预览和 `/screen.png` 必须共用 `Dashboard.Render`。
- 设置页通过草稿配置工作，校验并原子保存后才更新服务端内存；不要在浏览器另写一套硬编码选区。
- 保留 758×1024、`image.Gray` 和 `eips -g` 的 Kindle 兼容约束。
- 代码风格、注释和命名应与邻近文件一致，用户文档默认使用清晰中文。

## 本地流程

```sh
go mod verify
go test -count=1 ./...
go vet ./...
./scripts/build.sh
./scripts/package.sh
```

修改配置、鉴权或 API 时，至少补充正常、失败、未授权和保存回环测试。修改天气时覆盖空结果、跨午夜缓存、数组完整性和离线状态；修改显示时覆盖旋转像素和布局 bounds。

普通测试必须使用 `t.TempDir()`、`httptest` 和 fake refresher，不得写活动配置、访问真实天气或 SSH 到 Kindle。

## 跨平台要求

- 显式支持 Windows amd64、macOS arm64、Linux amd64/arm64。
- 不依赖当前工作目录、开发机字体、绝对路径或某台主机的 shell 工具。
- Unix 配置权限、Windows 路径和计划任务语义要分别考虑；交叉编译成功不等于真机验证完成。
- 新依赖需评估许可证、Go 版本和供应链成本，并更新 NOTICE。

## 发布与设备操作

- 不要提交活动配置、SSH 密钥、真实令牌、日志、诊断截图或天气缓存。
- 不要创建仓库、推送 tag、发布 Release、安装系统服务或操作 Kindle，除非任务明确授权。
- Windows 计划任务、Linux systemd、macOS LaunchAgent 和 Kindle framebuffer 的验证结果必须分别报告。
- 未运行的平台、工具或测试要明确标注“未验证”，不能用“应该可以”替代结果。

## Pull Request 建议内容

1. 问题与变更摘要。
2. 测试命令和实际输出。
3. 支持矩阵与兼容性影响。
4. 文档、CHANGELOG、许可证是否需要同步。
5. 尚未验证的平台和风险。
