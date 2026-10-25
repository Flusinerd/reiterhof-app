package dressage

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// Thresholds of the checks. The balance (each rein at least 40 % of the time on the reins) and the
// choreography values are the app's own, the time tolerance of 10 seconds is the one of the leaflet.
const (
	reinMinShare     = 0.40
	timeTolerance    = 10.0
	minGaitReinSecs  = 8.0
	arenaUseShare    = 0.8
	nonStandardShare = 0.25
	maxChangesPerMin = 4.0
	minGaitShare     = 0.12
)

func check(key string, ok, rule bool, text string) Check {
	return Check{Key: key, OK: ok, Rule: rule, Text: text}
}

func movementLabel(key string) string {
	if m, ok := MovementByKey(key); ok {
		return m.Label
	}
	return key
}

// structureChecks are the rules shared by both kinds: the lines chain, the test starts with the entry and
// the salute at X or G and ends with the salute on the centre line and the exit.
func structureChecks(lib *Library, p *Plan) []Check {
	var out []Check
	// Lines chain.
	chain, msg := true, "Jede Zeile beginnt dort, wo die vorige endet."
	for i, l := range p.Lines {
		want := State{"OUT", ""}
		if i > 0 {
			want = p.Lines[i-1].End
		}
		if l.Start != want {
			chain = false
			if i == 0 {
				msg = "Die erste Zeile muss das Einreiten bei A sein."
			} else {
				msg = fmt.Sprintf("Zeile %d beginnt nicht dort, wo Zeile %d endet (%s, %s).", i+1, i, nodeLabel(want), nodeLabel(l.Start))
			}
			break
		}
	}
	out = append(out, check("chain", chain, true, msg))
	// No line may start by turning back on the spot where the previous one ended.
	flow, flowMsg := true, "Jede Zeile lässt sich aus der vorigen heraus reiten."
	for i := 1; i < len(p.Lines) && chain; i++ {
		if turnsOnTheSpot(lib.ByID[p.Lines[i-1].Option], lib.ByID[p.Lines[i].Option]) {
			flow = false
			flowMsg = fmt.Sprintf("Zeile %d würde bei %s auf der Stelle kehrtmachen; dazwischen fehlt eine Wendung.", i+1, nodeLabel(p.Lines[i].Start))
			break
		}
	}
	out = append(out, check("flow", flow, true, flowMsg))
	n := len(p.Lines)
	openOK := n >= 3 && lib.ByID[p.Lines[0].Option] != nil && lib.ByID[p.Lines[0].Option].Kind == kindEinreiten
	out = append(out, check("opening", openOK, true, ternary(openOK,
		"Einreiten bei A, Halten und Grüßen auf der Mittellinie in Richtung C.",
		"Die Prüfung beginnt mit dem Einreiten bei A und dem Gruß auf der Mittellinie.")))
	closeOK := n >= 3
	if closeOK {
		a, b := lib.ByID[p.Lines[n-2].Option], lib.ByID[p.Lines[n-1].Option]
		closeOK = a != nil && b != nil && a.Kind == kindAbwenden && b.Kind == kindAusreiten
	}
	out = append(out, check("closing", closeOK, true, ternary(closeOK,
		"Gruß auf der Mittellinie in Richtung C am Ende, dann Verlassen des Vierecks bei A.",
		"Die Prüfung endet mit dem Halten und Grüßen auf der Mittellinie in Richtung C und dem Verlassen bei A.")))
	return out
}

func nodeLabel(s State) string {
	switch s.Node {
	case "OUT":
		return "außerhalb"
	case "DONE":
		return "draußen"
	}
	return strings.TrimSuffix(s.Node, "*")
}

func ternary(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}

