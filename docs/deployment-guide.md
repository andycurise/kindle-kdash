# kdash 部署与配置指南

部署分为两部分：一台运行 kdash 服务的 Windows/Linux/macOS 主机，以及一台只拉取灰度图的 Kindle PW2。请先完成服务端健康检查，再配置 Kindle。

## 1. 选择安装指南

| 平台 | 推荐方式 | 完整文档 |
| --- | --- | --- |
| Windows 11 x64 | PowerShell 安装器 + 当前用户登录计划任务 | [Windows 安装](install/windows.md) |
| Linux amd64/arm64 | Shell 安装器 + systemd | [Linux 安装](install/linux.md) |
| macOS Apple Silicon | Shell 安装器 + LaunchAgent | [macOS 安装](install/macos.md) |
| Kindle PW2 | USB 解压 + KUAL | [Kindle 安装](install/kindle-pw2.md) |

安装命令使用官方仓库 `andycurise/kindle-kdash`；对应 Release 发布后即可下载。

## 2. 首次配置

服务端配置示例为 `configs/kdash.json.example`。安装器会生成独立 `kdash.json` 和随机 `settings_token`；不要直接长期使用示例配置。

最重要的字段：

```json
{
  "location": {
    "name": "西安",
    "lat": 34.3416,
    "lon": 108.9398
  },
  "timezone": "Asia/Shanghai",
  "listen_addr": ":8088",
  "rotate": 90,
  "weather_refresh_seconds": 1800,
  "offline_grace_minutes": 120
}
```

- `listen_addr`：`:8088` 表示监听所有网卡；仅本机测试可改为 `127.0.0.1:8088`。
- `rotate`：90 或 270，对应两种横放方向。
- `settings_token`：由安装器生成，Web 保存、SSH 刷新和远程前光切换使用；不要在文档或截图中公开。
- `weather_cache_path`：相对路径按配置文件目录解析。
- `kindle_ssh`：仅可选立即刷新使用，主同步链路始终通过 HTTP。

配置文件包含未知字段时会启动失败，以防拼写错误被静默忽略。旧架构的 `output_png`、`stay_awake_cmds` 等字段应删除。

## 3. Kindle 地址与令牌

Kindle `client.conf` 示例：

```sh
SERVER_URL="http://192.168.1.100:8088/screen.png"
REFRESH_SECONDS=60
GHOST_CLEAR_SECONDS=3600
TOUCH_PAUSE_SECONDS=180
FRONTLIGHT_MODE="schedule"
SETTINGS_TOKEN="与服务端 settings_token 相同"
```

- `REFRESH_SECONDS=60` 按下一分钟整点刷新；其他正整数表示指定秒数。
- 远端 `/client.conf` 只按字段白名单解析，绝不 `source` 远端文本。
- 不使用 KUAL 远程前光切换时，可以不在 Kindle 保存管理令牌。
- 活动 `client.conf` 不应进入 Git 或被发布包覆盖。

## 4. 安全边界

服务默认可在局域网提供只读图片，但管理写接口需要鉴权。空令牌只允许携带 `X-KDash-Client: kdash` 的本机回环请求完成首次设置，不允许远程写入或跨站表单触发修改。

当前 HTTP 不加密，令牌可能被同一网络窃听；只在可信 LAN 使用，不要将端口映射到公网。高安全环境应通过 HTTPS 反向代理并限制来源网络。详见 [SECURITY.md](../SECURITY.md)。

## 5. 验证顺序

1. 请求 `http://127.0.0.1:8088/api/status`，检查状态和版本。
2. 打开 WebUI，保存一次非默认端口或样式配置，重启后确认未丢失。
3. 请求 `/screen.png`，确认是 758×1024 的 8-bit Gray PNG。
4. 在另一台局域网设备上访问图片，检查防火墙和 AP 隔离。
5. 配置 Kindle 后检查客户端日志和实际 framebuffer，不只看 `eips` 返回码。
6. 分别验证停止恢复和服务重启；不同平台的验证结果不可互相替代。

## 6. 常见问题

### Kindle 拉取失败

检查服务端进程、`SERVER_URL`、主机防火墙、局域网 AP 隔离和实际 DHCP 地址。不要猜测 Kindle IP，也不要为了排查而启用 USBNetwork 开机启动。

### 保存返回 401

在 WebUI 填入当前 `settings_token`。如果令牌只通过 `KDASH_SETTINGS_TOKEN` 提供，使用该环境变量的值；它不会写入配置文件。

### 修改端口后仍访问旧端口

当前启动脚本只传 `-config`。停止旧实例并确认新配置已保存，再启动；检查是否仍有其他 kdash 实例占用旧端口。

### 天气或 AQI 失败

区分 DNS、CA、HTTP、JSON 和数组完整性，不要关闭 TLS 验证。空气质量失败显示未知，夜间 UV=0 是合法值。明文 HTTP 国内空气质量镜像默认不再优先使用。

### Kindle 图片拉伸

确保使用本版本 `/screen.png` 的灰度输出，并通过 `eips -g` 显示；不要使用 `eips 0 0 <path>` 代替。

### KUAL 显示 exception logged

读取 `/var/tmp/KUAL.log`、客户端日志和实际进程，不要只根据包装提示重装或删除个人数据。
