package weekplan

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
)

// The unit plan of the week: the focus rotation over the days, what the model may propose and what
// the rules take over (docs/domains/training-planning.md, "Planner").

func onlyActivities(acts ...training.Activity) []training.AllowedActivity {
	var l []training.AllowedActivity
	for _, a := range acts {
		l = append(l, training.AllowedActivity{Activity: a, Mode: training.ModeOn})
	}
	return l
}

// unitWeek is a dressage horse that rides in the hall and the arena only, nothing done yet this
// week and no show, so that the focus and the exercise of a day are the only things that differ.
func unitWeek() Input {
	in := week()
	in.Profile.Shows = nil
	in.Profile.Allowed = onlyActivities(training.ActivityHall, training.ActivityArena)
	in.Recent = nil
	in.Level = training.LevelIntermediate
	in.AgeYears = 10
	in.Profile.Rhythm.RestDaysMin = 0
	in.Profile.Rhythm.SessionsMax = 0
	return in
}

func focusKeys(es []Entry) []string {
	var out []string
	for _, e := range es {
		if !e.Unit.Empty() {
			out = append(out, e.Unit.FocusKey)
		}
	}
	return out
}

func TestMergeRotatesTheFocusesOverTheWeek(t *testing.T) {
	in := unitWeek()
	in.Profile.Focuses = []training.FocusChoice{{Key: "losgelassenheit"}, {Key: "takt"}}
	es := Rules(in)
	keys := focusKeys(es)
	if len(keys) < 3 {
		t.Fatalf("units = %v", acts(es))
	}
	// Both focuses are trained, the one that was trained longest ago comes next: they alternate,
	// the first one starts.
	for i, k := range keys {
		if want := []string{"losgelassenheit", "takt"}[i%2]; k != want {
			t.Errorf("unit %d: focus %s, want %s (all: %v)", i+1, k, want, keys)
		}
	}
	if why := es[0].Unit.FocusWhy; why != "Schwerpunkt Losgelassenheit: noch nicht geübt." {
		t.Errorf("why = %q", why)
	}
	if why := es[1].Unit.FocusWhy; why != "Schwerpunkt Takt: noch nicht geübt." {
		t.Errorf("second why = %q", why)
	}
	// A later unit names the day it was last trained.
	if keys[2] == "losgelassenheit" && !strings.HasPrefix(es[2].Unit.FocusWhy, "Schwerpunkt Losgelassenheit: zuletzt ") {
		t.Errorf("third why = %q", es[2].Unit.FocusWhy)
	}
}

func TestMergeCountsTheDaysSomeoneElsePlanned(t *testing.T) {
	in := unitWeek()
	in.Profile.Focuses = []training.FocusChoice{{Key: "losgelassenheit"}, {Key: "takt"}}
	// Wednesday is planned by someone with the first focus: Thursday takes the other one.
	in.Days[2].Open, in.Days[2].Planned, in.Days[2].Minutes, in.Days[2].FocusKey = false, training.ActivityHall, 45, "losgelassenheit"
	es := Rules(in)
	if es[0].Unit.FocusKey != "takt" {
		t.Errorf("thursday = %+v", es[0].Unit)
	}
	// Without the planned focus the first one starts.
	in.Days[2].FocusKey = ""
	if es := Rules(in); es[0].Unit.FocusKey != "losgelassenheit" {
		t.Errorf("thursday without a planned focus = %+v", es[0].Unit)
	}
}

func TestMergeTakesTheModelsFocusWhenItIsACandidate(t *testing.T) {
	in := unitWeek()
	in.Profile.Focuses = []training.FocusChoice{{Key: "losgelassenheit"}, {Key: "takt"}}
	props := map[string]Proposal{
		"2026-09-30": {Activity: training.ActivityHall, Minutes: 45, FocusKey: "takt", Reason: "Takt"},
		"2026-10-01": {Activity: training.ActivityArena, Minutes: 45, FocusKey: "versammlung", Reason: "Versammlung"},
	}
	es := Merge(in, props)
	// takt is one of the candidates: taken although the rotation would start with the first one.
	if es[0].Source != SourceAI || es[0].Unit.FocusKey != "takt" {
		t.Errorf("wed = %+v", es[0])
	}
	// versammlung is no focus of the horse: the rotation chooses (losgelassenheit, takt was yesterday).
	if es[1].Source != SourceAI || es[1].Unit.FocusKey != "losgelassenheit" {
		t.Errorf("thu = %+v", es[1])
	}
}

