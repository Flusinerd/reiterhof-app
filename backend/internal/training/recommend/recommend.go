// Package recommend implements the rule-based "Was heute?" recommender (JAN-56).
//
// # Algorithm
//
// Recommend is pure and deterministic; "today" comes from Input.Today.
//
//  1. Visibility. Every activity that is off (or missing) in the training
//     profile is hidden with a German reason ("Springen ausgeblendet: laut
//     Profil nicht für Luna"). For users with role rider, the rider's rules
//     hide further activities (not on the allow-list, above max intensity, hack
//     when the rider may not hack alone). A rider without rules gets nothing
//     (fail closed). Conditional activities stay candidates and carry their
//     note in Recommendation.Note.
//  2. Overrides, in this order:
//     a. Active reha phase (status is not pause): exactly one recommendation,
//     the phase's activity, minutes clamped into [min,max] and to the
//     available time, reason names the phase. Rider rules still apply.
//     b. Mandatory rest day after a show (Rhythm.RestAfterShow and a show
//     yesterday): a single rest recommendation.
//     c. No candidate left: a single rest recommendation with the cause.
//  3. Hard filters: status pause or reha without phase keep only light
//     activities (max PauseMaxMinutes); a show today or tomorrow keeps only
//     light activities (max ShowLightMaxMinutes); frozen ground removes jumping.
//  4. Scoring. score = sum of weighted factors (constants Weight* below):
//     - variety: activity not done for >= 2 days (+ per idle day, capped), not
//     done at all in the recent sessions (+), done yesterday/today (-);
//     - recent load: after an intense day in the last 3 days (today,
//     yesterday, day before) light is preferred, intense penalised; after
//     no load at all medium/intense get a small bonus;
//     - show: 2-3 days ahead a last demanding session is fine (bonus for
//     medium/intense when there was no intense day in the last 2 days);
//     - weather: rain or wet/muddy/frozen ground favour the hall and penalise
//     arena and hack; wet ground penalises jumping; heat penalises medium and
//     intense, extreme cold outdoor work;
//     - time: the duration is cut to the available time and rhythm maximum;
//     less than the activity's minimum is penalised, a cut below 70 % of the
//     default duration is penalised slightly;
//     - status: pause/reha favour the remaining light activities;
//     - conditional activities and discipline fit apply small adjustments.
//     A rest recommendation competes only when the week already has
//     Rhythm.SessionsMax sessions (WeightWeekFull).
//  5. Ranking. Highest score first; ties are broken by the canonical activity
//     order (training.AllActivities, rest first). The top 3 are returned.
//     Reason is the sentence of the factor with the largest positive
//     contribution.
package recommend

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/load"
)

// Scoring weights. All contributions are added to the candidate's score.
const (
	WeightUntrained     = 2.0  // activity not in the recent sessions at all
	WeightPerIdleDay    = 0.4  // per day since the activity was last done (>= 2 days)
	MaxIdleDays         = 6    // idle days beyond this add nothing
	WeightRepeat        = -1.0 // activity was done yesterday
	WeightRepeatToday   = -1.5 // activity was already done today
	WeightRecoverLight  = 3.0  // intense recent day: light candidates
	WeightRecoverMedium = -1.5 // intense recent day: medium candidates
	WeightRecoverHard   = -3.0 // intense recent day: intense candidates
	WeightFresh         = 0.5  // no recent load: medium/intense candidates
	WeightShowLight     = 2.5  // show today/tomorrow: light candidates
	WeightShowLastHard  = 1.0  // show in 2-3 days: last demanding session
	WeightWeatherHall   = 3.0  // rain / bad ground: hall
	WeightWeatherArena  = -3.0 // rain / bad ground: arena
	WeightWeatherHack   = -2.0 // rain / bad ground: hack
	WeightFrozenExtra   = -1.0 // frozen ground: extra penalty for arena and hack
	WeightWetJumping    = -2.0 // wet or muddy ground: jumping
	WeightHeat          = -1.5 // temperature >= HeatC: medium and intense
	WeightCold          = -1.5 // temperature <= ColdC: arena and hack
	WeightTooShort      = -3.0 // available time below the activity's minimum
	WeightShortened     = -1.0 // duration cut below 70 % of the default
	WeightStatusLight   = 3.0  // pause / reha without phase: light activities
	WeightConditional   = -0.5 // conditional activities are slightly less attractive
	WeightDiscipline    = 0.5  // activity typical for the discipline
	WeightWeekFull      = 10.0 // rest, when the week already has enough sessions
)

