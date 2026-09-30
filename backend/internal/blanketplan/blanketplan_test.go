package blanketplan_test

import (
	"context"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/blanketplan"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func f(v float64) *float64 { return &v }
func b(v bool) *bool       { return &v }
func s(v string) *string   { return &v }

// Same shape as the seeded rules for Luna and Balu, built by hand.
var (
	luna = []blanketplan.Rule{
		{Position: 1, TempMin: f(12), Note: "none"},
		{Position: 2, TempMin: f(5), TempMax: f(12), Rain: b(true), BlanketID: s("rain")},
		{Position: 3, TempMin: f(5), TempMax: f(12), Note: "none"},
		{Position: 4, TempMin: f(0), TempMax: f(5), BlanketID: s("100"), Note: "rain blanket too when raining"},
		{Position: 5, TempMax: f(0), BlanketID: s("200")},
	}
	balu = []blanketplan.Rule{
		{Position: 1, TempMax: f(-5), BlanketID: s("150")},
		{Position: 2, Note: "none"},
	}
)

func id(r blanketplan.Rule) string {
	if r.BlanketID == nil {
		return ""
	}
	return *r.BlanketID
}

func TestRecommend(t *testing.T) {
	tests := []struct {
		name  string
		rules []blanketplan.Rule
		temp  float64
		rain  bool
		want  string // blanket id, "" = no blanket
	}{
		{"luna warm", luna, 20, false, ""},
		{"luna warm rain", luna, 20, true, ""},
		{"luna exactly 12", luna, 12, true, ""},
		{"luna 11.9 rain", luna, 11.9, true, "rain"},
		{"luna 8 dry", luna, 8, false, ""},
		{"luna exactly 5 dry", luna, 5, false, ""},
		{"luna exactly 5 rain", luna, 5, true, "rain"},
		{"luna 4.9 dry", luna, 4.9, false, "100"},
		{"luna 4.9 rain", luna, 4.9, true, "100"},
		{"luna exactly 0", luna, 0, false, "100"},
		{"luna -0.1", luna, -0.1, false, "200"},
		{"luna -10 rain", luna, -10, true, "200"},
		{"balu 10", balu, 10, true, ""},
		{"balu exactly -5", balu, -5, false, ""},
		{"balu -5.1", balu, -5.1, false, "150"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := blanketplan.Recommend(tt.rules, blanketplan.Forecast{NightMinC: tt.temp, WillRain: tt.rain})
			if !ok {
				t.Fatal("no rule matched")
			}
			if id(got) != tt.want {
				t.Errorf("blanket = %q, want %q (rule %+v)", id(got), tt.want, got)
			}
		})
	}
}

func TestRecommendNoMatchAndOrder(t *testing.T) {
	if _, ok := blanketplan.Recommend(nil, blanketplan.Forecast{}); ok {
		t.Error("empty rules matched")
	}
	gap := []blanketplan.Rule{{Position: 1, TempMax: f(0), BlanketID: s("x")}}
	if _, ok := blanketplan.Recommend(gap, blanketplan.Forecast{NightMinC: 3}); ok {
		t.Error("expected no match")
	}
	// Input order must not matter; lowest position wins.
	shuffled := []blanketplan.Rule{
		{Position: 2, BlanketID: s("second")},
		{Position: 1, BlanketID: s("first")},
	}
	got, _ := blanketplan.Recommend(shuffled, blanketplan.Forecast{})
	if id(got) != "first" {
		t.Errorf("got %q, want first", id(got))
	}
	if shuffled[0].Position != 2 {
		t.Error("input slice was modified")
	}
}

func TestChanged(t *testing.T) {
	a, c := blanketplan.Rule{BlanketID: s("a"), Note: "x"}, blanketplan.Rule{BlanketID: s("c")}
	none1, none2 := blanketplan.Rule{Position: 1}, blanketplan.Rule{Position: 2, Note: "other"}
	tests := []struct {
		name       string
		prev, next blanketplan.Rule
		want       bool
	}{
		{"same blanket", a, blanketplan.Rule{BlanketID: s("a"), Note: "y", Position: 9}, false},
		{"other blanket", a, c, true},
		{"blanket to none", a, none1, true},
		{"none to blanket", none1, a, true},
		{"none to none", none1, none2, false},
	}
	for _, tt := range tests {
		if got := blanketplan.Changed(tt.prev, tt.next); got != tt.want {
			t.Errorf("%s: Changed = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSeededRules(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()

	lunaRules, err := blanketplan.LoadRules(ctx, pool, seed.StableB, seed.HorseLuna)
	if err != nil {
		t.Fatal(err)
	}
	baluRules, err := blanketplan.LoadRules(ctx, pool, seed.StableB, seed.HorseBalu)
	if err != nil {
		t.Fatal(err)
	}
	if len(lunaRules) != 5 || len(baluRules) != 2 {
		t.Fatalf("loaded %d Luna and %d Balu rules, want 5 and 2", len(lunaRules), len(baluRules))
	}

	tests := []struct {
		name  string
		rules []blanketplan.Rule
		temp  float64
		rain  bool
		want  string
	}{
		{"luna 15", lunaRules, 15, true, ""},
		{"luna exactly 12", lunaRules, 12, false, ""},
		{"luna 8 dry", lunaRules, 8, false, ""},
		{"luna 8 rain", lunaRules, 8, true, seed.BlanketLunaRain},
		{"luna exactly 5 rain", lunaRules, 5, true, seed.BlanketLunaRain},
		{"luna 2 dry", lunaRules, 2, false, seed.BlanketLuna100},
		{"luna exactly 0", lunaRules, 0, false, seed.BlanketLuna100},
		{"luna -1", lunaRules, -1, false, seed.BlanketLuna200},
		{"balu 0", baluRules, 0, true, ""},
		{"balu exactly -5", baluRules, -5, false, ""},
		{"balu -6", baluRules, -6, false, seed.BlanketBalu150},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := blanketplan.Recommend(tt.rules, blanketplan.Forecast{NightMinC: tt.temp, WillRain: tt.rain})
			if !ok {
				t.Fatal("no rule matched")
			}
			if id(got) != tt.want {
				t.Errorf("blanket = %q, want %q", id(got), tt.want)
			}
		})
	}

	// The rain hint on the 0-5 rule is part of the rule's note.
	got, _ := blanketplan.Recommend(lunaRules, blanketplan.Forecast{NightMinC: 3, WillRain: true})
	if got.Note == "" {
		t.Error("0-5 rule lost its rain note")
	}

	// Another stable sees nothing.
	other, err := blanketplan.LoadRules(ctx, pool, "00000000-0000-4000-8000-0000000009ff", seed.HorseLuna)
	if err != nil || len(other) != 0 {
		t.Errorf("foreign stable: %d rules, err %v", len(other), err)
	}
}
