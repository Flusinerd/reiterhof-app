package recommend

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// The unit plan (docs/domains/training-planning.md, "Planner"): every unit that is not a rest day
// or a walker unit has one focus and three parts, Lösen, Arbeiten and Entspannen, each with at
// most one exercise.
//
// PlanUnit is pure and deterministic like Recommend: it picks the focus by the rotation below and
// then the exercises by their scores.
//
// # Focus rotation
//
// The candidates are the focuses of the person who rides (at most two), then those of the horse in
// priority order, that fit the activity. Focuses with a date in the past are ignored; a focus above
// the horse's phase is dropped when it was trained or planned in the last seven days. When a show is
// within 14 days and turnier is not among the focuses, it is added as the last candidate for hall,
// arena and jumping units, at most twice a week. Within seven days before a show the rotation
// shrinks to the horse's first focus and turnier. The candidate that was trained or planned longest
// ago wins (never is the longest), ties go to the earlier one. Without a candidate an activity has
// a fallback focus.
//
// # Exercise pick
//
// Per part the candidates are the exercises of the activity's libraries with the right phase, a
// level at or below the horse's (one above when all its prerequisites are mastered), all
// prerequisites mastered, no cavaletti the day after cavaletti, in the week before a show only
// known exercises, and the facility equipment of the unit's location. The highest score wins, ties
// go by library (the home library first), progression order and title.

// Scores of the exercise pick.
const (
	PickWeightTag           = 3.0  // Arbeiten: per tag shared with the focus
	PickWeightNext          = 2.0  // Arbeiten: follow-up of a mastered exercise
	PickWeightRetryHard     = 2.0  // Arbeiten: rated "Schwer" 3 to 7 days ago
	PickWeightBetter        = 1.5  // Arbeiten: rated "Besser" in the last 14 days
	PickWeightMastered      = -2.0 // Arbeiten: mastered
	PickWeightUsedRecently  = -3.0 // Arbeiten: used in the last three days
	PickWeightIntensity     = -2.0 // Arbeiten: demanding on a light unit, light on a demanding one
	PickWeightGoal          = 1.5  // Arbeiten: a lesson of the goal level that is not mastered
	PickWeightSuits         = 1.0  // Lösen and Arbeiten: the exercise suits a young or senior horse
	PickWeightWarmupTag     = 2.0  // Lösen: per tag of losgelassenheit, takt, mobilitaet
	PickWeightUsedYesterday = -3.0 // Lösen: used yesterday
	PickWeightCooldownTag   = 2.0  // Entspannen: per tag of losgelassenheit, mobilitaet
	PickWeightCooldown      = 1.0  // Entspannen: a cool-down exercise over one for any part
)

// Windows of the rotation and the pick, in days.
const (
	focusLookbackDays   = 14 // how far back the focus of a unit counts
	focusAboveWeekDays  = 7  // a focus above the horse's phase is planned at most once in this time
	showFocusDays       = 7  // before a show the rotation shrinks to the first focus
	showTurnierDays     = 14 // within this time before a show turnier is added
	turnierPerWeek      = 2  // automatic turnier units a week
	retryHardFrom       = 3  // rated "Schwer" from this many days ago
	retryHardTo         = 7  // up to this many days ago
	betterWithinDays    = 14 // rated "Besser" within this many days
	usedRecentlyDays    = 3  // used within this many days up to and including the day
	goalUnitsPerWeek    = 2  // units a week with a main exercise of the goal level
	neverDays           = 1 << 30
	maxExerciseTitleLen = 60
)

var warmupTags = []string{"losgelassenheit", "takt", "mobilitaet"}
var cooldownTags = []string{"losgelassenheit", "mobilitaet"}

// Part is one part of a unit.
type Part struct {
	// Exercise is nil for free work. Minutes is the length of the part.
	Exercise *training.Exercise
	Minutes  int
	// Why says in German why a rule chose this exercise (set-up, level goal); empty otherwise.
	Why string
	// InsteadOf is the exercise that fitted best but cannot be set up in the unit's location.
	InsteadOf *training.Exercise
}

