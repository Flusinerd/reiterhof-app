// Package dressage is the pure (no HTTP, no database, no clock) model of practice tests and freestyles
// (docs/domains/dressage-tests.md): the arena with its letters, a library of figures with their paths,
// the movement catalog with lowest class and source, speeds, a generator for practice tests and for
// freestyles (deterministic by seed) and the rule checks.
//
// Copyright: nothing in this package is taken from an FN Aufgabenheft, Leitfaden or score sheet.
// It holds facts (arena geometry, names of movements, the general rules of the Kür leaflet in our
// own words); requirements of a freestyle class are entered by the user.
package dressage

import (
	"math"
	"sort"
)

// Point is a position in the arena in metres: x across (0 = left long side seen from A), y along
// (0 = the short side with A).
type Point [2]float64

// Arena sizes.
const (
	Size20x40 = "20x40"
	Size20x60 = "20x60"
)

// Sizes lists the supported arena sizes.
var Sizes = []string{Size20x40, Size20x60}

// Width of both arenas in metres.
const Width = 20.0

// CornerRadius is the radius of the corners of the track in metres. The FN sources the app uses do
// not give a radius per class, so one radius is used for all classes (docs/domains/dressage-tests.md).
const CornerRadius = 3.0

// Rein directions: the direction of travel on the track. DirR is clockwise (right rein, "rechte
// Hand", A, K, E, H, C, M, B, F), DirL counter-clockwise (left rein).
const (
	DirR = "R"
	DirL = "L"
	// DirN and DirS are the directions on the centre line (towards C, towards A).
	DirN = "N"
	DirS = "S"
)

// Letter is a marked point of the arena.
type Letter struct {
	Name string `json:"name"`
	At   Point  `json:"at"`
	// Where is "short" (A, C), "left" or "right" (long sides) or "centre" (centre line).
	Where string `json:"where"`
}

// Arena is a dressage arena with its letters and the track around it.
type Arena struct {
	Size    string
	Length  float64
	letters []Letter
	byName  map[string]Letter
	track   []trackPiece
	perim   float64
	// ring lists the track letters clockwise starting at A.
	ring []string
}

type trackPiece struct {
	arc    bool
	p0, p1 Point // line
	c      Point // arc centre
	a0, a1 float64
	length float64
	s0     float64 // track position at the start of the piece
}

// NewArena builds the arena of a size, false for an unknown size.
func NewArena(size string) (*Arena, bool) {
	var l float64
	var left, right, centre []Letter
	switch size {
	case Size20x40:
		l = 40
		left = longSide("left", 0, []string{"K", "E", "H"}, []float64{6, 20, 34})
		right = longSide("right", Width, []string{"F", "B", "M"}, []float64{6, 20, 34})
		centre = centreLine([]string{"D", "X", "G"}, []float64{6, 20, 34})
	case Size20x60:
		l = 60
		left = longSide("left", 0, []string{"K", "V", "E", "S", "H"}, []float64{6, 18, 30, 42, 54})
		right = longSide("right", Width, []string{"F", "P", "B", "R", "M"}, []float64{6, 18, 30, 42, 54})
		centre = centreLine([]string{"D", "L", "X", "I", "G"}, []float64{6, 18, 30, 42, 54})
	default:
		return nil, false
	}
	a := &Arena{Size: size, Length: l, byName: map[string]Letter{}}
	a.letters = append(a.letters, Letter{"A", Point{10, 0}, "short"}, Letter{"C", Point{10, l}, "short"})
	a.letters = append(a.letters, left...)
	a.letters = append(a.letters, right...)
	a.letters = append(a.letters, centre...)
	for _, le := range a.letters {
		a.byName[le.Name] = le
	}
	// Clockwise: A, left side upwards, C, right side downwards.
	a.ring = append(a.ring, "A")
	for _, le := range left {
		a.ring = append(a.ring, le.Name)
	}
	a.ring = append(a.ring, "C")
	for i := len(right) - 1; i >= 0; i-- {
		a.ring = append(a.ring, right[i].Name)
	}
	a.buildTrack()
	return a, true
}

func longSide(where string, x float64, names []string, ys []float64) []Letter {
	out := make([]Letter, len(names))
	for i, n := range names {
		out[i] = Letter{n, Point{x, ys[i]}, where}
	}
	return out
}

func centreLine(names []string, ys []float64) []Letter {
	out := make([]Letter, len(names))
	for i, n := range names {
		out[i] = Letter{n, Point{10, ys[i]}, "centre"}
	}
	return out
}

// Letters returns all letters of the arena (A, C, the long sides, the centre line).
func (a *Arena) Letters() []Letter { return append([]Letter(nil), a.letters...) }

// Pos returns the position of a letter.
func (a *Arena) Pos(name string) (Point, bool) {
	l, ok := a.byName[name]
	return l.At, ok
}

// MustPos returns the position of a letter and panics for an unknown one (static data only).
func (a *Arena) MustPos(name string) Point {
	p, ok := a.Pos(name)
	if !ok {
		panic("dressage: unknown letter " + name)
	}
	return p
}

// TrackLetters lists the letters on the track clockwise from A (A, K, E, H, C, M, B, F).
func (a *Arena) TrackLetters() []string { return append([]string(nil), a.ring...) }

// OnTrack reports whether the letter is on the fence line.
func (a *Arena) OnTrack(name string) bool {
	l, ok := a.byName[name]
	return ok && l.Where != "centre"
}

