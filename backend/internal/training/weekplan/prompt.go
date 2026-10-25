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

// visibleExercises returns the exercises the model may see: the global library. The stable's own
// rows are free text and stay on the server, they only take part in the rules' pick.
func (in Input) visibleExercises() []*Exercise {
	var out []*Exercise
	for i := range in.Exercises {
		if !in.Exercises[i].Own {
			out = append(out, &in.Exercises[i])
		}
	}
	return out
}

// SystemPrompt is the fixed instruction for the model (JAN-92, docs/domains/training-planning.md).
// The owner chose the basis: the German training scale (Skala der Ausbildung) for the content of
// the units, the vielseitige Grundausbildung for the mix of the week, training science (load and
// recovery, periodisation) for the load and the three parts of a riding lesson for the unit. The
// hard limits are enforced again by recommend.Check and recommend.PlanUnit; the prompt aims at
// plans that pass them and make sense.
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
		fmt.Fprintf(&acts, "- %s = %s: %s; load factor %.1f; %d-%d minutes; %s\n",
			a, a.GermanName(), activityHints[a], load.Factor(a, 0), lo, hi, levelRanges(a))
	}
	return `You are an experienced riding instructor trained in the German FN system. You plan one training week for one horse at a private stable. The horse's welfare comes first: when in doubt, plan less.

INPUT (JSON)
- discipline: dressage, jumping, eventing, leisure, western or young_horse. level: beginner (basic training, class E-A), intermediate (A*-L*), advanced (L** and above); missing = unknown. horse_age_years; missing = unknown.
- phase: the horse's phase in the training scale: gewoehnung (Takt, Losgelassenheit, Anlehnung), schubkraft (developing thrust, up to Geraderichtung) or tragkraft (developing carrying power, Schwung to Versammlung). goal_level: the level class the owner trains the horse towards; missing = none.
- block: week 1 to 3 build up, week 4 (deload: true) is the Entlastungswoche; missing = no training blocks. restart_week: 1 to 3 while the horse returns to work after a break of 14 days or more (also given per day); missing = not returning.
- status: fit, reha (rehabilitation after an injury) or pause (training break).
- rhythm: sessions_min-sessions_max sessions and rest_days_min-rest_days_max rest days per week (Monday to Sunday, done and planned days count), max_minutes per session (0 = no limit), rest_after_show.
- activities: the only activities allowed for this horse. "conditional": allowed under a condition the owner knows. equipment: the facility equipment (poles, cavaletti, jumps, obstacles) of the location of the activity. available_equipment: what can be used there on a normal day; nothing when "shared" is true (the location is used together with others) unless the day says equipment_ok: true.
- history: sessions of the last 14 days. days_ago 1 = yesterday. load = minutes x intensity; a day above 60 was demanding. feel: how the horse felt (fresh, loose, tired, tense). feeling: how the session felt to the rider (hard, good, easy). exercise and rating: the library exercise practised and how it went (hard, better, solid). focus_key: what the unit was about.
- shows_in_days: competitions relative to today (0 = today, negative = past).
- today, tomorrow: rain, night_min_c (coldest temperature of the night) and, for today, the ground (dry, wet, muddy, frozen).
- focus_catalog: the focuses: key, tags (the exercise tags it matches), fits (the activities it can be trained in). horse_focuses: the owner's focuses for the horse in priority order; above_phase: the focus is above the horse's phase.
- exercises: candidates from the exercise library: key, library (dressage, cavaletti, jumping, western, groundwork, lunge, hack), level, phase (warmup, main, cooldown or any), minutes (typical duration), intensity (light, normal, demanding), German title, tags (aims of the training scale), requires (keys of exercises that have to be mastered first), equipment (the set-up it needs in the location), mastered (went solid before), practised (done at least once), next (key of the follow-up exercise).
- days: the week. Plan every day with "open": true, and only those. Closed days are context: planned = an activity someone already planned (focus_key = its focus), planned_minutes = its duration, rest = rest day, done = a session took place. reha = the unit the vet's rehabilitation plan allows that day. avoid (open days only) = activities the owner rejected for that day; never propose them. rider_focuses (open days only) = the focuses of the person who claimed the day; they come first. equipment_ok: false = nothing can be set up that day, true = set-up is possible that day even in a shared location.

ACTIVITIES (code = German name: meaning; intensity; sensible duration)
` + acts.String() + `
UNIT LEVELS
The load of a unit is minutes x load factor. Its level follows from the load: active recovery below 20, light below 30, normal below 45, demanding from 45. The activity list shows the minutes for each level. A rest day has no unit.

HOW TO PLAN
1. Week structure.
   a. The owner's structure is binding. days[].rule: rest = rest day, recovery / light / normal / demanding = one unit of that level, an activity code = that activity. quotas: units per week (Monday to Sunday, done and planned days count) - demanding units, active recovery units and units per activity. Reach every quota, never exceed the demanding and activity quotas.
   b. Where the owner set no rule: never two demanding days in a row; after two days with normal or demanding load a light unit, active recovery or a rest day; alternate the levels over the week. At most 2 demanding units per week, a horse of 18 or older at most 1.
   c. Keep at least rest_days_min rest days. Active recovery (walker, a hack at walk, easy lunging or groundwork, short) keeps the horse moving and is often better than a second rest day.
   d. load.week_limit, when given, caps the total load of the week (done, planned and your units): at most 20 % above the average of the two weeks before, so after a break build up slowly. In the Entlastungswoche (block.deload) no demanding unit and at most 80 % of the average of the three weeks before.
   e. Count done and planned days so that the week ends within sessions_min-sessions_max.
   f. Variety (a versatile basic training): never the same activity on two days in a row when two other allowed activities fit that day; with three or more activities allowed, at least three different ones in the week; a hack when it is allowed and the week has none yet. Jumping at most twice a week and never on consecutive days. Poles and cavaletti once or twice a week, never on consecutive days.
   g. Return to work (restart_week): week 1 only light units and active recovery, weeks 2 and 3 nothing demanding, build up slowly.
   h. Young horses (under 6): units of at most 30 minutes, at most 4 units a week, groundwork, lunging and calm hacks preferred. Horses of 18 or older: long walk phases, fewer demanding days.
2. Focus: every unit of hall, arena, lunge, jumping, groundwork or hack has exactly one focus_key from focus_catalog. Rest and walker: null.
   a. Candidates: the day's rider_focuses first, then horse_focuses in their order, only those whose fits contains the activity. A focus with above_phase true at most once a week, and not when it was trained in the last 7 days.
   b. Take the candidate that was trained longest ago (history focus_key, units you planned earlier in the week count); never trained counts as longest; ties go to the earlier candidate. So several focuses alternate over the week.
   c. When a show is within 14 days and turnier is not among the focuses, turnier may be the focus of hall, arena and jumping units, at most twice a week. In the 7 days before a show use only the first horse focus and turnier.
   d. When no candidate fits the activity: hall, arena and lunge - losgelassenheit; groundwork - vertrauen; hack - gelaende when it is one of the horse's focuses, else kondition; jumping - springen.
   e. Content follows the training scale (Skala der Ausbildung): Takt, Losgelassenheit, Anlehnung, Schwung, Geraderichtung, Versammlung, with Durchlaessigkeit as the aim. Build from the bottom: young horses, beginner or unknown level, after a break, in reha or after a tense session: Takt and Losgelassenheit.
3. Unit: Lösen (warmup), Arbeiten (main) and Entspannen (cooldown). The program splits the minutes (about a third warm-up, a sixth cool-down, longer for seniors and on cold nights); you only choose the exercises. Each part has at most one exercise, none twice in a unit; null = free work.
   a. warmup: phase warmup or any, tags losgelassenheit, takt or mobilitaet; not the one used yesterday. main: phase main or any, tags that meet the focus's tags; prefer the next step after a mastered exercise, an exercise rated hard 3 to 7 days ago, one rated better recently; avoid mastered ones and those used in the last 3 days; a demanding exercise not on a light unit, a light one not on a demanding unit. cooldown: phase cooldown or any, tags losgelassenheit or mobilitaet.
   b. Only exercises of the libraries that fit the activity: hall and arena - dressage and cavaletti (jumping horses: jumping, cavaletti and dressage; western horses: western, cavaletti and dressage), jumping - jumping and cavaletti, lunge - lunge and cavaletti, groundwork - groundwork, hack - hack. Walker and rest - null.
   c. Level at or below the horse's; one level above only when every exercise in requires is mastered and the horse is fit, an adult, not in the Entlastungswoche and not returning to work; at most 2 units a week with a main exercise of the goal_level, a lesson of it that is not mastered yet is preferred. Every exercise in requires has to be mastered. No cavaletti exercise the day after a cavaletti exercise. In the 7 days before a show only exercises with practised true or the tag turnier.
   d. Only exercises whose equipment is in the activity's available_equipment (equipment_ok false: none). Never plan set-up that is not there; take the next best exercise instead.
4. Discipline: dressage - mostly hall or arena, one hack or lunge for relaxation. jumping - gymnastic flatwork and hacks between the jumping units. eventing - flatwork, jumping and hacks for fitness. leisure - hacks and varied, relaxed work, little drill. western - arena work and hacks. young_horse - short units, groundwork, lunging and calm hacks, many breaks.
5. Shows: 2-3 days before at most one last demanding unit; the day before only light work; after a show a rest day when rest_after_show is true.
6. Weather: rain today or tomorrow - prefer the hall to arena and hack. Wet, muddy or frozen ground - no jumping or arena. night_min_c below 0 - frozen ground in the morning.
7. Reha: on a reha day take exactly the reha activity and its minutes, focus_key takt or losgelassenheit. Pause: only walker, groundwork or short lunging.

FOCUS TEXT
- focus: what the unit is about, 2 to 5 German words, e.g. "Übergänge Schritt-Trab", "Dehnungshaltung", "Takt im Arbeitstrab", "Kondition im Gelände", "Gelassenheit an der Hand". Empty for rest. It must fit the focus_key.

REASON
- One short, correct German sentence, at most 100 characters, shown to the owner next to the day.
- About the horse in the third person ("das Pferd", or no subject). Never address the horse or the reader: no "du", "dich", "dir", "dein".
- Use the German activity names from the list above.
- Name the concrete cause from the input: the previous days, variety, a show, the weather, the reha plan, the exercise rating. No general phrases like "um die Muskeln zu stärken".
- "heute" or "morgen" only when in_days is 0 or 1.
Good reasons: "Nach dem Ausritt gestern lockere Arbeit an der Longe." "Zwei Tage vor dem Turnier noch einmal konzentriert in der Halle." "Die Übergänge waren am Montag schwer, daher noch einmal ruhig üben." "Nach drei Arbeitstagen braucht das Pferd einen Ruhetag."

CHECK BEFORE ANSWERING
Every open day exactly once and no closed day; every day rule and every quota met; only allowed activities or rest; minutes within the sensible range and max_minutes and giving the level the day needs; sessions and rest days of the whole week within the rhythm; the week's load within load.week_limit and, in the Entlastungswoche, no demanding unit; no two demanding days in a row unless the owner's rules say so; no repeated activity, jumping and cavaletti limits kept; one focus_key per unit that fits the activity; every exercise key exists, fits its part and the activity, has its requires mastered and its equipment available; reasons follow the rules.

ANSWER
JSON only, no other text:
{"days":[{"day":"<day code from the input>","activity":"<activity code or rest>","minutes":<integer, 0 for rest>,"focus_key":"<catalog key or null>","focus":"<2 to 5 German words or empty>","warmup":"<key or null>","main":"<key or null>","cooldown":"<key or null>","reason":"<reason>"}]}`
}

