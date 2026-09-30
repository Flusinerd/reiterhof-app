package trainingapi

import (
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
)

func day(s string) time.Time {
	d, err := time.Parse(dateLayout, s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestContextLine(t *testing.T) {
	today := day("2026-03-25")
	shows := []training.Show{{Date: day("2026-03-28"), Name: "Turnier"}, {Date: day("2026-05-01"), Name: "Später"}}
	cases := []struct {
		name string
		in   contextInput
		want string
	}{
		{"rain and show", contextInput{Weather: &recommend.Weather{Rain: true, TempC: 5.6}, Today: today, Shows: shows}, "Regen, 6 °C · Turnier in 3 Tagen"},
		{"dry, negative", contextInput{Weather: &recommend.Weather{TempC: -2.4}, Today: today}, "Trocken, -2 °C"},
		{"ground", contextInput{Ground: "frozen", Today: today}, "Boden gefroren"},
		{"dry ground is silent", contextInput{Ground: "dry", Today: today}, ""},
		{"show tomorrow", contextInput{Today: today, Shows: []training.Show{{Date: day("2026-03-26"), Name: "Turnier"}}}, "Turnier morgen"},
		{"show today", contextInput{Today: today, Shows: []training.Show{{Date: today, Name: "Turnier"}}}, "Turnier heute"},
		{"past show", contextInput{Today: today, Shows: []training.Show{{Date: day("2026-03-20"), Name: "Turnier"}}}, ""},
		{"reha", contextInput{Today: today, Reha: &rehaOut{Phase: "Phase 2"}}, "Reha: Phase 2"},
	}
	for _, c := range cases {
		if got := contextLine(c.in); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestRiderRulesMapping(t *testing.T) {
	p := emptyProfile()
	if p.riderRules("u1", []string{"ride"}) != nil {
		t.Fatal("rider without rules entry must map to nil (fail closed)")
	}
	p.rbRules["u1"] = rbRule{AllowedActivities: []training.Activity{"hall", "rest", "hack"}, MaxIntensity: "medium"}
	r := p.riderRules("u1", nil)
	if len(r.AllowedActivities) != 2 || r.MaxIntensity != training.IntensityMedium || r.MayHackAlone || r.MayRideShows {
		t.Fatalf("rules = %+v", r)
	}
	r = p.riderRules("u1", []string{RuleHackAlone, RuleShows})
	if !r.MayHackAlone || !r.MayRideShows {
		t.Fatalf("keys must grant hack_alone and shows: %+v", r)
	}
}

func TestDefaultRhythmKeepsRestAfterShow(t *testing.T) {
	// A stored rhythm without rest_after_show must not switch the rule off.
	p := emptyProfile()
	if !p.rhythm.RestAfterShow {
		t.Fatal("empty profile must start from DefaultRhythm")
	}
	v, err := validateProfile(profileIn{Discipline: "dressage", Rhythm: &rhythmIn{}}, nil)
	if err != nil || !v.rhythm.RestAfterShow || v.rhythm.SessionsMax != 5 {
		t.Fatalf("rhythm = %+v, err %v", v.rhythm, err)
	}
}