// classCheck says whether every movement of the lines belongs to the class; required movements (a
// freestyle's own list) are exempt.
func classCheck(class string, p *Plan, required map[string]bool) Check {
	var bad []string
	for _, l := range p.Lines {
		for _, k := range l.Movements {
			m, ok := MovementByKey(k)
			if !ok || required[k] || m.Allowed(class) {
				continue
			}
			if s := fmt.Sprintf("„%s“ (%s)", m.Label, classNote(m)); !slices.Contains(bad, s) {
				bad = append(bad, s)
			}
		}
	}
	if len(bad) == 0 {
		return check("class", true, true, "Keine Lektion aus einer höheren Klasse als "+class+".")
	}
	return check("class", false, true, "Lektionen einer höheren Klasse als "+class+": "+strings.Join(bad, ", ")+".")
}

func classNote(m Movement) string {
	if m.LowestClass == "" {
		return NotSourced + ", erst ab L** angeboten"
	}
	return "ab " + m.LowestClass
}

// gaitReins is the time per gait on each rein.
func gaitReins(lib *Library, p *Plan) map[string][2]float64 {
	out := map[string][2]float64{}
	for _, l := range p.Lines {
		o := lib.ByID[l.Option]
		if o == nil || o.Length == 0 {
			continue
		}
		moving := math.Max(0, l.Seconds-o.Fixed)
		v := out[l.Gait]
		v[0] += moving * o.Left / o.Length
		v[1] += moving * o.Right / o.Length
		out[l.Gait] = v
	}
	return out
}

func reinShares(p *Plan) (left, right float64, ok bool) {
	t := p.Reins.Left + p.Reins.Right
	if t <= 0 {
		return 0, 0, false
	}
	return p.Reins.Left / t, p.Reins.Right / t, true
}

func balanceText(l, r float64) string {
	return fmt.Sprintf("links %d %%, rechts %d %%", int(math.Round(l*100)), int(math.Round(r*100)))
}

// CheckPractice checks a practice test: the structure (entry, salute, end), every gait on both
// reins, the reins in balance and the movements of the class. The last check is advice: the
// duration against the app's target window.
func CheckPractice(lib *Library, class string, p *Plan) []Check {
	out := structureChecks(lib, p)
	out = append(out, classCheck(class, p, nil))
	gr := gaitReins(lib, p)
	var missing []string
	for _, g := range []string{GaitWalk, GaitTrot, GaitCanter} {
		v := gr[g]
		if v[0] < minGaitReinSecs || v[1] < minGaitReinSecs {
			missing = append(missing, GaitLabel(g))
		}
	}
	out = append(out, check("gaits_both_reins", len(missing) == 0, true, ternary(len(missing) == 0,
		"Schritt, Trab und Galopp auf beiden Händen.",
		"Es fehlt die Arbeit auf beiden Händen im "+strings.Join(missing, ", ")+".")))
	l, r, ok := reinShares(p)
	bal := ok && l >= reinMinShare && r >= reinMinShare
	out = append(out, check("balance", bal, true, ternary(bal, "Beide Hände ausgewogen: "+balanceText(l, r)+".",
		"Die Hände sind nicht ausgewogen: "+balanceText(l, r)+"; jede Hand sollte mindestens 40 % haben.")))
	lo, hi := PracticeWindow(class, LengthShort)
	_, hi2 := PracticeWindow(class, LengthNormal)
	lo2, _ := PracticeWindow(class, LengthNormal)
	inWindow := (p.Seconds >= lo && p.Seconds <= hi) || (p.Seconds >= lo2 && p.Seconds <= hi2)
	out = append(out, check("duration", inWindow, false, fmt.Sprintf("Dauer %s.", MMSS(p.Seconds))))
	return out
}