// Other tunables.
const (
	HeatC               = 28.0 // degrees Celsius
	ColdC               = -10.0
	PauseMaxMinutes     = 20
	ShowLightMaxMinutes = 30
	RecentDays          = 14
	MaxRecommendations  = 3
)

// Role of the requesting user.
type Role string

const (
	RoleOwner Role = "owner"
	RoleRider Role = "rider"
)

// Ground condition; the empty value means unknown.
type Ground string

const (
	GroundDry    Ground = "dry"
	GroundWet    Ground = "wet"
	GroundFrozen Ground = "frozen"
	GroundMuddy  Ground = "muddy"
)

// Weather of today.
type Weather struct {
	Rain  bool
	TempC float64
}

// RehaPhase is the active phase of a rehabilitation plan.
type RehaPhase struct {
	Name       string
	Activity   training.Activity
	MinMinutes int
	MaxMinutes int
	Conditions string
}

// Input is everything the recommender needs.
type Input struct {
	Today   time.Time // stable-local calendar day
	Profile training.Profile
	// Recent are the sessions of the last 14 days (older ones are ignored).
	Recent  []training.Session
	Weather *Weather // nil: unknown
	Ground  Ground   // empty: unknown
	// AvailableMinutes is the time the user has; 0 means unknown/unlimited.
	AvailableMinutes int
	Role             Role
	RiderName        string
	Rider            *training.RiderRules // only used for RoleRider
	Reha             *RehaPhase           // active reha phase, if any
}

// Recommendation is one suggestion. Activity is training.ActivityRest for a
// rest day (Minutes 0, Intensity none).
type Recommendation struct {
	Activity  training.Activity  `json:"activity"`
	Minutes   int                `json:"minutes"`
	Intensity training.Intensity `json:"intensity"`
	Reason    string             `json:"reason"`         // German, one sentence
	Note      string             `json:"note,omitempty"` // condition of a conditional activity
	Score     float64            `json:"score"`
}

// Hidden is an activity excluded by profile or rider rules.
type Hidden struct {
	Activity training.Activity `json:"activity"`
	Reason   string            `json:"reason"` // German
}

// Result of Recommend. Recommendations always has at least one element.
type Result struct {
	Recommendations []Recommendation `json:"recommendations"`
	Hidden          []Hidden         `json:"hidden"`
}

// durations per activity: usual (def), shortest sensible (min) and longest sensible (max)
// unit. Recommend scores against def and min; Check fits proposals into [min, max].
type durations struct{ def, min, max int }

var durationTable = map[training.Activity]durations{
	training.ActivityWalker:     {30, 20, 60},
	training.ActivityGroundwork: {30, 15, 45},
	training.ActivityLunge:      {25, 15, 30},
	training.ActivityHack:       {60, 30, 120},
	training.ActivityHall:       {45, 30, 60},
	training.ActivityArena:      {45, 30, 60},
	training.ActivityJumping:    {40, 30, 45},
}

// MinutesRange is the shortest and the longest sensible unit of an activity (0, 0 for rest
// or unknown ones).
func MinutesRange(a training.Activity) (min, max int) {
	d := durationTable[a]
	return d.min, d.max
}

func DefaultMinutes(a training.Activity) int { return durationTable[a].def }

type disciplineInfo struct {
	name       string
	activities []training.Activity
}

var disciplines = map[string]disciplineInfo{
	"dressage":    {"Dressur", []training.Activity{training.ActivityHall, training.ActivityArena, training.ActivityLunge}},
	"jumping":     {"Springreiten", []training.Activity{training.ActivityJumping, training.ActivityArena, training.ActivityHall}},
	"eventing":    {"Vielseitigkeit", []training.Activity{training.ActivityJumping, training.ActivityHack, training.ActivityArena}},
	"leisure":     {"Freizeitreiten", []training.Activity{training.ActivityHack, training.ActivityArena}},
	"western":     {"Westernreiten", []training.Activity{training.ActivityArena, training.ActivityHack}},
	"young_horse": {"Jungpferdeausbildung", []training.Activity{training.ActivityGroundwork, training.ActivityLunge, training.ActivityHack}},
}

