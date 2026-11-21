package dashboard

import (
	"image"

	"github.com/andycurise/kindle-kdash/internal/config"
)

// Rect is the serializable rectangle shared by rendering and interactive
// selection. Width and Height are dimensions, not max coordinates.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Rectangle converts Rect to Go's half-open image.Rectangle convention.
func (r Rect) Rectangle() image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.Width, r.Y+r.Height)
}

// ModuleLayout describes one visible module and its interaction bounds.
type ModuleLayout struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Bounds    Rect   `json:"bounds"`
	Draggable bool   `json:"draggable"`
}

// Layout is the single geometry source for rendering and the web selection UI.
type Layout struct {
	Width   int            `json:"width"`
	Height  int            `json:"height"`
	Modules []ModuleLayout `json:"modules"`
}

func (l Layout) moduleMap() map[string]ModuleLayout {
	m := make(map[string]ModuleLayout, len(l.Modules))
	for _, module := range l.Modules {
		m[module.ID] = module
	}
	return m
}

func scaled(size int, ratio float64) int {
	v := int(float64(size) * ratio)
	if v < 1 && size > 0 {
		return 1
	}
	return v
}

func rect(x, y, w, h int) Rect {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return Rect{X: x, Y: y, Width: w, Height: h}
}

func fromRectangle(r image.Rectangle) Rect {
	return rect(r.Min.X, r.Min.Y, r.Dx(), r.Dy())
}

// clampInside moves an offset module back into its owning top-level region.
// It never lets user offsets cover another module or the footer.
func clampInside(r Rect, owner Rect) Rect {
	if owner.Width == 0 || owner.Height == 0 {
		return rect(owner.X, owner.Y, 0, 0)
	}
	if r.Height > owner.Height {
		r.Height = owner.Height
	}
	if r.X < owner.X {
		r.X = owner.X
	}
	if r.Y < owner.Y {
		r.Y = owner.Y
	}

	minW := maxInt(1, owner.Width/5)
	if r.X > owner.X+owner.Width-minW {
		r.X = owner.X + owner.Width - minW
	}
	if r.X+r.Width > owner.X+owner.Width {
		r.Width = owner.X + owner.Width - r.X
	}

	if r.Y+r.Height > owner.Y+owner.Height {
		r.Y = owner.Y + owner.Height - r.Height
	}
	return r
}

func translated(r Rect, dx, dy int, owner Rect) Rect {
	r.X += dx
	r.Y += dy
	return clampInside(r, owner)
}

// ComputeLayout computes every visible top-level and child module rectangle.
// Render consumes this same value; callers must not duplicate selection zones.
func ComputeLayout(data DashboardData, cfg *config.Config, w, h int) Layout {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	layout := Layout{Width: w, Height: h}
	add := func(id, title string, bounds Rect, draggable bool) {
		if bounds.Width <= 0 || bounds.Height <= 0 {
			return
		}
		layout.Modules = append(layout.Modules, ModuleLayout{ID: id, Title: title, Bounds: bounds, Draggable: draggable})
	}

	margin := scaled(h, 0.045)
	rightW := int(0.365 * float64(w))
	panelX1 := w - margin
	panelX0 := panelX1 - rightW
	if panelX0 < margin {
		panelX0 = margin
	}
	footerH := scaled(h, 0.060)
	footerY := h - footerH
	showFooter := cfg == nil || cfg.ShowFooter
	contentBottom := h
	if showFooter {
		contentBottom = footerY
	}
	calendar := rect(margin, 0, maxInt(0, panelX0-scaled(h, 0.035)-margin), contentBottom)
	weatherPanel := rect(panelX0, 0, maxInt(0, panelX1-panelX0), contentBottom)
	footer := rect(margin, footerY, maxInt(0, w-2*margin), footerH)

	add("calendar", "日历", calendar, false)
	add("weather", "天气", weatherPanel, false)
	if showFooter {
		add("footer", "状态栏", footer, false)
	}

	clockBase := rect(calendar.X, scaled(h, 0.115), calendar.Width, scaled(h, 0.250))
	dateBase := rect(calendar.X, scaled(h, 0.385), calendar.Width, scaled(h, 0.100))
	var clockOffX, clockOffY, dateOffX, dateOffY int
	if cfg != nil {
		clockOffX, clockOffY = cfg.ClockXOff, cfg.ClockYOff
		dateOffX, dateOffY = cfg.DateXOff, cfg.DateYOff
	}
	add("clock", "时钟", translated(clockBase, clockOffX, clockOffY, calendar), true)
	add("date", "公历与星期", translated(dateBase, dateOffX, dateOffY, calendar), true)

	nextY := scaled(h, 0.495)
	if cfg == nil || cfg.ShowLunar {
		lunar := rect(calendar.X, nextY, calendar.Width, scaled(h, 0.160))
		add("lunar", "农历与节气", lunar, false)
		nextY = lunar.Y + lunar.Height + scaled(h, 0.024)
	}
	if cfg == nil || cfg.ShowQuote {
		quoteBase := rect(calendar.X, nextY, calendar.Width, scaled(h, 0.185))
		var dx, dy int
		if cfg != nil {
			dx, dy = cfg.QuoteXOff, cfg.QuoteYOff
		}
		add("quote", "每日金句", translated(quoteBase, dx, dy, calendar), true)
	}

	nextY = scaled(h, 0.050)
	if cfg == nil || cfg.ShowCurrentWeather {
		current := rect(weatherPanel.X, nextY, weatherPanel.Width, scaled(h, 0.198))
		add("current_weather", "当前天气", current, false)
		nextY = current.Y + current.Height + scaled(h, 0.014)
	}
	if cfg == nil || cfg.ShowForecast {
		forecastGap := scaled(h, 0.016)
		cardH := scaled(h, 0.098)
		forecast := rect(weatherPanel.X, nextY, weatherPanel.Width, 3*cardH+2*forecastGap)
		add("forecast", "三日预报", forecast, false)
		nextY = forecast.Y + forecast.Height + scaled(h, 0.014)
	}
	bottomY := nextY + scaled(h, 0.008)
	bottom := rect(weatherPanel.X, bottomY, weatherPanel.Width, weatherPanel.Y+weatherPanel.Height-scaled(h, 0.012)-bottomY)
	add("bottom_right", "右下信息卡", bottom, false)
	return layout
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
