package recommend

import (
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// The structure groups 6 to 10 (docs/domains/training-planning.md, "Week structure additions"):
// today is a Wednesday, Monday is day(-2), Friday day(2).

// groupsIn is a horse with every activity allowed and no weekly maximum, so that a case shows one
// group at a time.
func groupsIn(f func(*Input)) Input {
	in := base(allOn())
	in.Profile.Rhythm.RestDaysMin = 0
	in.Profile.Rhythm.SessionsMax = 0
	if f != nil {
		f(&in)
	}
	return in
}

// checkCase is a proposal and what Check says about it.
type checkCase struct {
	name    string
	in      Input
	act     training.Activity
	minutes int
	ok      bool
	fitted  int    // minutes of an accepted proposal, 0 = not looked at
	why     string // part of the reason of a replaced proposal
}

func runChecks(t *testing.T, cases []checkCase) {
	t.Helper()
	for _, c := range cases {
		v := Check(c.in, c.act, c.minutes)
		if v.OK != c.ok {
			t.Errorf("%s: OK = %v, want %v (%+v)", c.name, v.OK, c.ok, v)
			continue
		}
		if c.ok && c.fitted != 0 && v.Recommendation.Minutes != c.fitted {
			t.Errorf("%s: minutes = %d, want %d", c.name, v.Recommendation.Minutes, c.fitted)
		}
		if !c.ok && c.why != "" {
			contains(t, v.Why, c.why)
		}
	}
}

func TestJumpingRules(t *testing.T) {
	jump, hall := training.ActivityJumping, training.ActivityHall
	two := func(in *Input) {
		in.Recent = []training.Session{sess(-2, jump, 36)}
		in.Ahead = &Ahead{OpenDays: 3, Units: []training.Session{sess(2, jump, 36)}}
	}
	runChecks(t, []checkCase{
		{"jumping the day before", groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-1, jump, 36)} }), jump, 30, false, 0, "Springen nicht an zwei Tagen hintereinander"},
		{"jumping the day after", groupsIn(func(in *Input) {
			in.Ahead = &Ahead{OpenDays: 3, Units: []training.Session{sess(1, jump, 36)}}
		}), jump, 30, false, 0, "Springen nicht an zwei Tagen hintereinander"},
		{"other activities are not affected", groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-1, jump, 36)} }), hall, 45, true, 45, ""},
		{"a gap day is fine", groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-2, jump, 36)} }), jump, 30, true, 30, ""},
		{"two units a week are the most", groupsIn(two), jump, 30, false, 0, "Springen höchstens 2× pro Woche"},
		{"a quota above two is capped at two", groupsIn(func(in *Input) {
			two(in)
			in.Profile.Rhythm.Quotas = training.Quotas{Activities: map[training.Activity]int{jump: 3}}
		}), jump, 30, false, 0, "Springen höchstens 2× pro Woche"},
		{"the owner's quota of one is stricter", groupsIn(func(in *Input) {
			in.Recent = []training.Session{sess(-2, jump, 36)}
			in.Profile.Rhythm.Quotas = training.Quotas{Activities: map[training.Activity]int{jump: 1}}
		}), jump, 30, false, 0, "Springen war diese Woche schon 1×"},
		{"one unit so far is fine", groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-2, jump, 36)} }), jump, 30, true, 30, ""},
	})
	// When nothing else is allowed the day becomes a rest day, hard like the safety rules.
	in := base(only(jump))
	in.Recent = []training.Session{sess(-1, jump, 36)}
	r := Recommend(in).Recommendations
	if len(r) != 1 || r[0].Activity != training.ActivityRest {
		t.Fatalf("recommendations = %+v", r)
	}
	contains(t, r[0].Reason, "Springen nicht an zwei Tagen hintereinander – heute Ruhetag.")
	if v := Check(in, jump, 30); v.OK || v.Recommendation.Activity != training.ActivityRest {
		t.Errorf("check = %+v", v)
	}
	in.Recent = []training.Session{sess(-2, jump, 36), sess(-3, jump, 36)}
	in.Ahead = nil
	// Monday and the Sunday before are in different weeks: only Monday counts.
	if v := Check(in, jump, 30); !v.OK {
		t.Errorf("one unit this week: %+v", v)
	}
}

