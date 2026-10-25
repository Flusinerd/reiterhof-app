package dressage

import "fmt"

// ClassInfo is a class of the picker.
type ClassInfo struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Practice bool   `json:"practice"`
}

// ArenaInfo is an arena size with its letters.
type ArenaInfo struct {
	Size    string   `json:"size"`
	Label   string   `json:"label"`
	Width   float64  `json:"width"`
	Length  float64  `json:"length"`
	Corner  float64  `json:"corner_radius"`
	Letters []Letter `json:"letters"`
}

// SpeedInfo is the speed range of a gait (m/min) and the cadence (strides per minute).
type SpeedInfo struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Cadence [2]int  `json:"cadence"`
}

// Link is an official source the app points to.
type Link struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

// CatalogInfo is the static part of GET /dressage/catalog.
type CatalogInfo struct {
	Arenas    []ArenaInfo          `json:"arenas"`
	Classes   []ClassInfo          `json:"classes"`
	Movements []Movement           `json:"movements"`
	Speeds    map[string]SpeedInfo `json:"speeds"`
	// SpeedSource names where the speeds and the cadence come from.
	SpeedSource string `json:"speed_source"`
	Links       []Link `json:"links"`
}

// Catalog returns the static catalog: arenas with letters, classes, movements, speeds and the links to the official sources.
func Catalog() CatalogInfo {
	c := CatalogInfo{Movements: Movements, SpeedSource: SpeedSource, Speeds: map[string]SpeedInfo{}}
	for _, size := range Sizes {
		a, _ := NewArena(size)
		c.Arenas = append(c.Arenas, ArenaInfo{Size: size, Label: map[string]string{Size20x40: "20 × 40 m", Size20x60: "20 × 60 m"}[size],
			Width: Width, Length: a.Length, Corner: CornerRadius, Letters: a.Letters()})
	}
	for _, k := range Classes {
		c.Classes = append(c.Classes, ClassInfo{Key: k, Label: "Klasse " + k, Practice: contains(PracticeClasses, k)})
	}
	for g, r := range SpeedRanges {
		c.Speeds[g] = SpeedInfo{Min: r.Min, Max: r.Max, Cadence: Cadence[g]}
	}
	c.Links = []Link{
		{Key: "fnverlag", Label: "Aufgabenheft und Leitfäden beim FNverlag", URL: "https://www.fnverlag.de"},
		{Key: "fn_kuer", Label: "Merkblatt Dressur-Kür bei der FN", URL: "https://www.pferdesport-deutschland.de"},
	}
	return c
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Evaluate rebuilds a plan from the specs of its lines (after the user reordered or removed lines)
// and runs the checks of its kind. An error means the specs do not describe lines of the library.
func Evaluate(kind, class, size string, seed int64, specs []Spec, req Requirement) (Plan, error) {
	lib, ok := LibraryFor(size)
	if !ok {
		return Plan{}, fmt.Errorf("Das Viereck ist unbekannt.")
	}
	if ClassRank(class) < 0 {
		return Plan{}, fmt.Errorf("Die Klasse ist unbekannt.")
	}
	if len(specs) == 0 || len(specs) > 120 {
		return Plan{}, fmt.Errorf("Die Aufgabe braucht zwischen 1 und 120 Zeilen.")
	}
	lines, err := BuildLines(lib, specs)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Kind: kind, Class: class, Arena: size, Seed: seed, Lines: lines}
	p.Totals(lib)
	switch kind {
	case KindPractice:
		p.Checks = CheckPractice(lib, class, &p)
		p.Requirements = []RequirementStatus{}
	case KindFreestyle:
		p.Checks, p.Requirements = CheckFreestyle(lib, class, req, &p)
	default:
		return Plan{}, fmt.Errorf("Die Art der Aufgabe ist unbekannt.")
	}
	return p, nil
}

// RuleErrors returns the texts of the failing rule checks.
func RuleErrors(checks []Check) []string {
	var out []string
	for _, c := range checks {
		if c.Rule && !c.OK {
			out = append(out, c.Text)
		}
	}
	return out
}

// Title is the default title of a plan: "Übungsaufgabe E · 20×40" or "Kür L* · 20×60".
func Title(kind, class, size string) string {
	name := "Übungsaufgabe"
	if kind == KindFreestyle {
		name = "Kür"
	}
	return fmt.Sprintf("%s %s · %s", name, class, map[string]string{Size20x40: "20×40", Size20x60: "20×60"}[size])
}
