// Package weekplan plans the open days of a training week (JAN-89).
//
// A language model may propose an activity and a duration per open day (Prompt, Parse);
// recommend.Check tests every proposal against the rules and the rule-based recommender
// fills every day the model left out or got wrong (Merge). Without a model the plan comes
// from the rules alone (Rules).
//
// Days are planned in date order. Each planned unit is added to the history as a simulated
// session, so variety, recent load and the weekly maximum of the later days see it.
//
// The package is pure like training/recommend: no HTTP, no database, no clock.
package weekplan

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/load"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
)

// Sources of a plan entry.
const (
	SourceAI    = "ai"
	SourceRules = "rules"
)

// MaxReasonRunes caps the model's reason text.
const MaxReasonRunes = 140

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

// Input is everything the planner needs.
type Input struct {
	Today   time.Time // stable-local calendar day
	Profile training.Profile
	// Recent are the logged sessions of the 14 days before the first day up to today.
	Recent []training.Session
	// Weather and Ground are today's; they apply to today only.
	Weather *recommend.Weather
	Ground  recommend.Ground
	Days    []Day // in date order
}

// Proposal is one day of the model's answer.
type Proposal struct {
	Activity training.Activity
	Minutes  int
	Reason   string
}

// Entry is the plan of one open day.
type Entry struct {
	Date           time.Time
	Recommendation recommend.Recommendation
	// Source is SourceAI for an accepted proposal of the model, else SourceRules.
	Source string
	// Replaced says in German why the model's proposal for this day was replaced.
	Replaced string
}

// Rules plans every open day with the rule-based recommender.
func Rules(in Input) []Entry { return Merge(in, nil) }

// Merge plans the open days: a proposal (keyed by date, YYYY-MM-DD) that passes
// recommend.Check is taken with its minutes fitted to the limits; a failing or missing one is
// replaced by the rules.
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
		if day.Equal(today) {
			rin.Weather, rin.Ground = in.Weather, in.Ground
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
		e.Recommendation.Score = 0
		if a := e.Recommendation.Activity; a != training.ActivityRest {
			history = append(history, simulated(day, a, e.Recommendation.Minutes))
		}
		out = append(out, e)
	}
	return out
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

// dayCodes are the keys the model uses for the days of the week, Monday first.
var dayCodes = []string{"mo", "di", "mi", "do", "fr", "sa", "so"}

func dayCode(t time.Time) string { return dayCodes[(int(t.Weekday())+6)%7] }

// SystemPrompt is the fixed instruction for the model. It names the German words and the
// sensible duration of every activity, because small models otherwise invent both.
var SystemPrompt = buildSystemPrompt()

func buildSystemPrompt() string {
	var acts strings.Builder
	for _, a := range training.AllActivities {
		lo, hi := recommend.MinutesRange(a)
		fmt.Fprintf(&acts, "- %s = %s (%s), %d-%d minutes\n", a, a.GermanName(), activityHints[a], lo, hi)
	}
	return `You plan the training week of one horse at a private riding stable. The reader is the horse's owner.
The user message is JSON:
- discipline, status (fit, reha = rehabilitation, pause = training break) and rhythm (sessions and rest days per week, max_minutes per session, rest_after_show).
- activities: the only activities allowed for this horse; "conditional": true means allowed with a condition the owner knows.
- history: sessions of the last days (days_ago 1 = yesterday) with load (minutes x intensity) and how the horse felt (fresh, loose, tired, tense).
- shows_in_days: competitions relative to today (0 = today, negative = past).
- today: weather and ground of today (only known for today).
- days: the days of this week. Plan every day with "open": true. Closed days are context: "planned" is an activity someone already planned, "rest" a rest day, "done" a day with a session. "reha" is the unit a vet's rehabilitation plan allows that day.

Activities (code = German name, meaning, sensible duration):
` + acts.String() + `
Plan a sensible week: vary the activities, alternate demanding and light days, respect the rhythm, keep the day before a show light and plan a rest day after a show when rest_after_show is true. After a tired or tense session or a demanding day, plan something light. In reha or pause only light work. On a reha day follow the reha unit exactly.
Use only the listed activities or "rest". Keep every duration within its sensible range and at most max_minutes (when above 0).

The reason is shown to the owner next to the day. Rules for the reason:
- One short, correct German sentence, at most 100 characters.
- It is about the horse in the third person ("das Pferd", or no subject at all). Never address the horse or the reader, never use "du", "dich", "dir", "dein".
- Use the German activity names from the list above.
- Give the concrete cause from the data: the previous days, variety, a show, the weather today, the reha plan. No general phrases like "um die Muskeln zu stärken".
- Do not use "heute" or "morgen" unless in_days is 0 or 1.
Good reasons: "Nach dem Ausritt gestern ist lockere Arbeit an der Longe gut." "Zwei Tage vor dem Turnier noch einmal konzentriert in der Halle." "Nach drei Arbeitstagen braucht das Pferd einen Ruhetag."

Answer with JSON only, no other text:
{"days":[{"day":"<day code from the input>","activity":"<activity code or rest>","minutes":<integer, 0 for rest>,"reason":"<the reason>"}]}`
}