func TestVarietyRules(t *testing.T) {
	hall, arena, lunge, jump := training.ActivityHall, training.ActivityArena, training.ActivityLunge, training.ActivityJumping
	yesterday := func(a training.Activity, allowed ...training.Activity) Input {
		in := base(only(allowed...))
		in.Profile.Rhythm.RestDaysMin = 0
		in.Recent = []training.Session{sess(-1, a, 36)}
		return in
	}
	runChecks(t, []checkCase{
		{"the activity of yesterday is out when two others remain", yesterday(hall, hall, arena, lunge), hall, 45, false, 0, "Halle war gestern schon dran – heute etwas anderes."},
		{"the others are fine", yesterday(hall, hall, arena, lunge), arena, 45, true, 45, ""},
		{"with one other left a repeat is allowed", yesterday(hall, hall, arena), hall, 45, true, 45, ""},
		{"a unit fixed for tomorrow counts as well", groupsIn(func(in *Input) {
			in.Ahead = &Ahead{OpenDays: 3, Units: []training.Session{sess(1, arena, 36), sess(3, lunge, 20)}}
		}), arena, 45, false, 0, "Platz ist morgen schon eingetragen – heute etwas anderes."},
		{"a unit fixed in two days does not", groupsIn(func(in *Input) {
			in.Ahead = &Ahead{OpenDays: 3, Units: []training.Session{sess(2, arena, 36), sess(3, lunge, 20)}}
		}), arena, 45, true, 45, ""},
		{"two days ago is no repeat", groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-2, hall, 36)} }), hall, 45, true, 45, ""},
		{"jumping the day before is the jumping rule, not variety", yesterday(jump, jump, hall, arena), jump, 30, false, 0, "Springen nicht an zwei Tagen hintereinander"},
	})

	// With three or more activities allowed the week needs three different ones: on the last
	// open day the activities already done are out.
	need := func(openDays int, allowed ...training.Activity) Input {
		in := base(only(allowed...))
		in.Profile.Rhythm.RestDaysMin = 0
		in.Recent = []training.Session{sess(-2, hall, 36)}
		in.Ahead = &Ahead{OpenDays: openDays}
		return in
	}
	runChecks(t, []checkCase{
		{"two more activities are needed in two days", need(2, hall, arena, lunge), hall, 45, false, 0, "Diese Woche fehlen noch: 2 weitere Aktivitäten."},
		{"a new activity satisfies the need", need(2, hall, arena, lunge), arena, 45, true, 45, ""},
		{"there is time left: no need yet", need(4, hall, arena, lunge), hall, 45, true, 45, ""},
		{"with two activities allowed there is no such need", need(2, hall, arena), hall, 45, true, 45, ""},
	})
	in := need(1, hall, arena, lunge)
	in.Recent = append(in.Recent, sess(-1, arena, 36))
	v := Check(in, hall, 45)
	if v.OK || v.Recommendation.Activity != lunge {
		t.Fatalf("the last open day needs the third activity: %+v", v)
	}
	contains(t, v.Why, "1 weitere Aktivität")
	if v := Check(in, lunge, 25); !v.OK {
		t.Errorf("the third activity: %+v", v)
	}

	// A quota counts as a different activity the week is going to have.
	in = need(2, hall, arena, lunge)
	in.Profile.Rhythm.Quotas = training.Quotas{Activities: map[training.Activity]int{arena: 1}}
	if v := Check(in, hall, 45); v.OK || v.Recommendation.Activity != arena {
		t.Errorf("the arena quota is open, and it is one of the three: %+v", v)
	}

	// Recommend: a repeat is not forbidden when only two activities fit, but it is penalised.
	two := base(only(hall, arena))
	two.Recent = []training.Session{sess(-1, hall, 20)}
	if got := acts(Recommend(two)); len(got) != 2 || got[0] != arena {
		t.Errorf("repeat penalty: %v", got)
	}
}

func TestDemandingMaximum(t *testing.T) {
	hall := training.ActivityHall
	// Hall: 56 minutes are normal (load 44.8), 57 to 60 demanding. A demanding unit on Monday and
	// one fixed for Friday.
	units := func(f func(*Input)) Input {
		return groupsIn(func(in *Input) {
			in.Recent = []training.Session{sess(-2, hall, 50)}
			in.Ahead = &Ahead{OpenDays: 3, Units: []training.Session{sess(2, hall, 50)}}
			if f != nil {
				f(in)
			}
		})
	}
	one := groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-2, hall, 50)} })
	runChecks(t, []checkCase{
		{"two demanding units are the default maximum", units(nil), hall, 60, true, 56, ""},
		{"one is fine", one, hall, 60, true, 60, ""},
		{"a quota of three is capped at two", units(func(in *Input) { in.Profile.Rhythm.Quotas = training.Quotas{Demanding: 3} }), hall, 60, true, 56, ""},
		{"a quota of one is stricter", groupsIn(func(in *Input) {
			in.Recent = []training.Session{sess(-2, hall, 50)}
			in.Profile.Rhythm.Quotas = training.Quotas{Demanding: 1}
		}), hall, 60, true, 56, ""},
		{"a senior has one", groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-2, hall, 50)}; in.AgeYears = 20 }), hall, 60, true, 56, ""},
		{"a senior without a demanding unit may have one", groupsIn(func(in *Input) { in.AgeYears = 20 }), hall, 60, true, 60, ""},
		{"17 is no senior", groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-2, hall, 50)}; in.AgeYears = 17 }), hall, 60, true, 60, ""},
		{"18 is", groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-2, hall, 50)}; in.AgeYears = 18 }), hall, 60, true, 56, ""},
		{"an unknown age counts as an adult", groupsIn(func(in *Input) { in.Recent = []training.Session{sess(-2, hall, 50)}; in.AgeYears = 0 }), hall, 60, true, 60, ""},
	})
	if demandingMax(training.Quotas{}, 10) != 2 || demandingMax(training.Quotas{Demanding: 1}, 10) != 1 ||
		demandingMax(training.Quotas{Demanding: 5}, 10) != 2 || demandingMax(training.Quotas{}, 18) != 1 || demandingMax(training.Quotas{Demanding: 2}, 30) != 1 {
		t.Error("demandingMax")
	}
}

