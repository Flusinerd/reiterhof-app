package training

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Exercise libraries (exercises.discipline since migration 0330 means the library).
const (
	LibraryDressage   = "dressage"
	LibraryJumping    = "jumping"
	LibraryCavaletti  = "cavaletti"
	LibraryGroundwork = "groundwork"
	LibraryLunge      = "lunge"
	LibraryHack       = "hack"
	LibraryWestern    = "western"
)

// Libraries lists every library.
var Libraries = []string{
	LibraryDressage, LibraryJumping, LibraryCavaletti, LibraryGroundwork, LibraryLunge, LibraryHack, LibraryWestern,
}

// ExerciseLibraries returns the libraries whose exercises fit an activity, the home library
// first: hall and arena use dressage and cavaletti (jumping horses: jumping, cavaletti,
// dressage; western horses: western, cavaletti, dressage), jumping uses jumping and cavaletti,
// lunge lunge and cavaletti, groundwork its own, hack the hack library; walker and rest have none.
func ExerciseLibraries(a Activity, horseDiscipline string) []string {
	switch a {
	case ActivityHall, ActivityArena:
		switch horseDiscipline {
		case "jumping":
			return []string{LibraryJumping, LibraryCavaletti, LibraryDressage}
		case "western":
			return []string{LibraryWestern, LibraryCavaletti, LibraryDressage}
		}
		return []string{LibraryDressage, LibraryCavaletti}
	case ActivityJumping:
		return []string{LibraryJumping, LibraryCavaletti}
	case ActivityLunge:
		return []string{LibraryLunge, LibraryCavaletti}
	case ActivityGroundwork:
		return []string{LibraryGroundwork}
	case ActivityHack:
		return []string{LibraryHack}
	}
	return nil
}

// HomeLibraries returns the libraries a horse of the discipline is trained in towards the next
// level: dressage and unknown disciplines dressage, jumping jumping and dressage, western western.
func HomeLibraries(discipline string) []string {
	switch discipline {
	case "jumping":
		return []string{LibraryJumping, LibraryDressage}
	case "western":
		return []string{LibraryWestern}
	}
	return []string{LibraryDressage}
}

// Parts of a unit (the three parts of a riding lesson) and the phases an exercise can have.
const (
	PartWarmup   = "warmup"   // Lösen
	PartMain     = "main"     // Arbeiten
	PartCooldown = "cooldown" // Entspannen
	// PhaseAny marks an exercise that fits every part (exercises.phase).
	PhaseAny = "any"
)

// Exercise intensities (exercises.intensity).
const (
	ExerciseLight     = "light"
	ExerciseNormal    = "normal"
	ExerciseDemanding = "demanding"
)

// Exercise is the pure view of a row of the exercise library, as the planner needs it. Mastered
// and Rated depend on the horse: a session with focus rating 3 ("Sitzt") and a session with any
// rating.
type Exercise struct {
	ID      string
	Library string
	Level   string // LevelBeginner, LevelIntermediate, LevelAdvanced, "" for none
	Title   string
	Tags    []string
	// Phase is PartWarmup, PartMain, PartCooldown or PhaseAny; "" counts as main.
	Phase string
	// Minutes is the typical duration, 0 when unknown.
	Minutes int
	// Intensity is light, normal or demanding; "" counts as normal.
	Intensity string
	// Requires lists the exercises (ids) that have to be mastered before this one is planned.
	Requires  []string
	Equipment []string
	// Suits lists "young" and "senior" for exercises that suit those horses.
	Suits    []string
	Mastered bool
	Rated    bool
	// NextID is the follow-up exercise of the progression, "" for none.
	NextID string
	// Own marks a row of the stable (free text): it never goes to the language model.
	Own bool
}

// FitsPart reports whether the exercise can be used in a part (Lösen, Arbeiten, Entspannen).
func (e Exercise) FitsPart(part string) bool {
	switch e.Phase {
	case PhaseAny:
		return true
	case "":
		return part == PartMain
	}
	return e.Phase == part
}

