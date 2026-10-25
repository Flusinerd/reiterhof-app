package recommend

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/load"
)

// Week structure (JAN-93).
//
// A unit has a level by its load (load.Score with no canter): active recovery below
// RecoveryBelow, light below LightBelow, normal up to DemandingFrom, demanding from there.
// structureFor turns the owner's week structure (Rhythm.Days, Rhythm.Quotas), the default
// structure rules, the minimum rest days and the load limit into limits for one day: a rest
// day, activities that are out, and a minute window per activity. Recommend and Check apply
// the same limits. Safety rules (profile, rider, reha, show, pause, frozen ground) come
// first; each group below is dropped when nothing would be left after it, except the load
// limit, which then makes the day a rest day.
//
//  1. The owner's rule for the weekday: rest, an activity, or a level (recovery, light,
//     normal, demanding).
//  2. Default rules, only on days without an owner's rule: no demanding day after a
//     demanding day; after two days with at least normal load only a light unit.
//  3. Needs of the week (only when planning a week, Input.Ahead): when the open days left, or
//     the sessions the rhythm still allows, are needed for the missing rest days or quota
//     units, today has to fill one of them.
//  4. Quota maximums: no more demanding units and no more units of an activity than set.
//  5. Load limit: when the two weeks before had at least MinHistorySessions sessions, the week
//     may carry at most LoadGrowth times their average load (at least MinLoadCap).
//  6. Jumping: at most two units a week (the owner's quota may lower it) and never on two days
//     in a row; the day becomes a rest day when nothing else is left.
//  7. Variety: the activity of the day before (or of a unit fixed for the day after) is out when
//     two other activities remain.
//  8. Block: in the Entlastungswoche no demanding unit and the load capped at DeloadFactor times
//     the average of the three weeks before.
//  9. Return to work (Input.RestartFrom): week 1 only light units, weeks 2 and 3 nothing
//     demanding; a rest day when nothing light is left.
//  10. Young horses (under 6): units of at most YoungMaxMinutes, at most YoungMaxUnits a week.
//     Seniors (18 and older): at most one demanding unit a week (the demanding maximum of the
//     other horses is two, see DefaultDemandingMax).
const (
	RecoveryBelow      = 20.0
	DemandingFrom      = 45.0
	LoadGrowth         = 1.2
	MinLoadCap         = 90.0
	MinHistorySessions = 3
)

// Limits of groups 4 and 6 to 10.
const (
	DefaultDemandingMax = 2  // demanding units a week, an owner's quota may lower it
	SeniorDemandingMax  = 1  // demanding units a week for a horse of 18 and older
	JumpingMax          = 2  // jumping units a week
	YoungMaxMinutes     = 30 // minutes of a unit of a young horse
	YoungMaxUnits       = 4  // units a week of a young horse
	VarietyActivities   = 3  // different activities a week asked for when at least this many are allowed
)

// Unit levels (UnitLevel).
const (
	LevelRecovery  = "recovery"
	LevelLight     = "light"
	LevelNormal    = "normal"
	LevelDemanding = "demanding"
)

// UnitLevel classifies the load of one unit.
func UnitLevel(unitLoad float64) string {
	switch {
	case unitLoad < RecoveryBelow:
		return LevelRecovery
	case unitLoad < load.LightBelow:
		return LevelLight
	case unitLoad < DemandingFrom:
		return LevelNormal
	}
	return LevelDemanding
}

// LevelLabel is the German label of a unit level.
func LevelLabel(level string) string {
	switch level {
	case LevelRecovery:
		return "aktive Erholung"
	case LevelLight:
		return "leicht"
	case LevelNormal:
		return "normal"
	case LevelDemanding:
		return "fordernd"
	}
	return level
}

// recoveryActivities may be used for active recovery.
var recoveryActivities = map[training.Activity]bool{
	training.ActivityWalker: true, training.ActivityGroundwork: true,
	training.ActivityLunge: true, training.ActivityHack: true,
}

// Ahead describes the rest of the week when a whole week is planned (weekplan); nil in
// "Was heute?".
type Ahead struct {
	OpenDays int                // open days from Today to Sunday, Today included
	Units    []training.Session // units already fixed on later days of the week
	RestDays int                // rest days already fixed on later days of the week
}

