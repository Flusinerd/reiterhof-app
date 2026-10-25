package dressage

import (
	"fmt"
	"math"
	"slices"
)

// Kinds of plan.
const (
	KindPractice  = "practice"
	KindFreestyle = "freestyle"
)

// Spec is the part of a line the user's edits may change: which figure, ridden in which gait and
// tempo. Everything else (path, length, text, seconds) is derived from the figure library.
type Spec struct {
	Option string `json:"option"`
	Gait   string `json:"gait"`
	Tempo  string `json:"tempo"`
}

// Line is one numbered line of a plan.
type Line struct {
	N         int      `json:"n"`
	Option    string   `json:"option"`
	Letters   string   `json:"letters"`
	Text      string   `json:"text"`
	Gait      string   `json:"gait"`
	Tempo     string   `json:"tempo"`
	Rein      string   `json:"rein"`
	Seconds   float64  `json:"seconds"`
	Meters    float64  `json:"meters"`
	Figure    string   `json:"figure"`
	Movements []string `json:"movements"`
	Straight  bool     `json:"straight"`
	Lateral   bool     `json:"lateral"`
	Counter   bool     `json:"counter,omitempty"`
	Degrees   int      `json:"degrees,omitempty"`
	Start     State    `json:"start"`
	End       State    `json:"end"`
	Path      []Point  `json:"path"`
}

// Spec returns the editable part of the line.
func (l Line) Spec() Spec { return Spec{Option: l.Option, Gait: l.Gait, Tempo: l.Tempo} }

// Check is the result of one rule or choreography criterion. Rule checks (Rule true) are the rules
// of the leaflet and the structure; a failing one is an error, the others are advice.
type Check struct {
	Key  string `json:"key"`
	OK   bool   `json:"ok"`
	Rule bool   `json:"rule"`
	Text string `json:"text"`
}

// MusicPart is a run of consecutive lines in the same gait: the part of the music the user cuts.
type MusicPart struct {
	Gait    string  `json:"gait"`
	Seconds float64 `json:"seconds"`
	From    int     `json:"from"`
	To      int     `json:"to"`
	// Cadence is the stride frequency range (strides per minute) of the gait, the basis for the tempo of the music.
	Cadence [2]int `json:"cadence"`
}

// ReinSeconds is the time on each rein.
type ReinSeconds struct {
	Left  float64 `json:"left"`
	Right float64 `json:"right"`
}

// GaitSeconds is the time in each gait.
type GaitSeconds struct {
	Walk   float64 `json:"walk"`
	Trot   float64 `json:"trot"`
	Canter float64 `json:"canter"`
}

// RequirementStatus is one required movement of a freestyle: how often it is asked and shown.
type RequirementStatus struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Rein  string `json:"rein"`
	Count int    `json:"count"`
	Shown int    `json:"shown"`
	Met   bool   `json:"met"`
}

// Plan is a generated or edited test.
type Plan struct {
	Kind         string              `json:"kind"`
	Class        string              `json:"class"`
	Arena        string              `json:"arena"`
	Seed         int64               `json:"seed"`
	Lines        []Line              `json:"lines"`
	Seconds      float64             `json:"seconds"`
	Reins        ReinSeconds         `json:"reins"`
	Gaits        GaitSeconds         `json:"gaits"`
	Music        []MusicPart         `json:"music"`
	Checks       []Check             `json:"checks"`
	Requirements []RequirementStatus `json:"requirements"`
}

// BuildLines turns specs into lines: path, length, text, movements and seconds come from the
// library. An unknown option or a gait the option cannot be ridden in is an error.
func BuildLines(lib *Library, specs []Spec) ([]Line, error) {
	lines := make([]Line, 0, len(specs))
	var prev *Spec
	for i, sp := range specs {
		o, ok := lib.ByID[sp.Option]
		if !ok {
			return nil, fmt.Errorf("Zeile %d: unbekannte Figur", i+1)
		}
		if !o.allows(sp.Gait) {
			return nil, fmt.Errorf("Zeile %d: %s ist im %s nicht vorgesehen", i+1, o.Verb, GaitLabel(sp.Gait))
		}
		if sp.Tempo == "" {
			sp.Tempo = TempoWorking
		}
		lines = append(lines, makeLine(lib, o, sp, prev, i+1))
		s := sp
		prev = &s
	}
	return lines, nil
}