// CheckFreestyle checks a freestyle against the rules of the Kür leaflet (in our own words) and the
// user's requirement set, and rates the choreography. It returns the checks and the status of every
// required movement.
func CheckFreestyle(lib *Library, class string, req Requirement, p *Plan) ([]Check, []RequirementStatus) {
	out := structureChecks(lib, p)
	required := req.requiredKeys()
	out = append(out, classCheck(class, p, required))

	// Every gait is shown.
	var missingGait []string
	for g, secs := range map[string]float64{GaitWalk: p.Gaits.Walk, GaitTrot: p.Gaits.Trot, GaitCanter: p.Gaits.Canter} {
		if secs < 5 {
			missingGait = append(missingGait, GaitLabel(g))
		}
	}
	slices.Sort(missingGait)
	out = append(out, check("gaits", len(missingGait) == 0, true, ternary(len(missingGait) == 0,
		"Schritt, Trab und Galopp werden gezeigt.",
		"Eine Grundgangart fehlt: "+strings.Join(missingGait, ", ")+".")))

	// Walk: only on straight and curved lines, at least the required distance.
	if walkReq := req.walkMeters(); walkReq > 0 || requiresWalk(req) {
		best, cur := 0.0, 0.0
		for _, l := range p.Lines {
			if l.Gait == GaitWalk && !l.Lateral && l.Figure != kindAusreiten {
				cur += l.Meters
				best = math.Max(best, cur)
			} else {
				cur = 0
			}
		}
		ok := best > 0 && best >= walkReq
		text := fmt.Sprintf("Schritt auf gerader oder gebogener Linie: zusammenhängend %d m", int(best))
		if walkReq > 0 {
			text += fmt.Sprintf(" (gefordert %d m)", int(walkReq))
		}
		if !ok && best == 0 {
			text = "Der geforderte Schritt wird nur im Seitengang oder beim Verkleinern des Vierecks gezeigt; er muss auf gerader oder gebogener Linie liegen."
		} else if !ok {
			text += " – zu kurz."
		} else {
			text += "."
		}
		out = append(out, check("walk", ok, true, text))
	}

	// Lengthenings: not on a clearly curved line, not through two corners.
	var badLengthening []int
	for _, l := range p.Lines {
		if l.Gait == GaitTrot && (l.Tempo == TempoMedium || l.Tempo == TempoExtended) && !l.Straight {
			badLengthening = append(badLengthening, l.N)
		}
	}
	out = append(out, check("lengthening", len(badLengthening) == 0, true, ternary(len(badLengthening) == 0,
		"Verstärkungen im Trab liegen auf geraden Linien.",
		fmt.Sprintf("Verstärkungen im Trab gehören auf gerade Linien, nicht auf Kreise oder durch zwei Ecken (Zeile %s).", joinInts(badLengthening)))))

	// Canter voltes in true canter.
	var badVolte []int
	for _, l := range p.Lines {
		if l.Counter && (l.Figure == "volte8" || l.Figure == "volte10") {
			badVolte = append(badVolte, l.N)
		}
	}
	out = append(out, check("canter_voltes", len(badVolte) == 0, true, ternary(len(badVolte) == 0,
		"Volten im Galopp im Handgalopp.",
		fmt.Sprintf("Volten im Galopp müssen im Handgalopp geritten werden, nicht im Außengalopp (Zeile %s).", joinInts(badVolte)))))

	// Kurzkehrt at most 180°.
	var badKK []int
	for _, l := range p.Lines {
		if slices.Contains(l.Movements, "kurzkehrt") && l.Degrees > 180 {
			badKK = append(badKK, l.N)
		}
	}
	out = append(out, check("kurzkehrt", len(badKK) == 0, true, ternary(len(badKK) == 0,
		"Kurzkehrt nicht mehr als 180°.",
		fmt.Sprintf("Eine Kurzkehrtwendung darf nicht mehr als 180° haben (Zeile %s).", joinInts(badKK)))))

	// Time window.
	if lo, hi, ok := req.Window(); ok {
		in := p.Seconds >= lo-timeTolerance && p.Seconds <= hi+timeTolerance
		text := fmt.Sprintf("Dauer %s, Zeitfenster %s bis %s.", MMSS(p.Seconds), MMSS(lo), MMSS(hi))
		if !in {
			text += " Mehr als 10 Sekunden außerhalb kosten Punkte."
		}
		out = append(out, check("time", in, true, text))
	} else {
		out = append(out, check("time", true, true, fmt.Sprintf("Dauer %s (kein Zeitfenster eingetragen).", MMSS(p.Seconds))))
	}

	// Required movements.
	status := requirementStatus(req, p)
	var unmet []string
	for _, s := range status {
		if !s.Met {
			unmet = append(unmet, s.Label)
		}
	}
	out = append(out, check("requirements", len(unmet) == 0, true, ternary(len(unmet) == 0,
		"Alle Pflichtlektionen sind enthalten.",
		"Pflichtlektionen fehlen oder kommen zu selten vor: "+strings.Join(unmet, ", ")+".")))

	out = append(out, choreography(lib, req, p)...)
	return out, status
}