func TestMergeRiderFocusesComeFirst(t *testing.T) {
	in := unitWeek()
	in.Profile.Focuses = []training.FocusChoice{{Key: "losgelassenheit"}}
	in.Days[2].RiderFocuses, in.Days[2].RiderName = []string{"takt"}, "Mia"
	es := Rules(in)
	if es[0].Unit.FocusKey != "takt" || es[0].Unit.FocusWhy != "Mias Schwerpunkt Takt." {
		t.Errorf("wed = %+v", es[0].Unit)
	}
	// The next day has no rider of its own: the horse's focus.
	if es[1].Unit.FocusKey != "losgelassenheit" {
		t.Errorf("thu = %+v", es[1].Unit)
	}
}

// exerciseLibrary is a small dressage library for the exercise pick of the week.
func exerciseLibrary() []Exercise {
	return []Exercise{
		{ID: "w1", Library: "dressage", Level: training.LevelBeginner, Title: "Schritt am langen Zügel", Phase: "warmup", Tags: []string{"losgelassenheit"}},
		{ID: "m1", Library: "dressage", Level: training.LevelBeginner, Title: "Schlangenlinien", Phase: "main", Tags: []string{"losgelassenheit", "biegung"}},
		{ID: "m2", Library: "dressage", Level: training.LevelBeginner, Title: "Zirkel verkleinern", Phase: "main", Tags: []string{"losgelassenheit", "biegung"}},
		{ID: "c1", Library: "dressage", Level: training.LevelBeginner, Title: "Dehnungshaltung", Phase: "cooldown", Tags: []string{"losgelassenheit", "mobilitaet"}},
	}
}

func TestMergeUsedExercisesCountForTheNextDays(t *testing.T) {
	in := unitWeek()
	in.Profile.Focuses = []training.FocusChoice{{Key: "losgelassenheit"}}
	in.Exercises = exerciseLibrary()
	es := Rules(in)
	if es[0].Exercise == nil || es[0].Exercise.ID != "m1" || es[0].Unit.Warmup.Exercise.ID != "w1" || es[0].Unit.Cooldown.Exercise.ID != "c1" {
		t.Fatalf("wed = %+v", es[0].Unit)
	}
	// m1 was used yesterday (-3), the tie with m2 is gone: m2.
	if es[1].Exercise == nil || es[1].Exercise.ID != "m2" {
		t.Errorf("thu main = %+v", es[1].Exercise)
	}
	// The warm-up of yesterday is out for the next day, too, but nothing else fits the warm-up.
	if es[1].Unit.Warmup.Exercise == nil {
		t.Errorf("thu warm-up = %+v", es[1].Unit.Warmup)
	}
	// Entry.Exercise is the main exercise of the unit.
	for _, e := range es {
		if e.Unit.Main.Exercise != e.Exercise && (e.Unit.Main.Exercise == nil || e.Exercise == nil || e.Unit.Main.Exercise.ID != e.Exercise.ID) {
			t.Errorf("%s: exercise %+v, main %+v", e.Date.Format(dateLayout), e.Exercise, e.Unit.Main.Exercise)
		}
	}
}

