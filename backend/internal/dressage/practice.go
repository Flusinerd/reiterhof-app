package dressage

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sort"
	"strings"
)

// Longest practice test in lines, entry and exit included.
const (
	MaxLinesShort  = 18
	MaxLinesNormal = 26
)

// MaxPracticeLines is the line limit of a practice test of a length.
func MaxPracticeLines(length string) int {
	if length == LengthShort {
		return MaxLinesShort
	}
	return MaxLinesNormal
}

// Limits the generator keeps for every plan (the tests assert the same numbers).
const (
	maxPerMovement  = 2 // a movement shows at most twice per gait
	maxLateralRein  = 2 // lateral lines per rein
	maxSerpentines  = 1 // serpentines per test
	practiceReinMin = 0.40
	// minTrackSeconds: a track line is at least this long (moving time), shorter ones are no lines of a test.
	minTrackSeconds = 6.0
)

// plainGaitKeys name only the gait; they come with every line and are not counted as movements.
var plainGaitKeys = []string{"arbeitstrab", "mittelschritt", "freier_schritt", "arbeitsgalopp"}

// structuralKeys are the lines that connect the parts of a test (changes of rein, centre line, salute).
var structuralKeys = []string{"handwechsel_diagonale", "handwechsel_halbe_bahn", "handwechsel_zirkel", "mittellinie", "halten_gruss"}

// leadKeys belong to the canter block with a change of lead.
var leadKeys = []string{"einfacher_galoppwechsel", "aussengalopp"}

// IsCountedMovement reports whether a movement key counts for the "at most twice" rule of a practice
// test: everything except the plain gaits and the salute.
func IsCountedMovement(key string) bool {
	return key != "halten_gruss" && !slices.Contains(plainGaitKeys, key)
}

// coreMovements are the movements that make a class: a test of the class shows them first. A short test
// shows the core of each gait plus at most two others, a normal test more. The lists are the app's own
// choice from the movements of the class (docs/domains/dressage-tests.md, "Practice generator").
var coreMovements = map[string]map[string][]string{
	"E": {
		GaitTrot:   {"zirkel", "schenkelweichen"},
		GaitWalk:   {"zirkel"},
		GaitCanter: {"zirkel"},
	},
	"A": {
		GaitTrot:   {"zirkel", "volte_10_trab", "mitteltrab", "schenkelweichen"},
		GaitWalk:   {"zirkel", "viereck_verkleinern_schritt"},
		GaitCanter: {"zirkel"},
	},
	"L": {
		GaitTrot:   {"zirkel", "volte_8_trab", "mitteltrab", "schlangenlinie_ganze_bahn_trab"},
		GaitWalk:   {"zirkel", "viereck_verkleinern_schritt"},
		GaitCanter: {"zirkel", "volte_8_galopp", "einfacher_galoppwechsel"},
	},
	"L**": {
		GaitTrot:   {"volte_8_trab", "mitteltrab", "schulterherein", "travers"},
		GaitWalk:   {"zirkel", "viereck_verkleinern_schritt"},
		GaitCanter: {"zirkel", "volte_8_galopp", "einfacher_galoppwechsel", "aussengalopp"},
	},
}

// CoreMovements returns the core movement keys of a class in a gait ("walk", "trot", "canter").
func CoreMovements(class, gait string) []string {
	return slices.Clone(coreMovements[class][gait])
}

// block is one part of a practice test: a gait, the rein it ends on and how many figures it shows.
type block struct {
	name string
	gait string
	rein string // "r1" or "r2": the rein the block ends on, "" for either
	// change: "rein" (exactly one change of rein), "lead" (exactly one simple change of lead), "".
	change string
	// free: the block is one free-walk line across the arena.
	free bool
	// trackOnly: the block is track lines only (the trot before the canter).
	trackOnly bool
	// final: the block ends at A, where the salute at the end starts.
	final              bool
	minFig, maxFig     int
	minLines, maxLines int
	// after is the number of lines needed after the change of rein (the work on the new rein).
	after int
}