type window struct{ lo, hi int }

func (w window) empty() bool { return w.lo > w.hi }

func intersect(a, b window) window {
	return window{max(a.lo, b.lo), min(a.hi, b.hi)}
}

// dayLimits are the limits of one day from the week structure.
type dayLimits struct {
	rest     string                       // non-empty: the day must be a rest day, why
	restNeed string                       // non-empty: a rest day fills a need of the week
	out      map[training.Activity]string // activities that are out, why
	win      map[training.Activity]window // minute window of the others
}

func unitLoad(a training.Activity, minutes int) float64 { return load.Score(minutes, a, 0) }

// UnitLoad is the planned load of a unit (no canter share).
func UnitLoad(a training.Activity, minutes int) float64 { return unitLoad(a, minutes) }

// LevelMinutes is the minute window in which a unit of a has the level, capped to the
// activity's sensible range; ok is false when the level is out of reach.
func LevelMinutes(a training.Activity, level string) (lo, hi int, ok bool) {
	dur := durationTable[a]
	w := intersect(levelWindow(a, level), window{dur.min, dur.max})
	return w.lo, w.hi, !w.empty()
}

// WeekLoadLimit is the load limit of the week starting on monday: LoadGrowth times the
// average of the two weeks before (at least MinLoadCap), or false when those weeks have
// fewer than MinHistorySessions sessions.
func WeekLoadLimit(recent []training.Session, monday time.Time) (float64, bool) {
	var sum float64
	n := 0
	for _, s := range recent {
		if d := training.DaysBetween(monday, s.Day); d >= -14 && d < 0 {
			sum += s.Load
			n++
		}
	}
	if n < MinHistorySessions {
		return 0, false
	}
	return math.Max(sum/2*LoadGrowth, MinLoadCap), true
}

// minutesBelow is the longest unit of a with a load below limit (-1 when none).
func minutesBelow(a training.Activity, limit float64) int {
	f := load.Factor(a, 0)
	if f <= 0 {
		return math.MaxInt32
	}
	m := int(math.Ceil(limit/f)) - 1
	for m >= 0 && float64(m)*f >= limit {
		m--
	}
	return m
}

// minutesFrom is the shortest unit of a with a load of at least limit.
func minutesFrom(a training.Activity, limit float64) int {
	f := load.Factor(a, 0)
	if f <= 0 {
		return math.MaxInt32
	}
	return int(math.Ceil(limit/f - 1e-9))
}

// levelWindow is the minute window in which a unit of a has the level.
func levelWindow(a training.Activity, level string) window {
	switch level {
	case LevelRecovery:
		if !recoveryActivities[a] {
			return window{1, 0}
		}
		return window{0, minutesBelow(a, RecoveryBelow)}
	case LevelLight:
		return window{0, minutesBelow(a, load.LightBelow)}
	case LevelNormal:
		return window{minutesFrom(a, load.LightBelow), minutesBelow(a, DemandingFrom)}
	case LevelDemanding:
		return window{minutesFrom(a, DemandingFrom), math.MaxInt32}
	}
	return window{0, math.MaxInt32}
}

var dayRuleLevels = map[string]string{
	training.DayRecovery: LevelRecovery, training.DayLight: LevelLight,
	training.DayNormal: LevelNormal, training.DayDemanding: LevelDemanding,
}

// weekState counts the units of the week: the sessions from Monday up to today and the units
// fixed on later days.
type weekState struct {
	monday            time.Time
	loadSoFar         float64
	units             int
	demanding         int
	recovery          int
	perActivity       map[training.Activity]int
	restDays          int     // past days of the week without a unit, plus fixed rest days ahead
	yesterday, twoAgo float64 // day loads
}

