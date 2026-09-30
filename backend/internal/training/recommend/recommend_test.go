package recommend

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// today is a Wednesday; the Monday of its week is 2026-09-28.
var today = time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)

func day(offset int) time.Time { return today.AddDate(0, 0, offset) }

func allOn() []training.AllowedActivity {
	var l []training.AllowedActivity
	for _, a := range training.AllActivities {
		l = append(l, training.AllowedActivity{Activity: a, Mode: training.ModeOn})
	}
	return l
}

func only(acts ...training.Activity) []training.AllowedActivity {
	var l []training.AllowedActivity
	for _, a := range acts {
		l = append(l, training.AllowedActivity{Activity: a, Mode: training.ModeOn})
	}
	return l
}

func base(allowed []training.AllowedActivity) Input {
	return Input{
		Today: today,
		Profile: training.Profile{
			HorseName: "Luna", Allowed: allowed, Rhythm: training.DefaultRhythm(), Status: training.StatusFit,
		},
		Role: RoleOwner,
	}
}

func sess(offset int, a training.Activity, l float64) training.Session {
	return training.Session{Day: day(offset), Activity: a, Load: l}
}

func acts(r Result) []training.Activity {
	var out []training.Activity
	for _, x := range r.Recommendations {
		out = append(out, x.Activity)
	}
	return out
}

func contains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%q does not contain %q", got, want)
	}
}

func TestDeterministicTieBreak(t *testing.T) {
	in := base(allOn())
	r1 := Recommend(in)
	r2 := Recommend(in)
	if !reflect.DeepEqual(r1, r2) {
		t.Fatal("not deterministic")
	}
	want := []training.Activity{training.ActivityHall, training.ActivityArena, training.ActivityHack}
	if !reflect.DeepEqual(acts(r1), want) {
		t.Errorf("got %v want %v", acts(r1), want)
	}
	if len(r1.Hidden) != 0 {
		t.Errorf("hidden: %v", r1.Hidden)
	}
}

func TestTopThreeOnly(t *testing.T) {
	if n := len(Recommend(base(allOn())).Recommendations); n != 3 {
		t.Errorf("got %d recommendations", n)
	}
	if n := len(Recommend(base(only(training.ActivityHall))).Recommendations); n != 1 {
		t.Errorf("got %d recommendations", n)
	}
}

func TestVarietyPrefersUntrainedActivity(t *testing.T) {
	in := base(only(training.ActivityHall, training.ActivityArena))
	in.Recent = []training.Session{sess(-1, training.ActivityHall, 20), sess(-6, training.ActivityArena, 20)}
	r := Recommend(in)
	if r.Recommendations[0].Activity != training.ActivityArena {
		t.Fatalf("got %v", acts(r))
	}
	contains(t, r.Recommendations[0].Reason, "seit 6 Tagen")
}

func TestOldSessionsAreIgnored(t *testing.T) {
	in := base(only(training.ActivityHall, training.ActivityArena))
	in.Recent = []training.Session{sess(-20, training.ActivityArena, 20), sess(-1, training.ActivityHall, 20)}
	r := Recommend(in)
	contains(t, r.Recommendations[0].Reason, "nicht an")
}

func TestIntenseRecentLoadPrefersLight(t *testing.T) {
	in := base(only(training.ActivityHall, training.ActivityJumping, training.ActivityWalker))
	in.Recent = []training.Session{sess(-1, training.ActivityJumping, 80)}
	r := Recommend(in)
	if r.Recommendations[0].Activity != training.ActivityWalker {
		t.Fatalf("got %v", acts(r))
	}
	contains(t, r.Recommendations[0].Reason, "intensiv")
	if acts(r)[2] != training.ActivityJumping {
		t.Errorf("jumping should be last: %v", acts(r))
	}
}

func TestShowTomorrowOnlyLight(t *testing.T) {
	in := base(allOn())
	in.Profile.Shows = []training.Show{{Date: day(1), Name: "Turnier Nord"}}
	r := Recommend(in)
	for _, x := range r.Recommendations {
		if x.Intensity != training.IntensityLight {
			t.Errorf("%v is not light", x.Activity)
		}
		if x.Minutes > ShowLightMaxMinutes {
			t.Errorf("%v: %d minutes", x.Activity, x.Minutes)
		}
	}
	contains(t, r.Recommendations[0].Reason, "Turnier Nord ist morgen")
}