func TestMergeReplacesExercisesTheRulesDoNotAllow(t *testing.T) {
	in := unitWeek()
	in.Profile.Focuses = []training.FocusChoice{{Key: "losgelassenheit"}}
	lib := exerciseLibrary()
	lib = append(lib,
		Exercise{ID: "g1", Library: "dressage", Level: training.LevelBeginner, Title: "Gelassenheitsparcours", Phase: "main", Tags: []string{"gelassenheit"}},
		Exercise{ID: "h1", Library: "dressage", Level: training.LevelIntermediate, Title: "Schulterherein", Phase: "main", Tags: []string{"losgelassenheit", "seitengaenge"}, Requires: []string{"m2"}})
	in.Exercises = lib
	byID := func(id string) *Exercise {
		for i := range in.Exercises {
			if in.Exercises[i].ID == id {
				return &in.Exercises[i]
			}
		}
		return nil
	}
	props := map[string]Proposal{
		// m2 instead of the rules' m1; the tag of g1 does not meet the focus; h1 needs m2 mastered.
		"2026-09-30": {Activity: training.ActivityHall, Minutes: 45, Main: byID("m2"), Reason: "m2"},
		"2026-10-01": {Activity: training.ActivityHall, Minutes: 45, Main: byID("g1"), Reason: "g1"},
		"2026-10-02": {Activity: training.ActivityHall, Minutes: 45, Main: byID("h1"), Reason: "h1"},
	}
	es := Merge(in, props)
	if es[0].Exercise == nil || es[0].Exercise.ID != "m2" || es[0].Replaced != "" || es[0].Source != SourceAI {
		t.Errorf("wed: %+v (replaced %q)", es[0].Exercise, es[0].Replaced)
	}
	if es[1].Exercise == nil || es[1].Exercise.ID == "g1" || es[1].Source != SourceAI ||
		es[1].Replaced != "Übung „Gelassenheitsparcours“ ersetzt: passt nicht zum Schwerpunkt Losgelassenheit." {
		t.Errorf("thu: %+v (replaced %q)", es[1].Exercise, es[1].Replaced)
	}
	if es[2].Exercise == nil || es[2].Exercise.ID == "h1" ||
		es[2].Replaced != "Übung „Schulterherein“ ersetzt: setzt Übungen voraus, die noch nicht sitzen." {
		t.Errorf("fri: %+v (replaced %q)", es[2].Exercise, es[2].Replaced)
	}
	// With m2 mastered h1 is allowed (one level above the horse's, all prerequisites sit): it is
	// taken, although the rules would pick another.
	lib2 := append([]Exercise(nil), lib...)
	lib2[2].Mastered = true
	in.Exercises = lib2
	in.Level = training.LevelBeginner
	es = Merge(in, map[string]Proposal{"2026-09-30": {Activity: training.ActivityHall, Minutes: 45, Main: byID("h1"), Reason: "h1"}})
	if es[0].Exercise == nil || es[0].Exercise.ID != "h1" || es[0].Replaced != "" {
		t.Errorf("above the level with its prerequisites: %+v (replaced %q)", es[0].Exercise, es[0].Replaced)
	}
}

func TestMergeThePlanOfTheModelFollowsTheStructureRules(t *testing.T) {
	hall := training.ActivityHall
	hallProposal := func(minutes int) map[string]Proposal {
		return map[string]Proposal{"2026-09-30": {Activity: hall, Minutes: minutes, Reason: "Halle"}}
	}
	cases := []struct {
		name string
		f    func(*Input)
		want int // minutes of the accepted proposal
	}{
		{"a build week", func(in *Input) { in.Block = recommend.Block{Week: 3} }, 60},
		{"the Entlastungswoche has no demanding unit", func(in *Input) { in.Block = recommend.Block{Week: 4, Deload: true} }, 56},
		{"the first week back is light", func(in *Input) { d := today; in.RestartFrom = &d }, 37},
		{"the second week back has nothing demanding", func(in *Input) { d := day(-7); in.RestartFrom = &d }, 56},
		{"a young horse has at most 30 minutes", func(in *Input) { in.AgeYears = 4 }, 30},
	}
	for _, c := range cases {
		in := unitWeek()
		c.f(&in)
		es := Merge(in, hallProposal(60))
		if es[0].Source != SourceAI || es[0].Recommendation.Minutes != c.want {
			t.Errorf("%s: %+v, want %d minutes", c.name, es[0].Recommendation, c.want)
		}
		// The unit has the minutes of the accepted proposal.
		u := es[0].Unit
		if got := u.Warmup.Minutes + u.Main.Minutes + u.Cooldown.Minutes; got != c.want {
			t.Errorf("%s: the unit has %d minutes", c.name, got)
		}
	}
	// The same horse planned by the rules: week 1 back has light units only.
	in := unitWeek()
	d := today
	in.RestartFrom = &d
	for _, e := range Rules(in) {
		if e.Recommendation.Activity != training.ActivityRest && recommend.UnitLoad(e.Recommendation.Activity, e.Recommendation.Minutes) >= 30 {
			t.Errorf("%s: %+v is not light", e.Date.Format(dateLayout), e.Recommendation)
		}
	}
}

