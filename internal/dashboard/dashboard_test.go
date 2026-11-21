package dashboard

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andycurise/kindle-kdash/internal/config"
	"github.com/andycurise/kindle-kdash/internal/display"
	"github.com/andycurise/kindle-kdash/internal/weather"
)

var sharedTestDashboard *Dashboard

func testDashboard(t *testing.T) *Dashboard {
	t.Helper()
	if sharedTestDashboard == nil {
		d, err := New()
		if err != nil {
			t.Fatal(err)
		}
		sharedTestDashboard = d
	}
	return sharedTestDashboard
}

func fixedNow() time.Time {
	return time.Date(2026, 11, 14, 13, 22, 0, 0, time.FixedZone("CST", 8*3600))
}

func fixedData() DashboardData {
	now := fixedNow()
	return DashboardData{Now: now, Day: DayInfo(now), Weather: weather.Example(now), Battery: 88, HasBattery: true}
}

func fullConfig() *config.Config {
	return &config.Config{
		Location:           config.Location{Name: "西安"},
		BottomRightModule:  "solar",
		ShowCurrentWeather: true,
		ShowForecast:       true,
		ShowLunar:          true,
		ShowQuote:          true,
		ShowFooter:         true,
	}
}

func moduleByID(t *testing.T, layout Layout, id string) ModuleLayout {
	t.Helper()
	for _, module := range layout.Modules {
		if module.ID == id {
			return module
		}
	}
	t.Fatalf("模块 %q 不存在: %+v", id, layout.Modules)
	return ModuleLayout{}
}

func TestComputeLayoutCoversEveryVisibleModule(t *testing.T) {
	data := fixedData()
	layout := ComputeLayout(data, fullConfig(), 1024, 758)
	if layout.Width != 1024 || layout.Height != 758 {
		t.Fatalf("画布尺寸错误: %+v", layout)
	}
	for _, id := range []string{"calendar", "weather", "footer", "clock", "date", "lunar", "quote", "current_weather", "forecast", "bottom_right"} {
		module := moduleByID(t, layout, id)
		if module.Bounds.Rectangle().Empty() {
			t.Errorf("%s 选区为空", id)
		}
		draggable := id == "clock" || id == "date" || id == "quote"
		if module.Draggable != draggable {
			t.Errorf("%s Draggable=%v，期望 %v", id, module.Draggable, draggable)
		}
	}
	clock := moduleByID(t, layout, "clock")
	footer := moduleByID(t, layout, "footer")
	if clock.Bounds.Rectangle().Max.Y > footer.Bounds.Y {
		t.Fatal("时钟选区覆盖 footer")
	}
	bottom := moduleByID(t, layout, "bottom_right")
	weatherPanel := moduleByID(t, layout, "weather")
	if !bottom.Bounds.Rectangle().In(weatherPanel.Bounds.Rectangle()) {
		t.Fatal("右下卡不在天气大区域内")
	}
}

