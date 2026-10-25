package dressage

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

// State is where a rider is between two figures: a letter (or centre line point) and the direction
// of travel, DirR/DirL on the track, DirN/DirS on the centre line. Node "OUT" is outside the arena
// before the test, "DONE" after leaving it.
type State struct {
	Node string `json:"node"`
	Dir  string `json:"dir"`
}

// Kinds of option that are not figures of the practice structure.
const (
	kindEinreiten = "einreiten"
	kindWeiter    = "weiter"
	kindAbwenden  = "abwenden_halt"
	kindAusreiten = "ausreiten"
)

// Option is one figure of the library with fixed start and end: a way to get from one state to
// another, with its path and length. Options are static data per arena size.
type Option struct {
	ID    string
	Kind  string
	Verb  string // German label without the gait, "Durch die ganze Bahn wechseln"
	Full  string // complete line text with %s for the gait phrase (instead of Verb), may be empty
	Start State
	End   State
	// Letters is the route as shown in a line, "K–X–M".
	Letters string
	Path    []Point
	Length  float64
	// Left and Right are the metres ridden on the left and the right rein.
	Left, Right float64
	// Straight lines allow lengthenings; Lateral work and "Viereck verkleinern" are no walk lines
	// for the required walk.
	Straight bool
	Lateral  bool
	// Changes marks a change of rein (End.Dir differs from Start.Dir on the track).
	Changes bool
	// CanterOK marks figures that can be ridden in canter: they keep the lead or change it by a
	// simple change of lead.
	CanterOK bool
	Gaits    []string
	// Fixed is the time in seconds of the halt, rein back or change of lead of the figure.
	Fixed   float64
	Degrees int  // turn of a Kurzkehrt
	Counter bool // part of the figure is ridden in counter canter
	// ReinOverride is the rein shown for the movements ("left", "right"), "" to derive it.
	ReinOverride string
}

// Rein is the rein of the movements shown on the option: the start direction, the end direction
// for a change of rein, "" for the centre line and the entry.
func (o Option) Rein() string {
	if o.ReinOverride != "" {
		return o.ReinOverride
	}
	d := o.Start.Dir
	if o.Changes {
		d = o.End.Dir
	}
	switch d {
	case DirR:
		return "right"
	case DirL:
		return "left"
	}
	return ""
}

// Text is the line text of the option in a gait and tempo.
func (o Option) Text(gait, tempo string) string {
	if o.Full != "" {
		if strings.Contains(o.Full, "%s") {
			return fmt.Sprintf(o.Full, GaitPhrase(gait, tempo))
		}
		return o.Full
	}
	return o.Verb + " im " + GaitPhrase(gait, tempo)
}

func (o Option) allows(gait string) bool {
	for _, g := range o.Gaits {
		if g == gait {
			return true
		}
	}
	return false
}

// Library is the options of one arena, indexed by start state and id.
type Library struct {
	Arena   *Arena
	All     []Option
	ByID    map[string]*Option
	byStart map[State][]*Option
	seen    map[string]bool
}

// From returns the options that start in a state.
func (l *Library) From(s State) []*Option { return l.byStart[s] }

var (
	libMu sync.Mutex
	libs  = map[string]*Library{}
)

// LibraryFor returns the (cached) figure library of an arena size.
func LibraryFor(size string) (*Library, bool) {
	libMu.Lock()
	defer libMu.Unlock()
	if l, ok := libs[size]; ok {
		return l, true
	}
	a, ok := NewArena(size)
	if !ok {
		return nil, false
	}
	l := &Library{Arena: a, ByID: map[string]*Option{}, byStart: map[State][]*Option{}}
	l.build()
	libs[size] = l
	return l, true
}

func dirName(d string) string {
	switch d {
	case DirR:
		return "R"
	case DirL:
		return "L"
	}
	return d
}