func practiceBlocks(short, lead bool) []block {
	canter := func(name, rein string) block {
		if short {
			return block{name: name, gait: GaitCanter, rein: rein, minFig: 1, maxFig: 1, minLines: 1, maxLines: 2}
		}
		return block{name: name, gait: GaitCanter, rein: rein, minFig: 1, maxFig: 2, minLines: 2, maxLines: 3}
	}
	c1, c2 := canter("canter 1", "r1"), canter("canter 2", "r2")
	if lead {
		c2.change, c2.rein, c2.minFig, c2.maxFig, c2.minLines, c2.maxLines = "lead", "r1", 1, 1, 2, 3
	}
	var bs []block
	if short {
		bs = []block{
			{name: "trot 1", gait: GaitTrot, rein: "r1", minFig: 1, maxFig: 1, minLines: 1, maxLines: 2},
			{name: "trot 2", gait: GaitTrot, rein: "r2", change: "rein", minFig: 1, maxFig: 1, minLines: 2, maxLines: 3, after: 1},
			{name: "walk 1", gait: GaitWalk, rein: "r2", minFig: 0, maxFig: 1, minLines: 1, maxLines: 2},
			{name: "free walk", gait: GaitWalk, rein: "r1", change: "rein", free: true, minLines: 1, maxLines: 1},
			{name: "walk 2", gait: GaitWalk, rein: "r1", minFig: 0, maxFig: 1, minLines: 1, maxLines: 2},
			{name: "trot before canter", gait: GaitTrot, rein: "r1", trackOnly: true, minLines: 1, maxLines: 1},
			c1,
			{name: "trot change", gait: GaitTrot, rein: "r2", change: "rein", minLines: 1, maxLines: 2},
			c2,
			{name: "trot end", gait: GaitTrot, final: true, minFig: 0, maxFig: 1, minLines: 1, maxLines: 1},
		}
	} else {
		bs = []block{
			{name: "trot 1", gait: GaitTrot, rein: "r1", minFig: 2, maxFig: 3, minLines: 2, maxLines: 5},
			{name: "trot 2", gait: GaitTrot, rein: "r2", change: "rein", minFig: 2, maxFig: 2, minLines: 3, maxLines: 5, after: 2},
			{name: "walk 1", gait: GaitWalk, rein: "r2", minFig: 0, maxFig: 1, minLines: 1, maxLines: 2},
			{name: "free walk", gait: GaitWalk, rein: "r1", change: "rein", free: true, minLines: 1, maxLines: 1},
			{name: "walk 2", gait: GaitWalk, rein: "r1", minFig: 0, maxFig: 1, minLines: 1, maxLines: 2},
			{name: "trot before canter", gait: GaitTrot, rein: "r1", trackOnly: true, minLines: 1, maxLines: 1},
			c1,
			{name: "trot change", gait: GaitTrot, rein: "r2", change: "rein", minLines: 1, maxLines: 2},
			c2,
			{name: "trot end", gait: GaitTrot, final: true, minFig: 0, maxFig: 1, minLines: 1, maxLines: 2},
		}
	}
	return bs
}

// bstate is the progress inside the current block.
type bstate struct {
	lines, figs int
	needFig     int
	changed     bool
	after       int
}

// pgen builds one practice test: the blocks in order, each filled by a depth-first search over the
// figures of the library; a complete plan has to pass every rule, the window and the limits.
type pgen struct {
	lib    *Library
	class  string
	rng    *rand.Rand
	short  bool
	lo, hi float64
	mid    float64
	cap    int

	allowed map[string]bool
	focus   map[string]bool
	hints   Hints
	palette map[string]map[string]bool // gait -> figure movements this test may show
	lead    bool

	r1, r2   string
	needs    []int
	blocks   []block
	expLines float64
	closeSec float64
	closeAt  string // X or G at the end

	specs    []Spec
	prev     *Spec
	state    State
	elapsed  float64
	nodes    int
	maxNodes int
	counts   map[string]int // gait|movement
	serp     int
	lateral  map[string]int
	kinds    []string
	usedOpt  map[string]int
	usedMove map[string]int
	result   *Plan
}

