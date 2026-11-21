package dashboard

import (
	"image/png"
	"testing"
)

func TestQuoteAndAlmanacRenderStableInk(t *testing.T) {
	d := testDashboard(t)
	data := fixedData()
	layout := ComputeLayout(data, fullConfig(), 1024, 758)
	quoteBounds := moduleByID(t, layout, "quote").Bounds.Rectangle()
	for _, mode := range []struct {
		quoteType string
		font      string
		size      int
	}{{"mao", "bold", 0}, {"famous", "bold", 34}, {"inspirational", "med", 0}, {"poetry", "kai", 0}, {"poetry", "wei", 0}, {"poetry", "xing", 0}, {"inspirational", "lively", 0}, {"almanac", "bold", 0}} {
		cfg := fullConfig()
		cfg.QuoteType, cfg.QuoteFont, cfg.QuoteSize = mode.quoteType, mode.font, mode.size
		img := d.Render(data, cfg, 1024, 758)
		if got := countDarkPixels(img, quoteBounds); got < 1000 {
			t.Errorf("%s/%s 金句卡有效像素过少: %d", mode.quoteType, mode.font, got)
		}
		// Encoding into a caller-owned temp directory catches invalid images
		// without leaving generated previews in the repository.
		if err := png.Encode(tempFile(t, "quote.png"), img); err != nil {
			t.Fatalf("PNG 编码失败: %v", err)
		}
	}
}

func TestSplitQuoteLinesGeometry(t *testing.T) {
	lines := splitQuoteLines("雄关漫道真如铁，而今迈步从头越。", 7)
	if len(lines) != 2 || lines[0] != "雄关漫道真如铁，" || lines[1] != "而今迈步从头越。" {
		t.Fatalf("金句分行不稳定: %#v", lines)
	}
	if got := splitQuoteLines("短句", 17); len(got) != 1 || got[0] != "短句" {
		t.Fatalf("短句不应拆分: %#v", got)
	}
}
