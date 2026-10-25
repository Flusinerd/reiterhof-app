package recommend

import (
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// Mondays of the block tests: w(0) is 2026-08-31.
var blockStart = time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

func w(n int) time.Time { return blockStart.AddDate(0, 0, 7*n) }

// weeks returns the session days of weeks with the given session counts, starting at w(0).
func weeks(counts ...int) []time.Time {
	var days []time.Time
	for i, n := range counts {
		for j := 0; j < n; j++ {
			days = append(days, w(i).AddDate(0, 0, j%7))
		}
	}
	return days
}

func TestBlockFor(t *testing.T) {
	long := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC) // every week below is over
	start := blockStart
	cases := []struct {
		name     string
		cycle    *time.Time
		target   int // week index
		today    time.Time
		sessions []time.Time
		min      int
		want     Block
	}{
		{"week 1", &start, 0, long, weeks(4, 4, 4, 4, 4, 4), 3, Block{1, false}},
		{"week 2", &start, 1, long, weeks(4, 4, 4, 4, 4, 4), 3, Block{2, false}},
		{"week 3", &start, 2, long, weeks(4, 4, 4, 4, 4, 4), 3, Block{3, false}},
		{"week 4 is the deload week", &start, 3, long, weeks(4, 4, 4, 4, 4, 4), 3, Block{4, true}},
		{"the next block starts", &start, 4, long, weeks(4, 4, 4, 4, 4, 4), 3, Block{1, false}},
		{"and goes on", &start, 5, long, weeks(4, 4, 4, 4, 4, 4), 3, Block{2, false}},
		{"a second deload week", &start, 7, long, weeks(4, 4, 4, 4, 4, 4, 4, 4), 3, Block{4, true}},
		// A weak week resets the block: the week after it is week 1 again.
		{"week 2 weak: week 3 starts again", &start, 2, long, weeks(4, 2, 4, 4, 4), 3, Block{1, false}},
		{"and counts on from there", &start, 3, long, weeks(4, 2, 4, 4, 4), 3, Block{2, false}},
		{"no deload for a block that was not trained", &start, 3, long, weeks(4, 4, 2, 4, 4), 3, Block{1, false}},
		{"a weak first week resets too", &start, 1, long, weeks(2, 4, 4), 3, Block{1, false}},
		{"a weak deload week does not touch the next block", &start, 4, long, weeks(4, 4, 4, 1, 4, 4), 3, Block{1, false}},
		{"nor the weeks after it", &start, 5, long, weeks(4, 4, 4, 1, 4, 4), 3, Block{2, false}},
		{"exactly sessions_min is enough", &start, 3, long, weeks(3, 3, 3), 3, Block{4, true}},
		{"no minimum, no reset", &start, 3, long, weeks(0, 0, 0), 0, Block{4, true}},
		// The week in progress and the weeks ahead are never weak.
		{"this week is not over", &start, 3, w(3).AddDate(0, 0, 2), weeks(4, 4, 4, 1), 3, Block{4, true}},
		{"planning next week", &start, 3, w(2).AddDate(0, 0, 3), weeks(4, 4, 1), 3, Block{4, true}},
		{"planning the week after next", &start, 4, w(2).AddDate(0, 0, 3), weeks(4, 4, 1), 3, Block{1, false}},
		// No block without a start, before it, or with a start in the middle of a week.
		{"no cycle start", nil, 0, long, weeks(4), 3, Block{}},
		{"before the cycle start", &start, -1, long, weeks(4), 3, Block{}},
		{"cycle start on a Wednesday", func() *time.Time { d := w(0).AddDate(0, 0, 2); return &d }(), 1, long, weeks(4, 4), 3, Block{2, false}},
	}
	for _, c := range cases {
		target := w(c.target)
		if got := BlockFor(c.cycle, target, c.today, c.sessions, c.min); got != c.want {
			t.Errorf("%s: BlockFor = %+v, want %+v", c.name, got, c.want)
		}
		// Any day of the week gives the same answer as its Monday.
		if got := BlockFor(c.cycle, target.AddDate(0, 0, 4), c.today, c.sessions, c.min); got != c.want {
			t.Errorf("%s (Friday as target): BlockFor = %+v, want %+v", c.name, got, c.want)
		}
	}
	if (Block{}).Active() || !(Block{Week: 1}).Active() {
		t.Error("Block.Active")
	}
}

