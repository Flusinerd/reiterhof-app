package dressage

import (
	"encoding/json"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func near(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %.3f, want %.3f ± %.3f", what, got, want, tol)
	}
}

func TestArenaLetters(t *testing.T) {
	a, ok := NewArena(Size20x40)
	if !ok {
		t.Fatal("no arena")
	}
	for name, want := range map[string]Point{
		"A": {10, 0}, "C": {10, 40}, "K": {0, 6}, "E": {0, 20}, "H": {0, 34},
		"F": {20, 6}, "B": {20, 20}, "M": {20, 34}, "D": {10, 6}, "X": {10, 20}, "G": {10, 34},
	} {
		if got := a.MustPos(name); got != want {
			t.Errorf("20x40 %s = %v, want %v", name, got, want)
		}
	}
	b, _ := NewArena(Size20x60)
	for name, want := range map[string]Point{
		"K": {0, 6}, "V": {0, 18}, "E": {0, 30}, "S": {0, 42}, "H": {0, 54},
		"F": {20, 6}, "P": {20, 18}, "B": {20, 30}, "R": {20, 42}, "M": {20, 54},
		"D": {10, 6}, "L": {10, 18}, "X": {10, 30}, "I": {10, 42}, "G": {10, 54}, "C": {10, 60},
	} {
		if got := b.MustPos(name); got != want {
			t.Errorf("20x60 %s = %v, want %v", name, got, want)
		}
	}
	if _, ok := NewArena("20x80"); ok {
		t.Error("unknown size accepted")
	}
	// Clockwise order around the arena.
	if got := strings.Join(a.TrackLetters(), ""); got != "AKEHCMBF" {
		t.Errorf("ring 20x40 = %s", got)
	}
	if got := strings.Join(b.TrackLetters(), ""); got != "AKVESHCMRBPF" {
		t.Errorf("ring 20x60 = %s", got)
	}
}

func TestArenaDistances(t *testing.T) {
	a, _ := NewArena(Size20x40)
	// Perimeter of the rounded rectangle: 2*(20+40) - 8r + 2πr.
	near(t, "perimeter", a.Perimeter(), 2*(20+40)-8*CornerRadius+2*math.Pi*CornerRadius, 1e-6)
	// Along a straight side between letters.
	near(t, "K–E", a.TrackDistance("K", "E", DirR), 14, 1e-6)
	near(t, "E–K left rein", a.TrackDistance("E", "K", DirL), 14, 1e-6)
	near(t, "K–H", a.TrackDistance("K", "H", DirR), 28, 1e-6)
	// Around a corner: 3 m straight, a quarter circle of radius 3, 3 m straight.
	near(t, "H–C", a.TrackDistance("H", "C", DirR), 3+3*math.Pi/2+7, 1e-6)
	near(t, "A–K", a.TrackDistance("A", "K", DirR), 7+3*math.Pi/2+3, 1e-6)
	near(t, "full lap", a.TrackDistance("A", "A", DirR), a.Perimeter(), 1e-6)
	// The path follows the distance.
	near(t, "path A–K", PathLength(a.TrackPath("A", "K", DirR)), a.TrackDistance("A", "K", DirR), 0.1)
	if got := a.LettersBetween("A", "E", DirR); !reflect.DeepEqual(got, []string{"A", "K", "E"}) {
		t.Errorf("letters A..E = %v", got)
	}
	if got := a.LettersBetween("A", "F", DirL); !reflect.DeepEqual(got, []string{"A", "F"}) {
		t.Errorf("letters A..F left = %v", got)
	}
	b, _ := NewArena(Size20x60)
	near(t, "20x60 V–E", b.TrackDistance("V", "E", DirR), 12, 1e-6)
	near(t, "20x60 perimeter", b.Perimeter(), 2*(20+60)-8*CornerRadius+2*math.Pi*CornerRadius, 1e-6)
}

func lib(t *testing.T, size string) *Library {
	t.Helper()
	l, ok := LibraryFor(size)
	if !ok {
		t.Fatal("no library")
	}
	return l
}

