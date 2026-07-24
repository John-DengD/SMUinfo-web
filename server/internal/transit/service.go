package transit

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Constants mirror TransitService.java verbatim.
const (
	statusActive = "ACTIVE"

	lineMetro16 = "METRO_16"
	lineBus1077 = "BUS_1077"

	dirToLongyang   = "TO_LONGYANG"
	dirToLingangAve = "TO_LINGANG_AVE"
	dirToShared     = "TO_SHARED"

	serviceNormal = "NORMAL"

	specialWindowMinutes = 30

	stationLingangHub = "临港大道枢纽站"
	stationSharedHub  = "临港共享区枢纽站"
)

// zone matches Java's ZoneId.of("Asia/Shanghai").
var zone = mustLoadZone("Asia/Shanghai")

func mustLoadZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// Querier is the subset of *gen.Queries the transit service needs.
type Querier interface {
	ListActiveTransitDepartures(ctx context.Context) ([]gen.TransitDeparture, error)
	ListTransitDepartures(ctx context.Context, arg gen.ListTransitDeparturesParams) ([]gen.TransitDeparture, error)
}

type Service struct {
	q Querier
	// now is injectable for tests; nil means time.Now().
	now func() time.Time
}

func NewService(q Querier) *Service {
	return &Service{q: q}
}

// ---- wire DTOs (camelCase JSON, mirrors TransitDTO.java) ----

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Departure struct {
	Time         string `json:"time"`
	MinutesUntil *int   `json:"minutesUntil"`
	Text         string `json:"text"`
	ServiceType  string `json:"serviceType"`
	ServiceLabel string `json:"serviceLabel"`
}

type NextResp struct {
	Now            string     `json:"now"`
	Line           string     `json:"line"`
	LineName       string     `json:"lineName"`
	Station        string     `json:"station"`
	StationName    string     `json:"stationName"`
	Direction      string     `json:"direction"`
	DirectionName  string     `json:"directionName"`
	ScheduleType   string     `json:"scheduleType"`
	StatusText     string     `json:"statusText"`
	Nearest        *Departure `json:"nearest"`
	Next           *Departure `json:"next"`
	SpecialNearest *Departure `json:"specialNearest"`
	SpecialNext    *Departure `json:"specialNext"`
	Lines          []Option   `json:"lines"`
	Stations       []Option   `json:"stations"`
	Directions     []Option   `json:"directions"`
}

// Next replicates TransitService.next(line, station, direction).
func (s *Service) Next(ctx context.Context, line, station, direction string) (*NextResp, error) {
	now := time.Now().In(zone)
	if s.now != nil {
		now = s.now().In(zone)
	}

	activeRows, err := s.q.ListActiveTransitDepartures(ctx)
	if err != nil {
		return nil, err
	}
	if len(activeRows) == 0 {
		return nil, httpx.Biz("暂无时刻数据")
	}

	selectedLine := selectValue(line, activeRows, lineCodeOf, lineMetro16)
	lineRows := filter(activeRows, func(r gen.TransitDeparture) bool { return r.LineCode == selectedLine })
	if len(lineRows) == 0 {
		selectedLine = activeRows[0].LineCode
		lineRows = filter(activeRows, func(r gen.TransitDeparture) bool { return r.LineCode == selectedLine })
	}

	selectedDirection := selectDirection(selectedLine, station, direction, lineRows)
	selectedStation := selectStation(selectedLine, station, selectedDirection, lineRows)
	scheduleType := scheduleTypeFor(selectedLine, now)

	departures, err := s.q.ListTransitDepartures(ctx, gen.ListTransitDeparturesParams{
		LineCode:      selectedLine,
		ScheduleType:  scheduleType,
		DirectionCode: selectedDirection,
		StationCode:   selectedStation,
	})
	if err != nil {
		return nil, err
	}
	if len(departures) == 0 {
		return nil, httpx.Biz("当前站点方向暂无时刻")
	}

	first := departures[0]
	resp := buildResponse(now, first, selectedLine, selectedStation, selectedDirection, departures)
	applySpecialDepartures(resp, now, departures)
	resp.Lines = options(activeRows, lineCodeOf, lineNameOf)
	resp.Stations = options(lineRows, stationCodeOf, stationNameOf)
	resp.Directions = options(lineRows, directionCodeOf, directionNameOf)
	return resp, nil
}

func buildResponse(now time.Time, first gen.TransitDeparture, line, station, direction string,
	departures []gen.TransitDeparture) *NextResp {
	resp := &NextResp{
		Now:           now.Format("2006-01-02 15:04"),
		Line:          line,
		LineName:      first.LineName,
		Station:       station,
		StationName:   first.StationName,
		Direction:     direction,
		DirectionName: first.DirectionName,
		ScheduleType:  first.ScheduleTypeName,
	}

	upcoming := upcomingDepartures(now, departures)
	if len(upcoming) == 0 {
		resp.StatusText = "今日末班已过"
		return resp
	}
	resp.Nearest = &upcoming[0]
	if len(upcoming) > 1 {
		resp.Next = &upcoming[1]
	}
	resp.StatusText = "正常"
	return resp
}

