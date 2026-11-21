package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultWeatherURL   = "https://api.open-meteo.com/v1/forecast"
	defaultWeatherCache = "data/cache_weather.json"
)

// Location 保留看板原有的地理位置 JSON 格式。
type Location struct {
	Name     string  `json:"name"`
	CityCode string  `json:"city_code,omitempty"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
}

// KindleSSH 是 主机 服务连接 Kindle HTTP 客户端所需的 SSH 配置。
type KindleSSH struct {
	Enabled        bool   `json:"enabled"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	PrivateKeyPath string `json:"private_key_path"`
	KnownHostsPath string `json:"known_hosts_path"`
}

// Config 只包含 主机 渲染服务和 Kindle 客户端协作所需配置。
type Config struct {
	Location  Location `json:"location"`
	Timezone  string   `json:"timezone"`
	ClockFont string   `json:"clock_font,omitempty"`
	ClockSize int      `json:"clock_size,omitempty"`
	ClockXOff int      `json:"clock_x_off,omitempty"`
	ClockYOff int      `json:"clock_y_off,omitempty"`
	DateFont  string   `json:"date_font,omitempty"`
	DateSize  int      `json:"date_size,omitempty"`
	DateXOff  int      `json:"date_x_off,omitempty"`
	DateYOff  int      `json:"date_y_off,omitempty"`

	FrontlightMode  string `json:"frontlight_mode,omitempty"`
	DayBrightness   int    `json:"day_brightness"`
	NightBrightness int    `json:"night_brightness"`
	DayStartHour    int    `json:"day_start_hour"`
	NightStartHour  int    `json:"night_start_hour"`

	BottomRightModule string `json:"bottom_right_module,omitempty"`
	QuoteType         string `json:"quote_type,omitempty"`
	QuoteFont         string `json:"quote_font,omitempty"`
	QuoteSize         int    `json:"quote_size,omitempty"`
	QuoteXOff         int    `json:"quote_x_off,omitempty"`
	QuoteYOff         int    `json:"quote_y_off,omitempty"`

	ShowCurrentWeather bool `json:"show_current_weather"`
	ShowForecast       bool `json:"show_forecast"`
	ShowLunar          bool `json:"show_lunar"`
	ShowQuote          bool `json:"show_quote"`
	ShowFooter         bool `json:"show_footer"`

	WeatherRefreshSeconds int    `json:"weather_refresh_seconds"`
	GhostClearMinutes     int    `json:"ghost_clear_minutes"`
	Rotate                int    `json:"rotate"`
	ScreenW               int    `json:"screen_w"`
	ScreenH               int    `json:"screen_h"`
	WeatherCachePath      string `json:"weather_cache_path,omitempty"`
	WeatherURL            string `json:"weather_url,omitempty"`
	OfflineGraceMin       int    `json:"offline_grace_minutes"`

	ListenAddr    string    `json:"listen_addr"`
	SettingsToken string    `json:"settings_token,omitempty"`
	KindleSSH     KindleSSH `json:"kindle_ssh"`

	// RuntimeSettingsToken 仅来自环境变量，绝不写入 API 响应或配置文件。
	RuntimeSettingsToken string `json:"-"`
	// BaseDir 不参与序列化；相对缓存路径始终相对配置文件目录解析。
	BaseDir string `json:"-"`
}

// Default 返回一份独立的默认配置。
func Default() *Config {
	return &Config{
		Location:              Location{Name: "西安", Lat: 34.3416, Lon: 108.9398},
		Timezone:              "Asia/Shanghai",
		ClockFont:             "bebas",
		DateFont:              "bold",
		FrontlightMode:        "schedule",
		DayBrightness:         10,
		NightBrightness:       0,
		DayStartHour:          7,
		NightStartHour:        22,
		BottomRightModule:     "solar",
		QuoteType:             "mao",
		QuoteFont:             "bold",
		ShowCurrentWeather:    true,
		ShowForecast:          true,
		ShowLunar:             true,
		ShowQuote:             true,
		ShowFooter:            true,
		WeatherRefreshSeconds: 1800,
		GhostClearMinutes:     60,
		Rotate:                90,
		ScreenW:               758,
		ScreenH:               1024,
		WeatherCachePath:      defaultWeatherCache,
		WeatherURL:            defaultWeatherURL,
		OfflineGraceMin:       120,
		ListenAddr:            ":8088",
		KindleSSH: KindleSSH{
			Port: 2222,
			User: "root",
		},
		BaseDir: ".",
	}
}