func requiresWalk(req Requirement) bool {
	for _, rm := range req.Movements {
		if m, ok := MovementByKey(rm.Key); ok && m.Gait == GaitWalk {
			return true
		}
	}
	return false
}

func joinInts(v []int) string {
	s := make([]string, len(v))
	for i, n := range v {
		s[i] = fmt.Sprint(n)
	}
	return strings.Join(s, ", ")
}

// requirementStatus counts how often each required movement is shown, per rein where asked.
func requirementStatus(req Requirement, p *Plan) []RequirementStatus {
	var out []RequirementStatus
	for _, rm := range req.Movements {
		label := movementLabel(rm.Key)
		reins := []string{rm.Rein}
		if rm.Rein == "both" {
			reins = []string{"left", "right"}
		}
		for _, rein := range reins {
			shown := 0
			for _, l := range p.Lines {
				if !slices.Contains(l.Movements, rm.Key) {
					continue
				}
				if (rein == "left" || rein == "right") && l.Rein != rein {
					continue
				}
				shown++
			}
			text := label
			switch rein {
			case "left":
				text += " links"
			case "right":
				text += " rechts"
			}
			out = append(out, RequirementStatus{Key: rm.Key, Label: text, Rein: rein, Count: max(1, rm.Count), Shown: shown, Met: shown >= max(1, rm.Count)})
		}
	}
	return out
}

// demanding reports whether the choreography counts a line as a highlight.
func demanding(l Line) bool {
	if l.Tempo == TempoMedium && l.Gait != GaitWalk || l.Tempo == TempoExtended || l.Lateral && l.Figure != "viereck" && l.Figure != "schenkelweichen" {
		return true
	}
	for _, k := range l.Movements {
		if m, ok := MovementByKey(k); ok && m.Demanding {
			return true
		}
	}
	return false
}