type factor struct {
	delta float64
	text  string
}

type candidate struct {
	activity training.Activity
	note     string
	minutes  int
	factors  []factor
}

func (c *candidate) add(delta float64, text string) {
	c.factors = append(c.factors, factor{delta, text})
}

func (c *candidate) score() float64 {
	var s float64
	for _, f := range c.factors {
		s += f.delta
	}
	return s
}

// Recommend returns the top recommendations for today.
func Recommend(in Input) Result {
	today := training.Day(in.Today)
	horse := in.Profile.HorseName
	if horse == "" {
		horse = "das Pferd"
	}
	rider := in.RiderName
	if rider == "" {
		rider = "den Reiter"
	}

	hidden, cands := visibility(in, horse, rider)
	res := Result{Hidden: hidden}
	one := func(r Recommendation) Result {
		res.Recommendations = []Recommendation{r}
		return res
	}

	// 2a. Active reha phase.
	if in.Reha != nil && in.Profile.Status != training.StatusPause {
		p := in.Reha
		if p.Activity == training.ActivityRest {
			// Box rest: nothing to do with the horse today.
			r := rest(fmt.Sprintf("Reha-Phase „%s“: heute keine Bewegung.", p.Name))
			r.Note = p.Conditions
			return one(r)
		}
		if reason := riderBlock(in, p.Activity, rider); reason != "" {
			return one(rest(fmt.Sprintf("Reha-Phase „%s“: %s ist für %s nicht freigegeben – heute lieber Ruhe.", p.Name, p.Activity.GermanName(), rider)))
		}
		minutes := p.MaxMinutes
		if in.AvailableMinutes > 0 && in.AvailableMinutes < minutes {
			minutes = in.AvailableMinutes
		}
		if minutes < p.MinMinutes {
			minutes = p.MinMinutes
		}
		reason := fmt.Sprintf("Reha-Phase „%s“: %s für %d Minuten.", p.Name, p.Activity.GermanName(), minutes)
		return one(Recommendation{
			Activity: p.Activity, Minutes: minutes, Intensity: training.DefaultIntensity(p.Activity),
			Reason: reason, Note: p.Conditions, Score: math.MaxFloat64,
		})
	}

	// 2b. Rest day after a show.
	if in.Profile.Rhythm.RestAfterShow {
		for _, s := range in.Profile.Shows {
			if training.DaysBetween(s.Date, today) == 1 {
				return one(rest(fmt.Sprintf("Gestern war %s – %s bekommt heute einen Ruhetag.", s.Name, horse)))
			}
		}
	}

	// 2c. Nothing to choose from.
	if len(cands) == 0 {
		return one(rest(emptyReason(in, hidden, horse, rider)))
	}

	// 3. Hard filters.
	recentOnly := in.Profile.Status == training.StatusPause || in.Profile.Status == training.StatusReha
	showDays, showName := nextShow(in.Profile.Shows, today)
	showNear := showDays >= 0 && showDays <= 1
	var kept []*candidate
	for _, c := range cands {
		if filterReason(in, c.activity, recentOnly, showNear, showDays, showName, horse) != "" {
			continue
		}
		kept = append(kept, c)
	}
	if len(kept) == 0 {
		return one(rest(filteredReason(in, recentOnly, showNear, showName, horse)))
	}

	// 4. Scoring.
	ctx := newContext(in, today, horse, showDays, showName, recentOnly, showNear)
	var ranked []Recommendation
	for _, c := range kept {
		ctx.score(c)
		ranked = append(ranked, Recommendation{
			Activity: c.activity, Minutes: c.minutes, Intensity: training.DefaultIntensity(c.activity),
			Reason: reasonOf(c, horse), Note: c.note, Score: c.score(),
		})
	}
	if r := in.Profile.Rhythm; r.SessionsMax > 0 && weekSessions(in.Recent, today) >= r.SessionsMax {
		rr := rest(fmt.Sprintf("Diese Woche gab es schon %d Einheiten – ein Ruhetag tut %s gut.", weekSessions(in.Recent, today), horse))
		rr.Score = WeightWeekFull
		ranked = append(ranked, rr)
	}

	// 5. Ranking with deterministic tie-break.
	sort.SliceStable(ranked, func(i, j int) bool {
		if math.Abs(ranked[i].Score-ranked[j].Score) > 1e-9 {
			return ranked[i].Score > ranked[j].Score
		}
		return order(ranked[i].Activity) < order(ranked[j].Activity)
	})
	if len(ranked) > MaxRecommendations {
		ranked = ranked[:MaxRecommendations]
	}
	res.Recommendations = ranked
	return res
}

