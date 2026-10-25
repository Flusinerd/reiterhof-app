package recommend

import (
	"sort"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// Periodisation and return to work (docs/domains/training-planning.md, "Rules with numbers").
//
// Blocks of four weeks from the profile's cycle_start: weeks 1 to 3 build, week 4 is the
// Entlastungswoche (no demanding unit, the load capped at 0.8 times the average of the three
// weeks before). The block starts again with week 1 when one of the weeks of the block so far
// had fewer than rhythm.sessions_min sessions: a horse that was not trained regularly gets no
// relief week.
//
// After a break (no session for RestartGapDays or more) the first week only has light and
// recovery units, the second and third at most normal ones, demanding units come back from the
// fourth week on.

// Block constants.
const (
	BlockWeeks   = 4   // weeks of a block
	DeloadFactor = 0.8 // load cap of the Entlastungswoche against the average of the three weeks before
	DeloadWeeks  = 3   // weeks the cap is measured over
)

// Return to work constants.
const (
	RestartGapDays = 14 // days without a session that make a break
	RestartWeeks   = 3  // weeks with restrictions after a break
)

// Block is the position of a week in the four-week training blocks. The zero value (Week 0)
// means there are no blocks (no cycle_start, or the week is before it).
type Block struct {
	Week   int  // 1 to 4
	Deload bool // week 4: the Entlastungswoche
}

// Active reports whether the week belongs to a block.
func (b Block) Active() bool { return b.Week > 0 }

// BlockFor returns the block position of the week starting on the Monday target. cycleStart is the
// profile's cycle_start (nil = no blocks), today the stable-local day (only weeks that ended
// before the week of today count as complete, the week in progress is never weak),
// sessionDays has one entry per session of the horse (any order, at least from cycleStart on) and
// sessionsMin is rhythm.sessions_min.
//
// Walking from cycleStart week by week: a block that has run four weeks starts again, and a block
// restarts when one of its weeks so far had fewer than sessionsMin sessions.
func BlockFor(cycleStart *time.Time, target, today time.Time, sessionDays []time.Time, sessionsMin int) Block {
	if cycleStart == nil {
		return Block{}
	}
	start := mondayOfDay(*cycleStart)
	target = mondayOfDay(target)
	if target.Before(start) {
		return Block{}
	}
	thisWeek := mondayOfDay(today)
	counts := map[time.Time]int{}
	for _, d := range sessionDays {
		counts[mondayOfDay(d)]++
	}
	weak := func(w time.Time) bool {
		return sessionsMin > 0 && w.Before(thisWeek) && counts[w] < sessionsMin
	}
	blockStart := start
	for w := start; !w.After(target); w = w.AddDate(0, 0, 7) {
		pos := training.DaysBetween(blockStart, w) / 7
		switch {
		case pos >= BlockWeeks:
			blockStart = w
		case pos > 0:
			for b := blockStart; b.Before(w); b = b.AddDate(0, 0, 7) {
				if weak(b) {
					blockStart = w
					break
				}
			}
		}
	}
	pos := training.DaysBetween(blockStart, target) / 7
	return Block{Week: pos + 1, Deload: pos == BlockWeeks-1}
}

// mondayOfDay returns the Monday of the week of t as a UTC date.
func mondayOfDay(t time.Time) time.Time {
	d := training.Day(t)
	return d.AddDate(0, 0, -training.WeekdayIndex(d))
}

// RestartAnchor returns the first day of the horse's return to work: today when the last session
// before today is RestartGapDays or more days ago, else the first session after the latest gap of
// that length, nil when there was no break (or no session at all: a horse without history is
// not returning). sessionDays are the days of the horse's sessions, any order and duplicates
// allowed; it must include the last session before the days it covers, so that the gap before
// the first one is seen. Days after today are ignored.
func RestartAnchor(sessionDays []time.Time, today time.Time) *time.Time {
	today = training.Day(today)
	var days []time.Time
	for _, d := range sessionDays {
		if d = training.Day(d); !d.After(today) {
			days = append(days, d)
		}
	}
	if len(days) == 0 {
		return nil
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	var anchor *time.Time
	for i := 1; i < len(days); i++ {
		if training.DaysBetween(days[i-1], days[i]) >= RestartGapDays {
			a := days[i]
			anchor = &a
		}
	}
	if training.DaysBetween(days[len(days)-1], today) >= RestartGapDays {
		anchor = &today
	}
	return anchor
}

// RestartWeek is the week of the return to work that day is in: 1 to RestartWeeks, 0 when the
// horse is not returning (no anchor, a day before it, or the restart weeks are over).
func RestartWeek(anchor *time.Time, day time.Time) int {
	if anchor == nil {
		return 0
	}
	d := training.DaysBetween(*anchor, day)
	if d < 0 {
		return 0
	}
	if w := 1 + d/7; w <= RestartWeeks {
		return w
	}
	return 0
}

// DeloadLoadLimit is the load cap of an Entlastungswoche: DeloadFactor times the average weekly
// load of the DeloadWeeks weeks before the week starting on weekStart, or false when those weeks
// have fewer than MinHistorySessions sessions.
func DeloadLoadLimit(recent []training.Session, weekStart time.Time) (float64, bool) {
	var sum float64
	n := 0
	for _, s := range recent {
		if d := training.DaysBetween(weekStart, s.Day); d >= -7*DeloadWeeks && d < 0 {
			sum += s.Load
			n++
		}
	}
	if n < MinHistorySessions {
		return 0, false
	}
	return sum / DeloadWeeks * DeloadFactor, true
}