func TestRectJSONAndRectangle(t *testing.T) {
	r := Rect{X: 3, Y: 4, Width: 5, Height: 6}
	if got, want := r.Rectangle(), image.Rect(3, 4, 8, 10); got != want {
		t.Fatalf("Rectangle=%v，期望 %v", got, want)
	}
	encoded, err := json.Marshal(ModuleLayout{ID: "clock", Title: "时钟", Bounds: r, Draggable: true})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"clock","title":"时钟","bounds":{"x":3,"y":4,"width":5,"height":6},"draggable":true}`
	if string(encoded) != want {
		t.Fatalf("JSON=%s，期望 %s", encoded, want)
	}
}

func TestClockWidths(t *testing.T) {
	d := testDashboard(t)
	r, err := newRenderer(1024, 758, d.fonts, d.faces)
	if err != nil {
		t.Fatal(err)
	}
	target := scaled(758, 0.265)
	for _, font := range []string{"bebas", "jetbrains", "montserrat", "playfair", "noto"} {
		key := clockWeightKey(font)
		w := r.textW(key, target, "13:30")
		t.Logf("font=%s key=%s target=%d textW=%d", font, key, target, w)
	}
}

func TestOffsetsAreClampedInsideOwners(t *testing.T) {
	cfg := fullConfig()
	cfg.ClockXOff, cfg.ClockYOff = 100000, 100000
	cfg.DateXOff, cfg.DateYOff = -100000, -100000
	cfg.QuoteXOff, cfg.QuoteYOff = 100000, -100000
	layout := ComputeLayout(fixedData(), cfg, 1024, 758)
	calendar := moduleByID(t, layout, "calendar").Bounds.Rectangle()
	footer := moduleByID(t, layout, "footer").Bounds.Rectangle()
	for _, id := range []string{"clock", "date", "quote"} {
		bounds := moduleByID(t, layout, id).Bounds.Rectangle()
		if !bounds.In(calendar) {
			t.Errorf("%s 偏移后越出 calendar: %v", id, bounds)
		}
		if bounds.Max.Y > footer.Min.Y {
			t.Errorf("%s 偏移后覆盖 footer", id)
		}
	}
	if got := moduleByID(t, layout, "date").Bounds; got.X != calendar.Min.X || got.Y != calendar.Min.Y {
		t.Fatalf("负日期偏移未夹紧: %+v", got)
	}
}

func TestOffsetsMoveModulesHorizontally(t *testing.T) {
	cfg := fullConfig()
	cfg.ClockXOff, cfg.DateXOff, cfg.QuoteXOff = 0, 0, 0
	base := ComputeLayout(fixedData(), cfg, 1024, 758)
	baseClockX := moduleByID(t, base, "clock").Bounds.X
	baseDateX := moduleByID(t, base, "date").Bounds.X
	baseQuoteX := moduleByID(t, base, "quote").Bounds.X

	cfg.ClockXOff = 50
	cfg.DateXOff = 40
	cfg.QuoteXOff = 30
	moved := ComputeLayout(fixedData(), cfg, 1024, 758)
	clockX := moduleByID(t, moved, "clock").Bounds.X
	dateX := moduleByID(t, moved, "date").Bounds.X
	quoteX := moduleByID(t, moved, "quote").Bounds.X

	if clockX != baseClockX+50 {
		t.Fatalf("ClockXOff=50 未生效: base=%d, moved=%d", baseClockX, clockX)
	}
	if dateX != baseDateX+40 {
		t.Fatalf("DateXOff=40 未生效: base=%d, moved=%d", baseDateX, dateX)
	}
	if quoteX != baseQuoteX+30 {
		t.Fatalf("QuoteXOff=30 未生效: base=%d, moved=%d", baseQuoteX, quoteX)
	}
}

func TestVisibilityControlsReflowAndOmitsSelections(t *testing.T) {
	cfg := &config.Config{}
	layout := ComputeLayout(fixedData(), cfg, 1024, 758)
	for _, hidden := range []string{"footer", "lunar", "quote", "current_weather", "forecast"} {
		for _, module := range layout.Modules {
			if module.ID == hidden {
				t.Errorf("隐藏模块仍输出交互项: %s", hidden)
			}
		}
	}
	bottom := moduleByID(t, layout, "bottom_right")
	if bottom.Bounds.Y != scaled(758, 0.058) {
		t.Fatalf("隐藏天气后右下卡未上移: %+v", bottom.Bounds)
	}
	if bottom.Bounds.Rectangle().Max.Y != 758-scaled(758, 0.012) {
		t.Fatalf("隐藏 footer 后未利用底部空间: %+v", bottom.Bounds)
	}
	if got := len(layout.Modules); got != 5 {
		t.Fatalf("可见模块数=%d，期望 5", got)
	}
}

func TestDashboardLayoutMatchesComputeLayout(t *testing.T) {
	d := testDashboard(t)
	data := fixedData()
	cfg := fullConfig()
	got, want := d.Layout(data, cfg, 1024, 758), ComputeLayout(data, cfg, 1024, 758)
	if len(got.Modules) != len(want.Modules) {
		t.Fatalf("模块数量不一致: %d != %d", len(got.Modules), len(want.Modules))
	}
	for i := range got.Modules {
		if got.Modules[i] != want.Modules[i] {
			t.Errorf("模块 %d 矩形不一致: %+v != %+v", i, got.Modules[i], want.Modules[i])
		}
	}
}

func TestRenderNilWeatherAndRotate(t *testing.T) {
	d := testDashboard(t)
	data := fixedData()
	data.Weather = nil
	img := d.Render(data, fullConfig(), 1024, 758)
	if img.Bounds() != image.Rect(0, 0, 1024, 758) {
		t.Fatalf("渲染尺寸=%v", img.Bounds())
	}
	if countDarkPixels(img, moduleByID(t, ComputeLayout(data, fullConfig(), 1024, 758), "current_weather").Bounds.Rectangle()) == 0 {
		t.Fatal("空天气当前卡没有绘制空态")
	}
	for _, direction := range []int{90, 270} {
		rotated := display.RotateForDevice(img, direction)
		if rotated.Bounds() != image.Rect(0, 0, 758, 1024) {
			t.Errorf("%d° 显示尺寸=%v", direction, rotated.Bounds())
		}
		gray := display.GrayImage(rotated)
		if gray.Bounds() != rotated.Bounds() {
			t.Errorf("%d° 灰度尺寸=%v", direction, gray.Bounds())
		}
	}
}

func TestExtremeOffsetsSizesAndFitDoNotPanic(t *testing.T) {
	d := testDashboard(t)
	cfg := fullConfig()
	cfg.ClockXOff, cfg.ClockYOff = -1<<30, 1<<30
	cfg.DateXOff, cfg.DateYOff = 1<<30, -1<<30
	cfg.QuoteXOff, cfg.QuoteYOff = -1<<30, -1<<30
	cfg.ClockSize, cfg.DateSize, cfg.QuoteSize = 1<<20, -100, 0
	for _, size := range []image.Point{{X: -1, Y: -1}, {X: 0, Y: 0}, {X: 1, Y: 1}, {X: 17, Y: 11}, {X: 100, Y: 80}, {X: 1024, Y: 758}} {
		img := d.Render(fixedData(), cfg, size.X, size.Y)
		if img.Bounds().Dx() < 1 || img.Bounds().Dy() < 1 {
			t.Errorf("尺寸 %v 返回非法画布 %v", size, img.Bounds())
		}
	}
	r, err := newRenderer(10, 10, d.fonts, d.faces)
	if err != nil {
		t.Fatal(err)
	}
	for _, maxW := range []int{-1, 0, 1} {
		if got := r.fit("bold", 0, "越界文本", maxW); got != 1 {
			t.Errorf("fit(size=0,maxW=%d)=%d，期望 1", maxW, got)
		}
	}
}

func TestOffsetsCannotPaintFooter(t *testing.T) {
	d := testDashboard(t)
	data := fixedData()
	baseCfg := fullConfig()
	base := d.Render(data, baseCfg, 1024, 758)
	footer := moduleByID(t, ComputeLayout(data, baseCfg, 1024, 758), "footer").Bounds.Rectangle()
	cfg := fullConfig()
	cfg.ClockYOff, cfg.DateYOff, cfg.QuoteYOff = 100000, 100000, 100000
	changed := d.Render(data, cfg, 1024, 758)
	for y := footer.Min.Y; y < footer.Max.Y; y++ {
		for x := footer.Min.X; x < footer.Max.X; x++ {
			if base.RGBAAt(x, y) != changed.RGBAAt(x, y) {
				t.Fatalf("偏移绘制污染 footer 像素 (%d,%d)", x, y)
			}
		}
	}
}

func TestAirQualityValidityAndZeroUV(t *testing.T) {
	now := fixedNow()
	w := weather.Example(now)
	w.Current.AirQualityValid = true
	w.Current.UVIndex = 0
	w.Daily.UVIndexMax = []float64{9}
	w.Current.AQI, w.Current.PM25, w.Current.PM10 = 42, 12, 23
	aqi, _, pm25, pm10, uv, _ := airQualityValues(w)
	if aqi != "42" || pm25 != "12" || pm10 != "23" || uv != "0.0" {
		t.Fatalf("有效空气质量值错误: %q %q %q %q", aqi, pm25, pm10, uv)
	}
	w.Current.AirQualityValid = false
	aqi, _, pm25, pm10, uv, _ = airQualityValues(w)
	if aqi != "--" || pm25 != "--" || pm10 != "--" || uv != "0.0" {
		t.Fatalf("无效空气质量应显示 --: %q %q %q %q", aqi, pm25, pm10, uv)
	}
}

func countDarkPixels(img *image.RGBA, bounds image.Rectangle) int {
	bounds = bounds.Intersect(img.Bounds())
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.R < 245 || c.G < 245 || c.B < 245 {
				count++
			}
		}
	}
	return count
}

func TestFooterRenderingWithChargingAndWiFi(t *testing.T) {
	d := testDashboard(t)
	cfg := fullConfig()
	data := fixedData()
	data.HasBattery = true
	data.Battery = 85
	data.IsCharging = true

	imgCharging := d.Render(data, cfg, 1024, 758)
	if imgCharging == nil {
		t.Fatal("充电状态渲染结果为空")
	}

	// 导出到测试临时目录，避免跨平台固定路径和并发冲突。
	var buf bytes.Buffer
	if err := png.Encode(&buf, imgCharging); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(t.TempDir(), "test_charging_land.png"), buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	data.IsCharging = false
	imgNormal := d.Render(data, cfg, 1024, 758)
	if imgNormal == nil {
		t.Fatal("正常状态渲染结果为空")
	}
}