func order(a training.Activity) int {
	if a == training.ActivityRest {
		return -1
	}
	for i, x := range training.AllActivities {
		if x == a {
			return i
		}
	}
	return len(training.AllActivities)
}

func rest(reason string) Recommendation {
	return Recommendation{Activity: training.ActivityRest, Intensity: training.IntensityNone, Reason: reason}
}

// visibility applies profile and rider rules.
func visibility(in Input, horse, rider string) ([]Hidden, []*candidate) {
	var hidden []Hidden
	var cands []*candidate
	for _, act := range training.AllActivities {
		var entry *training.AllowedActivity
		for i := range in.Profile.Allowed {
			if in.Profile.Allowed[i].Activity == act {
				entry = &in.Profile.Allowed[i]
			}
		}
		if entry == nil || (entry.Mode != training.ModeOn && entry.Mode != training.ModeConditional) {
			hidden = append(hidden, Hidden{act, fmt.Sprintf("%s ausgeblendet: laut Profil nicht für %s", act.GermanName(), horse)})
			continue
		}
		if reason := riderBlock(in, act, rider); reason != "" {
			hidden = append(hidden, Hidden{act, fmt.Sprintf("%s ausgeblendet: %s", act.GermanName(), reason)})
			continue
		}
		c := &candidate{activity: act}
		if entry.Mode == training.ModeConditional {
			c.note = entry.Note
		}
		cands = append(cands, c)
	}
	return hidden, cands
}

// riderBlock returns a German fragment why the rider may not do act, or "".
func riderBlock(in Input, act training.Activity, rider string) string {
	if in.Role != RoleRider {
		return ""
	}
	r := in.Rider
	if r == nil {
		return "keine Freigabe für " + rider
	}
	allowed := false
	for _, a := range r.AllowedActivities {
		if a == act {
			allowed = true
		}
	}
	switch {
	case !allowed:
		return fmt.Sprintf("für %s nicht freigegeben", rider)
	case act == training.ActivityHack && !r.MayHackAlone:
		return fmt.Sprintf("%s darf nicht allein ausreiten", rider)
	case r.MaxIntensity != training.IntensityNone && training.DefaultIntensity(act) > r.MaxIntensity:
		return fmt.Sprintf("zu intensiv für %s", rider)
	}
	return ""
}

func emptyReason(in Input, hidden []Hidden, horse, rider string) string {
	profileAllows := false
	for _, a := range in.Profile.Allowed {
		if a.Mode == training.ModeOn || a.Mode == training.ModeConditional {
			profileAllows = true
		}
	}
	if !profileAllows {
		return fmt.Sprintf("Im Profil von %s ist keine Aktivität freigegeben – heute Ruhetag.", horse)
	}
	return fmt.Sprintf("Für %s ist heute nichts freigegeben – bitte den Besitzer fragen.", rider)
}

func filteredReason(in Input, pause, show bool, showName, horse string) string {
	switch {
	case pause && in.Profile.Status == training.StatusPause:
		return fmt.Sprintf("%s macht gerade Pause und hat nichts Leichtes freigegeben – heute Ruhetag.", horse)
	case pause:
		return fmt.Sprintf("%s ist in der Reha und hat nichts Leichtes freigegeben – heute Ruhetag.", horse)
	case show:
		return fmt.Sprintf("%s steht kurz bevor und nur Intensives ist freigegeben – heute Ruhetag.", showName)
	}
	return fmt.Sprintf("Bei diesem Boden ist für %s heute nichts Passendes freigegeben – Ruhetag.", horse)
}

// filterReason applies the hard filters of step 3 to one activity. It returns a German
// sentence why the activity is out today, or "" when it passes.
func filterReason(in Input, act training.Activity, recentOnly, showNear bool, showDays int, showName, horse string) string {
	light := training.DefaultIntensity(act) == training.IntensityLight
	switch {
	case recentOnly && !light && in.Profile.Status == training.StatusPause:
		return fmt.Sprintf("%s macht gerade Pause – nur leichte Arbeit.", horse)
	case recentOnly && !light:
		return fmt.Sprintf("%s ist in der Reha – nur leichte Arbeit.", horse)
	case showNear && !light:
		when := "morgen"
		if showDays == 0 {
			when = "heute"
		}
		return fmt.Sprintf("%s ist %s – nur leichte Arbeit.", showName, when)
	case in.Ground == GroundFrozen && act == training.ActivityJumping:
		return "Der Boden ist gefroren – kein Springen."
	}
	return ""
}

