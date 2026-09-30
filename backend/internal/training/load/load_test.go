package load

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

func d(day int) time.Time { return time.Date(2026, 9, day, 10, 0, 0, 0, time.UTC) }

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestScore(t *testing.T) {
	tests := []struct {
		name   string
		min    int
		act    training.Activity
		canter float64
		want   float64
	}{
		{"walker", 30, training.ActivityWalker, 0, 9},
		{"groundwork", 30, training.ActivityGroundwork, 0, 12},
		{"lunge", 20, training.ActivityLunge, 0, 14},
		{"hack walk", 60, training.ActivityHack, 0, 36},
		{"hack full canter", 60, training.ActivityHack, 1, 66},
		{"hall half canter", 45, training.ActivityHall, 0.5, 45},
		{"arena", 30, training.ActivityArena, 0, 24},
		{"jumping", 40, training.ActivityJumping, 0, 48},
		{"canter clamped high", 10, training.ActivityHall, 5, 12},
		{"canter clamped low", 10, training.ActivityHall, -1, 8},
		{"zero minutes", 0, training.ActivityHall, 0, 0},
		{"unknown", 30, "dance", 0, 0},
	}
	for _, tt := range tests {
		if got := Score(tt.min, tt.act, tt.canter); !near(got, tt.want) {
			t.Errorf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}

func TestCanterIncreasesIntensity(t *testing.T) {
	for _, a := range []training.Activity{training.ActivityLunge, training.ActivityHack, training.ActivityHall, training.ActivityArena, training.ActivityJumping} {
		if Factor(a, 1) <= Factor(a, 0) {
			t.Errorf("%s: canter should raise intensity", a)
		}
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		l    float64
		want training.Intensity
	}{
		{0, training.IntensityNone}, {0.1, training.IntensityLight}, {29.9, training.IntensityLight},
		{30, training.IntensityMedium}, {60, training.IntensityMedium},
		{60.1, training.IntensityIntense}, {200, training.IntensityIntense},
	}
	for _, tt := range tests {
		if got := Classify(tt.l); got != tt.want {
			t.Errorf("Classify(%v)=%v want %v", tt.l, got, tt.want)
		}
	}
}

func TestDayLoadAndWeekLoad(t *testing.T) {
	sessions := []training.Session{
		{Day: d(28), Load: 20},
		{Day: d(28), Load: 15},
		{Day: d(30), Load: 70},
		{Day: d(27), Load: 99}, // previous week
	}
	if got := DayLoad(sessions, d(28)); !near(got, 35) {
		t.Errorf("DayLoad=%v", got)
	}
	w := WeekLoad(sessions, d(28))
	if w[0].Level != training.IntensityMedium || w[0].Sessions != 2 {
		t.Errorf("day0: %+v", w[0])
	}
	if w[1].Level != training.IntensityNone {
		t.Errorf("day1: %+v", w[1])
	}
	if w[2].Level != training.IntensityIntense {
		t.Errorf("day2: %+v", w[2])
	}
	if !w[6].Day.Equal(time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("day6=%v", w[6].Day)
	}
}

func week(loads ...float64) [7]Segment {
	var sessions []training.Session
	for i, l := range loads {
		if l > 0 {
			sessions = append(sessions, training.Session{Day: d(28 + i), Load: l})
		}
	}
	return WeekLoad(sessions, d(28))
}

func TestAssess(t *testing.T) {
	r := training.DefaultRhythm()
	tests := []struct {
		name string
		w    [7]Segment
		want string
	}{
		{"good mix", week(20, 40, 0, 70, 35), "4 von 5 Einheiten, gute Mischung"},
		{"three intense", week(70, 80, 0, 90), "Schon 3 intensive Tage – morgen eher locker"},
		{"too many", week(20, 40, 20, 40, 20, 40), "6 von 5 Einheiten – genug für diese Woche"},
		{"monotone", week(40, 40, 40, 40), "4 von 5 Einheiten – mehr Abwechslung wäre gut"},
		{"few", week(20, 0, 40), "2 von 5 Einheiten, noch Luft nach oben"},
		{"empty", week(), "0 von 5 Einheiten, noch Luft nach oben"},
	}
	for _, tt := range tests {
		if got := Assess(tt.w, r); got != tt.want {
			t.Errorf("%s: got %q want %q", tt.name, got, tt.want)
		}
	}
	if got := Assess(week(), training.Rhythm{}); !strings.Contains(got, "von 5") {
		t.Errorf("fallback target: %q", got)
	}
}
