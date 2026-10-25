package training

import (
	"math"
	"slices"
	"time"
)

// Training planning (docs/domains/training-planning.md): every unit has exactly one focus, taken
// from this static catalog. The first seven entries are the steps of the Skala der Ausbildung
// (FN Richtlinien Band 1: Takt, Losgelassenheit, Anlehnung, Schwung, Geraderichtung,
// Versammlung; Durchlässigkeit is the aim over all of them), the others are practice focuses.
// The German description and signs paraphrase the FN definitions as they are summarised on
// de.wikipedia.org (Takt (Reiten), Losgelassenheit, Anlehnung (Pferdesport), Schwung (Reiten),
// Geraderichten, Versammlung (Reiten), Durchlässigkeit (Reiten)).

// Focus groups.
const (
	GroupScale    = "scale"    // a step of the Skala der Ausbildung
	GroupPractice = "practice" // a practice focus
)

// Focus is one entry of the focus catalog. Keys are stable ASCII identifiers; every text is
// German UI text.
type Focus struct {
	Key   string
	Label string
	Group string
	// Phase is the catalog phase: 1 Gewöhnung, 2 Entwicklung der Schubkraft, 3 Entwicklung der
	// Tragkraft; 0 fits every phase of the horse.
	Phase int
	// Tags are the exercise tags (ExerciseTags) the focus matches.
	Tags []string
	// Fits lists the activities the focus can be trained in.
	Fits []Activity
	// Description says what the focus is (one sentence), Signs how to recognise it ("Woran du es
	// merkst", one sentence).
	Description string
	Signs       string
	// Needs is the warning shown when the focus is above the horse's phase, empty for the
	// focuses of phase 0 and 1.
	Needs string
}

// FitsActivity reports whether the focus can be trained in the activity.
func (f Focus) FitsActivity(a Activity) bool { return slices.Contains(f.Fits, a) }

// AllowedIn reports whether the focus belongs to the horse's phase or an earlier one.
func (f Focus) AllowedIn(p HorsePhase) bool { return f.Phase <= p.Rank() }

