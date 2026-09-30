package recommend

import (
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// today (Wednesday) is weekday index 2; Monday is day(-2).

func TestUnitLevelsAndWindows(t *testing.T) {
	for load, want := range map[float64]string{0: LevelRecovery, 19.9: LevelRecovery, 20: LevelLight, 29.9: LevelLight, 30: LevelNormal, 44.9: LevelNormal, 45: LevelDemanding} {
		if got := UnitLevel(load); got != want {
			t.Errorf("UnitLevel(%v) = %s, want %s", load, got, want)
		}
	}
	// Hall has factor 0.8: light up to 37 minutes, normal 38-56, demanding from 57.
	if w := levelWindow(training.ActivityHall, LevelLight); w.hi != 37 {
		t.Errorf("hall light = %+v", w)
	}
	if w := levelWindow(training.ActivityHall, LevelNormal); w.lo != 38 || w.hi != 56 {
		t.Errorf("hall normal = %+v", w)
	}
	if w := levelWindow(training.ActivityHall, LevelDemanding); w.lo != 57 {
		t.Errorf("hall demanding = %+v", w)
	}
	if w := levelWindow(training.ActivityHall, LevelRecovery); !w.empty() {
		t.Errorf("hall is no recovery activity: %+v", w)
	}
	if w := levelWindow(training.ActivityHack, LevelRecovery); w.hi != 33 {
		t.Errorf("hack recovery = %+v", w)
	}
	if LevelLabel(LevelDemanding) != "fordernd" || LevelLabel(LevelRecovery) != "aktive Erholung" {
		t.Error("labels")
	}
}

func TestOwnerRestDay(t *testing.T) {
	in := base(allOn())
	in.Profile.Rhythm.Days[2] = training.DayRule{Kind: training.DayRest}
	r := Recommend(in).Recommendations
	if len(r) != 1 || r[0].Activity != training.ActivityRest {
		t.Fatalf("recommendations = %+v", r)
	}
	contains(t, r[0].Reason, "Mittwoch Ruhetag")
	if v := Check(in, training.ActivityHall, 45); v.OK || v.Recommendation.Activity != training.ActivityRest {
		t.Errorf("check = %+v", v)
	}
}

func TestOwnerActivityDay(t *testing.T) {
	in := base(allOn())
	in.Profile.Rhythm.Days[2] = training.DayRule{Kind: training.DayActivity, Activity: training.ActivityHack}
	if a := acts(Recommend(in)); len(a) != 1 || a[0] != training.ActivityHack {
		t.Fatalf("recommendations = %v", a)
	}
	v := Check(in, training.ActivityHall, 45)
	if v.OK || v.Recommendation.Activity != training.ActivityHack {
		t.Fatalf("check = %+v", v)
	}
	contains(t, v.Why, "Mittwoch für Ausritt vorgesehen")
	// A day activity the profile does not allow is ignored (safety first).
	in.Profile.Allowed = only(training.ActivityHall, training.ActivityLunge)
	if v := Check(in, training.ActivityHall, 45); !v.OK {
		t.Errorf("hack not allowed, hall must pass: %+v", v)
	}
}

func TestOwnerLevelDays(t *testing.T) {
	in := base(allOn())
	in.Profile.Rhythm.Days[2] = training.DayRule{Kind: training.DayDemanding}
	v := Check(in, training.ActivityHall, 45)
	if !v.OK || v.Recommendation.Minutes != 57 {
		t.Errorf("demanding hall = %+v", v)
	}
	if v := Check(in, training.ActivityLunge, 25); v.OK {
		t.Errorf("lunge cannot be demanding within 30 minutes: %+v", v)
	}
	for _, r := range Recommend(in).Recommendations {
		if UnitLevel(unitLoad(r.Activity, r.Minutes)) != LevelDemanding {
			t.Errorf("recommendation %+v is not demanding", r)
		}
	}

	in.Profile.Rhythm.Days[2] = training.DayRule{Kind: training.DayRecovery}
	if v := Check(in, training.ActivityHack, 60); !v.OK || v.Recommendation.Minutes != 33 {
		t.Errorf("recovery hack = %+v", v)
	}
	if v := Check(in, training.ActivityHall, 30); v.OK {
		t.Errorf("hall is no recovery: %+v", v)
	}
	in.Profile.Rhythm.Days[2] = training.DayRule{Kind: training.DayLight}
	if v := Check(in, training.ActivityHall, 45); !v.OK || v.Recommendation.Minutes != 37 {
		t.Errorf("light hall = %+v", v)
	}
}

func TestDefaultStructureRules(t *testing.T) {
	in := base(allOn())
	in.Recent = []training.Session{sess(-1, training.ActivityJumping, 50)} // demanding yesterday
	if v := Check(in, training.ActivityHall, 60); !v.OK || v.Recommendation.Minutes != 56 {
		t.Errorf("after a demanding day: %+v", v)
	}
	in.Recent = []training.Session{sess(-2, training.ActivityHall, 36), sess(-1, training.ActivityHall, 36)}
	if v := Check(in, training.ActivityHall, 45); !v.OK || v.Recommendation.Minutes != 37 {
		t.Errorf("after two working days: %+v", v)
	}
	// The owner's rule for the day wins over the default rules.
	in.Profile.Rhythm.Days[2] = training.DayRule{Kind: training.DayNormal}
	if v := Check(in, training.ActivityHall, 45); !v.OK || v.Recommendation.Minutes != 45 {
		t.Errorf("owner's normal day: %+v", v)
	}
}

func TestQuotaMaximums(t *testing.T) {
	in := base(allOn())
	in.Profile.Rhythm.Quotas = training.Quotas{Demanding: 1, Activities: map[training.Activity]int{training.ActivityHack: 1}}
	in.Recent = []training.Session{sess(-2, training.ActivityHack, 50)} // Monday: demanding hack
	if v := Check(in, training.ActivityHall, 60); !v.OK || v.Recommendation.Minutes != 56 {
		t.Errorf("demanding quota used: %+v", v)
	}
	v := Check(in, training.ActivityHack, 45)
	if v.OK {
		t.Fatalf("hack quota used: %+v", v)
	}
	contains(t, v.Why, "Ausritt war diese Woche schon 1×")
}

func TestLoadLimit(t *testing.T) {
	in := base(allOn())
	// Two weeks before: 4 sessions, 200 load, average 100, limit 120. This week so far: 100.
	in.Recent = []training.Session{
		sess(-9, training.ActivityHall, 50), sess(-10, training.ActivityHall, 50),
		sess(-12, training.ActivityHall, 50), sess(-15, training.ActivityHall, 50),
		sess(-2, training.ActivityHall, 50), sess(-1, training.ActivityLunge, 20) /* 70 */, sess(-1, training.ActivityWalker, 30),
	}
	v := Check(in, training.ActivityHall, 45)
	if v.OK {
		t.Fatalf("hall 30 minutes (24) exceeds the remaining 20: %+v", v)
	}
	contains(t, v.Why, "20 % über dem Schnitt")
	if v := Check(in, training.ActivityWalker, 60); !v.OK || v.Recommendation.Minutes > 66 {
		t.Errorf("walker fits: %+v", v)
	}
	// Without enough history there is no limit.
	in.Recent = in.Recent[3:]
	if v := Check(in, training.ActivityHall, 45); !v.OK {
		t.Errorf("no history, no limit: %+v", v)
	}
}

func TestWeekNeeds(t *testing.T) {
	in := base(allOn())
	in.Profile.Rhythm.RestDaysMin = 0
	in.Profile.Rhythm.Quotas = training.Quotas{Activities: map[training.Activity]int{training.ActivityHack: 1}}
	in.Recent = []training.Session{sess(-2, training.ActivityHall, 36), sess(-1, training.ActivityLunge, 17)}
	in.Ahead = &Ahead{OpenDays: 1}
	v := Check(in, training.ActivityHall, 45)
	if v.OK || v.Recommendation.Activity != training.ActivityHack {
		t.Fatalf("the last open day must be the hack: %+v", v)
	}
	contains(t, v.Why, "1× Ausritt")
	// With more open days left, today is free.
	in.Ahead.OpenDays = 3
	if v := Check(in, training.ActivityHall, 45); !v.OK {
		t.Errorf("three open days: %+v", v)
	}
	// Missing rest days on the last open day force a rest day.
	in.Profile.Rhythm.RestDaysMin = 1
	in.Profile.Rhythm.Quotas = training.Quotas{}
	in.Ahead.OpenDays = 1
	if v := Check(in, training.ActivityHall, 45); v.OK || v.Recommendation.Activity != training.ActivityRest {
		t.Errorf("rest day needed: %+v", v)
	}
	// Fixed units ahead count for the quota.
	in.Profile.Rhythm.RestDaysMin = 0
	in.Profile.Rhythm.Quotas = training.Quotas{Activities: map[training.Activity]int{training.ActivityHack: 1}}
	in.Ahead = &Ahead{OpenDays: 1, Units: []training.Session{sess(2, training.ActivityHack, 36)}}
	if v := Check(in, training.ActivityHall, 45); !v.OK {
		t.Errorf("hack already planned on Friday: %+v", v)
	}
}
