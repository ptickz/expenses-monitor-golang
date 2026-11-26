package period

import (
	"errors"
	"testing"
	"time"
)

func date(y int, m time.Month, d int, loc *time.Location) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func mustCalendar(t *testing.T, startDay int, mode WeekMode, loc *time.Location) Calendar {
	t.Helper()
	c, err := New(startDay, mode, loc)
	if err != nil {
		t.Fatalf("New(%d, %q): %v", startDay, mode, err)
	}
	return c
}

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location %q: %v", name, err)
	}
	return loc
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		startDay int
		mode     WeekMode
		loc      *time.Location
		wantErr  bool
		unknown  bool
	}{
		{name: "valid", startDay: 15, mode: FromPeriodStart, loc: time.UTC},
		{name: "day 1", startDay: 1, mode: CalendarWeeks, loc: time.UTC},
		{name: "day 31", startDay: 31, mode: CalendarWeeks, loc: time.UTC},
		{name: "day 0", startDay: 0, mode: FromPeriodStart, loc: time.UTC, wantErr: true},
		{name: "day 32", startDay: 32, mode: FromPeriodStart, loc: time.UTC, wantErr: true},
		{name: "nil location", startDay: 15, mode: FromPeriodStart, loc: nil, wantErr: true},
		{name: "unknown mode", startDay: 15, mode: "BIWEEKLY", loc: time.UTC, wantErr: true, unknown: true},
		{name: "empty mode", startDay: 15, mode: "", loc: time.UTC, wantErr: true, unknown: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.startDay, tt.mode, tt.loc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if errors.Is(err, ErrUnknownWeekMode) != tt.unknown {
				t.Errorf("errors.Is(err, ErrUnknownWeekMode) = %v, want %v", !tt.unknown, tt.unknown)
			}
		})
	}
}

func TestPeriodAt(t *testing.T) {
	plus4 := time.FixedZone("+04", 4*60*60)

	tests := []struct {
		name      string
		startDay  int
		loc       *time.Location
		at        time.Time
		wantStart time.Time
		wantNext  time.Time
	}{
		{
			name: "middle of period", startDay: 15, loc: time.UTC,
			at:        time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC),
			wantStart: date(2026, 11, 15, time.UTC), wantNext: date(2026, 12, 15, time.UTC),
		},
		{
			name: "before start day belongs to previous period", startDay: 15, loc: time.UTC,
			at:        time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC),
			wantStart: date(2026, 10, 15, time.UTC), wantNext: date(2026, 11, 15, time.UTC),
		},
		{
			name: "exactly at start is inclusive", startDay: 15, loc: time.UTC,
			at:        date(2026, 11, 15, time.UTC),
			wantStart: date(2026, 11, 15, time.UTC), wantNext: date(2026, 12, 15, time.UTC),
		},
		{
			name: "last nanosecond before next start", startDay: 15, loc: time.UTC,
			at:        date(2026, 12, 15, time.UTC).Add(-time.Nanosecond),
			wantStart: date(2026, 11, 15, time.UTC), wantNext: date(2026, 12, 15, time.UTC),
		},
		{
			name: "year wrap forward", startDay: 15, loc: time.UTC,
			at:        date(2026, 12, 20, time.UTC),
			wantStart: date(2026, 12, 15, time.UTC), wantNext: date(2027, 1, 15, time.UTC),
		},
		{
			name: "year wrap backward", startDay: 15, loc: time.UTC,
			at:        date(2026, 1, 5, time.UTC),
			wantStart: date(2025, 12, 15, time.UTC), wantNext: date(2026, 1, 15, time.UTC),
		},
		{
			name: "day 31 clamps to end of february", startDay: 31, loc: time.UTC,
			at:        date(2026, 2, 10, time.UTC),
			wantStart: date(2026, 1, 31, time.UTC), wantNext: date(2026, 2, 28, time.UTC),
		},
		{
			name: "day 31 after clamped february start", startDay: 31, loc: time.UTC,
			at:        date(2026, 3, 1, time.UTC),
			wantStart: date(2026, 2, 28, time.UTC), wantNext: date(2026, 3, 31, time.UTC),
		},
		{
			name: "day 30 in leap february", startDay: 30, loc: time.UTC,
			at:        date(2028, 3, 10, time.UTC),
			wantStart: date(2028, 2, 29, time.UTC), wantNext: date(2028, 3, 30, time.UTC),
		},
		{
			name: "instant is read in calendar location", startDay: 15, loc: plus4,
			at:        time.Date(2026, 11, 14, 21, 0, 0, 0, time.UTC),
			wantStart: date(2026, 11, 15, plus4), wantNext: date(2026, 12, 15, plus4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mustCalendar(t, tt.startDay, FromPeriodStart, tt.loc)
			p := c.PeriodAt(tt.at)
			if !p.Start.Equal(tt.wantStart) || !p.NextStart.Equal(tt.wantNext) {
				t.Errorf("PeriodAt(%v) = [%v, %v), want [%v, %v)",
					tt.at, p.Start, p.NextStart, tt.wantStart, tt.wantNext)
			}
		})
	}
}

