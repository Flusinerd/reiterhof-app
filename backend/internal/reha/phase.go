// Package reha is the rehabilitation plan ("Reha-Plan", JAN-66, JAN-67, JAN-42): a vet-given
// sequence of phases that limits what a horse may do each day.
//
// This file is pure (no HTTP, no DB, no clock): phase parsing and validation, the date
// layout of a plan and the "allowed today" rule. Days are stable-local calendar dates
// normalised to midnight UTC (training.Day), so all arithmetic is by calendar day and a
// DST change never shifts a phase.
package reha

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// Limits of a plan.
const (
	MaxPhases      = 20
	MaxPhaseDays   = 365
	MaxTotalDays   = 730
	MaxMinutes     = 240
	maxNameLen     = 80
	maxConditions  = 500
	defaultMaxMins = 20 // legacy phases without max_minutes
)

// ActivityRest marks a phase without any exercise (box rest, "Boxenruhe"): 0 minutes.
const ActivityRest = string(training.ActivityRest)

// Phase is one step of a plan. It is also the stored form: reha_plans.phases is a JSON list of
// phases in this shape. Older data may carry weeks or duration_days instead of days; ParsePhases
// reads those, stored and returned phases always use days.
type Phase struct {
	Name       string `json:"name"`
	Days       int    `json:"days"`
	Activity   string `json:"activity"` // a training activity or "rest"
	MinMinutes int    `json:"min_minutes"`
	MaxMinutes int    `json:"max_minutes"`
	Conditions string `json:"conditions"`
}

// Rest reports whether the phase is a rest phase (no exercise).
func (p Phase) Rest() bool { return p.Activity == ActivityRest }

// ValidActivity reports whether a is a trainable activity or "rest".
func ValidActivity(a string) bool {
	return a == ActivityRest || training.Activity(a).Valid()
}

// ActivityLabel is the German label of a phase activity.
func ActivityLabel(a string) string { return training.Activity(a).GermanName() }

// storedPhase reads every accepted spelling of the duration.
type storedPhase struct {
	Phase
	Weeks        int `json:"weeks"`
	DurationDays int `json:"duration_days"`
}

// ParsePhases reads reha_plans.phases leniently (data written by hand or by older code):
// the duration is days, duration_days or weeks x 7, an empty name becomes "Phase N", missing
// or inconsistent minutes are repaired (max defaults to 20, min above max becomes max). It
// never validates the activity; use Phase.Active to see whether a phase can be used.
func ParsePhases(raw []byte) ([]Phase, error) {
	var in []storedPhase
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	out := make([]Phase, 0, len(in))
	for i, s := range in {
		p := s.Phase
		switch {
		case p.Days > 0:
		case s.DurationDays > 0:
			p.Days = s.DurationDays
		default:
			p.Days = s.Weeks * 7
		}
		if strings.TrimSpace(p.Name) == "" {
			p.Name = fmt.Sprintf("Phase %d", i+1)
		}
		if p.Rest() {
			p.MinMinutes, p.MaxMinutes = 0, 0
		} else {
			if p.MaxMinutes <= 0 {
				p.MaxMinutes = defaultMaxMins
			}
			if p.MinMinutes <= 0 || p.MinMinutes > p.MaxMinutes {
				p.MinMinutes = p.MaxMinutes
			}
		}
		out = append(out, p)
	}
	return out, nil
}

// Usable reports whether the phase has a length and a valid activity.
func (p Phase) Usable() bool { return p.Days > 0 && ValidActivity(p.Activity) }

// ValidatePhases checks phases from a client and returns their canonical form (trimmed
// texts, minutes 0 for rest phases). The error text is English and names the phase.
func ValidatePhases(in []Phase) ([]Phase, error) {
	if len(in) == 0 {
		return nil, errors.New("phases: at least one phase is required")
	}
	if len(in) > MaxPhases {
		return nil, fmt.Errorf("phases: at most %d phases", MaxPhases)
	}
	out := make([]Phase, 0, len(in))
	total := 0
	for i, p := range in {
		n := i + 1
		p.Name = strings.TrimSpace(p.Name)
		p.Conditions = strings.TrimSpace(p.Conditions)
		switch {
		case p.Name == "" || len([]rune(p.Name)) > maxNameLen:
			return nil, fmt.Errorf("phases[%d].name is required (max %d characters)", n, maxNameLen)
		case p.Days < 1 || p.Days > MaxPhaseDays:
			return nil, fmt.Errorf("phases[%d].days must be between 1 and %d", n, MaxPhaseDays)
		case !ValidActivity(p.Activity):
			return nil, fmt.Errorf("phases[%d].activity must be one of hall, arena, hack, lunge, jumping, groundwork, walker, rest", n)
		case len([]rune(p.Conditions)) > maxConditions:
			return nil, fmt.Errorf("phases[%d].conditions: max %d characters", n, maxConditions)
		}
		if p.Rest() {
			if p.MinMinutes != 0 || p.MaxMinutes != 0 {
				return nil, fmt.Errorf("phases[%d]: a rest phase has no minutes", n)
			}
		} else {
			switch {
			case p.MinMinutes < 1 || p.MaxMinutes > MaxMinutes:
				return nil, fmt.Errorf("phases[%d]: minutes must be between 1 and %d", n, MaxMinutes)
			case p.MinMinutes > p.MaxMinutes:
				return nil, fmt.Errorf("phases[%d]: min_minutes must not exceed max_minutes", n)
			}
		}
		total += p.Days
		out = append(out, p)
	}
	if total > MaxTotalDays {
		return nil, fmt.Errorf("phases: the plan may last at most %d days", MaxTotalDays)
	}
	return out, nil
}