// Verdict is the result of Check.
type Verdict struct {
	// OK is true when the proposal passes the rules.
	OK bool
	// Recommendation is the proposal with its minutes fitted to the limits when OK, else the
	// rule-based replacement (the top recommendation of Recommend, or a rest day when the
	// week already has enough sessions). Reason is empty for an accepted proposal.
	Recommendation Recommendation
	// Why says in German why the proposal was replaced; empty when OK.
	Why string
}

// Check tests a proposed activity (training.ActivityRest for a rest day) with a duration
// for in.Today against the hard rules of Recommend: visibility (profile, rider rules), the
// active reha phase, the rest day after a show, the filters for pause/reha, a near show and
// frozen ground, and in addition the weekly maximum of sessions (Rhythm.SessionsMax). The
// soft scoring does not apply: every activity that passes the rules is fine.
//
// A rest day always passes. Minutes are fitted into the limits: the reha phase range, the
// activity's sensible range (MinutesRange, e.g. at most 30 minutes on the lunge), the rhythm
// maximum, PauseMaxMinutes and ShowLightMaxMinutes; zero or
// negative minutes become the activity's default duration.
func Check(in Input, act training.Activity, minutes int) Verdict {
	today := training.Day(in.Today)
	horse := in.Profile.HorseName
	if horse == "" {
		horse = "das Pferd"
	}
	rider := in.RiderName
	if rider == "" {
		rider = "den Reiter"
	}
	if act == training.ActivityRest {
		return Verdict{OK: true, Recommendation: rest("")}
	}
	replace := func(why string) Verdict {
		return Verdict{Recommendation: Recommend(in).Recommendations[0], Why: why}
	}
	if !act.Valid() {
		return replace(fmt.Sprintf("„%s“ ist keine bekannte Aktivität.", act))
	}

	// Overrides of Recommend: reha phase and rest after a show fix the day.
	if in.Reha != nil && in.Profile.Status != training.StatusPause {
		forced := Recommend(in).Recommendations[0]
		if forced.Activity != act {
			return Verdict{Recommendation: forced, Why: forced.Reason}
		}
		m := minutes
		if m <= 0 || m > in.Reha.MaxMinutes {
			m = in.Reha.MaxMinutes
		}
		if m < in.Reha.MinMinutes {
			m = in.Reha.MinMinutes
		}
		forced.Minutes, forced.Reason, forced.Score = m, "", 0
		return Verdict{OK: true, Recommendation: forced}
	}
	if in.Profile.Rhythm.RestAfterShow {
		for _, s := range in.Profile.Shows {
			if training.DaysBetween(s.Date, today) == 1 {
				forced := Recommend(in).Recommendations[0]
				return Verdict{Recommendation: forced, Why: forced.Reason}
			}
		}
	}

	// Visibility and hard filters.
	hidden, cands := visibility(in, horse, rider)
	for _, h := range hidden {
		if h.Activity == act {
			return replace(h.Reason + ".")
		}
	}
	var cand *candidate
	for _, c := range cands {
		if c.activity == act {
			cand = c
		}
	}
	if cand == nil {
		return replace(fmt.Sprintf("%s ist für %s nicht freigegeben.", act.GermanName(), horse))
	}
	recentOnly := in.Profile.Status == training.StatusPause || in.Profile.Status == training.StatusReha
	showDays, showName := nextShow(in.Profile.Shows, today)
	showNear := showDays >= 0 && showDays <= 1
	if why := filterReason(in, act, recentOnly, showNear, showDays, showName, horse); why != "" {
		return replace(why)
	}
	if r := in.Profile.Rhythm; r.SessionsMax > 0 && weekSessions(in.Recent, today) >= r.SessionsMax {
		why := fmt.Sprintf("Diese Woche gab es schon %d Einheiten – mehr sieht der Rhythmus nicht vor.", weekSessions(in.Recent, today))
		return Verdict{Recommendation: rest(why), Why: why}
	}

	// Minutes.
	dur := durationTable[act]
	if minutes <= 0 {
		minutes = dur.def
	}
	if minutes < dur.min {
		minutes = dur.min
	}
	limit := dur.max
	if m := in.Profile.Rhythm.MaxMinutes; m > 0 && m < limit {
		limit = m
	}
	if recentOnly && PauseMaxMinutes < limit {
		limit = PauseMaxMinutes
	}
	if showNear && ShowLightMaxMinutes < limit {
		limit = ShowLightMaxMinutes
	}
	if minutes > limit {
		minutes = limit
	}
	return Verdict{OK: true, Recommendation: Recommendation{
		Activity: act, Minutes: minutes, Intensity: training.DefaultIntensity(act), Note: cand.note,
	}}
}

