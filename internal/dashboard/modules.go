package dashboard

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/andycurise/kindle-kdash/internal/config"
	"github.com/andycurise/kindle-kdash/internal/weather"
)

func fontSize(height int, ratio float64) int {
	return maxInt(1, int(float64(height)*ratio))
}

func renderCalendarHeading(r *renderer, bounds image.Rectangle, data DashboardData) {
	parts := strings.Split(data.Day.LunarCN, " ")
	yearPart := ""
	if len(parts) > 0 {
		yearPart = strings.TrimPrefix(parts[0], "农历")
	}
	size := fontSize(r.h, 0.036)
	text := r.fit("med", size, fmt.Sprintf("%d年 · %s", data.Now.Year(), yearPart), bounds.Dx())
	r.text(bounds.Min.X, scaled(r.h, 0.068), "med", text, fmt.Sprintf("%d年 · %s", data.Now.Year(), yearPart), inkMuted)
}

func renderClock(r *renderer, bounds image.Rectangle, data DashboardData, cfg *config.Config) {
	key := "clock-bebas"
	target := scaled(r.h, 0.265)
	if cfg != nil {
		if cfg.ClockFont != "" {
			key = clockWeightKey(cfg.ClockFont)
		}
		if cfg.ClockSize > 0 {
			target = cfg.ClockSize
		}
	}
	hour, minute := data.Now.Format("15"), data.Now.Format("04")
	// Fit the actual time, including its colon, rather than a fixed 88:88.
	size := r.fit(key, target, hour+":"+minute, maxInt(1, bounds.Dx()-scaled(r.h, 0.02)))
	baseline := bounds.Min.Y + scaled(r.h, 0.230)
	x := bounds.Min.X
	hourW := r.textW(key, size, hour)
	colonW := maxInt(1, r.textW(key, size, ":"))
	r.text(x, baseline, key, size, hour, ink)
	colonX := float64(x + hourW + colonW/2)
	dotR := math.Max(1, float64(size)*0.043)
	r.fillCircle(colonX, float64(baseline)-float64(size)*0.52, dotR, ink)
	r.fillCircle(colonX, float64(baseline)-float64(size)*0.20, dotR, ink)
	r.text(x+hourW+colonW, baseline, key, size, minute, ink)
}

func renderDate(r *renderer, bounds image.Rectangle, data DashboardData, cfg *config.Config) {
	key := "bold"
	target := scaled(r.h, 0.078)
	if cfg != nil {
		if cfg.DateFont != "" {
			key = dateWeightKey(cfg.DateFont)
		}
		if cfg.DateSize > 0 {
			target = cfg.DateSize
		}
	}
	monthDay := fmt.Sprintf("%d月%d日", int(data.Now.Month()), data.Now.Day())
	week := data.Day.WeekCN
	if week == "" {
		week = weekFull(data.Now.Weekday())
	}
	measure := func(size int) (dateW, weekSize, pad, gap, pillW int) {
		if isClockWeight(key) {
			dateW = r.textMixedDateW("bold", key, size, monthDay)
		} else {
			dateW = r.textW(key, size, monthDay)
		}
		weekSize = maxInt(scaled(r.h, 0.028), int(float64(size)*0.54))
		pad = scaled(r.h, 0.018)
		gap = scaled(r.h, 0.022)
		pillW = r.textW("bold", weekSize, week) + 2*pad
		return
	}
	dateW, weekSize, pad, gap, pillW := measure(target)
	if total := dateW + gap + pillW; total > bounds.Dx() && total > 0 {
		target = maxInt(1, int(float64(target)*float64(bounds.Dx())/float64(total)))
		dateW, weekSize, pad, gap, pillW = measure(target)
	}
	baseline := bounds.Min.Y + int(float64(bounds.Dy())*0.68)
	if isClockWeight(key) {
		r.textMixedDate(bounds.Min.X, baseline, "bold", key, target, monthDay, ink)
	} else {
		r.text(bounds.Min.X, baseline, key, target, monthDay, ink)
	}
	pillX := bounds.Min.X + dateW + gap
	pillH := maxInt(1, int(float64(target)*0.82))
	pillY := baseline - int(float64(target)*0.78)
	r.fillRoundRect(pillX, pillY, pillX+pillW, pillY+pillH, float64(pillH)/2, cardBgWarm)
	r.strokeRoundRect(pillX, pillY, pillX+pillW, pillY+pillH, float64(pillH)/2, 1.2, inkBorder)
	r.text(pillX+pad, baseline-int(float64(target)*0.20), "bold", weekSize, week, ink)
}