// Unit is the plan of one unit: its focus and the three parts. A part without minutes is absent.
type Unit struct {
	Warmup, Main, Cooldown Part
	// FocusKey is the catalog key of the focus, FocusWhy the German reason for it.
	FocusKey string
	FocusWhy string
	// EquipmentNote says in German that the unit is planned without set-up (a shared location, no
	// set-up that day), or what was left out for lack of equipment.
	EquipmentNote string
	// Replaced says why an exercise proposed by the language model was replaced (Hints).
	Replaced string
}

// Empty reports whether the unit has no plan (rest, walker).
func (u Unit) Empty() bool { return u.FocusKey == "" }

// Primary is the exercise older clients show for the day: the main one, else the first of the
// others.
func (u Unit) Primary() *training.Exercise {
	for _, p := range []Part{u.Main, u.Warmup, u.Cooldown} {
		if p.Exercise != nil {
			return p.Exercise
		}
	}
	return nil
}

// Exercises lists the exercises of the unit, main first.
func (u Unit) Exercises() []string {
	var out []string
	for _, p := range []Part{u.Main, u.Warmup, u.Cooldown} {
		if p.Exercise != nil {
			out = append(out, p.Exercise.ID)
		}
	}
	return out
}

// Hints are the proposals of the language model for a unit: a focus key and an exercise per part.
// PlanUnit takes a proposal when it is allowed and picks by the rules otherwise.
type Hints struct {
	FocusKey               string
	Warmup, Main, Cooldown *training.Exercise
}

// hasUnit reports whether units of the activity have a unit plan.
func hasUnit(a training.Activity) bool {
	switch a {
	case training.ActivityHall, training.ActivityArena, training.ActivityJumping, training.ActivityLunge,
		training.ActivityGroundwork, training.ActivityHack:
		return true
	}
	return false
}

// planner is what PlanUnit works with for one unit.
type planner struct {
	in      Input
	act     training.Activity
	day     time.Time
	minutes int
	phase   training.HorsePhase
	level   int // the horse's level rank, 0 when unknown
	goal    string
	senior  bool
	cold    bool
	libs    []string
	avail   []string
	restart int

	showDays int // days until the next show, -1 none
	show     training.Show

	exByID   map[string]training.Exercise
	mastered map[string]bool
	nextOf   map[string]bool // exercises that follow a mastered one
	uses     map[string][]use
	// yesterdayCavaletti: a cavaletti exercise was used yesterday.
	yesterdayCavaletti bool
	// aboveOK: exercises one level above the horse's are allowed (not in an Entlastungswoche or a
	// return to work, not for young horses or a status other than fit); goalUnits counts the units
	// of the week with a main exercise of the goal level so far.
	aboveOK   bool
	goalUnits int
}

// use is a use of an exercise in a session of the last days.
type use struct {
	daysAgo int
	rating  int // focus rating of the session when the exercise was its logged one, else 0
}

func newPlanner(in Input, act training.Activity, minutes int) *planner {
	day := training.Day(in.Today)
	if minutes <= 0 {
		minutes = DefaultMinutes(act)
	}
	p := &planner{in: in, act: act, day: day, minutes: minutes, goal: training.GoalLevel(in.Level, in.Profile.GoalLevel)}
	p.phase = training.Phase(in.Level, in.AgeYears, in.Profile.Status)
	p.level = max(training.LevelRank(in.Level), 0)
	p.senior = training.Senior(in.AgeYears)
	p.cold = in.Weather != nil && in.Weather.TempC < 5
	p.libs = training.ExerciseLibraries(act, in.Profile.Discipline)
	if training.IsLocation(act) {
		p.avail = training.Available(in.Equipment.Of(act), in.EquipmentOK)
	}
	p.restart = RestartWeek(in.RestartFrom, day)
	p.showDays, p.show = nextShowOn(in.Profile.Shows, day)
	p.aboveOK = !in.Block.Deload && p.restart == 0 && !training.Young(in.AgeYears) && in.Profile.Status == training.StatusFit

	p.exByID = map[string]training.Exercise{}
	p.mastered = map[string]bool{}
	for _, ex := range in.Exercises {
		p.exByID[ex.ID] = ex
		if ex.Mastered {
			p.mastered[ex.ID] = true
		}
	}
	p.nextOf = map[string]bool{}
	for _, ex := range in.Exercises {
		if ex.Mastered && ex.NextID != "" {
			p.nextOf[ex.NextID] = true
		}
	}
	p.uses = map[string][]use{}
	monday := day.AddDate(0, 0, -training.WeekdayIndex(day))
	for _, s := range in.Recent {
		d := training.DaysBetween(s.Day, day)
		if d < 0 || d >= focusLookbackDays {
			continue
		}
		for _, id := range s.UsedExercises() {
			u := use{daysAgo: d}
			if id == s.ExerciseID {
				u.rating = s.FocusRating
			}
			p.uses[id] = append(p.uses[id], u)
			if d == 1 && p.exByID[id].Library == training.LibraryCavaletti {
				p.yesterdayCavaletti = true
			}
		}
		if s.ExerciseID != "" && p.goal != "" && training.DaysBetween(monday, s.Day) >= 0 && p.exByID[s.ExerciseID].Level == p.goal {
			p.goalUnits++
		}
	}
	return p
}