func newWeekState(in Input, today time.Time) weekState {
	w := weekState{monday: today.AddDate(0, 0, -training.WeekdayIndex(today)), perActivity: map[training.Activity]int{}}
	count := func(s training.Session) {
		w.units++
		w.loadSoFar += s.Load
		switch UnitLevel(s.Load) {
		case LevelDemanding:
			w.demanding++
		case LevelRecovery:
			w.recovery++
		}
		w.perActivity[s.Activity]++
	}
	daysWithUnits := map[int]bool{}
	for _, s := range in.Recent {
		d := training.DaysBetween(w.monday, s.Day)
		switch {
		case d >= 0 && training.DaysBetween(s.Day, today) >= 0:
			count(s)
			daysWithUnits[d] = true
		}
	}
	for d := 0; d < training.WeekdayIndex(today); d++ {
		if !daysWithUnits[d] {
			w.restDays++
		}
	}
	if a := in.Ahead; a != nil {
		for _, s := range a.Units {
			count(s)
		}
		w.restDays += a.RestDays
	}
	w.yesterday = load.DayLoad(in.Recent, today.AddDate(0, 0, -1))
	w.twoAgo = load.DayLoad(in.Recent, today.AddDate(0, 0, -2))
	return w
}

// activitiesOn returns the activities of the units on a day: the sessions of Recent and the units
// fixed on later days of a planned week.
func activitiesOn(in Input, day time.Time) map[training.Activity]bool {
	out := map[training.Activity]bool{}
	for _, s := range in.Recent {
		if training.DaysBetween(s.Day, day) == 0 {
			out[s.Activity] = true
		}
	}
	if in.Ahead != nil {
		for _, s := range in.Ahead.Units {
			if training.DaysBetween(s.Day, day) == 0 {
				out[s.Activity] = true
			}
		}
	}
	return out
}

// allowedActivities counts the activities the profile allows (on or conditional).
func allowedActivities(p training.Profile) int {
	n := 0
	for _, al := range p.Allowed {
		if al.Activity.Valid() && (al.Mode == training.ModeOn || al.Mode == training.ModeConditional) {
			n++
		}
	}
	return n
}

// demandingMax is the number of demanding units a week the horse may have: the owner's quota
// when it is lower than the default, at most DefaultDemandingMax, and for seniors one.
func demandingMax(q training.Quotas, ageYears int) int {
	m := DefaultDemandingMax
	if q.Demanding > 0 {
		m = min(q.Demanding, DefaultDemandingMax)
	}
	if training.Senior(ageYears) {
		m = min(m, SeniorDemandingMax)
	}
	return m
}

// jumpingMax is the number of jumping units a week: the owner's quota when it is lower than
// JumpingMax, else JumpingMax.
func jumpingMax(q training.Quotas) int {
	if n := q.Activities[training.ActivityJumping]; n > 0 {
		return min(n, JumpingMax)
	}
	return JumpingMax
}

