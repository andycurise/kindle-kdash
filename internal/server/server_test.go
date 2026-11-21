package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andycurise/kindle-kdash/internal/config"
	"github.com/andycurise/kindle-kdash/internal/dashboard"
	"github.com/andycurise/kindle-kdash/internal/weather"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Location = config.Location{Name: "测试城市", Lat: 34.3416, Lon: 108.9398}
	cfg.WeatherCachePath = filepath.Join(dir, "weather-cache.json")
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	dash, err := dashboard.New()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "kdash.json")
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	srv := New(cfg, loc, dash, 1024, 758, 758, 1024, path)
	srv.weather = weather.Example(time.Now().In(loc))
	return srv, path
}

func request(t *testing.T, handler http.Handler, method, target string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:43210"
	req.Header.Set("X-KDash-Client", "kdash")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func TestReadOnlyEndpoints(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	for _, path := range []string{"/", "/app.css", "/app.js", "/api/config", "/api/layout", "/api/status", "/client.conf"} {
		response := request(t, handler, http.MethodGet, path, nil, nil)
		if response.Code != http.StatusOK {
			t.Errorf("GET %s: status=%d body=%s", path, response.Code, response.Body.String())
		}
	}

	response := request(t, handler, http.MethodGet, "/screen.png?batt=81&charging=1", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("screen status=%d body=%s", response.Code, response.Body.String())
	}
	img, err := png.Decode(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 758 || img.Bounds().Dy() != 1024 {
		t.Fatalf("screen bounds=%v", img.Bounds())
	}
	if _, ok := img.(*image.Gray); !ok {
		t.Fatalf("screen color model=%T, want *image.Gray", img)
	}

	response = request(t, handler, http.MethodGet, "/preview.png", nil, nil)
	img, err = png.Decode(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 1024 || img.Bounds().Dy() != 758 {
		t.Fatalf("preview bounds=%v", img.Bounds())
	}
	response = request(t, handler, http.MethodGet, "/screen.png?rotate=180", nil, nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unsupported rotate status=%d", response.Code)
	}
}

func TestSaveSettingsRoundTripPreservesZero(t *testing.T) {
	srv, path := newTestServer(t)
	candidate := srv.configSnapshot()
	candidate.DayBrightness = 0
	candidate.NightBrightness = 0
	candidate.DayStartHour = 0
	candidate.NightStartHour = 0
	candidate.ClockFont = "playfair"
	body, _ := json.Marshal(candidate)

	response := request(t, srv.Handler(), http.MethodPut, "/api/config", body, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("save status=%d body=%s", response.Code, response.Body.String())
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DayBrightness != 0 || loaded.DayStartHour != 0 || loaded.NightStartHour != 0 {
		t.Fatalf("zero values were not preserved: %+v", loaded)
	}
	if loaded.ClockFont != "playfair" {
		t.Fatalf("clock font=%q", loaded.ClockFont)
	}

	before, _ := os.ReadFile(path)
	candidate.FrontlightMode = "schedule\"\nMALICIOUS=1"
	body, _ = json.Marshal(candidate)
	response = request(t, srv.Handler(), http.MethodPut, "/api/config", body, nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid enum status=%d body=%s", response.Code, response.Body.String())
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("invalid save changed the configuration file")
	}
	if srv.configSnapshot().FrontlightMode != "schedule" {
		t.Fatal("invalid save changed the in-memory configuration")
	}
}

func TestSetFrontlightEndpoint(t *testing.T) {
	srv, path := newTestServer(t)
	response := request(t, srv.Handler(), http.MethodPost, "/api/set-frontlight?mode=off", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("set frontlight status=%d body=%s", response.Code, response.Body.String())
	}
	if srv.configSnapshot().FrontlightMode != "off" {
		t.Fatalf("expected frontlight mode to be off, got %s", srv.configSnapshot().FrontlightMode)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.FrontlightMode != "off" {
		t.Fatalf("expected persisted frontlight mode to be off, got %s", loaded.FrontlightMode)
	}

	// Invalid mode rejected
	response = request(t, srv.Handler(), http.MethodPost, "/api/set-frontlight?mode=invalid", nil, nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid mode, got %d", response.Code)
	}
}

func TestFrontlightRejectsGETAndRemoteWithoutToken(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()
	response := request(t, handler, http.MethodGet, "/api/set-frontlight?mode=off", nil, nil)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET frontlight status=%d, want 405", response.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/set-frontlight?mode=off", nil)
	req.RemoteAddr = "192.0.2.20:1234"
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("remote frontlight without token status=%d, want 401", response.Code)
	}

	srv.cfg.SettingsToken = "lan-secret"
	req = httptest.NewRequest(http.MethodPost, "/api/set-frontlight?mode=off", nil)
	req.RemoteAddr = "192.0.2.20:1234"
	req.Header.Set("X-KDash-Token", "lan-secret")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("remote frontlight with token status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestFrontlightSaveFailureDoesNotChangeMemory(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.configPath = t.TempDir()
	response := request(t, srv.Handler(), http.MethodPost, "/api/set-frontlight?mode=off", nil, nil)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("frontlight save failure status=%d body=%s", response.Code, response.Body.String())
	}
	if got := srv.configSnapshot().FrontlightMode; got != "schedule" {
		t.Fatalf("frontlight memory changed after failed save: %q", got)
	}
}

func TestEnvironmentSettingsTokenNeverPersisted(t *testing.T) {
	srv, path := newTestServer(t)
	const environmentToken = "environment-only-secret"
	srv.cfg.RuntimeSettingsToken = environmentToken
	candidate := srv.configSnapshot()
	candidate.ClockFont = "montserrat"
	body, _ := json.Marshal(candidate)
	response := request(t, srv.Handler(), http.MethodPut, "/api/config", body, map[string]string{"X-KDash-Token": environmentToken})
	if response.Code != http.StatusOK {
		t.Fatalf("save with environment token status=%d body=%s", response.Code, response.Body.String())
	}
	response = request(t, srv.Handler(), http.MethodPost, "/api/set-frontlight?mode=off", nil, map[string]string{"X-KDash-Token": environmentToken})
	if response.Code != http.StatusOK {
		t.Fatalf("frontlight with environment token status=%d body=%s", response.Code, response.Body.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), environmentToken) || strings.Contains(string(data), "runtime_settings_token") {
		t.Fatalf("environment token leaked into configuration: %s", data)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SettingsToken != "" || loaded.RuntimeSettingsToken != "" {
		t.Fatalf("persisted tokens are not empty: %+v", loaded)
	}
}

func TestShutdownRequiresLoopbackAndClosesOnce(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.RemoteAddr = "192.0.2.20:1234"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusForbidden {
		t.Fatalf("remote shutdown status=%d, want 403", response.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.RemoteAddr = "127.0.0.1:43210"
	req.Header.Set("Origin", "https://attacker.example")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("cross-site shutdown without client header status=%d, want 401", response.Code)
	}

	response = request(t, handler, http.MethodPost, "/api/shutdown", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("local shutdown status=%d body=%s", response.Code, response.Body.String())
	}
	select {
	case <-srv.shutdownCh:
	default:
		t.Fatal("shutdown channel was not closed")
	}
	response = request(t, handler, http.MethodPost, "/api/shutdown", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("repeated shutdown status=%d", response.Code)
	}
}

func TestConcurrentConfigSavesKeepDiskAndMemoryConsistent(t *testing.T) {
	srv, path := newTestServer(t)
	handler := srv.Handler()
	const saves = 20
	var wg sync.WaitGroup
	errorsCh := make(chan string, saves)
	for i := 0; i < saves; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			candidate := srv.configSnapshot()
			candidate.QuoteSize = 10 + index%2
			body, _ := json.Marshal(candidate)
			response := request(t, handler, http.MethodPut, "/api/config", body, nil)
			if response.Code != http.StatusOK {
				errorsCh <- response.Body.String()
			}
		}(i)
	}
	wg.Wait()
	close(errorsCh)
	for message := range errorsCh {
		t.Errorf("concurrent save failed: %s", message)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.QuoteSize != srv.configSnapshot().QuoteSize {
		t.Fatalf("disk quote size=%d, memory quote size=%d", loaded.QuoteSize, srv.configSnapshot().QuoteSize)
	}
}

func TestSaveFailureDoesNotChangeMemory(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.configPath = t.TempDir()
	candidate := srv.configSnapshot()
	candidate.ClockFont = "montserrat"
	body, _ := json.Marshal(candidate)
	response := request(t, srv.Handler(), http.MethodPut, "/api/config", body, nil)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("save failure status=%d body=%s", response.Code, response.Body.String())
	}
	if got := srv.configSnapshot().ClockFont; got != "bebas" {
		t.Fatalf("in-memory font changed after failed save: %q", got)
	}
}

func TestRefreshUsesInjectedFunction(t *testing.T) {
	srv, _ := newTestServer(t)
	candidate := srv.configSnapshot()
	candidate.KindleSSH.Enabled = true
	candidate.KindleSSH.Host = "192.0.2.20"
	candidate.KindleSSH.Port = 2222
	candidate.KindleSSH.User = "root"
	candidate.KindleSSH.PrivateKeyPath = "private-key"
	candidate.KindleSSH.KnownHostsPath = "known-hosts"
	var calls atomic.Int32
	srv.SetRefreshFunc(func(_ context.Context, got *config.Config) (string, error) {
		calls.Add(1)
		if got.KindleSSH.Host != candidate.KindleSSH.Host {
			t.Errorf("refresh host=%q", got.KindleSSH.Host)
		}
		return "fake refresh completed", nil
	})
	body, _ := json.Marshal(candidate)
	response := request(t, srv.Handler(), http.MethodPut, "/api/config?refresh=1", body, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("refresh save status=%d body=%s", response.Code, response.Body.String())
	}
	if calls.Load() != 1 {
		t.Fatalf("refresh calls=%d", calls.Load())
	}
	if !strings.Contains(response.Body.String(), "command_completed") {
		t.Fatalf("refresh response=%s", response.Body.String())
	}
}

func TestSettingsTokenAndSecretRedaction(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.cfg.SettingsToken = "test-secret-token"
	candidate := srv.configSnapshot()
	candidate.SettingsToken = ""
	body, _ := json.Marshal(candidate)

	response := request(t, srv.Handler(), http.MethodPut, "/api/config", body, nil)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status=%d", response.Code)
	}
	response = request(t, srv.Handler(), http.MethodPut, "/api/config", body, map[string]string{"X-KDash-Token": "test-secret-token"})
	if response.Code != http.StatusOK {
		t.Fatalf("valid token status=%d body=%s", response.Code, response.Body.String())
	}
	response = request(t, srv.Handler(), http.MethodGet, "/api/config", nil, nil)
	if strings.Contains(response.Body.String(), "test-secret-token") {
		t.Fatal("settings token leaked in GET /api/config")
	}
}

func TestInvalidDraftAndMethodsRejected(t *testing.T) {
	srv, _ := newTestServer(t)
	candidate := srv.configSnapshot()
	candidate.ClockSize = 1000000
	data, _ := json.Marshal(candidate)
	draft := base64.RawURLEncoding.EncodeToString(data)
	response := request(t, srv.Handler(), http.MethodGet, "/preview.png?draft="+draft, nil, nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid draft status=%d", response.Code)
	}
	response = request(t, srv.Handler(), http.MethodPost, "/screen.png", nil, nil)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST screen status=%d", response.Code)
	}
	response = request(t, srv.Handler(), http.MethodPut, "/api/config", []byte(`{"unknown_field":true}`), nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown config field status=%d", response.Code)
	}
}

func TestPowerStatusDefaultsAndUpdates(t *testing.T) {
	srv, _ := newTestServer(t)
	data := srv.dashboardData(time.Now(), srv.weather)
	if !data.HasBattery || data.Battery != 100 {
		t.Fatalf("expected initial HasBattery=true, Battery=100; got HasBattery=%v, Battery=%d", data.HasBattery, data.Battery)
	}

	handler := srv.Handler()
	request(t, handler, http.MethodGet, "/screen.png?batt=42&charging=1", nil, nil)
	data = srv.dashboardData(time.Now(), srv.weather)
	if !data.HasBattery || data.Battery != 42 || !data.IsCharging {
		t.Fatalf("expected updated HasBattery=true, Battery=42, IsCharging=true; got %+v", data)
	}
}