func makeLine(lib *Library, o *Option, sp Spec, prev *Spec, n int) Line {
	speed := Speed(sp.Gait, sp.Tempo)
	moving := o.Length / speed
	secs := moving + o.Fixed
	if prev != nil && (prev.Gait != sp.Gait || prev.Tempo != sp.Tempo) {
		secs += TransitionSeconds
	}
	keys := lineMovements(o, sp.Gait, sp.Tempo)
	return Line{
		N: n, Option: o.ID, Letters: o.Letters, Text: o.Text(sp.Gait, sp.Tempo), Gait: sp.Gait, Tempo: sp.Tempo,
		Rein: o.Rein(), Seconds: round1(secs), Meters: round1(o.Length), Figure: o.Kind, Movements: keys,
		Straight: o.Straight, Lateral: o.Lateral, Counter: o.Counter, Degrees: o.Degrees,
		Start: o.Start, End: o.End, Path: roundPath(o.Path),
	}
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func roundPath(p []Point) []Point {
	out := make([]Point, len(p))
	for i, q := range p {
		out[i] = Point{math.Round(q[0]*100) / 100, math.Round(q[1]*100) / 100}
	}
	return out
}

// lineMovements are the movement keys a figure shows in a gait and tempo.
func lineMovements(o *Option, gait, tempo string) []string {
	var keys []string
	switch o.Kind {
	case kindEinreiten, kindAbwenden:
		keys = append(keys, "halten_gruss", GaitMovementKey(gait, tempo))
	case kindWeiter:
		keys = append(keys, GaitMovementKey(gait, tempo))
	case kindAusreiten:
		keys = append(keys, GaitMovementKey(gait, tempo))
	default:
		keys = append(keys, GaitMovementKey(gait, tempo))
		keys = append(keys, figureMovementKeys(o.Kind, gait)...)
	}
	// Canter on a counter lead, a Kurzkehrt and the like come with the figure; drop duplicates.
	out := keys[:0]
	for _, k := range keys {
		if k != "" && !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// Totals fills the derived fields of a plan from its lines: seconds, time per rein and gait, music.
func (p *Plan) Totals(lib *Library) {
	p.Seconds, p.Reins, p.Gaits = 0, ReinSeconds{}, GaitSeconds{}
	for i := range p.Lines {
		l := &p.Lines[i]
		l.N = i + 1
		p.Seconds += l.Seconds
		o := lib.ByID[l.Option]
		if o == nil || o.Length == 0 {
			continue
		}
		moving := math.Max(0, l.Seconds-o.Fixed)
		p.Reins.Left += moving * o.Left / o.Length
		p.Reins.Right += moving * o.Right / o.Length
		inGait := l.Seconds
		switch o.Kind {
		case kindEinreiten, kindAbwenden, "rueckwaerts":
			inGait -= o.Fixed // the halt belongs to no gait
		case kindAusreiten:
			inGait = 0 // leaving on a long rein is no walk shown
		}
		switch l.Gait {
		case GaitWalk:
			p.Gaits.Walk += inGait
		case GaitTrot:
			p.Gaits.Trot += inGait
		case GaitCanter:
			p.Gaits.Canter += inGait
		}
	}
	p.Seconds = round1(p.Seconds)
	p.Reins = ReinSeconds{round1(p.Reins.Left), round1(p.Reins.Right)}
	p.Gaits = GaitSeconds{round1(p.Gaits.Walk), round1(p.Gaits.Trot), round1(p.Gaits.Canter)}
	p.Music = musicParts(p.Lines)
}

func boolf(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// musicParts groups consecutive lines of the same gait; the part boundaries are where the music
// changes.
func musicParts(lines []Line) []MusicPart {
	var parts []MusicPart
	for _, l := range lines {
		if n := len(parts); n > 0 && parts[n-1].Gait == l.Gait {
			parts[n-1].Seconds = round1(parts[n-1].Seconds + l.Seconds)
			parts[n-1].To = l.N
			continue
		}
		parts = append(parts, MusicPart{Gait: l.Gait, Seconds: l.Seconds, From: l.N, To: l.N, Cadence: Cadence[l.Gait]})
	}
	return parts
}

// Specs returns the specs of the plan's lines.
func (p Plan) Specs() []Spec {
	out := make([]Spec, len(p.Lines))
	for i, l := range p.Lines {
		out[i] = l.Spec()
	}
	return out
}

// MMSS formats seconds as m:ss.
func MMSS(sec float64) string {
	s := int(math.Round(sec))
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