// nextShowOn returns the days until the next show on or after day and the show; -1 without one.
func nextShowOn(shows []training.Show, day time.Time) (int, training.Show) {
	best, found := -1, training.Show{}
	for _, s := range shows {
		if d := training.DaysBetween(day, s.Date); d >= 0 && (best < 0 || d < best) {
			best, found = d, s
		}
	}
	return best, found
}

// PlanUnit plans the focus and the three parts of a unit of activity act with the given minutes
// for in.Today. in carries the horse and the history as for Recommend plus the exercises of the
// library, the rider's focuses and the equipment of the stable. The result is empty for activities
// without a unit plan (walker, rest). Hints are the proposals of a language model; what is not
// allowed is replaced by the rules (Unit.Replaced says why).
func PlanUnit(in Input, act training.Activity, minutes int, hints Hints) Unit {
	if !hasUnit(act) {
		return Unit{}
	}
	p := newPlanner(in, act, minutes)
	focus := p.chooseFocus(hints.FocusKey)
	f, _ := training.FocusByKey(focus.Key)
	u := Unit{FocusKey: focus.Key, FocusWhy: focus.Why}
	split := training.UnitSplit(act, p.minutes, p.senior, p.cold)
	level := UnitLevel(unitLoad(act, p.minutes))
	picked := map[string]bool{}
	var replaced []string
	var notes []string
	set := func(part string, dst *Part, mins int, hint *training.Exercise) {
		dst.Minutes = mins
		if mins <= 0 {
			return
		}
		pk := p.pickPart(part, f, level, picked, hint)
		if pk.replaced != "" {
			replaced = append(replaced, pk.replaced)
		}
		dst.Exercise, dst.InsteadOf, dst.Why = pk.ex, pk.insteadOf, pk.why
		if pk.ex != nil {
			picked[pk.ex.ID] = true
		}
		if pk.insteadOf != nil && pk.why != "" {
			notes = append(notes, pk.why)
		}
	}
	set(training.PartMain, &u.Main, split.Main, hints.Main)
	set(training.PartWarmup, &u.Warmup, split.Warmup, hints.Warmup)
	set(training.PartCooldown, &u.Cooldown, split.Cooldown, hints.Cooldown)
	u.EquipmentNote = EquipmentNote(act, in.Equipment, in.EquipmentOK)
	if u.EquipmentNote == "" && len(notes) > 0 {
		u.EquipmentNote = notes[0]
	}
	u.Replaced = strings.Join(replaced, " ")
	return u
}

// EquipmentNote says in German that a unit is planned without set-up: when its location is shared
// and the day does not say that set-up is possible, or the day says there is none. "" otherwise
// (and for activities without a location).
func EquipmentNote(a training.Activity, cfg training.StableEquipment, equipmentOK *bool) string {
	if !training.IsLocation(a) {
		return ""
	}
	switch {
	case equipmentOK != nil && !*equipmentOK:
		return "Heute ohne Aufbau geplant."
	case equipmentOK == nil && cfg.Of(a).Shared:
		name, _ := training.LocationName(a)
		return name + " geteilt: ohne Aufbau geplant."
	}
	return ""
}

// --- focus ----------------------------------------------------------------------------------------

// FocusPick is the focus of a unit with the German reason.
type FocusPick struct {
	Key string
	Why string
}

