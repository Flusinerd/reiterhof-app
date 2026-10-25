package dressage

import (
	"math"
	"math/rand"
	"slices"
	"sort"
)

// Hints are what the generator knows about the horse: which lessons it has mastered (a session
// rated 3 for a catalog exercise), the next lessons of its level progression and the exercises
// the chosen focus prefers. HasSessions false means the horse has no sessions: every movement of
// the class is allowed.
type Hints struct {
	HasSessions    bool
	Mastered       map[string]bool
	Next           map[string]bool
	FocusExercises map[string]bool
}

// phase is a stretch of a test in one gait: the generator fills it with figures until its time is
// used, the right rein is reached and the change of rein (if the phase has one) is done.
type phase struct {
	gait   string
	endDir string // "r1", "r2" or "" for either: the rein the phase ends on
	// startDir is the rein it starts on (set by the engine from the previous phase).
	change       bool // exactly one change of rein inside
	canterChange bool // the change is a simple change of lead (counter canter included)
	freeFirst    bool // the first line is a free walk across the arena
	final        bool // ends at A, where the salute at the end starts
	share        float64
}

var phasesA = []phase{
	{gait: GaitTrot, endDir: "r1", share: .16},
	{gait: GaitTrot, endDir: "r2", change: true, share: .16},
	{gait: GaitWalk, endDir: "r2", share: .06},
	{gait: GaitWalk, endDir: "r1", change: true, freeFirst: true, share: .09},
	{gait: GaitTrot, endDir: "r1", share: .05},
	{gait: GaitCanter, endDir: "r1", share: .15},
	{gait: GaitTrot, endDir: "r2", change: true, share: .07},
	{gait: GaitCanter, endDir: "r2", share: .15},
	{gait: GaitTrot, endDir: "r2", final: true, share: .11},
}

// phasesB adds a change of lead in canter (simple change of lead, counter canter) after the
// second canter phase; used from class L.
var phasesB = []phase{
	{gait: GaitTrot, endDir: "r1", share: .16},
	{gait: GaitTrot, endDir: "r2", change: true, share: .14},
	{gait: GaitWalk, endDir: "r2", share: .06},
	{gait: GaitWalk, endDir: "r1", change: true, freeFirst: true, share: .08},
	{gait: GaitTrot, endDir: "r1", share: .05},
	{gait: GaitCanter, endDir: "r1", share: .12},
	{gait: GaitTrot, endDir: "r2", change: true, share: .06},
	{gait: GaitCanter, endDir: "r2", share: .12},
	{gait: GaitCanter, endDir: "r1", change: true, canterChange: true, share: .04},
	{gait: GaitCanter, endDir: "r1", share: .04},
	{gait: GaitTrot, endDir: "r1", final: true, share: .13},
}

// item is one required showing of a movement of a freestyle.
type item struct {
	key   string
	rein  string // "left", "right" or "" for either
	gait  string
	tempo string
	phase int
	done  bool
}

type engine struct {
	lib   *Library
	class string
	rng   *rand.Rand

	allowed  map[string]bool // movement keys allowed by class and mastery
	required map[string]bool // movement keys the user requires (allowed whatever the class)
	focus    map[string]bool // movement keys the focus prefers
	items    []*item
	// noLateralWalk keeps lateral work and "Viereck verkleinern" out of the walk (required walk).
	noLateralWalk bool

	phases []phase
	r1, r2 string
	pi     int // index of the current phase

	specs   []Spec
	prev    *Spec
	state   State
	elapsed float64
	nodes   int

	usedKind map[string]int
	usedOpt  map[string]int
	usedMove map[string]int
	kinds    []string // kinds of the lines so far (for variety)
}

func (e *engine) allow(key string) bool {
	if e.required[key] {
		return true
	}
	return e.allowed[key]
}

// secondsOf is the time of a line after the previous one.
func (e *engine) secondsOf(o *Option, gait, tempo string) float64 {
	s := o.Length/Speed(gait, tempo) + o.Fixed
	if e.prev != nil && (e.prev.Gait != gait || e.prev.Tempo != tempo) {
		s += TransitionSeconds
	}
	return s
}

