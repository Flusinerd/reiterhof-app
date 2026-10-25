package training

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

// specTable is the focus catalog table of docs/domains/training-planning.md: key, label, group,
// phase, tags matched, fits activities.
const specTable = `takt|Takt|scale|1|takt,rhythmus|hall,arena,lunge,hack
losgelassenheit|Losgelassenheit|scale|1|losgelassenheit,biegung|hall,arena,lunge,hack
anlehnung|Anlehnung|scale|1|anlehnung|hall,arena,lunge
schwung|Schwung|scale|2|schwung|hall,arena,hack
geraderichten|Geraderichtung|scale|2|geraderichten,biegung,seitengaenge|hall,arena
versammlung|Versammlung|scale|3|versammlung|hall,arena
durchlaessigkeit|Durchlässigkeit|scale|2|durchlaessigkeit|hall,arena,lunge
seitengaenge|Seitengänge|practice|2|seitengaenge|hall,arena,groundwork
gelassenheit|Gelassenheit|practice|0|gelassenheit,vertrauen|groundwork,hack,arena
kondition|Kondition|practice|0|kondition,ausdauer|hack,arena,hall
springen|Springen|practice|0|springgymnastik,rhythmus|jumping,hall,arena
gelaende|Gelände|practice|0|gelaende,gelassenheit,kondition|hack
vertrauen|Vertrauen|practice|0|vertrauen,aufmerksamkeit|groundwork,lunge
mobilitaet|Beweglichkeit|practice|0|mobilitaet,losgelassenheit|hall,arena,lunge,groundwork,hack
turnier|Turniervorbereitung|practice|0|turnier,durchlaessigkeit,aufmerksamkeit|hall,arena,jumping`

func TestFocusCatalogMatchesTheSpec(t *testing.T) {
	rows := strings.Split(specTable, "\n")
	if len(Focuses) != 15 || len(rows) != 15 {
		t.Fatalf("catalog has %d entries, the spec table %d, want 15", len(Focuses), len(rows))
	}
	for i, row := range rows {
		c := strings.Split(row, "|")
		f := Focuses[i]
		var fits []string
		for _, a := range f.Fits {
			fits = append(fits, string(a))
		}
		got := fmt.Sprintf("%s|%s|%s|%d|%s|%s", f.Key, f.Label, f.Group, f.Phase, strings.Join(f.Tags, ","), strings.Join(fits, ","))
		if got != row {
			t.Errorf("entry %d (%s):\n got  %s\n want %s", i, c[0], got, row)
		}
	}
}

func TestFocusCatalogTexts(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range Focuses {
		if seen[f.Key] {
			t.Errorf("key %q twice", f.Key)
		}
		seen[f.Key] = true
		for _, tag := range f.Tags {
			if !ValidExerciseTag(tag) {
				t.Errorf("%s: tag %q is not in ExerciseTags", f.Key, tag)
			}
		}
		for _, a := range f.Fits {
			if !a.Valid() {
				t.Errorf("%s: fits unknown activity %q", f.Key, a)
			}
		}
		for name, text := range map[string]string{"description": f.Description, "signs": f.Signs} {
			// one sentence: it ends with a full stop and has no other
			if !strings.HasSuffix(text, ".") || strings.Contains(text, ". ") || strings.Contains(text, "\n") {
				t.Errorf("%s: %s is not one sentence: %q", f.Key, name, text)
			}
			if strings.Contains(text, " du ") || strings.Contains(text, "Du ") || strings.Contains(text, "dein") {
				t.Errorf("%s: %s addresses the reader: %q", f.Key, name, text)
			}
		}
		// Only the focuses above the first phase carry a warning for horses that are not there yet.
		if (f.Phase >= 2) != (f.Needs != "") {
			t.Errorf("%s: phase %d, needs %q", f.Key, f.Phase, f.Needs)
		}
	}
	if f, _ := FocusByKey("versammlung"); f.Needs != "Versammlung setzt Schwung und Geraderichtung voraus." {
		t.Errorf("versammlung warning = %q", f.Needs)
	}
	if _, ok := FocusByKey("nope"); ok || ValidFocusKey("") || FocusLabel("nope") != "" {
		t.Error("unknown key accepted")
	}
	if FocusLabel("geraderichten") != "Geraderichtung" || !ValidFocusKey(FocusTurnier) {
		t.Error("FocusLabel or turnier key")
	}
	// Every activity with a unit plan has at least one focus that fits it (the rotation never runs dry).
	for _, a := range []Activity{ActivityHall, ActivityArena, ActivityLunge, ActivityGroundwork, ActivityHack, ActivityJumping} {
		n := 0
		for _, f := range Focuses {
			if f.FitsActivity(a) {
				n++
			}
		}
		if n == 0 {
			t.Errorf("no focus fits %s", a)
		}
	}
}

