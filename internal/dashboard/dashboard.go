// Package dashboard renders the calendar/weather image served by the host.
// Kindle only downloads the resulting image; no device runtime belongs here.
package dashboard

import (
	"image"
	"image/draw"
	"time"

	"github.com/andycurise/kindle-kdash/internal/config"
	"github.com/andycurise/kindle-kdash/internal/weather"
	"golang.org/x/image/font"
)

// DashboardData is the immutable data snapshot for one frame.
type DashboardData struct {
	Now        time.Time
	Day        DayInfoData
	Weather    *weather.Weather
	Battery    int
	IsCharging bool
	HasBattery bool
}

// Dashboard reuses parsed fonts and faces across frames.
type Dashboard struct {
	fonts *fontSet
	faces map[string]font.Face
}

// New creates a dashboard using the fonts embedded in kdash/assets.
func New() (*Dashboard, error) {
	fonts, err := loadFonts()
	if err != nil {
		return nil, err
	}
	return &Dashboard{fonts: fonts, faces: map[string]font.Face{}}, nil
}

// Layout returns the exact module rectangles used by Render.
func (d *Dashboard) Layout(data DashboardData, cfg *config.Config, w, h int) Layout {
	return ComputeLayout(data, cfg, w, h)
}

// Render draws every visible module through its ComputeLayout rectangle.
func (d *Dashboard) Render(data DashboardData, cfg *config.Config, w, h int) *image.RGBA {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	r, err := newRenderer(w, h, d.fonts, d.faces)
	if err != nil {
		// New validates fonts before a Dashboard can be returned. Keep the
		// no-error Render contract explicit if that invariant is ever broken.
		panic(err)
	}
	layout := d.Layout(data, cfg, w, h)
	byID := layout.moduleMap()

	calendar, hasCalendar := byID["calendar"]
	weatherPanel, hasWeather := byID["weather"]
	footer, hasFooter := byID["footer"]
	if hasCalendar && hasWeather {
		sepX := (calendar.Bounds.Rectangle().Max.X + weatherPanel.Bounds.Rectangle().Min.X) / 2
		bottom := weatherPanel.Bounds.Rectangle().Max.Y
		if hasFooter {
			bottom = footer.Bounds.Y - scaled(h, 0.02)
		}
		if bottom > scaled(h, 0.05) {
			r.rect(sepX, scaled(h, 0.05), sepX+1, bottom, inkLine)
		}
	}

	for _, module := range layout.Modules {
		bounds := module.Bounds.Rectangle().Intersect(r.img.Bounds())
		if bounds.Empty() {
			continue
		}
		// A temporary canvas gives text and vector drawing the same hard clip.
		mr := r.clip(bounds)
		switch module.ID {
		case "calendar":
			renderCalendarHeading(mr, bounds, data)
		case "clock":
			renderClock(mr, bounds, data, cfg)
		case "date":
			renderDate(mr, bounds, data, cfg)
		case "lunar":
			renderLunar(mr, bounds, data)
		case "quote":
			renderQuote(mr, bounds, data, cfg)
		case "current_weather":
			renderCurrentWeather(mr, bounds, data)
		case "forecast":
			renderForecast(mr, bounds, data)
		case "bottom_right":
			if cfg != nil && cfg.BottomRightModule == "air" {
				renderAirQuality(mr, bounds, data)
			} else {
				renderSolar(mr, bounds, data)
			}
		case "footer":
			renderFooter(mr, bounds, data, cfg)
		}
		draw.Draw(r.img, bounds, mr.img, bounds.Min, draw.Over)
	}
	return r.img
}
