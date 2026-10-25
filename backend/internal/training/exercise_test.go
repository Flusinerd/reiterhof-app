package training

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestExerciseMethods(t *testing.T) {
	ex := Exercise{Phase: PartWarmup, Tags: []string{"takt", "losgelassenheit", "takt"}, Equipment: []string{"lunge", "poles", "poles", "cavaletti"}, Suits: []string{"young"}}
	for part, want := range map[string]bool{PartWarmup: true, PartMain: false, PartCooldown: false} {
		if ex.FitsPart(part) != want {
			t.Errorf("warmup exercise FitsPart(%s) = %v", part, !want)
		}
	}
	for _, p := range []string{PartWarmup, PartMain, PartCooldown} {
		if !(Exercise{Phase: PhaseAny}).FitsPart(p) {
			t.Errorf("an any exercise fits %s", p)
		}
	}
	if !(Exercise{}).FitsPart(PartMain) || (Exercise{}).FitsPart(PartWarmup) {
		t.Error("an exercise without phase counts as main")
	}
	if n := ex.SharedTags([]string{"takt", "balance", "losgelassenheit"}); n != 2 {
		t.Errorf("SharedTags = %d, want 2 (duplicates count once)", n)
	}
	if !ex.HasTag("takt") || ex.HasTag("schwung") {
		t.Error("HasTag")
	}
	if got := ex.FacilityEquipment(); !reflect.DeepEqual(got, []string{"poles", "cavaletti"}) {
		t.Errorf("FacilityEquipment = %v", got)
	}
	if (Exercise{Equipment: []string{"lunge", "tarp"}}).FacilityEquipment() != nil {
		t.Error("personal equipment is not facility equipment")
	}
	for age, want := range map[int]bool{0: false, 4: true, 6: false, 18: false} {
		if ex.SuitsHorse(age) != want {
			t.Errorf("young exercise SuitsHorse(%d) = %v", age, !want)
		}
	}
	senior := Exercise{Suits: []string{"senior"}}
	if !senior.SuitsHorse(19) || senior.SuitsHorse(10) || senior.SuitsHorse(4) {
		t.Error("senior exercise")
	}
}

func TestLevelRankAndLabels(t *testing.T) {
	for level, want := range map[string]int{LevelBeginner: 0, LevelIntermediate: 1, LevelAdvanced: 2, "": -1, "expert": -1} {
		if LevelRank(level) != want {
			t.Errorf("LevelRank(%q) = %d", level, LevelRank(level))
		}
	}
	for level, want := range map[string]string{LevelBeginner: "A", LevelIntermediate: "L", LevelAdvanced: "M", "": ""} {
		if ClassLabel(level) != want {
			t.Errorf("ClassLabel(%q) = %q", level, ClassLabel(level))
		}
	}
}

func TestEquipmentKeys(t *testing.T) {
	if len(FacilityEquipment) != 4 || len(PersonalEquipment) != 11 || len(EquipmentKeys) != 15 {
		t.Fatalf("keys: %d facility, %d personal, %d in all", len(FacilityEquipment), len(PersonalEquipment), len(EquipmentKeys))
	}
	seen := map[string]bool{}
	for _, k := range EquipmentKeys {
		if seen[k] {
			t.Errorf("key %q twice", k)
		}
		seen[k] = true
		if !ValidEquipment(k) || EquipmentLabel(k) == k {
			t.Errorf("key %q: valid %v, label %q", k, ValidEquipment(k), EquipmentLabel(k))
		}
	}
	for _, k := range []string{"poles", "cavaletti", "jumps", "obstacles"} {
		if !IsFacility(k) {
			t.Errorf("%s is facility equipment", k)
		}
	}
	for _, k := range []string{"lunge", "side_reins", "double_lunge", "kappzaum", "tarp", "umbrella", "cones", "flag", "whip", "neck_ring", "trailer"} {
		if IsFacility(k) || !ValidEquipment(k) {
			t.Errorf("%s is personal equipment", k)
		}
	}
	if ValidEquipment("trampoline") || ValidEquipment("") || EquipmentLabel("trampoline") != "trampoline" {
		t.Error("unknown key")
	}
	if EquipmentLabel("poles") != "Stangen" || EquipmentLabel("obstacles") != "Gelassenheitsparcours" {
		t.Error("labels")
	}
}