func d(offset int) time.Time { return today.AddDate(0, 0, offset) }

func TestRestartAnchor(t *testing.T) {
	cases := []struct {
		name string
		days []time.Time
		want int // day offset of the anchor against today, 99 for none
	}{
		{"no sessions at all: not returning", nil, 99},
		{"trained yesterday", []time.Time{d(-1)}, 99},
		{"13 days ago is no break", []time.Time{d(-13)}, 99},
		{"14 days ago is a break: back today", []time.Time{d(-14)}, 0},
		{"a long break", []time.Time{d(-60), d(-40)}, 0},
		{"a break before a session: that session is the anchor", []time.Time{d(-40), d(-20), d(-18), d(-10), d(-3)}, -20},
		{"a gap of 13 days is none", []time.Time{d(-30), d(-17), d(-5)}, 99},
		{"a gap of 14 days is a break", []time.Time{d(-30), d(-16), d(-5)}, -16},
		{"the latest gap wins", []time.Time{d(-90), d(-70), d(-30), d(-20), d(-12)}, -30},
		{"a break right before the last session", []time.Time{d(-90), d(-70), d(-30), d(-20), d(-2)}, -2},
		{"duplicates and any order", []time.Time{d(-10), d(-40), d(-10), d(-40), d(-26)}, -10},
		{"sessions after today are ignored", []time.Time{d(-30), d(3)}, 0},
		{"a session today ends the gap before it", []time.Time{d(-30), d(0)}, 0},
		{"time of day does not matter", []time.Time{d(-15).Add(18 * time.Hour)}, 0},
	}
	for _, c := range cases {
		got := RestartAnchor(c.days, today)
		switch {
		case c.want == 99 && got != nil:
			t.Errorf("%s: anchor %v, want none", c.name, got.Format("2006-01-02"))
		case c.want != 99 && got == nil:
			t.Errorf("%s: no anchor, want today%+d", c.name, c.want)
		case c.want != 99 && training.DaysBetween(today, *got) != c.want:
			t.Errorf("%s: anchor today%+d, want today%+d", c.name, training.DaysBetween(today, *got), c.want)
		}
	}
}

func TestRestartWeek(t *testing.T) {
	anchor := d(0)
	cases := map[int]int{-1: 0, 0: 1, 6: 1, 7: 2, 13: 2, 14: 3, 20: 3, 21: 0, 40: 0}
	for off, want := range cases {
		if got := RestartWeek(&anchor, d(off)); got != want {
			t.Errorf("day %+d after the anchor: week %d, want %d", off, got, want)
		}
	}
	if RestartWeek(nil, d(3)) != 0 {
		t.Error("no anchor, no restart")
	}
}

func TestDeloadLoadLimit(t *testing.T) {
	monday := d(-2) // Monday of the test week
	sess := func(off int, load float64) training.Session { return training.Session{Day: d(off), Load: load} }
	// Three weeks before: 300 load in four sessions, average 100 per week, cap 80.
	recent := []training.Session{sess(-3, 100), sess(-10, 100), sess(-15, 50), sess(-23, 50)}
	got, ok := DeloadLoadLimit(recent, monday)
	if !ok || got != 80 {
		t.Errorf("limit = %v, %v, want 80", got, ok)
	}
	// 21 days before the Monday still count, 22 do not; sessions of the week itself never do.
	edge := []training.Session{sess(-23, 90), sess(-24, 90), sess(-24, 90), sess(-2, 500), sess(-1, 500)}
	if got, ok := DeloadLoadLimit(edge, monday); ok {
		t.Errorf("only %v loads in the window? limit %v", edge, got)
	}
	edge = append(edge, sess(-23, 30), sess(-23, 30))
	if got, ok := DeloadLoadLimit(edge, monday); !ok || got != (90+30+30)/3.0*0.8 {
		t.Errorf("limit = %v, %v", got, ok)
	}
	if _, ok := DeloadLoadLimit(recent[:2], monday); ok {
		t.Error("two sessions are too little history for a limit")
	}
	if got, _ := DeloadLoadLimit(nil, monday); got != 0 {
		t.Errorf("limit without history = %v", got)
	}
}