func optionOf(t *testing.T, l *Library, kind, letters string, start State) *Option {
	t.Helper()
	for i := range l.All {
		o := &l.All[i]
		if o.Kind == kind && o.Letters == letters && o.Start == start {
			return o
		}
	}
	t.Fatalf("no option %s %s from %v", kind, letters, start)
	return nil
}

func TestFigureLengths(t *testing.T) {
	l := lib(t, Size20x40)
	near(t, "zirkel", optionOf(t, l, "zirkel", "A", State{"A", DirR}).Length, 2*math.Pi*10, 0.2)
	near(t, "volte 10", optionOf(t, l, "volte10", "K", State{"K", DirR}).Length, 2*math.Pi*5, 0.1)
	near(t, "volte 8", optionOf(t, l, "volte8", "E", State{"E", DirL}).Length, 2*math.Pi*4, 0.1)
	near(t, "diagonal K–M", optionOf(t, l, "diagonal", "K–X–M", State{"K", DirR}).Length, math.Hypot(20, 28), 0.01)
	near(t, "halbe Bahn E–B", optionOf(t, l, "halbe_bahn", "E–X–B", State{"E", DirR}).Length, 20, 0.01)
	near(t, "Mittellinie", optionOf(t, l, "mittellinie", "A–X–C", State{"A", DirR}).Length, 40, 0.01)
	near(t, "track K–E", optionOf(t, l, "track", "K–E", State{"K", DirR}).Length, 14, 0.01)
	// Through the circle: two half circles of 20 m.
	near(t, "durch den Zirkel", optionOf(t, l, "handwechsel_zirkel", "A–X–C", State{"A", DirR}).Length, 2*math.Pi*10, 0.3)
	// Shoulder-in lines are at least 12 m.
	for _, o := range l.All {
		if o.Kind == "schulterherein" && o.Length < 12 {
			t.Errorf("%s shorter than 12 m: %.1f", o.ID, o.Length)
		}
	}
	// The half pass moves at least 5 m sideways.
	for _, o := range l.All {
		if o.Kind == "traversale" {
			end := o.Path[len(o.Path)-1]
			if math.Abs(end[0]-10) < 5 {
				t.Errorf("%s moves only %.1f m sideways", o.ID, math.Abs(end[0]-10))
			}
		}
	}
	// Serpentines through the whole arena are longer than the centre line and stay inside.
	s := optionOf(t, l, "schlangenlinie", "A–C", State{"A", DirR})
	if s.Length <= 40 {
		t.Errorf("serpentine length %.1f", s.Length)
	}
}

func TestFiguresChain(t *testing.T) {
	for _, size := range Sizes {
		l := lib(t, size)
		for _, o := range l.All {
			if o.End.Node == "DONE" {
				continue
			}
			if len(l.From(o.End)) == 0 {
				t.Errorf("%s %s: nothing follows %v", size, o.ID, o.End)
			}
		}
	}
}

func TestSpeeds(t *testing.T) {
	for g, r := range SpeedRanges {
		if r.Min <= 0 || r.Max <= r.Min {
			t.Errorf("%s range %v", g, r)
		}
		prev := 0.0
		for _, tempo := range []string{TempoCollected, TempoWorking, TempoMedium, TempoExtended} {
			v := Speed(g, tempo) * 60
			if v < r.Min-1e-9 || v > r.Max+1e-9 || v < prev {
				t.Errorf("%s %s = %.1f m/min outside %v or not rising", g, tempo, v, r)
			}
			prev = v
		}
	}
	if Speed(GaitHalt, TempoWorking) != 0 {
		t.Error("halt has a speed")
	}
	near(t, "working trot", Speed(GaitTrot, TempoWorking)*60, 192+104*0.25, 1e-9)
	if SpeedSource == "" || !strings.Contains(SpeedSource, "Back & Clayton") {
		t.Error("speed source missing")
	}
	// Duration of a track piece: length / speed.
	l := lib(t, Size20x40)
	lines, err := BuildLines(l, []Spec{{Option: optionOf(t, l, "track", "K–E", State{"K", DirR}).ID, Gait: GaitTrot, Tempo: TempoWorking}})
	if err != nil {
		t.Fatal(err)
	}
	near(t, "seconds K–E", lines[0].Seconds, 14/Speed(GaitTrot, TempoWorking), 0.06)
}

