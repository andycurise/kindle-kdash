package weather

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andycurise/kindle-kdash/internal/config"
)

var testNow = time.Date(2026, 11, 14, 13, 22, 0, 0, time.FixedZone("CST", 8*3600))

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := config.Default()
	cfg.BaseDir = t.TempDir()
	cfg.WeatherCachePath = filepath.Join("nested", "cache_weather.json")
	cfg.Location = config.Location{Name: "西安", CityCode: "101110101", Lat: 34.3416, Lon: 108.9398}
	return cfg
}

func replaceAirQualityURLs(t *testing.T, chinaURL, openMeteoURL string) {
	t.Helper()
	oldChina, oldOpenMeteo := chinaAirQualityBaseURL, openMeteoAirQualityURL
	chinaAirQualityBaseURL = chinaURL
	openMeteoAirQualityURL = openMeteoURL
	t.Cleanup(func() {
		chinaAirQualityBaseURL = oldChina
		openMeteoAirQualityURL = oldOpenMeteo
	})
}

func writeCachedWeather(t *testing.T, cfg *config.Config, w *Weather) string {
	t.Helper()
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	path := cfg.ResolvePath(cfg.WeatherCachePath)
	if err := config.WriteFileAtomic(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDaysStartsAtTodayAndValidatesSeries(t *testing.T) {
	w := Example(testNow)
	if err := w.validate(); err != nil {
		t.Fatalf("示例天气应有效: %v", err)
	}
	days := w.Days(testNow)
	if len(days) != 3 || days[0].Label != "今天" || days[1].Label != "明天" || days[2].Label != "周一" {
		t.Fatalf("三天标签不符合预期: %+v", days)
	}
	if days[0].Sunrise != "07:15" || days[0].Sunset != "17:40" {
		t.Fatalf("日出日落解析错误: %+v", days[0])
	}

	withYesterday := Example(testNow.AddDate(0, 0, -1))
	days = withYesterday.Days(testNow)
	if len(days) != 2 || days[0].Label != "今天" || days[1].Label != "明天" {
		t.Fatalf("应从当天而非缓存首项开始: %+v", days)
	}
	if days[0].Code != withYesterday.Daily.WeatherCode[1] {
		t.Fatal("昨天缓存被错误标成今天")
	}
	if got := withYesterday.Days(testNow.AddDate(0, 0, 2)); got != nil {
		t.Fatalf("没有当天数据应返回 nil: %+v", got)
	}

	tests := []struct {
		name   string
		mutate func(*Weather)
	}{
		{"weather code", func(w *Weather) { w.Daily.WeatherCode = w.Daily.WeatherCode[:1] }},
		{"max temperature", func(w *Weather) { w.Daily.TMax = w.Daily.TMax[:1] }},
		{"min temperature", func(w *Weather) { w.Daily.TMin = w.Daily.TMin[:1] }},
		{"precipitation", func(w *Weather) { w.Daily.PrecipProb = w.Daily.PrecipProb[:1] }},
		{"sunrise", func(w *Weather) { w.Daily.Sunrise = w.Daily.Sunrise[:1] }},
		{"sunset", func(w *Weather) { w.Daily.Sunset = w.Daily.Sunset[:1] }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalid := Example(testNow)
			tt.mutate(invalid)
			if err := invalid.validate(); err == nil {
				t.Fatal("序列长度不足应返回错误")
			}
			if got := invalid.Days(testNow); got != nil {
				t.Fatalf("无效天气不应生成预报: %+v", got)
			}
		})
	}
}

func TestGetChinaAirQualityAndRelativeCachePath(t *testing.T) {
	payload, err := json.Marshal(Example(testNow))
	if err != nil {
		t.Fatal(err)
	}
	openMeteoCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/forecast":
			if got := r.URL.Query().Get("forecast_days"); got != "3" {
				t.Errorf("forecast_days=%q，期望 3", got)
			}
			if got := r.URL.Query().Get("timezone"); got != "Asia/Shanghai" {
				t.Errorf("timezone=%q，期望 Asia/Shanghai", got)
			}
			if got := r.URL.Query().Get("current"); !strings.Contains(got, "uv_index") {
				t.Errorf("主天气未请求 UV: %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(payload)
		case "/china/101110101":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":200,"data":{"pm25":12.5,"pm10":30,"quality":"优","forecast":[{"aqi":42,"notice":"适宜外出","fx":"北风","fl":"2级"}]}}`))
		case "/open-air":
			openMeteoCalled = true
			http.Error(w, "不应调用备用源", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	replaceAirQualityURLs(t, server.URL+"/china", server.URL+"/open-air")

	cfg := newTestConfig(t)
	cfg.WeatherURL = server.URL + "/forecast"
	fresh := Get(cfg, testNow)
	if fresh == nil || fresh.Cached || fresh.Offline {
		t.Fatalf("在线天气状态错误: %+v", fresh)
	}
	if !fresh.Current.AirQualityValid || fresh.Current.AirError != "" || fresh.Current.AQI != 42 {
		t.Fatalf("国内空气质量状态错误: %+v", fresh.Current)
	}
	if fresh.Current.PM25 != 12.5 || fresh.Current.PM10 != 30 || fresh.Current.WindDir != "北风" {
		t.Fatalf("国内空气质量字段错误: %+v", fresh.Current)
	}
	if openMeteoCalled {
		t.Fatal("国内空气质量成功时不应请求备用源")
	}
	path := cfg.ResolvePath(cfg.WeatherCachePath)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("在线天气未写入相对缓存路径 %s: %v", path, err)
	}
	cached := LoadCached(cfg, testNow.Add(time.Minute))
	if cached == nil || !cached.Cached || cached.Offline || !cached.Current.AirQualityValid {
		t.Fatalf("短期缓存状态错误: %+v", cached)
	}
}

func TestGetOpenMeteoAirQualityAcceptsZeroAndUVZero(t *testing.T) {
	payload, err := json.Marshal(Example(testNow))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/forecast":
			_, _ = w.Write(payload)
		case "/china/101110101":
			http.Error(w, "国内源不可用", http.StatusBadGateway)
		case "/open-air":
			if got := r.URL.Query().Get("current"); got != "us_aqi,pm10,pm2_5" {
				t.Errorf("Open-Meteo AQ current=%q", got)
			}
			_, _ = w.Write([]byte(`{"current":{"us_aqi":0,"pm10":0,"pm2_5":0}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	replaceAirQualityURLs(t, server.URL+"/china", server.URL+"/open-air")

	cfg := newTestConfig(t)
	cfg.WeatherURL = server.URL + "/forecast"
	fresh := Get(cfg, testNow)
	if fresh == nil {
		t.Fatal("主天气应获取成功")
	}
	if !fresh.Current.AirQualityValid || fresh.Current.AirError != "" {
		t.Fatalf("合法的全零空气质量应有效: %+v", fresh.Current)
	}
	if fresh.Current.AQI != 0 || fresh.Current.USAQI != 0 || fresh.Current.Quality != "优" {
		t.Fatalf("全零空气质量计算错误: %+v", fresh.Current)
	}
	if fresh.Current.UVIndex != 0 {
		t.Fatalf("UV=0 是合法值，不应被改写: %v", fresh.Current.UVIndex)
	}
}

func TestAirQualityFailureHasErrorAndInvalidState(t *testing.T) {
	payload, err := json.Marshal(Example(testNow))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/forecast":
			_, _ = w.Write(payload)
		case "/china/101110101":
			_, _ = w.Write([]byte(`{"status":200,"data":{"quality":"优","forecast":[]}}`))
		case "/open-air":
			http.Error(w, "备用源不可用", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	replaceAirQualityURLs(t, server.URL+"/china", server.URL+"/open-air")

	cfg := newTestConfig(t)
	cfg.WeatherURL = server.URL + "/forecast"
	fresh := Get(cfg, testNow)
	if fresh == nil {
		t.Fatal("空气质量失败不应丢弃主天气")
	}
	if fresh.Current.AirQualityValid || fresh.Current.AirError == "" {
		t.Fatalf("空气质量失败必须有 error/Validity: %+v", fresh.Current)
	}
	if fresh.Current.Quality != "未知" {
		t.Fatalf("失败时不应伪装成全零成功: %+v", fresh.Current)
	}
	cached := LoadCached(cfg, testNow)
	if cached == nil || cached.Current.AirQualityValid || cached.Current.AirError == "" {
		t.Fatalf("空气质量失败状态应写入并读回缓存: %+v", cached)
	}
}

func TestCacheFallbackAndCrossMidnightOffline(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.OfflineGraceMin = 180
	beforeMidnight := time.Date(2026, 11, 14, 23, 59, 0, 0, testNow.Location())
	afterMidnight := beforeMidnight.Add(2 * time.Minute)
	stale := Example(beforeMidnight)
	stale.Daily.Time = stale.Daily.Time[:1]
	stale.Daily.WeatherCode = stale.Daily.WeatherCode[:1]
	stale.Daily.TMax = stale.Daily.TMax[:1]
	stale.Daily.TMin = stale.Daily.TMin[:1]
	stale.Daily.PrecipProb = stale.Daily.PrecipProb[:1]
	stale.Daily.Sunrise = stale.Daily.Sunrise[:1]
	stale.Daily.Sunset = stale.Daily.Sunset[:1]
	writeCachedWeather(t, cfg, stale)

	cached := LoadCached(cfg, beforeMidnight.Add(-time.Minute))
	if cached == nil || !cached.Cached || cached.Offline {
		t.Fatalf("当天宽限期内的缓存不应离线: %+v", cached)
	}
	cached = LoadCached(cfg, afterMidnight)
	if cached == nil || !cached.Cached || !cached.Offline {
		t.Fatalf("跨午夜后缺少当天预报应离线: %+v", cached)
	}
	if got := cached.Days(afterMidnight); got != nil {
		t.Fatalf("昨天缓存不得标成今天: %+v", got)
	}

	// 主天气失败时仍返回该缓存；请求地址和 AQ 地址均为本地测试端点。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "offline", http.StatusServiceUnavailable)
	}))
	server.Close()
	replaceAirQualityURLs(t, server.URL, server.URL)
	cfg.WeatherURL = server.URL
	fallback := Get(cfg, afterMidnight)
	if fallback == nil || !fallback.Cached || !fallback.Offline {
		t.Fatalf("网络失败后的跨午夜缓存状态错误: %+v", fallback)
	}
}

func TestScheduleNext(t *testing.T) {
	now := time.Now()
	fresh := &Weather{Fetched: now, Cached: false}
	cached := &Weather{Fetched: now, Cached: true}

	w, next := ScheduleNext(nil, fresh, 1800)
	if w != fresh || next.Sub(now) < 1790*time.Second || next.Sub(now) > 1810*time.Second {
		t.Fatalf("在线成功调度错误: w=%+v, next=%v", w, next)
	}
	w, next = ScheduleNext(nil, cached, 1800)
	if w != cached || next.Sub(now) < 55*time.Second || next.Sub(now) > 65*time.Second {
		t.Fatalf("缓存调度错误: w=%+v, next=%v", w, next)
	}
	w, next = ScheduleNext(cached, nil, 1800)
	if w != cached || next.Sub(now) < 55*time.Second || next.Sub(now) > 65*time.Second {
		t.Fatalf("保留现有缓存重试调度错误: w=%+v, next=%v", w, next)
	}
	w, next = ScheduleNext(nil, nil, 1800)
	if w != nil || next.Sub(now) < 55*time.Second || next.Sub(now) > 65*time.Second {
		t.Fatalf("无缓存重试调度错误: w=%+v, next=%v", w, next)
	}
}

func TestExportedIconTextAndTimeHelpers(t *testing.T) {
	icons := []IconKind{IconSun, IconPartly, IconCloud, IconFog, IconDrizzle, IconRain, IconSleet, IconSnow, IconThunder}
	for i, icon := range icons {
		if int(icon) != i {
			t.Fatalf("IconKind 枚举顺序改变: %d != %d", int(icon), i)
		}
	}
	codes := []struct {
		code int
		kind IconKind
		text string
	}{
		{0, IconSun, "晴"},
		{2, IconPartly, "多云"},
		{3, IconCloud, "阴"},
		{45, IconFog, "雾"},
		{51, IconDrizzle, "毛毛雨"},
		{61, IconRain, "小雨"},
		{66, IconSleet, "冻雨"},
		{71, IconSnow, "小雪"},
		{95, IconThunder, "雷阵雨"},
	}
	for _, tc := range codes {
		if got := Icon(tc.code); got != tc.kind {
			t.Errorf("Icon(%d)=%v，期望 %v", tc.code, got, tc.kind)
		}
		if got := Text(tc.code); got != tc.text {
			t.Errorf("Text(%d)=%q，期望 %q", tc.code, got, tc.text)
		}
	}
	if got := WeekCN(time.Saturday); got != "周六" {
		t.Fatalf("WeekCN=%q", got)
	}
	if got := HHMM("2026-11-14T07:15"); got != "07:15" {
		t.Fatalf("HHMM=%q", got)
	}
}