func newPracticeGen(p PracticeParams, lib *Library, attempt int) *pgen {
	lo, hi := PracticeWindow(p.Class, p.Length)
	g := &pgen{
		lib: lib, class: p.Class, short: p.Length == LengthShort, lo: lo, hi: hi, mid: (lo + hi) / 2,
		cap: MaxPracticeLines(p.Length), hints: p.Hints,
		maxNodes: 4000,
		counts:   map[string]int{}, lateral: map[string]int{}, usedOpt: map[string]int{}, usedMove: map[string]int{},
	}
	g.rng = rand.New(rand.NewSource(p.Seed*7919 + int64(attempt)*1000003 + 17))
	return g
}

// GeneratePractice builds a practice test with the structure of a real test (own wording): entry and
// salute, trot on both reins with a change of rein, walk with a free walk across the arena, one canter
// block on each rein, trot, final salute. The figures come from the library and are chained so that every
// line starts where the last ended. The same inputs and seed always give the same plan.
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
	var best *Plan
	for i := 0; i < practiceAttempts; i++ {
		g := newPracticeGen(p, lib, i)
		if plan := g.build(); plan != nil {
			best = plan
			break
		}
	}
	if best == nil {
		return Plan{}, fmt.Errorf("%w (practice %s %s %s)", ErrNoPlan, p.Class, p.Arena, p.Length)
	}
	best.Seed = p.Seed
	return *best, nil
}

const practiceAttempts = 120

func (g *pgen) dirOf(tag string) string {
	switch tag {
	case "r1":
		return g.r1
	case "r2":
		return g.r2
	}
	return ""
}

func (g *pgen) build() *Plan {
	rng := g.rng
	g.allowed = allowedMovements(g.class, g.hints, rng)
	g.focus = focusMovements(g.hints)
	open, _, dir := (&engine{rng: rng}).opening()
	g.r1 = dir
	g.r2 = DirL
	if dir == DirL {
		g.r2 = DirR
	}
	g.closeAt = []string{"X", "G"}[rng.Intn(2)]
	g.makePalette()
	// A canter block with a simple change of lead from class L, when the class allows it.
	leadOK := ClassRank(g.class) >= ClassRank("L") && (g.allowed["einfacher_galoppwechsel"] || g.allowed["aussengalopp"])
	if leadOK {
		p := 0.55
		if g.focus["einfacher_galoppwechsel"] || g.focus["aussengalopp"] {
			p = 1
		}
		g.lead = rng.Float64() < p
	}
	g.blocks = practiceBlocks(g.short, g.lead)
	for _, b := range g.blocks {
		g.expLines += float64(b.minLines+b.maxLines) / 2
	}
	g.planFigures()
	g.expLines = math.Min(g.expLines, float64(g.cap-4))
	openLines, err := BuildLines(g.lib, open)
	if err != nil {
		return nil
	}
	g.specs = slices.Clone(open)
	last := open[len(open)-1]
	g.prev = &last
	g.state = State{"C", dir}
	for _, l := range openLines {
		g.elapsed += l.Seconds
	}
	cl, err := BuildLines(g.lib, g.closing(DirR))
	if err != nil {
		return nil
	}
	for _, l := range cl {
		g.closeSec += l.Seconds
	}
	if g.solve(0, g.newState(0)) {
		return g.result
	}
	return nil
}

func (g *pgen) closing(dir string) []Spec {
	return []Spec{
		{Option: kindAbwenden + ":" + g.closeAt + ":" + dir, Gait: GaitTrot, Tempo: TempoWorking},
		{Option: kindAusreiten + ":" + g.closeAt, Gait: GaitWalk, Tempo: TempoFree},
	}
}