// nextShow returns days until the next show (today counts as 0), or -1.
func nextShow(shows []training.Show, today time.Time) (int, string) {
	best, name := -1, ""
	for _, s := range shows {
		d := training.DaysBetween(today, s.Date)
		if d >= 0 && (best < 0 || d < best) {
			best, name = d, s.Name
		}
	}
	return best, name
}

// weekSessions counts sessions in the Monday-based week containing today.
func weekSessions(sessions []training.Session, today time.Time) int {
	offset := (int(today.Weekday()) + 6) % 7
	start := today.AddDate(0, 0, -offset)
	n := 0
	for _, s := range sessions {
		d := training.DaysBetween(start, s.Day)
		if d >= 0 && d < 7 && training.DaysBetween(s.Day, today) >= 0 {
			n++
		}
	}
	return n
}

type context struct {
	in         Input
	today      time.Time
	horse      string
	showDays   int
	showName   string
	recentOnly bool
	showNear   bool
	idle       map[training.Activity]int // days since last done, absent: never
	recentHard bool                      // intense day within today..today-2
	recentNone bool                      // no load at all within today..today-2
	hardLast2  bool                      // intense day within yesterday, day before
	heavy      bool                      // weather/ground is bad for outdoor work
	groundText string
}

func newContext(in Input, today time.Time, horse string, showDays int, showName string, recentOnly, showNear bool) *context {
	c := &context{in: in, today: today, horse: horse, showDays: showDays, showName: showName,
		recentOnly: recentOnly, showNear: showNear, idle: map[training.Activity]int{}, recentNone: true}
	var recent []training.Session
	for _, s := range in.Recent {
		d := training.DaysBetween(s.Day, today)
		if d < 0 || d >= RecentDays {
			continue
		}
		recent = append(recent, s)
		if cur, ok := c.idle[s.Activity]; !ok || d < cur {
			c.idle[s.Activity] = d
		}
	}
	for off := 0; off <= 2; off++ {
		l := load.DayLoad(recent, today.AddDate(0, 0, -off))
		if l > 0 {
			c.recentNone = false
		}
		if load.Classify(l) == training.IntensityIntense {
			c.recentHard = true
			if off >= 1 {
				c.hardLast2 = true
			}
		}
	}
	switch in.Ground {
	case GroundWet:
		c.groundText = "nass"
	case GroundMuddy:
		c.groundText = "matschig"
	case GroundFrozen:
		c.groundText = "gefroren"
	}
	c.heavy = (in.Weather != nil && in.Weather.Rain) || c.groundText != ""
	return c
}

