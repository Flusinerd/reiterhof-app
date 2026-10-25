package recommend

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// exr builds an exercise of the test library; the title is "Titel <id>".
func exr(id, library, level, phase string, tags ...string) training.Exercise {
	return training.Exercise{ID: id, Library: library, Level: level, Phase: phase, Title: "Titel " + id, Tags: tags}
}

// pickLib is a small dressage library. For the focus losgelassenheit (tags losgelassenheit,
// biegung) the scores are: Lösen w1 4, w2 2; Arbeiten m1 6, m2 3, m3 3; Entspannen c1 5, c2 2.
func pickLib() []training.Exercise {
	return []training.Exercise{
		exr("w1", "dressage", "beginner", "warmup", "losgelassenheit", "mobilitaet"),
		exr("w2", "dressage", "beginner", "warmup", "takt"),
		exr("m1", "dressage", "beginner", "main", "losgelassenheit", "biegung"),
		exr("m2", "dressage", "beginner", "main", "biegung"),
		exr("m3", "dressage", "intermediate", "main", "biegung"),
		exr("c1", "dressage", "beginner", "cooldown", "losgelassenheit", "mobilitaet"),
		exr("c2", "dressage", "beginner", "any", "mobilitaet"),
	}
}

func pickIn(lib []training.Exercise) Input {
	in := unitBase("losgelassenheit")
	in.Exercises = lib
	return in
}

// ids returns the exercise ids of the warm-up, the main part and the cool-down ("" for none).
func partIDs(u Unit) [3]string {
	id := func(p Part) string {
		if p.Exercise == nil {
			return ""
		}
		return p.Exercise.ID
	}
	return [3]string{id(u.Warmup), id(u.Main), id(u.Cooldown)}
}

func mutate(lib []training.Exercise, id string, f func(*training.Exercise)) []training.Exercise {
	out := append([]training.Exercise(nil), lib...)
	for i := range out {
		if out[i].ID == id {
			f(&out[i])
		}
	}
	return out
}

func session(offset int, id string, rating int) training.Session {
	return training.Session{Day: day(offset), Activity: training.ActivityHall, Minutes: 40, Load: 32, ExerciseID: id, FocusRating: rating}
}

func TestPickPlainUnit(t *testing.T) {
	u := PlanUnit(pickIn(pickLib()), training.ActivityHall, 45, Hints{})
	if got := partIDs(u); got != [3]string{"w1", "m1", "c1"} {
		t.Fatalf("parts = %v", got)
	}
	// The minutes are the unit split; the exercise is part of the part, nothing needs explaining.
	if u.Warmup.Minutes != 15 || u.Main.Minutes != 22 || u.Cooldown.Minutes != 8 {
		t.Errorf("minutes = %d/%d/%d", u.Warmup.Minutes, u.Main.Minutes, u.Cooldown.Minutes)
	}
	for _, p := range []Part{u.Warmup, u.Main, u.Cooldown} {
		if p.Why != "" || p.InsteadOf != nil {
			t.Errorf("part %+v needs no explanation", p)
		}
	}
	if u.Primary().ID != "m1" || !reflect.DeepEqual(u.Exercises(), []string{"m1", "w1", "c1"}) || u.EquipmentNote != "" || u.Replaced != "" {
		t.Errorf("unit = %+v", u)
	}
	if again := PlanUnit(pickIn(pickLib()), training.ActivityHall, 45, Hints{}); !reflect.DeepEqual(u, again) {
		t.Error("not deterministic")
	}
}