// levelRanges lists the minutes of each level of an activity for the system prompt.
func levelRanges(a training.Activity) string {
	var parts []string
	for _, lv := range []string{recommend.LevelRecovery, recommend.LevelLight, recommend.LevelNormal, recommend.LevelDemanding} {
		if lo, hi, ok := recommend.LevelMinutes(a, lv); ok {
			parts = append(parts, fmt.Sprintf("%s %d-%d", lv, lo, hi))
		}
	}
	if len(parts) == 0 {
		return "no level within its range"
	}
	return "levels: " + strings.Join(parts, ", ") + " minutes"
}

var ratingWords = map[int]string{1: "hard", 2: "better", 3: "solid"}

// Prompt returns the user message for the model. It holds only what the plan needs, and
// nothing that identifies a person or the horse: no names, no ids, no free text (notes,
// conditions, the level text, show and reha phase names, own-stable exercises) and no dates
// (days are relative to today). The level goes out only as a library level, the age in years;
// the focuses are catalog keys.
func Prompt(in Input) string {
	today := training.Day(in.Today)
	type activityIn struct {
		Activity           training.Activity `json:"activity"`
		Conditional        bool              `json:"conditional,omitempty"`
		Equipment          *[]string         `json:"equipment,omitempty"`
		AvailableEquipment *[]string         `json:"available_equipment,omitempty"`
		Shared             bool              `json:"shared,omitempty"`
	}
	type sessionIn struct {
		DaysAgo     int               `json:"days_ago"`
		Activity    training.Activity `json:"activity"`
		Minutes     int               `json:"minutes"`
		Load        float64           `json:"load"`
		CanterShare float64           `json:"canter_share,omitempty"`
		Feel        string            `json:"feel,omitempty"`
		Feeling     string            `json:"feeling,omitempty"`
		Exercise    string            `json:"exercise,omitempty"`
		Rating      string            `json:"rating,omitempty"`
		FocusKey    string            `json:"focus_key,omitempty"`
	}
	type rehaIn struct {
		Activity   training.Activity `json:"activity"`
		MinMinutes int               `json:"min_minutes,omitempty"`
		MaxMinutes int               `json:"max_minutes,omitempty"`
	}
	type dayIn struct {
		Day            string              `json:"day"`
		InDays         int                 `json:"in_days"`
		Rule           string              `json:"rule,omitempty"`
		Open           bool                `json:"open"`
		Done           bool                `json:"done,omitempty"`
		Planned        training.Activity   `json:"planned,omitempty"`
		PlannedMinutes int                 `json:"planned_minutes,omitempty"`
		FocusKey       string              `json:"focus_key,omitempty"`
		Rest           bool                `json:"rest,omitempty"`
		Reha           *rehaIn             `json:"reha,omitempty"`
		Avoid          []training.Activity `json:"avoid,omitempty"`
		RiderFocuses   []string            `json:"rider_focuses,omitempty"`
		EquipmentOK    *bool               `json:"equipment_ok,omitempty"`
		RestartWeek    int                 `json:"restart_week,omitempty"`
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
	type quotasIn struct {
		Demanding  int                       `json:"demanding,omitempty"`
		Recovery   int                       `json:"recovery,omitempty"`
		Activities map[training.Activity]int `json:"activities,omitempty"`
	}
	type loadIn struct {
		PreviousWeeksAverage *float64 `json:"previous_weeks_average,omitempty"`
		WeekLimit            *float64 `json:"week_limit,omitempty"`
	}
	type blockIn struct {
		Week   int  `json:"week"`
		Deload bool `json:"deload"`
	}
	type focusIn struct {
		Key  string              `json:"key"`
		Tags []string            `json:"tags"`
		Fits []training.Activity `json:"fits"`
	}
	type horseFocusIn struct {
		Key        string `json:"key"`
		AbovePhase bool   `json:"above_phase,omitempty"`
	}
	type exerciseIn struct {
		Key       string   `json:"key"`
		Library   string   `json:"library"`
		Level     string   `json:"level,omitempty"`
		Phase     string   `json:"phase,omitempty"`
		Minutes   int      `json:"minutes,omitempty"`
		Intensity string   `json:"intensity,omitempty"`
		Title     string   `json:"title"`
		Tags      []string `json:"tags,omitempty"`
		Requires  []string `json:"requires,omitempty"`
		Equipment []string `json:"equipment,omitempty"`
		Mastered  bool     `json:"mastered,omitempty"`
		Practised bool     `json:"practised,omitempty"`
		Next      string   `json:"next,omitempty"`
	}
	msg := struct {
		Discipline   string         `json:"discipline,omitempty"`
		Level        string         `json:"level,omitempty"`
		AgeYears     int            `json:"horse_age_years,omitempty"`
		Phase        string         `json:"phase"`
		GoalLevel    string         `json:"goal_level,omitempty"`
		Block        *blockIn       `json:"block,omitempty"`
		RestartWeek  int            `json:"restart_week,omitempty"`
		Status       string         `json:"status"`
		Rhythm       rhythmIn       `json:"rhythm"`
		Quotas       *quotasIn      `json:"quotas,omitempty"`
		Load         loadIn         `json:"load"`
		Activities   []activityIn   `json:"activities"`
		FocusCatalog []focusIn      `json:"focus_catalog"`
		HorseFocuses []horseFocusIn `json:"horse_focuses"`
		History      []sessionIn    `json:"history"`
		ShowsInDays  []int          `json:"shows_in_days"`
		Today        weatherIn      `json:"today"`
		Tomorrow     weatherIn      `json:"tomorrow"`
		Exercises    []exerciseIn   `json:"exercises"`
		Days         []dayIn        `json:"days"`
	}{
		Discipline: in.Profile.Discipline, Level: in.Level, AgeYears: in.AgeYears, Status: string(in.Profile.Status),
		Activities: []activityIn{}, FocusCatalog: []focusIn{}, HorseFocuses: []horseFocusIn{}, History: []sessionIn{},
		ShowsInDays: []int{}, Exercises: []exerciseIn{}, Days: []dayIn{},
	}
	phase := training.Phase(in.Level, in.AgeYears, in.Profile.Status)
	msg.Phase = string(phase)
	msg.GoalLevel = training.GoalLevel(in.Level, in.Profile.GoalLevel)
	if in.Block.Active() {
		msg.Block = &blockIn{Week: in.Block.Week, Deload: in.Block.Deload}
	}
	msg.RestartWeek = recommend.RestartWeek(in.RestartFrom, today)
	r := in.Profile.Rhythm
	msg.Rhythm = rhythmIn{r.SessionsMin, r.SessionsMax, r.RestDaysMin, r.RestDaysMax, r.MaxMinutes, r.RestAfterShow}
	if q := r.Quotas; !q.Empty() {
		qi := &quotasIn{Demanding: q.Demanding, Recovery: q.Recovery}
		for a, n := range q.Activities {
			if n > 0 {
				if qi.Activities == nil {
					qi.Activities = map[training.Activity]int{}
				}
				qi.Activities[a] = n
			}
		}
		msg.Quotas = qi
	}
	if len(in.Days) > 0 {
		monday := training.Day(in.Days[0].Date)
		if limit, ok := recommend.WeekLoadLimit(in.Recent, monday); ok {
			avg, lim := round1(previousAverage(in.Recent, monday)), round1(limit)
			msg.Load = loadIn{PreviousWeeksAverage: &avg, WeekLimit: &lim}
		}
	}
	for _, a := range in.Profile.Allowed {
		if !a.Activity.Valid() || (a.Mode != training.ModeOn && a.Mode != training.ModeConditional) {
			continue
		}
		ai := activityIn{Activity: a.Activity, Conditional: a.Mode == training.ModeConditional}
		if training.IsLocation(a.Activity) {
			loc := in.Equipment.Of(a.Activity)
			facility, available := []string{}, []string{}
			for _, k := range loc.Equipment {
				if training.IsFacility(k) {
					facility = append(facility, k)
				}
			}
			available = append(available, training.Available(loc, nil)...)
			ai.Equipment, ai.AvailableEquipment, ai.Shared = &facility, &available, loc.Shared
		}
		msg.Activities = append(msg.Activities, ai)
	}
	for _, f := range training.Focuses {
		msg.FocusCatalog = append(msg.FocusCatalog, focusIn{Key: f.Key, Tags: f.Tags, Fits: f.Fits})
	}
	for _, c := range in.Profile.Focuses {
		if f, ok := training.FocusByKey(c.Key); ok && c.Active(today) {
			msg.HorseFocuses = append(msg.HorseFocuses, horseFocusIn{Key: c.Key, AbovePhase: !f.AllowedIn(phase)})
		}
	}
	visible := in.visibleExercises()
	keys := map[string]string{} // exercise id -> key
	for i, ex := range visible {
		keys[ex.ID] = exerciseKey(i)
	}
	for i, ex := range visible {
		ei := exerciseIn{
			Key: exerciseKey(i), Library: ex.Library, Level: ex.Level, Phase: ex.Phase, Minutes: ex.Minutes,
			Intensity: ex.Intensity, Title: ex.Title, Tags: ex.Tags, Equipment: ex.FacilityEquipment(),
			Mastered: ex.Mastered, Practised: ex.Rated, Next: keys[ex.NextID],
		}
		for _, id := range ex.Requires {
			if k, ok := keys[id]; ok {
				ei.Requires = append(ei.Requires, k)
			}
		}
		msg.Exercises = append(msg.Exercises, ei)
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
			CanterShare: round1(s.CanterShare), Feel: feel(s.Feel), Feeling: feeling(s.Feeling),
		}
		if k, ok := keys[s.ExerciseID]; ok {
			si.Exercise, si.Rating = k, ratingWords[s.FocusRating]
		}
		if training.ValidFocusKey(s.FocusKey) {
			si.FocusKey = s.FocusKey
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
		di := dayIn{Day: dayCode(day), InDays: training.DaysBetween(today, day), Open: d.Open, Rest: d.Rest,
			Rule: dayRuleText(r.Days[training.WeekdayIndex(day)])}
		if !d.Open {
			di.Done = done[day.Format(dateLayout)]
			if d.Planned.Valid() {
				di.Planned = d.Planned
				if d.Minutes > 0 {
					di.PlannedMinutes = d.Minutes
				}
				if training.ValidFocusKey(d.FocusKey) {
					di.FocusKey = d.FocusKey
				}
			}
		} else {
			for _, a := range d.Exclude {
				if a.Valid() {
					di.Avoid = append(di.Avoid, a)
				}
			}
			for _, k := range d.RiderFocuses {
				if training.ValidFocusKey(k) {
					di.RiderFocuses = append(di.RiderFocuses, k)
				}
			}
			di.EquipmentOK = d.EquipmentOK
			di.RestartWeek = recommend.RestartWeek(in.RestartFrom, day)
		}
		if p := d.Reha; p != nil {
			di.Reha = &rehaIn{Activity: p.Activity, MinMinutes: p.MinMinutes, MaxMinutes: p.MaxMinutes}
		}
		msg.Days = append(msg.Days, di)
	}
	b, _ := json.Marshal(msg)
	return string(b)
}

// dayRuleText is the owner's rule of a day for the prompt: the kind, or the activity code.
func dayRuleText(d training.DayRule) string {
	if d.Kind == training.DayActivity {
		return string(d.Activity)
	}
	return d.Kind
}

// previousAverage is the average load of the two weeks before monday.
func previousAverage(recent []training.Session, monday time.Time) float64 {
	var sum float64
	for _, s := range recent {
		if d := training.DaysBetween(monday, s.Day); d >= -14 && d < 0 {
			sum += s.Load
		}
	}
	return sum / 2
}

// feel passes only the known feel values (they are chosen from a list in the app).
func feel(s string) string {
	switch s {
	case "fresh", "loose", "tired", "tense":
		return s
	}
	return ""
}

// feeling passes only the known values of "Wie war's?" (hard, good, easy).
func feeling(s string) string {
	switch s {
	case "hard", "good", "easy":
		return s
	}
	return ""
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// Parse reads the model's answer and returns the proposals for the open days, keyed by
// date (YYYY-MM-DD). Entries for unknown or closed days and unknown activities are dropped
// (the rules plan those days), unknown focus keys become no focus and unknown exercise keys no
// exercise (the rules pick); an answer that is not the expected JSON is an error. The key
// "exercise" of an earlier prompt version counts as the main exercise.
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
			FocusKey json.RawMessage `json:"focus_key"`
			Focus    json.RawMessage `json:"focus"`
			Warmup   json.RawMessage `json:"warmup"`
			Main     json.RawMessage `json:"main"`
			Cooldown json.RawMessage `json:"cooldown"`
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
	for i, ex := range in.visibleExercises() {
		byKey[exerciseKey(i)] = ex
	}
	exercise := func(msgs ...json.RawMessage) *Exercise {
		for _, m := range msgs {
			var key string
			if json.Unmarshal(m, &key) == nil && key != "" {
				return byKey[strings.ToUpper(strings.TrimSpace(key))]
			}
		}
		return nil
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
		var reason, focus, focusKey string
		_ = json.Unmarshal(d.Reason, &reason)
		_ = json.Unmarshal(d.Focus, &focus)
		_ = json.Unmarshal(d.FocusKey, &focusKey)
		focusKey = strings.ToLower(strings.TrimSpace(focusKey))
		if !training.ValidFocusKey(focusKey) {
			focusKey = ""
		}
		out[date.Format(dateLayout)] = Proposal{
			Activity: act, Minutes: minutes, Reason: cleanText(reason, MaxReasonRunes),
			Focus: cleanText(focus, MaxFocusRunes), FocusKey: focusKey,
			Warmup: exercise(d.Warmup), Main: exercise(d.Main, d.Exercise), Cooldown: exercise(d.Cooldown),
		}
	}
	return out, nil
}

// cleanText makes the model's text safe to show: one line, no control characters, capped.
func cleanText(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
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