// activityHints explain the activity codes to the model.
var activityHints = map[training.Activity]string{
	training.ActivityHall:       "riding in the indoor school",
	training.ActivityArena:      "riding in the outdoor arena",
	training.ActivityHack:       "hacking out in the countryside",
	training.ActivityLunge:      "lunging, the horse works on a circle around the person",
	training.ActivityJumping:    "jumping training",
	training.ActivityGroundwork: "groundwork, leading and exercises from the ground",
	training.ActivityWalker:     "horse walker, walk only",
}

// Prompt returns the user message for the model. It holds only what the plan needs, and
// nothing that identifies a person or the horse: no names, no ids, no free text (notes,
// conditions, the level, show and reha phase names) and no dates (days are relative to today).
func Prompt(in Input) string {
	today := training.Day(in.Today)
	type activityIn struct {
		Activity    training.Activity `json:"activity"`
		Conditional bool              `json:"conditional,omitempty"`
	}
	type sessionIn struct {
		DaysAgo     int               `json:"days_ago"`
		Activity    training.Activity `json:"activity"`
		Minutes     int               `json:"minutes"`
		Load        float64           `json:"load"`
		CanterShare float64           `json:"canter_share,omitempty"`
		Feel        string            `json:"feel,omitempty"`
	}
	type rehaIn struct {
		Activity   training.Activity `json:"activity"`
		MinMinutes int               `json:"min_minutes,omitempty"`
		MaxMinutes int               `json:"max_minutes,omitempty"`
	}
	type dayIn struct {
		Day     string            `json:"day"`
		InDays  int               `json:"in_days"`
		Open    bool              `json:"open"`
		Done    bool              `json:"done,omitempty"`
		Planned training.Activity `json:"planned,omitempty"`
		Rest    bool              `json:"rest,omitempty"`
		Reha    *rehaIn           `json:"reha,omitempty"`
	}
	type todayIn struct {
		Rain      *bool    `json:"rain,omitempty"`
		NightMinC *float64 `json:"night_min_c,omitempty"`
		Ground    string   `json:"ground,omitempty"`
	}
	type rhythmIn struct {
		SessionsMin   int  `json:"sessions_min"`
		SessionsMax   int  `json:"sessions_max"`
		RestDaysMin   int  `json:"rest_days_min"`
		RestDaysMax   int  `json:"rest_days_max"`
		MaxMinutes    int  `json:"max_minutes"`
		RestAfterShow bool `json:"rest_after_show"`
	}
	msg := struct {
		Discipline  string       `json:"discipline,omitempty"`
		Status      string       `json:"status"`
		Rhythm      rhythmIn     `json:"rhythm"`
		Activities  []activityIn `json:"activities"`
		History     []sessionIn  `json:"history"`
		ShowsInDays []int        `json:"shows_in_days"`
		Today       todayIn      `json:"today"`
		Days        []dayIn      `json:"days"`
	}{
		Discipline: in.Profile.Discipline, Status: string(in.Profile.Status),
		Activities: []activityIn{}, History: []sessionIn{}, ShowsInDays: []int{}, Days: []dayIn{},
	}
	r := in.Profile.Rhythm
	msg.Rhythm = rhythmIn{r.SessionsMin, r.SessionsMax, r.RestDaysMin, r.RestDaysMax, r.MaxMinutes, r.RestAfterShow}
	for _, a := range in.Profile.Allowed {
		if a.Activity.Valid() && (a.Mode == training.ModeOn || a.Mode == training.ModeConditional) {
			msg.Activities = append(msg.Activities, activityIn{a.Activity, a.Mode == training.ModeConditional})
		}
	}
	done := map[string]bool{}
	for _, s := range in.Recent {
		ago := training.DaysBetween(s.Day, today)
		done[training.Day(s.Day).Format(dateLayout)] = true
		if ago < 0 || ago >= recommend.RecentDays || !s.Activity.Valid() {
			continue
		}
		msg.History = append(msg.History, sessionIn{
			DaysAgo: ago, Activity: s.Activity, Minutes: s.Minutes, Load: round1(s.Load),
			CanterShare: round1(s.CanterShare), Feel: feel(s.Feel),
		})
	}
	for _, s := range in.Profile.Shows {
		if d := training.DaysBetween(today, s.Date); d >= -recommend.RecentDays && d <= 21 {
			msg.ShowsInDays = append(msg.ShowsInDays, d)
		}
	}
	if w := in.Weather; w != nil {
		rain, temp := w.Rain, math.Round(w.TempC)
		msg.Today.Rain, msg.Today.NightMinC = &rain, &temp
	}
	msg.Today.Ground = string(in.Ground)
	for _, d := range in.Days {
		day := training.Day(d.Date)
		di := dayIn{Day: dayCode(day), InDays: training.DaysBetween(today, day), Open: d.Open, Rest: d.Rest}
		if !d.Open {
			di.Done = done[day.Format(dateLayout)]
			if d.Planned.Valid() {
				di.Planned = d.Planned
			}
		}
		if p := d.Reha; p != nil {
			di.Reha = &rehaIn{Activity: p.Activity, MinMinutes: p.MinMinutes, MaxMinutes: p.MaxMinutes}
		}
		msg.Days = append(msg.Days, di)
	}
	b, _ := json.Marshal(msg)
	return string(b)
}