func TestPhase(t *testing.T) {
	cases := []struct {
		level  string
		age    int
		status Status
		want   HorsePhase
	}{
		{LevelBeginner, 10, StatusFit, PhaseGewoehnung},
		{"", 10, StatusFit, PhaseGewoehnung}, // unknown level
		{LevelIntermediate, 10, StatusFit, PhaseSchubkraft},
		{LevelAdvanced, 10, StatusFit, PhaseTragkraft},
		{LevelIntermediate, 0, StatusFit, PhaseSchubkraft}, // unknown age is not young
		{LevelIntermediate, 5, StatusFit, PhaseGewoehnung}, // under 6
		{LevelIntermediate, 6, StatusFit, PhaseSchubkraft},
		{LevelAdvanced, 3, StatusFit, PhaseGewoehnung},
		{LevelAdvanced, 12, StatusReha, PhaseGewoehnung},
		{LevelAdvanced, 12, StatusPause, PhaseGewoehnung},
		{LevelAdvanced, 20, StatusFit, PhaseTragkraft}, // seniors keep their phase
	}
	for _, c := range cases {
		if got := Phase(c.level, c.age, c.status); got != c.want {
			t.Errorf("Phase(%q, %d, %s) = %s, want %s", c.level, c.age, c.status, got, c.want)
		}
	}
	for p, want := range map[HorsePhase]int{PhaseGewoehnung: 1, PhaseSchubkraft: 2, PhaseTragkraft: 3, "": 1} {
		if p.Rank() != want {
			t.Errorf("%q.Rank() = %d", p, p.Rank())
		}
	}
	// Focuses allowed by phase: gewoehnung = phase 0 and 1, schubkraft adds 2, tragkraft adds 3.
	allowed := func(p HorsePhase) string {
		var keys []string
		for _, f := range Focuses {
			if f.AllowedIn(p) {
				keys = append(keys, f.Key)
			}
		}
		return strings.Join(keys, ",")
	}
	if got := allowed(PhaseGewoehnung); got != "takt,losgelassenheit,anlehnung,gelassenheit,kondition,springen,gelaende,vertrauen,mobilitaet,turnier" {
		t.Errorf("gewoehnung allows %s", got)
	}
	if got := allowed(PhaseSchubkraft); !strings.Contains(got, "schwung") || !strings.Contains(got, "seitengaenge") || strings.Contains(got, "versammlung") {
		t.Errorf("schubkraft allows %s", got)
	}
	if got := allowed(PhaseTragkraft); !strings.Contains(got, "versammlung") || len(strings.Split(got, ",")) != 15 {
		t.Errorf("tragkraft allows %s", got)
	}
}

func TestYoungAndSenior(t *testing.T) {
	for age, want := range map[int][2]bool{0: {false, false}, 3: {true, false}, 5: {true, false}, 6: {false, false}, 17: {false, false}, 18: {false, true}, 25: {false, true}} {
		if Young(age) != want[0] || Senior(age) != want[1] {
			t.Errorf("age %d: young %v senior %v", age, Young(age), Senior(age))
		}
	}
}