func renderLunar(r *renderer, bounds image.Rectangle, data DashboardData) {
	r.fillRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, cardBg)
	r.strokeRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, 1.2, inkBorder)
	pad := minInt(scaled(r.h, 0.035), maxInt(1, bounds.Dx()/12))
	mid := bounds.Min.X + bounds.Dx()/2
	r.rect(mid, bounds.Min.Y+bounds.Dy()/6, mid+1, bounds.Max.Y-bounds.Dy()/6, inkLine)

	lunarText := data.Day.LunarCN
	if parts := strings.Split(lunarText, " "); len(parts) >= 2 {
		lunarText = parts[1]
	}
	lunarText = strings.TrimSpace(strings.TrimPrefix(lunarText, "农历"))
	if lunarText == "" {
		lunarText = "农历暂无"
	}
	titleSize := fontSize(bounds.Dy(), 0.20)
	valueSize := fontSize(bounds.Dy(), 0.29)
	r.text(bounds.Min.X+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.36), "med", titleSize, "农历", inkMuted)
	r.text(bounds.Min.X+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.78), "bold", r.fit("bold", valueSize, lunarText, mid-bounds.Min.X-2*pad), lunarText, ink)

	title, value := "时令节气", data.Day.NextJieQi
	if len(data.Day.Badges) > 0 {
		title, value = "今日节日", strings.Join(data.Day.Badges, " · ")
	}
	if value == "" {
		value = "四季顺遂"
	}
	r.text(mid+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.36), "med", titleSize, title, inkMuted)
	r.text(mid+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.78), "bold", r.fit("bold", valueSize, value, bounds.Max.X-mid-2*pad), value, ink)
}

