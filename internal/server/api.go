package server

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andycurise/kindle-kdash/internal/config"
	"github.com/andycurise/kindle-kdash/internal/dashboard"
	"github.com/andycurise/kindle-kdash/internal/display"
	"github.com/andycurise/kindle-kdash/internal/weather"
	"github.com/andycurise/kindle-kdash/web"
)

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWeb)
	mux.HandleFunc("/screen.png", s.handleScreenPNG)
	mux.HandleFunc("/preview.png", s.handlePreviewPNG)
	mux.HandleFunc("/client.conf", s.handleClientConfig)
	mux.HandleFunc("/api/set-frontlight", s.handleSetFrontlight)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/layout", s.handleLayout)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/sync-kindle", s.handleSyncKindle)
	mux.HandleFunc("/api/shutdown", s.handleShutdown)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		mux.ServeHTTP(w, r)
	})
}

func requireMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不受支持")
	return false
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"status": "error",
		"error":  map[string]string{"code": code, "message": message},
	})
}

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	path := r.URL.Path
	if path != "/" && path != "/app.css" && path != "/app.js" {
		http.NotFound(w, r)
		return
	}
	name := "index.html"
	contentType := "text/html; charset=utf-8"
	if path == "/app.css" {
		name = "app.css"
		contentType = "text/css; charset=utf-8"
	} else if path == "/app.js" {
		name = "app.js"
		contentType = "text/javascript; charset=utf-8"
	}
	body, err := web.Files.ReadFile(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "asset_unavailable", "设置页资源不可用")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if r.Method == http.MethodGet {
		_, _ = w.Write(body)
	}
}

func decodeDraft(r *http.Request, base *config.Config) (*config.Config, error) {
	encoded := r.URL.Query().Get("draft")
	if encoded == "" {
		return base.Clone(), nil
	}
	if len(encoded) > 128*1024 {
		return nil, fmt.Errorf("草稿配置过大")
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("草稿编码无效: %w", err)
	}
	candidate := config.Default()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(candidate); err != nil {
		return nil, fmt.Errorf("草稿 JSON 无效: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, err
	}
	candidate.BaseDir = base.BaseDir
	candidate.SettingsToken = base.SettingsToken
	candidate.RuntimeSettingsToken = base.RuntimeSettingsToken
	if candidate.KindleSSH.PrivateKeyPath == "" {
		candidate.KindleSSH.PrivateKeyPath = base.KindleSSH.PrivateKeyPath
	}
	if candidate.KindleSSH.KnownHostsPath == "" {
		candidate.KindleSSH.KnownHostsPath = base.KindleSSH.KnownHostsPath
	}
	if err := candidate.Validate(); err != nil {
		return nil, err
	}
	return candidate, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("请求只能包含一个 JSON 对象")
		}
		return fmt.Errorf("JSON 末尾无效: %w", err)
	}
	return nil
}

func (s *Server) updatePower(r *http.Request) {
	s.batteryMu.Lock()
	defer s.batteryMu.Unlock()
	if raw := r.URL.Query().Get("batt"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value >= 0 && value <= 100 {
			s.battery = value
			s.hasPower = true
		}
	}
	if raw := r.URL.Query().Get("charging"); raw != "" {
		s.charging = raw == "1" || strings.EqualFold(raw, "true")
	}
}

func (s *Server) renderPNG(cfg *config.Config, data dashboard.DashboardData, rotate bool) (*bytes.Buffer, error) {
	s.renderMu.Lock()
	landscape := s.dashboard.Render(data, cfg, s.landscapeW, s.landscapeH)
	s.renderMu.Unlock()

	var output image.Image = landscape
	if rotate {
		output = display.GrayImage(display.RotateForDevice(landscape, cfg.Rotate))
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, output); err != nil {
		return nil, err
	}
	return &buf, nil
}

func servePNG(w http.ResponseWriter, buf *bytes.Buffer, now time.Time) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Dash-Time", now.Format("2006-01-02 15:04:05"))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	_, _ = w.Write(buf.Bytes())
}

