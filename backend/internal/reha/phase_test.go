package reha

import (
	"strings"
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// specPlan is the example of the specification.
func specPlan() []Phase {
	return []Phase{
		{Name: "Boxenruhe", Days: 5, Activity: "rest"},
		{Name: "Schritt führen", Days: 9, Activity: "groundwork", MinMinutes: 10, MaxMinutes: 20, Conditions: "nur Boden fest"},
		{Name: "Schritt reiten", Days: 14, Activity: "hall", MinMinutes: 20, MaxMinutes: 40},
		{Name: "Trab aufbauen", Days: 14, Activity: "hall", MinMinutes: 2, MaxMinutes: 15},
	}
}

func TestRamp(t *testing.T) {
	// 10 -> 20 over 9 days: rises by 1.25 per day, rounded half up.
	want := []int{10, 11, 13, 14, 15, 16, 18, 19, 20}
	for i, w := range want {
		if got := Ramp(10, 20, i, 9); got != w {
			t.Errorf("day %d: got %d, want %d", i, got, w)
		}
	}
	cases := []struct{ min, max, i, n, want int }{
		{15, 20, 0, 1, 20},  // single day allows max
		{15, 15, 3, 7, 15},  // flat
		{2, 15, 13, 14, 15}, // last day is max
		{2, 15, 0, 14, 2},
		{20, 40, 6, 14, 29}, // 20 + 20*6/13 = 29.23
		{10, 20, -1, 9, 10}, // out of range is clamped
		{10, 20, 99, 9, 20},
	}
	for _, c := range cases {
		if got := Ramp(c.min, c.max, c.i, c.n); got != c.want {
			t.Errorf("Ramp(%d,%d,%d,%d) = %d, want %d", c.min, c.max, c.i, c.n, got, c.want)
		}
	}
}

func TestTimeline(t *testing.T) {
	start := day(2026, 3, 20)
	tl := Timeline(start, specPlan())
	wantStart := []time.Time{day(2026, 3, 20), day(2026, 3, 25), day(2026, 4, 3), day(2026, 4, 17)}
	wantEnd := []time.Time{day(2026, 3, 24), day(2026, 4, 2), day(2026, 4, 16), day(2026, 4, 30)}
	for i, s := range tl {
		if !s.Start.Equal(wantStart[i]) || !s.End.Equal(wantEnd[i]) {
			t.Errorf("phase %d: %s..%s, want %s..%s", i, s.Start.Format("2006-01-02"), s.End.Format("2006-01-02"),
				wantStart[i].Format("2006-01-02"), wantEnd[i].Format("2006-01-02"))
		}
	}
	if got := EndDate(start, specPlan()); !got.Equal(day(2026, 4, 30)) {
		t.Errorf("EndDate = %s", got)
	}
	if TotalDays(specPlan()) != 42 {
		t.Errorf("TotalDays = %d", TotalDays(specPlan()))
	}
}

// The plan crosses the DST change of Europe/Berlin (2026-03-29 02:00 -> 03:00): dates are calendar
// dates, so a phase never loses or gains a day.
func TestLocateAcrossDST(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skip("no tzdata")
	}
	// "now" values in Berlin wall-clock time, converted to calendar days like the API does.
	start := day(2026, 3, 27)
	phases := []Phase{{Name: "A", Days: 3, Activity: "walker", MinMinutes: 10, MaxMinutes: 30}, {Name: "B", Days: 2, Activity: "lunge", MinMinutes: 10, MaxMinutes: 10}}
	cases := []struct {
		now       time.Time
		phase     string
		dayInPlan int
		minutes   int
	}{
		{time.Date(2026, 3, 27, 23, 59, 0, 0, berlin), "A", 0, 10},
		{time.Date(2026, 3, 28, 0, 0, 0, 0, berlin), "A", 1, 20},
		{time.Date(2026, 3, 29, 1, 59, 0, 0, berlin), "A", 2, 30},  // last minute before the jump
		{time.Date(2026, 3, 29, 3, 0, 0, 0, berlin), "A", 2, 30},   // first minute after (23 hour day)
		{time.Date(2026, 3, 30, 0, 0, 0, 0, berlin), "B", 3, 10},   // 47 hours after 03-28 00:00
		{time.Date(2026, 3, 31, 23, 59, 0, 0, berlin), "B", 4, 10}, // last day
	}
	for _, c := range cases {
		pos, ok := Locate(start, phases, c.now.In(berlin))
		if !ok || pos.Slot.Phase.Name != c.phase || pos.DayInPlan != c.dayInPlan || pos.Minutes != c.minutes {
			t.Errorf("%s: got %+v ok=%v, want %s day %d minutes %d", c.now.Format(time.RFC3339), pos, ok, c.phase, c.dayInPlan, c.minutes)
		}
	}
	if _, ok := Locate(start, phases, time.Date(2026, 4, 1, 0, 0, 0, 0, berlin)); ok {
		t.Error("after the last phase must not locate")
	}
	if _, ok := Locate(start, phases, time.Date(2026, 3, 26, 23, 0, 0, 0, berlin)); ok {
		t.Error("before the start must not locate")
	}

	// The autumn change (25 hour day) as well.
	start = day(2026, 10, 24)
	phases = []Phase{{Name: "A", Days: 2, Activity: "walker", MinMinutes: 5, MaxMinutes: 10}, {Name: "B", Days: 1, Activity: "walker", MinMinutes: 5, MaxMinutes: 5}}
	pos, ok := Locate(start, phases, time.Date(2026, 10, 26, 0, 30, 0, 0, berlin))
	if !ok || pos.Slot.Phase.Name != "B" {
		t.Errorf("autumn: got %+v ok=%v, want B", pos, ok)
	}
}