// makePalette picks the movements this test shows per gait: the core of the class (as far as the
// horse may do it) plus extras; movements of the focus and mastered ones are preferred as extras.
func (g *pgen) makePalette() {
	g.palette = map[string]map[string]bool{}
	for _, gait := range []string{GaitWalk, GaitTrot, GaitCanter} {
		avail := g.figureKeys(gait)
		pal := map[string]bool{}
		for _, k := range coreMovements[g.class][gait] {
			if avail[k] && g.allowed[k] && !slices.Contains(leadKeys, k) {
				pal[k] = true
			}
		}
		var pool, wanted []string
		for k := range avail {
			switch {
			case pal[k] || !g.allowed[k] || slices.Contains(leadKeys, k):
			case g.focus[k]:
				wanted = append(wanted, k)
			default:
				pool = append(pool, k)
			}
		}
		slices.Sort(pool)
		slices.Sort(wanted)
		limit, n := 2, g.rng.Intn(3)
		if !g.short {
			limit, n = 4, 2+g.rng.Intn(3)
		}
		// The focus comes first, then the horse's mastered movements are preferred over the rest.
		for _, k := range wanted {
			if limit > 0 {
				pal[k] = true
				limit--
				n--
			}
		}
		for ; n > 0 && limit > 0 && len(pool) > 0; n, limit = n-1, limit-1 {
			weights := make([]float64, len(pool))
			sum := 0.0
			for i, k := range pool {
				weights[i] = 1
				if g.mastered(k) {
					weights[i] = 2
				}
				sum += weights[i]
			}
			x := g.rng.Float64() * sum
			pick := len(pool) - 1
			for i, w := range weights {
				if x < w {
					pick = i
					break
				}
				x -= w
			}
			pal[pool[pick]] = true
			pool = slices.Delete(pool, pick, pick+1)
		}
		// A horse with few mastered lessons still rides circles (no class and no exercise needed).
		hasFigure := false
		for k := range pal {
			hasFigure = hasFigure || k != "mitteltrab" && k != "mittelgalopp"
		}
		if !hasFigure && avail["zirkel"] && g.allowed["zirkel"] {
			pal["zirkel"] = true
		}
		g.palette[gait] = pal
	}
}

func (g *pgen) mastered(key string) bool {
	m, ok := MovementByKey(key)
	if !ok {
		return false
	}
	for _, id := range m.ExerciseIDs {
		if g.hints.Mastered[id] {
			return true
		}
	}
	return false
}

// figureKeys are the movements (not the structure of the test) the library offers in a gait.
func (g *pgen) figureKeys(gait string) map[string]bool {
	out := map[string]bool{}
	for i := range g.lib.All {
		o := &g.lib.All[i]
		if isSpecial(o.Kind) || !o.allows(gait) || o.Changes && o.Kind != "diagonal" {
			continue
		}
		for _, k := range figureMovementKeys(o.Kind, gait) {
			if isPaletteKey(k) {
				out[k] = true
			}
		}
	}
	switch gait {
	case GaitTrot:
		out["mitteltrab"] = true
	case GaitCanter:
		out["mittelgalopp"] = true
	}
	return out
}

func isPaletteKey(k string) bool {
	return IsCountedMovement(k) && !slices.Contains(structuralKeys, k) && !slices.Contains(leadKeys, k)
}

func (g *pgen) newState(bi int) *bstate { return &bstate{needFig: g.needs[bi]} }

// planFigures decides how many figures each block shows: at random between its limits, but the
// movements of a gait (each at most twice) have to last for all blocks of the gait.
func (g *pgen) planFigures() {
	capacity := map[string]int{}
	serp := false
	for gait, pal := range g.palette {
		for k := range pal {
			switch {
			case k == "mitteltrab" || k == "mittelgalopp":
			case strings.HasPrefix(k, "schlangenlinie") || strings.HasPrefix(k, "einfache_schlangenlinie") || strings.HasPrefix(k, "doppelte_schlangenlinie"):
				if !serp {
					serp = true
					capacity[gait]++
				}
			default:
				capacity[gait] += maxPerMovement
			}
		}
	}
	if g.lead {
		capacity[GaitCanter]++ // the figure before the change of lead needs its own
	}
	g.needs = make([]int, len(g.blocks))
	focusWalk, walkFigures := false, 0
	for k := range g.palette[GaitWalk] {
		focusWalk = focusWalk || g.focus[k]
	}
	for i, b := range g.blocks {
		need := b.minFig
		if b.maxFig > b.minFig {
			need += g.rng.Intn(b.maxFig - b.minFig + 1)
		}
		if b.gait == GaitWalk && !b.free {
			// The walk is mostly medium walk on the track; a figure now and then, in a short test once.
			p := map[bool]float64{true: 0.4, false: 0.55}[g.short]
			need = 0
			if walkFigures == 0 && focusWalk || g.rng.Float64() < p && !(g.short && walkFigures > 0) {
				need = 1
				walkFigures++
			}
		}
		reserve := 0
		for _, later := range g.blocks[i+1:] {
			if later.gait == b.gait && later.minFig > 0 {
				reserve++
			}
		}
		need = max(0, min(need, capacity[b.gait]-reserve))
		g.needs[i] = need
		capacity[b.gait] -= need
	}
}

