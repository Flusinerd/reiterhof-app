// Package load computes the training load score and the week bar (JAN-62).
//
// Score = minutes x intensity(activity, canterShare). The intensity factor per
// activity is a base value plus a bonus that scales linearly with the canter
// share (0..1):
//
//	activity    base  bonus at 100% canter  range
//	walker      0.3   -                     0.3
//	groundwork  0.4   -                     0.4
//	lunge       0.7   +0.2                  0.7-0.9
//	hack        0.6   +0.5                  0.6-1.1
//	hall/arena  0.8   +0.4                  0.8-1.2
//	jumping     1.2   +0.2                  1.2-1.4
//
// Day classification (constants, to be fine-tuned later): light < 30,
// medium 30-60, intense > 60. A day without sessions is IntensityNone.
package load

import (
	"fmt"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// Day classification thresholds on the summed score of a day.
const (
	LightBelow   = 30.0 // load < 30: light
	IntenseAbove = 60.0 // load > 60: intense; 30..60 inclusive: medium
)

type intensity struct{ base, canterBonus float64 }

var table = map[training.Activity]intensity{
	training.ActivityWalker:     {0.3, 0},
	training.ActivityGroundwork: {0.4, 0},
	training.ActivityLunge:      {0.7, 0.2},
	training.ActivityHack:       {0.6, 0.5},
	training.ActivityHall:       {0.8, 0.4},
	training.ActivityArena:      {0.8, 0.4},
	training.ActivityJumping:    {1.2, 0.2},
}

// Factor returns the intensity factor; unknown activities yield 0.
// canterShare is clamped to 0..1.
func Factor(a training.Activity, canterShare float64) float64 {
	t, ok := table[a]
	if !ok {
		return 0
	}
	if canterShare < 0 {
		canterShare = 0
	} else if canterShare > 1 {
		canterShare = 1
	}
	return t.base + t.canterBonus*canterShare
}

// Score is the load of one session: minutes x intensity.
func Score(minutes int, a training.Activity, canterShare float64) float64 {
	if minutes <= 0 {
		return 0
	}
	return float64(minutes) * Factor(a, canterShare)
}

// Classify maps a day's summed load to a level.
func Classify(l float64) training.Intensity {
	switch {
	case l <= 0:
		return training.IntensityNone
	case l < LightBelow:
		return training.IntensityLight
	case l <= IntenseAbove:
		return training.IntensityMedium
	}
	return training.IntensityIntense
}

// DayLoad sums Session.Load of all sessions on the calendar day.
func DayLoad(sessions []training.Session, day time.Time) float64 {
	var sum float64
	for _, s := range sessions {
		if training.DaysBetween(s.Day, day) == 0 {
			sum += s.Load
		}
	}
	return sum
}

// Segment is one bar of the week view.
type Segment struct {
	Day      time.Time
	Load     float64
	Level    training.Intensity
	Sessions int
}

// WeekLoad returns 7 segments, weekStart .. weekStart+6.
func WeekLoad(sessions []training.Session, weekStart time.Time) [7]Segment {
	var w [7]Segment
	start := training.Day(weekStart)
	for i := range w {
		day := start.AddDate(0, 0, i)
		w[i].Day = day
		for _, s := range sessions {
			if training.DaysBetween(s.Day, day) == 0 {
				w[i].Load += s.Load
				w[i].Sessions++
			}
		}
		w[i].Level = Classify(w[i].Load)
	}
	return w
}

// Assess returns one German sentence judging the week against the rhythm.
// Rules, first match wins:
//  1. three or more intense days: "Schon N intensive Tage - morgen eher locker"
//  2. more sessions than rhythm.SessionsMax: enough for this week
//  3. at least SessionsMin sessions and two or more different levels: good mix
//  4. at least SessionsMin sessions, all the same level: ask for variety
//  5. otherwise: progress "X von Y Einheiten"
//
// Y is SessionsMax (falls back to 5 when the rhythm has none).
func Assess(week [7]Segment, r training.Rhythm) string {
	target := r.SessionsMax
	if target <= 0 {
		target = 5
	}
	minS := r.SessionsMin
	done, intense := 0, 0
	levels := map[training.Intensity]bool{}
	for _, s := range week {
		done += s.Sessions
		if s.Level == training.IntensityIntense {
			intense++
		}
		if s.Level != training.IntensityNone {
			levels[s.Level] = true
		}
	}
	switch {
	case intense >= 3:
		return fmt.Sprintf("Schon %d intensive Tage – morgen eher locker", intense)
	case done > target:
		return fmt.Sprintf("%d von %d Einheiten – das reicht für diese Woche", done, target)
	case done >= minS && done > 0 && len(levels) >= 2:
		return fmt.Sprintf("%d von %d Einheiten, gute Mischung", done, target)
	case done >= minS && done > 0:
		return fmt.Sprintf("%d von %d Einheiten, etwas mehr Abwechslung täte gut", done, target)
	}
	return fmt.Sprintf("%d von %d Einheiten, noch Luft nach oben", done, target)
}
