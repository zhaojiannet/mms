package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("加载时区 %s: %v", name, err)
	}
	return loc
}

func TestMonthBoundsLocalCalendar(t *testing.T) {
	sh := mustLoadLocation(t, "Asia/Shanghai")
	now := time.Date(2026, 10, 24, 15, 0, 0, 0, sh)

	bounds := monthBounds(now, 12)
	if len(bounds) != 13 {
		t.Fatalf("12 个月应有 13 个月初，got %d", len(bounds))
	}
	want := time.Date(2025, 11, 1, 0, 0, 0, 0, sh)
	for i, b := range bounds {
		if !b.Valid || !b.Time.Equal(want) {
			t.Errorf("bounds[%d] = %v，应为 %v", i, b.Time, want)
		}
		want = want.AddDate(0, 1, 0)
	}
}

func TestMonthBoundsCrossYearFromJanuary(t *testing.T) {
	sh := mustLoadLocation(t, "Asia/Shanghai")
	bounds := monthBounds(time.Date(2026, 1, 31, 23, 59, 0, 0, sh), 3)

	want := []time.Time{
		time.Date(2025, 11, 1, 0, 0, 0, 0, sh),
		time.Date(2025, 12, 1, 0, 0, 0, 0, sh),
		time.Date(2026, 1, 1, 0, 0, 0, 0, sh),
		time.Date(2026, 2, 1, 0, 0, 0, 0, sh),
	}
	if len(bounds) != len(want) {
		t.Fatalf("len = %d，应为 %d", len(bounds), len(want))
	}
	for i := range want {
		if !bounds[i].Time.Equal(want[i]) {
			t.Errorf("bounds[%d] = %v，应为 %v", i, bounds[i].Time, want[i])
		}
	}
}

func TestParseDateRangeEndIsNextLocalMidnightAcrossDST(t *testing.T) {
	ny := mustLoadLocation(t, "America/New_York")

	// 2026-03-08 是纽约夏令时开始日，当天只有 23 小时
	req := httptest.NewRequest(http.MethodGet, "/?start_date=2026-03-08&end_date=2026-03-08", nil)
	c := echo.New().NewContext(req, httptest.NewRecorder())

	start, end, err := parseDateRange(c, ny)
	if err != nil {
		t.Fatalf("parseDateRange: %v", err)
	}
	if want := time.Date(2026, 3, 8, 0, 0, 0, 0, ny); !start.Time.Equal(want) {
		t.Errorf("start = %v，应为 %v", start.Time, want)
	}
	if want := time.Date(2026, 3, 9, 0, 0, 0, 0, ny); !end.Time.Equal(want) {
		t.Errorf("end = %v，应为次日本地 0 点 %v", end.Time, want)
	}
}