func (l *Library) add(o Option) {
	if o.Length == 0 {
		o.Length = PathLength(o.Path)
	}
	if o.Left == 0 && o.Right == 0 && !o.neutral() {
		switch {
		case o.Start.Dir == o.End.Dir && o.Start.Dir == DirR:
			o.Right = o.Length
		case o.Start.Dir == o.End.Dir && o.Start.Dir == DirL:
			o.Left = o.Length
		default:
			o.Left, o.Right = o.Length/2, o.Length/2
		}
	}
	if o.ID == "" {
		o.ID = fmt.Sprintf("%s:%s:%s>%s%s", o.Kind, o.Letters, dirName(o.Start.Dir), o.End.Node, dirName(o.End.Dir))
	}
	if l.seen == nil {
		l.seen = map[string]bool{}
	}
	if l.seen[o.ID] {
		panic("dressage: duplicate option " + o.ID)
	}
	l.seen[o.ID] = true
	if !o.CanterOK {
		var g []string
		for _, x := range o.Gaits {
			if x != GaitCanter {
				g = append(g, x)
			}
		}
		o.Gaits = g
	}
	l.All = append(l.All, o)
}

// neutral options (changes of rein, centre line, entry and exit) belong to neither rein.
func (o Option) neutral() bool {
	switch o.Kind {
	case "diagonal", "halbe_bahn", "mittellinie", "rueckwaerts", "traversale", "handwechsel_zirkel",
		kindEinreiten, kindWeiter, kindAbwenden, kindAusreiten, "kurzkehrt", "aussengalopp_kurzseite", "diagonale_einfacher_wechsel":
		return true
	}
	return false
}

func (l *Library) index() {
	for i := range l.All {
		o := &l.All[i]
		l.ByID[o.ID] = o
		l.byStart[o.Start] = append(l.byStart[o.Start], o)
	}
}

var allGaits = []string{GaitWalk, GaitTrot, GaitCanter}

// headingAt is the heading (radians) of travel on the track at a letter in a direction.
func (a *Arena) headingAt(letter, dir string) float64 {
	s := a.TrackS(letter)
	d := 0.2
	if dir == DirL {
		d = -d
	}
	p0, p1 := a.PointAt(s-d), a.PointAt(s+d)
	return math.Atan2(p1[1]-p0[1], p1[0]-p0[0])
}