// Clone 返回配置副本。当前配置没有切片或映射字段，结构复制即为深拷贝。
func (cfg *Config) Clone() *Config {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	return &clone
}

// EffectiveSettingsToken 返回当前进程实际使用的管理令牌。
func (cfg *Config) EffectiveSettingsToken() string {
	if cfg == nil {
		return ""
	}
	if cfg.RuntimeSettingsToken != "" {
		return cfg.RuntimeSettingsToken
	}
	return cfg.SettingsToken
}

// ResolvePath 将相对路径解析到配置文件所在目录。
func (cfg *Config) ResolvePath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	base := "."
	if cfg != nil && cfg.BaseDir != "" {
		base = cfg.BaseDir
	}
	return filepath.Clean(filepath.Join(base, path))
}

// Load 在默认配置副本上解码 JSON，因此缺失的 bool 字段保持 true。
func Load(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("配置路径不能为空")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 %s: %w", path, err)
	}
	cfg := Default()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("解析 %s: %w", path, err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, fmt.Errorf("解析 %s: %w", path, err)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("解析配置路径 %s: %w", path, err)
	}
	cfg.BaseDir = filepath.Dir(absolutePath)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("校验 %s: %w", path, err)
	}
	return cfg, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("配置只能包含一个 JSON 对象")
		}
		return fmt.Errorf("JSON 末尾无效: %w", err)
	}
	return nil
}

// Save 校验并原子写入配置副本，任何失败都不会修改调用方传入的对象。
func Save(path string, cfg *Config) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("配置路径不能为空")
	}
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}
	candidate := cfg.Clone()
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("校验配置: %w", err)
	}
	data, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置: %w", err)
	}
	if err := WriteFileAtomic(path, data, 0600); err != nil {
		return fmt.Errorf("写入 %s: %w", path, err)
	}
	return nil
}

// Validate 校验所有枚举和数值边界。仅在全部校验成功后补 SSH 默认端口和用户名。
func (cfg *Config) Validate() error {
	if cfg == nil {
		return fmt.Errorf("配置不能为空")
	}
	candidate := *cfg
	if candidate.KindleSSH.Port == 0 {
		candidate.KindleSSH.Port = 2222
	}
	if strings.TrimSpace(candidate.KindleSSH.User) == "" {
		candidate.KindleSSH.User = "root"
	}
	if err := candidate.validate(); err != nil {
		return err
	}
	cfg.KindleSSH = candidate.KindleSSH
	return nil
}