func renderQuote(r *renderer, bounds image.Rectangle, data DashboardData, cfg *config.Config) {
	r.fillRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, cardBgWarm)
	r.strokeRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, 1.2, inkBorder)
	pad := minInt(scaled(r.h, 0.035), maxInt(1, bounds.Dx()/12))
	contentW := maxInt(1, bounds.Dx()-2*pad)
	qType, qFont, target := "mao", "bold", fontSize(bounds.Dy(), 0.23)
	if cfg != nil {
		if cfg.QuoteType != "" {
			qType = cfg.QuoteType
		}
		switch cfg.QuoteFont {
		case "bold", "med", "reg", "kai", "wei", "xing", "lively":
			qFont = cfg.QuoteFont
		}
		if cfg.QuoteSize > 0 {
			target = cfg.QuoteSize
		}
	}
	titleSize := fontSize(bounds.Dy(), 0.15)
	if qType == "almanac" {
		r.text(bounds.Min.X+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.25), "med", titleSize, "「 传统黄历 · 今日宜忌 」", inkMuted)
		yi, ji := dayYiJi(data.Now)
		badgeSize := minInt(target, fontSize(bounds.Dy(), 0.23))
		badgeW := r.textW("bold", badgeSize, "【宜】 ")
		for i, item := range []struct {
			badge string
			text  string
		}{{"【宜】", yi}, {"【忌】", ji}} {
			y := bounds.Min.Y + int(float64(bounds.Dy())*(0.53+0.23*float64(i)))
			r.text(bounds.Min.X+pad, y, "bold", badgeSize, item.badge, ink)
			size := r.fit(qFont, target, item.text, maxInt(1, contentW-badgeW))
			r.text(bounds.Min.X+pad+badgeW, y, qFont, size, item.text, ink)
		}
		r.textRight(bounds.Max.X-pad, bounds.Min.Y+int(float64(bounds.Dy())*0.94), "med", titleSize, "—— 《协纪辨方书》· 趋吉避凶", inkSoft)
		return
	}

	title := "「 每日金句 · 毛泽东选集 」"
	switch qType {
	case "poetry":
		title = "「 经典诗词 · 千古名篇 」"
	case "famous":
		title = "「 每日名言 · 哲学与人生 」"
	case "inspirational":
		title = "「 每日励志 · 向上成长 」"
	}
	r.text(bounds.Min.X+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.25), "med", titleSize, title, inkMuted)
	quote := getDailyQuote(qType, data.Now)
	lines := splitQuoteLines(strings.TrimSpace(quote.Text), 17)
	if len(lines) == 1 {
		text := "“" + strings.TrimRight(strings.TrimSpace(lines[0]), "。") + "”"
		size := r.fit(qFont, target, text, contentW)
		r.text(bounds.Min.X+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.67), qFont, size, text, ink)
	} else {
		first := "“" + strings.TrimSpace(lines[0])
		second := strings.TrimRight(strings.TrimSpace(lines[1]), "。") + "”"
		size := minInt(r.fit(qFont, target, first, contentW), r.fit(qFont, target, second, contentW))
		r.text(bounds.Min.X+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.52), qFont, size, first, ink)
		r.text(bounds.Min.X+pad+scaled(r.h, 0.012), bounds.Min.Y+int(float64(bounds.Dy())*0.76), qFont, size, second, ink)
	}
	if quote.Source != "" {
		source := "—— " + quote.Source
		size := r.fit("med", titleSize, source, contentW)
		r.textRight(bounds.Max.X-pad, bounds.Min.Y+int(float64(bounds.Dy())*0.94), "med", size, source, inkSoft)
	}
}

func renderCurrentWeather(r *renderer, bounds image.Rectangle, data DashboardData) {
	r.fillRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, cardBgWarm)
	r.strokeRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, 1.2, inkBorder)
	pad := maxInt(2, bounds.Dx()/10)
	temp, condition, sub := "--°", "暂无天气", "等待天气数据"
	kind := weather.IconCloud
	if w := data.Weather; w != nil {
		temp = fmt.Sprintf("%.0f°", w.Current.Temp)
		condition = weather.Text(w.Current.Code)
		sub = fmt.Sprintf("体感 %.0f° · 湿度 %d%%", w.Current.Apparent, w.Current.Humidity)
		kind = weather.Icon(w.Current.Code)
	}
	tempSize := minInt(scaled(r.h, 0.110), fontSize(bounds.Dy(), 0.58))
	tempSize = r.fit("bold", tempSize, temp, maxInt(1, bounds.Dx()*55/100))
	tempW := r.text(bounds.Min.X+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.72), "bold", tempSize, temp, ink)
	conditionSize := fontSize(bounds.Dy(), 0.22)
	r.text(bounds.Min.X+pad, bounds.Min.Y+int(float64(bounds.Dy())*0.94), "bold", conditionSize, condition, ink)
	drawIcon(r, kind, float64(bounds.Max.X-pad)-float64(bounds.Dy())*0.22, float64(bounds.Min.Y)+float64(bounds.Dy())*0.42, float64(bounds.Dy())*0.64, ink)
	subSize := r.fit("med", fontSize(bounds.Dy(), 0.16), sub, maxInt(1, bounds.Dx()-tempW-3*pad))
	r.textRight(bounds.Max.X-pad, bounds.Min.Y+int(float64(bounds.Dy())*0.94), "med", subSize, sub, inkSoft)
}