type wantSlot struct {
	start, next time.Time
	kind        SlotKind
}

func checkSlots(t *testing.T, got []WeekSlot, want []wantSlot) {
	t.Helper()
	if len(got) != len(want) {
		for _, s := range got {
			t.Logf("got #%d [%v, %v) kind=%d", s.Index, s.Start, s.NextStart, s.Kind)
		}
		t.Fatalf("got %d slots, want %d", len(got), len(want))
	}
	for i, w := range want {
		s := got[i]
		if s.Index != i+1 {
			t.Errorf("slot %d: Index = %d, want %d", i, s.Index, i+1)
		}
		if !s.Start.Equal(w.start) || !s.NextStart.Equal(w.next) {
			t.Errorf("slot %d: [%v, %v), want [%v, %v)", i, s.Start, s.NextStart, w.start, w.next)
		}
		if s.Kind != w.kind {
			t.Errorf("slot %d: Kind = %d, want %d", i, s.Kind, w.kind)
		}
	}
}

func TestWeekSlotsFromPeriodStart(t *testing.T) {
	u := time.UTC
	tests := []struct {
		name     string
		startDay int
		at       time.Time
		want     []wantSlot
	}{
		{
			name: "30-day period ends with a short 2-day slot", startDay: 15,
			at: date(2026, 11, 23, u),
			want: []wantSlot{
				{date(2026, 11, 15, u), date(2026, 11, 22, u), SlotRegular},
				{date(2026, 11, 22, u), date(2026, 11, 29, u), SlotRegular},
				{date(2026, 11, 29, u), date(2026, 12, 6, u), SlotRegular},
				{date(2026, 12, 6, u), date(2026, 12, 13, u), SlotRegular},
				{date(2026, 12, 13, u), date(2026, 12, 15, u), SlotRegular},
			},
		},
		{
			name: "28-day period splits into exactly 4 weeks", startDay: 1,
			at: date(2026, 2, 10, u),
			want: []wantSlot{
				{date(2026, 2, 1, u), date(2026, 2, 8, u), SlotRegular},
				{date(2026, 2, 8, u), date(2026, 2, 15, u), SlotRegular},
				{date(2026, 2, 15, u), date(2026, 2, 22, u), SlotRegular},
				{date(2026, 2, 22, u), date(2026, 3, 1, u), SlotRegular},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mustCalendar(t, tt.startDay, FromPeriodStart, u)
			checkSlots(t, c.WeekSlotsAt(tt.at), tt.want)
		})
	}
}

func TestWeekSlotsCalendarWeeks(t *testing.T) {
	u := time.UTC
	tests := []struct {
		name     string
		startDay int
		at       time.Time
		want     []wantSlot
	}{
		{
			name: "stubs on both ends are paired", startDay: 15,
			at: date(2026, 11, 23, u),
			want: []wantSlot{
				{date(2026, 11, 15, u), date(2026, 11, 16, u), SlotPairFirst},
				{date(2026, 11, 16, u), date(2026, 11, 23, u), SlotRegular},
				{date(2026, 11, 23, u), date(2026, 11, 30, u), SlotRegular},
				{date(2026, 11, 30, u), date(2026, 12, 7, u), SlotRegular},
				{date(2026, 12, 7, u), date(2026, 12, 14, u), SlotRegular},
				{date(2026, 12, 14, u), date(2026, 12, 15, u), SlotPairLast},
			},
		},
		{
			name: "period starting on monday has no leading stub", startDay: 1,
			at: date(2026, 6, 10, u),
			want: []wantSlot{
				{date(2026, 6, 1, u), date(2026, 6, 8, u), SlotRegular},
				{date(2026, 6, 8, u), date(2026, 6, 15, u), SlotRegular},
				{date(2026, 6, 15, u), date(2026, 6, 22, u), SlotRegular},
				{date(2026, 6, 22, u), date(2026, 6, 29, u), SlotRegular},
				{date(2026, 6, 29, u), date(2026, 7, 1, u), SlotRegular},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mustCalendar(t, tt.startDay, CalendarWeeks, u)
			checkSlots(t, c.WeekSlotsAt(tt.at), tt.want)
		})
	}
}