func applySpecialDepartures(resp *NextResp, now time.Time, departures []gen.TransitDeparture) {
	specialRows := filter(departures, func(r gen.TransitDeparture) bool {
		return r.ServiceType != "" && r.ServiceType != serviceNormal
	})
	all := upcomingDepartures(now, specialRows)
	special := make([]Departure, 0, len(all))
	for _, d := range all {
		if d.MinutesUntil != nil && *d.MinutesUntil <= specialWindowMinutes {
			special = append(special, d)
		}
	}
	if len(special) > 0 {
		resp.SpecialNearest = &special[0]
		if len(special) > 1 {
			resp.SpecialNext = &special[1]
		}
	}
}

// upcomingDepartures mirrors TransitService.upcoming: departures at/after the
// current minute (seconds/nanos truncated), capped at the first 2.
func upcomingDepartures(now time.Time, departures []gen.TransitDeparture) []Departure {
	currentMinute := minutesSinceMidnight(now)
	result := make([]Departure, 0, 2)
	for _, row := range departures {
		depMinute := timeToMinutes(row.DepartureTime)
		if depMinute < currentMinute {
			continue
		}
		minutes := depMinute - currentMinute
		mu := minutes
		if mu < 0 {
			mu = 0
		}
		text := "现在"
		if minutes > 0 {
			text = itoa(minutes) + "分钟后"
		}
		d := Departure{
			Time:         formatDepartureTime(row.DepartureTime),
			MinutesUntil: &mu,
			Text:         text,
			ServiceType:  row.ServiceType,
			ServiceLabel: row.ServiceLabel,
		}
		result = append(result, d)
		if len(result) == 2 {
			break
		}
	}
	return result
}

func selectDirection(line, station, direction string, rows []gen.TransitDeparture) string {
	if line == lineBus1077 {
		if direction == dirToLingangAve || direction == dirToShared {
			return direction
		}
		if station == stationLingangHub {
			return dirToShared
		}
		return dirToLingangAve
	}
	fallback := dirToLongyang
	for _, r := range rows {
		fallback = r.DirectionCode
		break
	}
	return selectValue(direction, rows, directionCodeOf, fallback)
}

func selectStation(line, station, direction string, rows []gen.TransitDeparture) string {
	if line == lineBus1077 {
		if direction == dirToShared {
			return stationLingangHub
		}
		return stationSharedHub
	}
	directionRows := filter(rows, func(r gen.TransitDeparture) bool { return r.DirectionCode == direction })
	return selectValue(station, directionRows, stationCodeOf, "罗山路")
}

func scheduleTypeFor(line string, t time.Time) string {
	day := t.Weekday()
	if line == lineBus1077 {
		switch day {
		case time.Friday:
			return "BUS_FRIDAY"
		case time.Sunday:
			return "BUS_SUNDAY"
		default:
			return "BUS_NORMAL"
		}
	}
	if day == time.Saturday || day == time.Sunday {
		return "WEEKEND"
	}
	return "WEEKDAY"
}

// selectValue mirrors TransitService.selectValue.
func selectValue(candidate string, rows []gen.TransitDeparture,
	getter func(gen.TransitDeparture) string, fallback string) string {
	if candidate != "" {
		for _, r := range rows {
			if getter(r) == candidate {
				return candidate
			}
		}
	}
	if fallback != "" {
		for _, r := range rows {
			if getter(r) == fallback {
				return fallback
			}
		}
	}
	for _, r := range rows {
		return getter(r)
	}
	return fallback
}

// options mirrors TransitService.options: distinct value->label preserving
// first-seen order.
func options(rows []gen.TransitDeparture,
	valueGetter, labelGetter func(gen.TransitDeparture) string) []Option {
	seen := map[string]bool{}
	out := make([]Option, 0)
	for _, r := range rows {
		v := valueGetter(r)
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, Option{Value: v, Label: labelGetter(r)})
	}
	return out
}

func filter(rows []gen.TransitDeparture, pred func(gen.TransitDeparture) bool) []gen.TransitDeparture {
	out := make([]gen.TransitDeparture, 0, len(rows))
	for _, r := range rows {
		if pred(r) {
			out = append(out, r)
		}
	}
	return out
}

// ---- pgtype.Time helpers ----

// minutesSinceMidnight truncates now to the minute (Java: withSecond(0).withNano(0)).
func minutesSinceMidnight(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

// timeToMinutes converts a PG TIME (microseconds since midnight) to whole
// minutes since midnight. Seed rows are always at minute granularity.
func timeToMinutes(t pgtype.Time) int {
	if !t.Valid {
		return 0
	}
	return int(t.Microseconds / 60_000_000)
}

// formatDepartureTime renders a PG TIME as "HH:mm" (Java DateTimeFormatter "HH:mm").
func formatDepartureTime(t pgtype.Time) string {
	total := timeToMinutes(t)
	h := total / 60
	m := total % 60
	return pad2(h) + ":" + pad2(m)
}

func pad2(v int) string {
	if v < 10 {
		return "0" + itoa(v)
	}
	return itoa(v)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ---- field getters ----

func lineCodeOf(r gen.TransitDeparture) string      { return r.LineCode }
func lineNameOf(r gen.TransitDeparture) string      { return r.LineName }
func stationCodeOf(r gen.TransitDeparture) string   { return r.StationCode }
func stationNameOf(r gen.TransitDeparture) string   { return r.StationName }
func directionCodeOf(r gen.TransitDeparture) string { return r.DirectionCode }
func directionNameOf(r gen.TransitDeparture) string { return r.DirectionName }