func renderForecast(r *renderer, bounds image.Rectangle, data DashboardData) {
	var days []weather.DayWeather
	if data.Weather != nil {
		days = data.Weather.Days(data.Now)
	}
	gap := maxInt(1, scaled(r.h, 0.016))
	cardH := maxInt(1, (bounds.Dy()-2*gap)/3)
	labels := []string{"今天", "明天", "后天"}
	for i := 0; i < 3; i++ {
		y0 := bounds.Min.Y + i*(cardH+gap)
		y1 := minInt(bounds.Max.Y, y0+cardH)
		bg := color.Color(paper)
		if i == 0 {
			bg = cardBg
		}
		r.fillRoundRect(bounds.Min.X, y0, bounds.Max.X, y1, 12, bg)
		r.strokeRoundRect(bounds.Min.X, y0, bounds.Max.X, y1, 12, 1, inkBorder)
		pad := maxInt(4, scaled(bounds.Dx(), 0.075))
		label, condition, tempRange, precip := labels[i], "--", "--° / --°", "降水 --"
		kind := weather.IconCloud
		if i < len(days) {
			d := days[i]
			label, condition = d.Label, weather.Text(d.Code)
			tempRange = fmt.Sprintf("%.0f° / %.0f°", d.TMax, d.TMin)
			precip = fmt.Sprintf("降水 %d%%", d.Precip)
			kind = d.Kind
		}
		labelSize := fontSize(cardH, 0.29)
		subSize := fontSize(cardH, 0.21)
		r.text(bounds.Min.X+pad, y0+int(float64(cardH)*0.43), "bold", labelSize, label, ink)
		r.text(bounds.Min.X+pad, y0+int(float64(cardH)*0.78), "med", subSize, condition, inkSoft)
		drawIcon(r, kind, float64(bounds.Min.X)+0.44*float64(bounds.Dx()), float64(y0+y1)/2, float64(cardH)*0.58, ink)
		r.textRight(bounds.Max.X-pad, y0+int(float64(cardH)*0.44), "bold", r.fit("bold", labelSize, tempRange, bounds.Dx()/3), tempRange, ink)
		r.textRight(bounds.Max.X-pad, y0+int(float64(cardH)*0.79), "reg", subSize, precip, inkMuted)
	}
}

