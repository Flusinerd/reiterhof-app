// Package blanketplan evaluates a horse's blanket rules against a forecast.
//
// Semantics (see docs/architecture.md): rules are evaluated by Position, the
// first matching rule wins; a rule matches when TempMin <= night minimum <
// TempMax (a nil bound is open), Rain is nil or equals the forecast and, for
// a rain rule, RainMinMM <= rain amount < RainMaxMM (a nil bound is open).
// A rule with a nil BlanketID means "no blanket".
package blanketplan

// Rule is one row of blanket_rules.
type Rule struct {
	Position  int
	TempMin   *float64 // inclusive lower bound, nil = unbounded
	TempMax   *float64 // exclusive upper bound, nil = unbounded
	Rain      *bool    // nil = any weather
	RainMinMM *float64 // inclusive lower bound of the rain amount, only with Rain = true
	RainMaxMM *float64 // exclusive upper bound of the rain amount, only with Rain = true
	BlanketID *string  // nil = no blanket
	Note      string
}

// Forecast is the input of the recommendation: the night minimum, whether rain
// is expected and how much (sum over the cover window, see weather.Summary).
type Forecast struct {
	NightMinC float64
	WillRain  bool
	RainMM    float64
}

// Matches reports whether the rule applies to the forecast.
func (r Rule) Matches(f Forecast) bool {
	if r.TempMin != nil && f.NightMinC < *r.TempMin {
		return false
	}
	if r.TempMax != nil && f.NightMinC >= *r.TempMax {
		return false
	}
	if r.Rain != nil && *r.Rain != f.WillRain {
		return false
	}
	if r.RainMinMM != nil && f.RainMM < *r.RainMinMM {
		return false
	}
	if r.RainMaxMM != nil && f.RainMM >= *r.RainMaxMM {
		return false
	}
	return true
}

// Recommend returns the first matching rule ordered by Position. The input
// slice is not modified. ok is false when no rule matches ("keine Regel").
func Recommend(rules []Rule, f Forecast) (rule Rule, ok bool) {
	for _, r := range rules {
		if !r.Matches(f) {
			continue
		}
		if !ok || r.Position < rule.Position {
			rule, ok = r, true
		}
	}
	return rule, ok
}

// Changed reports whether the recommendation differs between two rules in a way
// the user cares about: another blanket (or none vs. some). A different note or
// rule position with the same blanket is not a change.
func Changed(prev, next Rule) bool {
	switch {
	case prev.BlanketID == nil && next.BlanketID == nil:
		return false
	case prev.BlanketID == nil || next.BlanketID == nil:
		return true
	default:
		return *prev.BlanketID != *next.BlanketID
	}
}
