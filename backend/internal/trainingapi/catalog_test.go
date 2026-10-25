package trainingapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
)

// The exercise catalog (the global rows of the migrations 0250, 0280 and 0340) is data, and the planner of
// docs/domains/training-planning.md depends on its shape: known tags, phases, libraries and equipment, a
// summary, three to five steps and a source on every row, prerequisites and progressions that stay inside a
// library, and enough exercises for every focus. TestCatalog reads the rows straight from a migrated database
// and runs the checks as subtests; docs/domains/exercise-catalog.md describes the rules.

var (
	catalogLibraries   = []string{"dressage", "cavaletti", "jumping", "groundwork", "lunge", "hack", "western"}
	catalogLevels      = []string{"beginner", "intermediate", "advanced"}
	catalogPhases      = []string{"warmup", "main", "cooldown", "any"}
	catalogIntensities = []string{"light", "normal", "demanding"}
	catalogSuits       = []string{"young", "senior"}

	// Tags of exercises: the fixed list of the spec ("Exercise tags").
	catalogTags = []string{
		"takt", "rhythmus", "losgelassenheit", "biegung", "anlehnung", "schwung", "geraderichten", "versammlung",
		"durchlaessigkeit", "seitengaenge", "balance", "springgymnastik", "aufmerksamkeit", "vertrauen",
		"gelassenheit", "kondition", "ausdauer", "kraft", "gelaende", "mobilitaet", "turnier",
	}

	// Equipment keys: the fixed list of the spec ("Equipment and alternatives"). Facility equipment is planned
	// against the stable's equipment, personal equipment is always available.
	catalogFacilityKeys = []string{"poles", "cavaletti", "jumps", "obstacles"}
	catalogPersonalKeys = []string{"lunge", "side_reins", "double_lunge", "kappzaum", "tarp", "umbrella", "cones", "flag", "whip", "neck_ring", "trailer"}

	// ID ranges of the second catalog (migration 0340): the rows from 850 on.
	catalogRanges = map[string][2]int{
		"dressage": {850, 889}, "cavaletti": {890, 905}, "jumping": {906, 919}, "groundwork": {920, 939},
		"lunge": {940, 949}, "hack": {950, 969}, "western": {970, 989},
	}

	// Two links of the first catalog lead out of their library: the pole rows moved to the library cavaletti in
	// 0330, their follow-ups stayed (846 jumping to 821 cavaletti, 824 cavaletti to 825 jumping).
	catalogLegacyCrossLinks = map[string]string{
		catalogID(846): catalogID(821),
		catalogID(824): catalogID(825),
	}
)

// catalogFocuses is a copy of the focus catalog of the spec (tags matched, activities it fits).
var catalogFocuses = []struct {
	key  string
	tags []string
	fits []string
}{
	{"takt", []string{"takt", "rhythmus"}, []string{"hall", "arena", "lunge", "hack"}},
	{"losgelassenheit", []string{"losgelassenheit", "biegung"}, []string{"hall", "arena", "lunge", "hack"}},
	{"anlehnung", []string{"anlehnung"}, []string{"hall", "arena", "lunge"}},
	{"schwung", []string{"schwung"}, []string{"hall", "arena", "hack"}},
	{"geraderichten", []string{"geraderichten", "biegung", "seitengaenge"}, []string{"hall", "arena"}},
	{"versammlung", []string{"versammlung"}, []string{"hall", "arena"}},
	{"durchlaessigkeit", []string{"durchlaessigkeit"}, []string{"hall", "arena", "lunge"}},
	{"seitengaenge", []string{"seitengaenge"}, []string{"hall", "arena", "groundwork"}},
	{"gelassenheit", []string{"gelassenheit", "vertrauen"}, []string{"groundwork", "hack", "arena"}},
	{"kondition", []string{"kondition", "ausdauer"}, []string{"hack", "arena", "hall"}},
	{"springen", []string{"springgymnastik", "rhythmus"}, []string{"jumping", "hall", "arena"}},
	{"gelaende", []string{"gelaende", "gelassenheit", "kondition"}, []string{"hack"}},
	{"vertrauen", []string{"vertrauen", "aufmerksamkeit"}, []string{"groundwork", "lunge"}},
	{"mobilitaet", []string{"mobilitaet", "losgelassenheit"}, []string{"hall", "arena", "lunge", "groundwork", "hack"}},
	{"turnier", []string{"turnier", "durchlaessigkeit", "aufmerksamkeit"}, []string{"hall", "arena", "jumping"}},
}

