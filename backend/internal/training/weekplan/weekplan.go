// Package weekplan plans the open days of a training week (JAN-89, JAN-92).
//
// A language model may propose per open day an activity, a duration, a focus and an exercise
// of the library (Prompt, Parse; the system prompt is SystemPrompt). recommend.Check tests
// every proposal against the rules and the rule-based recommender fills every day the model
// left out or got wrong (Merge). Without a model the plan comes from the rules alone (Rules).
//
// Days are planned in date order. Each planned unit is added to the history as a simulated
// session, so variety, recent load and the weekly maximum of the later days see it.
//
// The package is pure like training/recommend: no HTTP, no database, no clock.
package weekplan

import (
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

// Day is one day of the week.
type Day struct {
	Date time.Time // UTC-normalised calendar date
	// Open days get a plan entry; closed days (done, claimed with an activity, rest, past)
	// are context only.
	Open bool
	// Planned is the activity someone already planned for a closed day that has no session
	// yet; it counts as a session for the following days.
	Planned training.Activity
	// Rest marks a planned or mandatory rest day.
	Rest bool
	// Reha is the unit the active reha plan allows that day, nil without one.
	Reha *recommend.RehaPhase
}

// Exercise is a candidate from the global exercise library.
type Exercise struct {
	ID       string
	Library  string // dressage, jumping, groundwork, lunge (training.ExerciseLibrary)
	Level    string // training.LevelBeginner, ...
	Title    string
	Tags     []string // aims of the training scale, e.g. takt, losgelassenheit
	Mastered bool     // a session with focus rating 3 ("Sitzt")
	NextID   string   // follow-up exercise, "" for none
}

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
	// progression order within each library.
	Exercises []Exercise
	Days      []Day // in date order
}

// Proposal is one day of the model's answer.
type Proposal struct {
	Activity training.Activity
	Minutes  int
	Focus    string
	Exercise *Exercise // nil when none or unknown
	Reason   string
}

// Entry is the plan of one open day.
type Entry struct {
	Date           time.Time
	Recommendation recommend.Recommendation
	// Focus is the content of the unit in a few German words (the model's; empty for rules).
	Focus string
	// Exercise is the library exercise for the unit, nil when the activity has none.
	Exercise *Exercise
	// Source is SourceAI for an accepted proposal of the model, else SourceRules.
	Source string
	// Replaced says in German why the model's proposal for this day was replaced.
	Replaced string
}

// Rules plans every open day with the rule-based recommender.
func Rules(in Input) []Entry { return Merge(in, nil) }

// Merge plans the open days: a proposal (keyed by date, YYYY-MM-DD) that passes
// recommend.Check is taken with its minutes fitted to the limits; a failing or missing one is
// replaced by the rules. A proposed exercise stays only when its library fits the activity;
// rule-based units get the first exercise of their library the horse has not mastered.
func Merge(in Input, proposals map[string]Proposal) []Entry {
	today := training.Day(in.Today)
	history := append([]training.Session(nil), in.Recent...)
	var out []Entry
	for _, d := range in.Days {
		day := training.Day(d.Date)
		if !d.Open {
			if d.Planned.Valid() && !d.Rest {
				history = append(history, simulated(day, d.Planned, 0))
			}
			continue
		}
		rin := recommend.Input{Today: day, Profile: in.Profile, Recent: history, Role: recommend.RoleOwner, Reha: d.Reha}
		switch training.DaysBetween(today, day) {
		case 0:
			rin.Weather, rin.Ground = in.Weather, in.Ground
		case 1:
			rin.Weather = in.Tomorrow
		}
		e := Entry{Date: day, Source: SourceRules}
		if p, ok := proposals[day.Format(dateLayout)]; ok {
			v := recommend.Check(rin, p.Activity, p.Minutes)
			if v.OK {
				e.Recommendation, e.Source = v.Recommendation, SourceAI
				e.Recommendation.Reason = p.Reason
				if e.Recommendation.Reason == "" {
					e.Recommendation.Reason = "Vorschlag der KI."
				}
				if p.Activity != training.ActivityRest {
					e.Focus = p.Focus
					if ex := p.Exercise; ex != nil && ex.Library == training.ExerciseLibrary(p.Activity, in.Profile.Discipline) {
						e.Exercise = ex
					}
				}
			} else {
				e.Recommendation, e.Replaced = v.Recommendation, v.Why
			}
		} else {
			e.Recommendation = recommend.Recommend(rin).Recommendations[0]
		}
		// A rule-based unit can still exceed the weekly maximum (the recommender only
		// prefers a rest day then); Check turns it into one.
		if e.Source == SourceRules && e.Recommendation.Activity != training.ActivityRest {
			if v := recommend.Check(rin, e.Recommendation.Activity, e.Recommendation.Minutes); !v.OK {
				e.Recommendation = v.Recommendation
			}
		}
		if e.Source == SourceRules {
			e.Exercise = in.firstOpenExercise(training.ExerciseLibrary(e.Recommendation.Activity, in.Profile.Discipline))
		}
		e.Recommendation.Score = 0
		if a := e.Recommendation.Activity; a != training.ActivityRest {
			history = append(history, simulated(day, a, e.Recommendation.Minutes))
		}
		out = append(out, e)
	}
	return out
}

// firstOpenExercise is the first exercise of a library the horse has not mastered yet (the
// last one when all are mastered), like "Was heute?" suggests it.
func (in Input) firstOpenExercise(library string) *Exercise {
	if library == "" {
		return nil
	}
	var last *Exercise
	for i := range in.Exercises {
		ex := &in.Exercises[i]
		if ex.Library != library {
			continue
		}
		if !ex.Mastered {
			return ex
		}
		last = ex
	}
	return last
}

func simulated(day time.Time, a training.Activity, minutes int) training.Session {
	if minutes <= 0 {
		minutes = recommend.DefaultMinutes(a)
	}
	return training.Session{Day: day, Activity: a, Minutes: minutes, Load: load.Score(minutes, a, 0)}
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