type cand struct {
	o     *Option
	tempo string
	secs  float64
	score float64
	item  *item
}

func (e *engine) dirOf(tag string) string {
	switch tag {
	case "r1":
		return e.r1
	case "r2":
		return e.r2
	}
	return ""
}

func isSpecial(kind string) bool {
	switch kind {
	case kindEinreiten, kindWeiter, kindAbwenden, kindAusreiten:
		return true
	}
	return false
}

// candidates lists the figures that can follow the current state in the phase.
func (e *engine) candidates(ph *phase, changes int, first bool, target, phaseStart float64) []cand {
	var out []cand
	defTempo := TempoWorking
	if ph.gait == GaitWalk {
		defTempo = TempoMedium
	}
	for _, o := range e.lib.From(e.state) {
		if isSpecial(o.Kind) || !o.allows(ph.gait) {
			continue
		}
		canterChangeKind := o.Kind == "diagonale_einfacher_wechsel" || o.Kind == "aussengalopp_kurzseite"
		switch {
		case canterChangeKind && !(ph.canterChange && e.mayChange(ph, changes)):
			continue
		case o.Changes && !canterChangeKind && ph.canterChange:
			continue
		case o.Changes && !canterChangeKind && !(ph.change && e.mayChange(ph, changes)):
			continue
		}
		if e.noLateralWalk && ph.gait == GaitWalk && o.Lateral {
			continue
		}
		if first && ph.freeFirst && !(o.Kind == "diagonal" || o.Kind == "halbe_bahn") {
			continue
		}
		tempi := []string{defTempo}
		if first && ph.freeFirst {
			tempi = []string{TempoFree}
		} else {
			for _, it := range e.items {
				if !it.done && it.phase == e.pi && it.gait == ph.gait && it.tempo != defTempo && it.tempo != "" && !slices.Contains(tempi, it.tempo) {
					tempi = append(tempi, it.tempo)
				}
			}
			if e.lib != nil && o.Straight && ph.gait == GaitTrot && e.allow("mitteltrab") && !slices.Contains(tempi, TempoMedium) && len(e.items) == 0 {
				tempi = append(tempi, TempoMedium)
			}
			if ph.gait == GaitCanter && o.Straight && e.allow("mittelgalopp") && !slices.Contains(tempi, TempoMedium) && len(e.items) == 0 {
				tempi = append(tempi, TempoMedium)
			}
		}
		for _, tempo := range tempi {
			if tempo != defTempo && !o.Straight && tempo != TempoFree && tempo != TempoCollected {
				continue
			}
			keys := lineMovements(o, ph.gait, tempo)
			ok := true
			for _, k := range keys {
				if !e.allow(k) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			c := cand{o: o, tempo: tempo, secs: e.secondsOf(o, ph.gait, tempo)}
			c.score = e.score(o, keys, ph, c.secs, target, phaseStart)
			if e.itemsLeft() {
				if e.hostsPending(o.End, ph) {
					c.score += 8 // leads to a figure that shows a required movement
				}
				c.score -= c.secs / 6 // little time for detours
			}
			for _, it := range e.items {
				if it.done || it.phase != e.pi || !slices.Contains(keys, it.key) {
					continue
				}
				if it.rein != "" && o.Rein() != it.rein {
					continue
				}
				c.item = it
				c.score += 20
				break
			}
			out = append(out, c)
		}
	}
	return out
}

// family groups the kinds that look alike in a test (three voltes in a row are no variety).
func family(kind string) string {
	switch kind {
	case "volte8", "volte10":
		return "volte"
	case "schulterherein", "travers", "renvers", "schenkelweichen", "traversale":
		return "lateral"
	case "einfache_schlange", "doppelte_schlange", "schlangenlinie":
		return "schlange"
	}
	return kind
}

func (e *engine) score(o *Option, keys []string, ph *phase, secs, target, phaseStart float64) float64 {
	s := e.rng.Float64() * 2
	fam := family(o.Kind)
	if n := len(e.kinds); n > 0 && family(e.kinds[n-1]) == fam {
		s -= 3
	}
	if n := len(e.kinds); n > 1 && family(e.kinds[n-2]) == fam {
		s -= 1
	}
	if fam != "track" {
		s += 1.5
	}
	s -= 0.6*float64(e.usedKind[o.Kind]) + 4*float64(e.usedOpt[o.ID])
	for _, k := range keys {
		if e.focus[k] {
			s += 2
		}
		if e.usedMove[k] == 0 {
			s += 1
		}
	}
	// Prefer a figure that fits the time left in the phase.
	remaining := target - (e.elapsed - phaseStart)
	if over := secs - math.Max(remaining, 0); over > 0 {
		s -= over / 3
	}
	return s
}

func (e *engine) push(c cand, gait string) (undo func()) {
	prevPrev, prevState, prevElapsed := e.prev, e.state, e.elapsed
	sp := Spec{Option: c.o.ID, Gait: gait, Tempo: c.tempo}
	e.specs = append(e.specs, sp)
	e.prev = &sp
	e.state = c.o.End
	e.elapsed += c.secs
	e.usedKind[c.o.Kind]++
	e.usedOpt[c.o.ID]++
	e.kinds = append(e.kinds, c.o.Kind)
	keys := lineMovements(c.o, gait, c.tempo)
	for _, k := range keys {
		e.usedMove[k]++
	}
	if c.item != nil {
		c.item.done = true
	}
	return func() {
		e.specs = e.specs[:len(e.specs)-1]
		e.prev, e.state, e.elapsed = prevPrev, prevState, prevElapsed
		e.usedKind[c.o.Kind]--
		e.usedOpt[c.o.ID]--
		e.kinds = e.kinds[:len(e.kinds)-1]
		for _, k := range keys {
			e.usedMove[k]--
		}
		if c.item != nil {
			c.item.done = false
		}
	}
}

// mayChange: a phase with a change of rein does it once; more changes only for required movements
// that need one, or to end on the rein the phase asks for.
func (e *engine) mayChange(ph *phase, changes int) bool {
	return changes == 0 || e.itemsLeft() || e.state.Dir != e.dirOf(ph.endDir)
}

// hostsPending reports whether a required movement of the phase can be shown by a figure that
// starts in the state.
func (e *engine) hostsPending(st State, ph *phase) bool {
	for _, o := range e.lib.From(st) {
		if isSpecial(o.Kind) || !o.allows(ph.gait) {
			continue
		}
		for _, it := range e.items {
			if it.done || it.phase != e.pi {
				continue
			}
			tempo := it.tempo
			if tempo == "" || it.gait != ph.gait {
				tempo = TempoWorking
				if ph.gait == GaitWalk {
					tempo = TempoMedium
				}
			}
			if slices.Contains(lineMovements(o, ph.gait, tempo), it.key) && (it.rein == "" || o.Rein() == it.rein) {
				return true
			}
		}
	}
	return false
}

func (e *engine) itemsLeft() bool {
	for _, it := range e.items {
		if !it.done && it.phase == e.pi {
			return true
		}
	}
	return false
}

// phaseDone reports whether the phase may end in the current state.
func (e *engine) phaseDone(ph *phase, changes, after int, target, phaseStart float64) bool {
	if e.elapsed-phaseStart < target || e.itemsLeft() {
		return false
	}
	if ph.change && (changes == 0 || after < 1) {
		return false // the rein after the change has to be ridden, too
	}
	if d := e.dirOf(ph.endDir); d != "" && e.state.Dir != d {
		return false
	}
	if e.state.Dir != DirR && e.state.Dir != DirL {
		return false
	}
	if ph.final {
		return e.state.Node == "A"
	}
	// The next phase needs a way to start from here.
	if e.pi+1 < len(e.phases) {
		next := e.phases[e.pi+1]
		if next.freeFirst {
			for _, o := range e.lib.From(e.state) {
				if (o.Kind == "diagonal" || o.Kind == "halbe_bahn") && o.allows(GaitWalk) {
					return true
				}
			}
			return false
		}
	}
	return true
}

const (
	maxPhaseLines = 14
	maxNodes      = 12000
	beam          = 10
)

// run fills the phase by depth-first search with backtracking over the figure options; the
// candidates are tried in order of their score (variety, focus, required movements, time).
func (e *engine) run(ph *phase, target float64, changes, after int, phaseStart float64, depth int) bool {
	if e.phaseDone(ph, changes, after, target, phaseStart) {
		return true
	}
	if depth >= maxPhaseLines || e.nodes > maxNodes {
		return false
	}
	e.nodes++
	cands := e.candidates(ph, changes, depth == 0, target, phaseStart)
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	if len(cands) > beam {
		cands = cands[:beam]
	}
	for _, c := range cands {
		undo := e.push(c, ph.gait)
		ch, af := changes, after
		switch {
		case c.o.Changes:
			ch, af = ch+1, 0
		case after >= 0:
			af = after + 1
		}
		if e.run(ph, target, ch, af, phaseStart, depth+1) {
			return true
		}
		undo()
	}
	return false
}

// movementOK is the filter of the practice generator: the class and, for horses with sessions,
// the mastery of the lessons.
func allowedMovements(class string, h Hints, rng *rand.Rand) map[string]bool {
	out := map[string]bool{}
	var nextCands []string
	for _, m := range Movements {
		if !m.Allowed(class) {
			continue
		}
		if !h.HasSessions || m.Basis || len(m.ExerciseIDs) == 0 {
			out[m.Key] = true
			continue
		}
		mastered, next := false, false
		for _, id := range m.ExerciseIDs {
			mastered = mastered || h.Mastered[id]
			next = next || h.Next[id]
		}
		switch {
		case mastered:
			out[m.Key] = true
		case next:
			nextCands = append(nextCands, m.Key)
		}
	}
	rng.Shuffle(len(nextCands), func(i, j int) { nextCands[i], nextCands[j] = nextCands[j], nextCands[i] })
	for i, k := range nextCands {
		if i >= 2 {
			break
		}
		out[k] = true
	}
	return out
}

// focusMovements are the movements that train an exercise the focus prefers.
func focusMovements(h Hints) map[string]bool {
	out := map[string]bool{}
	for _, m := range Movements {
		for _, id := range m.ExerciseIDs {
			if h.FocusExercises[id] {
				out[m.Key] = true
			}
		}
	}
	return out
}

// newEngine prepares an engine; the opening is chosen from the seed.
func newEngine(lib *Library, class string, rng *rand.Rand) *engine {
	return &engine{lib: lib, class: class, rng: rng, required: map[string]bool{}, focus: map[string]bool{},
		usedKind: map[string]int{}, usedOpt: map[string]int{}, usedMove: map[string]int{}}
}

// openingAndClosing builds the fixed lines before the first and after the last phase: entry at A,
// salute at X or G, to C and onto the track; at the end from A up the centre line, salute, leave.
func (e *engine) opening() (specs []Spec, node string, dir string) {
	node = []string{"X", "G"}[e.rng.Intn(2)]
	dir = []string{DirR, DirL}[e.rng.Intn(2)]
	return []Spec{
		{Option: kindEinreiten + ":" + node, Gait: GaitTrot, Tempo: TempoWorking},
		{Option: kindWeiter + ":" + node + ":" + dir, Gait: GaitTrot, Tempo: TempoWorking},
	}, node, dir
}

func (e *engine) closing(dir string) []Spec {
	node := []string{"X", "G"}[e.rng.Intn(2)]
	return []Spec{
		{Option: kindAbwenden + ":" + node + ":" + dir, Gait: GaitTrot, Tempo: TempoWorking},
		{Option: kindAusreiten + ":" + node, Gait: GaitWalk, Tempo: TempoFree},
	}
}

// fixedSeconds is the time of the opening and closing lines.
func (e *engine) fixedSeconds(open, closeSpecs []Spec) float64 {
	lines, err := BuildLines(e.lib, append(slices.Clone(open), closeSpecs...))
	if err != nil {
		return 0
	}
	t := 0.0
	for _, l := range lines {
		t += l.Seconds
	}
	return t
}

func clampf(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
