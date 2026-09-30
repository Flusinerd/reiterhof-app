package weekplan

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
)

// Caps of the model's texts.
const (
	MaxReasonRunes = 140
	MaxFocusRunes  = 60
)

// dayCodes are the keys the model uses for the days of the week, Monday first.
var dayCodes = []string{"mo", "di", "mi", "do", "fr", "sa", "so"}

func dayCode(t time.Time) string { return dayCodes[(int(t.Weekday())+6)%7] }

// exerciseKey is the short key of a candidate exercise in the prompt ("E1", "E2", ...), so the
// model never sees database ids.
func exerciseKey(i int) string { return fmt.Sprintf("E%d", i+1) }

// SystemPrompt is the fixed instruction for the model (JAN-92). The owner chose the basis:
// the German training scale (Skala der Ausbildung) for the content of the units and general
// training theory (load and recovery, variety) for the structure of the week. The hard limits
// are enforced again by recommend.Check; the prompt aims at plans that pass it and make sense.
var SystemPrompt = buildSystemPrompt()

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

var intensityWords = map[training.Intensity]string{
	training.IntensityLight: "light", training.IntensityMedium: "medium", training.IntensityIntense: "demanding",
}

func buildSystemPrompt() string {
	var acts strings.Builder
	for _, a := range training.AllActivities {
		lo, hi := recommend.MinutesRange(a)
		fmt.Fprintf(&acts, "- %s = %s: %s; %s; %d-%d minutes\n",
			a, a.GermanName(), activityHints[a], intensityWords[training.DefaultIntensity(a)], lo, hi)
	}
	return `You are an experienced riding instructor trained in the German FN system. You plan one training week for one horse at a private stable. The horse's welfare comes first: when in doubt, plan less.

INPUT (JSON)
- discipline: dressage, jumping, eventing, leisure, western or young_horse. level: beginner (basic training, class E-A), intermediate (A*-L*), advanced (L** and above); missing = unknown. horse_age_years; missing = unknown.
- status: fit, reha (rehabilitation after an injury) or pause (training break).
- rhythm: sessions_min-sessions_max sessions and rest_days_min-rest_days_max rest days per week (Monday to Sunday, done and planned days count), max_minutes per session (0 = no limit), rest_after_show.
- activities: the only activities allowed for this horse. "conditional": allowed under a condition the owner knows.
- history: sessions of the last 14 days. days_ago 1 = yesterday. load = minutes x intensity; a day above 60 was demanding. feel: how the horse felt (fresh, loose, tired, tense). exercise and rating: the library exercise practised and how it went (hard, better, solid).
- shows_in_days: competitions relative to today (0 = today, negative = past).
- today, tomorrow: rain, night_min_c (coldest temperature of the night) and, for today, the ground (dry, wet, muddy, frozen).
- exercises: candidates from the exercise library: key, library (dressage, jumping, groundwork, lunge), level, German title, tags (aims of the training scale), mastered (went solid before), next (key of the follow-up exercise).
- days: the week. Plan every day with "open": true, and only those. Closed days are context: planned = an activity someone already planned, rest = rest day, done = a session took place. reha = the unit the vet's rehabilitation plan allows that day.

ACTIVITIES (code = German name: meaning; intensity; sensible duration)
` + acts.String() + `
HOW TO PLAN
1. Week structure (load and recovery): alternate demanding, medium and light days. Never two demanding days in a row. After a demanding day, a tired or tense horse or a long break, plan something light or a rest day. No more than 3 working days in a row when rest_days_min is 1 or more; spread the rest days. Count done and planned days so that the week ends within sessions_min-sessions_max. The same activity at most twice in a row; use hacks, lunging or groundwork for variety where allowed.
2. Content (Skala der Ausbildung): Takt, Losgelassenheit, Anlehnung, Schwung, Geraderichtung, Versammlung, with Durchlaessigkeit as the aim. Build from the bottom: every ridden unit starts with Losgelassenheit. Young horses (under 6), beginner or unknown level, after a break, in reha or after a tense session: Takt and Losgelassenheit. Intermediate: add Anlehnung, Schwung and Durchlaessigkeit. Advanced and fit: Geraderichtung and Versammlung, at most twice a week, never on consecutive days.
3. Discipline: dressage - mostly hall or arena, one hack or lunge for relaxation. jumping - jumping at most twice a week and never on consecutive days, gymnastic flatwork and hacks in between. eventing - flatwork, jumping (at most twice) and hacks for fitness. leisure - hacks and varied, relaxed work, little drill. western - arena work and hacks. young_horse - short units, groundwork, lunging and calm hacks, many breaks.
4. Age: under 6 short units; over 18 long walk phases and fewer demanding days.
5. Shows: 2-3 days before at most one last demanding unit; the day before only light work; after a show a rest day when rest_after_show is true.
6. Weather: rain today or tomorrow - prefer the hall to arena and hack. Wet, muddy or frozen ground - no jumping or arena. night_min_c below 0 - frozen ground in the morning.
7. Reha: on a reha day take exactly the reha activity and its minutes, focus Takt and Losgelassenheit. Pause: only walker, groundwork or short lunging.

FOCUS AND EXERCISE
- focus: what the unit is about, 2 to 5 German words, e.g. "Übergänge Schritt-Trab", "Dehnungshaltung", "Takt im Arbeitstrab", "Kondition im Gelände", "Gelassenheit an der Hand". Empty for rest. It must fit the training scale step you chose.
- exercise: the key of one candidate whose library fits the activity: hall or arena - dressage (jumping for jumping horses), jumping - jumping, lunge - lunge, groundwork - groundwork. hack, walker and rest - null. Prefer the next step after a mastered exercise; repeat an exercise rated hard; match its tags to the focus; stay at or below the horse's level (one level above only when the horse is fit and the easier exercises are mastered). Never invent keys; use null when nothing fits.

REASON
- One short, correct German sentence, at most 100 characters, shown to the owner next to the day.
- About the horse in the third person ("das Pferd", or no subject). Never address the horse or the reader: no "du", "dich", "dir", "dein".
- Use the German activity names from the list above.
- Name the concrete cause from the input: the previous days, variety, a show, the weather, the reha plan, the exercise rating. No general phrases like "um die Muskeln zu stärken".
- "heute" or "morgen" only when in_days is 0 or 1.
Good reasons: "Nach dem Ausritt gestern lockere Arbeit an der Longe." "Zwei Tage vor dem Turnier noch einmal konzentriert in der Halle." "Die Übergänge waren am Montag schwer, daher noch einmal ruhig üben." "Nach drei Arbeitstagen braucht das Pferd einen Ruhetag."

CHECK BEFORE ANSWERING
Every open day exactly once and no closed day; only allowed activities or rest; minutes within the sensible range and max_minutes; sessions and rest days of the whole week within the rhythm; no two demanding days in a row; every exercise key exists and fits the activity; reasons follow the rules.

ANSWER
JSON only, no other text:
{"days":[{"day":"<day code from the input>","activity":"<activity code or rest>","minutes":<integer, 0 for rest>,"focus":"<focus or empty>","exercise":"<key or null>","reason":"<reason>"}]}`
}