// buildTrack builds the rounded rectangle clockwise from A.
func (a *Arena) buildTrack() {
	r, l := CornerRadius, a.Length
	type seg struct {
		arc    bool
		p0, p1 Point
		c      Point
		a0, a1 float64
	}
	pi := math.Pi
	segs := []seg{
		{p0: Point{10, 0}, p1: Point{r, 0}},
		{arc: true, c: Point{r, r}, a0: -pi / 2, a1: -pi}, // bottom-left, clockwise = decreasing angle
		{p0: Point{0, r}, p1: Point{0, l - r}},
		{arc: true, c: Point{r, l - r}, a0: pi, a1: pi / 2},
		{p0: Point{r, l}, p1: Point{Width - r, l}},
		{arc: true, c: Point{Width - r, l - r}, a0: pi / 2, a1: 0},
		{p0: Point{Width, l - r}, p1: Point{Width, r}},
		{arc: true, c: Point{Width - r, r}, a0: 0, a1: -pi / 2},
		{p0: Point{Width - r, 0}, p1: Point{10, 0}},
	}
	s := 0.0
	for _, g := range segs {
		p := trackPiece{arc: g.arc, p0: g.p0, p1: g.p1, c: g.c, a0: g.a0, a1: g.a1, s0: s}
		if g.arc {
			p.length = r * math.Abs(g.a1-g.a0)
		} else {
			p.length = dist(g.p0, g.p1)
		}
		s += p.length
		a.track = append(a.track, p)
	}
	a.perim = s
}

// Perimeter is the length of the track in metres.
func (a *Arena) Perimeter() float64 { return a.perim }

// PointAt returns the point of the track s metres clockwise from A.
func (a *Arena) PointAt(s float64) Point {
	s = math.Mod(s, a.perim)
	if s < 0 {
		s += a.perim
	}
	for _, p := range a.track {
		if s <= p.s0+p.length+1e-9 {
			t := s - p.s0
			if p.arc {
				ang := p.a0 + (p.a1-p.a0)*t/p.length
				return Point{p.c[0] + CornerRadius*math.Cos(ang), p.c[1] + CornerRadius*math.Sin(ang)}
			}
			f := t / p.length
			return Point{p.p0[0] + (p.p1[0]-p.p0[0])*f, p.p0[1] + (p.p1[1]-p.p0[1])*f}
		}
	}
	return Point{10, 0}
}

// TrackS returns the track position of a letter on the fence line, clockwise from A.
func (a *Arena) TrackS(name string) float64 {
	p := a.MustPos(name)
	for _, t := range a.track {
		if t.arc {
			continue
		}
		if onSegment(t.p0, t.p1, p) {
			return t.s0 + dist(t.p0, p)
		}
	}
	panic("dressage: letter not on the track " + name)
}

func onSegment(a, b, p Point) bool {
	d := dist(a, b)
	return math.Abs(dist(a, p)+dist(p, b)-d) < 1e-6
}

// TrackDistance is the distance along the track from one letter to another in the direction dir
// (DirR clockwise, DirL counter-clockwise); the same letter gives a full lap.
func (a *Arena) TrackDistance(from, to, dir string) float64 {
	d := a.TrackS(to) - a.TrackS(from)
	if dir == DirL {
		d = -d
	}
	d = math.Mod(d, a.perim)
	if d <= 1e-9 {
		d += a.perim
	}
	return d
}

// TrackPath returns the points of the track from one letter to another in the direction dir.
func (a *Arena) TrackPath(from, to, dir string) []Point {
	total := a.TrackDistance(from, to, dir)
	s0 := a.TrackS(from)
	sign := 1.0
	if dir == DirL {
		sign = -1
	}
	n := int(math.Ceil(total / 0.75))
	out := make([]Point, 0, n+1)
	for i := 0; i <= n; i++ {
		d := math.Min(total, float64(i)*total/float64(n))
		out = append(out, a.PointAt(s0+sign*d))
	}
	return out
}

// LettersBetween returns the track letters passed from one letter to another (inclusive).
func (a *Arena) LettersBetween(from, to, dir string) []string {
	total := a.TrackDistance(from, to, dir)
	type ls struct {
		n string
		d float64
	}
	var in []ls
	for _, n := range a.ring {
		d := a.TrackDistance(from, n, dir)
		if n == from {
			d = 0
		}
		if d <= total+1e-9 {
			in = append(in, ls{n, d})
		}
	}
	sort.Slice(in, func(i, j int) bool { return in[i].d < in[j].d })
	out := make([]string, len(in))
	for i, x := range in {
		out[i] = x.n
	}
	return out
}

// SameSide reports whether two track letters are on the same straight long side (no corner between).
func (a *Arena) SameSide(x, y string) bool {
	lx, ly := a.byName[x], a.byName[y]
	return lx.Where == ly.Where && (lx.Where == "left" || lx.Where == "right")
}

// NodePos is the position of a state node: a letter, "X*" and "G*" (the salute point after the
// final halt), "OUT" and "DONE" (the entry at A).
func (a *Arena) NodePos(node string) (Point, bool) {
	switch node {
	case "OUT", "DONE":
		return a.Pos("A")
	}
	if n := len(node); n > 1 && node[n-1] == '*' {
		node = node[:n-1]
	}
	return a.Pos(node)
}

func dist(p, q Point) float64 { return math.Hypot(p[0]-q[0], p[1]-q[1]) }

// PathLength is the length of a polyline in metres.
func PathLength(pts []Point) float64 {
	l := 0.0
	for i := 1; i < len(pts); i++ {
		l += dist(pts[i-1], pts[i])
	}
	return l
}
