package recommend

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// unitBase is a dressage horse of 10 years at level intermediate (Schubkraft), fit, with the
// given focuses of the owner, all activities allowed, no history.
func unitBase(focuses ...string) Input {
	in := base(allOn())
	in.Profile.Discipline = "dressage"
	in.Level = training.LevelIntermediate
	in.AgeYears = 10
	for _, k := range focuses {
		in.Profile.Focuses = append(in.Profile.Focuses, training.FocusChoice{Key: k})
	}
	return in
}

// trainedFocus is a session of the horse with a focus key.
func trainedFocus(offset int, a training.Activity, focus string) training.Session {
	return training.Session{Day: day(offset), Activity: a, Minutes: 40, Load: 32, FocusKey: focus}
}

func withShow(in Input, inDays int) Input {
	in.Profile.Shows = []training.Show{{Date: day(inDays), Name: "Turnier"}}
	return in
}

func TestFocusRotation(t *testing.T) {
	hall, arena, lunge, hack, ground, jump := training.ActivityHall, training.ActivityArena, training.ActivityLunge,
		training.ActivityHack, training.ActivityGroundwork, training.ActivityJumping
	until := func(in Input, key string, offset int) Input {
		d := day(offset)
		in.Profile.Focuses = append(in.Profile.Focuses, training.FocusChoice{Key: key, Until: &d})
		return in
	}
	cases := []struct {
		name string
		in   Input
		act  training.Activity
		key  string
		why  string
	}{
		// Without a focus the activity decides.
		{"hall without focuses", unitBase(), hall, "losgelassenheit", "Ohne Schwerpunkt im Profil: Losgelassenheit kommt immer zuerst."},
		{"arena without focuses", unitBase(), arena, "losgelassenheit", "Ohne Schwerpunkt im Profil: Losgelassenheit kommt immer zuerst."},
		{"lunge without focuses", unitBase(), lunge, "losgelassenheit", "Ohne Schwerpunkt im Profil: Losgelassenheit kommt immer zuerst."},
		{"groundwork without focuses", unitBase(), ground, "vertrauen", "Ohne Schwerpunkt im Profil: Vertrauen ist die Basis der Bodenarbeit."},
		{"hack without focuses", unitBase(), hack, "kondition", "Ohne Schwerpunkt im Profil: Kondition passt zum Ausritt."},
		{"jumping without focuses", unitBase(), jump, "springen", "Ohne Schwerpunkt im Profil: Springgymnastik steht beim Springen im Mittelpunkt."},
		// Focuses that do not fit the activity: the fallback, with its own words.
		{"geraderichten on a hack", unitBase("geraderichten"), hack, "kondition", "Kein Schwerpunkt im Profil passt zum Ausritt: Kondition passt zum Ausritt."},
		{"versammlung on the lunge", unitBase("versammlung"), lunge, "losgelassenheit", "Kein Schwerpunkt im Profil passt zur Longe: Losgelassenheit kommt immer zuerst."},
		{"takt in groundwork", unitBase("takt"), ground, "vertrauen", "Kein Schwerpunkt im Profil passt zur Bodenarbeit: Vertrauen ist die Basis der Bodenarbeit."},
		// The rotation: the candidate trained longest ago, never first, ties by priority.
		{"two focuses, none trained", unitBase("takt", "losgelassenheit"), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"takt yesterday", func() Input {
			in := unitBase("takt", "losgelassenheit")
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "losgelassenheit", "Schwerpunkt Losgelassenheit: noch nicht geübt."},
		{"takt longer ago than losgelassenheit", func() Input {
			in := unitBase("takt", "losgelassenheit")
			in.Recent = []training.Session{trainedFocus(-3, hall, "takt"), trainedFocus(-1, arena, "losgelassenheit")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: zuletzt am Sonntag."},
		{"a date for an older one", func() Input {
			in := unitBase("takt", "losgelassenheit")
			in.Recent = []training.Session{trainedFocus(-2, hall, "takt"), trainedFocus(-8, hall, "losgelassenheit")}
			return in
		}(), hall, "losgelassenheit", "Schwerpunkt Losgelassenheit: zuletzt am 22.09."},
		{"yesterday", func() Input {
			in := unitBase("takt")
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: zuletzt gestern."},
		{"today", func() Input {
			in := unitBase("takt")
			in.Recent = []training.Session{trainedFocus(0, hall, "takt")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: heute schon geübt."},
		{"equal days: the order of the profile", func() Input {
			in := unitBase("anlehnung", "takt")
			in.Recent = []training.Session{trainedFocus(-4, hall, "anlehnung"), trainedFocus(-4, hall, "takt")}
			return in
		}(), hall, "anlehnung", "Schwerpunkt Anlehnung: zuletzt am Samstag."},
		{"the lookback is 14 days", func() Input {
			in := unitBase("takt", "losgelassenheit")
			in.Recent = []training.Session{trainedFocus(-20, hall, "takt"), trainedFocus(-1, hall, "losgelassenheit")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"a focus of another activity is not skipped by the history", func() Input {
			in := unitBase("takt", "gelassenheit")
			in.Recent = []training.Session{trainedFocus(-1, hack, "takt")} // takt on a hack counts
			return in
		}(), arena, "gelassenheit", "Schwerpunkt Gelassenheit: noch nicht geübt."},
		// A focus above the horse's phase at most once a week.
		{"above the phase, not trained", func() Input {
			in := unitBase("geraderichten", "takt")
			in.Level = training.LevelBeginner
			return in
		}(), hall, "geraderichten", "Schwerpunkt Geraderichtung: noch nicht geübt."},
		{"above the phase, trained 5 days ago: dropped", func() Input {
			in := unitBase("geraderichten", "takt")
			in.Level = training.LevelBeginner
			in.Recent = []training.Session{trainedFocus(-5, hall, "geraderichten")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"above the phase, planned yesterday: dropped", func() Input {
			in := unitBase("geraderichten", "takt")
			in.Level = ""
			in.Recent = []training.Session{trainedFocus(-1, hall, "geraderichten")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"above the phase, 8 days ago: allowed again", func() Input {
			in := unitBase("geraderichten", "takt")
			in.Level = training.LevelBeginner
			in.Recent = []training.Session{trainedFocus(-8, hall, "geraderichten"), trainedFocus(-2, hall, "takt")}
			return in
		}(), hall, "geraderichten", "Schwerpunkt Geraderichtung: zuletzt am 22.09."},
		{"within the phase the week does not matter", func() Input {
			in := unitBase("geraderichten", "takt")
			in.Recent = []training.Session{trainedFocus(-5, hall, "geraderichten"), trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "geraderichten", "Schwerpunkt Geraderichtung: zuletzt am Freitag."},
		{"a young horse is in the first phase", func() Input {
			in := unitBase("schwung", "takt")
			in.AgeYears = 4
			in.Recent = []training.Session{trainedFocus(-2, hall, "schwung")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"a horse in reha is in the first phase", func() Input {
			in := unitBase("schwung", "takt")
			in.Profile.Status = training.StatusReha
			in.Recent = []training.Session{trainedFocus(-2, hall, "schwung")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		// A date on a focus.
		{"until yesterday is over", until(unitBase(), "takt", -1), hall, "losgelassenheit", "Ohne Schwerpunkt im Profil: Losgelassenheit kommt immer zuerst."},
		{"until today still counts", until(unitBase(), "takt", 0), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"until next week counts", until(unitBase("anlehnung"), "takt", 7), hall, "anlehnung", "Schwerpunkt Anlehnung: noch nicht geübt."},
		// The rider's focuses come first.
		{"rider focus", func() Input {
			in := unitBase("takt", "losgelassenheit")
			in.RiderFocuses, in.RiderName = []string{"gelassenheit"}, "Mia"
			return in
		}(), arena, "gelassenheit", "Mias Schwerpunkt Gelassenheit."},
		{"rider name ending in s", func() Input {
			in := unitBase("takt")
			in.RiderFocuses, in.RiderName = []string{"gelassenheit"}, "Lukas"
			return in
		}(), arena, "gelassenheit", "Lukas' Schwerpunkt Gelassenheit."},
		{"rider without a name", func() Input {
			in := unitBase("takt")
			in.RiderFocuses = []string{"gelassenheit"}
			return in
		}(), arena, "gelassenheit", "Eigener Schwerpunkt Gelassenheit."},
		{"rider focus that does not fit the hall", func() Input {
			in := unitBase("takt")
			in.RiderFocuses, in.RiderName = []string{"gelassenheit"}, "Mia"
			return in
		}(), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"rider focus trained yesterday: the horse's turn", func() Input {
			in := unitBase("takt")
			in.RiderFocuses, in.RiderName = []string{"anlehnung"}, "Mia"
			in.Recent = []training.Session{trainedFocus(-1, hall, "anlehnung")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"rider and horse have the same focus", func() Input {
			in := unitBase("takt", "anlehnung")
			in.RiderFocuses, in.RiderName = []string{"takt"}, "Mia"
			return in
		}(), hall, "takt", "Mias Schwerpunkt Takt."},
		{"an unknown rider focus is ignored", func() Input {
			in := unitBase("takt")
			in.RiderFocuses = []string{"tempo"}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		// A show within 14 days: turnier is the last candidate for hall, arena and jumping, twice a week.
		{"show in 10 days, turnier is due", func() Input {
			in := withShow(unitBase("takt"), 10)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "turnier", "Turnier am 10.10.: Aufgabe und Parcours üben."},
		{"show in 10 days, takt is due first", withShow(unitBase("takt"), 10), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"show in 14 days", func() Input {
			in := withShow(unitBase("takt"), 14)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), arena, "turnier", "Turnier am 14.10.: Aufgabe und Parcours üben."},
		{"show in 15 days", func() Input {
			in := withShow(unitBase("takt"), 15)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: zuletzt gestern."},
		{"turnier at most twice a week", func() Input {
			in := withShow(unitBase("takt"), 10)
			in.Recent = []training.Session{trainedFocus(-2, hall, "turnier"), trainedFocus(-1, hall, "turnier")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: noch nicht geübt."},
		{"turnier of last week does not count", func() Input {
			in := withShow(unitBase("takt"), 10)
			in.Recent = []training.Session{trainedFocus(-5, hall, "turnier"), trainedFocus(-4, hall, "turnier"), trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "turnier", "Turnier am 10.10.: Aufgabe und Parcours üben."},
		{"no turnier on the lunge", func() Input {
			in := withShow(unitBase("takt"), 10)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), lunge, "takt", "Schwerpunkt Takt: zuletzt gestern."},
		{"no turnier on a hack", withShow(unitBase(), 5), hack, "kondition", "Kein Schwerpunkt-Platz"},
		{"turnier for jumping", func() Input {
			in := withShow(unitBase("springen"), 7)
			in.Recent = []training.Session{trainedFocus(-1, jump, "springen")}
			return in
		}(), jump, "turnier", "Turnier am 07.10.: Aufgabe und Parcours üben."},
		{"a past show does nothing", func() Input {
			in := withShow(unitBase("takt"), -3)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "takt", "Schwerpunkt Takt: zuletzt gestern."},
		{"the owner's turnier with a date", func() Input {
			in := withShow(until(unitBase("takt"), "turnier", 10), 10)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "turnier", "Turnier am 10.10.: Aufgabe und Parcours üben."},
		{"the owner's turnier without a show", func() Input {
			in := until(unitBase("takt"), "turnier", 10)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "turnier", "Schwerpunkt Turniervorbereitung: noch nicht geübt."},
		{"an expired turnier is added by the planner", func() Input {
			in := withShow(until(unitBase("takt"), "turnier", -2), 10)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "turnier", "Turnier am 10.10.: Aufgabe und Parcours üben."},
		// Within seven days before a show: the first focus and turnier.
		{"week before a show: the first focus", withShow(unitBase("geraderichten", "takt", "turnier"), 5), hall, "geraderichten", "Vor dem Turnier bleibt es beim Schwerpunkt Geraderichtung."},
		{"week before a show: then turnier", func() Input {
			in := withShow(unitBase("geraderichten", "takt", "turnier"), 5)
			in.Recent = []training.Session{trainedFocus(-1, hall, "geraderichten")}
			return in
		}(), hall, "turnier", "Turnier am 05.10.: Aufgabe und Parcours üben."},
		{"week before a show: the automatic turnier", func() Input {
			in := withShow(unitBase("takt", "anlehnung"), 3)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), hall, "turnier", "Turnier am 03.10.: Aufgabe und Parcours üben."},
		{"week before a show: the rider's focus waits", func() Input {
			in := withShow(unitBase("takt"), 4)
			in.RiderFocuses, in.RiderName = []string{"anlehnung"}, "Mia"
			return in
		}(), hall, "takt", "Vor dem Turnier bleibt es beim Schwerpunkt Takt."},
		{"week before a show: the show is tomorrow", withShow(unitBase("takt"), 1), hall, "takt", "Vor dem Turnier bleibt es beim Schwerpunkt Takt."},
		{"week before a show: first focus does not fit", withShow(unitBase("geraderichten", "gelaende"), 5), hack, "gelaende", "Kein Schwerpunkt im Profil passt zum Ausritt: Gelände passt zum Ausritt."},
		{"week before a show: the other focuses wait", func() Input {
			in := withShow(unitBase("takt", "anlehnung"), 6)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), lunge, "takt", "Vor dem Turnier bleibt es beim Schwerpunkt Takt."},
		{"8 days before a show the rotation is free", func() Input {
			in := withShow(unitBase("takt", "anlehnung"), 8)
			in.Recent = []training.Session{trainedFocus(-1, hall, "takt")}
			return in
		}(), lunge, "anlehnung", "Schwerpunkt Anlehnung: noch nicht geübt."},
	}
	for _, c := range cases {
		if c.name == "no turnier on a hack" {
			// Three focuses of the catalog never fit: the fallback names the activity.
			u := PlanUnit(c.in, c.act, 60, Hints{})
			if u.FocusKey != c.key || !strings.HasPrefix(u.FocusWhy, "Ohne Schwerpunkt im Profil: ") {
				t.Errorf("%s: %s / %q", c.name, u.FocusKey, u.FocusWhy)
			}
			continue
		}
		u := PlanUnit(c.in, c.act, 45, Hints{})
		if u.FocusKey != c.key || u.FocusWhy != c.why {
			t.Errorf("%s: focus %s %q, want %s %q", c.name, u.FocusKey, u.FocusWhy, c.key, c.why)
		}
	}
}

func TestFocusHints(t *testing.T) {
	in := unitBase("takt", "losgelassenheit")
	hall := training.ActivityHall
	// The model's focus is taken when it is a candidate.
	if u := PlanUnit(in, hall, 45, Hints{FocusKey: "losgelassenheit"}); u.FocusKey != "losgelassenheit" || u.FocusWhy != "Schwerpunkt Losgelassenheit: noch nicht geübt." {
		t.Errorf("hint taken: %s %q", u.FocusKey, u.FocusWhy)
	}
	// Otherwise the rules pick: a focus of the catalog that is not a candidate, an unknown key.
	for _, hint := range []string{"versammlung", "tempo", "anlehnung"} {
		if u := PlanUnit(in, hall, 45, Hints{FocusKey: hint}); u.FocusKey != "takt" {
			t.Errorf("hint %q: focus %s, want the rules' takt", hint, u.FocusKey)
		}
	}
	// The same hint as the fallback of an activity is fine, and the fallback is what comes out.
	if u := PlanUnit(unitBase(), hall, 45, Hints{FocusKey: "kondition"}); u.FocusKey != "losgelassenheit" {
		t.Errorf("fallback with a hint: %s", u.FocusKey)
	}
	// No unit plan for walker and rest.
	for _, a := range []training.Activity{training.ActivityWalker, training.ActivityRest} {
		if u := PlanUnit(in, a, 30, Hints{FocusKey: "takt"}); !u.Empty() || u.Main.Minutes != 0 || u.Exercises() != nil {
			t.Errorf("%s has a unit: %+v", a, u)
		}
	}
}

func TestFocusRotationIsDeterministic(t *testing.T) {
	in := unitBase("takt", "losgelassenheit", "anlehnung")
	in.Recent = []training.Session{trainedFocus(-3, training.ActivityHall, "takt")}
	a := PlanUnit(in, training.ActivityHall, 45, Hints{})
	b := PlanUnit(in, training.ActivityHall, 45, Hints{})
	if !reflect.DeepEqual(a, b) || a.FocusKey != "losgelassenheit" {
		t.Errorf("%+v vs %+v", a, b)
	}
}