// Focuses is the catalog in display order: the scale first, then the practice focuses.
var Focuses = []Focus{
	{
		Key: "takt", Label: "Takt", Group: GroupScale, Phase: 1,
		Tags: []string{"takt", "rhythmus"}, Fits: []Activity{ActivityHall, ActivityArena, ActivityLunge, ActivityHack},
		Description: "Takt ist das gleichmäßige Zeitmaß der Tritte und Sprünge: Der Schritt geht im Viertakt, der Trab im Zweitakt, der Galopp im Dreitakt.",
		Signs:       "Der Hufschlag klingt gleichmäßig, auch in Wendungen, Übergängen und bei Tempowechseln.",
	},
	{
		Key: "losgelassenheit", Label: "Losgelassenheit", Group: GroupScale, Phase: 1,
		Tags: []string{"losgelassenheit", "biegung"}, Fits: []Activity{ActivityHall, ActivityArena, ActivityLunge, ActivityHack},
		Description: "Losgelassenheit heißt, dass das Pferd körperlich und seelisch entspannt geht: Der Rücken schwingt, der Hals dehnt sich nach vorn, die Bewegungen sind ohne Eile.",
		Signs:       "Das Pferd schnaubt ab, kaut zufrieden, trägt den Schweif locker und tritt mit schwingendem Rücken.",
	},
	{
		Key: "anlehnung", Label: "Anlehnung", Group: GroupScale, Phase: 1,
		Tags: []string{"anlehnung"}, Fits: []Activity{ActivityHall, ActivityArena, ActivityLunge},
		Description: "Anlehnung ist die weiche, gleichmäßige Verbindung von der Reiterhand zum Pferdemaul, die aus dem Untertreten der Hinterhand und dem schwingenden Rücken entsteht.",
		Signs:       "Das Pferd kaut zufrieden am Gebiss, steht federnd an der Hand und stützt sich nicht auf die Zügel.",
	},
	{
		Key: "schwung", Label: "Schwung", Group: GroupScale, Phase: 2,
		Tags: []string{"schwung"}, Fits: []Activity{ActivityHall, ActivityArena, ActivityHack},
		Description: "Schwung ist der energische Impuls der Hinterhand, der auf die ganze Vorwärtsbewegung übertragen wird.",
		Signs:       "Die Tritte werden raumgreifend und elastisch, die Hinterbeine fußen energisch ab und schwingen nach vorn durch.",
		Needs:       "Schwung setzt Losgelassenheit und Anlehnung voraus.",
	},
	{
		Key: "geraderichten", Label: "Geraderichtung", Group: GroupScale, Phase: 2,
		Tags: []string{"geraderichten", "biegung", "seitengaenge"}, Fits: []Activity{ActivityHall, ActivityArena},
		Description: "Geraderichten heißt, dass das Pferd hufschlagdeckend geht: Die Hinterhufe treten in die Spur der Vorderhufe, auf geraden wie auf gebogenen Linien.",
		Signs:       "Das Pferd steht auf beiden Händen gleichmäßig an den Zügeln und lässt sich nach links und rechts gleich gut stellen und biegen.",
		Needs:       "Geraderichtung setzt Anlehnung und Schwung voraus.",
	},
	{
		Key: "versammlung", Label: "Versammlung", Group: GroupScale, Phase: 3,
		Tags: []string{"versammlung"}, Fits: []Activity{ActivityHall, ActivityArena},
		Description: "Versammlung ist der Zustand, in dem das Pferd die Hinterhand stärker beugt und mehr Last aufnimmt, sodass die Vorhand leichter wird.",
		Signs:       "Die Hinterhand senkt sich, die Vorhand richtet sich auf, die Tritte werden kürzer und erhabener, ohne dass der Takt leidet.",
		Needs:       "Versammlung setzt Schwung und Geraderichtung voraus.",
	},
	{
		Key: "durchlaessigkeit", Label: "Durchlässigkeit", Group: GroupScale, Phase: 2,
		Tags: []string{"durchlaessigkeit"}, Fits: []Activity{ActivityHall, ActivityArena, ActivityLunge},
		Description: "Durchlässigkeit ist das Ziel der gesamten Ausbildung: Das Pferd nimmt die Hilfen des Reiters zwanglos, gehorsam und ohne Zögern an.",
		Signs:       "Schon feine Hilfen genügen, das Pferd geht ohne Widerstand mit und bleibt dabei locker.",
		Needs:       "Durchlässigkeit setzt Takt, Losgelassenheit und Anlehnung voraus.",
	},
	{
		Key: "seitengaenge", Label: "Seitengänge", Group: GroupPractice, Phase: 2,
		Tags: []string{"seitengaenge"}, Fits: []Activity{ActivityHall, ActivityArena, ActivityGroundwork},
		Description: "Seitengänge wie Schenkelweichen, Schulterherein und Travers verbinden Vorwärts- und Seitwärtsbewegung und gymnastizieren das Pferd.",
		Signs:       "Das Pferd weicht auf den seitwärts treibenden Schenkel, bleibt im Takt und behält Stellung und Biegung.",
		Needs:       "Seitengänge setzen Losgelassenheit und Anlehnung voraus.",
	},
	{
		Key: "gelassenheit", Label: "Gelassenheit", Group: GroupPractice, Phase: 0,
		Tags: []string{"gelassenheit", "vertrauen"}, Fits: []Activity{ActivityGroundwork, ActivityHack, ActivityArena},
		Description: "Gelassenheit heißt, dass das Pferd bei Neuem und Ungewohntem ruhig bleibt und dem Menschen vertraut.",
		Signs:       "Bei Plane, Brücke oder Schirm bleibt das Pferd neugierig statt schreckhaft und beruhigt sich nach einem Erschrecken schnell wieder.",
	},
	{
		Key: "kondition", Label: "Kondition", Group: GroupPractice, Phase: 0,
		Tags: []string{"kondition", "ausdauer"}, Fits: []Activity{ActivityHack, ActivityArena, ActivityHall},
		Description: "Kondition ist die Ausdauer, mit der das Pferd Belastung über längere Zeit trägt; sie wächst langsam und in kleinen Schritten.",
		Signs:       "Atmung und Puls beruhigen sich nach der Arbeit schnell wieder, und das Pferd wirkt am Ende noch frisch.",
	},
	{
		Key: "springen", Label: "Springen", Group: GroupPractice, Phase: 0,
		Tags: []string{"springgymnastik", "rhythmus"}, Fits: []Activity{ActivityJumping, ActivityHall, ActivityArena},
		Description: "Springen schult Rhythmus, Balance und Zutrauen über Stangen, Cavaletti, Gymnastikreihen und Sprünge.",
		Signs:       "Das Pferd springt aus gleichmäßigem Rhythmus auf gerader Linie und bleibt dabei aufmerksam und ruhig.",
	},
	{
		Key: "gelaende", Label: "Gelände", Group: GroupPractice, Phase: 0,
		Tags: []string{"gelaende", "gelassenheit", "kondition"}, Fits: []Activity{ActivityHack},
		Description: "Gelände heißt, das Pferd draußen auf Wegen, Hügeln und über kleine Hindernisse zu reiten, aufmerksam und vorwärts.",
		Signs:       "Das Pferd geht ruhig und vorwärts, nimmt Steigungen im Takt und lässt sich auch unterwegs vom Reiter leiten.",
	},
	{
		Key: "vertrauen", Label: "Vertrauen", Group: GroupPractice, Phase: 0,
		Tags: []string{"vertrauen", "aufmerksamkeit"}, Fits: []Activity{ActivityGroundwork, ActivityLunge},
		Description: "Vertrauen ist die Grundlage der Zusammenarbeit: Das Pferd achtet auf den Menschen und folgt einem leichten Signal.",
		Signs:       "Das Pferd reagiert auf ein leichtes Signal, behält den Menschen im Blick und gibt bei Druck nach.",
	},
	{
		Key: "mobilitaet", Label: "Beweglichkeit", Group: GroupPractice, Phase: 0,
		Tags: []string{"mobilitaet", "losgelassenheit"}, Fits: []Activity{ActivityHall, ActivityArena, ActivityLunge, ActivityGroundwork, ActivityHack},
		Description: "Beweglichkeit heißt, dass das Pferd Hals, Rücken und Gliedmaßen locker und weit bewegt; Dehnung und Biegung halten es geschmeidig.",
		Signs:       "Das Pferd lässt sich mühelos biegen, dehnt den Hals frei nach vorn und arbeitet auf beiden Seiten gleich gut.",
	},
	{
		Key: "turnier", Label: "Turniervorbereitung", Group: GroupPractice, Phase: 0,
		Tags: []string{"turnier", "durchlaessigkeit", "aufmerksamkeit"}, Fits: []Activity{ActivityHall, ActivityArena, ActivityJumping},
		Description: "Turniervorbereitung heißt, die Prüfung so zu üben, wie sie geritten wird: Aufgabe oder Parcours, Aufwärmen auf dem Turnierplatz, Einreiten, Halten und Grüßen.",
		Signs:       "Aufgabe oder Parcours gelingen sicher von Anfang bis Ende, und das Pferd bleibt auch in fremder Umgebung konzentriert.",
	},
}