type focusCand struct {
	key   string
	rider bool // a focus of the person who rides
}

// lastTrained returns how many days ago the focus was trained or planned (within the lookback),
// false when it was not.
func (p *planner) lastTrained(key string) (int, bool) {
	best, ok := 0, false
	for _, s := range p.in.Recent {
		if s.FocusKey != key {
			continue
		}
		if d := training.DaysBetween(s.Day, p.day); d >= 0 && d < focusLookbackDays && (!ok || d < best) {
			best, ok = d, true
		}
	}
	return best, ok
}

// turnierThisWeek counts the units of the week up to the day that had turnier as their focus.
func (p *planner) turnierThisWeek() int {
	monday := p.day.AddDate(0, 0, -training.WeekdayIndex(p.day))
	n := 0
	for _, s := range p.in.Recent {
		if s.FocusKey == training.FocusTurnier && training.DaysBetween(monday, s.Day) >= 0 && training.DaysBetween(s.Day, p.day) >= 0 {
			n++
		}
	}
	return n
}

// horseFocuses returns the keys of the horse's focuses that still apply, in priority order.
func (p *planner) horseFocuses() []string {
	var out []string
	for _, c := range p.in.Profile.Focuses {
		if c.Active(p.day) && training.ValidFocusKey(c.Key) && !slices.Contains(out, c.Key) {
			out = append(out, c.Key)
		}
	}
	return out
}

// candidates returns the focuses the rotation chooses from; pinned says that a show is near and the
// rotation is down to the first focus (and turnier).
func (p *planner) candidates() (cands []focusCand, pinned bool) {
	horse := p.horseFocuses()
	var rider []string
	for _, k := range p.in.RiderFocuses {
		if training.ValidFocusKey(k) && !slices.Contains(rider, k) {
			rider = append(rider, k)
		}
	}
	pinned = p.showDays >= 0 && p.showDays <= showFocusDays
	var list []focusCand
	if pinned {
		if len(horse) > 0 {
			list = append(list, focusCand{key: horse[0]})
			if slices.Contains(horse, training.FocusTurnier) && horse[0] != training.FocusTurnier {
				list = append(list, focusCand{key: training.FocusTurnier})
			}
		}
	} else {
		for _, k := range rider {
			list = append(list, focusCand{key: k, rider: true})
		}
		for _, k := range horse {
			if !slices.Contains(rider, k) {
				list = append(list, focusCand{key: k})
			}
		}
	}
	hasTurnier := slices.Contains(horse, training.FocusTurnier) || slices.Contains(rider, training.FocusTurnier)
	for _, c := range list {
		f, _ := training.FocusByKey(c.key)
		if !f.FitsActivity(p.act) {
			continue
		}
		// A focus above the horse's phase is planned at most once a week.
		if !f.AllowedIn(p.phase) {
			if d, ok := p.lastTrained(c.key); ok && d <= focusAboveWeekDays {
				continue
			}
		}
		cands = append(cands, c)
	}
	// Shortly before a show turnier is a candidate by itself, last in the list.
	if p.showDays >= 0 && p.showDays <= showTurnierDays && !hasTurnier && autoTurnier(p.act) && p.turnierThisWeek() < turnierPerWeek {
		cands = append(cands, focusCand{key: training.FocusTurnier})
	}
	return cands, pinned
}

// autoTurnier reports whether the planner adds turnier by itself for the activity.
func autoTurnier(a training.Activity) bool {
	return a == training.ActivityHall || a == training.ActivityArena || a == training.ActivityJumping
}