// catalogHomeLibrary is the first ("home") library of an activity: where the main exercise of a unit comes from
// for a horse without a special discipline.
var catalogHomeLibrary = map[string]string{
	"hall": "dressage", "arena": "dressage", "jumping": "jumping", "lunge": "lunge", "groundwork": "groundwork", "hack": "hack",
}

type catalogRow struct {
	ID        string
	Library   string
	Level     string
	Tags      []string
	Title     string
	Summary   string
	Steps     []string
	Phase     string
	Minutes   int
	Intensity string
	Requires  []string
	Equipment []string
	Suits     []string
	Sources   []string
	Next      string
}

// catalogID is the ID of the catalog row with number n (the last three digits of the ID).
func catalogID(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-000000000%03d", n) }

// catalogNumber is the number of a catalog ID (see catalogID); -1 for any other ID.
func catalogNumber(id string) int {
	const prefix = "00000000-0000-4000-8000-000000000"
	if !strings.HasPrefix(id, prefix) || len(id) != len(prefix)+3 {
		return -1
	}
	var n int
	if _, err := fmt.Sscanf(id[len(prefix):], "%d", &n); err != nil {
		return -1
	}
	return n
}

func (r catalogRow) short() string { return fmt.Sprintf("%s %q", r.ID[len(r.ID)-3:], r.Title) }

func (r catalogRow) hasTag(tag string) bool { return slices.Contains(r.Tags, tag) }

func (r catalogRow) facility() []string {
	var out []string
	for _, k := range r.Equipment {
		if slices.Contains(catalogFacilityKeys, k) {
			out = append(out, k)
		}
	}
	return out
}

func (r catalogRow) sharedTags(tags []string) int {
	n := 0
	for _, t := range tags {
		if r.hasTag(t) {
			n++
		}
	}
	return n
}

func levelRank(level string) int { return slices.Index(catalogLevels, level) }

func loadCatalog(t *testing.T) []catalogRow {
	t.Helper()
	pool := dbtest.New(t)
	rows, err := pool.Query(context.Background(), `
		SELECT id::text, COALESCE(discipline, ''), COALESCE(level, ''), goal_tags, title, summary, steps, phase,
		       COALESCE(minutes, 0), intensity, requires::text[], equipment, suits, sources,
		       COALESCE(next_exercise_id::text, '')
		FROM exercises WHERE stable_id IS NULL ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []catalogRow
	for rows.Next() {
		var r catalogRow
		var steps []byte
		if err := rows.Scan(&r.ID, &r.Library, &r.Level, &r.Tags, &r.Title, &r.Summary, &steps, &r.Phase,
			&r.Minutes, &r.Intensity, &r.Requires, &r.Equipment, &r.Suits, &r.Sources, &r.Next); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(steps, &r.Steps); err != nil {
			t.Fatalf("%s: steps are no JSON list of strings: %v", r.short(), err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestCatalog(t *testing.T) {
	rows := loadCatalog(t)
	byID := make(map[string]catalogRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	if len(rows) < 150 {
		t.Fatalf("the global catalog has only %d rows", len(rows))
	}
	t.Run("rows are well formed", func(t *testing.T) { checkWellFormed(t, rows) })
	t.Run("prerequisites and progressions stay in the library", func(t *testing.T) { checkReferences(t, rows, byID) })
	t.Run("every library has warm-up and cool-down", func(t *testing.T) { checkWarmupAndCooldown(t, rows) })
	t.Run("every focus is covered", func(t *testing.T) { checkFocusCoverage(t, rows) })
	t.Run("facility exercises have alternatives", func(t *testing.T) { checkAlternatives(t, rows) })
	t.Run("hack has the Kondition chain", func(t *testing.T) { checkKonditionChain(t, byID) })
	t.Run("cavaletti variants", func(t *testing.T) { checkCavalettiVariants(t, byID) })
	t.Run("groundwork stations and loading", func(t *testing.T) { checkGroundwork(t, byID) })
	t.Run("western basics", func(t *testing.T) { checkWestern(t, byID) })
	t.Run("rehearsals for a show", func(t *testing.T) { checkTurnier(t, rows) })
	t.Run("young and senior variants", func(t *testing.T) { checkSuits(t, rows) })
}

func checkWellFormed(t *testing.T, rows []catalogRow) {
	// ISES wording of the new steps: no "muss" (one aid at a time, a light signal first, the release named).
	forbidden := regexp.MustCompile(`(?i)\b(muss|müssen|musst|müsst)\b`)
	url := regexp.MustCompile(`^https?://\S+$`)
	titles := map[string]string{}
	for _, r := range rows {
		who := r.short()
		if !slices.Contains(catalogLibraries, r.Library) {
			t.Errorf("%s: unknown library %q", who, r.Library)
		}
		if !slices.Contains(catalogLevels, r.Level) {
			t.Errorf("%s: unknown level %q", who, r.Level)
		}
		if !slices.Contains(catalogPhases, r.Phase) {
			t.Errorf("%s: unknown phase %q", who, r.Phase)
		}
		if !slices.Contains(catalogIntensities, r.Intensity) {
			t.Errorf("%s: unknown intensity %q", who, r.Intensity)
		}
		if r.Minutes < 1 || r.Minutes > 120 {
			t.Errorf("%s: minutes = %d", who, r.Minutes)
		}
		if len(r.Tags) == 0 {
			t.Errorf("%s: no tags", who)
		}
		seen := map[string]bool{}
		for _, tag := range r.Tags {
			if !slices.Contains(catalogTags, tag) {
				t.Errorf("%s: unknown tag %q", who, tag)
			}
			if seen[tag] {
				t.Errorf("%s: tag %q twice", who, tag)
			}
			seen[tag] = true
		}
		for _, k := range r.Equipment {
			if !slices.Contains(catalogFacilityKeys, k) && !slices.Contains(catalogPersonalKeys, k) {
				t.Errorf("%s: unknown equipment key %q", who, k)
			}
		}
		for _, s := range r.Suits {
			if !slices.Contains(catalogSuits, s) {
				t.Errorf("%s: unknown suits value %q", who, s)
			}
		}
		if prev, dup := titles[r.Library+"/"+r.Title]; dup {
			t.Errorf("%s: title also used by %s in the same library", who, prev)
		}
		titles[r.Library+"/"+r.Title] = r.ID[len(r.ID)-3:]
		if strings.TrimSpace(r.Title) == "" || strings.TrimSpace(r.Summary) == "" {
			t.Errorf("%s: empty title or summary", who)
		}
		if n := len(r.Steps); n < 3 || n > 5 {
			t.Errorf("%s: %d steps, want 3 to 5", who, n)
		}
		for _, s := range r.Steps {
			if strings.TrimSpace(s) == "" {
				t.Errorf("%s: empty step", who)
			}
		}
		if len(r.Sources) == 0 {
			t.Errorf("%s: no source", who)
		}
		for _, s := range r.Sources {
			if !url.MatchString(s) {
				t.Errorf("%s: source %q is no URL", who, s)
			}
		}

		// The rows of the second catalog sit in the ID range of their library and follow the wording rules.
		if num := catalogNumber(r.ID); num >= 850 {
			if rg, ok := catalogRanges[r.Library]; !ok || num < rg[0] || num > rg[1] {
				t.Errorf("%s: ID outside the range of the library %s", who, r.Library)
			}
			for _, s := range r.Steps {
				if forbidden.MatchString(s) {
					t.Errorf("%s: a step uses \"muss\": %s", who, s)
				}
				if strings.Contains(s, `"`) {
					t.Errorf("%s: a step contains a straight double quote: %s", who, s)
				}
			}
		}
	}
}

// Prerequisites and the follow-up exercise exist and stay inside the library (two legacy links excepted); a
// prerequisite is never of a higher level and a follow-up is never easier than its predecessor; neither runs in
// a circle.
func checkReferences(t *testing.T, rows []catalogRow, byID map[string]catalogRow) {
	for _, r := range rows {
		who := r.short()
		for _, q := range r.Requires {
			p, ok := byID[q]
			switch {
			case !ok:
				t.Errorf("%s: requires %s, which does not exist", who, q)
			case q == r.ID:
				t.Errorf("%s: requires itself", who)
			case p.Library != r.Library:
				t.Errorf("%s: requires %q of the library %s, not %s", who, p.Title, p.Library, r.Library)
			case levelRank(p.Level) > levelRank(r.Level):
				t.Errorf("%s: requires %q, which is of a higher level", who, p.Title)
			}
		}
		if r.Next != "" {
			n, ok := byID[r.Next]
			switch {
			case !ok:
				t.Errorf("%s: next exercise %s does not exist", who, r.Next)
			case r.Next == r.ID:
				t.Errorf("%s: is its own follow-up", who)
			case n.Library != r.Library && catalogLegacyCrossLinks[r.ID] != r.Next:
				t.Errorf("%s: follow-up %q is in the library %s, not %s", who, n.Title, n.Library, r.Library)
			case levelRank(n.Level) < levelRank(r.Level):
				t.Errorf("%s: is followed by %q, which is of a lower level", who, n.Title)
			}
		}
		seen := map[string]bool{r.ID: true}
		for cur := r.Next; cur != ""; cur = byID[cur].Next {
			if seen[cur] {
				t.Errorf("%s: the progression runs in a circle", who)
				break
			}
			seen[cur] = true
		}
		if requireCycle(byID, r.ID, map[string]bool{}) {
			t.Errorf("%s: prerequisites run in a circle", who)
		}
	}
}

func requireCycle(byID map[string]catalogRow, id string, path map[string]bool) bool {
	if path[id] {
		return true
	}
	path[id] = true
	defer delete(path, id)
	for _, q := range byID[id].Requires {
		if requireCycle(byID, q, path) {
			return true
		}
	}
	return false
}

// Every library has at least two exercises for the Lösen part and one for Entspannen.
func checkWarmupAndCooldown(t *testing.T, rows []catalogRow) {
	for _, lib := range catalogLibraries {
		var warm, cool int
		for _, r := range rows {
			if r.Library != lib {
				continue
			}
			if r.Phase == "warmup" || r.Phase == "any" {
				warm++
			}
			if r.Phase == "cooldown" || r.Phase == "any" {
				cool++
			}
		}
		if warm < 2 || cool < 1 {
			t.Errorf("library %s: %d warm-up and %d cool-down exercises, want at least 2 and 1", lib, warm, cool)
		}
	}
}

// Every focus has at least three beginner exercises for the Arbeiten part in every library its activities draw
// from (the home library of each activity in "fits"), and at least two of them need no facility equipment.
func checkFocusCoverage(t *testing.T, rows []catalogRow) {
	for _, f := range catalogFocuses {
		var libs []string
		for _, activity := range f.fits {
			lib, ok := catalogHomeLibrary[activity]
			if !ok {
				t.Fatalf("focus %s: no home library for the activity %q", f.key, activity)
			}
			if !slices.Contains(libs, lib) {
				libs = append(libs, lib)
			}
		}
		for _, lib := range libs {
			var total, free int
			for _, r := range rows {
				if r.Library != lib || r.Level != "beginner" || r.Phase != "main" || r.sharedTags(f.tags) == 0 {
					continue
				}
				total++
				if len(r.facility()) == 0 {
					free++
				}
			}
			if total < 3 {
				t.Errorf("focus %s, library %s: %d beginner main exercises, want at least 3", f.key, lib, total)
			}
			if free < 2 {
				t.Errorf("focus %s, library %s: %d of them need no facility equipment, want at least 2", f.key, lib, free)
			}
		}
	}
}

// Every exercise with facility equipment has an alternative without: same library, phase equal or "any", at
// least two shared tags, no facility equipment, level at or below (docs/domains/training-planning.md,
// "Alternatives").
func checkAlternatives(t *testing.T, rows []catalogRow) {
	for _, r := range rows {
		if len(r.facility()) == 0 {
			continue
		}
		found := false
		for _, a := range rows {
			if a.ID == r.ID || a.Library != r.Library || len(a.facility()) > 0 {
				continue
			}
			if (a.Phase == r.Phase || a.Phase == "any") && levelRank(a.Level) <= levelRank(r.Level) && a.sharedTags(r.Tags) >= 2 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s (%s, %v): no alternative without facility equipment", r.short(), r.Library, r.facility())
		}
	}
}

// requireRows checks that the named catalog rows exist in the library.
func requireRows(t *testing.T, byID map[string]catalogRow, library, what string, numbers ...int) []catalogRow {
	t.Helper()
	var out []catalogRow
	for _, n := range numbers {
		r, ok := byID[catalogID(n)]
		if !ok {
			t.Errorf("%s: row %d is missing", what, n)
			continue
		}
		if r.Library != library {
			t.Errorf("%s: row %d %q is in the library %s, not %s", what, n, r.Title, r.Library, library)
		}
		out = append(out, r)
	}
	return out
}

// hack: walk, uphill walk, steady trot stretch, trot and canter stretches, intervals; no facility equipment, as a
// hack has no location with set-up.
func checkKonditionChain(t *testing.T, byID map[string]catalogRow) {
	chain := requireRows(t, byID, "hack", "Kondition chain", 953, 954, 955, 956, 957)
	if len(chain) != 5 {
		return
	}
	if chain[0].Level != "beginner" || chain[4].Level != "advanced" {
		t.Errorf("the chain runs from %s to %s, want beginner to advanced", chain[0].Level, chain[4].Level)
	}
	for i, r := range chain {
		if !r.hasTag("kondition") {
			t.Errorf("%s has no tag kondition", r.short())
		}
		if i > 0 && levelRank(r.Level) < levelRank(chain[i-1].Level) {
			t.Errorf("%s is easier than %s", r.short(), chain[i-1].short())
		}
		if i < 4 && r.Next != chain[i+1].ID {
			t.Errorf("%s should be followed by %s", r.short(), chain[i+1].short())
		}
	}
	for _, r := range byID {
		if r.Library == "hack" && len(r.facility()) > 0 {
			t.Errorf("%s needs %v, but a hack has no location with set-up", r.short(), r.facility())
		}
	}
}

func checkCavalettiVariants(t *testing.T, byID map[string]catalogRow) {
	requireRows(t, byID, "cavaletti", "walk", 820, 890)
	requireRows(t, byID, "cavaletti", "trot, straight", 821, 891, 892)
	requireRows(t, byID, "cavaletti", "trot, bent", 823, 893)
	requireRows(t, byID, "cavaletti", "canter", 824, 895)
	for _, r := range requireRows(t, byID, "cavaletti", "lunge", 896) {
		if !slices.Contains(r.Equipment, "lunge") || !slices.Contains(r.Equipment, "cavaletti") {
			t.Errorf("%s: equipment %v, want lunge and cavaletti", r.short(), r.Equipment)
		}
	}
}

func checkGroundwork(t *testing.T, byID map[string]catalogRow) {
	stations := []struct {
		what      string
		number    int
		equipment string
	}{
		{"tarp", 839, "tarp"}, {"umbrella", 928, "umbrella"}, {"flutter curtain", 929, "obstacles"},
		{"bridge", 930, "obstacles"}, {"narrow passage", 931, "obstacles"}, {"pole labyrinth", 932, "poles"},
		{"trailer loading", 934, "trailer"},
	}
	for _, s := range stations {
		for _, r := range requireRows(t, byID, "groundwork", s.what, s.number) {
			if !slices.Contains(r.Equipment, s.equipment) {
				t.Errorf("%s %s: equipment %v, want %s", s.what, r.short(), r.Equipment, s.equipment)
			}
		}
	}
	requireRows(t, byID, "groundwork", "leading", 830, 920, 924)
}

func checkWestern(t *testing.T, byID map[string]catalogRow) {
	for what, numbers := range map[string][]int{
		"stop": {973}, "back up": {974}, "turn on the forehand": {975}, "turn on the haunches": {983},
		"sidepass": {984}, "circles with speed control": {976}, "lope transitions": {979},
		"trail obstacles": {980, 981, 982},
	} {
		requireRows(t, byID, "western", what, numbers...)
	}
}

// Rehearsals for a show (tag turnier): at least four in dressage and three in jumping, all for the Arbeiten
// part, none demanding except a full course in jumping.
func checkTurnier(t *testing.T, rows []catalogRow) {
	for lib, want := range map[string]int{"dressage": 4, "jumping": 3} {
		n := 0
		for _, r := range rows {
			if r.Library != lib || !r.hasTag("turnier") {
				continue
			}
			n++
			if r.Phase != "main" {
				t.Errorf("%s: phase %s, want main", r.short(), r.Phase)
			}
			fullCourse := lib == "jumping" && strings.HasPrefix(r.Title, "Parcours in ")
			if r.Intensity == "demanding" && !fullCourse {
				t.Errorf("%s is demanding, only a full course may be", r.short())
			}
		}
		if n < want {
			t.Errorf("library %s: %d exercises tagged turnier, want at least %d", lib, n, want)
		}
	}
	for _, r := range rows {
		if r.hasTag("turnier") && r.Library != "dressage" && r.Library != "jumping" {
			t.Errorf("%s (%s) is tagged turnier, but only dressage and jumping have rehearsals", r.short(), r.Library)
		}
	}
}

// Young and senior horses have variants in every library.
func checkSuits(t *testing.T, rows []catalogRow) {
	for _, lib := range catalogLibraries {
		for _, who := range catalogSuits {
			n := 0
			for _, r := range rows {
				if r.Library == lib && slices.Contains(r.Suits, who) {
					n++
				}
			}
			if n == 0 {
				t.Errorf("library %s: no exercise suits %s horses", lib, who)
			}
		}
	}
}