// choreography rates the criteria of the B-note of the leaflet in plain words; the checks are
// advice (the criteria are artistic), never a promise of a score.
func choreography(lib *Library, req Requirement, p *Plan) []Check {
	var out []Check
	a := lib.Arena

	// The whole arena is used.
	cols, rows := 4, int(a.Length/5)
	seen := map[[2]int]bool{}
	for _, l := range p.Lines {
		for _, pt := range l.Path {
			cx := min(cols-1, max(0, int(pt[0]/5)))
			cy := min(rows-1, max(0, int(pt[1]/5)))
			seen[[2]int{cx, cy}] = true
		}
	}
	use := float64(len(seen)) / float64(cols*rows)
	out = append(out, check("arena_use", use >= arenaUseShare, false, fmt.Sprintf("Viereck genutzt: %d von %d Feldern%s.", len(seen), cols*rows,
		ternary(use >= arenaUseShare, "", " – nutze auch Mittellinie, Viertellinien und Diagonalen"))))

	// Both reins in balance.
	if l, r, ok := reinShares(p); ok {
		bal := l >= reinMinShare && r >= reinMinShare
		out = append(out, check("balance", bal, false, ternary(bal, "Beide Hände ausgewogen: "+balanceText(l, r)+".",
			"Die Hände sind nicht ausgewogen: "+balanceText(l, r)+".")))
	}

	// Highlights towards C.
	total, towardsC := 0, 0
	required := req.requiredKeys()
	for _, l := range p.Lines {
		hi := demanding(l)
		for _, k := range l.Movements {
			if required[k] && !isBasicKey(k) {
				hi = true
			}
		}
		if !hi || len(l.Path) == 0 {
			continue
		}
		total++
		if l.Path[len(l.Path)/2][1] > a.Length/2 {
			towardsC++
		}
	}
	out = append(out, check("highlights_c", total == 0 || towardsC*2 >= total, false,
		fmt.Sprintf("Höhepunkte in Richtung C: %d von %d.", towardsC, total)))

	// Not only standard lines.
	var std, all float64
	for _, l := range p.Lines {
		all += l.Meters
		switch l.Figure {
		case "track", "diagonal", kindEinreiten, kindWeiter, kindAbwenden, kindAusreiten:
			std += l.Meters
		}
	}
	nonStd := 0.0
	if all > 0 {
		nonStd = 1 - std/all
	}
	out = append(out, check("lines", nonStd >= nonStandardShare, false,
		fmt.Sprintf("Nicht nur Standardlinien: %d %% der Strecke auf Zirkeln, Volten, Mittel- und Viertellinien.", int(math.Round(nonStd*100)))))

	// The same lesson not always on the same line.
	repeats := map[string]int{}
	for _, l := range p.Lines {
		for _, k := range l.Movements {
			if isBasicKey(k) {
				continue
			}
			repeats[k+"|"+l.Option+"|"+l.Gait]++
		}
	}
	var same []string
	for k, n := range repeats {
		if n > 1 {
			same = append(same, movementLabel(strings.SplitN(k, "|", 2)[0]))
		}
	}
	slices.Sort(same)
	same = slices.Compact(same)
	out = append(out, check("same_line", len(same) == 0, false, ternary(len(same) == 0,
		"Keine Lektion immer auf derselben Linie.", "Dieselbe Lektion auf derselben Linie mehrfach: "+strings.Join(same, ", ")+".")))

	// Alternate demanding and calm passages, build to a finale.
	run, maxRun, finale := 0, 0, false
	content := p.Lines
	if len(content) > 4 {
		content = content[2 : len(content)-2]
	}
	for i, l := range content {
		if demanding(l) {
			run++
			maxRun = max(maxRun, run)
			if i >= len(content)-3 {
				finale = true
			}
		} else {
			run = 0
		}
	}
	out = append(out, check("alternation", maxRun <= 2, false, ternary(maxRun <= 2,
		"Anspruchsvolle und ruhige Teile wechseln sich ab.", "Mehr als zwei anspruchsvolle Zeilen hintereinander; plane eine ruhige Zeile dazwischen.")))
	out = append(out, check("finale", finale, false, ternary(finale,
		"Die Kür steigert sich zu einem Finale.", "Setze eine anspruchsvolle Lektion in die letzten Zeilen vor dem Schlussgruß, damit sich die Kür steigert.")))

	// Few changes of gait; each gait in balance.
	changes := 0
	for i := 1; i < len(p.Lines); i++ {
		if p.Lines[i].Gait != p.Lines[i-1].Gait {
			changes++
		}
	}
	perMin := 0.0
	if p.Seconds > 0 {
		perMin = float64(changes) / (p.Seconds / 60)
	}
	out = append(out, check("gait_changes", perMin <= maxChangesPerMin, false,
		fmt.Sprintf("%d Wechsel der Gangart, %.1f pro Minute%s.", changes, perMin, ternary(perMin <= maxChangesPerMin, "", " – zu unruhig für die Musik"))))
	moving := p.Gaits.Walk + p.Gaits.Trot + p.Gaits.Canter
	bal := moving > 0 && p.Gaits.Walk/moving >= minGaitShare && p.Gaits.Trot/moving >= minGaitShare && p.Gaits.Canter/moving >= minGaitShare
	out = append(out, check("gait_balance", bal, false, fmt.Sprintf("Gangarten: Schritt %s, Trab %s, Galopp %s.",
		MMSS(p.Gaits.Walk), MMSS(p.Gaits.Trot), MMSS(p.Gaits.Canter))))
	return out
}

// isBasicKey marks the gait movements and the common figures every plan has.
func isBasicKey(k string) bool {
	m, ok := MovementByKey(k)
	return ok && m.Basis
}