func TestShowTodayOnlyLight(t *testing.T) {
	in := base(allOn())
	in.Profile.Shows = []training.Show{{Date: day(0), Name: "Heimturnier"}}
	r := Recommend(in)
	contains(t, r.Recommendations[0].Reason, "heute")
	for _, x := range r.Recommendations {
		if x.Intensity != training.IntensityLight {
			t.Errorf("%v is not light", x.Activity)
		}
	}
}

func TestShowInThreeDaysAllowsLastDemandingSession(t *testing.T) {
	in := base(only(training.ActivityHall))
	in.Profile.Shows = []training.Show{{Date: day(3), Name: "Kreismeisterschaft"}}
	in.Recent = []training.Session{sess(-2, training.ActivityHall, 20)}
	r := Recommend(in)
	if r.Recommendations[0].Activity != training.ActivityHall {
		t.Fatalf("got %v", acts(r))
	}
	contains(t, r.Recommendations[0].Reason, "in 3 Tagen")
}

func TestShowInThreeDaysNoBonusAfterIntenseDay(t *testing.T) {
	in := base(only(training.ActivityHall))
	in.Profile.Shows = []training.Show{{Date: day(3), Name: "Kreismeisterschaft"}}
	in.Recent = []training.Session{sess(-1, training.ActivityHall, 80)}
	if r := Recommend(in).Recommendations[0].Reason; strings.Contains(r, "Kreismeisterschaft") {
		t.Errorf("show bonus must not apply: %q", r)
	}
}

func TestRestDayAfterShow(t *testing.T) {
	in := base(allOn())
	in.Profile.Shows = []training.Show{{Date: day(-1), Name: "Turnier Süd"}}
	r := Recommend(in)
	if len(r.Recommendations) != 1 || r.Recommendations[0].Activity != training.ActivityRest {
		t.Fatalf("got %v", acts(r))
	}
	if r.Recommendations[0].Minutes != 0 || r.Recommendations[0].Intensity != training.IntensityNone {
		t.Errorf("rest: %+v", r.Recommendations[0])
	}
	contains(t, r.Recommendations[0].Reason, "Gestern war Turnier Süd")

	in.Profile.Rhythm.RestAfterShow = false
	if Recommend(in).Recommendations[0].Activity == training.ActivityRest {
		t.Error("rule disabled but still rest")
	}
}

func TestRainPrefersHall(t *testing.T) {
	in := base(only(training.ActivityArena, training.ActivityHack, training.ActivityHall))
	in.Weather = &Weather{Rain: true, TempC: 12}
	r := Recommend(in)
	want := []training.Activity{training.ActivityHall, training.ActivityArena, training.ActivityHack}
	// hack is penalised more than arena; both behind the hall
	if r.Recommendations[0].Activity != training.ActivityHall {
		t.Fatalf("got %v", acts(r))
	}
	contains(t, r.Recommendations[0].Reason, "Regen")
	if !reflect.DeepEqual(acts(r), want) && !reflect.DeepEqual(acts(r), []training.Activity{training.ActivityHall, training.ActivityHack, training.ActivityArena}) {
		t.Errorf("unexpected order %v", acts(r))
	}
	if r.Recommendations[1].Score >= r.Recommendations[0].Score {
		t.Error("outdoor not penalised")
	}
}

func TestWetGroundPrefersHall(t *testing.T) {
	for _, g := range []Ground{GroundWet, GroundMuddy} {
		in := base(only(training.ActivityArena, training.ActivityHall, training.ActivityJumping))
		in.Ground = g
		r := Recommend(in)
		if r.Recommendations[0].Activity != training.ActivityHall {
			t.Errorf("%s: got %v", g, acts(r))
		}
		contains(t, r.Recommendations[0].Reason, "Boden")
	}
}

func TestFrozenGroundNoJumping(t *testing.T) {
	in := base(only(training.ActivityJumping, training.ActivityHall))
	in.Ground = GroundFrozen
	r := Recommend(in)
	if !reflect.DeepEqual(acts(r), []training.Activity{training.ActivityHall}) {
		t.Errorf("got %v", acts(r))
	}
	contains(t, r.Recommendations[0].Reason, "gefroren")

	in = base(only(training.ActivityJumping))
	in.Ground = GroundFrozen
	if Recommend(in).Recommendations[0].Activity != training.ActivityRest {
		t.Error("only jumping on frozen ground should be rest")
	}
}

