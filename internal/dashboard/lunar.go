package dashboard

// 农历/节气/节日：基于 github.com/6tail/lunar-go（1900–2100 精确历表），完全本地计算。

import (
	"container/list"
	"fmt"
	"strings"
	"time"

	"github.com/6tail/lunar-go/calendar"
)

type DayInfoData struct {
	Clock     string   // 14:05
	DateCN    string   // 2026年11月14日
	WeekCN    string   // 星期六
	LunarCN   string   // 农历丙午马年 九月廿六
	LunarDay  int      // 农历日期 (1-30)
	MoonPhase string   // 峨眉月 / 上弦月 / 满月 等
	Badges    []string // 今日节日/节气
	NextJieQi string   // 小雪 · 8天后（无节日时展示）
}

func moonPhaseName(lunarDay int) string {
	switch {
	case lunarDay == 1 || lunarDay >= 30:
		return "新月"
	case lunarDay >= 2 && lunarDay <= 6:
		return "峨眉月"
	case lunarDay == 7 || lunarDay == 8:
		return "上弦月"
	case lunarDay >= 9 && lunarDay <= 14:
		return "盈凸月"
	case lunarDay == 15 || lunarDay == 16:
		return "满月"
	case lunarDay >= 17 && lunarDay <= 21:
		return "亏凸月"
	case lunarDay == 22 || lunarDay == 23:
		return "下弦月"
	case lunarDay >= 24 && lunarDay <= 29:
		return "残月"
	default:
		return "月相"
	}
}

// DayInfo returns the local solar/lunar labels used by a frame.
func DayInfo(now time.Time) DayInfoData {
	return dayInfo(now)
}

func dayInfo(now time.Time) DayInfoData {
	lunar := calendar.NewLunarFromDate(now)
	solar := calendar.NewSolarFromDate(now)

	m := lunar.GetMonth()
	leap := m < 0
	if m < 0 {
		m = -m
	}
	mnames := []string{"", "正", "二", "三", "四", "五", "六", "七", "八", "九", "十", "冬", "腊"}
	mm := mnames[m] + "月"
	if leap {
		mm = "闰" + mm
	}

	ld := lunar.GetDay()
	info := DayInfoData{
		Clock:  now.Format("15:04"),
		DateCN: fmt.Sprintf("%d年%d月%d日", now.Year(), int(now.Month()), now.Day()),
		WeekCN: weekFull(now.Weekday()),
		LunarCN: fmt.Sprintf("农历%s%s年 %s%s",
			lunar.GetYearInGanZhi(), lunar.GetYearShengXiao(), mm, lunar.GetDayInChinese()),
		LunarDay:  ld,
		MoonPhase: moonPhaseName(ld),
	}

	for _, src := range []*list.List{lunar.GetFestivals(), solar.GetFestivals()} {
		for e := src.Front(); e != nil; e = e.Next() {
			if s, ok := e.Value.(string); ok && s != "" {
				info.Badges = append(info.Badges, s)
			}
		}
	}

	// 今日恰逢节气/中气
	if jq := lunar.GetCurrentJieQi(); jq != nil {
		s := jq.GetSolar()
		if s.GetYear() == now.Year() && int(s.GetMonth()) == int(now.Month()) && s.GetDay() == now.Day() {
			info.Badges = append(info.Badges, jq.GetName())
		}
	}

	// 下一个节气倒计时（当日无节日/节气时展示）
	if jq := lunar.GetNextJieQi(); jq != nil {
		s := jq.GetSolar()
		target := time.Date(s.GetYear(), time.Month(s.GetMonth()), s.GetDay(), 0, 0, 0, 0, now.Location())
		days := calendarDaysBetween(now, target)
		if days > 0 {
			info.NextJieQi = fmt.Sprintf("%s · %d天后", jq.GetName(), days)
		}
	}

	seen := map[string]bool{}
	var uniq []string
	for _, b := range info.Badges {
		if !seen[b] {
			seen[b] = true
			uniq = append(uniq, b)
		}
	}
	info.Badges = uniq
	return info
}

func calendarDaysBetween(from, to time.Time) int {
	fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(toDay.Sub(fromDay).Hours() / 24)
}

func weekFull(d time.Weekday) string {
	names := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	return names[int(d)]
}

// dayYiJi 返回当天老黄历的“宜”和“忌”事项精选（最多 5-6 项，空格隔开）
func dayYiJi(now time.Time) (yi string, ji string) {
	lunar := calendar.NewLunarFromDate(now)
	var yiItems, jiItems []string

	if listYi := lunar.GetDayYi(); listYi != nil {
		for e := listYi.Front(); e != nil; e = e.Next() {
			if s, ok := e.Value.(string); ok && s != "" {
				yiItems = append(yiItems, s)
			}
		}
	}
	if listJi := lunar.GetDayJi(); listJi != nil {
		for e := listJi.Front(); e != nil; e = e.Next() {
			if s, ok := e.Value.(string); ok && s != "" {
				jiItems = append(jiItems, s)
			}
		}
	}

	limit := 6
	if len(yiItems) > limit {
		yiItems = yiItems[:limit]
	}
	if len(jiItems) > limit {
		jiItems = jiItems[:limit]
	}

	if len(yiItems) == 0 {
		yi = "诸事皆宜"
	} else {
		yi = strings.Join(yiItems, " ")
	}
	if len(jiItems) == 0 {
		ji = "诸事不忌"
	} else {
		ji = strings.Join(jiItems, " ")
	}
	return yi, ji
}
