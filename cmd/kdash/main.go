package main

// kdash runs on an always-on Windows, Linux, or macOS host. Kindle clients
// only download the rendered grayscale PNG over the local network.

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andycurise/kindle-kdash/internal/config"
	"github.com/andycurise/kindle-kdash/internal/dashboard"
	"github.com/andycurise/kindle-kdash/internal/display"
	"github.com/andycurise/kindle-kdash/internal/server"
	"github.com/andycurise/kindle-kdash/internal/weather"

	_ "time/tzdata"
)

var version = "0.4.0"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configPath := flag.String("config", "kdash.json", "服务端配置文件路径")
	listen := flag.String("listen", "", "HTTP 监听地址，例如 :8088；默认读取配置")
	serve := flag.String("serve", "", "兼容旧启动脚本：等同 -listen")
	offline := flag.Bool("offline", false, "仅开发：使用示例天气，不发起天气网络请求")
	preview := flag.String("preview", "", "输出旋转后的 Kindle 灰度 PNG 后退出，不联网")
	previewLandscape := flag.String("preview-landscape", "", "输出未旋转的横版 PNG 后退出")
	screen := flag.String("screen", "", "预览物理竖屏尺寸，例如 758x1024")
	showVersion := flag.Bool("version", false, "显示版本后退出")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return nil
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	if token := os.Getenv("KDASH_SETTINGS_TOKEN"); token != "" {
		cfg.RuntimeSettingsToken = token
	}
	if *listen != "" {
		cfg.ListenAddr = *listen
	} else if *serve != "" {
		cfg.ListenAddr = *serve
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("启动参数或配置无效: %w", err)
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return fmt.Errorf("时区加载失败 %q: %w", cfg.Timezone, err)
	}

	portraitW, portraitH := cfg.ScreenW, cfg.ScreenH
	if *screen != "" {
		portraitW, portraitH, err = display.ParseScreenSize(*screen)
		if err != nil {
			return err
		}
	}
	if portraitW == 0 || portraitH == 0 {
		portraitW, portraitH = 758, 1024
	}
	landscapeW, landscapeH := portraitH, portraitW

	dash, err := dashboard.New()
	if err != nil {
		return fmt.Errorf("初始化渲染器失败: %w", err)
	}
	log.Printf("kdash %s 城市=%s 横版画布=%dx%d Kindle灰度图=%dx%d", version, cfg.Location.Name, landscapeW, landscapeH, portraitW, portraitH)

	if *preview != "" || *previewLandscape != "" {
		return renderPreviews(dash, cfg, loc, *preview, *previewLandscape, landscapeW, landscapeH)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server.Version = version
	srv := server.New(cfg, loc, dash, landscapeW, landscapeH, portraitW, portraitH, *configPath)
	if *offline {
		srv.UseStaticWeather(weather.Example(time.Now().In(loc)))
		log.Print("开发离线模式：使用示例天气，不发起天气网络请求")
	}
	return srv.Run(ctx)
}

func renderPreviews(dash *dashboard.Dashboard, cfg *config.Config, loc *time.Location, portraitPath, landscapePath string, landscapeW, landscapeH int) error {
	now := time.Now().In(loc)
	weatherData := weather.LoadCached(cfg, now)
	if weatherData == nil {
		weatherData = weather.Example(now)
	}
	data := dashboard.DashboardData{
		Now:        now,
		Day:        dashboard.DayInfo(now),
		Weather:    weatherData,
		Battery:    85,
		HasBattery: true,
	}
	landscape := dash.Render(data, cfg, landscapeW, landscapeH)

	if portraitPath != "" {
		var buf bytes.Buffer
		portrait := display.GrayImage(display.RotateForDevice(landscape, cfg.Rotate))
		if err := png.Encode(&buf, portrait); err != nil {
			return fmt.Errorf("设备预览编码失败: %w", err)
		}
		if err := config.WriteFileAtomic(portraitPath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("写设备预览失败: %w", err)
		}
		log.Printf("设备方向预览已写入 %s (%d 字节)", portraitPath, buf.Len())
	}
	if landscapePath != "" {
		var buf bytes.Buffer
		if err := png.Encode(&buf, landscape); err != nil {
			return fmt.Errorf("横版预览编码失败: %w", err)
		}
		if err := config.WriteFileAtomic(landscapePath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("写横版预览失败: %w", err)
		}
		log.Printf("横版预览已写入 %s (%d 字节)", landscapePath, buf.Len())
	}
	return nil
}