func TestHeatPrefersLight(t *testing.T) {
	in := base(only(training.ActivityHall, training.ActivityWalker))
	in.Weather = &Weather{TempC: 31}
	if got := Recommend(in).Recommendations[0].Activity; got != training.ActivityWalker {
		t.Errorf("got %v", got)
	}
}

func TestAvailableTimeCutsDuration(t *testing.T) {
	in := base(only(training.ActivityHall))
	in.AvailableMinutes = 40
	if m := Recommend(in).Recommendations[0].Minutes; m != 40 {
		t.Errorf("minutes=%d", m)
	}
	in.Profile.Rhythm.MaxMinutes = 30
	in.AvailableMinutes = 0
	if m := Recommend(in).Recommendations[0].Minutes; m != 30 {
		t.Errorf("max minutes: %d", m)
	}
}

func TestAvailableTimePenalisesLongActivities(t *testing.T) {
	in := base(only(training.ActivityHack, training.ActivityWalker))
	in.AvailableMinutes = 20 // below hack's 30 minute minimum
	r := Recommend(in)
	if r.Recommendations[0].Activity != training.ActivityWalker || r.Recommendations[0].Minutes != 20 {
		t.Errorf("got %+v", r.Recommendations)
	}
	if r.Recommendations[1].Minutes != 20 {
		t.Errorf("hack should be cut to 20: %+v", r.Recommendations[1])
	}
}

func TestPauseOnlyLight(t *testing.T) {
	in := base(allOn())
	in.Profile.Status = training.StatusPause
	r := Recommend(in)
	for _, x := range r.Recommendations {
		if x.Intensity != training.IntensityLight || x.Minutes > PauseMaxMinutes {
			t.Errorf("pause: %+v", x)
		}
	}
	contains(t, r.Recommendations[0].Reason, "Trainingspause")

	in = base(only(training.ActivityHall, training.ActivityJumping))
	in.Profile.Status = training.StatusPause
	if Recommend(in).Recommendations[0].Activity != training.ActivityRest {
		t.Error("pause with only demanding activities should be rest")
	}
}

func TestRehaPhaseOverride(t *testing.T) {
	in := base(only(training.ActivityHall, training.ActivityJumping))
	in.Profile.Status = training.StatusReha
	in.Reha = &RehaPhase{Name: "Phase 2 Schrittarbeit", Activity: training.ActivityWalker, MinMinutes: 10, MaxMinutes: 25, Conditions: "nur auf ebenem Boden"}
	in.AvailableMinutes = 60
	r := Recommend(in)
	if len(r.Recommendations) != 1 {
		t.Fatalf("got %d", len(r.Recommendations))
	}
	x := r.Recommendations[0]
	if x.Activity != training.ActivityWalker || x.Minutes != 25 || x.Note != "nur auf ebenem Boden" {
		t.Errorf("got %+v", x)
	}
	contains(t, x.Reason, "Phase 2 Schrittarbeit")

	in.AvailableMinutes = 15
	if m := Recommend(in).Recommendations[0].Minutes; m != 15 {
		t.Errorf("minutes=%d", m)
	}
	in.AvailableMinutes = 5
	if m := Recommend(in).Recommendations[0].Minutes; m != 10 {
		t.Errorf("min clamp: %d", m)
	}
}

// A rest phase (box rest) allows nothing: one rest recommendation, for owners and riders alike.
func TestRehaRestPhase(t *testing.T) {
	for _, role := range []Role{RoleOwner, RoleRider} {
		in := base(allOn())
		in.Profile.Status = training.StatusReha
		in.Role = role
		in.Rider = &training.RiderRules{AllowedActivities: training.AllActivities}
		in.Reha = &RehaPhase{Name: "Boxenruhe", Activity: training.ActivityRest, Conditions: "Nur Handgrasen"}
		r := Recommend(in)
		if len(r.Recommendations) != 1 || r.Recommendations[0].Activity != training.ActivityRest {
			t.Fatalf("role %v: got %v", role, acts(r))
		}
		contains(t, r.Recommendations[0].Reason, "Boxenruhe")
		if r.Recommendations[0].Note != "Nur Handgrasen" {
			t.Errorf("note = %q", r.Recommendations[0].Note)
		}
	}
}