func (l *Library) build() {
	a := l.Arena
	dirs := []string{DirR, DirL}
	ringFor := func(dir string) []string {
		r := a.TrackLetters()
		if dir == DirL {
			out := []string{r[0]}
			for i := len(r) - 1; i >= 1; i-- {
				out = append(out, r[i])
			}
			return out
		}
		return r
	}
	sideY := func(name string) float64 { return a.MustPos(name)[1] }
	inwardSign := func(name string) float64 {
		if a.MustPos(name)[0] > 10 {
			return -1
		}
		return 1
	}

	// Track sections, lateral work along the long side.
	for _, dir := range dirs {
		ring := ringFor(dir)
		n := len(ring)
		for i := 0; i < n; i++ {
			for k := 1; k <= 4; k++ {
				from, to := ring[i], ring[(i+k)%n]
				path := a.TrackPath(from, to, dir)
				l.add(Option{
					Kind: "track", Verb: "Auf dem Hufschlag", Letters: from + "–" + to,
					Start: State{from, dir}, End: State{to, dir}, Path: path, Length: a.TrackDistance(from, to, dir),
					Straight: onePiece(a, from, to, dir),
					CanterOK: true, Gaits: allGaits,
				})
				if a.SameSide(from, to) && a.TrackDistance(from, to, dir) >= 12 && onePiece(a, from, to, dir) {
					for _, lat := range []struct{ kind, verb string }{{"schulterherein", "Schulterherein"}, {"travers", "Travers"}, {"renvers", "Renvers"}} {
						l.add(Option{Kind: lat.kind, Verb: lat.verb, Letters: from + "–" + to,
							Start: State{from, dir}, End: State{to, dir}, Path: path, Length: a.TrackDistance(from, to, dir),
							Lateral: true, Gaits: []string{GaitTrot}})
					}
				}
			}
		}
	}

	// Circles and voltes.
	for _, dir := range dirs {
		for _, name := range a.TrackLetters() {
			start := a.MustPos(name)
			h := a.headingAt(name, dir)
			cw := dir == DirR
			if name == "A" || name == "C" || name == "E" || name == "B" {
				l.add(Option{Kind: "zirkel", Verb: "Zirkel", Letters: name, Start: State{name, dir}, End: State{name, dir},
					Path: circleFrom(start, h, 20, cw, 2*math.Pi), CanterOK: true, Gaits: allGaits})
			}
			for _, v := range []struct {
				kind string
				d    float64
				verb string
			}{{"volte10", 10, "Volte (10 m)"}, {"volte8", 8, "Volte (8 m)"}} {
				l.add(Option{Kind: v.kind, Verb: v.verb, Letters: name, Start: State{name, dir}, End: State{name, dir},
					Path: circleFrom(start, h, v.d, cw, 2*math.Pi), CanterOK: true, Gaits: allGaits})
			}
		}
	}

	// Changes of rein: long diagonals.
	diag := func(from, to string, startDir, endDir string) Option {
		return Option{Kind: "diagonal", Verb: "Durch die ganze Bahn wechseln", Letters: from + "–X–" + to,
			Start: State{from, startDir}, End: State{to, endDir}, Path: []Point{a.MustPos(from), a.MustPos(to)},
			Straight: true, Changes: true, Gaits: allGaits}
	}
	for _, d := range []struct{ from, to, sd, ed string }{
		{"K", "M", DirR, DirL}, {"F", "H", DirL, DirR}, {"M", "K", DirR, DirL}, {"H", "F", DirL, DirR},
	} {
		l.add(diag(d.from, d.to, d.sd, d.ed))
		// Canter: simple change of lead over the diagonal.
		o := diag(d.from, d.to, d.sd, d.ed)
		o.Kind, o.Fixed, o.CanterOK, o.Gaits = "diagonale_einfacher_wechsel", SimpleChangeSeconds, true, []string{GaitCanter}
		o.Straight = false
		o.Full = "Durch die ganze Bahn wechseln, bei X einfacher Galoppwechsel"
		l.add(o)
		// Canter: diagonal, counter canter along the short side to the next letter, simple change there.
		end := "C"
		endDir := d.ed
		if d.to == "K" || d.to == "F" {
			end = "A"
		}
		tail := a.TrackPath(d.to, end, endDir)
		l.add(Option{Kind: "aussengalopp_kurzseite", Full: "Durch die ganze Bahn wechseln, im Außengalopp bis " + end + ", dort einfacher Galoppwechsel",
			Letters: d.from + "–X–" + d.to + "–" + end, Start: State{d.from, d.sd}, End: State{end, endDir},
			Path:    concat([]Point{a.MustPos(d.from), a.MustPos(d.to)}, tail),
			Length:  dist(a.MustPos(d.from), a.MustPos(d.to)) + a.TrackDistance(d.to, end, endDir),
			Changes: true, CanterOK: true, Counter: true, Fixed: SimpleChangeSeconds, Gaits: []string{GaitCanter}, ReinOverride: reinWord(endDir)})
	}
	// Across the arena between letters of the long sides.
	pairs := [][2]string{{"E", "B"}}
	if a.Size == Size20x60 {
		pairs = [][2]string{{"V", "P"}, {"E", "B"}, {"S", "R"}}
	}
	for _, p := range pairs {
		le, ri := p[0], p[1]
		for _, c := range []struct {
			from, to, sd, ed string
		}{{le, ri, DirR, DirL}, {le, ri, DirL, DirR}, {ri, le, DirR, DirL}, {ri, le, DirL, DirR}} {
			letters := c.from + "–" + c.to
			if le == "E" {
				letters = c.from + "–X–" + c.to
			}
			l.add(Option{Kind: "halbe_bahn", Verb: "Durch die halbe Bahn wechseln", Letters: letters,
				Start: State{c.from, c.sd}, End: State{c.to, c.ed}, Path: []Point{a.MustPos(c.from), a.MustPos(c.to)},
				Straight: true, Changes: true, Gaits: allGaits})
		}
	}
	// Centre line family: from A to C and from C to A.
	for _, startName := range []string{"A", "C"} {
		endName := "C"
		if startName == "C" {
			endName = "A"
		}
		runs := []struct {
			kind, verb, suffix string
			path               []Point
			straight           bool
			fixed              float64
			gaits              []string
		}{
			{"mittellinie", "Auf der Mittellinie", "", []Point{{10, 0}, {10, a.Length}}, true, 0, allGaits},
			{"rueckwaerts", "Auf der Mittellinie, bei X halten, 4 Tritte rückwärtsrichten", "", []Point{{10, 0}, {10, a.Length}}, true,
				HaltSeconds + 4*ReinBackStepSec + 2*TransitionSeconds, []string{GaitWalk, GaitTrot}},
		}
		for _, loops := range []int{3, 4} {
			for _, left := range []bool{true, false} {
				side := "rechts"
				if left {
					side = "links"
				}
				runs = append(runs, struct {
					kind, verb, suffix string
					path               []Point
					straight           bool
					fixed              float64
					gaits              []string
				}{"schlangenlinie", fmt.Sprintf("Schlangenlinie durch die ganze Bahn (%d Bögen, erster Bogen %s)", loops, side),
					fmt.Sprintf("%d%s", loops, side[:1]), serpentine(0, a.Length, loops, 8, left), false, 0, []string{GaitWalk, GaitTrot}})
			}
		}
		for _, run := range runs {
			path := run.path
			if startName == "C" {
				path = reverse(run.path)
			}
			for _, sd := range dirs {
				for _, ed := range dirs {
					letters := startName + "–" + endName
					if run.kind != "schlangenlinie" {
						letters = startName + "–X–" + endName
					}
					o := Option{Kind: run.kind, Verb: run.verb, Letters: letters, Start: State{startName, sd}, End: State{endName, ed},
						Path: path, Straight: run.straight, Changes: sd != ed, Fixed: run.fixed, Gaits: run.gaits}
					if run.kind == "mittellinie" {
						o.CanterOK = sd == ed
						if o.CanterOK {
							o.Gaits = allGaits
						}
					}
					o.ID = fmt.Sprintf("%s%s:%s:%s>%s%s", run.kind, run.suffix, letters, dirName(sd), endName, dirName(ed))
					if run.kind == "schlangenlinie" {
						o.Left, o.Right = 0, 0
						o.Length = PathLength(path)
						o.Left, o.Right = o.Length/2, o.Length/2
					}
					l.add(o)
				}
			}
		}
	}
	// Through the circle (20×40 only): half circle at A, counter circle at C.
	if a.Size == Size20x40 {
		half1 := circleFrom(Point{10, 0}, math.Pi, 20, true, math.Pi) // A cw to X
		half2 := arcPoints(Point{10, 30}, 10, -math.Pi/2, math.Pi)    // X ccw to C
		base := concat(half1, half2)
		for _, c := range []struct {
			path []Point
			st   State
			en   State
			l    string
		}{
			{base, State{"A", DirR}, State{"C", DirL}, "A–X–C"},
			{mirrorX(base), State{"A", DirL}, State{"C", DirR}, "A–X–C"},
			{rotate180(base, a.Length), State{"C", DirR}, State{"A", DirL}, "C–X–A"},
			{rotate180(mirrorX(base), a.Length), State{"C", DirL}, State{"A", DirR}, "C–X–A"},
		} {
			l.add(Option{Kind: "handwechsel_zirkel", Verb: "Durch den Zirkel wechseln", Letters: c.l, Start: c.st, End: c.en,
				Path: c.path, Changes: true, CanterOK: false, Gaits: []string{GaitWalk, GaitTrot}})
		}
	}
	// Leg yield, "Viereck verkleinern", single and double serpentines, shoulder-in lines along a long side.
	for _, dir := range dirs {
		ring := ringFor(dir)
		n := len(ring)
		for i := 0; i < n; i++ {
			for k := 1; k <= 3; k++ {
				from, to := ring[i], ring[(i+k)%n]
				if !a.SameSide(from, to) || !onePiece(a, from, to, dir) {
					continue
				}
				x0 := a.MustPos(from)[0]
				in := inwardSign(from)
				y0, y1 := sideY(from), sideY(to)
				sg := 1.0
				if y1 < y0 {
					sg = -1
				}
				d := math.Abs(y1 - y0)
				st, en := State{from, dir}, State{to, dir}
				letters := from + "–" + to
				if k == 1 && d >= 12 {
					l.add(Option{Kind: "schenkelweichen", Verb: "Schenkelweichen von der Viertellinie zum Hufschlag", Letters: letters, Start: st, End: en,
						Path:    rounded([]Point{{x0, y0}, {x0 + in*5, y0 + sg*3}, {x0 + in*5, y1 - sg*8}, {x0, y1}}, 2),
						Lateral: true, Gaits: []string{GaitWalk, GaitTrot}})
					l.add(Option{Kind: "einfache_schlange", Verb: "Einfache Schlangenlinie an der langen Seite", Letters: letters, Start: st, End: en,
						Path: bows(x0, y0, y1, 1, 5), Gaits: []string{GaitWalk, GaitTrot}})
				}
				if d >= 20 {
					l.add(Option{Kind: "viereck", Verb: "Viereck verkleinern und vergrößern", Letters: letters, Start: st, End: en,
						Path:    rounded([]Point{{x0, y0}, {x0 + in*5, y0 + sg*5}, {x0 + in*5, y1 - sg*5}, {x0, y1}}, 2.5),
						Lateral: true, Gaits: []string{GaitWalk, GaitTrot}})
				}
				if d >= 24 {
					l.add(Option{Kind: "doppelte_schlange", Verb: "Doppelte Schlangenlinie an der langen Seite", Letters: letters, Start: st, End: en,
						Path: bows(x0, y0, y1, 2, 5), Gaits: []string{GaitWalk, GaitTrot}})
				}
			}
		}
	}
	// Short turn (Kurzkehrt) at the letters of the long sides: 180° turn, back to the track in the other direction.
	for _, dir := range dirs {
		for _, name := range a.TrackLetters() {
			if !a.OnTrack(name) || a.byName[name].Where == "short" {
				continue
			}
			p := a.MustPos(name)
			in := inwardSign(name)
			heading := a.headingAt(name, dir)
			sg := math.Sin(heading) // +1 up, -1 down
			c := Point{p[0] + in*1.5, p[1] + 2*sg}
			var a0, sweep float64
			a0 = math.Atan2(0, -in*1.5)
			// Turn towards the arena: for an upward heading on the left side that is clockwise.
			cw := (dir == DirR)
			sweep = math.Pi
			if cw {
				sweep = -math.Pi
			}
			semi := arcPoints(c, 1.5, a0, sweep)
			path := concat([]Point{p, {p[0], p[1] + 2*sg}}, semi, []Point{{p[0], p[1]}})
			flip := DirL
			if dir == DirL {
				flip = DirR
			}
			l.add(Option{Kind: "kurzkehrt", Verb: "Kurzkehrt", Full: "Kurzkehrt im %s", Letters: name, Start: State{name, dir}, End: State{name, flip},
				Path: path, Changes: true, Degrees: 180, Gaits: []string{GaitWalk}})
		}
	}
	// Half pass from the centre line to the track.
	{
		mid := a.Length / 2
		for _, side := range []float64{0, Width} {
			endName, endDir := "E", DirL
			if side == Width {
				endName, endDir = "B", DirR
			}
			for _, sd := range dirs {
				path := rounded([]Point{{10, a.Length}, {10, a.Length - 6}, {side, mid}}, 3)
				l.add(Option{Kind: "traversale", Verb: "Traversale von der Mittellinie zum Hufschlag", Letters: "C–G–" + endName,
					Start: State{"C", sd}, End: State{endName, endDir}, Path: path, Changes: sd != endDir, Lateral: true, Gaits: []string{GaitTrot}})
				// From A, the same turned around the centre of the arena.
				en2, ed2 := "B", DirL
				if endName == "B" {
					en2, ed2 = "E", DirR
				}
				l.add(Option{Kind: "traversale", Verb: "Traversale von der Mittellinie zum Hufschlag", Letters: "A–D–" + en2,
					Start: State{"A", sd}, End: State{en2, ed2}, Path: rotate180(path, a.Length), Changes: sd != ed2, Lateral: true, Gaits: []string{GaitTrot}})
			}
		}
	}
	// Entry, exit and the centre line of the salute.
	l.addSalute()
	l.index()
}

