package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultAndLoadPreserveMissingAndExplicitBooleans(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("默认配置无效: %v", err)
	}
	if !cfg.ShowCurrentWeather || !cfg.ShowForecast || !cfg.ShowLunar || !cfg.ShowQuote || !cfg.ShowFooter {
		t.Fatalf("显示模块默认应全部开启: %+v", cfg)
	}
	if cfg.ListenAddr != ":8088" || cfg.WeatherCachePath != "data/cache_weather.json" {
		t.Fatalf("服务默认值错误: %+v", cfg)
	}
	if cfg.KindleSSH.Port != 2222 || cfg.KindleSSH.User != "root" {
		t.Fatalf("SSH 默认值错误: %+v", cfg.KindleSSH)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "kdash.json")
	data := []byte(`{
		"location":{"name":"西安","lat":34.3416,"lon":108.9398},
		"show_current_weather":false,
		"show_quote":false,
		"day_brightness":0,
		"night_brightness":0,
		"day_start_hour":0,
		"night_start_hour":0,
		"kindle_ssh":{"enabled":false,"port":0,"user":""}
	}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if loaded.ShowCurrentWeather || loaded.ShowQuote {
		t.Fatal("显式 false 不应被默认值覆盖")
	}
	if !loaded.ShowForecast || !loaded.ShowLunar || !loaded.ShowFooter {
		t.Fatal("JSON 缺失的显示开关应保持 true")
	}
	if loaded.DayBrightness != 0 || loaded.NightBrightness != 0 || loaded.DayStartHour != 0 || loaded.NightStartHour != 0 {
		t.Fatalf("合法的 0 亮度/小时被修改: %+v", loaded)
	}
	if loaded.KindleSSH.Port != 2222 || loaded.KindleSSH.User != "root" {
		t.Fatalf("SSH 缺省值未补齐: %+v", loaded.KindleSSH)
	}
	if loaded.BaseDir != dir {
		t.Fatalf("BaseDir=%q，期望 %q", loaded.BaseDir, dir)
	}
	if got, want := loaded.ResolvePath(loaded.WeatherCachePath), filepath.Join(dir, "data", "cache_weather.json"); got != want {
		t.Fatalf("ResolvePath=%q，期望 %q", got, want)
	}
}

func TestValidateRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"timezone", func(c *Config) { c.Timezone = "Not/A_Real_Zone" }},
		{"latitude", func(c *Config) { c.Location.Lat = 91 }},
		{"longitude", func(c *Config) { c.Location.Lon = -181 }},
		{"clock font", func(c *Config) { c.ClockFont = "unknown" }},
		{"date font", func(c *Config) { c.DateFont = "unknown" }},
		{"quote font", func(c *Config) { c.QuoteFont = "unknown" }},
		{"quote type", func(c *Config) { c.QuoteType = "unknown" }},
		{"frontlight", func(c *Config) { c.FrontlightMode = "auto" }},
		{"bottom module", func(c *Config) { c.BottomRightModule = "unknown" }},
		{"clock size", func(c *Config) { c.ClockSize = -1 }},
		{"date size", func(c *Config) { c.DateSize = 201 }},
		{"quote size", func(c *Config) { c.QuoteSize = 61 }},
		{"offset", func(c *Config) { c.ClockXOff = 201 }},
		{"negative offset", func(c *Config) { c.QuoteYOff = -201 }},
		{"brightness", func(c *Config) { c.DayBrightness = 25 }},
		{"hour", func(c *Config) { c.NightStartHour = 24 }},
		{"refresh", func(c *Config) { c.WeatherRefreshSeconds = 59 }},
		{"ghost clear", func(c *Config) { c.GhostClearMinutes = -1 }},
		{"offline grace", func(c *Config) { c.OfflineGraceMin = -1 }},
		{"rotate", func(c *Config) { c.Rotate = 180 }},
		{"screen pair", func(c *Config) { c.ScreenH = 0 }},
		{"negative screen", func(c *Config) { c.ScreenW = -1 }},
		{"cache path", func(c *Config) { c.WeatherCachePath = "" }},
		{"weather URL", func(c *Config) { c.WeatherURL = "file:///tmp/weather.json" }},
		{"listen address", func(c *Config) { c.ListenAddr = "8088" }},
		{"listen port", func(c *Config) { c.ListenAddr = ":0" }},
		{"SSH port", func(c *Config) { c.KindleSSH.Port = 65536 }},
		{"SSH host", func(c *Config) { c.KindleSSH.Enabled = true }},
		{"SSH key", func(c *Config) {
			c.KindleSSH.Enabled = true
			c.KindleSSH.Host = "kindle.example.test"
		}},
		{"SSH known hosts", func(c *Config) {
			c.KindleSSH.Enabled = true
			c.KindleSSH.Host = "kindle.example.test"
			c.KindleSSH.PrivateKeyPath = "key"
		}},
		{"SSH host option injection", func(c *Config) {
			c.KindleSSH.Host = "-oProxyCommand=bad"
		}},
		{"SSH user whitespace", func(c *Config) {
			c.KindleSSH.User = "root admin"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("非法配置应返回错误")
			}
		})
	}
}

func TestValidateAcceptsBoundaryValues(t *testing.T) {
	cfg := Default()
	cfg.ClockSize = 0
	cfg.DateSize = 0
	cfg.QuoteSize = 0
	cfg.ClockXOff = -200
	cfg.ClockYOff = 200
	cfg.DayBrightness = 0
	cfg.NightBrightness = 0
	cfg.DayStartHour = 0
	cfg.NightStartHour = 0
	cfg.GhostClearMinutes = 0
	cfg.OfflineGraceMin = 0
	cfg.Rotate = 270
	cfg.ScreenW = 0
	cfg.ScreenH = 0
	cfg.KindleSSH = KindleSSH{
		Enabled:        true,
		Host:           "kindle.example.test",
		Port:           0,
		User:           "",
		PrivateKeyPath: "id_ed25519",
		KnownHostsPath: "known_hosts",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("边界值应合法: %v", err)
	}
	if cfg.KindleSSH.Port != 2222 || cfg.KindleSSH.User != "root" {
		t.Fatalf("Validate 未补 SSH 默认值: %+v", cfg.KindleSSH)
	}
}

func TestCloneSaveRoundTripAndInputImmutability(t *testing.T) {
	cfg := Default()
	cfg.ShowForecast = false
	cfg.DayBrightness = 0
	cfg.NightBrightness = 0
	cfg.DayStartHour = 0
	cfg.NightStartHour = 0
	cfg.KindleSSH = KindleSSH{
		Enabled:        true,
		Host:           "kindle.example.test",
		Port:           0,
		User:           "",
		PrivateKeyPath: "id_ed25519",
		KnownHostsPath: "known_hosts",
	}
	before := cfg.Clone()
	path := filepath.Join(t.TempDir(), "nested", "kdash.json")
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}
	if !reflect.DeepEqual(cfg, before) {
		t.Fatalf("Save 修改了输入:\n got=%+v\nwant=%+v", cfg, before)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "base_dir") || strings.Contains(string(data), "output_png") || strings.Contains(string(data), "stay_awake_cmds") {
		t.Fatalf("配置包含不应序列化的字段: %s", data)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	ssh := raw["kindle_ssh"].(map[string]any)
	if ssh["port"] != float64(2222) || ssh["user"] != "root" {
		t.Fatalf("保存的 SSH 默认值错误: %+v", ssh)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}
	if loaded.ShowForecast || loaded.KindleSSH.Host != "kindle.example.test" {
		t.Fatalf("配置往返错误: %+v", loaded)
	}
	if loaded.DayBrightness != 0 || loaded.NightBrightness != 0 || loaded.DayStartHour != 0 || loaded.NightStartHour != 0 {
		t.Fatalf("保存往返丢失了合法零值: %+v", loaded)
	}

	clone := loaded.Clone()
	clone.Location.Name = "另一个城市"
	if loaded.Location.Name == clone.Location.Name {
		t.Fatal("Clone 未返回独立副本")
	}
}

func TestLoadRejectsUnknownFieldsAndMultipleObjects(t *testing.T) {
	for name, data := range map[string]string{
		"unknown field":    `{"listen_addr":":8088","listen_addr_typo":":9000"}`,
		"multiple objects": `{}` + "\n" + `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "kdash.json")
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("invalid configuration should be rejected")
			}
		})
	}
}

func TestSaveUsesPrivatePermissionsAndSkipsRuntimeToken(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not authoritative on Windows")
	}
	path := filepath.Join(t.TempDir(), "配置 目录", "kdash.json")
	cfg := Default()
	cfg.SettingsToken = "file-secret"
	cfg.RuntimeSettingsToken = "environment-secret"
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "environment-secret") || strings.Contains(string(data), "RuntimeSettingsToken") {
		t.Fatalf("runtime token was persisted: %s", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("configuration mode=%#o, want 0600", got)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("configuration mode widened to %#o", got)
	}
}

func TestSaveRejectsInvalidConfigWithoutWritingOrMutating(t *testing.T) {
	cfg := Default()
	cfg.FrontlightMode = "invalid; rm -rf /"
	before := cfg.Clone()
	path := filepath.Join(t.TempDir(), "kdash.json")
	if err := Save(path, cfg); err == nil {
		t.Fatal("非法配置不应保存")
	}
	if !reflect.DeepEqual(cfg, before) {
		t.Fatal("失败的 Save 修改了输入")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("非法配置不应生成文件，stat err=%v", err)
	}
}
