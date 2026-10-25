package dressage

import (
	"slices"
	"strings"
	"testing"
)

// sp builds a spec; the id of a library option is found by kind, letters and start state.
func sp(id, gait, tempo string) Spec { return Spec{Option: id, Gait: gait, Tempo: tempo} }

func trotAll(ids ...string) []Spec {
	var out []Spec
	for _, id := range ids {
		out = append(out, sp(id, GaitTrot, TempoWorking))
	}
	return out
}

func eval(t *testing.T, kind, class string, req Requirement, specs []Spec) Plan {
	t.Helper()
	p, err := Evaluate(kind, class, Size20x40, 1, specs, req)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	return p
}

func failing(p Plan) map[string]string {
	out := map[string]string{}
	for _, c := range p.Checks {
		if !c.OK {
			out[c.Key] = c.Text
		}
	}
	return out
}

func idOf(t *testing.T, kind, letters string, start State) string {
	t.Helper()
	return optionOf(t, lib(t, Size20x40), kind, letters, start).ID
}

// route builds the lines of a short test around the given middle part: entry, salute at X, to C and
// onto the rein dir, the middle part (specs), then from A the salute at X and the exit.
func route(dir string, middle []Spec) []Spec { return routeTo(dir, dir, middle) }

// routeTo is route with another rein at the end.
func routeTo(dir, endDir string, middle []Spec) []Spec {
	open := trotAll("einreiten:X", "weiter:X:"+dir)
	closing := []Spec{sp("abwenden_halt:X:"+endDir, GaitTrot, TempoWorking), sp("ausreiten:X", GaitWalk, TempoFree)}
	return slices.Concat(open, middle, closing)
}

func TestCheckStructure(t *testing.T) {
	l := lib(t, Size20x40)
	trackCA := idOf(t, "track", "C–A", State{"C", DirR})
	good := eval(t, KindFreestyle, "L**", Requirement{}, route(DirR, trotAll(trackCA)))
	for _, k := range []string{"chain", "opening", "closing"} {
		if f := failing(good); f[k] != "" {
			t.Errorf("good plan fails %s: %s", k, f[k])
		}
	}
	// No entry at A: a salute at the start on the centre line is required.
	noOpen := eval(t, KindFreestyle, "L**", Requirement{}, route(DirR, trotAll(trackCA))[1:])
	if f := failing(noOpen); f["opening"] == "" || f["chain"] == "" {
		t.Errorf("missing entry not reported: %v", f)
	}
	// No salute at the end.
	all := route(DirR, trotAll(trackCA))
	noClose := eval(t, KindFreestyle, "L**", Requirement{}, all[:len(all)-2])
	if f := failing(noClose); f["closing"] == "" {
		t.Errorf("missing salute at the end not reported: %v", f)
	}
	// A line that does not start where the last one ended.
	broken := route(DirR, trotAll(idOf(t, "track", "C–M", State{"C", DirR}), idOf(t, "track", "B–F", State{"B", DirR}), idOf(t, "track", "F–A", State{"F", DirR})))
	f := failing(eval(t, KindPractice, "L", Requirement{}, broken))
	if !strings.Contains(f["chain"], "beginnt nicht dort, wo Zeile 3 endet") {
		t.Errorf("chain message = %q", f["chain"])
	}
	if errs := RuleErrors(eval(t, KindPractice, "L", Requirement{}, broken).Checks); len(errs) == 0 {
		t.Error("RuleErrors empty for a broken chain")
	}
	_ = l
}

func TestCheckFreestyleGaits(t *testing.T) {
	trackCA := idOf(t, "track", "C–A", State{"C", DirR})
	p := eval(t, KindFreestyle, "L**", Requirement{}, route(DirR, trotAll(trackCA)))
	if f := failing(p); f["gaits"] == "" || !strings.Contains(f["gaits"], "Schritt") || !strings.Contains(f["gaits"], "Galopp") {
		t.Errorf("trot only: %v", f)
	}
	// All three gaits: walk and canter on the track pieces.
	specs := []Spec{
		sp(idOf(t, "track", "C–M", State{"C", DirR}), GaitTrot, TempoWorking),
		sp(idOf(t, "track", "M–B", State{"M", DirR}), GaitWalk, TempoMedium),
		sp(idOf(t, "track", "B–A", State{"B", DirR}), GaitCanter, TempoWorking),
	}
	if f := failing(eval(t, KindFreestyle, "L**", Requirement{}, route(DirR, specs))); f["gaits"] != "" {
		t.Errorf("three gaits: %s", f["gaits"])
	}
}