func TestHintKeys(t *testing.T) {
	cases := []struct {
		phase  HorsePhase
		chosen []string
		want   string
	}{
		{PhaseGewoehnung, nil, "takt,losgelassenheit,anlehnung"},
		{PhaseGewoehnung, []string{"losgelassenheit"}, "takt,anlehnung"},
		{PhaseGewoehnung, []string{"takt", "losgelassenheit", "anlehnung"}, ""},
		{PhaseSchubkraft, []string{"geraderichten", "turnier"}, "schwung,durchlaessigkeit"},
		{PhaseTragkraft, nil, "versammlung"},
	}
	for _, c := range cases {
		got := HintKeys(c.phase, c.chosen)
		if got == nil || strings.Join(got, ",") != c.want {
			t.Errorf("HintKeys(%s, %v) = %v, want %q", c.phase, c.chosen, got, c.want)
		}
	}
}

func TestFocusChoiceActive(t *testing.T) {
	day := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	until := func(offset int) *time.Time { d := day.AddDate(0, 0, offset); return &d }
	for _, c := range []struct {
		until *time.Time
		want  bool
	}{{nil, true}, {until(5), true}, {until(0), true}, {until(-1), false}} {
		if got := (FocusChoice{Key: "takt", Until: c.until}).Active(day); got != c.want {
			t.Errorf("Active with until %v = %v, want %v", c.until, got, c.want)
		}
	}
}

func TestExerciseTags(t *testing.T) {
	if len(ExerciseTags) != 21 {
		t.Errorf("%d tags, the spec lists 21", len(ExerciseTags))
	}
	for _, tag := range []string{"takt", "turnier", "mobilitaet", "ausdauer", "kraft", "gelaende", "balance"} {
		if !ValidExerciseTag(tag) {
			t.Errorf("tag %q missing", tag)
		}
	}
	if ValidExerciseTag("Takt") || ValidExerciseTag("") || ValidExerciseTag("speed") {
		t.Error("unknown tag accepted")
	}
	if slices.Contains(ExerciseTags, "") {
		t.Error("empty tag")
	}
}