func renderSolar(r *renderer, bounds image.Rectangle, data DashboardData) {
	r.fillRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, cardBg)
	r.strokeRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, 1.2, inkBorder)
	sunrise, sunset := "06:30", "18:00"
	if data.Weather != nil {
		if days := data.Weather.Days(data.Now); len(days) > 0 {
			if days[0].Sunrise != "" && days[0].Sunset != "" {
				sunrise, sunset = days[0].Sunrise, days[0].Sunset
			}
		}
	}
	rise, okRise := parseClockMinutes(sunrise)
	set, okSet := parseClockMinutes(sunset)
	if !okRise || !okSet || set <= rise {
		rise, set = 390, 1080
	}
	nowMinutes := data.Now.Hour()*60 + data.Now.Minute()
	isDay := nowMinutes >= rise && nowMinutes <= set
	progress := 0.0
	status, line1, line2 := "", "", ""
	if isDay {
		progress = float64(nowMinutes-rise) / float64(set-rise)
		status = fmt.Sprintf("距日落 %dh %dm", (set-nowMinutes)/60, (set-nowMinutes)%60)
		line1 = fmt.Sprintf("白昼日照 %dh %dm", (set-rise)/60, (set-rise)%60)
		line2 = fmt.Sprintf("白昼已过 %d%%", int(progress*100))
	} else {
		night := 1440 - set + rise
		elapsed, remaining := 0, 0
		if nowMinutes > set {
			elapsed, remaining = nowMinutes-set, 1440-nowMinutes+rise
		} else {
			elapsed, remaining = 1440-set+nowMinutes, rise-nowMinutes
		}
		if night > 0 {
			progress = float64(elapsed) / float64(night)
		}
		status = fmt.Sprintf("距日出 %dh %dm", remaining/60, remaining%60)
		line1 = data.Day.MoonPhase
		if line1 == "" {
			line1 = "月相"
		}
		line2 = fmt.Sprintf("夜间已过 %d%%", int(progress*100))
	}
	progress = math.Max(0, math.Min(1, progress))
	pad := maxInt(3, bounds.Dx()/11)
	headerY := bounds.Min.Y + fontSize(bounds.Dy(), 0.18)
	title := "太阳轨迹"
	if isDay {
		r.drawMiniSun(float64(bounds.Min.X+pad+8), float64(headerY-8), 6, ink)
	} else {
		title = "昼夜更替"
		r.drawMiniMoon(float64(bounds.Min.X+pad+8), float64(headerY-8), 5.5, ink)
	}
	statusSize := r.fit("med", fontSize(bounds.Dy(), 0.10), status, maxInt(1, bounds.Dx()/2))
	statusW := r.textW("med", statusSize, status)
	statusX := bounds.Max.X - pad - statusW
	titleX := bounds.Min.X + pad + 24
	titleSize := r.fit("bold", fontSize(bounds.Dy(), 0.12), title, maxInt(1, statusX-titleX-12))
	r.text(titleX, headerY, "bold", titleSize, title, ink)
	r.textRight(bounds.Max.X-pad, headerY, "med", statusSize, status, inkSoft)

	arcX0 := float64(bounds.Min.X + pad + 12)
	arcX1 := float64(bounds.Max.X - pad - 12)
	horizonY := bounds.Min.Y + int(float64(bounds.Dy())*0.67)
	baseY := float64(horizonY) - math.Min(22, float64(bounds.Dy())*0.08)
	arcH := float64(bounds.Dy()) * 0.26
	r.strokeLine(float64(bounds.Min.X+pad), float64(horizonY), float64(bounds.Max.X-pad), float64(horizonY), 1.2, inkBorder)
	for i := 0; i <= 26; i++ {
		t := float64(i) / 26
		x := arcX0 + t*(arcX1-arcX0)
		y := baseY - arcH*math.Sin(math.Pi*t)
		r.fillCircle(x, y, 1.4, inkBorder)
	}
	targetX := arcX0 + progress*(arcX1-arcX0)
	targetY := baseY - arcH*math.Sin(math.Pi*progress)
	radius := math.Max(2, math.Min(14.5, math.Min(float64(bounds.Dy())*0.075, float64(bounds.Dx())*0.05)))
	if isDay {
		r.drawCartoonSun(targetX, targetY, radius, data.Now)
	} else {
		r.drawCartoonMoon(targetX, targetY, math.Max(2, radius-1.5), data.Now, data.Day.LunarDay)
	}
	timeSize := fontSize(bounds.Dy(), 0.105)
	labelSize := fontSize(bounds.Dy(), 0.085)
	r.text(bounds.Min.X+pad, horizonY+int(float64(bounds.Dy())*0.13), "bold", timeSize, sunrise, ink)
	r.text(bounds.Min.X+pad, horizonY+int(float64(bounds.Dy())*0.25), "med", labelSize, "日出", inkMuted)
	r.textRight(bounds.Max.X-pad, horizonY+int(float64(bounds.Dy())*0.13), "bold", timeSize, sunset, ink)
	r.textRight(bounds.Max.X-pad, horizonY+int(float64(bounds.Dy())*0.25), "med", labelSize, "日落", inkMuted)
	centerX := (bounds.Min.X + bounds.Max.X) / 2
	centerW := maxInt(1, bounds.Dx()-2*pad-2*r.textW("bold", timeSize, sunrise)-16)
	line1Size := r.fit("bold", timeSize, line1, centerW)
	line1W := r.textW("bold", line1Size, line1)
	r.text(centerX-line1W/2, horizonY+int(float64(bounds.Dy())*0.13), "bold", line1Size, line1, ink)
	line2Size := r.fit("med", labelSize, line2, centerW)
	line2W := r.textW("med", line2Size, line2)
	r.text(centerX-line2W/2, horizonY+int(float64(bounds.Dy())*0.25), "med", line2Size, line2, inkMuted)
}