func TestMergeEquipmentOfTheLocations(t *testing.T) {
	poles := Exercise{ID: "p1", Library: "cavaletti", Level: training.LevelBeginner, Title: "Trabstangen", Phase: "main",
		Tags: []string{"losgelassenheit", "biegung", "takt"}, Equipment: []string{"poles"}}
	plain := Exercise{ID: "m1", Library: "dressage", Level: training.LevelBeginner, Title: "Schlangenlinien", Phase: "main", Tags: []string{"losgelassenheit"}}
	in := unitWeek()
	in.Profile.Focuses = []training.FocusChoice{{Key: "losgelassenheit"}}
	in.Exercises = []Exercise{poles, plain}
	in.Equipment = training.StableEquipment{training.ActivityHall: {Equipment: []string{"poles"}}}
	es := Rules(in)
	if es[0].Exercise == nil || es[0].Exercise.ID != "p1" || es[0].Unit.EquipmentNote != "" {
		t.Fatalf("with poles: %+v", es[0].Unit)
	}

	// The hall has no poles: the plain exercise instead, with the reason.
	in.Equipment = training.StableEquipment{training.ActivityHall: {Equipment: []string{}}}
	in.Days[2].Exclude = []training.Activity{training.ActivityArena}
	es = Rules(in)
	u := es[0].Unit
	if es[0].Recommendation.Activity != training.ActivityHall || es[0].Exercise == nil || es[0].Exercise.ID != "m1" ||
		u.Main.InsteadOf == nil || u.Main.InsteadOf.ID != "p1" ||
		u.Main.Why != "Ohne Stangen in der Halle: Schlangenlinien statt Trabstangen." || u.EquipmentNote != u.Main.Why {
		t.Errorf("without poles: %+v", u.Main)
	}

	// A shared hall is planned without set-up unless the day says it can be set up.
	in.Equipment = training.StableEquipment{training.ActivityHall: {Equipment: []string{"poles"}, Shared: true}}
	es = Rules(in)
	if es[0].Exercise.ID != "m1" || es[0].Unit.Main.Why != "Halle geteilt, ohne Aufbau: Schlangenlinien statt Trabstangen." {
		t.Errorf("shared hall: %+v", es[0].Unit.Main)
	}
	yes, no := true, false
	in.Days[2].EquipmentOK = &yes
	if es = Rules(in); es[0].Exercise.ID != "p1" || es[0].Unit.EquipmentNote != "" {
		t.Errorf("shared hall, set up: %+v", es[0].Unit)
	}
	// A day without set-up in a hall with poles.
	in.Equipment = training.StableEquipment{training.ActivityHall: {Equipment: []string{"poles"}}}
	in.Days[2].EquipmentOK = &no
	es = Rules(in)
	if es[0].Exercise.ID != "m1" || es[0].Unit.Main.Why != "Heute ohne Aufbau: Schlangenlinien statt Trabstangen." {
		t.Errorf("no set-up: %+v", es[0].Unit.Main)
	}

	// The model proposes the pole exercise for the hall without poles: replaced, with the reason.
	in.Equipment = training.StableEquipment{training.ActivityHall: {Equipment: []string{}}}
	in.Days[2].EquipmentOK = nil
	props := map[string]Proposal{"2026-09-30": {Activity: training.ActivityHall, Minutes: 45, Main: &in.Exercises[0], Reason: "Stangen"}}
	es = Merge(in, props)
	if es[0].Source != SourceAI || es[0].Exercise.ID != "m1" ||
		!strings.Contains(es[0].Replaced, "Übung „Trabstangen“ ersetzt: braucht Stangen.") {
		t.Errorf("proposal: %+v (replaced %q)", es[0].Exercise, es[0].Replaced)
	}
}