func (s *Server) handleScreenPNG(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	s.updatePower(r)
	cfg, err := decodeDraft(r, s.configSnapshot())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_draft", err.Error())
		return
	}
	if raw := r.URL.Query().Get("rotate"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || (value != 90 && value != 270) {
			writeError(w, http.StatusBadRequest, "invalid_rotate", "rotate 仅支持 90 或 270")
			return
		}
		cfg.Rotate = value
	}
	now := time.Now().In(s.loc)
	weatherData := s.weatherSnapshot()
	if r.URL.Query().Get("sample") == "1" || (r.URL.Query().Get("draft") != "" && weatherData == nil) {
		weatherData = weather.Example(now)
	}
	buf, err := s.renderPNG(cfg, s.dashboardData(now, weatherData), true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "render_failed", "屏幕渲染失败: "+err.Error())
		return
	}
	servePNG(w, buf, now)
}

func (s *Server) handlePreviewPNG(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	cfg, err := decodeDraft(r, s.configSnapshot())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_draft", err.Error())
		return
	}
	s.updatePower(r)
	now := time.Now().In(s.loc)
	weatherData := s.weatherSnapshot()
	if weatherData == nil || r.URL.Query().Get("sample") == "1" {
		weatherData = weather.Example(now)
	}
	buf, err := s.renderPNG(cfg, s.dashboardData(now, weatherData), false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "render_failed", "预览渲染失败: "+err.Error())
		return
	}
	servePNG(w, buf, now)
}

func (s *Server) handleLayout(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	cfg, err := decodeDraft(r, s.configSnapshot())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_draft", err.Error())
		return
	}
	now := time.Now().In(s.loc)
	weatherData := s.weatherSnapshot()
	if weatherData == nil || r.URL.Query().Get("sample") == "1" {
		weatherData = weather.Example(now)
	}
	data := s.dashboardData(now, weatherData)
	s.renderMu.Lock()
	layout := dashboard.ComputeLayout(data, cfg, s.landscapeW, s.landscapeH)
	s.renderMu.Unlock()
	writeJSON(w, http.StatusOK, layout)
}

func (s *Server) handleClientConfig(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	cfg := s.configSnapshot()
	body := fmt.Sprintf(`FRONTLIGHT_MODE=%s
DAY_BRIGHTNESS=%d
NIGHT_BRIGHTNESS=%d
DAY_START_HOUR=%d
NIGHT_START_HOUR=%d
GHOST_CLEAR_SECONDS=%d
`, cfg.FrontlightMode, cfg.DayBrightness, cfg.NightBrightness, cfg.DayStartHour, cfg.NightStartHour, cfg.GhostClearMinutes*60)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, body)
}

