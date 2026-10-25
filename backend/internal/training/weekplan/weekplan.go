// Package weekplan plans the open days of a training week (JAN-89, JAN-92).
//
// A language model may propose per open day an activity, a duration, a focus (a catalog key
// and a few German words) and an exercise for each part of the unit (Prompt, Parse; the system
// prompt is SystemPrompt). recommend.Check tests every proposal against the rules and the
// rule-based recommender fills every day the model left out or got wrong (Merge); the unit of
// a day, its focus and the exercises of its parts, is planned by recommend.PlanUnit, which takes
// the model's proposal when it is allowed. Without a model the plan comes from the rules alone
// (Rules).
//
// Days are planned in date order. Each planned unit is added to the history as a simulated
// session with its focus and exercises, so variety, recent load, the weekly maximum, the focus
// rotation and the exercise pick of the later days see it.
//
// The package is pure like training/recommend: no HTTP, no database, no clock.
package weekplan

import (
	"slices"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/load"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
)

// Sources of a plan entry.
const (
	SourceAI    = "ai"
	SourceRules = "rules"
)

const dateLayout = "2006-01-02"

// Texts of a day whose activity the owner rejected (German, shown in the app).
const (
	replacedRejected = "Vom Besitzer abgelehnt."
	reasonNoOther    = "Keine andere Aktivität passt heute."
)

// Day is one day of the week.
type Day struct {
	Date time.Time // UTC-normalised calendar date
	// Open days get a plan entry; closed days (done, claimed with an activity, rest, past)
	// are context only.
	Open bool
	// Planned is the activity someone already planned for a closed day that has no session
	// yet; it counts as a session for the following days.
	Planned training.Activity
	// Minutes is the planned duration of a closed day's Planned activity; 0 = the activity's
	// default duration.
	Minutes int
	// Exclude lists activities the owner rejected for an open day; neither the model's
	// proposal nor the rules may pick them.
	Exclude []training.Activity
	// Rest marks a planned or mandatory rest day.
	Rest bool
	// Reha is the unit the active reha plan allows that day, nil without one.
	Reha *recommend.RehaPhase
	// EquipmentOK is the day's override of the set-up of the location: true when someone can set
	// it up, false when not, nil for the location's default.
	EquipmentOK *bool
	// RiderFocuses are the focuses (catalog keys, at most two) of the person who claimed an open
	// day and RiderName their first name for the reasons; they replace nothing, the rider's focuses
	// come first in the rotation of that day.
	RiderFocuses []string
	RiderName    string
	// FocusKey and Exercises (ids, main first) are what was planned for a closed day's Planned
	// unit; they count for the rotation and the exercise pick of the following days.
	FocusKey  string
	Exercises []string
}

// Exercise is a candidate from the exercise library (the pure view of a row, see
// training.Exercise).
type Exercise = training.Exercise

// Input is everything the planner needs.
type Input struct {
	Today   time.Time // stable-local calendar day
	Profile training.Profile
	// AgeYears is the horse's age (0 = unknown); Level its training level as
	// training.LevelClass gives it ("" = unknown).
	AgeYears int
	Level    string
	// Recent are the logged sessions of the 14 days before the first day up to today.
	Recent []training.Session
	// Weather and Ground are today's, Tomorrow is tomorrow's forecast; nil when unknown.
	Weather  *recommend.Weather
	Tomorrow *recommend.Weather
	Ground   recommend.Ground
	// Exercises are the candidates of the libraries that fit the allowed activities, in
	// progression order within each library, with the stable's own rows (Exercise.Own), which the
	// rules use and the model never sees.
	Exercises []Exercise
	// Block is the place of the week in the four-week blocks (the zero value: no blocks),
	// RestartFrom the first day of the return to work after a break (nil: not returning) and
	// Equipment the facility equipment of the stable's locations.
	Block       recommend.Block
	RestartFrom *time.Time
	Equipment   training.StableEquipment
	Days        []Day // in date order
}