func airQualityValues(w *weather.Weather) (aqi, quality, pm25, pm10, uv, uvText string) {
	quality, uv, uvText = "暂无数据", "0.0", "最弱"
	if w == nil {
		return "--", quality, "--", "--", uv, uvText
	}
	uvValue := w.Current.UVIndex // Zero is valid; never substitute the daily max.
	uv = fmt.Sprintf("%.1f", uvValue)
	uvText = uvLevel(uvValue)
	if !w.Current.AirQualityValid {
		return "--", quality, "--", "--", uv, uvText
	}
	aqiValue := w.Current.AQI
	if aqiValue <= 0 {
		aqiValue = w.Current.USAQI
	}
	aqi = fmt.Sprintf("%d", aqiValue)
	if aqiValue <= 0 {
		aqi = "--"
	}
	quality = w.Current.Quality
	if quality == "" && aqiValue > 0 {
		quality = aqiLevel(aqiValue)
	}
	if quality == "" {
		quality = "未知"
	}
	return aqi, quality, fmt.Sprintf("%.0f", w.Current.PM25), fmt.Sprintf("%.0f", w.Current.PM10), uv, uvText
}

func renderAirQuality(r *renderer, bounds image.Rectangle, data DashboardData) {
	r.fillRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, cardBg)
	r.strokeRoundRect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, 14, 1.2, inkBorder)
	pad := maxInt(4, bounds.Dx()/10)
	aqi, quality, pm25, pm10, uv, uvText := airQualityValues(data.Weather)
	headerY := bounds.Min.Y + int(float64(bounds.Dy())*0.16)
	r.drawMiniLeaf(float64(bounds.Min.X+pad+8), float64(headerY-7), 6.5, ink)
	badgeSize := fontSize(bounds.Dy(), 0.09)
	badgeW := r.textW("bold", badgeSize, quality) + 18
	badgeX := bounds.Max.X - pad - badgeW
	titleX := bounds.Min.X + pad + 24
	titleSize := r.fit("bold", fontSize(bounds.Dy(), 0.115), "环境空气质量", maxInt(1, badgeX-titleX-12))
	r.text(titleX, headerY, "bold", titleSize, "环境空气质量", ink)
	badgeY := headerY - fontSize(bounds.Dy(), 0.13)
	r.fillRoundRect(badgeX, badgeY, bounds.Max.X-pad, badgeY+fontSize(bounds.Dy(), 0.13), 8, cardBgWarm)
	r.strokeRoundRect(badgeX, badgeY, bounds.Max.X-pad, badgeY+fontSize(bounds.Dy(), 0.13), 8, 1, inkBorder)
	r.text(badgeX+9, headerY-2, "bold", badgeSize, quality, ink)

	heroY := bounds.Min.Y + int(float64(bounds.Dy())*0.48)
	aqiSize := fontSize(bounds.Dy(), 0.28)
	aqiW := r.text(bounds.Min.X+pad, heroY, "bold", aqiSize, aqi, ink)
	r.text(bounds.Min.X+pad, heroY+fontSize(bounds.Dy(), 0.09), "med", fontSize(bounds.Dy(), 0.08), "国标 AQI", inkMuted)
	notice := ""
	if data.Weather != nil && data.Weather.Current.AirQualityValid {
		notice = data.Weather.Current.Notice
	}
	line1, line2 := splitNoticeLines(notice, quality)
	noticeX := bounds.Min.X + pad + aqiW + maxInt(10, pad/2)
	noticeW := maxInt(1, bounds.Max.X-pad-noticeX)
	r.text(noticeX, heroY-int(float64(bounds.Dy())*0.07), "bold", r.fit("bold", fontSize(bounds.Dy(), 0.10), line1, noticeW), line1, ink)
	r.text(noticeX, heroY+int(float64(bounds.Dy())*0.06), "med", r.fit("med", fontSize(bounds.Dy(), 0.085), line2, noticeW), line2, inkSoft)

	gap := maxInt(2, bounds.Dx()/35)
	tileY0 := bounds.Min.Y + int(float64(bounds.Dy())*0.63)
	tileY1 := bounds.Min.Y + int(float64(bounds.Dy())*0.93)
	tileH := tileY1 - tileY0
	tileW := maxInt(1, (bounds.Dx()-2*pad-2*gap)/3)
	tiles := []struct{ label, value, sub string }{{"PM2.5", pm25, "μg/m³"}, {"PM10", pm10, "μg/m³"}, {"紫外线", uv, uvText}}
	for i, tile := range tiles {
		x0 := bounds.Min.X + pad + i*(tileW+gap)
		x1 := x0 + tileW
		r.fillRoundRect(x0, tileY0, x1, tileY1, 10, cardBgWarm)
		r.strokeRoundRect(x0, tileY0, x1, tileY1, 10, 1, inkBorder)
		labelSize := fontSize(tileH, 0.28)
		labelW := r.textW("med", labelSize, tile.label)
		r.text((x0+x1-labelW)/2, tileY0+int(float64(tileH)*0.36), "med", labelSize, tile.label, inkMuted)
		value := strings.TrimSpace(tile.value + " " + tile.sub)
		valueSize := r.fit("bold", fontSize(tileH, 0.32), value, maxInt(1, tileW-8))
		valueW := r.textW("bold", valueSize, value)
		r.text((x0+x1-valueW)/2, tileY0+int(float64(tileH)*0.78), "bold", valueSize, value, ink)
	}
}