// FocusTurnier is the key of the one focus with date logic (an owner sets it until a show date,
// the planner adds it by itself shortly before a show).
const FocusTurnier = "turnier"

// FocusByKey returns the catalog entry of a key.
func FocusByKey(key string) (Focus, bool) {
	for _, f := range Focuses {
		if f.Key == key {
			return f, true
		}
	}
	return Focus{}, false
}

// FocusLabel returns the German label of a focus key ("" for an unknown key).
func FocusLabel(key string) string {
	f, _ := FocusByKey(key)
	return f.Label
}

// ValidFocusKey reports whether key is in the catalog.
func ValidFocusKey(key string) bool {
	_, ok := FocusByKey(key)
	return ok
}

// FocusChoice is a focus an owner picked for a horse: the catalog key and an optional last day
// (inclusive; a focus with a date in the past is ignored).
type FocusChoice struct {
	Key   string     `json:"key"`
	Until *time.Time `json:"until,omitempty"`
}

// Active reports whether the choice still applies on day.
func (c FocusChoice) Active(day time.Time) bool {
	return c.Until == nil || DaysBetween(day, *c.Until) >= 0
}

// MaxHorseFocuses and MaxRiderFocuses limit the focuses of a horse and of a rider.
const (
	MaxHorseFocuses = 3
	MaxRiderFocuses = 2
)

