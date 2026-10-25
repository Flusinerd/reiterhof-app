package dressage

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"slices"
)

// Lengths of a practice test.
const (
	LengthShort  = "short"
	LengthNormal = "normal"
)

// PracticeWindow is the app's own target duration of a practice test in seconds: short 3–4 minutes,
// normal 5–6 minutes (E, A) or 6–7 minutes (L, L**). These are our targets, not FN figures.
func PracticeWindow(class, length string) (lo, hi float64) {
	switch {
	case length == LengthShort:
		return 210, 300
	case ClassRank(class) >= ClassRank("L"):
		return 360, 420
	}
	return 300, 360
}

// PracticeParams are the inputs of GeneratePractice.
type PracticeParams struct {
	Class  string
	Arena  string
	Length string // LengthShort or LengthNormal
	Seed   int64
	Hints  Hints
}

// ErrNoPlan says the generator found no plan for the inputs.
var ErrNoPlan = errors.New("dressage: no plan found")

const attempts = 250

// GeneratePractice builds a practice test: own structure (entry, salute, work on both reins in
// trot, walk and canter, salute at the end), figures from the library chained so every line starts
// where the last ended. The same inputs and seed always give the same plan.
func GeneratePractice(p PracticeParams) (Plan, error) {
	if !slices.Contains(PracticeClasses, p.Class) {
		return Plan{}, fmt.Errorf("dressage: unknown class %q", p.Class)
	}
	lib, ok := LibraryFor(p.Arena)
	if !ok {
		return Plan{}, fmt.Errorf("dressage: unknown arena %q", p.Arena)
	}
	if p.Length != LengthShort {
		p.Length = LengthNormal
	}
	lo, hi := PracticeWindow(p.Class, p.Length)
	g := &generator{
		kind: KindPractice, class: p.Class, lib: lib, seed: p.Seed, hints: p.Hints,
		lo: lo, hi: hi, mid: (lo + hi) / 2, short: p.Length == LengthShort,
	}
	return g.run()
}

type generator struct {
	kind   string
	class  string
	lib    *Library
	seed   int64
	hints  Hints
	lo, hi float64
	mid    float64
	req    *Requirement
	short  bool
	failed map[int]int // attempts that failed per phase (diagnosis)
}

func (g *generator) run() (Plan, error) {
	scale := 0.8
	g.failed = map[int]int{}
	var best *Plan
	bestPenalty := 1 << 30
	for i := 0; i < attempts; i++ {
		plan, total := g.attempt(i, scale)
		if plan == nil {
			scale *= 0.9
			continue
		}
		pen := 0
		for _, c := range plan.Checks {
			if c.Rule && !c.OK {
				pen += 100
			}
		}
		if total < g.lo-2 {
			pen += int(g.lo - total)
		}
		if total > g.hi+2 {
			pen += int(total - g.hi)
		}
		if pen < bestPenalty {
			bestPenalty, best = pen, plan
		}
		if pen == 0 {
			break
		}
		// Steer the time budget of the phases towards the middle of the window.
		scale = clampf(scale*g.mid/total, 0.2, 2)
	}
	if best == nil {
		return Plan{}, fmt.Errorf("%w (phases that failed: %v)", ErrNoPlan, g.failed)
	}
	return *best, nil
}

// attempt builds one candidate plan; nil when a phase could not be filled.
func (g *generator) attempt(i int, scale float64) (*Plan, float64) {
	rng := rand.New(rand.NewSource(g.seed*7919 + int64(i)*1000003 + 17))
	e := newEngine(g.lib, g.class, rng)
	if g.kind == KindPractice {
		e.allowed = allowedMovements(g.class, g.hints, rng)
		e.focus = focusMovements(g.hints)
	} else {
		e.allowed = allowedMovements(g.class, Hints{}, rng)
		for _, m := range g.req.Movements {
			e.required[m.Key] = true
		}
		e.noLateralWalk = g.req.walkMeters() > 0
	}
	open, _, dir := e.opening()
	e.r1 = dir
	e.r2 = DirL
	if dir == DirL {
		e.r2 = DirR
	}
	var phases []phase
	switch {
	case g.kind == KindPractice && !g.short && ClassRank(g.class) >= ClassRank("L") && rng.Intn(2) == 0:
		phases = slices.Clone(phasesB)
	case g.kind == KindFreestyle && g.req.needsCanterChange():
		phases = slices.Clone(phasesB)
	default:
		phases = slices.Clone(phasesA)
	}
	if g.kind == KindFreestyle {
		for i := range phases {
			phases[i].freeFirst = false
		}
		g.assignItems(e, phases)
	}
	e.phases = phases

	fixed := e.fixedSeconds(open, e.closing(DirR))
	budget := (g.mid - fixed) * scale
	if budget < 10 {
		budget = 10
	}
	openLines, err := BuildLines(g.lib, open)
	if err != nil {
		return nil, 0
	}
	e.specs = slices.Clone(open)
	last := open[len(open)-1]
	e.prev = &last
	e.state = State{"C", dir}
	for _, l := range openLines {
		e.elapsed += l.Seconds
	}
	shares := phaseShares(phases, g.kind == KindPractice, g.kind == KindFreestyle && g.req.walkMeters() > 0, g.req.walkSeconds(), budget)
	for pi := range phases {
		e.pi = pi
		ph := phases[pi]
		if !e.run(&ph, shares[pi], 0, -1, e.elapsed, 0) {
			g.failed[pi]++
			return nil, 0
		}
	}
	e.specs = append(e.specs, e.closing(e.state.Dir)...)
	lines, err := BuildLines(g.lib, e.specs)
	if err != nil {
		return nil, 0
	}
	plan := &Plan{Kind: g.kind, Class: g.class, Arena: g.lib.Arena.Size, Seed: g.seed, Lines: lines}
	plan.Totals(g.lib)
	if g.kind == KindPractice {
		plan.Checks = CheckPractice(g.lib, g.class, plan)
		plan.Requirements = []RequirementStatus{}
	} else {
		plan.Checks, plan.Requirements = CheckFreestyle(g.lib, g.class, *g.req, plan)
	}
	return plan, plan.Seconds
}

