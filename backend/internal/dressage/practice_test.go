package dressage

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// runs groups consecutive lines of one gait.
func gaitRuns(lines []Line) [][]Line {
	var out [][]Line
	for _, l := range lines {
		if n := len(out); n > 0 && out[n-1][0].Gait == l.Gait {
			out[n-1] = append(out[n-1], l)
			continue
		}
		out = append(out, []Line{l})
	}
	return out
}

func isChangeLine(l Line, lib *Library) bool {
	o := lib.ByID[l.Option]
	return o != nil && o.Changes && l.Figure != kindAbwenden && l.Figure != kindAusreiten
}

// assertPracticeQuality checks the properties of a practice test a trainer expects.
func assertPracticeQuality(t *testing.T, what string, p Plan, class, size, length string) {
	t.Helper()
	l := lib(t, size)
	rulesOK(t, what, p)

	// Line count.
	if limit := MaxPracticeLines(length); len(p.Lines) > limit {
		t.Errorf("%s: %d lines, at most %d", what, len(p.Lines), limit)
	}
	// Track-only lines are merged: no two track lines in a row in the same gait.
	for i := 1; i < len(p.Lines); i++ {
		a, b := p.Lines[i-1], p.Lines[i]
		if a.Figure == "track" && b.Figure == "track" && a.Gait == b.Gait && a.Tempo == b.Tempo {
			t.Errorf("%s: lines %d and %d are both track lines (%s, %s)", what, a.N, b.N, a.Letters, b.Letters)
		}
		if a.Option == b.Option {
			t.Errorf("%s: line %d repeats line %d", what, b.N, a.N)
		}
	}
	// Duration and reins.
	lo, hi := PracticeWindow(class, length)
	if p.Seconds < lo || p.Seconds > hi {
		t.Errorf("%s: %.0f s outside %.0f–%.0f", what, p.Seconds, lo, hi)
	}
	if share := p.Reins.Left / (p.Reins.Left + p.Reins.Right); share < 0.4 || share > 0.6 {
		t.Errorf("%s: left rein %.0f %%", what, share*100)
	}

	// Phases: trot, walk, trot, canter, trot, canter, trot, leave in walk.
	runs := gaitRuns(p.Lines)
	var order []string
	for _, r := range runs {
		order = append(order, r[0].Gait)
	}
	want := []string{GaitTrot, GaitWalk, GaitTrot, GaitCanter, GaitTrot, GaitCanter, GaitTrot, GaitWalk}
	if !slices.Equal(order, want) {
		t.Fatalf("%s: gait order %v, want %v", what, order, want)
	}
	trot1, walk, conn, canter1, change, canter2, trot2 := runs[0], runs[1], runs[2], runs[3], runs[4], runs[5], runs[6]
	changes := func(r []Line) (n int) {
		for _, x := range r {
			if isChangeLine(x, l) {
				n++
			}
		}
		return
	}
	nonTrack := func(r []Line) (n int) {
		for _, x := range r {
			if x.Figure != "track" {
				n++
			}
		}
		return
	}
	// Trot on one rein, then one change of rein and work on the other.
	if changes(trot1) != 1 {
		t.Errorf("%s: first trot part has %d changes of rein", what, changes(trot1))
	}
	ci := slices.IndexFunc(trot1, func(x Line) bool { return isChangeLine(x, l) })
	if ci < 3 {
		t.Errorf("%s: no work on the first rein before the change of rein (line %d)", what, ci+1)
	}
	if after := trot1[ci+1:]; len(after) == 0 || nonTrack(after) == 0 {
		t.Errorf("%s: no figure on the second rein after the change of rein", what)
	}
	dir1 := trot1[2].Start.Dir
	dir2 := oppositeDir(dir1)
	if trot1[len(trot1)-1].End.Dir != dir2 {
		t.Errorf("%s: trot part ends on %s, want %s", what, trot1[len(trot1)-1].End.Dir, dir2)
	}
	// Walk: medium walk, free walk across the arena, medium walk.
	if len(walk) < 3 || walk[0].Tempo != TempoMedium || walk[len(walk)-1].Tempo != TempoMedium {
		t.Errorf("%s: walk part is not medium–free–medium", what)
	}
	free := 0
	for _, x := range walk {
		if x.Tempo == TempoFree {
			free++
			if !isChangeLine(x, l) {
				t.Errorf("%s: free walk is not across the arena (%s)", what, x.Letters)
			}
			if class == "E" || class == "A" {
				if x.Figure != "diagonal" {
					t.Errorf("%s: free walk in class %s is on %s, not on a diagonal", what, class, x.Figure)
				}
			}
		}
	}
	if free != 1 || changes(walk) != 1 {
		t.Errorf("%s: %d free walk lines, %d changes in walk", what, free, changes(walk))
	}
	// Trot to the letter where the canter starts: track only.
	if len(conn) != 1 || conn[0].Figure != "track" {
		t.Errorf("%s: trot before the canter has %d lines", what, len(conn))
	}
	// Canter: two blocks, no gait change inside (the runs are single), each with at most 2 figures.
	if changes(canter1) != 0 {
		t.Errorf("%s: first canter block changes the rein", what)
	}
	if nonTrack(canter1) > 2 || nonTrack(canter2) > 2 || nonTrack(canter1) == 0 || nonTrack(canter2) == 0 {
		t.Errorf("%s: canter figures %d and %d, want 1 or 2 each", what, nonTrack(canter1), nonTrack(canter2))
	}
	if canter1[0].Start.Dir != dir1 {
		t.Errorf("%s: first canter block starts on %s, want %s", what, canter1[0].Start.Dir, dir1)
	}
	if canter2[0].Start.Dir != dir2 {
		t.Errorf("%s: second canter block starts on %s, want %s", what, canter2[0].Start.Dir, dir2)
	}
	if n := changes(canter2); n > 1 || n == 1 && ClassRank(class) < ClassRank("L") {
		t.Errorf("%s: %d changes of lead in the second canter block", what, n)
	}
	// Trot change of rein between the canters, one change in it.
	if changes(change) != 1 {
		t.Errorf("%s: trot between the canter blocks has %d changes of rein", what, changes(change))
	}
	if len(trot2) < 2 || changes(trot2) != 0 {
		t.Errorf("%s: final trot has %d changes of rein", what, changes(trot2))
	}
	if last := trot2[len(trot2)-1]; last.Figure != kindAbwenden {
		t.Errorf("%s: trot part does not end with the salute (%s)", what, last.Figure)
	}

	// Limits per movement, lateral work and serpentines.
	counts := map[string]int{}
	lateral := map[string]int{}
	serp, extras := 0, map[string]map[string]bool{}
	for _, x := range p.Lines {
		for _, k := range x.Movements {
			if !IsCountedMovement(k) {
				continue
			}
			counts[x.Gait+"|"+k]++
			if counts[x.Gait+"|"+k] > 2 {
				t.Errorf("%s: %s in %s more than twice", what, k, x.Gait)
			}
			if slices.Contains(structuralKeys, k) || slices.Contains(coreMovements[class][x.Gait], k) {
				continue
			}
			if extras[x.Gait] == nil {
				extras[x.Gait] = map[string]bool{}
			}
			extras[x.Gait][k] = true
		}
		if x.Lateral {
			lateral[x.Rein]++
			if lateral[x.Rein] > 2 {
				t.Errorf("%s: more than 2 lateral lines on the %s rein", what, x.Rein)
			}
		}
		if isSerpentine(x.Figure) {
			serp++
		}
	}
	if serp > 1 {
		t.Errorf("%s: %d serpentines", what, serp)
	}
	// A short test shows the core of the class plus at most 2 extras per gait, a normal one up to 4.
	limit := 2
	if length != LengthShort {
		limit = 4
	}
	for gait, set := range extras {
		// A change of lead belongs to the canter of the class and counts with its figure.
		if len(set) > limit {
			t.Errorf("%s: %s shows %d movements beyond the core of class %s: %v", what, gait, len(set), class, set)
		}
	}
}