func TestCheckFreestyleWalkOnlyOnStraightOrCurvedLines(t *testing.T) {
	req := Requirement{Class: "L*", Movements: []ReqMovement{{Key: "mittelschritt", Count: 1, MinMeters: 40}}}
	// Walk only as "Viereck verkleinern": fails.
	viereck := idOf(t, "viereck", "M–F", State{"M", DirR})
	bad := []Spec{
		sp(idOf(t, "track", "C–M", State{"C", DirR}), GaitTrot, TempoWorking),
		sp(viereck, GaitWalk, TempoMedium),
		sp(idOf(t, "track", "F–A", State{"F", DirR}), GaitTrot, TempoWorking),
	}
	if f := failing(eval(t, KindFreestyle, "L*", req, route(DirR, bad))); !strings.Contains(f["walk"], "Seitengang oder beim Verkleinern") {
		t.Errorf("walk as Viereck verkleinern: %q", f["walk"])
	}
	// Walk on a track piece that is too short (14 m) fails, 57 m passes.
	short := []Spec{
		sp(idOf(t, "track", "C–M", State{"C", DirR}), GaitTrot, TempoWorking),
		sp(idOf(t, "track", "M–B", State{"M", DirR}), GaitWalk, TempoMedium),
		sp(idOf(t, "track", "B–A", State{"B", DirR}), GaitTrot, TempoWorking),
	}
	if f := failing(eval(t, KindFreestyle, "L*", req, route(DirR, short))); !strings.Contains(f["walk"], "zu kurz") {
		t.Errorf("short walk: %q", f["walk"])
	}
	long := []Spec{sp(idOf(t, "track", "C–A", State{"C", DirR}), GaitWalk, TempoMedium)}
	if f := failing(eval(t, KindFreestyle, "L*", req, route(DirR, long))); f["walk"] != "" {
		t.Errorf("long walk: %q", f["walk"])
	}
}

func TestCheckFreestyleLengtheningOnStraightLinesOnly(t *testing.T) {
	// Medium trot on a circle: fails. On the long side between two letters: passes.
	circle := []Spec{
		sp(idOf(t, "zirkel", "C", State{"C", DirR}), GaitTrot, TempoMedium),
		sp(idOf(t, "track", "C–A", State{"C", DirR}), GaitTrot, TempoWorking),
	}
	if f := failing(eval(t, KindFreestyle, "L**", Requirement{}, route(DirR, circle))); !strings.Contains(f["lengthening"], "nicht auf Kreise") {
		t.Errorf("lengthening on a circle: %q", f["lengthening"])
	}
	// Through two corners (track from H to M around C): not a straight line either.
	corners := []Spec{sp(idOf(t, "track", "C–B", State{"C", DirR}), GaitTrot, TempoMedium), sp(idOf(t, "track", "B–A", State{"B", DirR}), GaitTrot, TempoWorking)}
	if f := failing(eval(t, KindFreestyle, "L**", Requirement{}, route(DirR, corners))); f["lengthening"] == "" {
		t.Error("lengthening through corners accepted")
	}
	diag := []Spec{
		sp(idOf(t, "track", "C–M", State{"C", DirR}), GaitTrot, TempoWorking),
		sp(idOf(t, "track", "M–F", State{"M", DirR}), GaitTrot, TempoMedium),
		sp(idOf(t, "track", "F–A", State{"F", DirR}), GaitTrot, TempoWorking),
	}
	if f := failing(eval(t, KindFreestyle, "L**", Requirement{}, route(DirR, diag))); f["lengthening"] != "" {
		t.Errorf("lengthening on a straight line: %q", f["lengthening"])
	}
}

func TestCheckFreestyleCanterVoltesInTrueCanter(t *testing.T) {
	specs := []Spec{
		sp(idOf(t, "volte8", "C", State{"C", DirR}), GaitCanter, TempoWorking),
		sp(idOf(t, "track", "C–A", State{"C", DirR}), GaitTrot, TempoWorking),
	}
	l := lib(t, Size20x40)
	p := eval(t, KindFreestyle, "L*", Requirement{}, route(DirR, specs))
	if f := failing(p); f["canter_voltes"] != "" {
		t.Errorf("volte in true canter: %q", f["canter_voltes"])
	}
	// The same line ridden in counter canter breaks the rule.
	bad := p
	bad.Lines = slices.Clone(p.Lines)
	for i := range bad.Lines {
		if bad.Lines[i].Figure == "volte8" {
			bad.Lines[i].Counter = true
		}
	}
	checks, _ := CheckFreestyle(l, "L*", Requirement{}, &bad)
	if c := find(checks, "canter_voltes"); c.OK || !strings.Contains(c.Text, "Handgalopp") {
		t.Errorf("volte in counter canter: %+v", c)
	}
}