// chooseFocus picks the focus of the unit; hint is the model's proposal ("" for none), taken when
// it is one of the candidates.
func (p *planner) chooseFocus(hint string) FocusPick {
	cands, pinned := p.candidates()
	if len(cands) == 0 {
		return p.fallbackFocus()
	}
	chosen := -1
	if hint != "" {
		chosen = slices.IndexFunc(cands, func(c focusCand) bool { return c.key == hint })
	}
	if chosen < 0 {
		bestDays := -1
		for i, c := range cands {
			days := neverDays
			if d, ok := p.lastTrained(c.key); ok {
				days = d
			}
			if days > bestDays {
				chosen, bestDays = i, days
			}
		}
	}
	c := cands[chosen]
	f, _ := training.FocusByKey(c.key)
	days, trained := p.lastTrained(c.key)
	switch {
	case c.key == training.FocusTurnier && p.showDays >= 0 && p.showDays <= showTurnierDays:
		return FocusPick{c.key, fmt.Sprintf("Turnier am %s: Aufgabe und Parcours üben.", p.show.Date.Format("02.01."))}
	case c.rider:
		return FocusPick{c.key, fmt.Sprintf("%s Schwerpunkt %s.", possessive(p.in.RiderName), f.Label)}
	case pinned:
		return FocusPick{c.key, fmt.Sprintf("Vor dem Turnier bleibt es beim Schwerpunkt %s.", f.Label)}
	}
	return FocusPick{c.key, fmt.Sprintf("Schwerpunkt %s: %s.", f.Label, p.lastText(days, trained))}
}

// lastText says when a focus was last trained: "zuletzt am Montag", "noch nicht geübt".
func (p *planner) lastText(daysAgo int, trained bool) string {
	switch {
	case !trained:
		return "noch nicht geübt"
	case daysAgo == 0:
		return "heute schon geübt"
	case daysAgo == 1:
		return "zuletzt gestern"
	case daysAgo < 7:
		return "zuletzt am " + training.GermanWeekday(training.WeekdayIndex(p.day.AddDate(0, 0, -daysAgo)))
	}
	// The sentence adds the final full stop to the date ("zuletzt am 22.09.").
	return "zuletzt am " + p.day.AddDate(0, 0, -daysAgo).Format("02.01")
}

// possessive is the genitive of a first name for "Mias Schwerpunkt", "Lukas' Schwerpunkt".
func possessive(name string) string {
	// Only the first name counts, a full name is shortened.
	name = strings.TrimSpace(name)
	if f := strings.Fields(name); len(f) > 0 {
		name = f[0]
	}
	switch {
	case name == "":
		return "Eigener"
	case strings.HasSuffix(name, "s"), strings.HasSuffix(name, "ß"), strings.HasSuffix(name, "x"), strings.HasSuffix(name, "z"):
		return name + "'"
	}
	return name + "s"
}

// fallbackFocus is the focus of an activity when no focus of the profile fits: Losgelassenheit comes
// first for riding and lunging, Vertrauen for groundwork, Gelände (when the horse has it) or
// Kondition for hacks, Springen for jumping.
func (p *planner) fallbackFocus() FocusPick {
	var key, tail string
	switch p.act {
	case training.ActivityGroundwork:
		key, tail = "vertrauen", "Vertrauen ist die Basis der Bodenarbeit."
	case training.ActivityJumping:
		key, tail = "springen", "Springgymnastik steht beim Springen im Mittelpunkt."
	case training.ActivityHack:
		if slices.Contains(p.horseFocuses(), "gelaende") {
			key, tail = "gelaende", "Gelände passt zum Ausritt."
		} else {
			key, tail = "kondition", "Kondition passt zum Ausritt."
		}
	default:
		key, tail = "losgelassenheit", "Losgelassenheit kommt immer zuerst."
	}
	head := "Ohne Schwerpunkt im Profil: "
	if len(p.horseFocuses()) > 0 || len(p.in.RiderFocuses) > 0 {
		head = fmt.Sprintf("Kein Schwerpunkt im Profil passt %s: ", dative(p.act))
	}
	return FocusPick{key, head + tail}
}

// dative is "zur Halle", "zum Ausritt" for the texts about the focus.
func dative(a training.Activity) string {
	switch a {
	case training.ActivityHall:
		return "zur Halle"
	case training.ActivityArena:
		return "zum Platz"
	case training.ActivityLunge:
		return "zur Longe"
	case training.ActivityJumping:
		return "zum Springen"
	case training.ActivityGroundwork:
		return "zur Bodenarbeit"
	case training.ActivityHack:
		return "zum Ausritt"
	}
	return "zu " + a.GermanName()
}

// --- exercises ------------------------------------------------------------------------------------