// HasTag reports whether the exercise carries the tag.
func (e Exercise) HasTag(tag string) bool { return slices.Contains(e.Tags, tag) }

// SharedTags counts the tags the exercise shares with tags.
func (e Exercise) SharedTags(tags []string) int {
	n := 0
	seen := map[string]bool{}
	for _, t := range e.Tags {
		if !seen[t] && slices.Contains(tags, t) {
			n++
		}
		seen[t] = true
	}
	return n
}

// FacilityEquipment lists the equipment of the exercise that has to be set up in the stable.
func (e Exercise) FacilityEquipment() []string {
	var out []string
	for _, k := range e.Equipment {
		if IsFacility(k) && !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// SuitsHorse reports whether the exercise is meant for a horse of that age: young (under 6) or
// senior (18 and older).
func (e Exercise) SuitsHorse(ageYears int) bool {
	return (Young(ageYears) && slices.Contains(e.Suits, "young")) || (Senior(ageYears) && slices.Contains(e.Suits, "senior"))
}

// LevelRank orders the library levels: 0 beginner, 1 intermediate, 2 advanced, -1 for none or
// unknown.
func LevelRank(level string) int {
	switch level {
	case LevelBeginner:
		return 0
	case LevelIntermediate:
		return 1
	case LevelAdvanced:
		return 2
	}
	return -1
}

// ClassLabel is the class an exercise level leads to in German riding classes, used in the
// texts about the level goal: intermediate is L, advanced M.
func ClassLabel(level string) string {
	switch level {
	case LevelBeginner:
		return "A"
	case LevelIntermediate:
		return "L"
	case LevelAdvanced:
		return "M"
	}
	return ""
}

// Equipment of the stable (docs/domains/training-planning.md, "Equipment and alternatives").
// Facility equipment is set up in the stable and planned against what the stable has; personal
// equipment is always available.
var (
	FacilityEquipment = []string{"poles", "cavaletti", "jumps", "obstacles"}
	PersonalEquipment = []string{
		"lunge", "side_reins", "double_lunge", "kappzaum", "tarp", "umbrella", "cones", "flag", "whip", "neck_ring", "trailer",
	}
	// EquipmentKeys is the fixed list of all equipment keys (exercises.equipment).
	EquipmentKeys = slices.Concat(FacilityEquipment, PersonalEquipment)
)

// IsFacility reports whether key is facility equipment.
func IsFacility(key string) bool { return slices.Contains(FacilityEquipment, key) }

// ValidEquipment reports whether key is in EquipmentKeys.
func ValidEquipment(key string) bool { return slices.Contains(EquipmentKeys, key) }

var equipmentLabels = map[string]string{
	"poles": "Stangen", "cavaletti": "Cavaletti", "jumps": "Sprünge", "obstacles": "Gelassenheitsparcours",
	"lunge": "Longe", "side_reins": "Ausbindezügel", "double_lunge": "Doppellonge", "kappzaum": "Kappzaum",
	"tarp": "Plane", "umbrella": "Schirm", "cones": "Hütchen", "flag": "Flatterband", "whip": "Gerte",
	"neck_ring": "Halsring", "trailer": "Anhänger",
}

// EquipmentLabel is the German name of an equipment key (the key itself when unknown).
func EquipmentLabel(key string) string {
	if l, ok := equipmentLabels[key]; ok {
		return l
	}
	return key
}

// Locations are the places of a stable a unit takes place in, one per activity; hack, walker and
// rest have none.
var Locations = []Activity{ActivityHall, ActivityArena, ActivityLunge, ActivityJumping, ActivityGroundwork}

// IsLocation reports whether the activity takes place in a location with equipment.
func IsLocation(a Activity) bool { return slices.Contains(Locations, a) }

// Location is the equipment of one location of the stable. Shared means the location is used
// together with others: set-up needs an agreement.
type Location struct {
	Equipment []string `json:"equipment"`
	Shared    bool     `json:"shared"`
}

// DefaultLocation is what a stable without entry has: poles in the hall, the arena and the
// ground work place, jumps and poles for jumping, nothing for the lunge; nothing shared.
func DefaultLocation(a Activity) Location {
	switch a {
	case ActivityHall, ActivityArena, ActivityGroundwork:
		return Location{Equipment: []string{"poles"}}
	case ActivityJumping:
		return Location{Equipment: []string{"jumps", "poles"}}
	}
	return Location{Equipment: []string{}}
}

// StableEquipment is the equipment per location (stables.training_equipment); a missing
// location has its default.
type StableEquipment map[Activity]Location

// Of returns the entry of the location of activity a, or the default.
func (s StableEquipment) Of(a Activity) Location {
	if l, ok := s[a]; ok {
		return l
	}
	return DefaultLocation(a)
}

// Available is the facility equipment a unit can use: what the location has, or nothing when the
// day's override says no set-up (equipmentOK false) or the location is shared and the override
// does not say that set-up is possible (equipmentOK true). Personal equipment is not listed, it
// is always available.
func Available(loc Location, equipmentOK *bool) []string {
	if equipmentOK != nil && !*equipmentOK {
		return nil
	}
	if loc.Shared && (equipmentOK == nil || !*equipmentOK) {
		return nil
	}
	var out []string
	for _, k := range loc.Equipment {
		if IsFacility(k) && !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// FitsEquipment reports whether every piece of facility equipment the exercise needs is in
// available.
func FitsEquipment(ex Exercise, available []string) bool {
	for _, k := range ex.FacilityEquipment() {
		if !slices.Contains(available, k) {
			return false
		}
	}
	return true
}

// MissingEquipment lists the facility equipment of the exercise that is not in available.
func MissingEquipment(ex Exercise, available []string) []string {
	var out []string
	for _, k := range ex.FacilityEquipment() {
		if !slices.Contains(available, k) {
			out = append(out, k)
		}
	}
	return out
}

// LocationName is the German name of the location of an activity, and the same with the
// preposition ("in der Halle").
func LocationName(a Activity) (name, with string) {
	switch a {
	case ActivityHall:
		return "Halle", "in der Halle"
	case ActivityArena:
		return "Platz", "auf dem Platz"
	case ActivityLunge:
		return "Longierhalle", "in der Longierhalle"
	case ActivityJumping:
		return "Springplatz", "auf dem Springplatz"
	case ActivityGroundwork:
		return "Bodenarbeitsplatz", "am Bodenarbeitsplatz"
	}
	return a.GermanName(), "bei " + a.GermanName()
}

// JoinGerman joins words with commas and "und": "A", "A und B", "A, B und C".
func JoinGerman(words []string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	}
	return strings.Join(words[:len(words)-1], ", ") + " und " + words[len(words)-1]
}

// GoalLevel is the level class a horse is trained towards: the stored goal when it is above the
// horse's level class, else the class above it (none for an advanced horse or an unknown level).
func GoalLevel(levelClass, stored string) string {
	cur := LevelRank(levelClass)
	if r := LevelRank(stored); r > 0 && r > cur {
		return stored
	}
	switch cur {
	case 0:
		return LevelIntermediate
	case 1:
		return LevelAdvanced
	}
	return ""
}

// ProgressResult is how far a horse is on the way to its goal level.
type ProgressResult struct {
	Goal     string // goal level class
	Mastered int
	Total    int
	// Next are up to three exercises of the goal level that are not mastered yet and whose
	// prerequisites all are, in progression order.
	Next []Exercise
	// Ready is true when the horse mastered at least 80 % of the goal level's lessons and there
	// are at least three of them.
	Ready bool
}

// Thresholds of Progress.
const (
	progressReadyPercent = 80
	progressReadyMin     = 3
	progressNext         = 3
)

// Progress counts the lessons of the goal level: total = exercises of the goal level in the home
// libraries with phase main or any, mastered = those with at least one session rated 3 ("Sitzt"),
// next = up to three unmastered ones whose prerequisites are all mastered, in the order of
// exercises (progression order). Ready = mastered is at least 80 % of total and total is at least 3.
func Progress(goal string, home []string, exercises []Exercise, sessions []Session) ProgressResult {
	res := ProgressResult{Goal: goal}
	mastered := map[string]bool{}
	for _, s := range sessions {
		if s.ExerciseID != "" && s.FocusRating == 3 {
			mastered[s.ExerciseID] = true
		}
	}
	for _, ex := range exercises {
		if ex.Level != goal || !slices.Contains(home, ex.Library) || !ex.FitsPart(PartMain) {
			continue
		}
		res.Total++
		if mastered[ex.ID] {
			res.Mastered++
			continue
		}
		if len(res.Next) < progressNext && allIn(ex.Requires, mastered) {
			res.Next = append(res.Next, ex)
		}
	}
	res.Ready = res.Total >= progressReadyMin && res.Mastered*100 >= progressReadyPercent*res.Total
	return res
}

func allIn(ids []string, set map[string]bool) bool {
	for _, id := range ids {
		if !set[id] {
			return false
		}
	}
	return true
}

// Text is the German sentence about the progress: "Auf dem Weg zu L: 7 von 12 Lektionen
// gefestigt.", with a goal date "Ziel L bis 15.03.: noch 5 Lektionen.", when ready "Luna ist
// bereit für L: 10 von 12 Lektionen gefestigt." until is the goal date, nil for none.
func (p ProgressResult) Text(horse string, until *time.Time) string {
	label := ClassLabel(p.Goal)
	switch {
	case p.Ready:
		return fmt.Sprintf("%s ist bereit für %s: %d von %d Lektionen gefestigt.", horse, label, p.Mastered, p.Total)
	case until != nil:
		left := p.Total - p.Mastered
		noun := "Lektionen"
		if left == 1 {
			noun = "Lektion"
		}
		return fmt.Sprintf("Ziel %s bis %s: noch %d %s.", label, until.Format("02.01."), left, noun)
	}
	return fmt.Sprintf("Auf dem Weg zu %s: %d von %d Lektionen gefestigt.", label, p.Mastered, p.Total)
}

// LevelUpText is the warning that a horse may be ready for the next level class.
func (p ProgressResult) LevelUpText(horse string) string {
	label := ClassLabel(p.Goal)
	return fmt.Sprintf("%s hat %d von %d %s-Lektionen gefestigt – Einstufung auf %s anpassen?", horse, p.Mastered, p.Total, label, label)
}

// Alternatives are the exercises that can replace one that needs set-up: the same library, phase
// equal or any, at least two shared tags, no facility equipment, a level at or below. At most
// limit of them, ordered by shared tags (most first), level (easiest first) and title. An
// exercise without facility equipment needs no alternative.
func Alternatives(of Exercise, pool []Exercise, limit int) []Exercise {
	if len(of.FacilityEquipment()) == 0 {
		return nil
	}
	type scored struct {
		ex     Exercise
		shared int
	}
	var out []scored
	for _, c := range pool {
		if c.ID == of.ID || c.Library != of.Library || len(c.FacilityEquipment()) > 0 {
			continue
		}
		if cp := phaseOrMain(c.Phase); cp != phaseOrMain(of.Phase) && cp != PhaseAny {
			continue
		}
		if lr := max(LevelRank(c.Level), 0); lr > max(LevelRank(of.Level), 0) {
			continue
		}
		if n := c.SharedTags(of.Tags); n >= 2 {
			out = append(out, scored{c, n})
		}
	}
	slices.SortFunc(out, func(a, b scored) int {
		return cmp.Or(
			cmp.Compare(b.shared, a.shared),
			cmp.Compare(max(LevelRank(a.ex.Level), 0), max(LevelRank(b.ex.Level), 0)),
			strings.Compare(a.ex.Title, b.ex.Title),
			strings.Compare(a.ex.ID, b.ex.ID),
		)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	res := make([]Exercise, len(out))
	for i, s := range out {
		res[i] = s.ex
	}
	return res
}

func phaseOrMain(p string) string {
	if p == "" {
		return PartMain
	}
	return p
}