func TestRehaPhaseBlockedForRider(t *testing.T) {
	in := base(allOn())
	in.Reha = &RehaPhase{Name: "Phase 1", Activity: training.ActivityLunge, MinMinutes: 10, MaxMinutes: 15}
	in.Role = RoleRider
	in.RiderName = "Anna"
	in.Rider = &training.RiderRules{AllowedActivities: []training.Activity{training.ActivityHall}}
	r := Recommend(in)
	if len(r.Recommendations) != 1 || r.Recommendations[0].Activity != training.ActivityRest {
		t.Fatalf("got %v", acts(r))
	}
	contains(t, r.Recommendations[0].Reason, "Phase 1")
}

func TestRehaWithoutPhaseIsLightOnly(t *testing.T) {
	in := base(allOn())
	in.Profile.Status = training.StatusReha
	r := Recommend(in)
	for _, x := range r.Recommendations {
		if x.Intensity != training.IntensityLight {
			t.Errorf("reha: %+v", x)
		}
	}
	contains(t, r.Recommendations[0].Reason, "Reha")
}

func TestEmptyAllowedList(t *testing.T) {
	r := Recommend(base(nil))
	if len(r.Recommendations) != 1 || r.Recommendations[0].Activity != training.ActivityRest {
		t.Fatalf("got %v", acts(r))
	}
	contains(t, r.Recommendations[0].Reason, "keine Aktivität freigegeben")
	if len(r.Hidden) != len(training.AllActivities) {
		t.Errorf("hidden: %d", len(r.Hidden))
	}

	off := base([]training.AllowedActivity{{Activity: training.ActivityHall, Mode: training.ModeOff}})
	if Recommend(off).Recommendations[0].Activity != training.ActivityRest {
		t.Error("all off should be rest")
	}
}

func TestHiddenByProfile(t *testing.T) {
	in := base(only(training.ActivityHall))
	r := Recommend(in)
	var found bool
	for _, h := range r.Hidden {
		if h.Activity == training.ActivityJumping {
			found = true
			if h.Reason != "Springen ausgeblendet: laut Profil nicht für Luna" {
				t.Errorf("reason: %q", h.Reason)
			}
		}
	}
	if !found {
		t.Error("jumping not hidden")
	}
	if len(r.Hidden) != len(training.AllActivities)-1 {
		t.Errorf("hidden: %d", len(r.Hidden))
	}
}

func TestConditionalActivityShowsNote(t *testing.T) {
	in := base([]training.AllowedActivity{{Activity: training.ActivityHack, Mode: training.ModeConditional, Note: "nur mit Begleitung"}})
	r := Recommend(in)
	x := r.Recommendations[0]
	if x.Activity != training.ActivityHack || x.Note != "nur mit Begleitung" {
		t.Errorf("got %+v", x)
	}
}

func TestConditionalRanksBelowUnconditional(t *testing.T) {
	in := base([]training.AllowedActivity{
		{Activity: training.ActivityHall, Mode: training.ModeConditional, Note: "nur bis 30 Minuten"},
		{Activity: training.ActivityArena, Mode: training.ModeOn},
	})
	if got := Recommend(in).Recommendations[0].Activity; got != training.ActivityArena {
		t.Errorf("got %v", got)
	}
}

func riderInput(rules *training.RiderRules) Input {
	in := base(allOn())
	in.Role = RoleRider
	in.RiderName = "Anna"
	in.Rider = rules
	return in
}

func TestRiderRestrictions(t *testing.T) {
	in := riderInput(&training.RiderRules{
		AllowedActivities: []training.Activity{training.ActivityHall, training.ActivityHack, training.ActivityJumping},
		MaxIntensity:      training.IntensityMedium,
	})
	r := Recommend(in)
	if !reflect.DeepEqual(acts(r), []training.Activity{training.ActivityHall}) {
		t.Fatalf("got %v", acts(r))
	}
	reasons := map[training.Activity]string{}
	for _, h := range r.Hidden {
		reasons[h.Activity] = h.Reason
	}
	if reasons[training.ActivityHack] != "Ausritt ausgeblendet: Anna darf nicht allein ausreiten" {
		t.Errorf("hack: %q", reasons[training.ActivityHack])
	}
	if reasons[training.ActivityJumping] != "Springen ausgeblendet: zu intensiv für Anna" {
		t.Errorf("jumping: %q", reasons[training.ActivityJumping])
	}
	if reasons[training.ActivityArena] != "Platz ausgeblendet: für Anna nicht freigegeben" {
		t.Errorf("arena: %q", reasons[training.ActivityArena])
	}

	in.Rider.MayHackAlone = true
	got := acts(Recommend(in))
	if !reflect.DeepEqual(got, []training.Activity{training.ActivityHall, training.ActivityHack}) {
		t.Errorf("with hack: %v", got)
	}
}