// TotalDays is the summed length of all phases.
func TotalDays(phases []Phase) int {
	n := 0
	for _, p := range phases {
		n += p.Days
	}
	return n
}

// Slot is a phase placed on the calendar. Start and End are inclusive dates.
type Slot struct {
	Phase Phase
	Index int // 0-based
	Start time.Time
	End   time.Time
}

// Timeline places the phases one after the other, beginning on start.
func Timeline(start time.Time, phases []Phase) []Slot {
	day := training.Day(start)
	out := make([]Slot, 0, len(phases))
	for i, p := range phases {
		n := p.Days
		if n < 1 {
			n = 0
		}
		s := Slot{Phase: p, Index: i, Start: day, End: day.AddDate(0, 0, n-1)}
		out = append(out, s)
		day = day.AddDate(0, 0, n)
	}
	return out
}

// EndDate is the last day of the plan (inclusive); start minus one day for an empty plan.
func EndDate(start time.Time, phases []Phase) time.Time {
	return training.Day(start).AddDate(0, 0, TotalDays(phases)-1)
}

// Position is where a calendar day lies in a plan.
type Position struct {
	Slot        Slot
	DayInPhase  int // 0-based
	DayInPlan   int // 0-based
	Minutes     int // allowed minutes on this day (ramp), 0 for rest phases
	PhaseNumber int // 1-based, Slot.Index + 1
}

// Locate finds the phase that covers day. ok is false before the start, after the last
// phase and for a phase without length.
func Locate(start time.Time, phases []Phase, day time.Time) (Position, bool) {
	elapsed := training.DaysBetween(start, day)
	if elapsed < 0 {
		return Position{}, false
	}
	for _, s := range Timeline(start, phases) {
		if s.Phase.Days < 1 {
			continue
		}
		if d := training.DaysBetween(s.Start, day); d >= 0 && d < s.Phase.Days {
			return Position{
				Slot: s, DayInPhase: d, DayInPlan: elapsed, PhaseNumber: s.Index + 1,
				Minutes: Ramp(s.Phase.MinMinutes, s.Phase.MaxMinutes, d, s.Phase.Days),
			}, true
		}
	}
	return Position{}, false
}

// Ramp is the allowed minutes on day dayInPhase (0-based) of a phase of the given length:
// it rises linearly from min on the first day to max on the last, rounded half up. A phase of
// one day allows max.
func Ramp(minM, maxM, dayInPhase, days int) int {
	if days <= 1 || maxM <= minM {
		return maxM
	}
	if dayInPhase <= 0 {
		return minM
	}
	if dayInPhase >= days-1 {
		return maxM
	}
	steps := days - 1
	num := minM*steps + (maxM-minM)*dayInPhase
	return (2*num + steps) / (2 * steps)
}

// State of a plan relative to a day.
const (
	StateUpcoming = "upcoming" // before the start date
	StateRunning  = "running"
	StateFinished = "finished" // after the last phase (the plan is still active until ended)
)

// StateOn classifies the day against the plan.
func StateOn(start time.Time, phases []Phase, day time.Time) string {
	switch {
	case training.DaysBetween(start, day) < 0:
		return StateUpcoming
	case training.DaysBetween(start, day) >= TotalDays(phases):
		return StateFinished
	}
	return StateRunning
}

// Allowed is "Heute erlaubt": what the horse may do on one day.
type Allowed struct {
	Phase       Phase
	PhaseNumber int // 1-based
	Phases      int
	DayInPhase  int // 1-based
	Minutes     int // the ramp value of the day; 0 for rest
}

// AllowedOn returns the allowed unit of the day, or nil when no usable phase covers it.
func AllowedOn(start time.Time, phases []Phase, day time.Time) *Allowed {
	pos, ok := Locate(start, phases, day)
	if !ok || !pos.Slot.Phase.Usable() {
		return nil
	}
	return &Allowed{
		Phase: pos.Slot.Phase, PhaseNumber: pos.PhaseNumber, Phases: len(phases),
		DayInPhase: pos.DayInPhase + 1, Minutes: pos.Minutes,
	}
}

// RuleText is the one-line rule for helpers, e.g. "Reha: Schritt führen 14 min, nur Boden fest".
// Conditions are cut after 200 characters.
func (a Allowed) RuleText() string {
	if a.Phase.Rest() {
		s := "Reha: " + a.Phase.Name + " – heute keine Bewegung"
		if c := clip(a.Phase.Conditions, 200); c != "" {
			s += ", " + c
		}
		return s
	}
	s := fmt.Sprintf("Reha: %s %d min", a.Phase.Name, a.Minutes)
	if c := clip(a.Phase.Conditions, 200); c != "" {
		s += ", " + c
	}
	return s
}

// NoUnitText is the rule text of an active plan that has no unit on the day.
const NoUnitText = "Reha-Plan aktiv: für diesen Tag ist keine Einheit vorgesehen"

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