var ratingWords = map[int]string{1: "hard", 2: "better", 3: "solid"}

// Prompt returns the user message for the model. It holds only what the plan needs, and
// nothing that identifies a person or the horse: no names, no ids, no free text (notes,
// conditions, the level text, show and reha phase names, own-stable exercises) and no dates
// (days are relative to today). The level goes out only as a library level, the age in years.
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
		Exercise    string            `json:"exercise,omitempty"`
		Rating      string            `json:"rating,omitempty"`
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
	type weatherIn struct {
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
	type exerciseIn struct {
		Key      string   `json:"key"`
		Library  string   `json:"library"`
		Level    string   `json:"level,omitempty"`
		Title    string   `json:"title"`
		Tags     []string `json:"tags,omitempty"`
		Mastered bool     `json:"mastered,omitempty"`
		Next     string   `json:"next,omitempty"`
	}
	msg := struct {
		Discipline  string       `json:"discipline,omitempty"`
		Level       string       `json:"level,omitempty"`
		AgeYears    int          `json:"horse_age_years,omitempty"`
		Status      string       `json:"status"`
		Rhythm      rhythmIn     `json:"rhythm"`
		Activities  []activityIn `json:"activities"`
		History     []sessionIn  `json:"history"`
		ShowsInDays []int        `json:"shows_in_days"`
		Today       weatherIn    `json:"today"`
		Tomorrow    weatherIn    `json:"tomorrow"`
		Exercises   []exerciseIn `json:"exercises"`
		Days        []dayIn      `json:"days"`
	}{
		Discipline: in.Profile.Discipline, Level: in.Level, AgeYears: in.AgeYears, Status: string(in.Profile.Status),
		Activities: []activityIn{}, History: []sessionIn{}, ShowsInDays: []int{}, Exercises: []exerciseIn{}, Days: []dayIn{},
	}
	r := in.Profile.Rhythm
	msg.Rhythm = rhythmIn{r.SessionsMin, r.SessionsMax, r.RestDaysMin, r.RestDaysMax, r.MaxMinutes, r.RestAfterShow}
	for _, a := range in.Profile.Allowed {
		if a.Activity.Valid() && (a.Mode == training.ModeOn || a.Mode == training.ModeConditional) {
			msg.Activities = append(msg.Activities, activityIn{a.Activity, a.Mode == training.ModeConditional})
		}
	}
	keys := map[string]string{} // exercise id -> key
	for i, ex := range in.Exercises {
		keys[ex.ID] = exerciseKey(i)
	}
	for i, ex := range in.Exercises {
		msg.Exercises = append(msg.Exercises, exerciseIn{
			Key: exerciseKey(i), Library: ex.Library, Level: ex.Level, Title: ex.Title,
			Tags: ex.Tags, Mastered: ex.Mastered, Next: keys[ex.NextID],
		})
	}
	done := map[string]bool{}
	for _, s := range in.Recent {
		ago := training.DaysBetween(s.Day, today)
		done[training.Day(s.Day).Format(dateLayout)] = true
		if ago < 0 || ago >= recommend.RecentDays || !s.Activity.Valid() {
			continue
		}
		si := sessionIn{
			DaysAgo: ago, Activity: s.Activity, Minutes: s.Minutes, Load: round1(s.Load),
			CanterShare: round1(s.CanterShare), Feel: feel(s.Feel),
		}
		if k, ok := keys[s.ExerciseID]; ok {
			si.Exercise, si.Rating = k, ratingWords[s.FocusRating]
		}
		msg.History = append(msg.History, si)
	}
	for _, s := range in.Profile.Shows {
		if d := training.DaysBetween(today, s.Date); d >= -recommend.RecentDays && d <= 21 {
			msg.ShowsInDays = append(msg.ShowsInDays, d)
		}
	}
	weather := func(w *recommend.Weather) weatherIn {
		if w == nil {
			return weatherIn{}
		}
		rain, temp := w.Rain, math.Round(w.TempC)
		return weatherIn{Rain: &rain, NightMinC: &temp}
	}
	msg.Today, msg.Tomorrow = weather(in.Weather), weather(in.Tomorrow)
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
// (the rules plan those days), unknown exercise keys become no exercise; an answer that is
// not the expected JSON is an error.
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
			Focus    json.RawMessage `json:"focus"`
			Exercise json.RawMessage `json:"exercise"`
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
	byKey := map[string]*Exercise{}
	for i := range in.Exercises {
		byKey[exerciseKey(i)] = &in.Exercises[i]
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
		var reason, focus, key string
		_ = json.Unmarshal(d.Reason, &reason)
		_ = json.Unmarshal(d.Focus, &focus)
		_ = json.Unmarshal(d.Exercise, &key)
		out[date.Format(dateLayout)] = Proposal{
			Activity: act, Minutes: minutes, Reason: cleanText(reason, MaxReasonRunes),
			Focus: cleanText(focus, MaxFocusRunes), Exercise: byKey[strings.ToUpper(strings.TrimSpace(key))],
		}
	}
	return out, nil
}

// cleanText makes the model's text safe to show: one line, no control characters, capped.
func cleanText(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == ' ' || r == ' ' {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		s = strings.TrimSpace(string(r[:max-1])) + "…"
	}
	return s
}