type pcand struct {
	o     *Option
	tempo string
	secs  float64
	score float64
	keys  []string
}

// blockDone: the block may end in the current state.
func (g *pgen) blockDone(b *block, s *bstate) bool {
	if s.lines < b.minLines || s.figs < s.needFig {
		return false
	}
	if b.change != "" && !s.changed {
		return false
	}
	if b.change == "rein" && s.after < b.after {
		return false
	}
	if d := g.dirOf(b.rein); d != "" && g.state.Dir != d {
		return false
	}
	if g.state.Dir != DirR && g.state.Dir != DirL {
		return false
	}
	if b.final && g.state.Node != "A" {
		return false
	}
	return true
}

func (g *pgen) minRemaining(bi int, s *bstate) int {
	n := max(0, g.blocks[bi].minLines-s.lines)
	for _, b := range g.blocks[bi+1:] {
		n += b.minLines
	}
	return n
}

// canStart reports whether the block can begin from the current state at all.
func (g *pgen) canStart(b *block) bool {
	if !b.free {
		return true
	}
	for _, o := range g.lib.From(g.state) {
		if g.freeLine(o) {
			return true
		}
	}
	return false
}

// freeLine: the lines a free walk may be ridden on: a long diagonal (E, A) or also across the arena (L, L**).
func (g *pgen) freeLine(o *Option) bool {
	if !o.Changes || !o.allows(GaitWalk) {
		return false
	}
	switch o.Kind {
	case "diagonal":
		return true
	case "halbe_bahn":
		return ClassRank(g.class) >= ClassRank("L")
	}
	return false
}

func (g *pgen) solve(bi int, s *bstate) bool {
	g.nodes++
	if g.nodes > g.maxNodes {
		return false
	}
	b := &g.blocks[bi]
	if len(g.specs)+g.minRemaining(bi, s)+2 > g.cap || g.elapsed+g.closeSec*0.8 > g.hi {
		return false
	}
	if g.blockDone(b, s) {
		if bi+1 == len(g.blocks) {
			if g.finish() {
				return true
			}
		} else if g.canStart(&g.blocks[bi+1]) {
			if g.solve(bi+1, g.newState(bi+1)) {
				return true
			}
		}
	}
	if s.lines >= b.maxLines || g.nodes > g.maxNodes {
		return false
	}
	cands := g.candidates(bi, s)
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	if len(cands) > 8 {
		cands = cands[:8]
	}
	for _, c := range cands {
		undo := g.push(c, b.gait)
		ns := *s
		ns.lines++
		isChange := c.o.Changes
		switch {
		case isChange:
			ns.changed = true
			ns.after = 0
		case c.o.Kind != "track":
			ns.figs++
			if s.changed {
				ns.after++
			}
		default:
			if s.changed {
				ns.after++
			}
		}
		if g.solve(bi, &ns) {
			return true
		}
		undo()
		if g.nodes > g.maxNodes {
			return false
		}
	}
	return false
}