func reinWord(dir string) string {
	if dir == DirR {
		return "right"
	}
	return "left"
}

// onePiece reports that the track route from to without passing a corner (both letters on one
// long side and the shorter way in the direction of travel).
func onePiece(a *Arena, from, to, dir string) bool {
	if !a.SameSide(from, to) {
		return false
	}
	d := a.TrackDistance(from, to, dir)
	direct := math.Abs(a.MustPos(from)[1] - a.MustPos(to)[1])
	return math.Abs(d-direct) < 1e-6
}

// SaluteNodes are the centre line points the salute is shown at: X, or G.
func (l *Library) addSalute() {
	a := l.Arena
	for _, node := range []string{"X", "G"} {
		y := a.MustPos(node)[1]
		l.add(Option{ID: kindEinreiten + ":" + node, Kind: kindEinreiten, Letters: "A–" + node,
			Full:  "Einreiten im %s, bei " + node + " halten, grüßen",
			Start: State{"OUT", ""}, End: State{node, DirN}, Path: []Point{{10, 0}, {10, y}}, Fixed: SaluteSeconds,
			Straight: true, Gaits: []string{GaitTrot}})
		for _, end := range dirs2() {
			word := "rechts"
			if end == DirL {
				word = "links"
			}
			l.add(Option{ID: kindWeiter + ":" + node + ":" + end, Kind: kindWeiter, Letters: node + "–C",
				Full:  "Geradeaus im %s, bei C " + word + " abwenden",
				Start: State{node, DirN}, End: State{"C", end}, Path: []Point{{10, y}, {10, a.Length}}, Straight: true, Gaits: []string{GaitTrot}})
		}
		for _, d := range dirs2() {
			l.add(Option{ID: kindAbwenden + ":" + node + ":" + d, Kind: kindAbwenden, Letters: "A–" + node,
				Full:  "Bei A abwenden, auf der Mittellinie im %s nach " + node + ", halten, grüßen",
				Start: State{"A", d}, End: State{node + "*", DirN}, Path: []Point{{10, 0}, {10, y}}, Fixed: SaluteSeconds,
				Straight: true, Gaits: []string{GaitTrot}})
		}
		// Leave the arena on the track at A.
		var out []Point
		path := []Point{{10, y}, {20, y}}
		if node == "X" {
			out = concat(path, a.TrackPath("B", "A", DirR))
		} else {
			out = concat(path, a.TrackPath("M", "A", DirR))
		}
		l.add(Option{ID: kindAusreiten + ":" + node, Kind: kindAusreiten, Letters: node + "–A",
			Full:  "Im %s zum Hufschlag und bei A hinausreiten",
			Start: State{node + "*", DirN}, End: State{"DONE", ""}, Path: out, Gaits: []string{GaitWalk}})
	}
}

func dirs2() []string { return []string{DirR, DirL} }