func TestSpecPlanToday(t *testing.T) {
	start := day(2026, 3, 20)
	cases := []struct {
		date    time.Time
		name    string
		minutes int
		rest    bool
		dayIn   int
	}{
		{day(2026, 3, 20), "Boxenruhe", 0, true, 1},
		{day(2026, 3, 24), "Boxenruhe", 0, true, 5},
		{day(2026, 3, 25), "Schritt führen", 10, false, 1},
		{day(2026, 3, 28), "Schritt führen", 14, false, 4}, // 10 + 10*3/8 = 13.75
		{day(2026, 4, 2), "Schritt führen", 20, false, 9},
		{day(2026, 4, 3), "Schritt reiten", 20, false, 1},
		{day(2026, 4, 30), "Trab aufbauen", 15, false, 14},
	}
	for _, c := range cases {
		a := AllowedOn(start, specPlan(), c.date)
		if a == nil {
			t.Fatalf("%s: no allowed unit", c.date.Format("2006-01-02"))
		}
		if a.Phase.Name != c.name || a.Minutes != c.minutes || a.Phase.Rest() != c.rest || a.DayInPhase != c.dayIn || a.Phases != 4 {
			t.Errorf("%s: got %s %d min rest=%v day %d", c.date.Format("2006-01-02"), a.Phase.Name, a.Minutes, a.Phase.Rest(), a.DayInPhase)
		}
	}
	if AllowedOn(start, specPlan(), day(2026, 5, 1)) != nil || AllowedOn(start, specPlan(), day(2026, 3, 19)) != nil {
		t.Error("outside the plan there is no allowed unit")
	}
}

func TestRuleText(t *testing.T) {
	a := AllowedOn(day(2026, 3, 20), specPlan(), day(2026, 3, 28))
	if got, want := a.RuleText(), "Reha: Schritt führen 14 min, nur Boden fest"; got != want {
		t.Errorf("RuleText = %q, want %q", got, want)
	}
	r := AllowedOn(day(2026, 3, 20), specPlan(), day(2026, 3, 21))
	if got := r.RuleText(); got != "Reha: Boxenruhe – heute keine Bewegung" {
		t.Errorf("rest RuleText = %q", got)
	}
	long := Allowed{Phase: Phase{Name: "X", Activity: "walker", Conditions: strings.Repeat("ab ", 200)}, Minutes: 5}
	if n := len([]rune(long.RuleText())); n > 230 {
		t.Errorf("conditions not clipped: %d runes", n)
	}
}