func (x *context) score(c *candidate) {
	act := c.activity
	name := act.GermanName()
	lvl := training.DefaultIntensity(act)
	h := x.horse

	// Variety.
	if d, ok := x.idle[act]; !ok {
		c.add(WeightUntrained, fmt.Sprintf("%s stand in den letzten %d Tagen nicht an – das bringt Abwechslung für %s.", name, RecentDays, h))
	} else if d >= 2 {
		n := d
		if n > MaxIdleDays {
			n = MaxIdleDays
		}
		c.add(WeightPerIdleDay*float64(n), fmt.Sprintf("%s war seit %d Tagen nicht dran.", name, d))
	} else if d == 1 {
		c.add(WeightRepeat, "")
	} else {
		c.add(WeightRepeatToday, "")
	}

	// Recent load.
	if x.recentHard {
		switch lvl {
		case training.IntensityLight:
			c.add(WeightRecoverLight, fmt.Sprintf("Die letzten Tage waren intensiv – %s ist heute schonend für %s.", name, h))
		case training.IntensityMedium:
			c.add(WeightRecoverMedium, "")
		default:
			c.add(WeightRecoverHard, "")
		}
	} else if x.recentNone && lvl != training.IntensityLight {
		c.add(WeightFresh, fmt.Sprintf("%s hatte zuletzt wenig Belastung – heute ist wieder eine kräftigere Einheit drin.", h))
	}

	// Shows.
	if x.showNear && lvl == training.IntensityLight {
		when := "morgen"
		if x.showDays == 0 {
			when = "heute"
		}
		c.add(WeightShowLight, fmt.Sprintf("%s ist %s – da passt nur leichte Arbeit wie %s.", x.showName, when, name))
	} else if (x.showDays == 2 || x.showDays == 3) && lvl != training.IntensityLight && !x.hardLast2 {
		c.add(WeightShowLastHard, fmt.Sprintf("%s ist in %d Tagen – jetzt passt noch eine letzte fordernde Einheit.", x.showName, x.showDays))
	}

	// Weather and ground.
	rain := x.in.Weather != nil && x.in.Weather.Rain
	if x.heavy {
		cause := "Bei Regen"
		if !rain {
			cause = "Der Boden ist " + x.groundText
		}
		switch act {
		case training.ActivityHall:
			if rain {
				c.add(WeightWeatherHall, "Bei Regen ist die Halle die beste Wahl.")
			} else {
				c.add(WeightWeatherHall, cause+" – die Halle ist die bessere Wahl.")
			}
		case training.ActivityArena:
			c.add(WeightWeatherArena, "")
		case training.ActivityHack:
			c.add(WeightWeatherHack, "")
		}
	}
	if x.in.Ground == GroundFrozen && (act == training.ActivityArena || act == training.ActivityHack) {
		c.add(WeightFrozenExtra, "")
	}
	if (x.in.Ground == GroundWet || x.in.Ground == GroundMuddy) && act == training.ActivityJumping {
		c.add(WeightWetJumping, "")
	}
	if w := x.in.Weather; w != nil {
		if w.TempC >= HeatC && lvl != training.IntensityLight {
			c.add(WeightHeat, "")
		}
		if w.TempC <= ColdC && (act == training.ActivityArena || act == training.ActivityHack) {
			c.add(WeightCold, "")
		}
	}

	// Time.
	dur := durationTable[act]
	minutes := dur.def
	if m := x.in.Profile.Rhythm.MaxMinutes; m > 0 && m < minutes {
		minutes = m
	}
	if a := x.in.AvailableMinutes; a > 0 && a < minutes {
		minutes = a
	}
	if x.recentOnly && PauseMaxMinutes < minutes {
		minutes = PauseMaxMinutes
	}
	if x.showNear && ShowLightMaxMinutes < minutes {
		minutes = ShowLightMaxMinutes
	}
	switch {
	case minutes < dur.min:
		c.add(WeightTooShort, "")
	case float64(minutes) < 0.7*float64(dur.def) && !x.recentOnly && !x.showNear:
		c.add(WeightShortened, "")
	}
	c.minutes = minutes

	// Status.
	if x.recentOnly {
		if x.in.Profile.Status == training.StatusPause {
			c.add(WeightStatusLight, fmt.Sprintf("%s ist in der Trainingspause – %s ist schonend.", h, name))
		} else {
			c.add(WeightStatusLight, fmt.Sprintf("%s ist in der Reha – %s ist schonend.", h, name))
		}
	}

	if c.note != "" {
		c.add(WeightConditional, "")
	}
	if info, ok := disciplines[x.in.Profile.Discipline]; ok {
		for _, a := range info.activities {
			if a == act {
				c.add(WeightDiscipline, fmt.Sprintf("%s passt zu %s.", name, info.name))
			}
		}
	}
}

// reasonOf returns the sentence of the strongest positive factor.
func reasonOf(c *candidate, horse string) string {
	best := -1
	for i, f := range c.factors {
		if f.text == "" || f.delta <= 0 {
			continue
		}
		if best < 0 || f.delta > c.factors[best].delta {
			best = i
		}
	}
	if best >= 0 {
		return c.factors[best].text
	}
	return fmt.Sprintf("%s passt heute für %s.", c.activity.GermanName(), horse)
}