func TestWeekSlotsCoverPeriod(t *testing.T) {
	locs := []*time.Location{time.UTC, mustLoad(t, "Europe/Berlin")}
	modes := []WeekMode{FromPeriodStart, CalendarWeeks}
	startDays := []int{1, 15, 28, 29, 30, 31}

	for _, loc := range locs {
		for _, mode := range modes {
			for _, sd := range startDays {
				c := mustCalendar(t, sd, mode, loc)
				for day := date(2026, 1, 1, loc); day.Year() < 2029; day = day.AddDate(0, 0, 1) {
					at := day.Add(13 * time.Hour)
					p := c.PeriodAt(at)
					slots := c.WeekSlotsAt(at)

					if at.Before(p.Start) || !at.Before(p.NextStart) {
						t.Fatalf("%s %s day=%d at=%v: outside its period [%v, %v)", loc, mode, sd, at, p.Start, p.NextStart)
					}
					if len(slots) == 0 {
						t.Fatalf("%s %s day=%d at=%v: no slots", loc, mode, sd, at)
					}
					if !slots[0].Start.Equal(p.Start) || !slots[len(slots)-1].NextStart.Equal(p.NextStart) {
						t.Fatalf("%s %s day=%d at=%v: slots do not span period", loc, mode, sd, at)
					}
					for i, s := range slots {
						if s.Index != i+1 {
							t.Fatalf("%s %s day=%d at=%v: slot %d Index = %d", loc, mode, sd, at, i, s.Index)
						}
						if !s.Start.Before(s.NextStart) {
							t.Fatalf("%s %s day=%d at=%v: empty slot %d [%v, %v)", loc, mode, sd, at, i, s.Start, s.NextStart)
						}
						if d := daysBetween(s.Start, s.NextStart); d > daysInWeek {
							t.Fatalf("%s %s day=%d at=%v: slot %d is %d days", loc, mode, sd, at, i, d)
						}
						if s.Start.Hour() != 0 || s.Start.Minute() != 0 {
							t.Fatalf("%s %s day=%d at=%v: slot %d starts at %v, not midnight", loc, mode, sd, at, i, s.Start)
						}
						if i > 0 && !slots[i-1].NextStart.Equal(s.Start) {
							t.Fatalf("%s %s day=%d at=%v: gap or overlap before slot %d", loc, mode, sd, at, i)
						}
					}
				}
			}
		}
	}
}

func TestDaysBetween(t *testing.T) {
	berlin := mustLoad(t, "Europe/Berlin")
	tests := []struct {
		name string
		a, b time.Time
		want int
	}{
		{"same day", date(2026, 11, 15, time.UTC), date(2026, 11, 15, time.UTC), 0},
		{"15 Nov to 14 Dec", date(2026, 11, 15, time.UTC), date(2026, 12, 14, time.UTC), 29},
		{"ignores time of day", time.Date(2026, 11, 15, 23, 59, 0, 0, time.UTC), time.Date(2026, 11, 16, 0, 1, 0, 0, time.UTC), 1},
		{"across spring DST (23h day)", date(2026, 3, 28, berlin), date(2026, 3, 30, berlin), 2},
		{"across autumn DST (25h day)", date(2026, 10, 24, berlin), date(2026, 10, 26, berlin), 2},
		{"leap year february", date(2028, 2, 1, time.UTC), date(2028, 3, 1, time.UTC), 29},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := daysBetween(tt.a, tt.b); got != tt.want {
				t.Errorf("daysBetween = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValidateStartDay(t *testing.T) {
	tests := []struct {
		day  int
		want bool
	}{
		{day: -1, want: false},
		{day: 0, want: false},
		{day: 1, want: true},
		{day: 15, want: true},
		{day: 31, want: true},
		{day: 32, want: false},
	}
	for _, tt := range tests {
		if got := ValidateStartDay(tt.day); got != tt.want {
			t.Errorf("ValidateStartDay(%d) = %t, want %t", tt.day, got, tt.want)
		}
	}
}
