package dashboard

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSolarDayAndNightHaveStableGeometry(t *testing.T) {
	d := testDashboard(t)
	for _, hour := range []int{8, 12, 17, 20, 22, 23} {
		now := time.Date(2026, 11, 14, hour, 30, 0, 0, time.FixedZone("CST", 8*3600))
		data := DashboardData{Now: now, Day: DayInfo(now), Weather: fixedData().Weather}
		cfg := fullConfig()
		layout := ComputeLayout(data, cfg, 1024, 758)
		bounds := moduleByID(t, layout, "bottom_right").Bounds.Rectangle()
		img := d.Render(data, cfg, 1024, 758)
		if got := countDarkPixels(img, bounds); got < 2000 {
			t.Errorf("%02d:30 太阳/月相卡有效像素过少: %d", hour, got)
		}
		// The border and horizon have deterministic geometry at every phase.
		if img.RGBAAt((bounds.Min.X+bounds.Max.X)/2, bounds.Min.Y).R > 230 {
			t.Errorf("%02d:30 卡片上边框缺失", hour)
		}
	}
}

func TestAirQualityModeRendersWithinBottomCard(t *testing.T) {
	d := testDashboard(t)
	cfg := fullConfig()
	cfg.BottomRightModule = "air"
	data := fixedData()
	data.Weather.Current.AirQualityValid = false
	data.Weather.Current.UVIndex = 0
	layout := ComputeLayout(data, cfg, 1024, 758)
	bounds := moduleByID(t, layout, "bottom_right").Bounds.Rectangle()
	img := d.Render(data, cfg, 1024, 758)
	if got := countDarkPixels(img, bounds); got < 2000 {
		t.Fatalf("空气质量空态有效像素过少: %d", got)
	}
}

func tempFile(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}
