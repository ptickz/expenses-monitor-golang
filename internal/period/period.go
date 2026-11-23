package period

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type WeekMode string

const (
	FromPeriodStart WeekMode = "FROM_PERIOD_START"
	CalendarWeeks   WeekMode = "CALENDAR_WEEKS"
	daysInWeek      int      = 7
)

var weekModes = []WeekMode{FromPeriodStart, CalendarWeeks}

func (m WeekMode) Valid() bool {
	return slices.Contains(weekModes, m)
}

var ErrUnknownWeekMode = errors.New("unknown week mode")

type Calendar struct {
	startDay int
	weekMode WeekMode
	loc      *time.Location
}

type Period struct {
	Start     time.Time
	NextStart time.Time
}

type SlotKind int

const (
	SlotRegular SlotKind = iota
	SlotPairFirst
	SlotPairLast
)

type WeekSlot struct {
	Index     int
	Start     time.Time
	NextStart time.Time
	Kind      SlotKind
}

func New(startDay int, mode WeekMode, loc *time.Location) (Calendar, error) {
	if startDay < 1 || startDay > 31 {
		return Calendar{}, fmt.Errorf("start day %d out of range 1..31", startDay)
	}
	if loc == nil {
		return Calendar{}, errors.New("nil location")
	}
	if !mode.Valid() {
		return Calendar{}, fmt.Errorf("%w: %q", ErrUnknownWeekMode, mode)
	}
	return Calendar{startDay: startDay, weekMode: mode, loc: loc}, nil
}

func (c Calendar) WeekSlotsAt(t time.Time) []WeekSlot {
	var slots []WeekSlot
	switch c.weekMode {
	case CalendarWeeks:
		slots = c.weekSlotsByCalendarWeeks(t)
	case FromPeriodStart:
		slots = c.weekSlotsByDays(t)
	default:
		panic(ErrUnknownWeekMode)
	}

	return slots
}

func (c Calendar) weekSlotsByCalendarWeeks(t time.Time) []WeekSlot {
	period := c.PeriodAt(t)
	start := period.Start
	i := 1
	var slots []WeekSlot
	for start.Before(period.NextStart) {
		next := start.AddDate(0, 0, daysToNextMonday(start))
		if next.After(period.NextStart) {
			next = period.NextStart
		}
		slots = append(slots, WeekSlot{
			Index:     i,
			Start:     start,
			NextStart: next,
			Kind:      SlotRegular,
		})
		i++
		start = next
	}

	if len(slots) >= 2 {
		first, last := &slots[0], &slots[len(slots)-1]
		if isStub(*first) && isStub(*last) {
			first.Kind = SlotPairFirst
			last.Kind = SlotPairLast
		}
	}

	return slots
}

func (c Calendar) weekSlotsByDays(t time.Time) []WeekSlot {
	p := c.PeriodAt(t)
	count := countWeekSlots(p)
	var slots []WeekSlot

	for i := range count {
		slots = append(slots, slotAt(p, i))
	}

	return slots
}

func countWeekSlots(p Period) int {
	days := daysBetween(p.Start, p.NextStart)
	weekSlotsAmount := days / daysInWeek
	if days%daysInWeek != 0 {
		weekSlotsAmount++
	}

	return weekSlotsAmount
}

func dateUTC(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func daysBetween(a, b time.Time) int {
	return int(dateUTC(b).Sub(dateUTC(a)) / (24 * time.Hour))
}

func slotAt(p Period, i int) WeekSlot {
	start := addDays(p.Start, i*daysInWeek)
	weekEnd := addDays(start, daysInWeek)
	nextStart := weekEnd
	if p.NextStart.Before(weekEnd) {
		nextStart = p.NextStart
	}
	return WeekSlot{
		Index:     i + 1,
		Start:     start,
		NextStart: nextStart,
		Kind:      SlotRegular,
	}
}

func addDays(t time.Time, d int) time.Time {
	return t.AddDate(0, 0, d)
}

func isStub(s WeekSlot) bool {
	return s.NextStart.Before(addDays(s.Start, daysInWeek))
}

func daysToNextMonday(t time.Time) int {
	sinceMonday := (int(t.Weekday()) + 6) % daysInWeek
	return daysInWeek - sinceMonday
}

func (c Calendar) PeriodAt(t time.Time) Period {
	t = t.In(c.loc)
	year := t.Year()
	month := t.Month()
	thisStart := c.periodStartIn(year, month)
	if !t.Before(thisStart) {
		return Period{
			Start:     thisStart,
			NextStart: c.periodStartIn(year, month+1),
		}
	}
	return Period{
		Start:     c.periodStartIn(year, month-1),
		NextStart: thisStart,
	}
}

func (c Calendar) periodStartIn(year int, month time.Month) time.Time {
	day := min(daysInMonth(year, month), c.startDay)
	return time.Date(year, month, day, 0, 0, 0, 0, c.loc)
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