func TestSecondsWithHaltAndTransition(t *testing.T) {
	l := lib(t, Size20x40)
	in := &l.ByID["einreiten:X"].Length
	specs := []Spec{
		{Option: "einreiten:X", Gait: GaitTrot, Tempo: TempoWorking},
		{Option: optionOf(t, l, "track", "K–E", State{"K", DirR}).ID, Gait: GaitWalk, Tempo: TempoMedium},
	}
	lines, err := BuildLines(l, specs)
	if err != nil {
		t.Fatal(err)
	}
	near(t, "entry with salute", lines[0].Seconds, *in/Speed(GaitTrot, TempoWorking)+SaluteSeconds, 0.06)
	near(t, "walk after trot", lines[1].Seconds, 14/Speed(GaitWalk, TempoMedium)+TransitionSeconds, 0.06)
	if _, err := BuildLines(l, []Spec{{Option: "gibt-es-nicht", Gait: GaitTrot}}); err == nil {
		t.Error("unknown option accepted")
	}
	if _, err := BuildLines(l, []Spec{{Option: optionOf(t, l, "kurzkehrt", "E", State{"E", DirR}).ID, Gait: GaitCanter}}); err == nil {
		t.Error("Kurzkehrt in canter accepted")
	}
}

func TestMovementCatalog(t *testing.T) {
	seen := map[string]bool{}
	idRe := regexp.MustCompile(`^00000000-0000-4000-8000-0000000008\d\d$`)
	for _, m := range Movements {
		if seen[m.Key] {
			t.Errorf("duplicate movement %s", m.Key)
		}
		seen[m.Key] = true
		if m.Label == "" || m.Source == "" {
			t.Errorf("%s: label or source missing", m.Key)
		}
		if m.LowestClass != "" && ClassRank(m.LowestClass) < 0 {
			t.Errorf("%s: unknown class %q", m.Key, m.LowestClass)
		}
		if m.LowestClass == "" && (m.ClassNote != NotSourced || m.Rank() != ClassRank("L**")) {
			t.Errorf("%s: unsourced movement must be offered from L** with a note", m.Key)
		}
		if m.LowestClass != "" && (m.Source == "" || !strings.Contains(m.Source, "(") && !strings.Contains(m.Source, "de.wikipedia")) {
			t.Errorf("%s: source %q names no document", m.Key, m.Source)
		}
		for _, id := range m.ExerciseIDs {
			if !idRe.MatchString(id) {
				t.Errorf("%s: odd exercise id %s", m.Key, id)
			}
		}
		for _, f := range m.Figures {
			if !slices.Contains(allKinds(), f) {
				t.Errorf("%s: unknown figure kind %s", m.Key, f)
			}
		}
	}
	// A movement on the figures of the library must exist in the catalog.
	for _, size := range Sizes {
		for _, o := range lib(t, size).All {
			for _, g := range o.Gaits {
				for _, tempo := range []string{TempoWorking, TempoMedium, TempoFree} {
					for _, k := range lineMovements(&o, g, tempo) {
						if _, ok := MovementByKey(k); !ok {
							t.Fatalf("%s: line movement %q is not in the catalog", o.ID, k)
						}
					}
				}
			}
		}
	}
	// Spot checks of sourced classes.
	for key, want := range map[string]string{
		"volte_10_trab": "A", "volte_8_trab": "L", "volte_8_galopp": "L", "aussengalopp": "L",
		"mitteltrab": "A", "rueckwaertsrichten": "A", "schenkelweichen": "E", "fliegender_wechsel": "M*",
		"schulterherein": "L**", "einfacher_galoppwechsel": "L",
	} {
		m, ok := MovementByKey(key)
		if !ok || m.LowestClass != want {
			t.Errorf("%s: class %q, want %q", key, m.LowestClass, want)
		}
	}
	if m, _ := MovementByKey("versammelter_trab"); m.LowestClass != "" || m.Allowed("L") || !m.Allowed("L**") {
		t.Error("unsourced movement must be offered from L** only")
	}
}