func TestDefaultLocations(t *testing.T) {
	want := map[Activity]Location{
		ActivityHall:       {Equipment: []string{"poles"}},
		ActivityArena:      {Equipment: []string{"poles"}},
		ActivityJumping:    {Equipment: []string{"jumps", "poles"}},
		ActivityLunge:      {Equipment: []string{}},
		ActivityGroundwork: {Equipment: []string{"poles"}},
	}
	for _, a := range Locations {
		if got := DefaultLocation(a); !reflect.DeepEqual(got, want[a]) {
			t.Errorf("DefaultLocation(%s) = %+v, want %+v", a, got, want[a])
		}
		if !IsLocation(a) {
			t.Errorf("%s is a location", a)
		}
	}
	for _, a := range []Activity{ActivityHack, ActivityWalker, ActivityRest} {
		if IsLocation(a) {
			t.Errorf("%s has no location", a)
		}
	}
	s := StableEquipment{ActivityHall: {Equipment: []string{"poles", "cavaletti"}, Shared: true}}
	if got := s.Of(ActivityHall); !got.Shared || len(got.Equipment) != 2 {
		t.Errorf("stored location = %+v", got)
	}
	if got := s.Of(ActivityJumping); !reflect.DeepEqual(got, DefaultLocation(ActivityJumping)) {
		t.Errorf("a missing location takes the default: %+v", got)
	}
	if got := (StableEquipment(nil)).Of(ActivityLunge); got.Equipment == nil || len(got.Equipment) != 0 {
		t.Errorf("nil config, lunge = %+v", got)
	}
}

func ptr[T any](v T) *T { return &v }

