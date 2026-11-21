package dashboard

import (
	"testing"
	"time"
)

func TestDayInfoKnownDate(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 11, 14, 13, 22, 0, 0, loc)
	info := DayInfo(now)
	if info.Clock != "13:22" || info.DateCN != "2026年11月14日" || info.WeekCN != "星期六" {
		t.Fatalf("公历信息错误: %+v", info)
	}
	if info.LunarCN != "农历丙午马年 十月初六" {
		t.Fatalf("农历信息错误: %q", info.LunarCN)
	}
	if info.NextJieQi == "" {
		t.Fatal("无节日当天应给出下一个节气")
	}
	if alias := dayInfo(now); alias.LunarCN != info.LunarCN {
		t.Fatal("包内 dayInfo 别名结果不一致")
	}
}

func TestCalendarDaysBetweenAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 3, 7, 12, 0, 0, 0, loc)
	to := time.Date(2026, 3, 9, 0, 0, 0, 0, loc)
	if got := calendarDaysBetween(from, to); got != 2 {
		t.Fatalf("calendar day difference=%d, want 2 across spring DST", got)
	}
	from = time.Date(2026, 10, 31, 12, 0, 0, 0, loc)
	to = time.Date(2026, 11, 2, 0, 0, 0, 0, loc)
	if got := calendarDaysBetween(from, to); got != 2 {
		t.Fatalf("calendar day difference=%d, want 2 across autumn DST", got)
	}
}

func TestLunarYiJi(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	yi, ji := dayYiJi(time.Date(2026, 11, 15, 9, 0, 0, 0, loc))
	if yi == "" || ji == "" {
		t.Fatalf("黄历宜忌不能为空: yi=%q ji=%q", yi, ji)
	}
}
