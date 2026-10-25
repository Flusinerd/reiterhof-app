package dressage

import (
	"fmt"
	"slices"
	"strings"
)

// ReqMovement is one required movement of a freestyle as the user enters it: the movement of the
// app's list, how often and on which rein. MinMeters is an optional distance for walk movements
// (a required walk that has to be shown continuously).
type ReqMovement struct {
	Key       string  `json:"key"`
	Count     int     `json:"count"`
	Rein      string  `json:"rein"` // "either" (default), "left", "right", "both"
	MinMeters float64 `json:"min_meters,omitempty"`
}

// Requirement is the set of required movements and the time window of a freestyle class. The app
// ships none: the user enters them from their own Aufgabenheft (docs/domains/dressage-tests.md).
type Requirement struct {
	Class      string        `json:"class"`
	Label      string        `json:"label"`
	Movements  []ReqMovement `json:"movements"`
	MinSeconds int           `json:"min_seconds"`
	MaxSeconds int           `json:"max_seconds"`
}

// ValidateRequirement returns the German reasons why a requirement set cannot be used; empty when it is fine.
func ValidateRequirement(r Requirement) []string {
	var out []string
	if ClassRank(r.Class) < 0 {
		out = append(out, "Die Klasse ist unbekannt.")
	}
	if len([]rune(r.Label)) > 80 {
		out = append(out, "Die Bezeichnung ist zu lang (höchstens 80 Zeichen).")
	}
	if len(r.Movements) > 40 {
		out = append(out, "Zu viele Pflichtlektionen (höchstens 40).")
	}
	seen := map[string]bool{}
	for _, m := range r.Movements {
		mv, ok := MovementByKey(m.Key)
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("Die Lektion „%s“ gibt es nicht.", m.Key))
			continue
		case m.Count < 1 || m.Count > 10:
			out = append(out, fmt.Sprintf("„%s“: Die Anzahl muss zwischen 1 und 10 liegen.", mv.Label))
		}
		if !slices.Contains([]string{"", "either", "left", "right", "both"}, m.Rein) {
			out = append(out, fmt.Sprintf("„%s“: Die Hand ist unbekannt.", mv.Label))
		}
		if m.MinMeters < 0 || m.MinMeters > 300 || (m.MinMeters > 0 && mv.Gait != GaitWalk) {
			out = append(out, fmt.Sprintf("„%s“: Eine Mindeststrecke gibt es nur für Schrittlektionen (bis 300 m).", mv.Label))
		}
		id := m.Key + "/" + m.Rein
		if seen[id] {
			out = append(out, fmt.Sprintf("„%s“ steht doppelt in der Liste.", mv.Label))
		}
		seen[id] = true
	}
	if r.MinSeconds != 0 || r.MaxSeconds != 0 {
		if r.MinSeconds < 30 || r.MaxSeconds <= r.MinSeconds || r.MaxSeconds > 900 {
			out = append(out, "Die Zeit muss zwischen 30 Sekunden und 15 Minuten liegen, die längste größer als die kürzeste.")
		}
	}
	return out
}

// walkMeters is the longest continuous walk the user requires.
func (r *Requirement) walkMeters() float64 {
	if r == nil {
		return 0
	}
	m := 0.0
	for _, rm := range r.Movements {
		if mv, ok := MovementByKey(rm.Key); ok && mv.Gait == GaitWalk && rm.MinMeters > m {
			m = rm.MinMeters
		}
	}
	return m
}

// walkSeconds is the time such a walk needs at the speed of the medium walk, with a margin.
func (r *Requirement) walkSeconds() float64 {
	w := r.walkMeters()
	if w == 0 {
		return 0
	}
	return w/Speed(GaitWalk, TempoMedium)*1.3 + 10
}

// needsCanterChange reports whether a movement is required that is ridden with a change of lead.
func (r *Requirement) needsCanterChange() bool {
	if r == nil {
		return false
	}
	for _, rm := range r.Movements {
		if rm.Key == "einfacher_galoppwechsel" || rm.Key == "aussengalopp" {
			return true
		}
	}
	return false
}

func (r Requirement) requiredKeys() map[string]bool {
	out := map[string]bool{}
	for _, m := range r.Movements {
		out[m.Key] = true
	}
	return out
}

// Window returns the time window in seconds; ok is false when the user entered none.
func (r Requirement) Window() (lo, hi float64, ok bool) {
	if r.MinSeconds == 0 && r.MaxSeconds == 0 {
		return 0, 0, false
	}
	return float64(r.MinSeconds), float64(r.MaxSeconds), true
}

// FreestyleParams are the inputs of GenerateFreestyle.
type FreestyleParams struct {
	Class       string
	Arena       string
	Seed        int64
	Requirement Requirement
}

// DefaultFreestyleSeconds is the target duration when the user entered no time window (our own value).
const DefaultFreestyleSeconds = 240.0

// GenerateFreestyle builds a freestyle that shows the user's required movements in the window of
// the user's requirement set, with the choreography criteria of the Kür leaflet as scoring.
func GenerateFreestyle(p FreestyleParams) (Plan, error) {
	if ClassRank(p.Class) < 0 {
		return Plan{}, fmt.Errorf("dressage: unknown class %q", p.Class)
	}
	lib, ok := LibraryFor(p.Arena)
	if !ok {
		return Plan{}, fmt.Errorf("dressage: unknown arena %q", p.Arena)
	}
	if errs := ValidateRequirement(p.Requirement); len(errs) > 0 {
		return Plan{}, fmt.Errorf("dressage: %s", strings.Join(errs, " "))
	}
	req := p.Requirement
	req.Class = p.Class
	lo, hi := DefaultFreestyleSeconds-30, DefaultFreestyleSeconds+30
	if l, h, ok := req.Window(); ok {
		lo, hi = l, h
	}
	g := &generator{class: p.Class, lib: lib, seed: p.Seed, lo: lo, hi: hi, mid: (lo + hi) / 2, req: &req}
	return g.run()
}