func TestMergeLeavesRestAndWalkerWithoutAUnit(t *testing.T) {
	in := unitWeek()
	in.Profile.Allowed = onlyActivities(training.ActivityWalker, training.ActivityHall)
	in.Exercises = exerciseLibrary()
	props := map[string]Proposal{
		"2026-09-30": {Activity: training.ActivityWalker, Minutes: 30, FocusKey: "takt", Main: &in.Exercises[1], Reason: "Führanlage"},
		"2026-10-01": {Activity: training.ActivityRest, Reason: "Ruhe", Main: &in.Exercises[1]},
	}
	es := Merge(in, props)
	for _, e := range es[:2] {
		if !e.Unit.Empty() || e.Exercise != nil || e.Unit.FocusKey != "" {
			t.Errorf("%s: %+v", e.Recommendation.Activity, e.Unit)
		}
	}
	if es[0].Recommendation.Activity != training.ActivityWalker || es[1].Recommendation.Activity != training.ActivityRest {
		t.Errorf("activities = %v", acts(es))
	}
}

func TestMergeDoesNotUseTheModelsExerciseOfARejectedProposal(t *testing.T) {
	in := unitWeek()
	in.Exercises = exerciseLibrary()
	in.Days[2].Exclude = []training.Activity{training.ActivityHall}
	props := map[string]Proposal{"2026-09-30": {Activity: training.ActivityHall, Minutes: 45, Main: &in.Exercises[2], Reason: "Halle"}}
	es := Merge(in, props)
	if es[0].Source != SourceRules || es[0].Replaced != "Vom Besitzer abgelehnt." {
		t.Fatalf("wed = %+v", es[0])
	}
	if es[0].Exercise == nil || es[0].Exercise.ID != "m1" {
		t.Errorf("the rules pick the main exercise: %+v", es[0].Exercise)
	}
}

// --- the prompt -----------------------------------------------------------------------------------

func promptMessage(t *testing.T, in Input) map[string]any {
	t.Helper()
	var msg map[string]any
	if err := json.Unmarshal([]byte(Prompt(in)), &msg); err != nil {
		t.Fatal(err)
	}
	return msg
}