func allKinds() []string {
	k := map[string]bool{}
	for _, size := range Sizes {
		for _, o := range lib2(size).All {
			k[o.Kind] = true
		}
	}
	var out []string
	for x := range k {
		out = append(out, x)
	}
	return out
}

func lib2(size string) *Library { l, _ := LibraryFor(size); return l }

// fnTestName matches names of official tests ("Aufgabe E", "Aufgabe A", "L 1", "E 11", "A 5").
var fnTestName = regexp.MustCompile(`\bAufgabe\s+(E|A|L|M|S)\*{0,3}\b|\b(E|A|L|M|S)\*{0,3}\s?\d{1,2}\b`)

func TestNoFNTestNamesInCatalog(t *testing.T) {
	var all []string
	c := Catalog()
	for _, m := range c.Movements {
		all = append(all, m.Key, m.Label, m.Source, m.ClassNote)
	}
	for _, k := range c.Classes {
		all = append(all, k.Key, k.Label)
	}
	for _, a := range c.Arenas {
		all = append(all, a.Label)
	}
	for _, l := range c.Links {
		all = append(all, l.Label)
	}
	all = append(all, c.SpeedSource, NotSourced)
	for _, size := range Sizes {
		l := lib(t, size)
		for _, o := range l.All {
			all = append(all, o.Verb, o.Full)
			for _, g := range o.Gaits {
				all = append(all, o.Text(g, TempoWorking))
			}
		}
	}
	for _, kind := range []string{KindPractice, KindFreestyle} {
		for _, class := range Classes {
			for _, size := range Sizes {
				all = append(all, Title(kind, class, size))
				if !regexp.MustCompile(`^(Übungsaufgabe|Kür) (E|A|L|L\*|L\*\*|M\*|M\*\*) · 20×(40|60)$`).MatchString(Title(kind, class, size)) {
					t.Errorf("odd title %q", Title(kind, class, size))
				}
			}
		}
	}
	for _, s := range all {
		if loc := fnTestName.FindString(s); loc != "" {
			t.Errorf("FN-style test name %q in %q", loc, s)
		}
	}
	// The pattern itself catches what it should.
	for _, bad := range []string{"Aufgabe E", "Aufgabe A", "L 1", "E 11", "Aufgabe L*"} {
		if !fnTestName.MatchString(bad) {
			t.Errorf("pattern misses %q", bad)
		}
	}
	// Generated plans: no line text names a test.
	p, err := GeneratePractice(PracticeParams{Class: "A", Arena: Size20x40, Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range p.Lines {
		if fnTestName.MatchString(line.Text) {
			t.Errorf("line text %q looks like an FN test name", line.Text)
		}
	}
}

func TestNoShippedRequirements(t *testing.T) {
	// The catalog carries classes and movements only: no required movements, no time windows.
	b, _ := json.Marshal(Catalog())
	for _, banned := range []string{"min_seconds", "max_seconds", "required", "requirements", "Pflichtlektion"} {
		if strings.Contains(string(b), banned) {
			t.Errorf("catalog contains %q", banned)
		}
	}
}

func rulesOK(t *testing.T, what string, p Plan) {
	t.Helper()
	for _, c := range p.Checks {
		if c.Rule && !c.OK {
			t.Errorf("%s: rule %s fails: %s", what, c.Key, c.Text)
		}
	}
}

func TestPracticeGenerator(t *testing.T) {
	for _, class := range PracticeClasses {
		for _, size := range Sizes {
			for _, length := range []string{LengthShort, LengthNormal} {
				lo, hi := PracticeWindow(class, length)
				for seed := int64(1); seed <= 6; seed++ {
					p, err := GeneratePractice(PracticeParams{Class: class, Arena: size, Length: length, Seed: seed})
					what := class + " " + size + " " + length
					if err != nil {
						t.Fatalf("%s seed %d: %v", what, seed, err)
					}
					rulesOK(t, what, p)
					if p.Seconds < lo-2.5 || p.Seconds > hi+2.5 {
						t.Errorf("%s seed %d: %.0f s outside %.0f–%.0f", what, seed, p.Seconds, lo, hi)
					}
					// Structure: entry at A, salute at X or G, salute at the end, both reins in balance.
					first, last := p.Lines[0], p.Lines[len(p.Lines)-1]
					if !strings.HasPrefix(first.Text, "Einreiten") || !slices.Contains(first.Movements, "halten_gruss") {
						t.Errorf("%s: first line %q", what, first.Text)
					}
					if last.Figure != kindAusreiten || p.Lines[len(p.Lines)-2].Figure != kindAbwenden {
						t.Errorf("%s: no salute at the end", what)
					}
					total := p.Reins.Left + p.Reins.Right
					if p.Reins.Left/total < 0.4 || p.Reins.Right/total < 0.4 {
						t.Errorf("%s seed %d: reins %.0f/%.0f", what, seed, p.Reins.Left, p.Reins.Right)
					}
					// Every gait on both reins; trot comes first and canter follows a trot phase.
					gr := gaitReins(lib(t, size), &p)
					for _, g := range []string{GaitWalk, GaitTrot, GaitCanter} {
						if gr[g][0] < 8 || gr[g][1] < 8 {
							t.Errorf("%s seed %d: %s reins %v", what, seed, g, gr[g])
						}
					}
					if p.Lines[1].Gait != GaitTrot {
						t.Errorf("%s: test does not start in trot", what)
					}
					firstCanter := slices.IndexFunc(p.Lines, func(l Line) bool { return l.Gait == GaitCanter })
					if firstCanter < 1 || p.Lines[firstCanter-1].Gait != GaitTrot {
						t.Errorf("%s: canter does not follow a trot phase", what)
					}
					// Class: no movement above it; free walk only in the classes E and A structure.
					for _, l := range p.Lines {
						for _, k := range l.Movements {
							if m, _ := MovementByKey(k); !m.Allowed(class) {
								t.Errorf("%s: %s (class %q) in a class %s test", what, k, m.LowestClass, class)
							}
						}
					}
					if p.Music[0].Gait != GaitTrot {
						t.Errorf("%s: music starts with %s", what, p.Music[0].Gait)
					}
				}
			}
		}
	}
}

func TestPracticeChainsLines(t *testing.T) {
	for _, size := range Sizes {
		l := lib(t, size)
		p, err := GeneratePractice(PracticeParams{Class: "L", Arena: size, Seed: 11})
		if err != nil {
			t.Fatal(err)
		}
		if p.Lines[0].Start.Node != "OUT" {
			t.Errorf("first line starts at %v", p.Lines[0].Start)
		}
		for i := 1; i < len(p.Lines); i++ {
			if p.Lines[i].Start != p.Lines[i-1].End {
				t.Errorf("line %d starts at %v, line %d ends at %v", i+1, p.Lines[i].Start, i, p.Lines[i-1].End)
			}
			// The drawn paths touch as well.
			a, b := p.Lines[i-1].Path, p.Lines[i].Path
			if d := dist(a[len(a)-1], b[0]); d > 0.1 {
				t.Errorf("line %d path starts %.2f m from the end of line %d", i+1, d, i)
			}
		}
		if len(p.Lines) < 10 || l == nil {
			t.Errorf("only %d lines", len(p.Lines))
		}
		near(t, "plan seconds are the sum of the lines", p.Seconds, sumSeconds(p), 0.2)
		var music float64
		for _, m := range p.Music {
			music += m.Seconds
		}
		near(t, "music parts add up", music, p.Seconds, 0.2)
	}
}

func sumSeconds(p Plan) float64 {
	s := 0.0
	for _, l := range p.Lines {
		s += l.Seconds
	}
	return s
}

func TestPracticeDeterministicBySeed(t *testing.T) {
	gen := func(seed int64) string {
		p, err := GeneratePractice(PracticeParams{Class: "A", Arena: Size20x40, Length: LengthNormal, Seed: seed})
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(p)
		return string(b)
	}
	if gen(5) != gen(5) {
		t.Error("same seed gave different plans")
	}
	distinct := map[string]bool{}
	for s := int64(1); s <= 6; s++ {
		distinct[gen(s)] = true
	}
	if len(distinct) < 5 {
		t.Errorf("only %d different plans in 6 seeds", len(distinct))
	}
}

func TestPracticeOnlyMasteredPlusNext(t *testing.T) {
	// A horse with sessions: volte and schenkelweichen mastered, Schlangenlinie and Rückwärtsrichten
	// are the next lessons, nothing else counts.
	h := Hints{HasSessions: true,
		Mastered: map[string]bool{ex("805")[0]: true, ex("804")[0]: true},
		Next:     map[string]bool{ex("802")[0]: true, ex("806")[0]: true, ex("813")[0]: true},
	}
	for seed := int64(1); seed <= 8; seed++ {
		p, err := GeneratePractice(PracticeParams{Class: "L", Arena: Size20x40, Seed: seed, Hints: h})
		if err != nil {
			t.Fatal(err)
		}
		next := map[string]bool{}
		for _, l := range p.Lines {
			for _, k := range l.Movements {
				m, _ := MovementByKey(k)
				if m.Basis || len(m.ExerciseIDs) == 0 {
					continue
				}
				mastered, isNext := false, false
				for _, id := range m.ExerciseIDs {
					mastered = mastered || h.Mastered[id]
					isNext = isNext || h.Next[id]
				}
				switch {
				case mastered:
				case isNext:
					next[k] = true
				default:
					t.Errorf("seed %d: %s is neither mastered nor next", seed, k)
				}
			}
		}
		if len(next) > 2 {
			t.Errorf("seed %d: %d next movements %v, at most 2", seed, len(next), next)
		}
	}
	// Without sessions every movement of the class is allowed: a class L test can show the movements of L.
	seen := false
	for seed := int64(1); seed <= 20 && !seen; seed++ {
		p, _ := GeneratePractice(PracticeParams{Class: "L", Arena: Size20x40, Seed: seed})
		for _, l := range p.Lines {
			seen = seen || slices.Contains(l.Movements, "einfacher_galoppwechsel") || slices.Contains(l.Movements, "volte_8_galopp")
		}
	}
	if !seen {
		t.Error("no movement of class L in 20 class L tests without sessions")
	}
}

func TestPracticeFocusPrefersMovements(t *testing.T) {
	count := func(h Hints) int {
		n := 0
		for seed := int64(1); seed <= 12; seed++ {
			p, err := GeneratePractice(PracticeParams{Class: "A", Arena: Size20x40, Seed: seed, Hints: h})
			if err != nil {
				t.Fatal(err)
			}
			for _, l := range p.Lines {
				if slices.Contains(l.Movements, "rueckwaertsrichten") {
					n++
				}
			}
		}
		return n
	}
	with := count(Hints{FocusExercises: map[string]bool{ex("806")[0]: true}})
	without := count(Hints{})
	if with <= without {
		t.Errorf("focus on the rein back exercise: %d lines with, %d without", with, without)
	}
}

func TestPracticeRejectsUnknownInput(t *testing.T) {
	if _, err := GeneratePractice(PracticeParams{Class: "M**", Arena: Size20x40}); err == nil {
		t.Error("class without practice test accepted")
	}
	if _, err := GeneratePractice(PracticeParams{Class: "E", Arena: "30x60"}); err == nil {
		t.Error("unknown arena accepted")
	}
}

// fixture: a requirement set made up for the tests (not an official one).
func fixtureRequirement() Requirement {
	return Requirement{Class: "L*", Label: "Eigene Pflichtlektionen", MinSeconds: 200, MaxSeconds: 420, Movements: []ReqMovement{
		{Key: "mittelschritt", Count: 1, MinMeters: 30},
		{Key: "volte_10_trab", Count: 1, Rein: "both"},
		{Key: "mitteltrab", Count: 1},
		{Key: "kurzkehrt", Count: 1},
		{Key: "volte_8_galopp", Count: 1, Rein: "both"},
		{Key: "einfacher_galoppwechsel", Count: 2},
		{Key: "versammelter_trab", Count: 1},
	}}
}

func TestFreestyleGenerator(t *testing.T) {
	req := fixtureRequirement()
	for _, size := range Sizes {
		for seed := int64(1); seed <= 4; seed++ {
			p, err := GenerateFreestyle(FreestyleParams{Class: "L*", Arena: size, Seed: seed, Requirement: req})
			if err != nil {
				t.Fatalf("%s seed %d: %v", size, seed, err)
			}
			rulesOK(t, size, p)
			for _, s := range p.Requirements {
				if !s.Met || s.Shown < s.Count {
					t.Errorf("%s seed %d: %s shown %d of %d", size, seed, s.Label, s.Shown, s.Count)
				}
			}
			if len(p.Requirements) != 9 { // volte_10_trab and volte_8_galopp count once per rein
				t.Errorf("%d requirement rows", len(p.Requirements))
			}
			if p.Seconds < 200-timeTolerance || p.Seconds > 420+timeTolerance {
				t.Errorf("%s seed %d: %.0f s outside the window", size, seed, p.Seconds)
			}
			if len(p.Music) < 3 {
				t.Errorf("music parts %v", p.Music)
			}
			// Music: gait per part, cadence from the source.
			for _, m := range p.Music {
				if m.Cadence[0] == 0 {
					t.Errorf("music part %v without cadence", m)
				}
			}
			// Choreography is rated in plain words.
			var haveBalance bool
			for _, c := range p.Checks {
				if c.Key == "balance" && strings.Contains(c.Text, "links") && strings.Contains(c.Text, "rechts") {
					haveBalance = true
				}
			}
			if !haveBalance {
				t.Error("no balance sentence")
			}
		}
	}
	a, _ := GenerateFreestyle(FreestyleParams{Class: "L*", Arena: Size20x40, Seed: 9, Requirement: req})
	b, _ := GenerateFreestyle(FreestyleParams{Class: "L*", Arena: Size20x40, Seed: 9, Requirement: req})
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Error("freestyle is not deterministic by seed")
	}
}

func TestFreestyleWithoutRequirements(t *testing.T) {
	p, err := GenerateFreestyle(FreestyleParams{Class: "A", Arena: Size20x40, Seed: 2, Requirement: Requirement{Class: "A"}})
	if err != nil {
		t.Fatal(err)
	}
	rulesOK(t, "A", p)
	if len(p.Requirements) != 0 {
		t.Error("requirements without a set")
	}
}

func TestValidateRequirement(t *testing.T) {
	if errs := ValidateRequirement(fixtureRequirement()); len(errs) != 0 {
		t.Errorf("fixture invalid: %v", errs)
	}
	for name, r := range map[string]Requirement{
		"class":      {Class: "Z"},
		"movement":   {Class: "A", Movements: []ReqMovement{{Key: "gibt-es-nicht", Count: 1}}},
		"count":      {Class: "A", Movements: []ReqMovement{{Key: "arbeitstrab", Count: 0}}},
		"rein":       {Class: "A", Movements: []ReqMovement{{Key: "arbeitstrab", Count: 1, Rein: "oben"}}},
		"min meters": {Class: "A", Movements: []ReqMovement{{Key: "arbeitstrab", Count: 1, MinMeters: 40}}},
		"time":       {Class: "A", MinSeconds: 300, MaxSeconds: 200},
		"duplicate":  {Class: "A", Movements: []ReqMovement{{Key: "arbeitstrab", Count: 1}, {Key: "arbeitstrab", Count: 2}}},
	} {
		if len(ValidateRequirement(r)) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := GenerateFreestyle(FreestyleParams{Class: "A", Arena: Size20x40, Requirement: Requirement{Class: "A", Movements: []ReqMovement{{Key: "x", Count: 1}}}}); err == nil {
		t.Error("generator accepted an invalid requirement")
	}
}