func splitNoticeLines(notice, quality string) (string, string) {
	if notice == "" {
		switch {
		case strings.Contains(quality, "暂无"), strings.Contains(quality, "未知"):
			return "空气质量暂不可用", "等待有效监测数据"
		case strings.Contains(quality, "优"):
			return "空气优良 · 清新", "适宜开窗与户外运动"
		case strings.Contains(quality, "良"):
			return "空气良好 · 适宜", "可正常进行各项户外活动"
		case strings.Contains(quality, "轻度"):
			return "敏感人群微防护", "适度减少剧烈户外活动"
		case strings.Contains(quality, "中度"):
			return "轻微浑浊宜少外出", "敏感人群出门佩戴口罩"
		default:
			return "雾霾天气注意防护", "外出请佩戴专业防尘口罩"
		}
	}
	for _, sep := range []string{"，", "、", "；", "。"} {
		if idx := strings.Index(notice, sep); idx > 0 && idx < len(notice)-len(sep) {
			return strings.TrimSpace(notice[:idx]), strings.TrimSpace(notice[idx+len(sep):])
		}
	}
	runes := []rune(notice)
	if len(runes) > 12 {
		return string(runes[:12]), string(runes[12:])
	}
	return notice, "出门请注意防护"
}

func parseClockMinutes(value string) (int, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, false
	}
	hour, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	minute, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func renderFooter(r *renderer, bounds image.Rectangle, data DashboardData, cfg *config.Config) {
	r.rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+1, inkLine)
	location := ""
	if cfg != nil {
		location = cfg.Location.Name
	}
	left := location
	if data.Weather != nil {
		left += fmt.Sprintf(" · 更新于 %s", data.Weather.Fetched.Format("15:04"))
		if data.Weather.Offline {
			left += " (离线)"
		} else if data.Weather.Cached {
			left += " (缓存)"
		}
	} else {
		left += " · 等待天气数据"
	}
	size := fontSize(bounds.Dy(), 0.50)
	baseline := bounds.Min.Y + int(float64(bounds.Dy())*0.72)
	leftSize := r.fit("reg", size, left, maxInt(1, bounds.Dx()*48/100))
	r.text(bounds.Min.X, baseline, "reg", leftSize, left, inkMuted)

	curX := bounds.Max.X

	// 1. 最右侧：风向风级 (如果有)
	if data.Weather != nil && data.Weather.Current.WindDir != "" {
		windText := strings.TrimSpace(data.Weather.Current.WindDir + " " + data.Weather.Current.WindScale)
		windW := r.textW("reg", size, windText)
		r.text(curX-windW, baseline, "reg", size, windText, inkMuted)
		curX -= windW + scaled(r.h, 0.012)

		// 间隔点
		dotText := "·"
		dotW := r.textW("reg", size, dotText)
		r.text(curX-dotW, baseline, "reg", size, dotText, inkSoft)
		curX -= dotW + scaled(r.h, 0.012)
	}

	// 2. 电池信息：纯矢量自绘电池图标（充电中显示自绘闪电符号，正常状态显示电量进度条）+ 电量百分比
	if data.HasBattery {
		battText := fmt.Sprintf("%d%%", data.Battery)
		battW := r.textW("reg", size, battText)
		r.text(curX-battW, baseline, "reg", size, battText, inkMuted)
		curX -= battW + scaled(r.h, 0.008)

		iconW, iconH := 24.0, 13.0
		iconY := float64(baseline) - float64(size)*0.74
		iconX := float64(curX) - iconW - 3.0
		r.drawBatteryIcon(iconX, iconY, iconW, iconH, data.Battery, data.IsCharging, inkMuted)
		curX = int(iconX) - scaled(r.h, 0.018)
	}

	// 3. Wi-Fi 图标：规整、对称且像素对齐的矢量图标，彻底去除“WiFi 在线”多余文字
	wifiSize := 16.0
	wifiCX := float64(curX) - wifiSize*0.6
	wifiCY := float64(baseline) - float64(size)*0.45
	r.drawWiFiIcon(wifiCX, wifiCY, wifiSize, inkMuted)
}

