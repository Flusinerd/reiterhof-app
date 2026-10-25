package dressage

import "slices"

// Classes in ascending order. A class label is the German class of the test ("L*", "L**").
var Classes = []string{"E", "A", "L", "L*", "L**", "M*", "M**"}

// PracticeClasses are the classes a practice test can be generated for.
var PracticeClasses = []string{"E", "A", "L", "L**"}

// ClassRank orders the classes; -1 for an unknown one.
func ClassRank(class string) int { return slices.Index(Classes, class) }

// unsourcedRank is the rank from which a movement without a source for its lowest class is offered:
// L** (docs/domains/dressage-tests.md, "Movements").
var unsourcedRank = ClassRank("L**")

// NotSourced is the German note of a movement whose lowest class is not documented.
const NotSourced = "Klasse nicht belegt"

// Sources of the lowest class of a movement. Every one names a document the implementation opened;
// none is a test text or a score sheet.
const (
	srcBasis = "PM-Forum „Hufschlagfiguren auf der richtigen Linie“, FN Pferd und Mensch 02/2023 " +
		"(magazin.pferdesport-deutschland.de/project/hufschlagfiguren-auf-der-richtigen-linie-02-2023/): " +
		"Hufschlagfiguren und Grundgangarten sind die Grundlage jeder Dressurprüfung; Klasse E als unterste Klasse gesetzt"
	srcVolte10 = "de.wikipedia.org „Volte (Reiten)“: Volten im Trab mit 10 m Durchmesser in Klasse A"
	srcVolte8  = "FN Pferd und Mensch 07/2023 „Lektion im Fokus“ (magazin.pferdesport-deutschland.de/project/lektion-im-fokus-07-2023/) " +
		"und de.wikipedia.org „Volte (Reiten)“: Volten mit 8 m ab Klasse L, im Trab und Galopp"
	srcRueck = "FN Pferd und Mensch 11/12 2021 „Lektion im Fokus“ (magazin.pferdesport-deutschland.de/project/lektion-im-fokus-11-12-2021/): " +
		"Rückwärtsrichten kommt in Dressurprüfungen von Klasse A bis S vor"
	srcSchenkel = "FN Pferd und Mensch 08/2022 „10 Tipps“ (magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/): " +
		"Schenkelweichen in vielen Dressurprüfungen der Klassen E und A"
	srcMitteltrab = "FN Pferd und Mensch 07/2022 „Lektion im Fokus“ (magazin.pferdesport-deutschland.de/project/lektion-im-fokus-07-2022/): " +
		"Mitteltrab in fast allen Dressuraufgaben von Klasse A bis Grand Prix"
	srcAussen = "FN Pferd und Mensch 08/2023 „10 Tipps“ (magazin.pferdesport-deutschland.de/project/10-tipps-08-2023/) " +
		"und de.wikipedia.org „Außengalopp“: ab Klasse L verlangt"
	srcWechselL = "FN/DRV Merkblatt Dressur-Kür, gültig ab 01/2024, Abschnitt Galoppwechsel (allgemeine Regel): " +
		"der einfache Galoppwechsel gehört in die Klasse L, die Schrittzahl liegt bei 3 bis 5"
	srcFliegend = "de.wikipedia.org „Galoppwechsel“ (Richtlinien Bd. 2): der fliegende Wechsel wird in der M-Dressur verlangt"
	srcLateral  = "FN/DRV Merkblatt Dressur-Kür, gültig ab 01/2024, Abschnitt Seitengänge (allgemeine Regel): Seitengänge ab Klasse L**, " +
		"Schulterherein mindestens 12 m, Traversalen mindestens 5 m zu jeder Seite; Anlage 2024: Travers und Renvers ab L**"
	srcKurzkehrt = "FN/DRV Merkblatt Dressur-Kür, gültig ab 01/2024, Abschnitt Kurzkehrtwendungen (allgemeine Regel): " +
		"Wendung aus dem Mittelschritt oder Trab, höchstens 180°, im Merkblatt als Lektion der Klasse L beschrieben; " +
		"FN Pferd und Mensch 01/2022 „Lektion im Fokus“ (Ausführung)"
	srcAnlageA = "FN/DRV Merkblatt Dressur-Kür, Anlage gültig ab 01/2024 (allgemeine Regel): Schlangenlinien jeder Art im Trab " +
		"sowie Viereck verkleinern und vergrößern im Schritt und Trab ab Klasse A"
	srcAnlageL = "FN/DRV Merkblatt Dressur-Kür, Anlage gültig ab 01/2024 (allgemeine Regel): alle Schlangenlinien im Trab und Galopp ab Klasse L"
)