func find(checks []Check, key string) Check {
	for _, c := range checks {
		if c.Key == key {
			return c
		}
	}
	return Check{}
}

func TestCheckFreestyleKurzkehrtNotMoreThan180(t *testing.T) {
	specs := []Spec{
		sp(idOf(t, "track", "C–E", State{"C", DirL}), GaitWalk, TempoMedium),
		sp(idOf(t, "kurzkehrt", "E", State{"E", DirL}), GaitWalk, TempoMedium),
		sp(idOf(t, "track", "E–B", State{"E", DirR}), GaitTrot, TempoWorking),
		sp(idOf(t, "track", "B–A", State{"B", DirR}), GaitTrot, TempoWorking),
	}
	p := eval(t, KindFreestyle, "L*", Requirement{}, routeTo(DirL, DirR, specs))
	if f := failing(p); f["kurzkehrt"] != "" || f["chain"] != "" {
		t.Errorf("Kurzkehrt of 180°: %v", f)
	}
	bad := p
	bad.Lines = slices.Clone(p.Lines)
	for i := range bad.Lines {
		if bad.Lines[i].Figure == "kurzkehrt" {
			bad.Lines[i].Degrees = 270
		}
	}
	checks, _ := CheckFreestyle(lib(t, Size20x40), "L*", Requirement{}, &bad)
	if c := find(checks, "kurzkehrt"); c.OK || !strings.Contains(c.Text, "180°") {
		t.Errorf("Kurzkehrt of 270°: %+v", c)
	}
}

func TestCheckFreestyleNoMovementOfAHigherClass(t *testing.T) {
	specs := []Spec{
		sp(idOf(t, "track", "C–E", State{"C", DirL}), GaitTrot, TempoWorking),
		sp(idOf(t, "schulterherein", "E–K", State{"E", DirL}), GaitTrot, TempoWorking),
		sp(idOf(t, "track", "K–A", State{"K", DirL}), GaitTrot, TempoWorking),
	}
	route := route(DirL, specs)
	if f := failing(eval(t, KindFreestyle, "L*", Requirement{}, route)); !strings.Contains(f["class"], "Schulterherein") || !strings.Contains(f["class"], "ab L**") {
		t.Errorf("shoulder-in in L*: %q", f["class"])
	}
	if f := failing(eval(t, KindFreestyle, "L**", Requirement{}, route)); f["class"] != "" {
		t.Errorf("shoulder-in in L**: %q", f["class"])
	}
	// What the user requires is allowed whatever the class.
	req := Requirement{Class: "L*", Movements: []ReqMovement{{Key: "schulterherein", Count: 1}}}
	if f := failing(eval(t, KindFreestyle, "L*", req, route)); f["class"] != "" || f["requirements"] != "" {
		t.Errorf("required shoulder-in: %v", f)
	}
	// A movement without a source for its class is offered from L** only.
	collected := []Spec{sp(idOf(t, "track", "C–A", State{"C", DirR}), GaitTrot, TempoCollected)}
	if f := failing(eval(t, KindFreestyle, "L*", Requirement{}, route2(DirR, collected))); !strings.Contains(f["class"], NotSourced) {
		t.Errorf("collected trot in L*: %q", f["class"])
	}
	if f := failing(eval(t, KindFreestyle, "L**", Requirement{}, route2(DirR, collected))); f["class"] != "" {
		t.Errorf("collected trot in L**: %q", f["class"])
	}
	// Practice tests follow the same class rule.
	if f := failing(eval(t, KindPractice, "E", Requirement{}, route)); f["class"] == "" {
		t.Error("shoulder-in in a class E practice test accepted")
	}
}

func route2(dir string, m []Spec) []Spec { return route(dir, m) }