func clockWeightKey(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "bebas", "bebasneue", "clock-bebas":
		return "clock-bebas"
	case "jetbrains", "jetbrainsmono", "clock-jetbrains":
		return "clock-jetbrains"
	case "montserrat", "clock-montserrat":
		return "clock-montserrat"
	case "playfair", "playfairdisplay", "clock-playfair":
		return "clock-playfair"
	case "noto", "notosans", "bold", "default":
		return "bold"
	default:
		return "clock-bebas"
	}
}

func isClockWeight(value string) bool {
	return strings.HasPrefix(value, "clock-")
}

func dateWeightKey(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "med", "medium", "noto-med":
		return "med"
	case "reg", "regular", "noto-reg":
		return "reg"
	case "playfair", "serif":
		return "clock-playfair"
	case "bebas":
		return "clock-bebas"
	case "jetbrains":
		return "clock-jetbrains"
	case "montserrat":
		return "clock-montserrat"
	default:
		return "bold"
	}
}

func aqiLevel(aqi int) string {
	switch {
	case aqi <= 0:
		return "未知"
	case aqi <= 50:
		return "优"
	case aqi <= 100:
		return "良"
	case aqi <= 150:
		return "轻度污染"
	case aqi <= 200:
		return "中度污染"
	case aqi <= 300:
		return "重度污染"
	default:
		return "严重污染"
	}
}

func uvLevel(uv float64) string {
	switch {
	case uv < 0.5:
		return "最弱"
	case uv < 3:
		return "较弱"
	case uv < 6:
		return "中等"
	case uv < 8:
		return "较强"
	case uv < 11:
		return "极强"
	default:
		return "危险"
	}
}

func splitQuoteLines(s string, maxCharsPerLine int) []string {
	runes := []rune(s)
	if len(runes) <= maxCharsPerLine {
		return []string{s}
	}
	best, mid := -1, len(runes)/2
	for i, ru := range runes {
		if ru == '，' || ru == '、' || ru == '；' {
			if best == -1 || absVal(i-mid) < absVal(best-mid) {
				best = i
			}
		}
	}
	if best > 0 && best < len(runes)-1 {
		return []string{string(runes[:best+1]), string(runes[best+1:])}
	}
	return []string{string(runes[:mid]), string(runes[mid:])}
}

func absVal(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