func TestDeloadWeek(t *testing.T) {
	hall, walker := training.ActivityHall, training.ActivityWalker
	week := func(f func(*Input)) Input {
		return groupsIn(func(in *Input) {
			in.Block = Block{Week: 4, Deload: true}
			if f != nil {
				f(in)
			}
		})
	}
	// Three weeks before with 50 load each: the cap is 0.8 × 50 = 40.
	history := []training.Session{sess(-9, hall, 50), sess(-16, hall, 50), sess(-23, hall, 50)}
	runChecks(t, []checkCase{
		{"a build week has the demanding unit", groupsIn(func(in *Input) { in.Block = Block{Week: 3} }), hall, 60, true, 60, ""},
		{"no demanding unit in the Entlastungswoche", week(nil), hall, 60, true, 56, ""},
		{"the load is capped at 0.8 of the three weeks before: 40 is 50 minutes", week(func(in *Input) { in.Recent = history }), hall, 56, true, 50, ""},
		{"a unit under the cap is untouched", week(func(in *Input) { in.Recent = history }), hall, 45, true, 45, ""},
		{"nothing is left of the cap: the hall is replaced",
			week(func(in *Input) { in.Recent = append([]training.Session{sess(-2, hall, 30)}, history...) }), hall, 45, false, 0,
			"Entlastungswoche: die Wochenlast bleibt unter 80 % des Schnitts der drei Vorwochen."},
		{"a short light activity still fits the 10 left",
			week(func(in *Input) { in.Recent = append([]training.Session{sess(-2, hall, 30)}, history...) }), walker, 60, true, 33, ""},
		{"without enough history there is no cap", week(func(in *Input) { in.Recent = history[:2] }), hall, 56, true, 56, ""},
	})
	// A day without anything under the cap is a rest day.
	in := week(func(in *Input) { in.Recent = append([]training.Session{sess(-2, hall, 40)}, history...) })
	in.Profile.Allowed = only(hall)
	r := Recommend(in).Recommendations
	if len(r) != 1 || r[0].Activity != training.ActivityRest {
		t.Fatalf("recommendations = %+v", r)
	}
	contains(t, r[0].Reason, "Entlastungswoche")
	// A demanding day of the owner wins over the week: the group is skipped when nothing is left.
	in = week(func(in *Input) { in.Profile.Rhythm.Days[2] = training.DayRule{Kind: training.DayDemanding} })
	if v := Check(in, hall, 60); !v.OK || v.Recommendation.Minutes != 60 {
		t.Errorf("the owner's demanding day: %+v", v)
	}
}