func TestRiderWithoutRulesGetsNothing(t *testing.T) {
	r := Recommend(riderInput(nil))
	if r.Recommendations[0].Activity != training.ActivityRest {
		t.Errorf("got %v", acts(r))
	}
	contains(t, r.Recommendations[0].Reason, "Anna")
}

func TestOwnerIgnoresRiderRules(t *testing.T) {
	in := riderInput(&training.RiderRules{})
	in.Role = RoleOwner
	if len(Recommend(in).Hidden) != 0 {
		t.Error("owner should not be filtered")
	}
}

func TestProfileOffBeatsRiderAllowList(t *testing.T) {
	in := riderInput(&training.RiderRules{AllowedActivities: []training.Activity{training.ActivityJumping, training.ActivityHall}, MayHackAlone: true})
	in.Profile.Allowed = only(training.ActivityHall)
	r := Recommend(in)
	if !reflect.DeepEqual(acts(r), []training.Activity{training.ActivityHall}) {
		t.Errorf("got %v", acts(r))
	}
}

func TestFullWeekSuggestsRest(t *testing.T) {
	in := base(only(training.ActivityHall, training.ActivityWalker))
	in.Recent = []training.Session{
		sess(-2, training.ActivityHall, 30), sess(-2, training.ActivityWalker, 10),
		sess(-1, training.ActivityHall, 30), sess(-1, training.ActivityWalker, 10),
		sess(0, training.ActivityHall, 30),
	}
	r := Recommend(in)
	if r.Recommendations[0].Activity != training.ActivityRest {
		t.Fatalf("got %v", acts(r))
	}
	contains(t, r.Recommendations[0].Reason, "5 Einheiten")

	// The same sessions are spread over the previous week: no forced rest.
	in.Recent = nil
	for i := 0; i < 5; i++ {
		in.Recent = append(in.Recent, sess(-5-i, training.ActivityHall, 30))
	}
	if Recommend(in).Recommendations[0].Activity == training.ActivityRest {
		t.Error("previous week must not count")
	}
}

func TestDisciplineNudge(t *testing.T) {
	in := base(only(training.ActivityHack, training.ActivityJumping))
	in.Profile.Discipline = "jumping"
	in.Recent = []training.Session{sess(-2, training.ActivityHack, 20), sess(-2, training.ActivityJumping, 20)}
	r := Recommend(in)
	if r.Recommendations[0].Activity != training.ActivityJumping {
		t.Errorf("got %v", acts(r))
	}
}

func TestAllReasonsAreNonEmptyGerman(t *testing.T) {
	inputs := []Input{
		base(allOn()),
		func() Input { i := base(allOn()); i.Profile.Status = training.StatusPause; return i }(),
		func() Input { i := base(allOn()); i.Weather = &Weather{Rain: true}; return i }(),
		func() Input {
			i := base(allOn())
			i.Recent = []training.Session{sess(-1, training.ActivityHall, 90)}
			return i
		}(),
	}
	for n, in := range inputs {
		for _, x := range Recommend(in).Recommendations {
			if x.Reason == "" || !strings.HasSuffix(x.Reason, ".") {
				t.Errorf("input %d: %+v", n, x)
			}
		}
	}
}

func TestCheckAcceptsAllowedActivityAndFitsMinutes(t *testing.T) {
	in := base(allOn())
	v := Check(in, training.ActivityHack, 90)
	if !v.OK || v.Why != "" || v.Recommendation.Activity != training.ActivityHack {
		t.Fatalf("verdict = %+v", v)
	}
	if v.Recommendation.Minutes != 60 { // rhythm maximum
		t.Errorf("minutes = %d, want 60", v.Recommendation.Minutes)
	}
	if v := Check(in, training.ActivityLunge, 0); !v.OK || v.Recommendation.Minutes != 25 {
		t.Errorf("zero minutes should become the default duration: %+v", v)
	}
	in.Profile.Rhythm.MaxMinutes = 0
	if v := Check(in, training.ActivityHack, 500); v.Recommendation.Minutes != MaxPlanMinutes {
		t.Errorf("no rhythm maximum: minutes = %d", v.Recommendation.Minutes)
	}
}