// candidate tests an exercise for a part of the unit. It returns "" when the exercise is allowed and
// else a short German reason why not ("braucht Stangen"). ignoreEquipment leaves the set-up out.
func (p *planner) candidate(ex training.Exercise, part string, ignoreEquipment bool) string {
	switch {
	case !slices.Contains(p.libs, ex.Library):
		return "gehört nicht zu dieser Aktivität"
	case !ex.FitsPart(part):
		return "gehört nicht in diesen Teil der Einheit"
	}
	rank := max(training.LevelRank(ex.Level), 0)
	switch {
	case rank > p.level+1:
		return "ist für das Pferd zu schwer"
	case rank == p.level+1 && (!p.aboveOK || (part == training.PartMain && p.goalUnits >= goalUnitsPerWeek)):
		return "passt gerade nicht zum Trainingsstand"
	}
	for _, id := range ex.Requires {
		if !p.mastered[id] {
			return "setzt Übungen voraus, die noch nicht sitzen"
		}
	}
	switch {
	case ex.Library == training.LibraryCavaletti && p.yesterdayCavaletti:
		return "Cavaletti gab es gestern schon"
	case p.showDays >= 0 && p.showDays <= showFocusDays && !ex.Rated && !ex.HasTag(training.FocusTurnier):
		return "vor dem Turnier nur bekannte Übungen"
	}
	if !ignoreEquipment {
		if missing := training.MissingEquipment(ex, p.avail); len(missing) > 0 {
			labels := make([]string, len(missing))
			for i, k := range missing {
				labels[i] = training.EquipmentLabel(k)
			}
			return "braucht " + training.JoinGerman(labels)
		}
	}
	return ""
}

// score rates an exercise for a part of a unit of the given level.
func (p *planner) score(ex training.Exercise, part string, focus training.Focus, unitLevel string) float64 {
	var s float64
	if ex.SuitsHorse(p.in.AgeYears) {
		s += PickWeightSuits
	}
	switch part {
	case training.PartMain:
		s += PickWeightTag * float64(ex.SharedTags(focus.Tags))
		if p.nextOf[ex.ID] {
			s += PickWeightNext
		}
		hard, better, recent := false, false, false
		for _, u := range p.uses[ex.ID] {
			hard = hard || (u.rating == 1 && u.daysAgo >= retryHardFrom && u.daysAgo <= retryHardTo)
			better = better || (u.rating == 2 && u.daysAgo < betterWithinDays)
			recent = recent || u.daysAgo < usedRecentlyDays
		}
		if hard {
			s += PickWeightRetryHard
		}
		if better {
			s += PickWeightBetter
		}
		if recent {
			s += PickWeightUsedRecently
		}
		if ex.Mastered {
			s += PickWeightMastered
		}
		switch {
		case ex.Intensity == training.ExerciseDemanding && (unitLevel == LevelLight || unitLevel == LevelRecovery),
			ex.Intensity == training.ExerciseLight && unitLevel == LevelDemanding:
			s += PickWeightIntensity
		}
		if p.goalBoost(ex) {
			s += PickWeightGoal
		}
	case training.PartWarmup:
		s += PickWeightWarmupTag * float64(countTags(ex, warmupTags))
		for _, u := range p.uses[ex.ID] {
			if u.daysAgo == 1 {
				s += PickWeightUsedYesterday
				break
			}
		}
	case training.PartCooldown:
		s += PickWeightCooldownTag * float64(countTags(ex, cooldownTags))
		if ex.Phase == training.PartCooldown {
			s += PickWeightCooldown
		}
	}
	return s
}

func countTags(ex training.Exercise, tags []string) int {
	n := 0
	for _, t := range tags {
		if ex.HasTag(t) {
			n++
		}
	}
	return n
}

// goalBoost reports whether the exercise is a lesson of the goal level that is not mastered yet,
// in a week where the horse may work on it (see planner.aboveOK).
func (p *planner) goalBoost(ex training.Exercise) bool {
	return p.goal != "" && p.aboveOK && ex.Level == p.goal && !ex.Mastered && p.goalUnits < goalUnitsPerWeek
}

type pick struct {
	ex        *training.Exercise
	insteadOf *training.Exercise
	why       string
	replaced  string
}

// libRank orders the libraries of the activity, the home library first.
func (p *planner) libRank(library string) int {
	if i := slices.Index(p.libs, library); i >= 0 {
		return i
	}
	return len(p.libs)
}