// HorsePhase is the phase of the horse's training by the Skala der Ausbildung (FN: three
// overlapping phases). It decides which focuses fit the horse.
type HorsePhase string

const (
	PhaseGewoehnung HorsePhase = "gewoehnung" // Takt, Losgelassenheit, Anlehnung
	PhaseSchubkraft HorsePhase = "schubkraft" // Entwicklung der Schubkraft
	PhaseTragkraft  HorsePhase = "tragkraft"  // Entwicklung der Tragkraft
)

// Rank is the highest catalog phase the horse's phase allows: 1, 2 or 3.
func (p HorsePhase) Rank() int {
	switch p {
	case PhaseSchubkraft:
		return 2
	case PhaseTragkraft:
		return 3
	}
	return 1
}

// Ages that count as young and senior (rules with numbers of the planning spec).
const (
	YoungBelow = 6  // a horse under 6 is young
	SeniorFrom = 18 // a horse of 18 and older is a senior
)

// Young reports whether a horse of that age (0 = unknown) is young.
func Young(ageYears int) bool { return ageYears > 0 && ageYears < YoungBelow }

// Senior reports whether a horse of that age (0 = unknown) is a senior.
func Senior(ageYears int) bool { return ageYears >= SeniorFrom }

// Phase returns the horse's phase: Gewöhnung when the level class is beginner or unknown, the
// horse is under 6 or the status is not fit; Schubkraft for intermediate and Tragkraft for
// advanced horses. levelClass is LevelClass of the profile's level, ageYears 0 when unknown.
func Phase(levelClass string, ageYears int, status Status) HorsePhase {
	if status != StatusFit || Young(ageYears) {
		return PhaseGewoehnung
	}
	switch levelClass {
	case LevelIntermediate:
		return PhaseSchubkraft
	case LevelAdvanced:
		return PhaseTragkraft
	}
	return PhaseGewoehnung
}

// HintKeys returns the scale focuses that belong to the horse's phase and are not in chosen,
// in catalog order: what the owner could pick next.
func HintKeys(p HorsePhase, chosen []string) []string {
	out := []string{}
	for _, f := range Focuses {
		if f.Group == GroupScale && f.Phase == p.Rank() && !slices.Contains(chosen, f.Key) {
			out = append(out, f.Key)
		}
	}
	return out
}

// ExerciseTags is the fixed list of aims an exercise can have (exercises.goal_tags). A test over
// the catalog refuses every other tag.
var ExerciseTags = []string{
	"takt", "rhythmus", "losgelassenheit", "biegung", "anlehnung", "schwung", "geraderichten",
	"versammlung", "durchlaessigkeit", "seitengaenge", "balance", "springgymnastik",
	"aufmerksamkeit", "vertrauen", "gelassenheit", "kondition", "ausdauer", "kraft", "gelaende",
	"mobilitaet", "turnier",
}

// ValidExerciseTag reports whether tag is in ExerciseTags.
func ValidExerciseTag(tag string) bool { return slices.Contains(ExerciseTags, tag) }

// Split is the division of a unit into its three parts, in minutes: Lösen (warm-up), Arbeiten
// (main part) and Entspannen (cool-down). The zero value means "no unit plan".
type Split struct{ Warmup, Main, Cooldown int }