func TestCheckRestAlwaysPasses(t *testing.T) {
	in := base(allOn())
	in.Profile.Status = training.StatusReha
	in.Reha = &RehaPhase{Name: "Phase 1", Activity: training.ActivityWalker, MinMinutes: 10, MaxMinutes: 20}
	if v := Check(in, training.ActivityRest, 0); !v.OK || v.Recommendation.Activity != training.ActivityRest {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestCheckReplacesHiddenActivity(t *testing.T) {
	in := base(only(training.ActivityHall, training.ActivityLunge))
	v := Check(in, training.ActivityJumping, 40)
	if v.OK || v.Recommendation.Activity == training.ActivityJumping {
		t.Fatalf("verdict = %+v", v)
	}
	contains(t, v.Why, "Springen ausgeblendet")
	if v.Recommendation.Activity != Recommend(in).Recommendations[0].Activity {
		t.Errorf("replacement %v is not the top recommendation", v.Recommendation.Activity)
	}
	if v := Check(in, "swimming", 30); v.OK {
		t.Errorf("unknown activity passed: %+v", v)
	}
}

func TestCheckHardFilters(t *testing.T) {
	in := base(allOn())
	in.Profile.Shows = []training.Show{{Date: day(1), Name: "Turnier Nord"}}
	v := Check(in, training.ActivityHall, 45)
	if v.OK {
		t.Fatalf("medium work the day before a show passed: %+v", v)
	}
	contains(t, v.Why, "Turnier Nord ist morgen")
	if v := Check(in, training.ActivityLunge, 45); !v.OK || v.Recommendation.Minutes != ShowLightMaxMinutes {
		t.Errorf("light work before a show: %+v", v)
	}

	in = base(allOn())
	in.Ground = GroundFrozen
	if v := Check(in, training.ActivityJumping, 30); v.OK || !strings.Contains(v.Why, "gefroren") {
		t.Errorf("jumping on frozen ground: %+v", v)
	}

	in = base(allOn())
	in.Profile.Status = training.StatusPause
	if v := Check(in, training.ActivityHack, 60); v.OK || !strings.Contains(v.Why, "Pause") {
		t.Errorf("hack during a pause: %+v", v)
	}
	if v := Check(in, training.ActivityWalker, 60); !v.OK || v.Recommendation.Minutes != PauseMaxMinutes {
		t.Errorf("walker during a pause: %+v", v)
	}
}

func TestCheckRehaPhaseAndRestAfterShow(t *testing.T) {
	in := base(allOn())
	in.Profile.Status = training.StatusReha
	in.Reha = &RehaPhase{Name: "Phase 2", Activity: training.ActivityWalker, MinMinutes: 10, MaxMinutes: 25}
	if v := Check(in, training.ActivityHall, 45); v.OK || v.Recommendation.Activity != training.ActivityWalker {
		t.Errorf("reha phase: %+v", v)
	}
	if v := Check(in, training.ActivityWalker, 5); !v.OK || v.Recommendation.Minutes != 10 {
		t.Errorf("reha minimum: %+v", v)
	}
	if v := Check(in, training.ActivityWalker, 40); !v.OK || v.Recommendation.Minutes != 25 {
		t.Errorf("reha maximum: %+v", v)
	}

	in = base(allOn())
	in.Profile.Shows = []training.Show{{Date: day(-1), Name: "Turnier Süd"}}
	v := Check(in, training.ActivityHall, 45)
	if v.OK || v.Recommendation.Activity != training.ActivityRest {
		t.Fatalf("rest after show: %+v", v)
	}
	contains(t, v.Why, "Turnier Süd")
}

func TestCheckWeekMaximum(t *testing.T) {
	in := base(allOn()) // Wednesday; SessionsMax 5
	in.Recent = []training.Session{sess(-2, training.ActivityHall, 30), sess(-1, training.ActivityHack, 30)}
	if v := Check(in, training.ActivityArena, 45); !v.OK {
		t.Fatalf("two sessions this week: %+v", v)
	}
	in.Profile.Rhythm.SessionsMax = 2
	v := Check(in, training.ActivityArena, 45)
	if v.OK || v.Recommendation.Activity != training.ActivityRest {
		t.Fatalf("week full: %+v", v)
	}
	contains(t, v.Why, "2 Einheiten")
}