func TestReturnToWork(t *testing.T) {
	hall, jump, hack := training.ActivityHall, training.ActivityJumping, training.ActivityHack
	from := func(offset int) func(*Input) {
		return func(in *Input) {
			d := day(offset)
			in.RestartFrom = &d
		}
	}
	runChecks(t, []checkCase{
		{"week 1: light only, the hall is cut to 37 minutes", groupsIn(from(0)), hall, 45, true, 37, ""},
		{"week 1: jumping has no light unit", groupsIn(from(0)), jump, 30, false, 0, "Wiederantrainieren, Woche 1: nur leichte Einheiten."},
		{"week 1: the hack is light up to 49 minutes", groupsIn(from(0)), hack, 90, true, 49, ""},
		{"week 1 lasts seven days", groupsIn(from(-6)), hall, 45, true, 37, ""},
		{"week 2: nothing demanding", groupsIn(from(-7)), hall, 60, true, 56, ""},
		{"week 2: a normal unit is fine", groupsIn(from(-7)), hall, 45, true, 45, ""},
		{"week 3: nothing demanding", groupsIn(from(-14)), hall, 60, true, 56, ""},
		{"week 3: jumping at 30 minutes is normal", groupsIn(from(-14)), jump, 30, true, 30, ""},
		{"week 4: back to normal", groupsIn(from(-21)), hall, 60, true, 60, ""},
		{"not returning", groupsIn(nil), hall, 60, true, 60, ""},
		{"an anchor in the future is not returning", groupsIn(from(1)), hall, 60, true, 60, ""},
	})
	in := base(only(jump))
	from(0)(&in)
	r := Recommend(in).Recommendations
	if len(r) != 1 || r[0].Activity != training.ActivityRest {
		t.Fatalf("recommendations = %+v", r)
	}
	contains(t, r[0].Reason, "Wiederantrainieren, Woche 1: nur leichte Einheiten")
	// The recommender offers light units only.
	in = groupsIn(from(0))
	for _, rec := range Recommend(in).Recommendations {
		if UnitLevel(unitLoad(rec.Activity, rec.Minutes)) == LevelDemanding || unitLoad(rec.Activity, rec.Minutes) >= 30 {
			t.Errorf("recommendation %+v is not light", rec)
		}
	}
}

func TestYoungHorses(t *testing.T) {
	hall, arena, lunge, ground, jump := training.ActivityHall, training.ActivityArena, training.ActivityLunge,
		training.ActivityGroundwork, training.ActivityJumping
	young := func(f func(*Input)) Input {
		return groupsIn(func(in *Input) {
			in.AgeYears = 4
			if f != nil {
				f(in)
			}
		})
	}
	fourUnits := func(in *Input) {
		in.Recent = []training.Session{sess(-2, hall, 20), sess(-1, lunge, 20)}
		in.Ahead = &Ahead{OpenDays: 3, Units: []training.Session{sess(2, arena, 20), sess(3, ground, 20)}}
	}
	runChecks(t, []checkCase{
		{"units are at most 30 minutes", young(nil), hall, 60, true, 30, ""},
		{"a short unit is untouched", young(nil), lunge, 20, true, 20, ""},
		{"jumping has no unit of 30 minutes below its minimum", young(nil), jump, 40, true, 30, ""},
		{"six is the first age of an adult horse", groupsIn(func(in *Input) { in.AgeYears = 6 }), hall, 60, true, 60, ""},
		{"five is young", groupsIn(func(in *Input) { in.AgeYears = 5 }), hall, 60, true, 30, ""},
		{"three units so far are fine", young(func(in *Input) {
			in.Recent = []training.Session{sess(-2, hall, 20), sess(-1, lunge, 20)}
			in.Ahead = &Ahead{OpenDays: 3, Units: []training.Session{sess(2, arena, 20)}}
		}), ground, 20, true, 20, ""},
		{"four units are the most", young(fourUnits), ground, 20, false, 0, "Junge Pferde bekommen höchstens 4 Einheiten pro Woche – heute Ruhetag."},
		{"the cap wins over a demanding day of the owner", young(func(in *Input) {
			in.Profile.Rhythm.Days[2] = training.DayRule{Kind: training.DayDemanding}
		}), hall, 60, true, 30, ""},
		{"an unknown age is no young horse", groupsIn(func(in *Input) { in.AgeYears = 0 }), hall, 60, true, 60, ""},
	})
	in := young(fourUnits)
	r := Recommend(in).Recommendations
	if len(r) != 1 || r[0].Activity != training.ActivityRest {
		t.Fatalf("recommendations = %+v", r)
	}
	contains(t, r[0].Reason, "höchstens 4 Einheiten pro Woche")
	for _, rec := range Recommend(young(nil)).Recommendations {
		if rec.Minutes > YoungMaxMinutes {
			t.Errorf("recommendation %+v is longer than %d minutes", rec, YoungMaxMinutes)
		}
	}
}

func TestSeniorsKeepOneDemandingUnit(t *testing.T) {
	hall := training.ActivityHall
	// A demanding day of the owner for a senior who had a demanding unit this week: the owner's
	// rule wins, like it does for the other quotas.
	in := groupsIn(func(in *Input) {
		in.AgeYears = 20
		in.Recent = []training.Session{sess(-2, hall, 50)}
		in.Profile.Rhythm.Days[2] = training.DayRule{Kind: training.DayDemanding}
	})
	if v := Check(in, hall, 60); !v.OK || v.Recommendation.Minutes != 60 {
		t.Errorf("owner's demanding day: %+v", v)
	}
	// Without the rule the second demanding unit is cut to a normal one.
	in.Profile.Rhythm.Days[2] = training.DayRule{}
	if v := Check(in, hall, 60); !v.OK || v.Recommendation.Minutes != 56 {
		t.Errorf("senior: %+v", v)
	}
}