// feel passes only the known feel values (they are chosen from a list in the app).
func feel(s string) string {
	switch s {
	case "fresh", "loose", "tired", "tense":
		return s
	}
	return ""
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// Parse reads the model's answer and returns the proposals for the open days, keyed by
// date (YYYY-MM-DD). Entries for unknown or closed days and unknown activities are dropped
// (the rules plan those days); an answer that is not the expected JSON is an error.
func Parse(in Input, answer string) (map[string]Proposal, error) {
	answer = strings.TrimSpace(answer)
	// Tolerate a Markdown code fence around the JSON.
	if strings.HasPrefix(answer, "```") {
		answer = strings.TrimPrefix(strings.TrimPrefix(answer, "```json"), "```")
		answer = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(answer), "```"))
	}
	var raw struct {
		Days []struct {
			Day      string          `json:"day"`
			Activity string          `json:"activity"`
			Minutes  json.Number     `json:"minutes"`
			Reason   json.RawMessage `json:"reason"`
		} `json:"days"`
	}
	dec := json.NewDecoder(strings.NewReader(answer))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("weekplan: answer is not JSON: %w", err)
	}
	if raw.Days == nil {
		return nil, fmt.Errorf("weekplan: answer has no days")
	}
	open := map[string]time.Time{}
	for _, d := range in.Days {
		if d.Open {
			open[dayCode(d.Date)] = training.Day(d.Date)
		}
	}
	out := map[string]Proposal{}
	for _, d := range raw.Days {
		date, ok := open[strings.ToLower(strings.TrimSpace(d.Day))]
		if !ok {
			continue
		}
		act := training.Activity(strings.ToLower(strings.TrimSpace(d.Activity)))
		if act != training.ActivityRest && !act.Valid() {
			continue
		}
		minutes := 0
		if f, err := d.Minutes.Float64(); err == nil && f > 0 && f < 1000 {
			minutes = int(math.Round(f))
		}
		var reason string
		_ = json.Unmarshal(d.Reason, &reason)
		out[date.Format(dateLayout)] = Proposal{Activity: act, Minutes: minutes, Reason: cleanReason(reason)}
	}
	return out, nil
}

// cleanReason makes the model's text safe to show: one line, no control characters, capped.
func cleanReason(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == ' ' || r == ' ' {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > MaxReasonRunes {
		s = strings.TrimSpace(string(r[:MaxReasonRunes-1])) + "…"
	}
	return s
}
