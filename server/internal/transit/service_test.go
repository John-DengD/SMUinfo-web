package transit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// ---- stub Querier ----

type stubQuerier struct {
	active      []gen.TransitDeparture
	lastParams  gen.ListTransitDeparturesParams
	filtered    []gen.TransitDeparture
	filterByArg bool
}

func (s *stubQuerier) ListActiveTransitDepartures(_ context.Context) ([]gen.TransitDeparture, error) {
	return s.active, nil
}

func (s *stubQuerier) ListTransitDepartures(_ context.Context, arg gen.ListTransitDeparturesParams) ([]gen.TransitDeparture, error) {
	s.lastParams = arg
	if s.filterByArg {
		out := make([]gen.TransitDeparture, 0)
		for _, r := range s.active {
			if r.Status == statusActive && r.LineCode == arg.LineCode && r.ScheduleType == arg.ScheduleType &&
				r.DirectionCode == arg.DirectionCode && r.StationCode == arg.StationCode {
				out = append(out, r)
			}
		}
		return out, nil
	}
	return s.filtered, nil
}

// pgTime builds a PG TIME from HH:mm.
func pgTime(h, m int) pgtype.Time {
	return pgtype.Time{Microseconds: int64(h*60+m) * 60_000_000, Valid: true}
}

func mkRow(line, sched, dir, station string, h, m int, svcType, svcLabel string, sort int32) gen.TransitDeparture {
	return gen.TransitDeparture{
		LineCode: line, LineName: line + "-name",
		StationCode: station, StationName: station + "-name",
		DirectionCode: dir, DirectionName: dir + "-name",
		ScheduleType: sched, ScheduleTypeName: sched + "-name",
		DepartureTime: pgTime(h, m),
		ServiceType:   svcType, ServiceLabel: svcLabel,
		SortOrder: sort, Status: statusActive,
	}
}

func withNow(s *Service, t time.Time) *Service { s.now = func() time.Time { return t }; return s }

// A Wednesday (weekday) at 08:00 in Asia/Shanghai.
func wednesday0800() time.Time {
	return time.Date(2026, 7, 22, 8, 0, 0, 0, zone) // 2026-07-22 is a Wednesday
}

func TestNextEmptyActive(t *testing.T) {
	svc := NewService(&stubQuerier{active: nil})
	_, err := svc.Next(context.Background(), "", "", "")
	var be httpx.BizError
	if !errors.As(err, &be) || be.Msg != "暂无时刻数据" {
		t.Fatalf("expected 暂无时刻数据 BizError, got %v", err)
	}
}

func TestScheduleTypeSelection(t *testing.T) {
	cases := []struct {
		line string
		day  time.Time
		want string
	}{
		{lineMetro16, time.Date(2026, 7, 22, 8, 0, 0, 0, zone), "WEEKDAY"}, // Wed
		{lineMetro16, time.Date(2026, 7, 25, 8, 0, 0, 0, zone), "WEEKEND"}, // Sat
		{lineMetro16, time.Date(2026, 7, 26, 8, 0, 0, 0, zone), "WEEKEND"}, // Sun
		{lineBus1077, time.Date(2026, 7, 24, 8, 0, 0, 0, zone), "BUS_FRIDAY"},
		{lineBus1077, time.Date(2026, 7, 26, 8, 0, 0, 0, zone), "BUS_SUNDAY"},
		{lineBus1077, time.Date(2026, 7, 22, 8, 0, 0, 0, zone), "BUS_NORMAL"}, // Wed
		{lineBus1077, time.Date(2026, 7, 25, 8, 0, 0, 0, zone), "BUS_NORMAL"}, // Sat
	}
	for _, c := range cases {
		if got := scheduleTypeFor(c.line, c.day); got != c.want {
			t.Errorf("scheduleTypeFor(%s,%v)=%s want %s", c.line, c.day.Weekday(), got, c.want)
		}
	}
}

func TestNextDefaultsAndUpcoming(t *testing.T) {
	// Two metro lines exist; default should pick METRO_16, TO_LONGYANG, 罗山路, WEEKDAY.
	rows := []gen.TransitDeparture{
		mkRow(lineMetro16, "WEEKDAY", "TO_LONGYANG", "罗山路", 7, 42, "EXPRESS_DIRECT", "大站/直达", 1000),
		mkRow(lineMetro16, "WEEKDAY", "TO_LONGYANG", "罗山路", 8, 15, "EXPRESS_DIRECT", "大站/直达", 1001),
		mkRow(lineMetro16, "WEEKDAY", "TO_LONGYANG", "罗山路", 8, 45, "EXPRESS_DIRECT", "大站/直达", 1002),
		mkRow(lineMetro16, "WEEKDAY", "TO_DISHUI", "龙阳路", 9, 0, "EXPRESS_DIRECT", "大站/直达", 1400),
	}
	svc := withNow(NewService(&stubQuerier{active: rows, filterByArg: true}), wednesday0800())
	resp, err := svc.Next(context.Background(), "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Line != lineMetro16 || resp.Direction != "TO_LONGYANG" || resp.Station != "罗山路" || resp.ScheduleType != "WEEKDAY-name" {
		t.Fatalf("bad defaults: %+v", resp)
	}
	if resp.Now != "2026-07-22 08:00" {
		t.Fatalf("now=%q", resp.Now)
	}
	// 07:42 is in the past; nearest is 08:15 (15 min), next is 08:45 (45 min).
	if resp.Nearest == nil || resp.Nearest.Time != "08:15" || resp.Nearest.MinutesUntil == nil || *resp.Nearest.MinutesUntil != 15 {
		t.Fatalf("nearest=%+v", resp.Nearest)
	}
	if resp.Nearest.Text != "15分钟后" {
		t.Fatalf("nearest text=%q", resp.Nearest.Text)
	}
	if resp.Next == nil || resp.Next.Time != "08:45" || *resp.Next.MinutesUntil != 45 {
		t.Fatalf("next=%+v", resp.Next)
	}
	if resp.StatusText != "正常" {
		t.Fatalf("statusText=%q", resp.StatusText)
	}
	// Options: only one line here, direction/station options come from lineRows.
	if len(resp.Lines) != 1 || resp.Lines[0].Value != lineMetro16 {
		t.Fatalf("lines=%+v", resp.Lines)
	}
	if len(resp.Directions) != 2 {
		t.Fatalf("directions=%+v", resp.Directions)
	}
}