// Proposal is one day of the model's answer.
type Proposal struct {
	Activity training.Activity
	Minutes  int
	// Focus is the content of the unit in a few German words, FocusKey the catalog focus ("" when
	// none or unknown).
	Focus    string
	FocusKey string
	// Warmup, Main and Cooldown are the exercises of the parts (nil when none or unknown).
	Warmup, Main, Cooldown *Exercise
	Reason                 string
}

// Entry is the plan of one open day.
type Entry struct {
	Date           time.Time
	Recommendation recommend.Recommendation
	// Focus is the content of the unit in a few German words (the model's; empty for rules).
	Focus string
	// Exercise is the exercise older clients show for the day: the main exercise of the unit,
	// nil when the unit has none.
	Exercise *Exercise
	// Unit is the plan of the unit: its focus (key and reason), the three parts with their
	// minutes and exercises and the note about the set-up. It is empty for a rest day and for the
	// activities without a unit plan (walker).
	Unit recommend.Unit
	// Source is SourceAI for an accepted proposal of the model, else SourceRules.
	Source string
	// Replaced says in German why the model's proposal for this day, or an exercise of it, was
	// replaced.
	Replaced string
	// Level is the unit's level by its load (recommend.UnitLevel), "" for a rest day.
	Level string
}

// Rules plans every open day with the rule-based recommender.
func Rules(in Input) []Entry { return Merge(in, nil) }

// Merge plans the open days: a proposal (keyed by date, YYYY-MM-DD) that passes
// recommend.Check is taken with its minutes fitted to the limits; a failing or missing one is
// replaced by the rules. The unit of every day that has one is planned by recommend.PlanUnit:
// the model's focus and exercises are taken when the rules allow them (Entry.Replaced says why an
// exercise was not), everything else the rules pick.
func Merge(in Input, proposals map[string]Proposal) []Entry {
	today := training.Day(in.Today)
	history := append([]training.Session(nil), in.Recent...)
	var out []Entry
	for i, d := range in.Days {
		day := training.Day(d.Date)
		if !d.Open {
			if d.Planned.Valid() && !d.Rest {
				history = append(history, simulated(day, d.Planned, d.Minutes, d.FocusKey, d.Exercises))
			}
			continue
		}
		rin := in.recommendInput(today, day, history, d, in.Days[i:])
		e := Entry{Date: day, Source: SourceRules}
		p, proposed := proposals[day.Format(dateLayout)]
		if proposed && slices.Contains(d.Exclude, p.Activity) {
			proposed, e.Replaced = false, replacedRejected
		}
		var hints recommend.Hints
		if proposed {
			v := recommend.Check(rin, p.Activity, p.Minutes)
			if v.OK {
				e.Recommendation, e.Source = v.Recommendation, SourceAI
				e.Recommendation.Reason = p.Reason
				if e.Recommendation.Reason == "" {
					e.Recommendation.Reason = "Vorschlag der KI."
				}
				if p.Activity != training.ActivityRest {
					e.Focus = p.Focus
					hints = recommend.Hints{FocusKey: p.FocusKey, Warmup: p.Warmup, Main: p.Main, Cooldown: p.Cooldown}
				}
			} else {
				e.Recommendation, e.Replaced = v.Recommendation, v.Why
				if slices.Contains(d.Exclude, e.Recommendation.Activity) {
					e.Recommendation = firstAllowed(rin, d.Exclude)
				}
			}
		} else {
			e.Recommendation = firstAllowed(rin, d.Exclude)
		}
		// A rule-based unit can still exceed the weekly maximum (the recommender only
		// prefers a rest day then); Check turns it into one.
		if e.Source == SourceRules && e.Recommendation.Activity != training.ActivityRest {
			if v := recommend.Check(rin, e.Recommendation.Activity, e.Recommendation.Minutes); !v.OK {
				e.Recommendation = v.Recommendation
				if slices.Contains(d.Exclude, e.Recommendation.Activity) {
					e.Recommendation = noOtherActivity(rin)
				}
			}
		}
		e.Recommendation.Score = 0
		if a := e.Recommendation.Activity; a != training.ActivityRest {
			e.Unit = recommend.PlanUnit(rin, a, e.Recommendation.Minutes, hints)
			if ex := e.Unit.Primary(); ex != nil {
				e.Exercise = ex
			}
			if e.Unit.Replaced != "" {
				e.Replaced = strings.TrimSpace(e.Replaced + " " + e.Unit.Replaced)
			}
			history = append(history, simulated(day, a, e.Recommendation.Minutes, e.Unit.FocusKey, e.Unit.Exercises()))
			e.Level = recommend.UnitLevel(recommend.UnitLoad(a, e.Recommendation.Minutes))
		}
		out = append(out, e)
	}
	return out
}