func (g *pgen) candidates(bi int, s *bstate) []pcand {
	b := &g.blocks[bi]
	var out []pcand
	prevOpt := g.lib.ByID[g.prev.Option]
	defTempo := TempoWorking
	if b.gait == GaitWalk {
		defTempo = TempoMedium
	}
	remaining := math.Max(1, g.expLines-float64(len(g.specs)-2))
	want := (g.mid - g.elapsed - g.closeSec) / remaining
	for _, o := range g.lib.From(g.state) {
		if isSpecial(o.Kind) || !o.allows(b.gait) || o.Kind == "kurzkehrt" {
			continue
		}
		if turnsOnTheSpot(prevOpt, o) {
			continue
		}
		if o.Kind == "rueckwaerts" && b.gait != GaitWalk {
			continue // the rein back is shown from the walk
		}
		if o.Kind == "mittellinie" && !o.Changes {
			continue // the centre line is a change of rein here, not a short cut
		}
		isTrack := o.Kind == "track"
		if isTrack && o.Length/Speed(b.gait, defTempo) < minTrackSeconds {
			continue // no snippets of a few seconds
		}
		leadKind := o.Kind == "diagonale_einfacher_wechsel" || o.Kind == "aussengalopp_kurzseite"
		// Changes of rein and of lead belong to the blocks that ask for them.
		switch {
		case leadKind:
			if b.change != "lead" || s.changed {
				continue
			}
		case o.Changes:
			if b.change != "rein" || s.changed || !slices.Contains(reinChangers, o.Kind) {
				continue
			}
		}
		if b.change == "lead" && b.gait == GaitCanter && o.Changes && !leadKind {
			continue
		}
		if b.free && (!o.Changes || !g.freeLine(o)) {
			continue
		}
		if b.trackOnly && !isTrack {
			continue
		}
		if isTrack && g.prev.Gait == b.gait && prevOpt != nil && prevOpt.Kind == "track" {
			continue
		}
		figure := !isTrack && !o.Changes && !leadKind
		if !isTrack && g.usedOpt[o.ID] > 0 {
			continue // no figure twice on the same letters
		}
		if isLoop(o.Kind) && isLoop(prevOpt.Kind) {
			continue // a volte right after a circle at the same letter is no new figure
		}
		if figure {
			if s.figs >= b.maxFig {
				continue
			}
			// Work on the new rein after the change of rein; before the change of lead in the lead block.
			if b.change == "rein" && !s.changed || b.change == "lead" && s.changed {
				continue
			}
		}
		if b.change == "lead" && s.changed && !isTrack {
			continue
		}
		tempos := []string{defTempo}
		switch {
		case b.free:
			tempos = []string{TempoFree}
		case o.Fixed > 0:
		case o.Straight && b.gait == GaitTrot && g.allowed["mitteltrab"] && g.palette[GaitTrot]["mitteltrab"]:
			tempos = append(tempos, TempoMedium)
		case o.Straight && b.gait == GaitCanter && g.allowed["mittelgalopp"] && g.palette[GaitCanter]["mittelgalopp"]:
			tempos = append(tempos, TempoMedium)
		}
		for _, tempo := range tempos {
			keys := lineMovements(o, b.gait, tempo)
			if !g.keysOK(o, keys, b.gait) {
				continue
			}
			secs := g.secs(o, b.gait, tempo)
			c := pcand{o: o, tempo: tempo, secs: secs, keys: keys}
			c.score = g.score(c, b, s, figure, want)
			out = append(out, c)
		}
	}
	return out
}

func (g *pgen) secs(o *Option, gait, tempo string) float64 {
	t := o.Length/Speed(gait, tempo) + o.Fixed
	if g.prev != nil && (g.prev.Gait != gait || g.prev.Tempo != tempo) {
		t += TransitionSeconds
	}
	return t
}

// keysOK: the class and mastery, the palette of the test and the limits per movement.
func (g *pgen) keysOK(o *Option, keys []string, gait string) bool {
	for _, k := range keys {
		if !g.allowed[k] {
			return false
		}
		if slices.Contains(leadKeys, k) {
			if !g.lead {
				return false
			}
		} else if isPaletteKey(k) && !g.palette[gait][k] {
			return false
		}
		if IsCountedMovement(k) && g.counts[gait+"|"+k] >= maxPerMovement {
			return false
		}
	}
	if isSerpentine(o.Kind) && g.serp >= maxSerpentines {
		return false
	}
	if o.Lateral && g.lateral[o.Rein()] >= maxLateralRein {
		return false
	}
	return true
}

// reinChangers are the figures that change the rein in a practice test.
var reinChangers = []string{"diagonal", "halbe_bahn", "handwechsel_zirkel", "mittellinie", "schlangenlinie", "traversale"}

func isLoop(kind string) bool { return kind == "zirkel" || kind == "volte8" || kind == "volte10" }

func isSerpentine(kind string) bool {
	return kind == "schlangenlinie" || kind == "einfache_schlange" || kind == "doppelte_schlange"
}