func (cfg *Config) validate() error {
	if strings.TrimSpace(cfg.Timezone) == "" {
		return fmt.Errorf("timezone 不能为空")
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return fmt.Errorf("timezone %q 无效: %w", cfg.Timezone, err)
	}
	if !finiteInRange(cfg.Location.Lat, -90, 90) {
		return fmt.Errorf("location.lat 必须在 -90 到 90 之间")
	}
	if !finiteInRange(cfg.Location.Lon, -180, 180) {
		return fmt.Errorf("location.lon 必须在 -180 到 180 之间")
	}

	if err := enum("clock_font", cfg.ClockFont, "bebas", "jetbrains", "montserrat", "playfair", "noto"); err != nil {
		return err
	}
	if err := enum("date_font", cfg.DateFont, "bold", "med", "reg", "serif", "bebas", "jetbrains", "montserrat"); err != nil {
		return err
	}
	if err := enum("quote_font", cfg.QuoteFont, "bold", "med", "reg", "kai", "wei", "xing", "lively"); err != nil {
		return err
	}
	if err := enum("quote_type", cfg.QuoteType, "mao", "almanac", "famous", "inspirational", "poetry"); err != nil {
		return err
	}
	if err := enum("frontlight_mode", cfg.FrontlightMode, "schedule", "off", "on"); err != nil {
		return err
	}
	if err := enum("bottom_right_module", cfg.BottomRightModule, "solar", "air"); err != nil {
		return err
	}

	if err := bounded("clock_size", cfg.ClockSize, 0, 400); err != nil {
		return err
	}
	if err := bounded("date_size", cfg.DateSize, 0, 200); err != nil {
		return err
	}
	if err := bounded("quote_size", cfg.QuoteSize, 0, 60); err != nil {
		return err
	}
	for name, value := range map[string]int{
		"clock_x_off": cfg.ClockXOff,
		"clock_y_off": cfg.ClockYOff,
		"date_x_off":  cfg.DateXOff,
		"date_y_off":  cfg.DateYOff,
		"quote_x_off": cfg.QuoteXOff,
		"quote_y_off": cfg.QuoteYOff,
	} {
		if err := bounded(name, value, -200, 200); err != nil {
			return err
		}
	}
	if err := bounded("day_brightness", cfg.DayBrightness, 0, 24); err != nil {
		return err
	}
	if err := bounded("night_brightness", cfg.NightBrightness, 0, 24); err != nil {
		return err
	}
	if err := bounded("day_start_hour", cfg.DayStartHour, 0, 23); err != nil {
		return err
	}
	if err := bounded("night_start_hour", cfg.NightStartHour, 0, 23); err != nil {
		return err
	}
	if err := bounded("weather_refresh_seconds", cfg.WeatherRefreshSeconds, 60, 86400); err != nil {
		return err
	}
	if err := bounded("ghost_clear_minutes", cfg.GhostClearMinutes, 0, 525600); err != nil {
		return err
	}
	if err := bounded("offline_grace_minutes", cfg.OfflineGraceMin, 0, 525600); err != nil {
		return err
	}
	if cfg.Rotate != 90 && cfg.Rotate != 270 {
		return fmt.Errorf("rotate 只能为 90 或 270")
	}
	if err := bounded("screen_w", cfg.ScreenW, 0, 8192); err != nil {
		return err
	}
	if err := bounded("screen_h", cfg.ScreenH, 0, 8192); err != nil {
		return err
	}
	if (cfg.ScreenW == 0) != (cfg.ScreenH == 0) {
		return fmt.Errorf("screen_w/screen_h 必须同时为正数，或同时为 0")
	}
	if strings.TrimSpace(cfg.WeatherCachePath) == "" {
		return fmt.Errorf("weather_cache_path 不能为空")
	}
	if err := validateHTTPURL("weather_url", cfg.WeatherURL); err != nil {
		return err
	}
	if err := validateListenAddr(cfg.ListenAddr); err != nil {
		return err
	}
	if err := bounded("kindle_ssh.port", cfg.KindleSSH.Port, 1, 65535); err != nil {
		return err
	}
	if cfg.KindleSSH.Enabled {
		if strings.TrimSpace(cfg.KindleSSH.Host) == "" {
			return fmt.Errorf("kindle_ssh.enabled 为 true 时 host 不能为空")
		}
		if strings.TrimSpace(cfg.KindleSSH.PrivateKeyPath) == "" {
			return fmt.Errorf("kindle_ssh.enabled 为 true 时 private_key_path 不能为空")
		}
		if strings.TrimSpace(cfg.KindleSSH.KnownHostsPath) == "" {
			return fmt.Errorf("kindle_ssh.enabled 为 true 时 known_hosts_path 不能为空")
		}
	}
	if strings.ContainsAny(cfg.KindleSSH.Host, " \t\r\n") || strings.HasPrefix(cfg.KindleSSH.Host, "-") {
		return fmt.Errorf("kindle_ssh.host 包含非法字符")
	}
	if strings.ContainsAny(cfg.KindleSSH.User, " \t\r\n@") || strings.HasPrefix(cfg.KindleSSH.User, "-") {
		return fmt.Errorf("kindle_ssh.user 包含非法字符")
	}
	return nil
}

func enum(name, value string, allowed ...string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("%s %q 无效，可选值: %s", name, value, strings.Join(allowed, ", "))
}

func bounded(name string, value, min, max int) error {
	if value < min || value > max {
		return fmt.Errorf("%s 必须在 %d 到 %d 之间，当前为 %d", name, min, max, value)
	}
	return nil
}

func finiteInRange(value, min, max float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= min && value <= max
}

func validateHTTPURL(name, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s 无效: %w", name, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%s 必须是有效的 http/https 地址", name)
	}
	return nil
}

func validateListenAddr(addr string) error {
	_, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("listen_addr %q 无效: %w", addr, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("listen_addr 端口必须在 1 到 65535 之间")
	}
	return nil
}