func TestPracticeQuality(t *testing.T) {
	for _, class := range PracticeClasses {
		for _, size := range Sizes {
			for _, length := range []string{LengthShort, LengthNormal} {
				for seed := int64(1); seed <= 10; seed++ {
					what := fmt.Sprintf("%s %s %s seed %d", class, size, length, seed)
					p, err := GeneratePractice(PracticeParams{Class: class, Arena: size, Length: length, Seed: seed})
					if err != nil {
						t.Fatalf("%s: %v", what, err)
					}
					assertPracticeQuality(t, what, p, class, size, length)
				}
			}
		}
	}
}

func TestPracticeQualityWithHorseHints(t *testing.T) {
	hints := []Hints{
		{HasSessions: true},
		{HasSessions: true, Mastered: map[string]bool{ex("805")[0]: true, ex("804")[0]: true},
			Next: map[string]bool{ex("802")[0]: true, ex("806")[0]: true, ex("813")[0]: true}},
		{FocusExercises: map[string]bool{ex("806")[0]: true, ex("811")[0]: true, ex("815")[0]: true}},
	}
	for hi, h := range hints {
		for _, class := range PracticeClasses {
			for _, size := range Sizes {
				for _, length := range []string{LengthShort, LengthNormal} {
					for seed := int64(1); seed <= 4; seed++ {
						what := fmt.Sprintf("hints %d %s %s %s seed %d", hi, class, size, length, seed)
						p, err := GeneratePractice(PracticeParams{Class: class, Arena: size, Length: length, Seed: seed, Hints: h})
						if err != nil {
							t.Fatalf("%s: %v", what, err)
						}
						assertPracticeQuality(t, what, p, class, size, length)
					}
				}
			}
		}
	}
}

func TestPracticeVariesBySeed(t *testing.T) {
	for _, class := range PracticeClasses {
		for _, size := range Sizes {
			for _, length := range []string{LengthShort, LengthNormal} {
				plans := map[string]bool{}
				for seed := int64(1); seed <= 10; seed++ {
					p, err := GeneratePractice(PracticeParams{Class: class, Arena: size, Length: length, Seed: seed})
					if err != nil {
						t.Fatal(err)
					}
					again, _ := GeneratePractice(PracticeParams{Class: class, Arena: size, Length: length, Seed: seed})
					a, _ := json.Marshal(p)
					b, _ := json.Marshal(again)
					if string(a) != string(b) {
						t.Errorf("%s %s %s seed %d: not deterministic", class, size, length, seed)
					}
					var ids []string
					for _, x := range p.Lines {
						ids = append(ids, x.Option+"/"+x.Gait+"/"+x.Tempo)
					}
					plans[strings.Join(ids, ";")] = true
				}
				if len(plans) < 9 {
					t.Errorf("%s %s %s: only %d different plans in 10 seeds", class, size, length, len(plans))
				}
			}
		}
	}
}