func TestAvailable(t *testing.T) {
	cases := []struct {
		name string
		loc  Location
		ok   *bool
		want []string
	}{
		{"location equipment", Location{Equipment: []string{"poles", "cavaletti"}}, nil, []string{"poles", "cavaletti"}},
		{"override true", Location{Equipment: []string{"poles"}}, ptr(true), []string{"poles"}},
		{"override false: no set-up today", Location{Equipment: []string{"poles", "cavaletti"}}, ptr(false), nil},
		{"shared without override: none", Location{Equipment: []string{"poles"}, Shared: true}, nil, nil},
		{"shared, override false: none", Location{Equipment: []string{"poles"}, Shared: true}, ptr(false), nil},
		{"shared, override true: set-up possible", Location{Equipment: []string{"poles", "cavaletti"}, Shared: true}, ptr(true), []string{"poles", "cavaletti"}},
		{"personal equipment is not listed", Location{Equipment: []string{"lunge", "poles", "tarp"}}, nil, []string{"poles"}},
		{"duplicates collapse", Location{Equipment: []string{"poles", "poles"}}, nil, []string{"poles"}},
		{"empty location", Location{Equipment: []string{}}, ptr(true), nil},
		{"default lunge", DefaultLocation(ActivityLunge), nil, nil},
		{"default jumping", DefaultLocation(ActivityJumping), nil, []string{"jumps", "poles"}},
	}
	for _, c := range cases {
		got := Available(c.loc, c.ok)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: Available = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestFitsEquipment(t *testing.T) {
	trot := Exercise{Equipment: []string{"poles", "lunge"}}
	course := Exercise{Equipment: []string{"cavaletti", "jumps"}}
	free := Exercise{Equipment: []string{"lunge", "tarp"}}
	none := Exercise{}
	for _, c := range []struct {
		ex        Exercise
		available []string
		fits      bool
		missing   []string
	}{
		{trot, []string{"poles"}, true, nil},
		{trot, nil, false, []string{"poles"}},
		{course, []string{"cavaletti"}, false, []string{"jumps"}},
		{course, []string{"jumps", "cavaletti", "poles"}, true, nil},
		{free, nil, true, nil},
		{none, nil, true, nil},
		{course, nil, false, []string{"cavaletti", "jumps"}},
	} {
		if got := FitsEquipment(c.ex, c.available); got != c.fits {
			t.Errorf("FitsEquipment(%v, %v) = %v", c.ex.Equipment, c.available, got)
		}
		if got := MissingEquipment(c.ex, c.available); !slices.Equal(got, c.missing) {
			t.Errorf("MissingEquipment(%v, %v) = %v, want %v", c.ex.Equipment, c.available, got, c.missing)
		}
	}
}

func TestLocationTexts(t *testing.T) {
	for a, want := range map[Activity][2]string{
		ActivityHall: {"Halle", "in der Halle"}, ActivityArena: {"Platz", "auf dem Platz"},
		ActivityLunge: {"Longierhalle", "in der Longierhalle"}, ActivityJumping: {"Springplatz", "auf dem Springplatz"},
		ActivityGroundwork: {"Bodenarbeitsplatz", "am Bodenarbeitsplatz"},
	} {
		if n, w := LocationName(a); n != want[0] || w != want[1] {
			t.Errorf("LocationName(%s) = %q, %q", a, n, w)
		}
	}
	for in, want := range map[string]string{"": "", "A": "A", "A,B": "A und B", "A,B,C": "A, B und C"} {
		var words []string
		if in != "" {
			words = strings.Split(in, ",")
		}
		if got := JoinGerman(words); got != want {
			t.Errorf("JoinGerman(%v) = %q, want %q", words, got, want)
		}
	}
}

func TestGoalLevel(t *testing.T) {
	cases := []struct{ level, stored, want string }{
		{LevelBeginner, "", LevelIntermediate},
		{LevelIntermediate, "", LevelAdvanced},
		{LevelAdvanced, "", ""},
		{"", "", ""}, // an unknown level has no default goal
		{LevelBeginner, LevelAdvanced, LevelAdvanced},
		{LevelBeginner, LevelIntermediate, LevelIntermediate},
		{LevelIntermediate, LevelAdvanced, LevelAdvanced},
		{LevelIntermediate, LevelIntermediate, LevelAdvanced}, // a stored goal that is not above the level is ignored
		{LevelAdvanced, LevelIntermediate, ""},
		{LevelAdvanced, LevelAdvanced, ""},
		{"", LevelAdvanced, LevelAdvanced},
		{"", LevelIntermediate, LevelIntermediate},
		{LevelBeginner, LevelBeginner, LevelIntermediate},
		{LevelBeginner, "bogus", LevelIntermediate},
	}
	for _, c := range cases {
		if got := GoalLevel(c.level, c.stored); got != c.want {
			t.Errorf("GoalLevel(%q, %q) = %q, want %q", c.level, c.stored, got, c.want)
		}
	}
}

func lesson(id, library, level, phase string, requires ...string) Exercise {
	return Exercise{ID: id, Library: library, Level: level, Phase: phase, Title: "Lektion " + id, Requires: requires}
}

func rated3(ids ...string) []Session {
	var s []Session
	for _, id := range ids {
		s = append(s, Session{ExerciseID: id, FocusRating: 3})
	}
	return s
}

func ids(list []Exercise) string {
	var out []string
	for _, e := range list {
		out = append(out, e.ID)
	}
	return strings.Join(out, ",")
}

func TestProgress(t *testing.T) {
	lib := []Exercise{
		lesson("b1", "dressage", LevelBeginner, PartMain), // another level: not counted
		lesson("l1", "dressage", LevelIntermediate, PartMain),
		lesson("l2", "dressage", LevelIntermediate, PartMain, "l1"),
		lesson("l3", "dressage", LevelIntermediate, PhaseAny, "l2"),
		lesson("l4", "dressage", LevelIntermediate, PartWarmup),   // a warm-up is no lesson
		lesson("l5", "dressage", LevelIntermediate, PartCooldown), // neither is a cool-down
		lesson("l6", "cavaletti", LevelIntermediate, PartMain),    // not a home library
		lesson("l7", "dressage", LevelIntermediate, "", "l1"),     // no phase counts as main
		lesson("a1", "dressage", LevelAdvanced, PartMain),
	}
	home := []string{"dressage"}

	p := Progress(LevelIntermediate, home, lib, rated3("l1", "b1"))
	if p.Total != 4 || p.Mastered != 1 || p.Ready || p.Goal != LevelIntermediate {
		t.Fatalf("progress = %+v", p)
	}
	// l2 and l7 are open (their prerequisite is mastered); l3 waits for l2.
	if got := ids(p.Next); got != "l2,l7" {
		t.Errorf("next = %s, want l2,l7", got)
	}

	// Ratings below 3 do not master; unknown exercise ids and sessions without exercise are ignored.
	ratings := []Session{{ExerciseID: "l1", FocusRating: 2}, {ExerciseID: "l1", FocusRating: 1}, {FocusRating: 3}, {ExerciseID: "zzz", FocusRating: 3}}
	if p := Progress(LevelIntermediate, home, lib, ratings); p.Mastered != 0 || ids(p.Next) != "l1" {
		t.Errorf("progress = %+v, next %s", p, ids(p.Next))
	}

	// Another home library adds its lessons; advanced counts the advanced ones.
	if p := Progress(LevelIntermediate, []string{"dressage", "cavaletti"}, lib, nil); p.Total != 5 {
		t.Errorf("two home libraries: total = %d", p.Total)
	}
	if p := Progress(LevelAdvanced, home, lib, rated3("a1")); p.Total != 1 || p.Mastered != 1 || p.Ready {
		t.Errorf("advanced: %+v (total below 3 is never ready)", p)
	}

	// Next shows at most three, in the order of the library.
	var many []Exercise
	for i := 1; i <= 6; i++ {
		many = append(many, lesson(fmt.Sprintf("m%d", i), "dressage", LevelIntermediate, PartMain))
	}
	if got := ids(Progress(LevelIntermediate, home, many, rated3("m1")).Next); got != "m2,m3,m4" {
		t.Errorf("next of six = %s", got)
	}
	if p := Progress(LevelIntermediate, home, nil, nil); p.Total != 0 || p.Ready || len(p.Next) != 0 {
		t.Errorf("empty library: %+v", p)
	}
}

func TestProgressReady(t *testing.T) {
	build := func(total, mastered int) ProgressResult {
		var lib []Exercise
		var done []string
		for i := 0; i < total; i++ {
			id := fmt.Sprintf("x%02d", i)
			lib = append(lib, lesson(id, "dressage", LevelIntermediate, PartMain))
			if i < mastered {
				done = append(done, id)
			}
		}
		return Progress(LevelIntermediate, []string{"dressage"}, lib, rated3(done...))
	}
	for _, c := range []struct {
		total, mastered int
		ready           bool
	}{
		{12, 10, true}, {12, 9, false}, {10, 8, true}, {10, 7, false}, {5, 4, true}, {5, 3, false},
		{3, 3, true}, {3, 2, false}, {2, 2, false}, {1, 1, false}, {0, 0, false},
	} {
		if got := build(c.total, c.mastered).Ready; got != c.ready {
			t.Errorf("%d of %d mastered: ready = %v, want %v", c.mastered, c.total, got, c.ready)
		}
	}
}

func TestProgressTexts(t *testing.T) {
	until := time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC)
	p := ProgressResult{Goal: LevelIntermediate, Mastered: 7, Total: 12}
	if got := p.Text("Luna", nil); got != "Auf dem Weg zu L: 7 von 12 Lektionen gefestigt." {
		t.Errorf("default text = %q", got)
	}
	if got := p.Text("Luna", &until); got != "Ziel L bis 15.03.: noch 5 Lektionen." {
		t.Errorf("goal date text = %q", got)
	}
	if got := (ProgressResult{Goal: LevelAdvanced, Mastered: 11, Total: 12}).Text("Luna", &until); got != "Ziel M bis 15.03.: noch 1 Lektion." {
		t.Errorf("singular = %q", got)
	}
	if got := (ProgressResult{Goal: LevelAdvanced, Mastered: 0, Total: 1}).Text("Luna", nil); got != "Auf dem Weg zu M: 0 von 1 Lektion gefestigt." {
		t.Errorf("one lesson = %q", got)
	}
	ready := ProgressResult{Goal: LevelIntermediate, Mastered: 10, Total: 12, Ready: true}
	if got := ready.Text("Luna", nil); got != "Luna ist bereit für L: 10 von 12 Lektionen gefestigt." {
		t.Errorf("ready text = %q", got)
	}
	if got := ready.Text("Luna", &until); got != "Luna ist bereit für L: 10 von 12 Lektionen gefestigt." {
		t.Errorf("ready beats the date: %q", got)
	}
	if got := ready.LevelUpText("Luna"); got != "Luna hat 10 von 12 L-Lektionen gefestigt – Einstufung auf L anpassen?" {
		t.Errorf("level-up text = %q", got)
	}
}

func TestAlternatives(t *testing.T) {
	trot := Exercise{
		ID: "trot", Library: "cavaletti", Level: LevelIntermediate, Title: "Trabstangen", Phase: PartMain,
		Tags: []string{"takt", "rhythmus", "losgelassenheit"}, Equipment: []string{"poles"},
	}
	pool := []Exercise{
		trot,
		{ID: "a", Library: "cavaletti", Level: LevelBeginner, Title: "Tempi an der Longe", Phase: PartMain, Tags: []string{"takt", "rhythmus"}},
		{ID: "b", Library: "cavaletti", Level: LevelIntermediate, Title: "Zirkel ohne Stangen", Phase: PhaseAny, Tags: []string{"takt", "rhythmus", "losgelassenheit"}, Equipment: []string{"lunge"}},
		{ID: "c", Library: "cavaletti", Level: LevelBeginner, Title: "Schlangenlinien", Phase: PartMain, Tags: []string{"takt", "losgelassenheit"}},
		{ID: "d", Library: "cavaletti", Level: LevelAdvanced, Title: "Zu schwer", Phase: PartMain, Tags: []string{"takt", "rhythmus"}},
		{ID: "e", Library: "dressage", Level: LevelBeginner, Title: "Andere Bibliothek", Phase: PartMain, Tags: []string{"takt", "rhythmus"}},
		{ID: "f", Library: "cavaletti", Level: LevelBeginner, Title: "Anderer Teil", Phase: PartCooldown, Tags: []string{"takt", "rhythmus"}},
		{ID: "g", Library: "cavaletti", Level: LevelBeginner, Title: "Nur ein Tag", Phase: PartMain, Tags: []string{"takt"}},
		{ID: "h", Library: "cavaletti", Level: LevelBeginner, Title: "Mit Aufbau", Phase: PartMain, Tags: []string{"takt", "rhythmus"}, Equipment: []string{"cavaletti"}},
		{ID: "i", Library: "cavaletti", Level: "", Title: "Ohne Stufe", Phase: PartMain, Tags: []string{"takt", "rhythmus"}},
	}
	// Shared tags first (b has 3), then the level (easiest first; no level counts as beginner),
	// then the title.
	if got := ids(Alternatives(trot, pool, 5)); got != "b,i,c,a" {
		t.Errorf("alternatives = %s, want b,i,c,a", got)
	}
	if got := ids(Alternatives(trot, pool, 2)); got != "b,i" {
		t.Errorf("limit 2 = %s", got)
	}
	// An exercise without facility equipment needs no alternative.
	if got := Alternatives(pool[1], pool, 5); got != nil {
		t.Errorf("alternatives of an exercise without set-up = %v", got)
	}
	// A warm-up looks for warm-ups and exercises of any phase.
	warm := Exercise{ID: "w", Library: "cavaletti", Level: LevelBeginner, Phase: PartWarmup, Tags: []string{"takt", "rhythmus"}, Equipment: []string{"poles"}}
	pool2 := []Exercise{
		warm,
		{ID: "w1", Library: "cavaletti", Level: LevelBeginner, Title: "A", Phase: PartWarmup, Tags: []string{"takt", "rhythmus"}},
		{ID: "w2", Library: "cavaletti", Level: LevelBeginner, Title: "B", Phase: PhaseAny, Tags: []string{"takt", "rhythmus"}},
		{ID: "w3", Library: "cavaletti", Level: LevelBeginner, Title: "C", Phase: PartMain, Tags: []string{"takt", "rhythmus"}},
		{ID: "w4", Library: "cavaletti", Level: LevelIntermediate, Title: "D", Phase: PartWarmup, Tags: []string{"takt", "rhythmus"}},
	}
	if got := ids(Alternatives(warm, pool2, 5)); got != "w1,w2" {
		t.Errorf("warm-up alternatives = %s, want w1,w2", got)
	}
	// An exercise without phase counts as main.
	mainless := Exercise{ID: "m", Library: "cavaletti", Level: LevelBeginner, Tags: []string{"takt", "rhythmus"}, Equipment: []string{"poles"}}
	pool3 := []Exercise{mainless, {ID: "m1", Library: "cavaletti", Level: LevelBeginner, Title: "A", Phase: PartMain, Tags: []string{"takt", "rhythmus"}}}
	if got := ids(Alternatives(mainless, pool3, 5)); got != "m1" {
		t.Errorf("phase-less alternatives = %s", got)
	}
}