func (s *Server) handleSetFrontlight(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "设置访问令牌无效")
		return
	}
	mode := r.URL.Query().Get("mode")
	if mode != "schedule" && mode != "off" && mode != "on" {
		writeError(w, http.StatusBadRequest, "invalid_mode", "背光模式必须为 schedule, off 或 on")
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	candidate := s.cfg.Clone()
	if candidate.FrontlightMode != mode {
		candidate.FrontlightMode = mode
		if s.configPath != "" {
			if err := config.Save(s.configPath, candidate); err != nil {
				writeError(w, http.StatusInternalServerError, "save_failed", "背光设置保存失败: "+err.Error())
				return
			}
		}
		s.cfg = candidate
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"mode":   mode,
	})
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func (s *Server) authorized(r *http.Request) bool {
	token := s.configSnapshot().EffectiveSettingsToken()
	if token == "" {
		// A custom header cannot be sent by a cross-origin HTML form. Require it
		// for the loopback-only bootstrap exception to prevent browser CSRF.
		return isLoopbackRequest(r) && r.Header.Get("X-KDash-Client") == "kdash"
	}
	provided := r.Header.Get("X-KDash-Token")
	return subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"config":   publicConfig(s.configSnapshot()),
			"defaults": publicConfig(config.Default()),
		})
		return
	case http.MethodPut:
		// Continue below.
	default:
		requireMethod(w, r, http.MethodGet, http.MethodPut)
		return
	}
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "设置访问令牌无效")
		return
	}
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "json_required", "设置保存需要 application/json")
		return
	}

	current, candidate, ok := s.saveConfigRequest(w, r)
	if !ok {
		return
	}

	response := map[string]any{
		"status":           "ok",
		"message":          "设置已保存到服务端",
		"config":           publicConfig(candidate),
		"restart_required": candidate.ListenAddr != current.ListenAddr,
	}
	if r.URL.Query().Get("refresh") == "1" {
		response["refresh"] = s.refreshResult(r.Context(), candidate)
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) saveConfigRequest(w http.ResponseWriter, r *http.Request) (*config.Config, *config.Config, bool) {
	// Serialize the complete read/decode/write/commit transaction with frontlight
	// updates. The optional SSH refresh runs after this lock is released.
	s.configMu.Lock()
	defer s.configMu.Unlock()

	current := s.cfg.Clone()
	candidate := config.Default()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(candidate); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "配置 JSON 无效: "+err.Error())
		return nil, nil, false
	}
	if err := ensureJSONEnd(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return nil, nil, false
	}
	candidate.BaseDir = current.BaseDir
	candidate.SettingsToken = current.SettingsToken
	candidate.RuntimeSettingsToken = current.RuntimeSettingsToken
	if candidate.KindleSSH.PrivateKeyPath == "" {
		candidate.KindleSSH.PrivateKeyPath = current.KindleSSH.PrivateKeyPath
	}
	if candidate.KindleSSH.KnownHostsPath == "" {
		candidate.KindleSSH.KnownHostsPath = current.KindleSSH.KnownHostsPath
	}
	if err := candidate.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_config", err.Error())
		return nil, nil, false
	}
	if err := config.Save(s.configPath, candidate); err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", "配置保存失败: "+err.Error())
		return nil, nil, false
	}
	s.cfg = candidate.Clone()
	return current, candidate, true
}

func (s *Server) refreshResult(ctx context.Context, cfg *config.Config) map[string]any {
	if !cfg.KindleSSH.Enabled {
		return map[string]any{
			"status":  "waiting_for_poll",
			"message": "设置已生效；Kindle 将在下次 HTTP 拉取时更新",
		}
	}
	refreshCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	output, err := s.refresh(refreshCtx, cfg)
	if err != nil {
		return map[string]any{
			"status":  "error",
			"message": "设置已保存，立即刷新失败；Kindle 仍会在下次 HTTP 拉取时更新",
			"error":   err.Error(),
			"output":  output,
		}
	}
	return map[string]any{
		"status":  "command_completed",
		"message": "立即刷新命令已完成；实际画面请以 framebuffer 为准",
		"output":  output,
	}
}

func (s *Server) handleSyncKindle(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "设置访问令牌无效")
		return
	}
	cfg := s.configSnapshot()
	if !cfg.KindleSSH.Enabled {
		writeError(w, http.StatusConflict, "refresh_disabled", "未启用 SSH 立即刷新；Kindle 会通过 HTTP 自动更新")
		return
	}
	result := s.refreshResult(r.Context(), cfg)
	status := http.StatusOK
	if result["status"] == "error" {
		status = http.StatusBadGateway
	}
	writeJSON(w, status, result)
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !isLoopbackRequest(r) {
		writeError(w, http.StatusForbidden, "loopback_required", "优雅停止仅允许本机调用")
		return
	}
	if !s.authorized(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "设置访问令牌无效")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "stopping"})
	s.requestShutdown()
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	cfg := s.configSnapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"version":       Version,
		"server_time":   time.Now().In(s.loc).Format("2006-01-02 15:04:05 MST"),
		"uptime_sec":    int64(time.Since(s.startTime).Seconds()),
		"city":          cfg.Location.Name,
		"screen_w":      s.portraitW,
		"screen_h":      s.portraitH,
		"landscape_w":   s.landscapeW,
		"landscape_h":   s.landscapeH,
		"weather":       s.weatherSnapshot(),
		"ssh_refresh":   cfg.KindleSSH.Enabled,
		"auth_required": cfg.EffectiveSettingsToken() != "",
	})
}