// Movement is what the rider shows: a gait and tempo, a figure or a lesson. Lines of a plan list the
// keys of the movements they show.
type Movement struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Gait  string `json:"gait"`
	Tempo string `json:"tempo"`
	// LowestClass is the lowest class that asks for the movement, "" when it is not sourced.
	LowestClass string `json:"lowest_class"`
	// ClassNote is NotSourced for a movement without class.
	ClassNote string `json:"class_note,omitempty"`
	Source    string `json:"source"`
	// Rein is "left", "right", "either" (one rein is enough) or "both" (to be shown on both reins).
	Rein string `json:"rein"`
	// ExerciseIDs are the catalog exercises that train the movement: it is mastered when one of them
	// has a session rated 3.
	ExerciseIDs []string `json:"exercise_ids"`
	// Figures are the kinds of figure the movement can be shown on (see Option.Kind).
	Figures []string `json:"-"`
	// StraightOnly marks tempo movements that need a straight line (lengthenings).
	StraightOnly bool `json:"-"`
	// Basis marks the movements every test has (gaits, halt, the common figures): they do not
	// depend on the mastery of a catalog exercise.
	Basis bool `json:"basis,omitempty"`
	// Demanding marks the movements the choreography treats as highlights.
	Demanding bool `json:"demanding,omitempty"`
}

// Rank is the lowest class as a number; a movement without source starts at L**.
func (m Movement) Rank() int {
	if m.LowestClass == "" {
		return unsourcedRank
	}
	return ClassRank(m.LowestClass)
}

// Allowed reports whether the movement may be shown in a test of the class.
func (m Movement) Allowed(class string) bool { return m.Rank() <= ClassRank(class) }

// Exercise ids of the global catalog (migrations 0250 and 0340).
func ex(n ...string) []string {
	out := make([]string, len(n))
	for i, s := range n {
		out[i] = "00000000-0000-4000-8000-000000000" + s
	}
	return out
}

var gaitKinds = []string{"track", "diagonal", "halbe_bahn", "mittellinie", "zirkel"}
var straightKinds = []string{"track", "diagonal", "halbe_bahn", "mittellinie"}

// Movements is the catalog in display order.
var Movements = buildMovements()

var movementIndex = func() map[string]*Movement {
	m := map[string]*Movement{}
	for i := range Movements {
		m[Movements[i].Key] = &Movements[i]
	}
	return m
}()

// MovementByKey returns a movement of the catalog.
func MovementByKey(key string) (Movement, bool) {
	m, ok := movementIndex[key]
	if !ok {
		return Movement{}, false
	}
	return *m, true
}