func TestCheckFreestyleTimeWindow(t *testing.T) {
	specs := route(DirR, trotAll(idOf(t, "track", "C–A", State{"C", DirR})))
	p := eval(t, KindFreestyle, "L*", Requirement{Class: "L*", MinSeconds: 600, MaxSeconds: 700}, specs)
	if f := failing(p); !strings.Contains(f["time"], "Mehr als 10 Sekunden außerhalb") {
		t.Errorf("too short: %q", f["time"])
	}
	win := Requirement{Class: "L*", MinSeconds: int(p.Seconds) - 20, MaxSeconds: int(p.Seconds) + 20}
	if f := failing(eval(t, KindFreestyle, "L*", win, specs)); f["time"] != "" {
		t.Errorf("inside the window: %q", f["time"])
	}
	// Up to 10 seconds outside costs nothing.
	edge := Requirement{Class: "L*", MinSeconds: int(p.Seconds) + 8, MaxSeconds: int(p.Seconds) + 80}
	if f := failing(eval(t, KindFreestyle, "L*", edge, specs)); f["time"] != "" {
		t.Errorf("8 s short: %q", f["time"])
	}
	long := Requirement{Class: "L*", MinSeconds: 30, MaxSeconds: int(p.Seconds) - 30}
	if f := failing(eval(t, KindFreestyle, "L*", long, specs)); f["time"] == "" {
		t.Error("too long accepted")
	}
}

func TestCheckFreestyleRequirements(t *testing.T) {
	specs := route(DirR, []Spec{
		sp(idOf(t, "volte10", "C", State{"C", DirR}), GaitTrot, TempoWorking),
		sp(idOf(t, "track", "C–A", State{"C", DirR}), GaitTrot, TempoWorking),
	})
	req := Requirement{Class: "L*", Movements: []ReqMovement{{Key: "volte_10_trab", Count: 1, Rein: "right"}}}
	p := eval(t, KindFreestyle, "L*", req, specs)
	if f := failing(p); f["requirements"] != "" || len(p.Requirements) != 1 || !p.Requirements[0].Met || p.Requirements[0].Shown != 1 {
		t.Errorf("met: %v %+v", f, p.Requirements)
	}
	// Twice asked, once shown; or on the other rein.
	for _, r := range []ReqMovement{{Key: "volte_10_trab", Count: 2, Rein: "right"}, {Key: "volte_10_trab", Count: 1, Rein: "left"}, {Key: "volte_10_trab", Count: 1, Rein: "both"}} {
		q := eval(t, KindFreestyle, "L*", Requirement{Class: "L*", Movements: []ReqMovement{r}}, specs)
		if f := failing(q); !strings.Contains(f["requirements"], "Volte 10 m im Trab") {
			t.Errorf("%+v: %v", r, f)
		}
	}
}

func TestCheckPracticeRules(t *testing.T) {
	// A short test in trot only: the walk and the canter are missing on both reins, the reins are out of balance.
	specs := route(DirR, trotAll(idOf(t, "track", "C–A", State{"C", DirR})))
	f := failing(eval(t, KindPractice, "A", Requirement{}, specs))
	if f["gaits_both_reins"] == "" || f["balance"] == "" {
		t.Errorf("trot on one rein: %v", f)
	}
	// A generated test passes every rule; removing a line breaks the chain.
	p, err := GeneratePractice(PracticeParams{Class: "A", Arena: Size20x40, Seed: 4})
	if err != nil {
		t.Fatal(err)
	}
	rulesOK(t, "generated", p)
	edited := p.Specs()
	edited = slices.Delete(edited, 4, 5)
	q := eval(t, KindPractice, "A", Requirement{}, edited)
	if len(RuleErrors(q.Checks)) == 0 {
		t.Error("removing a line is not reported")
	}
	// Reordering keeps lines but the chain breaks as well.
	re := p.Specs()
	re[3], re[4] = re[4], re[3]
	if len(RuleErrors(eval(t, KindPractice, "A", Requirement{}, re).Checks)) == 0 && p.Lines[3].End != p.Lines[4].End {
		t.Error("reordering is not reported")
	}
	// Unknown figure.
	if _, err := Evaluate(KindPractice, "A", Size20x40, 1, []Spec{{Option: "x", Gait: GaitTrot}}, Requirement{}); err == nil {
		t.Error("unknown figure accepted")
	}
}

func TestMusicParts(t *testing.T) {
	p, err := GeneratePractice(PracticeParams{Class: "A", Arena: Size20x40, Seed: 8})
	if err != nil {
		t.Fatal(err)
	}
	for i, m := range p.Music {
		if i > 0 && p.Music[i-1].Gait == m.Gait {
			t.Errorf("consecutive parts %d and %d share the gait %s", i, i+1, m.Gait)
		}
		if m.Cadence != Cadence[m.Gait] {
			t.Errorf("part %d cadence %v", i, m.Cadence)
		}
		for n := m.From; n <= m.To; n++ {
			if p.Lines[n-1].Gait != m.Gait {
				t.Errorf("line %d is not %s", n, m.Gait)
			}
		}
	}
	if p.Music[len(p.Music)-1].To != len(p.Lines) || p.Music[0].From != 1 {
		t.Error("music parts do not cover the lines")
	}
}
