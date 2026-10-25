package dressage

import "testing"

func TestMergeTracks(t *testing.T) {
	l := lib(t, Size20x40)
	var a, b Spec
	for _, o := range l.From(State{"H", DirR}) {
		if o.Kind == "track" && o.End.Node == "C" {
			a = Spec{Option: o.ID, Gait: GaitTrot, Tempo: TempoWorking}
		}
	}
	for _, o := range l.From(State{"C", DirR}) {
		if o.Kind == "track" && o.End.Node == "M" {
			b = Spec{Option: o.ID, Gait: GaitTrot, Tempo: TempoWorking}
		}
	}
	if a.Option == "" || b.Option == "" {
		t.Fatal("no track options H–C and C–M")
	}
	got := mergeTracks(l, []Spec{a, b})
	if len(got) != 1 || l.ByID[got[0].Option].Letters != "H–M" || l.ByID[got[0].Option].Kind != "track" {
		t.Fatalf("merged %v", got)
	}
	// A different gait is no merge.
	b.Gait = GaitCanter
	if got := mergeTracks(l, []Spec{a, b}); len(got) != 2 {
		t.Errorf("tracks in different gaits merged: %v", got)
	}
}

func TestFreestyleMergesTracks(t *testing.T) {
	for _, size := range Sizes {
		for seed := int64(1); seed <= 6; seed++ {
			p, err := GenerateFreestyle(FreestyleParams{Class: "L*", Arena: size, Seed: seed, Requirement: fixtureRequirement()})
			if err != nil {
				t.Fatal(err)
			}
			for i := 1; i < len(p.Lines); i++ {
				a, b := p.Lines[i-1], p.Lines[i]
				if a.Figure != "track" || b.Figure != "track" || a.Gait != b.Gait || a.Tempo != b.Tempo {
					continue
				}
				// Two track lines in a row are left only when four letters do not cover them.
				if _, ok := mergeTrackPair(lib(t, size), a.Spec(), b.Spec()); ok {
					t.Errorf("%s seed %d: lines %d and %d are track lines that merge (%s, %s)", size, seed, a.N, b.N, a.Letters, b.Letters)
				}
			}
		}
	}
}