func buildMovements() []Movement {
	var out []Movement
	add := func(m Movement) {
		if m.LowestClass == "" {
			m.ClassNote = NotSourced
			if m.Source == "" {
				m.Source = "keine Quelle für die Klasse gefunden"
			}
		}
		if m.Rein == "" {
			m.Rein = "either"
		}
		if m.ExerciseIDs == nil {
			m.ExerciseIDs = []string{}
		}
		out = append(out, m)
	}
	// Gaits, tempi, halt.
	add(Movement{Key: "halten_gruss", Label: "Einreiten, Halten und Grüßen", Gait: GaitHalt, LowestClass: "E", Source: srcBasis, Basis: true,
		ExerciseIDs: ex("856", "876"), Figures: []string{"einreiten", "abwenden_halt"}})
	add(Movement{Key: "arbeitstrab", Label: "Arbeitstrab", Gait: GaitTrot, Tempo: TempoWorking, LowestClass: "E", Source: srcBasis, Basis: true, Rein: "both", Figures: gaitKinds})
	add(Movement{Key: "mittelschritt", Label: "Mittelschritt", Gait: GaitWalk, Tempo: TempoMedium, LowestClass: "E", Source: srcBasis, Basis: true, Rein: "both", Figures: gaitKinds})
	add(Movement{Key: "freier_schritt", Label: "Freier Schritt am langen Zügel", Gait: GaitWalk, Tempo: TempoFree, LowestClass: "E", Source: srcBasis, Basis: true,
		ExerciseIDs: ex("850", "853"), Figures: gaitKinds})
	add(Movement{Key: "arbeitsgalopp", Label: "Arbeitsgalopp", Gait: GaitCanter, Tempo: TempoWorking, LowestClass: "E", Source: srcBasis, Basis: true, Rein: "both", Figures: []string{"track", "zirkel", "mittellinie"}})
	add(Movement{Key: "mitteltrab", Label: "Mitteltrab", Gait: GaitTrot, Tempo: TempoMedium, LowestClass: "A", Source: srcMitteltrab, Rein: "both",
		ExerciseIDs: ex("814", "855"), Figures: straightKinds, StraightOnly: true, Demanding: true})
	add(Movement{Key: "versammelter_schritt", Label: "Versammelter Schritt", Gait: GaitWalk, Tempo: TempoCollected, Figures: gaitKinds})
	add(Movement{Key: "versammelter_trab", Label: "Versammelter Trab", Gait: GaitTrot, Tempo: TempoCollected, Figures: gaitKinds})
	add(Movement{Key: "versammelter_galopp", Label: "Versammelter Galopp", Gait: GaitCanter, Tempo: TempoCollected, Figures: []string{"track", "zirkel"}})
	add(Movement{Key: "starker_schritt", Label: "Starker Schritt", Gait: GaitWalk, Tempo: TempoExtended, Figures: straightKinds, StraightOnly: true, Demanding: true})
	add(Movement{Key: "starker_trab", Label: "Starker Trab", Gait: GaitTrot, Tempo: TempoExtended, Figures: straightKinds, StraightOnly: true, Demanding: true})
	add(Movement{Key: "mittelgalopp", Label: "Mittelgalopp", Gait: GaitCanter, Tempo: TempoMedium, ExerciseIDs: ex("884"), Figures: straightKinds, StraightOnly: true, Demanding: true})
	add(Movement{Key: "starker_galopp", Label: "Starker Galopp", Gait: GaitCanter, Tempo: TempoExtended, Figures: straightKinds, StraightOnly: true, Demanding: true})
	add(Movement{Key: "aussengalopp", Label: "Außengalopp", Gait: GaitCanter, Tempo: TempoWorking, LowestClass: "L", Source: srcAussen,
		ExerciseIDs: ex("816"), Figures: []string{"aussengalopp_kurzseite"}, Demanding: true})
	add(Movement{Key: "einfacher_galoppwechsel", Label: "Einfacher Galoppwechsel", Gait: GaitCanter, Tempo: TempoWorking, LowestClass: "L", Source: srcWechselL,
		ExerciseIDs: ex("815"), Figures: []string{"diagonale_einfacher_wechsel", "aussengalopp_kurzseite"}, Demanding: true})
	add(Movement{Key: "fliegender_wechsel", Label: "Fliegender Galoppwechsel", Gait: GaitCanter, Tempo: TempoWorking, LowestClass: "M*", Source: srcFliegend,
		ExerciseIDs: ex("885"), Demanding: true})

	// Figures.
	add(Movement{Key: "zirkel", Label: "Zirkel (20 m)", Gait: "", LowestClass: "E", Source: srcBasis, Basis: true, ExerciseIDs: ex("809"), Figures: []string{"zirkel"}})
	add(Movement{Key: "handwechsel_diagonale", Label: "Durch die ganze Bahn wechseln", LowestClass: "E", Source: srcBasis, Basis: true, Figures: []string{"diagonal"}})
	add(Movement{Key: "handwechsel_halbe_bahn", Label: "Durch die halbe Bahn wechseln", LowestClass: "E", Source: srcBasis, Basis: true, Figures: []string{"halbe_bahn"}})
	add(Movement{Key: "handwechsel_zirkel", Label: "Durch den Zirkel wechseln", LowestClass: "E", Source: srcBasis, Basis: true, Figures: []string{"handwechsel_zirkel"}})
	add(Movement{Key: "mittellinie", Label: "Mittellinie", LowestClass: "E", Source: srcBasis, Basis: true, Figures: []string{"mittellinie"}})
	add(Movement{Key: "schenkelweichen", Label: "Schenkelweichen", Gait: GaitTrot, Tempo: TempoWorking, LowestClass: "E", Source: srcSchenkel,
		ExerciseIDs: ex("804", "868", "869"), Figures: []string{"schenkelweichen"}})
	add(Movement{Key: "rueckwaertsrichten", Label: "Rückwärtsrichten", Gait: GaitHalt, LowestClass: "A", Source: srcRueck,
		ExerciseIDs: ex("806"), Figures: []string{"rueckwaerts"}})
	for _, v := range []struct {
		d     string
		gait  string
		class string
		src   string
	}{
		{"10", GaitTrot, "A", srcVolte10}, {"10", GaitWalk, "", ""}, {"10", GaitCanter, "", ""},
		{"8", GaitTrot, "L", srcVolte8}, {"8", GaitCanter, "L", srcVolte8}, {"8", GaitWalk, "", ""},
	} {
		add(Movement{Key: "volte_" + v.d + "_" + gaitSuffix(v.gait), Label: "Volte " + v.d + " m im " + GaitLabel(v.gait), Gait: v.gait, Tempo: TempoWorking,
			LowestClass: v.class, Source: v.src, ExerciseIDs: ex("805"), Figures: []string{"volte" + v.d}})
	}
	for _, s := range []struct{ key, label, kind string }{
		{"schlangenlinie_ganze_bahn", "Schlangenlinie durch die ganze Bahn", "schlangenlinie"},
		{"einfache_schlangenlinie", "Einfache Schlangenlinie", "einfache_schlange"},
		{"doppelte_schlangenlinie", "Doppelte Schlangenlinie", "doppelte_schlange"},
	} {
		for _, g := range []struct {
			gait, class, src string
		}{{GaitTrot, "A", srcAnlageA}, {GaitCanter, "L", srcAnlageL}, {GaitWalk, "", ""}} {
			add(Movement{Key: s.key + "_" + gaitSuffix(g.gait), Label: s.label + " im " + GaitLabel(g.gait), Gait: g.gait, Tempo: TempoWorking,
				LowestClass: g.class, Source: g.src, ExerciseIDs: ex("802"), Figures: []string{s.kind}})
		}
	}
	add(Movement{Key: "viereck_verkleinern_trab", Label: "Viereck verkleinern und vergrößern im Trab", Gait: GaitTrot, Tempo: TempoWorking, LowestClass: "A", Source: srcAnlageA,
		ExerciseIDs: ex("813"), Figures: []string{"viereck"}})
	add(Movement{Key: "viereck_verkleinern_schritt", Label: "Viereck verkleinern und vergrößern im Schritt", Gait: GaitWalk, Tempo: TempoMedium, LowestClass: "A", Source: srcAnlageA,
		ExerciseIDs: ex("813"), Figures: []string{"viereck"}})
	add(Movement{Key: "viereck_verkleinern_galopp", Label: "Viereck verkleinern und vergrößern im Galopp", Gait: GaitCanter, Tempo: TempoWorking, ExerciseIDs: ex("813"), Figures: []string{"viereck"}})
	add(Movement{Key: "kurzkehrt", Label: "Kurzkehrt", Gait: GaitWalk, Tempo: TempoMedium, LowestClass: "L", Source: srcKurzkehrt,
		ExerciseIDs: ex("807"), Figures: []string{"kurzkehrt"}, Demanding: true})
	add(Movement{Key: "halbe_schrittpirouette", Label: "Halbe Schrittpirouette", Gait: GaitWalk, Tempo: TempoCollected, ExerciseIDs: ex("807"), Demanding: true})
	// Lateral work.
	add(Movement{Key: "schulterherein", Label: "Schulterherein", Gait: GaitTrot, Tempo: TempoWorking, LowestClass: "L**", Source: srcLateral,
		ExerciseIDs: ex("810"), Figures: []string{"schulterherein"}, Demanding: true})
	add(Movement{Key: "travers", Label: "Travers", Gait: GaitTrot, Tempo: TempoWorking, LowestClass: "L**", Source: srcLateral,
		ExerciseIDs: ex("811"), Figures: []string{"travers"}, Demanding: true})
	add(Movement{Key: "renvers", Label: "Renvers", Gait: GaitTrot, Tempo: TempoWorking, LowestClass: "L**", Source: srcLateral,
		ExerciseIDs: ex("817"), Figures: []string{"renvers"}, Demanding: true})
	add(Movement{Key: "traversale_trab", Label: "Traversale im Trab", Gait: GaitTrot, Tempo: TempoWorking, LowestClass: "L**", Source: srcLateral,
		ExerciseIDs: ex("812"), Figures: []string{"traversale"}, Demanding: true})
	return out
}