func TestUnitSplit(t *testing.T) {
	cases := []struct {
		name         string
		a            Activity
		total        int
		senior, cold bool
		want         Split
	}{
		// hall, arena: Lösen clamp(round(T x 0.33), 10, 25), Entspannen clamp(round(T x 0.17), 5, 10)
		{"hall 45", ActivityHall, 45, false, false, Split{15, 22, 8}},
		{"hall 60", ActivityHall, 60, false, false, Split{20, 30, 10}},
		{"hall 30", ActivityHall, 30, false, false, Split{10, 15, 5}},
		{"hall 25 is the shortest with all minimums", ActivityHall, 25, false, false, Split{10, 10, 5}},
		{"hall 90 hits both caps", ActivityHall, 90, false, false, Split{25, 55, 10}},
		{"arena 50", ActivityArena, 50, false, false, Split{17, 24, 9}},
		{"hall 24 too short: proportions", ActivityHall, 24, false, false, Split{8, 12, 4}},
		// jumping: Lösen at least 15
		{"jumping 40", ActivityJumping, 40, false, false, Split{15, 18, 7}},
		{"jumping 30", ActivityJumping, 30, false, false, Split{15, 10, 5}},
		{"jumping 45", ActivityJumping, 45, false, false, Split{15, 22, 8}},
		{"jumping 60 beats the minimum", ActivityJumping, 60, false, false, Split{20, 30, 10}},
		// lunge, groundwork, hack: fixed parts, the rest is Arbeiten
		{"lunge 25", ActivityLunge, 25, false, false, Split{10, 10, 5}},
		{"lunge 30", ActivityLunge, 30, false, false, Split{10, 15, 5}},
		{"lunge 15 has no main part", ActivityLunge, 15, false, false, Split{10, 0, 5}},
		{"groundwork 30", ActivityGroundwork, 30, false, false, Split{5, 20, 5}},
		{"hack 60", ActivityHack, 60, false, false, Split{10, 40, 10}},
		{"hack 30", ActivityHack, 30, false, false, Split{10, 10, 10}},
		// walker and rest have no unit plan
		{"walker", ActivityWalker, 30, false, false, Split{}},
		{"rest", ActivityRest, 0, false, false, Split{}},
		{"unknown", "swimming", 30, false, false, Split{}},
		{"no minutes", ActivityHall, 0, false, false, Split{}},
		{"negative minutes", ActivityHall, -5, true, true, Split{}},
		// seniors and cold nights: +5 Lösen each (cap 30), seniors +5 Entspannen, from Arbeiten
		{"senior hall 45", ActivityHall, 45, true, false, Split{20, 12, 13}},
		{"cold hall 45", ActivityHall, 45, false, true, Split{20, 17, 8}},
		{"senior and cold hall 60", ActivityHall, 60, true, true, Split{30, 15, 15}},
		{"senior and cold hall 90 caps Lösen at 30", ActivityHall, 90, true, true, Split{30, 45, 15}},
		{"senior hall 30 keeps Arbeiten at 10", ActivityHall, 30, true, false, Split{15, 10, 5}},
		{"cold jumping 40", ActivityJumping, 40, false, true, Split{20, 13, 7}},
		{"senior lunge 25 keeps 5 for Arbeiten", ActivityLunge, 25, true, false, Split{15, 5, 5}},
		{"cold hack 60", ActivityHack, 60, false, true, Split{15, 35, 10}},
		{"senior and cold hack 60", ActivityHack, 60, true, true, Split{20, 25, 15}},
		{"cold groundwork 30", ActivityGroundwork, 30, false, true, Split{10, 15, 5}},
	}
	for _, c := range cases {
		if got := UnitSplit(c.a, c.total, c.senior, c.cold); got != c.want {
			t.Errorf("%s: UnitSplit(%s, %d, senior %v, cold %v) = %+v, want %+v", c.name, c.a, c.total, c.senior, c.cold, got, c.want)
		}
	}
}

// The parts always add up to the unit, never go negative and respect the minimums.
func TestUnitSplitInvariants(t *testing.T) {
	for _, a := range []Activity{ActivityHall, ActivityArena, ActivityJumping, ActivityLunge, ActivityGroundwork, ActivityHack} {
		for total := 1; total <= 240; total++ {
			for _, senior := range []bool{false, true} {
				for _, cold := range []bool{false, true} {
					s := UnitSplit(a, total, senior, cold)
					if s.Total() != total || s.Warmup < 0 || s.Main < 0 || s.Cooldown < 0 {
						t.Fatalf("UnitSplit(%s, %d, %v, %v) = %+v", a, total, senior, cold, s)
					}
					if s.Warmup > 30 && s.Warmup > UnitSplit(a, total, false, false).Warmup {
						t.Fatalf("%s %d: warm-up %d exceeds the cap", a, total, s.Warmup)
					}
					base := UnitSplit(a, total, false, false)
					if s.Warmup < base.Warmup || s.Cooldown < base.Cooldown || s.Main > base.Main {
						t.Fatalf("%s %d: extras shortened a part: %+v vs %+v", a, total, s, base)
					}
					enough := (a == ActivityHall || a == ActivityArena || a == ActivityJumping) && total >= 30
					if enough && s.Main < 10 {
						t.Fatalf("%s %d (senior %v, cold %v): Arbeiten %d is below 10", a, total, senior, cold, s.Main)
					}
				}
			}
		}
	}
	if !(Split{}).Empty() || (Split{1, 0, 0}).Empty() {
		t.Error("Split.Empty")
	}
}