func TestNextLastDeparturePassed(t *testing.T) {
	rows := []gen.TransitDeparture{
		mkRow(lineMetro16, "WEEKDAY", "TO_LONGYANG", "罗山路", 7, 0, "NORMAL", "普通车", 1000),
	}
	svc := withNow(NewService(&stubQuerier{active: rows, filterByArg: true}), wednesday0800())
	resp, err := svc.Next(context.Background(), "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusText != "今日末班已过" || resp.Nearest != nil {
		t.Fatalf("expected last-departure-passed, got %+v", resp)
	}
}

func TestNextSpecialDepartures(t *testing.T) {
	// A special (non-NORMAL) departure within 30 min should populate specialNearest.
	rows := []gen.TransitDeparture{
		mkRow(lineMetro16, "WEEKDAY", "TO_LONGYANG", "罗山路", 8, 10, "EXPRESS_DIRECT", "大站/直达", 1000),
		mkRow(lineMetro16, "WEEKDAY", "TO_LONGYANG", "罗山路", 8, 25, "EXPRESS_DIRECT", "大站/直达", 1001),
		mkRow(lineMetro16, "WEEKDAY", "TO_LONGYANG", "罗山路", 9, 0, "EXPRESS_DIRECT", "大站/直达", 1002), // 60 min > window
	}
	svc := withNow(NewService(&stubQuerier{active: rows, filterByArg: true}), wednesday0800())
	resp, err := svc.Next(context.Background(), "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.SpecialNearest == nil || resp.SpecialNearest.Time != "08:10" {
		t.Fatalf("specialNearest=%+v", resp.SpecialNearest)
	}
	if resp.SpecialNext == nil || resp.SpecialNext.Time != "08:25" {
		t.Fatalf("specialNext=%+v", resp.SpecialNext)
	}
}

func TestNextBus1077DirectionStationDefaults(t *testing.T) {
	rows := []gen.TransitDeparture{
		mkRow(lineBus1077, "BUS_NORMAL", "TO_LINGANG_AVE", stationSharedHub, 8, 30, "NORMAL", "普通车", 3000),
		mkRow(lineBus1077, "BUS_NORMAL", "TO_SHARED", stationLingangHub, 8, 40, "NORMAL", "普通车", 3100),
	}
	q := &stubQuerier{active: rows, filterByArg: true}
	svc := withNow(NewService(q), wednesday0800()) // Wed -> BUS_NORMAL
	// No direction/station: bus defaults to TO_LINGANG_AVE + 临港共享区枢纽站.
	resp, err := svc.Next(context.Background(), lineBus1077, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Direction != "TO_LINGANG_AVE" || resp.Station != stationSharedHub {
		t.Fatalf("bus defaults dir=%q station=%q", resp.Direction, resp.Station)
	}
	if q.lastParams.ScheduleType != "BUS_NORMAL" {
		t.Fatalf("schedule=%q", q.lastParams.ScheduleType)
	}
	// Requesting TO_SHARED flips station to 临港大道枢纽站.
	resp2, err := svc.Next(context.Background(), lineBus1077, "", "TO_SHARED")
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Direction != "TO_SHARED" || resp2.Station != stationLingangHub {
		t.Fatalf("bus TO_SHARED dir=%q station=%q", resp2.Direction, resp2.Station)
	}
}

func TestNextNoScheduleRows(t *testing.T) {
	// Active rows exist but none match the computed schedule/direction/station.
	rows := []gen.TransitDeparture{
		mkRow(lineMetro16, "WEEKEND", "TO_LONGYANG", "罗山路", 8, 0, "NORMAL", "普通车", 1600),
	}
	svc := withNow(NewService(&stubQuerier{active: rows, filterByArg: true}), wednesday0800()) // Wed -> WEEKDAY, no match
	_, err := svc.Next(context.Background(), "", "", "")
	var be httpx.BizError
	if !errors.As(err, &be) || be.Msg != "当前站点方向暂无时刻" {
		t.Fatalf("expected 当前站点方向暂无时刻, got %v", err)
	}
}

func TestFormatDepartureTime(t *testing.T) {
	if got := formatDepartureTime(pgTime(7, 3)); got != "07:03" {
		t.Fatalf("got %q", got)
	}
	if got := formatDepartureTime(pgTime(23, 54)); got != "23:54" {
		t.Fatalf("got %q", got)
	}
}