func TestPromptCarriesPhaseBlockFocusesAndEquipment(t *testing.T) {
	in := unitWeek()
	in.Profile.Allowed = allOn()
	in.Level = training.LevelIntermediate
	in.Profile.GoalLevel = ""
	in.Block = recommend.Block{Week: 4, Deload: true}
	d := day(-3)
	in.RestartFrom = &d
	until := day(5)
	past := day(-1)
	in.Profile.Focuses = []training.FocusChoice{{Key: "losgelassenheit"}, {Key: "versammlung", Until: &until}, {Key: "takt", Until: &past}, {Key: "bogus"}}
	in.Equipment = training.StableEquipment{
		training.ActivityHall:  {Equipment: []string{"poles", "cavaletti", "lunge"}},
		training.ActivityArena: {Equipment: []string{"poles"}, Shared: true},
	}
	in.Days[2].RiderFocuses, in.Days[2].RiderName = []string{"takt", "bogus"}, "Mia"
	in.Days[3].EquipmentOK = new(bool)
	in.Days[1].Open, in.Days[1].Planned, in.Days[1].Minutes, in.Days[1].FocusKey = false, training.ActivityHall, 40, "anlehnung"
	in.Recent = []training.Session{{Day: day(-1), Activity: training.ActivityHall, Minutes: 40, Load: 32, FocusKey: "takt"}}
	msg := promptMessage(t, in)
	if msg["phase"] != "schubkraft" || msg["goal_level"] != "advanced" {
		t.Errorf("phase %v, goal %v", msg["phase"], msg["goal_level"])
	}
	if b := msg["block"].(map[string]any); b["week"] != float64(4) || b["deload"] != true {
		t.Errorf("block = %v", b)
	}
	if msg["restart_week"] != float64(1) {
		t.Errorf("restart_week = %v", msg["restart_week"])
	}
	// The catalog: all fifteen focuses with their tags and activities.
	catalog := msg["focus_catalog"].([]any)
	if len(catalog) != len(training.Focuses) {
		t.Fatalf("focus_catalog has %d entries", len(catalog))
	}
	first := catalog[0].(map[string]any)
	if first["key"] != "takt" || len(first["tags"].([]any)) != 2 || len(first["fits"].([]any)) != 4 {
		t.Errorf("first focus = %v", first)
	}
	// The horse's focuses in order, without the expired and the unknown one, with the flag above the phase.
	hf := msg["horse_focuses"].([]any)
	if len(hf) != 2 || hf[0].(map[string]any)["key"] != "losgelassenheit" || hf[0].(map[string]any)["above_phase"] != nil ||
		hf[1].(map[string]any)["key"] != "versammlung" || hf[1].(map[string]any)["above_phase"] != true {
		t.Errorf("horse_focuses = %v", hf)
	}
	// Equipment per activity: the facility equipment only, nothing available in a shared location.
	byActivity := map[string]map[string]any{}
	for _, a := range msg["activities"].([]any) {
		m := a.(map[string]any)
		byActivity[m["activity"].(string)] = m
	}
	if hall := byActivity["hall"]; len(hall["equipment"].([]any)) != 2 || len(hall["available_equipment"].([]any)) != 2 || hall["shared"] != nil {
		t.Errorf("hall = %v", hall)
	}
	if arena := byActivity["arena"]; len(arena["equipment"].([]any)) != 1 || len(arena["available_equipment"].([]any)) != 0 || arena["shared"] != true {
		t.Errorf("arena = %v", arena)
	}
	if lunge := byActivity["lunge"]; len(lunge["equipment"].([]any)) != 0 || lunge["available_equipment"] == nil {
		t.Errorf("lunge has a location without equipment: %v", lunge)
	}
	if hack := byActivity["hack"]; hack["equipment"] != nil || hack["available_equipment"] != nil {
		t.Errorf("hack has no location: %v", hack)
	}
	days := msg["days"].([]any)
	wed, thu := days[2].(map[string]any), days[3].(map[string]any)
	if rf := wed["rider_focuses"].([]any); len(rf) != 1 || rf[0] != "takt" {
		t.Errorf("rider_focuses = %v", wed["rider_focuses"])
	}
	if thu["equipment_ok"] != false || wed["equipment_ok"] != nil {
		t.Errorf("equipment_ok: wed %v, thu %v", wed["equipment_ok"], thu["equipment_ok"])
	}
	if tue := days[1].(map[string]any); tue["planned"] != "hall" || tue["focus_key"] != "anlehnung" {
		t.Errorf("tuesday = %v", tue)
	}
	if h := msg["history"].([]any)[0].(map[string]any); h["focus_key"] != "takt" {
		t.Errorf("history = %v", h)
	}
	// A horse without blocks, goal or return to work does not get the fields.
	plain := promptMessage(t, unitWeek())
	for _, k := range []string{"block", "restart_week"} {
		if _, has := plain[k]; has {
			t.Errorf("%s without a value: %v", k, plain[k])
		}
	}
	if plain["phase"] != "schubkraft" {
		t.Errorf("phase = %v", plain["phase"])
	}
}