func TestStateOn(t *testing.T) {
	start := day(2026, 3, 20)
	for _, c := range []struct {
		d    time.Time
		want string
	}{
		{day(2026, 3, 19), StateUpcoming}, {day(2026, 3, 20), StateRunning},
		{day(2026, 4, 30), StateRunning}, {day(2026, 5, 1), StateFinished},
	} {
		if got := StateOn(start, specPlan(), c.d); got != c.want {
			t.Errorf("%s: %s, want %s", c.d.Format("2006-01-02"), got, c.want)
		}
	}
}

func TestParsePhasesLegacy(t *testing.T) {
	raw := `[
		{"name":"Wochen","weeks":2,"activity":"walker","min_minutes":15,"max_minutes":20},
		{"name":"","duration_days":3,"activity":"lunge","min_minutes":0,"max_minutes":0},
		{"name":"Falsch","days":4,"activity":"walker","min_minutes":50,"max_minutes":10},
		{"name":"Box","days":2,"activity":"rest","min_minutes":9,"max_minutes":9}
	]`
	got, err := ParsePhases([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	want := []Phase{
		{Name: "Wochen", Days: 14, Activity: "walker", MinMinutes: 15, MaxMinutes: 20},
		{Name: "Phase 2", Days: 3, Activity: "lunge", MinMinutes: 20, MaxMinutes: 20},
		{Name: "Falsch", Days: 4, Activity: "walker", MinMinutes: 10, MaxMinutes: 10},
		{Name: "Box", Days: 2, Activity: "rest"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d phases", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("phase %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	if _, err := ParsePhases([]byte(`{"not":"a list"}`)); err == nil {
		t.Error("an object must be rejected")
	}
	// An unknown activity parses but is not usable.
	bad, _ := ParsePhases([]byte(`[{"name":"x","days":3,"activity":"dancing","max_minutes":10}]`))
	if AllowedOn(day(2026, 1, 1), bad, day(2026, 1, 1)) != nil {
		t.Error("unknown activity must not be allowed")
	}
}

func TestValidatePhases(t *testing.T) {
	ok := func(p ...Phase) []Phase { return p }
	good := Phase{Name: " Schritt ", Days: 5, Activity: "walker", MinMinutes: 10, MaxMinutes: 20, Conditions: " eben "}
	out, err := ValidatePhases(ok(good))
	if err != nil {
		t.Fatal(err)
	}
	if out[0].Name != "Schritt" || out[0].Conditions != "eben" {
		t.Errorf("not trimmed: %+v", out[0])
	}
	if _, err := ValidatePhases(specPlan()); err != nil {
		t.Errorf("spec plan: %v", err)
	}

	mod := func(f func(p *Phase)) []Phase { p := good; f(&p); return []Phase{p} }
	bad := map[string][]Phase{
		"empty":            nil,
		"no name":          mod(func(p *Phase) { p.Name = "  " }),
		"long name":        mod(func(p *Phase) { p.Name = strings.Repeat("x", 81) }),
		"zero days":        mod(func(p *Phase) { p.Days = 0 }),
		"too many days":    mod(func(p *Phase) { p.Days = 366 }),
		"unknown activity": mod(func(p *Phase) { p.Activity = "dancing" }),
		"min above max":    mod(func(p *Phase) { p.MinMinutes = 30 }),
		"zero min":         mod(func(p *Phase) { p.MinMinutes = 0 }),
		"max too large":    mod(func(p *Phase) { p.MaxMinutes = 241 }),
		"long conditions":  mod(func(p *Phase) { p.Conditions = strings.Repeat("x", 501) }),
		"rest with mins":   mod(func(p *Phase) { p.Activity = "rest" }),
		"too many phases": func() []Phase {
			l := make([]Phase, 21)
			for i := range l {
				l[i] = good
			}
			return l
		}(),
		"too long": []Phase{{Name: "a", Days: 365, Activity: "rest"}, {Name: "b", Days: 365, Activity: "rest"}, {Name: "c", Days: 1, Activity: "rest"}},
	}
	for name, ph := range bad {
		if _, err := ValidatePhases(ph); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