func gaitSuffix(g string) string {
	switch g {
	case GaitWalk:
		return "schritt"
	case GaitTrot:
		return "trab"
	case GaitCanter:
		return "galopp"
	}
	return g
}

// GaitMovementKey is the key of the movement that names a gait and tempo.
func GaitMovementKey(gait, tempo string) string {
	switch gait {
	case GaitHalt:
		return "halten_gruss"
	case GaitWalk:
		switch tempo {
		case TempoFree:
			return "freier_schritt"
		case TempoCollected:
			return "versammelter_schritt"
		case TempoExtended:
			return "starker_schritt"
		}
		return "mittelschritt"
	case GaitTrot:
		switch tempo {
		case TempoMedium:
			return "mitteltrab"
		case TempoExtended:
			return "starker_trab"
		case TempoCollected:
			return "versammelter_trab"
		}
		return "arbeitstrab"
	case GaitCanter:
		switch tempo {
		case TempoMedium:
			return "mittelgalopp"
		case TempoExtended:
			return "starker_galopp"
		case TempoCollected:
			return "versammelter_galopp"
		}
		return "arbeitsgalopp"
	}
	return ""
}

// figureMovementKeys lists the movements an option of a kind shows in a gait.
func figureMovementKeys(kind, gait string) []string {
	s := gaitSuffix(gait)
	switch kind {
	case "zirkel":
		return []string{"zirkel"}
	case "volte10":
		return []string{"volte_10_" + s}
	case "volte8":
		return []string{"volte_8_" + s}
	case "diagonal":
		return []string{"handwechsel_diagonale"}
	case "halbe_bahn":
		return []string{"handwechsel_halbe_bahn"}
	case "handwechsel_zirkel":
		return []string{"handwechsel_zirkel"}
	case "mittellinie":
		return []string{"mittellinie"}
	case "schlangenlinie":
		return []string{"schlangenlinie_ganze_bahn_" + s}
	case "einfache_schlange":
		return []string{"einfache_schlangenlinie_" + s}
	case "doppelte_schlange":
		return []string{"doppelte_schlangenlinie_" + s}
	case "viereck":
		return []string{"viereck_verkleinern_" + s}
	case "schenkelweichen", "schulterherein", "travers", "renvers", "kurzkehrt":
		return []string{kind}
	case "traversale":
		return []string{"traversale_trab"}
	case "aussengalopp_kurzseite":
		return []string{"handwechsel_diagonale", "aussengalopp", "einfacher_galoppwechsel"}
	case "diagonale_einfacher_wechsel":
		return []string{"handwechsel_diagonale", "einfacher_galoppwechsel"}
	case "rueckwaerts":
		return []string{"rueckwaertsrichten"}
	}
	return nil
}