// phaseShares gives each phase its share of the time budget; the two walk phases get at least the
// time a required walk distance needs.
func phaseShares(phases []phase, floors, walkRequired bool, walkSec, budget float64) []float64 {
	out := make([]float64, len(phases))
	walk := 0.0
	for i, p := range phases {
		out[i] = p.share * budget
		// A phase needs a figure or two: at least 10 s in walk and 12 s in canter (both reins need some), 3 s in trot.
		if floors {
			out[i] = math.Max(out[i], map[string]float64{GaitWalk: 10, GaitCanter: 12, GaitTrot: 3}[p.gait])
		}
		if p.gait == GaitWalk {
			walk += out[i]
		}
	}
	if walkRequired && walk < walkSec && walk > 0 {
		for i, p := range phases {
			if p.gait == GaitWalk {
				out[i] *= walkSec / walk
			}
		}
	}
	return out
}

// assignItems spreads the required showings of a freestyle over the phases that fit their gait
// and rein; ones that need a change of rein go to the phases with one.
func (g *generator) assignItems(e *engine, phases []phase) {
	load := make([]int, len(phases))
	for _, rm := range g.req.Movements {
		m, ok := MovementByKey(rm.Key)
		if !ok {
			continue
		}
		count := max(1, rm.Count)
		reins := []string{""}
		switch rm.Rein {
		case "left", "right":
			reins = []string{rm.Rein}
		case "both":
			reins = []string{"left", "right"}
		}
		for _, rein := range reins {
			for n := 0; n < count; n++ {
				best := -1
				for pi, ph := range phases {
					if !phaseFits(e, ph, m, rein) {
						continue
					}
					if best < 0 || load[pi] < load[best] {
						best = pi
					}
				}
				if best < 0 {
					best = 0
				}
				load[best]++
				e.items = append(e.items, &item{key: rm.Key, rein: rein, gait: itemGait(m), tempo: itemTempo(m), phase: best})
			}
		}
	}
}

func itemGait(m Movement) string {
	if m.Gait == GaitHalt {
		return ""
	}
	return m.Gait
}

func itemTempo(m Movement) string {
	if m.Gait == "" || m.Gait == GaitHalt {
		return ""
	}
	return m.Tempo
}

// changerKinds are figures that change the rein: movements shown only on them need a phase with a change.
var changerKinds = []string{"diagonal", "halbe_bahn", "handwechsel_zirkel", "kurzkehrt", "diagonale_einfacher_wechsel", "aussengalopp_kurzseite"}

func phaseFits(e *engine, ph phase, m Movement, rein string) bool {
	needsChange, canterChange := len(m.Figures) > 0, false
	for _, f := range m.Figures {
		if !slices.Contains(changerKinds, f) {
			needsChange = false
		}
		if f == "diagonale_einfacher_wechsel" || f == "aussengalopp_kurzseite" {
			canterChange = true
		}
	}
	if canterChange {
		return ph.canterChange
	}
	if needsChange && !ph.change {
		return false
	}
	switch m.Gait {
	case GaitWalk, GaitTrot, GaitCanter:
		if ph.gait != m.Gait {
			return false
		}
	case GaitHalt:
		if ph.gait == GaitCanter {
			return false
		}
	}
	if m.Gait == "" && ph.gait == GaitCanter && !slices.Contains(m.Figures, "zirkel") && len(m.Figures) > 0 && m.Key != "zirkel" {
		// figure movements without a gait (centre line, changes of rein) are not shown in canter
		return false
	}
	if rein != "" {
		return phaseCoversRein(e, ph, rein)
	}
	return true
}

// phaseCoversRein: a phase is ridden on its end rein, a phase with a change also on the rein before.
func phaseCoversRein(e *engine, ph phase, rein string) bool {
	end := e.dirOf(ph.endDir)
	if reinWord(end) == rein {
		return true
	}
	return ph.change && reinWord(oppositeDir(end)) == rein
}

func oppositeDir(d string) string {
	if d == DirR {
		return DirL
	}
	return DirR
}