func TestPickScoresOfTheMainPart(t *testing.T) {
	lib := pickLib()
	next := func(l []training.Exercise) []training.Exercise {
		return mutate(l, "m1", func(e *training.Exercise) { e.NextID = "m2" })
	}
	masteredM1 := mutate(lib, "m1", func(e *training.Exercise) { e.Mastered = true })
	cases := []struct {
		name string
		lib  []training.Exercise
		hist []training.Session
		unit int // minutes
		want string
	}{
		{"tags: two shared beat one", lib, nil, 45, "m1"},
		{"mastered counts -2, still ahead", mutate(lib, "m1", func(e *training.Exercise) { e.Mastered = true }), nil, 45, "m1"},
		{"the follow-up of a mastered exercise +2", mutate(next(lib), "m1", func(e *training.Exercise) { e.Mastered = true }), nil, 45, "m2"},
		{"used yesterday -3, rated hard 4 days ago +2",
			lib, []training.Session{session(-1, "m1", 3), session(-4, "m2", 1)}, 45, "m2"},
		{"used yesterday alone: a tie, the order of the library", lib, []training.Session{session(-1, "m1", 0)}, 45, "m1"},
		{"used 2 days ago still counts", lib, []training.Session{session(-2, "m1", 0), session(-5, "m2", 1)}, 45, "m2"},
		{"used 3 days ago is fine again", lib, []training.Session{session(-3, "m1", 0), session(-5, "m2", 1)}, 45, "m1"},
		// m1 is mastered (4), m2 and m3 score 3 each: only the retry (+2) lifts m2 above m1.
		{"rated hard 3 days ago +2", masteredM1, []training.Session{session(-3, "m2", 1)}, 45, "m2"},
		{"rated hard 2 days ago is too early for a retry and counts as used", masteredM1, []training.Session{session(-2, "m2", 1)}, 45, "m1"},
		{"rated hard 7 days ago +2", masteredM1, []training.Session{session(-7, "m2", 1)}, 45, "m2"},
		{"rated hard 8 days ago is over", masteredM1, []training.Session{session(-8, "m2", 1)}, 45, "m1"},
		{"the rating belongs to the logged exercise, not to the others of the unit",
			masteredM1, []training.Session{{Day: day(-5), Activity: training.ActivityHall, Minutes: 40, Load: 32, ExerciseID: "m3", Exercises: []string{"m2"}, FocusRating: 1}}, 45, "m3"},
		{"rated better +1.5", mutate(lib, "m1", func(e *training.Exercise) { e.Mastered = true }),
			[]training.Session{session(-5, "m3", 2)}, 45, "m3"},
		{"rated better 14 days ago is over", mutate(lib, "m1", func(e *training.Exercise) { e.Mastered = true }),
			[]training.Session{session(-14, "m3", 2)}, 45, "m1"},
		{"a demanding exercise on a light unit -2",
			mutate(append(append([]training.Exercise(nil), lib...), exr("m4", "dressage", "beginner", "main", "losgelassenheit", "biegung")), "m1",
				func(e *training.Exercise) { e.Intensity = training.ExerciseDemanding }), nil, 30, "m4"},
		{"a demanding exercise on a normal unit is fine",
			mutate(append(append([]training.Exercise(nil), lib...), exr("m4", "dressage", "beginner", "main", "losgelassenheit", "biegung")), "m1",
				func(e *training.Exercise) { e.Intensity = training.ExerciseDemanding }), nil, 45, "m1"},
		{"a light exercise on a demanding unit -2",
			mutate(append(append([]training.Exercise(nil), lib...), exr("m4", "dressage", "beginner", "main", "losgelassenheit", "biegung")), "m1",
				func(e *training.Exercise) { e.Intensity = training.ExerciseLight }), nil, 60, "m4"},
		{"no shared tag: only when nothing else fits",
			[]training.Exercise{exr("a", "dressage", "beginner", "main", "kraft"), exr("b", "dressage", "beginner", "main", "kraft")}, nil, 45, "a"},
		{"a tagged exercise beats an untagged one with a better score",
			[]training.Exercise{mutate(lib, "m2", func(e *training.Exercise) { e.Tags = []string{"kraft"}; e.NextID = "" })[3],
				mutate(lib, "m1", func(e *training.Exercise) { e.Tags = []string{"biegung"}; e.Mastered = true })[2]}, nil, 45, "m1"},
	}
	for _, c := range cases {
		in := pickIn(c.lib)
		in.Recent = c.hist
		u := PlanUnit(in, training.ActivityHall, c.unit, Hints{})
		if got := partIDs(u)[1]; got != c.want {
			t.Errorf("%s: main = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPickWarmupAndCooldown(t *testing.T) {
	lib := pickLib()
	senior := mutate(mutate(lib, "w1", func(e *training.Exercise) { e.Tags = []string{"losgelassenheit"}; e.Suits = nil }),
		"w2", func(e *training.Exercise) { e.Tags = []string{"takt"}; e.Suits = []string{"senior"} })
	cases := []struct {
		name string
		lib  []training.Exercise
		in   func(*Input)
		want [3]string // warm-up, main, cool-down
	}{
		{"tags of losgelassenheit, takt and mobilitaet count 2 each, a cool-down exercise +1", lib, nil, [3]string{"w1", "m1", "c1"}},
		{"used yesterday -3", lib, func(in *Input) { in.Recent = []training.Session{session(-1, "w1", 0)} }, [3]string{"w2", "m1", "c1"}},
		{"used 2 days ago is no penalty for the warm-up", lib, func(in *Input) { in.Recent = []training.Session{session(-2, "w1", 0)} }, [3]string{"w1", "m1", "c1"}},
		{"suits a senior +1: takt and the suit (3) beat losgelassenheit (2)", senior, func(in *Input) { in.AgeYears = 20 }, [3]string{"w2", "m1", "c1"}},
		{"a lone exercise for any part goes to the main part, which is picked first",
			[]training.Exercise{exr("x", "dressage", "beginner", "any", "mobilitaet")}, nil, [3]string{"", "x", ""}},
	}
	for _, c := range cases {
		in := pickIn(c.lib)
		if c.in != nil {
			c.in(&in)
		}
		if got := partIDs(PlanUnit(in, training.ActivityHall, 45, Hints{})); got != c.want {
			t.Errorf("%s: parts = %v, want %v", c.name, got, c.want)
		}
	}
	// An exercise is used once in a unit: the exercise for any part that is best for two parts goes
	// to the main part (picked first), the others take the next best.
	best := exr("x", "dressage", "beginner", "any", "losgelassenheit", "biegung", "mobilitaet", "takt")
	both := mutate(append(pickLib(), best), "m1", func(e *training.Exercise) { e.Mastered = true })
	if got := partIDs(PlanUnit(pickIn(both), training.ActivityHall, 45, Hints{})); got != [3]string{"w1", "x", "c1"} {
		t.Errorf("parts = %v, want x as the main exercise only", got)
	}
	// A suitable exercise of a young horse.
	young := mutate(pickLib(), "m2", func(e *training.Exercise) {
		e.Suits = []string{"young"}
		e.Tags = []string{"losgelassenheit", "biegung"}
	})
	in := pickIn(young)
	in.AgeYears = 4
	if got := partIDs(PlanUnit(in, training.ActivityHall, 30, Hints{}))[1]; got != "m2" {
		t.Errorf("young horse main = %q, want the exercise that suits it (m2: 6+1 against m1: 6)", got)
	}
}

func TestPickLevelsAndPrerequisites(t *testing.T) {
	above := func(extra ...string) []training.Exercise {
		return []training.Exercise{
			exr("w1", "dressage", "beginner", "warmup", "losgelassenheit"),
			func() training.Exercise {
				e := exr("g1", "dressage", "intermediate", "main", "losgelassenheit", "biegung")
				e.Requires = extra
				return e
			}(),
			exr("c1", "dressage", "beginner", "cooldown", "mobilitaet"),
		}
	}
	beginner := func(lib []training.Exercise) Input {
		in := pickIn(lib)
		in.Level = training.LevelBeginner
		return in
	}
	main := func(in Input) string { return partIDs(PlanUnit(in, training.ActivityHall, 45, Hints{}))[1] }

	// One level above the horse's is a candidate when all its prerequisites are mastered.
	if got := main(beginner(above())); got != "g1" {
		t.Errorf("one level above without prerequisites: %q", got)
	}
	lib := above("p1")
	lib = append(lib, exr("p1", "dressage", "beginner", "main", "kraft"))
	if got := main(beginner(lib)); got == "g1" {
		t.Errorf("prerequisite not mastered: %q", got)
	}
	lib = mutate(lib, "p1", func(e *training.Exercise) { e.Mastered = true })
	if got := main(beginner(lib)); got != "g1" {
		t.Errorf("prerequisite mastered: %q", got)
	}
	// Two levels above never.
	adv := mutate(above(), "g1", func(e *training.Exercise) { e.Level = "advanced" })
	if got := main(beginner(adv)); got != "" {
		t.Errorf("two levels above: %q", got)
	}
	// An unknown level counts as beginner; the advanced horse has nothing above.
	unknown := beginner(adv)
	unknown.Level = ""
	if got := main(unknown); got != "" {
		t.Errorf("unknown level, two levels above: %q", got)
	}
	advHorse := pickIn(adv)
	advHorse.Level = training.LevelAdvanced
	if got := main(advHorse); got != "g1" {
		t.Errorf("advanced horse, advanced exercise: %q", got)
	}
	// An easier exercise is always fine; prerequisites count for it too.
	easy := []training.Exercise{func() training.Exercise {
		e := exr("m1", "dressage", "beginner", "main", "losgelassenheit")
		e.Requires = []string{"m0"}
		return e
	}()}
	if got := main(pickIn(easy)); got != "" {
		t.Errorf("easy exercise with an open prerequisite: %q", got)
	}

	// Not in an Entlastungswoche, a return to work, for young horses or a horse that is not fit.
	blocked := map[string]func(*Input){
		"deload week":   func(in *Input) { in.Block = Block{Week: 4, Deload: true} },
		"building week": func(in *Input) { in.Block = Block{Week: 2} },
		"return to work": func(in *Input) {
			a := today
			in.RestartFrom = &a
		},
		"young horse": func(in *Input) { in.AgeYears = 4 },
		"reha":        func(in *Input) { in.Profile.Status = training.StatusReha },
		"pause":       func(in *Input) { in.Profile.Status = training.StatusPause },
	}
	for name, f := range blocked {
		in := beginner(above())
		f(&in)
		want := ""
		if name == "building week" {
			want = "g1"
		}
		if got := main(in); got != want {
			t.Errorf("%s: main = %q, want %q", name, got, want)
		}
	}
	// A return to work that is over is no restriction.
	over := today.AddDate(0, 0, -30)
	in := beginner(above())
	in.RestartFrom = &over
	if got := main(in); got != "g1" {
		t.Errorf("old return to work: %q", got)
	}

	// At most two units a week with a main exercise of the goal level (intermediate for a beginner).
	in = beginner(above())
	in.Recent = []training.Session{session(-2, "g1", 0), session(-1, "g1", 0)}
	if got := main(in); got != "" {
		t.Errorf("third goal unit this week: %q", got)
	}
	in.Recent = []training.Session{session(-4, "g1", 0), session(-1, "g1", 0)} // one of them last week (Sunday)
	if got := main(in); got != "g1" {
		t.Errorf("two goal units but one last week: %q", got)
	}
	// The cap is for the main part: a warm-up of the goal level is still fine.
	lib = append(above(), exr("w2", "dressage", "intermediate", "warmup", "losgelassenheit", "mobilitaet", "takt"))
	in = beginner(lib)
	in.Recent = []training.Session{session(-2, "g1", 0), session(-1, "g1", 0)}
	if got := partIDs(PlanUnit(in, training.ActivityHall, 45, Hints{})); got[0] != "w2" || got[1] != "" {
		t.Errorf("parts with the cap reached: %v", got)
	}
}

func TestPickCavalettiNotOnTwoDaysInARow(t *testing.T) {
	lib := append(pickLib(),
		exr("cv1", "cavaletti", "beginner", "warmup", "losgelassenheit", "mobilitaet", "takt"),
		exr("cv2", "cavaletti", "beginner", "main", "losgelassenheit", "biegung", "takt"))
	cv := func(extra func(*Input)) [3]string {
		in := pickIn(lib)
		if extra != nil {
			extra(&in)
		}
		return partIDs(PlanUnit(in, training.ActivityHall, 45, Hints{}))
	}
	// Cavaletti rows compete with the dressage ones (the same tie-break: the home library first).
	if got := cv(nil); got != [3]string{"cv1", "m1", "c1"} && got != [3]string{"w1", "m1", "c1"} {
		t.Errorf("parts = %v", got)
	}
	if got := cv(nil); got[0] != "cv1" {
		t.Errorf("the cavaletti warm-up scores higher: %v", got)
	}
	// The day after cavaletti (logged exercise or an exercise of the unit plan): none.
	for name, hist := range map[string][]training.Session{
		"logged":    {session(-1, "cv2", 3)},
		"unit plan": {{Day: day(-1), Activity: training.ActivityHall, Minutes: 40, Load: 32, Exercises: []string{"cv1"}}},
	} {
		got := cv(func(in *Input) { in.Recent = hist })
		if got[0] != "w1" || got[1] != "m1" {
			t.Errorf("%s: parts = %v, want no cavaletti the day after", name, got)
		}
	}
	// Two days after it is fine again.
	if got := cv(func(in *Input) { in.Recent = []training.Session{session(-2, "cv2", 3)} }); got[0] != "cv1" {
		t.Errorf("two days after: %v", got)
	}
	// Dressage the day before is no reason.
	if got := cv(func(in *Input) { in.Recent = []training.Session{session(-1, "m2", 3)} }); got[0] != "cv1" {
		t.Errorf("after dressage: %v", got)
	}
}

func TestPickLibrariesOfTheActivity(t *testing.T) {
	lib := []training.Exercise{
		exr("d1", "dressage", "beginner", "main", "losgelassenheit", "biegung"),
		exr("j1", "jumping", "beginner", "main", "losgelassenheit", "biegung"),
		exr("cv", "cavaletti", "beginner", "main", "losgelassenheit", "biegung"),
		exr("l1", "lunge", "beginner", "main", "losgelassenheit", "biegung"),
		exr("g1", "groundwork", "beginner", "main", "losgelassenheit", "biegung"),
		exr("h1", "hack", "beginner", "main", "losgelassenheit", "biegung"),
		exr("w1", "western", "beginner", "main", "losgelassenheit", "biegung"),
	}
	cases := []struct {
		name       string
		act        training.Activity
		discipline string
		want       string
	}{
		{"hall: dressage before cavaletti on a tie", training.ActivityHall, "dressage", "d1"},
		{"arena of a jumping horse: jumping first", training.ActivityArena, "jumping", "j1"},
		{"hall of a western horse: western first", training.ActivityHall, "western", "w1"},
		{"jumping day: jumping", training.ActivityJumping, "dressage", "j1"},
		{"lunge: lunge", training.ActivityLunge, "dressage", "l1"},
		{"groundwork", training.ActivityGroundwork, "dressage", "g1"},
		{"hack", training.ActivityHack, "dressage", "h1"},
	}
	for _, c := range cases {
		in := pickIn(lib)
		in.Profile.Discipline = c.discipline
		u := PlanUnit(in, c.act, 45, Hints{})
		if got := partIDs(u)[1]; got != c.want {
			t.Errorf("%s: main = %q, want %q", c.name, got, c.want)
		}
	}
	// Without the home library's exercises the others of the activity still count; a walker has none.
	only := []training.Exercise{lib[2]}
	if got := partIDs(PlanUnit(pickIn(only), training.ActivityHall, 45, Hints{}))[1]; got != "cv" {
		t.Errorf("only cavaletti: %q", got)
	}
	if u := PlanUnit(pickIn(lib), training.ActivityWalker, 30, Hints{}); u.Exercises() != nil {
		t.Errorf("walker: %v", u.Exercises())
	}
	// The cavaletti row of a lunge day needs the lunge's set-up: poles are not there by default.
	cvPoles := mutate(lib, "cv", func(e *training.Exercise) { e.Equipment = []string{"poles"} })
	in := pickIn(cvPoles)
	in.Profile.Discipline = "dressage"
	if got := partIDs(PlanUnit(in, training.ActivityLunge, 25, Hints{}))[1]; got != "l1" {
		t.Errorf("lunge without poles: %q", got)
	}
}

func TestPickInTheWeekBeforeAShow(t *testing.T) {
	lib := pickLib()
	lib = mutate(lib, "m1", func(e *training.Exercise) { e.Rated = false })
	lib = mutate(lib, "m2", func(e *training.Exercise) { e.Rated = true })
	lib = append(lib, func() training.Exercise {
		e := exr("t1", "dressage", "beginner", "main", "turnier", "kraft")
		return e
	}())
	main := func(showIn int) string {
		in := pickIn(lib)
		if showIn > 0 {
			in = withShow(in, showIn)
		}
		return partIDs(PlanUnit(in, training.ActivityHall, 30, Hints{}))[1]
	}
	// No new exercise in the 7 days before a show: only the ones the horse has rated, or rehearsals
	// (tag turnier); outside the week everything counts.
	if got := main(0); got != "m1" {
		t.Errorf("no show: %q", got)
	}
	if got := main(8); got != "m1" {
		t.Errorf("show in 8 days: %q", got)
	}
	if got := main(7); got != "m2" {
		t.Errorf("show in 7 days: %q, want the rated m2", got)
	}
	if got := main(2); got != "m2" {
		t.Errorf("show in 2 days: %q", got)
	}
	// The warm-up and the cool-down follow the same rule.
	in := withShow(pickIn(lib), 3)
	if got := partIDs(PlanUnit(in, training.ActivityHall, 45, Hints{})); got[0] != "" || got[2] != "" {
		t.Errorf("unknown warm-up and cool-down before a show: %v", got)
	}
	rated := mutate(mutate(lib, "w2", func(e *training.Exercise) { e.Rated = true }), "c2", func(e *training.Exercise) { e.Rated = true })
	in = withShow(pickIn(rated), 3)
	if got := partIDs(PlanUnit(in, training.ActivityHall, 45, Hints{})); got[0] != "w2" || got[2] != "c2" {
		t.Errorf("known warm-up and cool-down before a show: %v", got)
	}
	// A rehearsal is allowed before the show even when new, and fits the focus turnier.
	in = withShow(unitBase("turnier"), 2)
	in.Exercises = lib
	if got := partIDs(PlanUnit(in, training.ActivityHall, 45, Hints{}))[1]; got != "t1" {
		t.Errorf("rehearsal: %q", got)
	}
}

func TestPickEquipment(t *testing.T) {
	poles := []string{"poles"}
	lunge := []training.Exercise{
		func() training.Exercise {
			e := exr("t1", "cavaletti", "beginner", "main", "takt", "rhythmus")
			e.Title, e.Equipment = "Trabstangen", poles
			return e
		}(),
		func() training.Exercise {
			e := exr("t2", "lunge", "beginner", "main", "takt")
			e.Title, e.Equipment = "Tempiwechsel an der Longe", []string{"lunge"}
			return e
		}(),
	}
	run := func(lib []training.Exercise, act training.Activity, f func(*Input)) Unit {
		in := unitBase("takt")
		in.Exercises = lib
		if f != nil {
			f(&in)
		}
		return PlanUnit(in, act, 25, Hints{})
	}

	// No poles in the lunge hall: the alternative, and the unit says so in German.
	u := run(lunge, training.ActivityLunge, nil)
	if u.Main.Exercise == nil || u.Main.Exercise.ID != "t2" || u.Main.InsteadOf == nil || u.Main.InsteadOf.ID != "t1" {
		t.Fatalf("main = %+v", u.Main)
	}
	const want = "Ohne Stangen in der Longierhalle: Tempiwechsel an der Longe statt Trabstangen."
	if u.Main.Why != want || u.EquipmentNote != want {
		t.Errorf("why = %q, note = %q, want %q", u.Main.Why, u.EquipmentNote, want)
	}
	// With poles at the lunge the better exercise is planned and nothing is said.
	u = run(lunge, training.ActivityLunge, func(in *Input) {
		in.Equipment = training.StableEquipment{training.ActivityLunge: {Equipment: poles}}
	})
	if u.Main.Exercise.ID != "t1" || u.Main.InsteadOf != nil || u.Main.Why != "" || u.EquipmentNote != "" {
		t.Errorf("with poles: %+v / %q", u.Main, u.EquipmentNote)
	}
	// The set-up of the day: possible although the location is shared, not possible, shared.
	hall := []training.Exercise{lunge[0], lunge[1]}
	hall[0].Library, hall[1].Library = "dressage", "dressage"
	shared := training.StableEquipment{training.ActivityHall: {Equipment: poles, Shared: true}}
	u = run(hall, training.ActivityHall, func(in *Input) { in.Equipment = shared })
	if u.Main.Exercise.ID != "t2" || u.Main.Why != "Halle geteilt, ohne Aufbau: Tempiwechsel an der Longe statt Trabstangen." || u.EquipmentNote != "Halle geteilt: ohne Aufbau geplant." {
		t.Errorf("shared hall: %+v / %q", u.Main, u.EquipmentNote)
	}
	yes, no := true, false
	u = run(hall, training.ActivityHall, func(in *Input) { in.Equipment, in.EquipmentOK = shared, &yes })
	if u.Main.Exercise.ID != "t1" || u.Main.InsteadOf != nil || u.EquipmentNote != "" {
		t.Errorf("shared hall, set-up agreed: %+v / %q", u.Main, u.EquipmentNote)
	}
	u = run(hall, training.ActivityHall, func(in *Input) { in.EquipmentOK = &no })
	if u.Main.Exercise.ID != "t2" || u.Main.Why != "Heute ohne Aufbau: Tempiwechsel an der Longe statt Trabstangen." || u.EquipmentNote != "Heute ohne Aufbau geplant." {
		t.Errorf("no set-up today: %+v / %q", u.Main, u.EquipmentNote)
	}
	// Nothing without set-up fits: free work, and what was left out is named.
	u = run(lunge[:1], training.ActivityLunge, nil)
	if u.Main.Exercise != nil || u.Main.InsteadOf == nil || u.Main.Why != "Ohne Stangen in der Longierhalle: freie Arbeit statt Trabstangen." {
		t.Errorf("no alternative: %+v", u.Main)
	}
	// Several pieces of equipment are named together; personal equipment never matters.
	multi := mutate(hall, "t1", func(e *training.Exercise) { e.Equipment = []string{"poles", "cavaletti", "lunge"} })
	u = run(multi, training.ActivityHall, nil)
	if u.Main.Why != "Ohne Cavaletti in der Halle: Tempiwechsel an der Longe statt Trabstangen." {
		t.Errorf("one missing: %q", u.Main.Why)
	}
	u = run(multi, training.ActivityHall, func(in *Input) {
		in.Equipment = training.StableEquipment{training.ActivityHall: {Equipment: []string{"jumps"}}}
	})
	if u.Main.Why != "Ohne Stangen und Cavaletti in der Halle: Tempiwechsel an der Longe statt Trabstangen." {
		t.Errorf("two missing: %q", u.Main.Why)
	}
	free := []training.Exercise{func() training.Exercise {
		e := exr("f1", "dressage", "beginner", "main", "takt")
		e.Equipment = []string{"lunge", "tarp", "side_reins"}
		return e
	}()}
	if u := run(free, training.ActivityHall, nil); u.Main.Exercise == nil || u.Main.InsteadOf != nil {
		t.Errorf("personal equipment: %+v", u.Main)
	}
	// A hack has no location, so no facility equipment.
	obstacle := []training.Exercise{func() training.Exercise {
		e := exr("o1", "hack", "beginner", "main", "kondition")
		e.Equipment = []string{"obstacles"}
		return e
	}()}
	if u := run(obstacle, training.ActivityHack, func(in *Input) { in.Profile.Focuses = nil }); u.Main.Exercise != nil || u.Main.InsteadOf == nil {
		t.Errorf("hack with obstacles: %+v", u.Main)
	}
	// The warm-up and the cool-down are checked too.
	warm := []training.Exercise{func() training.Exercise {
		e := exr("w1", "dressage", "beginner", "warmup", "losgelassenheit")
		e.Equipment = poles
		return e
	}()}
	u = run(warm, training.ActivityHall, func(in *Input) { in.EquipmentOK = &no })
	if u.Warmup.Exercise != nil || u.Warmup.InsteadOf == nil || !strings.HasPrefix(u.Warmup.Why, "Heute ohne Aufbau: freie Arbeit statt ") {
		t.Errorf("warm-up: %+v", u.Warmup)
	}
}

func TestPickGoalLevel(t *testing.T) {
	lib := []training.Exercise{
		exr("b1", "dressage", "beginner", "main", "losgelassenheit", "biegung"),
		exr("g1", "dressage", "intermediate", "main", "losgelassenheit", "biegung"),
	}
	lib[1].Title = "Schulterherein"
	beginner := func(extra func(*Input)) Unit {
		in := pickIn(lib)
		in.Level = training.LevelBeginner
		if extra != nil {
			extra(&in)
		}
		return PlanUnit(in, training.ActivityHall, 45, Hints{})
	}
	// The goal (the class above) gets +1.5 and a reason; the other exercise would tie otherwise.
	u := beginner(nil)
	if u.Main.Exercise.ID != "g1" || u.Main.Why != "Richtung L: Schulterherein ist die nächste Lektion." {
		t.Fatalf("goal lesson: %+v", u.Main)
	}
	// Without the boost the order of the library decides (b1 first).
	for name, f := range map[string]func(*Input){
		"mastered": func(in *Input) { in.Exercises = mutate(lib, "g1", func(e *training.Exercise) { e.Mastered = true }) },
		"deload":   func(in *Input) { in.Block = Block{Week: 4, Deload: true} },
		"young":    func(in *Input) { in.AgeYears = 4 },
		"reha":     func(in *Input) { in.Profile.Status = training.StatusReha },
		"restart": func(in *Input) {
			a := today
			in.RestartFrom = &a
		},
		"cap": func(in *Input) {
			in.Recent = []training.Session{session(-2, "g1", 0), session(-1, "g1", 0)}
		},
	} {
		u := beginner(f)
		if u.Main.Exercise == nil || u.Main.Exercise.ID != "b1" || u.Main.Why != "" {
			t.Errorf("%s: main %+v", name, u.Main)
		}
	}
	// An explicit goal two classes up does not change what is planned today; the level does.
	u = beginner(func(in *Input) { in.Profile.GoalLevel = training.LevelAdvanced })
	if u.Main.Exercise.ID != "b1" || u.Main.Why != "" {
		t.Errorf("goal two classes up: %+v", u.Main)
	}
	// An intermediate horse works towards M.
	adv := []training.Exercise{lib[0], exr("a1", "dressage", "advanced", "main", "losgelassenheit", "biegung"), lib[1]}
	adv[1].Title = "Traversale"
	in := pickIn(adv)
	u = PlanUnit(in, training.ActivityHall, 45, Hints{})
	if u.Main.Exercise.ID != "a1" || u.Main.Why != "Richtung M: Traversale ist die nächste Lektion." {
		t.Errorf("towards M: %+v", u.Main)
	}
	// An advanced horse and a horse of unknown level have no goal.
	in = pickIn(lib)
	in.Level = training.LevelAdvanced
	if u := PlanUnit(in, training.ActivityHall, 45, Hints{}); u.Main.Why != "" {
		t.Errorf("advanced horse: %+v", u.Main)
	}
	in.Level = ""
	if u := PlanUnit(in, training.ActivityHall, 45, Hints{}); u.Main.Why != "" || u.Main.Exercise.ID != "b1" {
		t.Errorf("unknown level: %+v", u.Main)
	}
	// An explanation about set-up comes first; the goal lesson is not mentioned then.
	eq := mutate(lib, "g1", func(e *training.Exercise) { e.Equipment = []string{"jumps"} })
	in = pickIn(eq)
	in.Level = training.LevelBeginner
	u = PlanUnit(in, training.ActivityHall, 45, Hints{})
	if u.Main.Exercise.ID != "b1" || u.Main.InsteadOf == nil || !strings.HasPrefix(u.Main.Why, "Ohne Sprünge in der Halle: ") {
		t.Errorf("goal lesson without set-up: %+v", u.Main)
	}
}

func TestPlanUnitMinutes(t *testing.T) {
	in := pickIn(pickLib())
	cases := []struct {
		name string
		act  training.Activity
		min  int
		f    func(*Input)
		want [3]int
	}{
		{"hall 45", training.ActivityHall, 45, nil, [3]int{15, 22, 8}},
		{"hall 60", training.ActivityHall, 60, nil, [3]int{20, 30, 10}},
		{"a senior", training.ActivityHall, 45, func(in *Input) { in.AgeYears = 20 }, [3]int{20, 12, 13}},
		{"a cold night", training.ActivityHall, 45, func(in *Input) { in.Weather = &Weather{TempC: 4.9} }, [3]int{20, 17, 8}},
		{"5 degrees is not cold", training.ActivityHall, 45, func(in *Input) { in.Weather = &Weather{TempC: 5} }, [3]int{15, 22, 8}},
		{"lunge 25", training.ActivityLunge, 25, nil, [3]int{10, 10, 5}},
		{"lunge 15 has no main part", training.ActivityLunge, 15, nil, [3]int{10, 0, 5}},
		{"hack 60", training.ActivityHack, 60, nil, [3]int{10, 40, 10}},
		{"groundwork 30", training.ActivityGroundwork, 30, nil, [3]int{5, 20, 5}},
		{"jumping 40", training.ActivityJumping, 40, nil, [3]int{15, 18, 7}},
		{"no minutes: the default of the activity", training.ActivityHall, 0, nil, [3]int{15, 22, 8}},
	}
	for _, c := range cases {
		i := in
		if c.f != nil {
			c.f(&i)
		}
		u := PlanUnit(i, c.act, c.min, Hints{})
		if got := [3]int{u.Warmup.Minutes, u.Main.Minutes, u.Cooldown.Minutes}; got != c.want {
			t.Errorf("%s: minutes = %v, want %v", c.name, got, c.want)
		}
		if c.name == "lunge 15 has no main part" && u.Main.Exercise != nil {
			t.Errorf("an absent part has no exercise: %+v", u.Main)
		}
	}
	// A hack has no library exercises here: the parts are free work with their minutes.
	u := PlanUnit(in, training.ActivityHack, 60, Hints{})
	if u.Main.Exercise != nil || u.Warmup.Exercise != nil || u.Main.Minutes != 40 || u.Primary() != nil {
		t.Errorf("hack unit = %+v", u)
	}
}

func TestPlanUnitHints(t *testing.T) {
	ex := func(id string) *training.Exercise {
		for _, e := range pickLib() {
			if e.ID == id {
				return &e
			}
		}
		t.Fatalf("no exercise %s", id)
		return nil
	}
	plan := func(h Hints, f func(*Input)) Unit {
		in := pickIn(pickLib())
		if f != nil {
			f(&in)
		}
		return PlanUnit(in, training.ActivityHall, 45, h)
	}
	// What the model proposes is taken when it is allowed, even when the rules would pick another.
	u := plan(Hints{Warmup: ex("w2"), Main: ex("m2"), Cooldown: ex("c2")}, nil)
	if got := partIDs(u); got != [3]string{"w2", "m2", "c2"} || u.Replaced != "" {
		t.Fatalf("parts = %v, replaced %q", got, u.Replaced)
	}
	// Otherwise the rules pick, and Replaced says why.
	bad := func(label string, e *training.Exercise, f func(*Input), reason string) {
		t.Helper()
		var h Hints
		part := e.Phase
		switch part {
		case "warmup":
			h.Warmup = e
		case "cooldown":
			h.Cooldown = e
		default:
			h.Main = e
		}
		u := plan(h, f)
		want := "Übung „Titel " + e.ID + "“ ersetzt: " + reason + "."
		if u.Replaced != want {
			t.Errorf("%s: replaced = %q, want %q", label, u.Replaced, want)
		}
		got := partIDs(u)
		if got[0] != "w1" || got[1] != "m1" || got[2] != "c1" {
			t.Errorf("%s: the rules' parts = %v", label, got)
		}
	}
	other := exr("x1", "jumping", "beginner", "main", "losgelassenheit")
	bad("library", &other, nil, "gehört nicht zu dieser Aktivität")
	wrongPart := exr("x2", "dressage", "beginner", "cooldown", "losgelassenheit")
	h := Hints{Main: &wrongPart}
	if u := plan(h, nil); u.Replaced != "Übung „Titel x2“ ersetzt: gehört nicht in diesen Teil der Einheit." {
		t.Errorf("part: %q", u.Replaced)
	}
	noTag := exr("x3", "dressage", "beginner", "main", "kraft")
	bad("tags", &noTag, nil, "passt nicht zum Schwerpunkt Losgelassenheit")
	hard := exr("x4", "dressage", "advanced", "main", "losgelassenheit")
	bad("level, two above the horse's", &hard, func(in *Input) { in.Level = training.LevelBeginner }, "ist für das Pferd zu schwer")
	needs := exr("x5", "dressage", "beginner", "main", "losgelassenheit")
	needs.Requires = []string{"never"}
	bad("prerequisite", &needs, nil, "setzt Übungen voraus, die noch nicht sitzen")
	poles := exr("x6", "dressage", "beginner", "main", "losgelassenheit")
	poles.Equipment = []string{"cavaletti"}
	bad("equipment", &poles, nil, "braucht Cavaletti")
	// Twice the same exercise in a unit: the second is replaced.
	any := exr("x7", "dressage", "beginner", "any", "losgelassenheit", "biegung", "mobilitaet")
	u = PlanUnit(pickIn(append(pickLib(), any)), training.ActivityHall, 45, Hints{Main: &any, Warmup: &any})
	if u.Main.Exercise.ID != "x7" || u.Warmup.Exercise.ID != "w1" || u.Replaced != "Übung „Titel x7“ ersetzt: ist in dieser Einheit schon eingeplant." {
		t.Errorf("twice: %v, replaced %q", partIDs(u), u.Replaced)
	}
	// Several replaced exercises are listed one after the other.
	u = plan(Hints{Main: &noTag, Warmup: &wrongPart}, nil)
	if !strings.Contains(u.Replaced, "x3") || !strings.Contains(u.Replaced, "x2") || strings.Count(u.Replaced, "ersetzt") != 2 {
		t.Errorf("replaced = %q", u.Replaced)
	}
	// The model does not change the minutes.
	if u.Main.Minutes != 22 {
		t.Errorf("minutes = %d", u.Main.Minutes)
	}
}