// Total is the length of the unit.
func (s Split) Total() int { return s.Warmup + s.Main + s.Cooldown }

// Empty reports whether the split has no minutes at all (walker, rest).
func (s Split) Empty() bool { return s.Total() == 0 }

// Limits of the unit split (rules with numbers, "Unit").
const (
	splitWarmupShare   = 0.33
	splitCooldownShare = 0.17
	splitWarmupMin     = 10
	splitWarmupMax     = 25
	splitJumpingWarm   = 15 // jumping: Lösen at least this long
	splitCooldownMin   = 5
	splitCooldownMax   = 10
	splitMainMin       = 10 // hall, arena, jumping: Arbeiten at least this long
	splitWarmupCap     = 30 // with the extra minutes of seniors and cold nights
	splitExtra         = 5
	// The extra minutes of seniors and cold nights never take the main part of a lunge,
	// groundwork or hack unit below this (their base split has no minimum).
	splitOtherMainMin = 5
)

// UnitSplit divides a unit of totalMinutes minutes of activity a into Lösen, Arbeiten and
// Entspannen:
//
//	hall, arena, jumping  Lösen clamp(round(T × 0.33), 10, 25), jumping at least 15;
//	                      Arbeiten the rest, at least 10; Entspannen clamp(round(T × 0.17), 5, 10)
//	lunge                 10 (walk, no side reins), the rest, 5
//	groundwork            5, the rest, 5
//	hack                  10 (walk), the rest, 10 (walk)
//	walker, rest          no unit plan
//
// Seniors and a night minimum below 5 °C (cold) each add 5 minutes to Lösen (at most 30 in all),
// seniors 5 to Entspannen too ("warm-up and cool-down 5 minutes longer"); the time comes off
// Arbeiten, which keeps its minimum (10 minutes, 5 for lunge, groundwork and hack). The parts
// always add up to totalMinutes; a hall, arena or jumping unit too short for the minimums is
// divided in the proportions of the base split.
func UnitSplit(a Activity, totalMinutes int, senior, cold bool) Split {
	t := totalMinutes
	if t <= 0 {
		return Split{}
	}
	var warm, cool, mainMin int
	switch a {
	case ActivityHall, ActivityArena, ActivityJumping:
		warm = clampInt(int(math.Round(float64(t)*splitWarmupShare)), splitWarmupMin, splitWarmupMax)
		if a == ActivityJumping {
			warm = max(warm, splitJumpingWarm)
		}
		cool = clampInt(int(math.Round(float64(t)*splitCooldownShare)), splitCooldownMin, splitCooldownMax)
		mainMin = splitMainMin
		if t < warm+cool+mainMin {
			// Too short for the minimums: keep the proportions.
			w := int(math.Round(float64(t) * splitWarmupShare))
			c := int(math.Round(float64(t) * splitCooldownShare))
			return Split{Warmup: w, Main: t - w - c, Cooldown: c}
		}
	case ActivityLunge:
		warm, cool, mainMin = 10, 5, splitOtherMainMin
	case ActivityGroundwork:
		warm, cool, mainMin = 5, 5, splitOtherMainMin
	case ActivityHack:
		warm, cool, mainMin = 10, 10, splitOtherMainMin
	default:
		return Split{}
	}
	// A unit shorter than its fixed parts: cut the cool-down, then the warm-up.
	warm = min(warm, t)
	cool = min(cool, t-warm)

	room := t - warm - cool - mainMin // minutes the extras may take from the main part
	extraWarm := 0
	if senior {
		extraWarm += splitExtra
	}
	if cold {
		extraWarm += splitExtra
	}
	extraWarm = min(extraWarm, max(splitWarmupCap-warm, 0), max(room, 0))
	warm += extraWarm
	room -= extraWarm
	if senior {
		cool += min(splitExtra, max(room, 0))
	}
	return Split{Warmup: warm, Main: t - warm - cool, Cooldown: cool}
}

func clampInt(v, lo, hi int) int { return max(lo, min(v, hi)) }