func TestPromptExercisesCarryTheirMetadata(t *testing.T) {
	in := unitWeek()
	in.Exercises = []Exercise{
		{ID: "a", Library: "dressage", Level: training.LevelBeginner, Title: "Anreiten", Phase: "warmup", Minutes: 8, Intensity: "light", Tags: []string{"takt"}, Mastered: true, NextID: "c"},
		{ID: "own", Library: "dressage", Title: "Mein Geheimtipp", Own: true},
		{ID: "c", Library: "cavaletti", Level: training.LevelIntermediate, Title: "Cavaletti im Trab", Phase: "main", Minutes: 12, Intensity: "demanding",
			Requires: []string{"a", "own", "unknown"}, Equipment: []string{"cavaletti", "whip"}, Rated: true},
	}
	msg := promptMessage(t, in)
	ex := msg["exercises"].([]any)
	if len(ex) != 2 {
		t.Fatalf("own exercises stay on the server: %v", ex)
	}
	a, c := ex[0].(map[string]any), ex[1].(map[string]any)
	if a["key"] != "E1" || a["phase"] != "warmup" || a["minutes"] != float64(8) || a["intensity"] != "light" || a["mastered"] != true ||
		a["next"] != "E2" || a["practised"] != nil {
		t.Errorf("a = %v", a)
	}
	// The keys count the visible exercises only; requires lists keys of visible exercises, the
	// equipment the facility equipment only.
	if c["key"] != "E2" || c["practised"] != true || len(c["requires"].([]any)) != 1 || c["requires"].([]any)[0] != "E1" ||
		len(c["equipment"].([]any)) != 1 || c["equipment"].([]any)[0] != "cavaletti" {
		t.Errorf("c = %v", c)
	}
	if strings.Contains(Prompt(in), "Geheimtipp") || strings.Contains(Prompt(in), `"own"`) {
		t.Error("an own exercise reached the prompt")
	}
	// Parse sees the same keys: E2 is the cavaletti exercise, E3 (the own exercise would be the
	// second by position) is unknown.
	got, err := Parse(in, `{"days":[{"day":"mi","activity":"hall","minutes":45,"main":"E2"},{"day":"do","activity":"hall","minutes":45,"main":"E3"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if p := got["2026-09-30"]; p.Main == nil || p.Main.ID != "c" {
		t.Errorf("wed = %+v", p)
	}
	if p := got["2026-10-01"]; p.Main != nil {
		t.Errorf("thu = %+v", p)
	}
}

func TestPromptWithUnitPlanLeaksNothing(t *testing.T) {
	in := unitWeek()
	in.Profile.HorseName, in.Profile.Level = "Luna", "L bei Anna Muster"
	in.Profile.Shows = []training.Show{{Date: day(3), Name: "Turnier bei Familie Muster"}}
	in.Profile.Allowed[0] = training.AllowedActivity{Activity: training.ActivityHall, Mode: training.ModeConditional, Note: "nur mit Anna"}
	until := day(9)
	in.Profile.Focuses = []training.FocusChoice{{Key: "losgelassenheit", Until: &until}}
	in.Days[2].RiderFocuses, in.Days[2].RiderName = []string{"takt"}, "Mia"
	in.Exercises = []Exercise{
		{ID: "id-1", Library: "dressage", Level: training.LevelBeginner, Title: "Schlangenlinien", Phase: "main", Tags: []string{"biegung"}},
		{ID: "id-own", Library: "dressage", Title: "Annas Lieblingsübung", Own: true, Tags: []string{"biegung"}},
	}
	in.Recent = []training.Session{{Day: day(-1), Activity: training.ActivityHall, Minutes: 40, Load: 32, ExerciseID: "id-own", FocusKey: "takt"}}
	p := Prompt(in)
	for _, leak := range []string{"Luna", "Anna", "Muster", "Mia", "Lieblings", "id-1", "id-own", "2026", "L bei", "Turnier bei"} {
		if strings.Contains(p, leak) {
			t.Errorf("prompt contains %q: %s", leak, p)
		}
	}
	// The rider's name is not even part of the entry the model gets back.
	for _, want := range []string{`"rider_focuses":["takt"]`, `"horse_focuses":[{"key":"losgelassenheit"}]`} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt lacks %s", want)
		}
	}
	// The history does not name an own exercise: only keys of the global library.
	if h := promptMessage(t, in)["history"].([]any)[0].(map[string]any); h["exercise"] != nil || h["focus_key"] != "takt" {
		t.Errorf("history = %v", h)
	}
}

func TestSystemPromptDescribesTheUnitPlan(t *testing.T) {
	for _, want := range []string{
		"focus_catalog", "horse_focuses", "above_phase", "rider_focuses", "restart_week", "Entlastungswoche",
		"warmup", "cooldown", `"focus_key":"<catalog key or null>"`, `"warmup":"<key or null>"`, `"cooldown":"<key or null>"`,
		"available_equipment", "equipment_ok", "requires", "goal_level", "vertrauen", "gelaende", "practised",
		"At most 2 demanding units per week", "at most 30 minutes", "never the same activity on two days in a row",
	} {
		if !strings.Contains(SystemPrompt, want) {
			t.Errorf("the system prompt lacks %q", want)
		}
	}
	// The answer format of the earlier prompt is gone.
	if strings.Contains(SystemPrompt, `"exercise":"<key or null>"`) {
		t.Error("the system prompt still asks for the key \"exercise\"")
	}
}