// best returns the allowed exercise with the highest score; nil without one.
func (p *planner) best(part string, focus training.Focus, unitLevel string, picked map[string]bool, ignoreEquipment bool) *training.Exercise {
	type scored struct {
		i     int
		score float64
		tags  int
	}
	var all []scored
	for i, ex := range p.in.Exercises {
		if picked[ex.ID] || p.candidate(ex, part, ignoreEquipment) != "" {
			continue
		}
		all = append(all, scored{i, p.score(ex, part, focus, unitLevel), ex.SharedTags(focus.Tags)})
	}
	// Arbeiten: an exercise that shares no tag with the focus only counts when nothing else fits.
	if part == training.PartMain {
		if tagged := slices.DeleteFunc(slices.Clone(all), func(s scored) bool { return s.tags == 0 }); len(tagged) > 0 {
			all = tagged
		}
	}
	if len(all) == 0 {
		return nil
	}
	slices.SortFunc(all, func(a, b scored) int {
		ea, eb := p.in.Exercises[a.i], p.in.Exercises[b.i]
		return cmp.Or(
			cmp.Compare(b.score, a.score),
			cmp.Compare(p.libRank(ea.Library), p.libRank(eb.Library)),
			cmp.Compare(a.i, b.i),
			strings.Compare(ea.Title, eb.Title),
		)
	})
	ex := p.in.Exercises[all[0].i]
	return &ex
}

// pickPart chooses the exercise of a part: the model's proposal when it is allowed (and, for the main
// part, fits the focus), else the best by score; InsteadOf is set when the best one needs set-up
// the location does not have.
func (p *planner) pickPart(part string, focus training.Focus, unitLevel string, picked map[string]bool, hint *training.Exercise) pick {
	var pk pick
	if hint != nil {
		why := p.candidate(*hint, part, false)
		if why == "" && picked[hint.ID] {
			why = "ist in dieser Einheit schon eingeplant"
		}
		if why == "" && part == training.PartMain && len(focus.Tags) > 0 && hint.SharedTags(focus.Tags) == 0 {
			why = "passt nicht zum Schwerpunkt " + focus.Label
		}
		if why == "" {
			ex := *hint
			pk.ex = &ex
			return pk
		}
		pk.replaced = fmt.Sprintf("Übung „%s“ ersetzt: %s.", shorten(hint.Title), why)
	}
	pk.ex = p.best(part, focus, unitLevel, picked, false)
	if want := p.best(part, focus, unitLevel, picked, true); want != nil && (pk.ex == nil || pk.ex.ID != want.ID) && !training.FitsEquipment(*want, p.avail) {
		pk.insteadOf = want
		pk.why = p.equipmentWhy(want, pk.ex)
	}
	if pk.insteadOf == nil && pk.ex != nil && part == training.PartMain && p.goalBoost(*pk.ex) {
		pk.why = fmt.Sprintf("Richtung %s: %s ist die nächste Lektion.", training.ClassLabel(p.goal), pk.ex.Title)
	}
	return pk
}

func shorten(title string) string {
	if r := []rune(title); len(r) > maxExerciseTitleLen {
		return string(r[:maxExerciseTitleLen-1]) + "…"
	}
	return title
}

// equipmentWhy says in German that the exercise that fitted best needs set-up the unit's location
// does not have today, and what is done instead.
func (p *planner) equipmentWhy(wanted, chosen *training.Exercise) string {
	name, with := training.LocationName(p.act)
	var head string
	switch {
	case p.in.EquipmentOK != nil && !*p.in.EquipmentOK:
		head = "Heute ohne Aufbau"
	case p.in.EquipmentOK == nil && p.in.Equipment.Of(p.act).Shared:
		head = name + " geteilt, ohne Aufbau"
	default:
		labels := []string{}
		for _, k := range training.MissingEquipment(*wanted, p.avail) {
			labels = append(labels, training.EquipmentLabel(k))
		}
		head = fmt.Sprintf("Ohne %s %s", training.JoinGerman(labels), with)
	}
	if chosen == nil {
		return fmt.Sprintf("%s: freie Arbeit statt %s.", head, wanted.Title)
	}
	return fmt.Sprintf("%s: %s statt %s.", head, chosen.Title, wanted.Title)
}