// structureFor returns the limits of in.Today for the candidate activities; base is the
// minute window each activity already has from the safety rules and the rhythm.
func structureFor(in Input, today time.Time, base map[training.Activity]window) dayLimits {
	lim := dayLimits{out: map[training.Activity]string{}, win: map[training.Activity]window{}}
	for a, w := range base {
		lim.win[a] = w
	}
	r := in.Profile.Rhythm
	idx := training.WeekdayIndex(today)
	dayName := training.GermanWeekday(idx)
	rule := r.Days[idx]
	st := newWeekState(in, today)

	// apply narrows the windows by one group; it is dropped when nothing would be left.
	apply := func(group func(a training.Activity, w window) (window, string)) bool {
		next := map[training.Activity]window{}
		out := map[training.Activity]string{}
		left := false
		for a, w := range lim.win {
			nw, why := group(a, w)
			if why == "" && nw.empty() {
				why = "passt heute nicht"
			}
			if why != "" {
				out[a] = why
				continue
			}
			next[a] = nw
			left = true
		}
		if !left {
			return false
		}
		lim.win = next
		for a, why := range out {
			lim.out[a] = why
		}
		return true
	}

	// 1. The owner's rule for the weekday.
	switch rule.Kind {
	case training.DayRest:
		lim.rest = fmt.Sprintf("Laut Wochenstruktur ist %s Ruhetag.", dayName)
		return lim
	case training.DayActivity:
		apply(func(a training.Activity, w window) (window, string) {
			if a != rule.Activity {
				return w, fmt.Sprintf("Laut Wochenstruktur ist %s für %s vorgesehen.", dayName, rule.Activity.GermanName())
			}
			return w, ""
		})
	case training.DayRecovery, training.DayLight, training.DayNormal, training.DayDemanding:
		level := dayRuleLevels[rule.Kind]
		apply(func(a training.Activity, w window) (window, string) {
			nw := intersect(w, levelWindow(a, level))
			if nw.empty() {
				return nw, fmt.Sprintf("Laut Wochenstruktur ist %s für %s vorgesehen; %s passt dafür nicht.", dayName, dayKindText(rule.Kind), a.GermanName())
			}
			return nw, ""
		})
	default:
		// 2. Default rules.
		if UnitLevel(st.yesterday) == LevelDemanding {
			apply(func(a training.Activity, w window) (window, string) {
				nw := intersect(w, window{0, minutesBelow(a, DemandingFrom)})
				if nw.empty() {
					return nw, "Gestern war fordernd – heute kein fordernder Tag."
				}
				return nw, ""
			})
		}
		if st.yesterday >= load.LightBelow && st.twoAgo >= load.LightBelow {
			apply(func(a training.Activity, w window) (window, string) {
				nw := intersect(w, window{0, minutesBelow(a, load.LightBelow)})
				if nw.empty() {
					return nw, "Nach zwei Arbeitstagen in Folge heute nur leicht."
				}
				return nw, ""
			})
		}
	}

	q := r.Quotas
	// 3. Needs of the week.
	if a := in.Ahead; a != nil && a.OpenDays > 0 {
		restNeed := r.RestDaysMin - st.restDays
		demandNeed := min(q.Demanding, demandingMax(q, in.AgeYears)) - st.demanding
		recNeed := q.Recovery - st.recovery
		actNeed := map[training.Activity]int{}
		units := max(demandNeed, 0) + max(recNeed, 0)
		for act, n := range q.Activities {
			if act == training.ActivityJumping {
				n = min(n, JumpingMax)
			}
			if need := n - st.perActivity[act]; need > 0 {
				actNeed[act] = need
				units += need
			}
		}
		// Variety is a need like the quotas: with three or more activities allowed the week
		// has at least three different ones. Activities a quota still asks for count as used.
		varietyNeed := 0
		if allowedActivities(in.Profile) >= VarietyActivities {
			used := 0
			for _, n := range st.perActivity {
				if n > 0 {
					used++
				}
			}
			for act := range actNeed {
				if st.perActivity[act] == 0 {
					used++
				}
			}
			varietyNeed = max(VarietyActivities-used, 0)
			units += varietyNeed
		}
		// Open days left for units: without the missing rest days and within the rhythm's
		// maximum of sessions.
		slots := a.OpenDays - max(restNeed, 0)
		if r.SessionsMax > 0 {
			slots = min(slots, r.SessionsMax-st.units)
		}
		switch {
		case restNeed > 0 && restNeed >= a.OpenDays:
			lim.rest = "Diese Woche fehlt noch ein Ruhetag."
			return lim
		case units > 0 && units >= slots:
			missing := needsText(demandNeed, recNeed, actNeed, varietyNeed)
			if restNeed > 0 && rule.Kind == training.DayFree {
				lim.restNeed = "Diese Woche fehlt noch ein Ruhetag."
			}
			apply(func(act training.Activity, w window) (window, string) {
				switch {
				case actNeed[act] > 0:
					return w, ""
				case demandNeed > 0:
					if nw := intersect(w, levelWindow(act, LevelDemanding)); !nw.empty() {
						return nw, ""
					}
				}
				if recNeed > 0 {
					if nw := intersect(w, levelWindow(act, LevelRecovery)); !nw.empty() {
						return nw, ""
					}
				}
				if varietyNeed > 0 && st.perActivity[act] == 0 {
					return w, ""
				}
				return w, "Diese Woche fehlen noch: " + missing + "."
			})
		}
	}

	// 4. Quota maximums: demanding units (at most two, fewer with a quota, one for seniors) and
	// the quota of an activity.
	if max := demandingMax(q, in.AgeYears); st.demanding >= max {
		why := fmt.Sprintf("Diese Woche gab es schon %d fordernde Einheiten.", st.demanding)
		if training.Senior(in.AgeYears) && max == SeniorDemandingMax {
			why = "Pferde ab 18 Jahren bekommen höchstens eine fordernde Einheit pro Woche."
		}
		apply(func(a training.Activity, w window) (window, string) {
			nw := intersect(w, window{0, minutesBelow(a, DemandingFrom)})
			if nw.empty() {
				return nw, why
			}
			return nw, ""
		})
	}
	if len(q.Activities) > 0 {
		apply(func(a training.Activity, w window) (window, string) {
			if n := q.Activities[a]; n > 0 && st.perActivity[a] >= n {
				return w, fmt.Sprintf("%s war diese Woche schon %d×.", a.GermanName(), st.perActivity[a])
			}
			return w, ""
		})
	}

	// 5. Load limit.
	if limit, ok := WeekLoadLimit(in.Recent, st.monday); ok {
		remaining := limit - st.loadSoFar
		why := fmt.Sprintf("Die Wochenlast läge sonst mehr als %d %% über dem Schnitt der beiden Vorwochen.", int(math.Round((LoadGrowth-1)*100)))
		if !apply(func(a training.Activity, w window) (window, string) {
			f := load.Factor(a, 0)
			hi := math.MaxInt32
			if f > 0 {
				hi = int(math.Floor(remaining/f + 1e-9))
			}
			nw := intersect(w, window{0, hi})
			if nw.empty() || nw.hi <= 0 {
				return window{1, 0}, why
			}
			return nw, ""
		}) {
			lim.rest = why
		}
	}
	if lim.rest != "" {
		return lim
	}

	// 6. Jumping: at most JumpingMax units a week and never on two days in a row; hard rules, a
	// day with nothing else left becomes a rest day.
	if _, has := lim.win[training.ActivityJumping]; has {
		why := ""
		switch {
		case activitiesOn(in, today.AddDate(0, 0, -1))[training.ActivityJumping] || activitiesOn(in, today.AddDate(0, 0, 1))[training.ActivityJumping]:
			why = "Springen nicht an zwei Tagen hintereinander"
		case st.perActivity[training.ActivityJumping] >= jumpingMax(q):
			why = fmt.Sprintf("Springen höchstens %d× pro Woche", jumpingMax(q))
		}
		if why != "" && !apply(func(a training.Activity, w window) (window, string) {
			if a == training.ActivityJumping {
				return w, why + "."
			}
			return w, ""
		}) {
			lim.rest = why + " – heute Ruhetag."
			return lim
		}
	}

	// 7. Variety: the activity of the day before, or of a unit fixed for the day after, is out
	// when two other activities remain.
	before, after := activitiesOn(in, today.AddDate(0, 0, -1)), activitiesOn(in, today.AddDate(0, 0, 1))
	if len(before) > 0 || len(after) > 0 {
		others := 0
		for a := range lim.win {
			if !before[a] && !after[a] {
				others++
			}
		}
		if others >= 2 {
			apply(func(a training.Activity, w window) (window, string) {
				switch {
				case before[a]:
					return w, fmt.Sprintf("%s war gestern schon dran – heute etwas anderes.", a.GermanName())
				case after[a]:
					return w, fmt.Sprintf("%s ist morgen schon eingetragen – heute etwas anderes.", a.GermanName())
				}
				return w, ""
			})
		}
	}

	// 8. Block: the Entlastungswoche has no demanding unit and a lower load.
	if in.Block.Deload {
		apply(func(a training.Activity, w window) (window, string) {
			nw := intersect(w, window{0, minutesBelow(a, DemandingFrom)})
			if nw.empty() {
				return nw, "Entlastungswoche: diese Woche nichts Forderndes."
			}
			return nw, ""
		})
		if limit, ok := DeloadLoadLimit(in.Recent, st.monday); ok {
			remaining := limit - st.loadSoFar
			why := fmt.Sprintf("Entlastungswoche: die Wochenlast bleibt unter %d %% des Schnitts der drei Vorwochen.", int(math.Round(DeloadFactor*100)))
			if !apply(func(a training.Activity, w window) (window, string) {
				f := load.Factor(a, 0)
				hi := math.MaxInt32
				if f > 0 {
					hi = int(math.Floor(remaining/f + 1e-9))
				}
				nw := intersect(w, window{0, hi})
				if nw.empty() || nw.hi <= 0 {
					return window{1, 0}, why
				}
				return nw, ""
			}) {
				lim.rest = why
				return lim
			}
		}
	}

	// 9. Return to work: light units in the first week, nothing demanding in the next two.
	switch wk := RestartWeek(in.RestartFrom, today); wk {
	case 1:
		why := "Wiederantrainieren, Woche 1: nur leichte Einheiten."
		if !apply(func(a training.Activity, w window) (window, string) {
			nw := intersect(w, window{0, minutesBelow(a, load.LightBelow)})
			if nw.empty() {
				return nw, why
			}
			return nw, ""
		}) {
			lim.rest = "Wiederantrainieren, Woche 1: nur leichte Einheiten – dafür ist heute nichts freigegeben."
			return lim
		}
	case 2, 3:
		apply(func(a training.Activity, w window) (window, string) {
			nw := intersect(w, window{0, minutesBelow(a, DemandingFrom)})
			if nw.empty() {
				return nw, fmt.Sprintf("Wiederantrainieren, Woche %d: nichts Forderndes.", wk)
			}
			return nw, ""
		})
	}

	// 10. Young horses: short units and few of them.
	if training.Young(in.AgeYears) {
		if st.units >= YoungMaxUnits {
			lim.rest = fmt.Sprintf("Junge Pferde bekommen höchstens %d Einheiten pro Woche – heute Ruhetag.", YoungMaxUnits)
			return lim
		}
		capText := fmt.Sprintf("Junge Pferde trainieren höchstens %d Minuten", YoungMaxMinutes)
		why := capText + "."
		if !apply(func(a training.Activity, w window) (window, string) {
			nw := intersect(w, window{0, YoungMaxMinutes})
			if nw.empty() {
				return nw, why
			}
			return nw, ""
		}) {
			// Nothing short is left of the day's windows (an owner's demanding day): the cap
			// is a protection of the horse and wins over the rule for the day.
			next := map[training.Activity]window{}
			for a := range lim.win {
				if w := intersect(base[a], window{0, YoungMaxMinutes}); !w.empty() {
					next[a] = w
				} else {
					lim.out[a] = why
				}
			}
			if len(next) == 0 {
				lim.rest = capText + " – dafür ist heute nichts freigegeben."
				return lim
			}
			lim.win = next
		}
	}

	// Only narrowed windows are limits; the base window stays the caller's business (the
	// recommender may still cut a unit below its minimum for the available time).
	for a, w := range lim.win {
		if w == base[a] {
			delete(lim.win, a)
		}
	}
	return lim
}

func dayKindText(kind string) string {
	switch kind {
	case training.DayRecovery:
		return "aktive Erholung"
	case training.DayLight:
		return "eine leichte Einheit"
	case training.DayNormal:
		return "eine normale Einheit"
	case training.DayDemanding:
		return "eine fordernde Einheit"
	}
	return kind
}

func needsText(demand, recovery int, acts map[training.Activity]int, variety int) string {
	var parts []string
	if demand > 0 {
		parts = append(parts, fmt.Sprintf("%d× fordernd", demand))
	}
	if recovery > 0 {
		parts = append(parts, fmt.Sprintf("%d× aktive Erholung", recovery))
	}
	var names []string
	for a, n := range acts {
		names = append(names, fmt.Sprintf("%d× %s", n, a.GermanName()))
	}
	sort.Strings(names)
	parts = append(parts, names...)
	switch {
	case variety == 1:
		parts = append(parts, "1 weitere Aktivität")
	case variety > 1:
		parts = append(parts, fmt.Sprintf("%d weitere Aktivitäten", variety))
	}
	return strings.Join(parts, ", ")
}