// recommendInput is the input of the recommender for an open day: the history so far, what the
// day's claimant brings (focuses) and the stable's set-up.
func (in Input) recommendInput(today, day time.Time, history []training.Session, d Day, rest []Day) recommend.Input {
	rin := recommend.Input{
		Today: day, Profile: in.Profile, Recent: history, Role: recommend.RoleOwner, Reha: d.Reha, Ahead: ahead(rest),
		AgeYears: in.AgeYears, Level: in.Level, Block: in.Block, RestartFrom: in.RestartFrom,
		Exercises: in.Exercises, RiderFocuses: d.RiderFocuses, RiderName: d.RiderName,
		Equipment: in.Equipment, EquipmentOK: d.EquipmentOK,
	}
	switch training.DaysBetween(today, day) {
	case 0:
		rin.Weather, rin.Ground = in.Weather, in.Ground
	case 1:
		rin.Weather = in.Tomorrow
	}
	return rin
}

// firstAllowed is the best recommendation whose activity is not excluded; without one it is a
// rest day.
func firstAllowed(rin recommend.Input, exclude []training.Activity) recommend.Recommendation {
	for _, r := range recommend.Recommend(rin).Recommendations {
		if !slices.Contains(exclude, r.Activity) {
			return r
		}
	}
	return noOtherActivity(rin)
}

// noOtherActivity is the rest day for a day on which every recommendation was rejected.
func noOtherActivity(rin recommend.Input) recommend.Recommendation {
	r := recommend.Check(rin, training.ActivityRest, 0).Recommendation
	r.Reason = reasonNoOther
	return r
}

// ahead describes the week from days[0] (the day being planned) on: the open days including
// it, the units and rest days fixed on the later days.
func ahead(days []Day) *recommend.Ahead {
	a := &recommend.Ahead{}
	for i, d := range days {
		switch {
		case d.Open:
			a.OpenDays++
		case i == 0:
		case d.Rest:
			a.RestDays++
		case d.Planned.Valid():
			a.Units = append(a.Units, simulated(training.Day(d.Date), d.Planned, d.Minutes, d.FocusKey, d.Exercises))
		}
	}
	return a
}

// simulated is the session a planned unit counts as for the days after it. exercises are the ids of
// the unit's exercises, the main one first.
func simulated(day time.Time, a training.Activity, minutes int, focusKey string, exercises []string) training.Session {
	if minutes <= 0 {
		minutes = recommend.DefaultMinutes(a)
	}
	s := training.Session{Day: day, Activity: a, Minutes: minutes, Load: load.Score(minutes, a, 0), FocusKey: focusKey}
	if len(exercises) > 0 {
		s.ExerciseID, s.Exercises = exercises[0], exercises
	}
	return s
}

// OpenDays reports whether any day needs a plan.
func (in Input) OpenDays() bool {
	for _, d := range in.Days {
		if d.Open {
			return true
		}
	}
	return false
}