func (g *pgen) score(c pcand, b *block, s *bstate, figure bool, want float64) float64 {
	sc := g.rng.Float64() * 2
	o := c.o
	fam := family(o.Kind)
	if n := len(g.kinds); n > 0 && family(g.kinds[n-1]) == fam {
		sc -= 3
	}
	if n := len(g.kinds); n > 1 && family(g.kinds[n-2]) == fam {
		sc -= 1
	}
	if figure {
		sc += 2
	}
	if o.Kind == "track" && s.figs < s.needFig {
		sc -= 1
	}
	sc -= 4 * float64(g.usedOpt[o.ID])
	for _, k := range c.keys {
		if !IsCountedMovement(k) {
			continue
		}
		if g.usedMove[k] == 0 {
			sc += 1
		} else {
			sc -= 1.5 * float64(g.usedMove[k])
		}
		if g.focus[k] {
			sc += 4
			if g.usedMove[k] == 0 {
				sc += 6
			}
		}
		for _, core := range coreMovements[g.class][b.gait] {
			if core == k {
				sc += 0.8
			}
		}
	}
	if g.hostsFocus(o.End, b.gait) && !isFocusLine(g, c.keys) {
		sc += 3 // leads to a figure of the focus
	}
	if want > 0 {
		sc -= 2.5 * math.Abs(c.secs-want) / math.Max(want, 6)
	}
	return sc
}

func isFocusLine(g *pgen, keys []string) bool {
	for _, k := range keys {
		if g.focus[k] {
			return true
		}
	}
	return false
}

// hostsFocus reports whether a figure of the focus that is not shown yet starts in the state.
func (g *pgen) hostsFocus(st State, gait string) bool {
	if len(g.focus) == 0 {
		return false
	}
	for _, o := range g.lib.From(st) {
		if isSpecial(o.Kind) || !o.allows(gait) || o.Kind == "track" {
			continue
		}
		for _, k := range figureMovementKeys(o.Kind, gait) {
			if g.focus[k] && g.usedMove[k] == 0 && g.palette[gait][k] {
				return true
			}
		}
	}
	return false
}

func (g *pgen) push(c pcand, gait string) (undo func()) {
	prevPrev, prevState, prevElapsed := g.prev, g.state, g.elapsed
	sp := Spec{Option: c.o.ID, Gait: gait, Tempo: c.tempo}
	g.specs = append(g.specs, sp)
	g.prev = &sp
	g.state = c.o.End
	g.elapsed += c.secs
	g.kinds = append(g.kinds, c.o.Kind)
	g.usedOpt[c.o.ID]++
	for _, k := range c.keys {
		g.usedMove[k]++
		if IsCountedMovement(k) {
			g.counts[gait+"|"+k]++
		}
	}
	if isSerpentine(c.o.Kind) {
		g.serp++
	}
	if c.o.Lateral {
		g.lateral[c.o.Rein()]++
	}
	return func() {
		g.specs = g.specs[:len(g.specs)-1]
		g.prev, g.state, g.elapsed = prevPrev, prevState, prevElapsed
		g.kinds = g.kinds[:len(g.kinds)-1]
		g.usedOpt[c.o.ID]--
		for _, k := range c.keys {
			g.usedMove[k]--
			if IsCountedMovement(k) {
				g.counts[gait+"|"+k]--
			}
		}
		if isSerpentine(c.o.Kind) {
			g.serp--
		}
		if c.o.Lateral {
			g.lateral[c.o.Rein()]--
		}
	}
}

// finish closes the plan and accepts it when the rules, the window, the reins and the line limit hold.
func (g *pgen) finish() bool {
	if g.state.Node != "A" {
		return false
	}
	specs := append(slices.Clone(g.specs), g.closing(g.state.Dir)...)
	if len(specs) > g.cap {
		return false
	}
	lines, err := BuildLines(g.lib, specs)
	if err != nil {
		return false
	}
	plan := &Plan{Kind: KindPractice, Class: g.class, Arena: g.lib.Arena.Size, Lines: lines}
	plan.Totals(g.lib)
	if plan.Seconds < g.lo || plan.Seconds > g.hi {
		return false
	}
	plan.Checks = CheckPractice(g.lib, g.class, plan)
	plan.Requirements = []RequirementStatus{}
	for _, c := range plan.Checks {
		if c.Rule && !c.OK {
			return false
		}
	}
	l, r, ok := reinShares(plan)
	if !ok || l < practiceReinMin || r < practiceReinMin {
		return false
	}
	g.result = plan
	return true
}
