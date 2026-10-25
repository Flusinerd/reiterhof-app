package trainingapi_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func exerciseIDs(out map[string]any) []string {
	var ids []string
	for _, x := range list(out["exercises"]) {
		ids = append(ids, str(obj(x)["id"]))
	}
	return ids
}

// planningLibrary is a small library of its own for the filter tests.
func planningLibrary(e *env) {
	e.clearLibrary()
	e.addExercise(exSpec{id: exID(1), library: "dressage", level: "beginner", title: "A Schlangenlinien", phase: "main", intensity: "light",
		tags: []string{"takt", "losgelassenheit"}, minutes: 10, suits: []string{"young"}})
	e.addExercise(exSpec{id: exID(2), library: "dressage", level: "beginner", title: "B Aufwärmen", phase: "warmup", tags: []string{"losgelassenheit"}})
	e.addExercise(exSpec{id: exID(3), library: "dressage", level: "intermediate", title: "C Übergänge", phase: "any", tags: []string{"schwung"}})
	e.addExercise(exSpec{id: exID(4), library: "cavaletti", level: "beginner", title: "D Trabstangen", phase: "main",
		tags: []string{"takt", "rhythmus"}, equipment: []string{"poles", "lunge"}, requires: []string{exID(1)}})
	e.addExercise(exSpec{id: exID(5), library: "groundwork", level: "beginner", title: "E Abkühlen", phase: "cooldown", tags: []string{"gelassenheit"}})
	e.addExercise(exSpec{id: exID(6), library: "dressage", level: "beginner", title: "F Eigene Übung", phase: "main", tags: []string{"takt"}, own: true})
}

func TestExerciseFiltersAndFields(t *testing.T) {
	e := newEnv(t)
	planningLibrary(e)
	get := func(user, query string) []string {
		return exerciseIDs(e.call(user, http.MethodGet, "/api/v1/exercises"+query, "", http.StatusOK))
	}
	ids := func(n ...int) []string {
		var out []string
		for _, i := range n {
			out = append(out, exID(i))
		}
		return out
	}
	same := func(name string, got []string, want ...int) {
		t.Helper()
		g := append([]string(nil), got...)
		w := ids(want...)
		if len(g) != len(w) {
			t.Errorf("%s: %v, want %v", name, g, w)
			return
		}
		seen := map[string]bool{}
		for _, id := range g {
			seen[id] = true
		}
		for _, id := range w {
			if !seen[id] {
				t.Errorf("%s: %v, want %v", name, g, w)
				return
			}
		}
	}

	same("all, own row included", get(seed.UserJan, ""), 1, 2, 3, 4, 5, 6)
	same("library", get(seed.UserJan, "?library=cavaletti"), 4)
	same("discipline is the older name", get(seed.UserJan, "?discipline=cavaletti"), 4)
	same("library wins over discipline", get(seed.UserJan, "?library=groundwork&discipline=cavaletti"), 5)
	same("level", get(seed.UserJan, "?level=intermediate"), 3)
	same("tag", get(seed.UserJan, "?tag=takt"), 1, 4, 6)
	// phase: Lösen, Arbeiten, Entspannen also list the exercises that fit every part
	same("warmup", get(seed.UserJan, "?phase=warmup"), 2, 3)
	same("main", get(seed.UserJan, "?phase=main"), 1, 3, 4, 6)
	same("cooldown", get(seed.UserJan, "?phase=cooldown"), 3, 5)
	same("any", get(seed.UserJan, "?phase=any"), 3)
	same("unknown phase", get(seed.UserJan, "?phase=sprint"))
	// focus: the exercises with one of the focus's tags
	same("focus takt (tags takt, rhythmus)", get(seed.UserJan, "?focus=takt"), 1, 4, 6)
	same("focus schwung", get(seed.UserJan, "?focus=schwung"), 3)
	same("focus mobilitaet (mobilitaet, losgelassenheit)", get(seed.UserJan, "?focus=mobilitaet"), 1, 2)
	same("unknown focus", get(seed.UserJan, "?focus=tempo"))
	same("combined", get(seed.UserJan, "?focus=takt&library=dressage&phase=main"), 1, 6)
	// A row of the stable is global once its stable is NULL.
	e.exec(`UPDATE exercises SET stable_id = NULL WHERE id = $1`, exID(6))
	same("global now", get(seed.UserJan, "?library=dressage&phase=main"), 1, 3, 6)

	// Items carry the new fields; the older ones stay.
	byID := map[string]map[string]any{}
	for _, x := range list(e.call(seed.UserJan, http.MethodGet, "/api/v1/exercises", "", http.StatusOK)["exercises"]) {
		byID[str(obj(x)["id"])] = obj(x)
	}
	a, d := byID[exID(1)], byID[exID(4)]
	if a["library"] != "dressage" || a["discipline"] != "dressage" || a["phase"] != "main" || num(a["minutes"]) != 10 || a["intensity"] != "light" ||
		!reflect.DeepEqual(strs(a["suits"]), []string{"young"}) || a["summary"] != "Wozu" || !reflect.DeepEqual(strs(a["equipment"]), []string{}) ||
		num(a["step_count"]) != 2 || a["steps"] != nil || a["global"] != true {
		t.Fatalf("exercise A = %v", a)
	}
	if d["library"] != "cavaletti" || d["minutes"] != nil || d["intensity"] != "normal" || !reflect.DeepEqual(strs(d["equipment"]), []string{"poles", "lunge"}) {
		t.Fatalf("exercise D = %v", d)
	}
	if _, has := a["available"]; has {
		t.Errorf("available without a horse: %v", a)
	}
	if _, has := a["requires"]; has {
		t.Errorf("the list carries no prerequisites: %v", a)
	}
}

func TestExerciseDetailRequiresSourcesAlternatives(t *testing.T) {
	e := newEnv(t)
	planningLibrary(e)
	// Alternatives for D (cavaletti, poles, tags takt and rhythmus): the same library, phase main or any,
	// at least two shared tags, no facility equipment, a level at or below, best first.
	e.addExercise(exSpec{id: exID(10), library: "cavaletti", level: "beginner", title: "G Zirkel", tags: []string{"takt", "rhythmus", "balance"}, equipment: []string{"lunge"}})
	e.addExercise(exSpec{id: exID(11), library: "cavaletti", level: "beginner", title: "H Nur ein Tag", tags: []string{"takt"}})
	e.addExercise(exSpec{id: exID(12), library: "cavaletti", level: "beginner", title: "I Mit Cavaletti", tags: []string{"takt", "rhythmus"}, equipment: []string{"cavaletti"}})
	e.addExercise(exSpec{id: exID(13), library: "cavaletti", level: "beginner", title: "J Anderer Teil", phase: "cooldown", tags: []string{"takt", "rhythmus"}})
	e.addExercise(exSpec{id: exID(14), library: "cavaletti", level: "beginner", title: "K Immer", phase: "any", tags: []string{"takt", "rhythmus"}})
	e.addExercise(exSpec{id: exID(15), library: "cavaletti", level: "intermediate", title: "L Zu schwer", tags: []string{"takt", "rhythmus"}})
	e.addExercise(exSpec{id: exID(16), library: "dressage", level: "beginner", title: "M Andere Bibliothek", tags: []string{"takt", "rhythmus"}})
	e.exec(`UPDATE exercises SET sources = '{https://example.org/a,Buch B}' WHERE id = $1`, exID(4))

	d := e.call(seed.UserMia, http.MethodGet, "/api/v1/exercises/"+exID(4), "", http.StatusOK)
	if req := list(d["requires"]); len(req) != 1 || obj(req[0])["id"] != exID(1) || obj(req[0])["title"] != "A Schlangenlinien" {
		t.Fatalf("requires = %v", d["requires"])
	}
	if !reflect.DeepEqual(strs(d["sources"]), []string{"https://example.org/a", "Buch B"}) {
		t.Fatalf("sources = %v", d["sources"])
	}
	alts := list(d["alternatives"])
	// Three shared tags first (G), then two: the level, then the title (K before J is no candidate).
	if len(alts) != 2 || obj(alts[0])["id"] != exID(10) || obj(alts[1])["id"] != exID(14) {
		t.Fatalf("alternatives = %v", d["alternatives"])
	}
	if g := obj(alts[0]); g["title"] != "G Zirkel" || !reflect.DeepEqual(strs(g["equipment"]), []string{"lunge"}) {
		t.Fatalf("alternative G = %v", g)
	}
	if d["phase"] != "main" || d["library"] != "cavaletti" || num(d["step_count"]) != 2 || len(list(d["steps"])) != 2 {
		t.Fatalf("detail = %v", d)
	}

	// An exercise without facility equipment, prerequisites or sources: empty lists, not null.
	a := e.call(seed.UserMia, http.MethodGet, "/api/v1/exercises/"+exID(1), "", http.StatusOK)
	for _, k := range []string{"requires", "sources", "alternatives"} {
		if a[k] == nil || len(list(a[k])) != 0 {
			t.Errorf("%s = %v, want []", k, a[k])
		}
	}
	// Prerequisites outside the stable's view are left out.
	e.exec(`UPDATE exercises SET requires = ARRAY[$2::uuid, $3::uuid] WHERE id = $1`, exID(4), exID(1), "00000000-0000-4000-8000-00000000abcd")
	if req := list(e.call(seed.UserMia, http.MethodGet, "/api/v1/exercises/"+exID(4), "", http.StatusOK)["requires"]); len(req) != 1 {
		t.Fatalf("requires = %v", req)
	}
}

func TestExerciseAvailable(t *testing.T) {
	e := newEnv(t)
	e.clearLibrary()
	e.addExercise(exSpec{id: exID(1), library: "dressage", level: "beginner", title: "Stangen", tags: []string{"takt"}, equipment: []string{"poles"}})
	e.addExercise(exSpec{id: exID(2), library: "dressage", level: "beginner", title: "Cavaletti", tags: []string{"takt"}, equipment: []string{"cavaletti", "lunge"}})
	e.addExercise(exSpec{id: exID(3), library: "dressage", level: "beginner", title: "Frei", tags: []string{"takt"}, equipment: []string{"lunge"}})
	e.addExercise(exSpec{id: exID(4), library: "lunge", level: "beginner", title: "Longe mit Stangen", tags: []string{"takt"}, equipment: []string{"poles"}})
	e.addExercise(exSpec{id: exID(5), library: "hack", level: "beginner", title: "Ausritt mit Hindernis", tags: []string{"takt"}, equipment: []string{"obstacles"}})
	e.exec(`UPDATE stables SET training_equipment = '{}'`)
	path := "/api/v1/exercises?horse=" + luna
	available := func(user string) map[string]bool {
		out := map[string]bool{}
		for _, x := range list(e.call(user, http.MethodGet, path, "", http.StatusOK)["exercises"]) {
			ok, present := obj(x)["available"].(bool)
			if !present {
				t.Fatalf("no available in %v", x)
			}
			out[str(obj(x)["title"])] = ok
		}
		return out
	}
	set := func(body string) { e.call(seed.UserJan, http.MethodPut, equipmentPath, body, http.StatusOK) }
	check := func(name string, got map[string]bool, want map[string]bool) {
		t.Helper()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: available = %v, want %v", name, got, want)
		}
	}

	// Defaults: poles in the hall and the arena (dressage), none for the lunge; cavaletti and
	// obstacles nowhere; personal equipment and none are always there.
	check("defaults", available(seed.UserJan), map[string]bool{"Stangen": true, "Cavaletti": false, "Frei": true, "Longe mit Stangen": false, "Ausritt mit Hindernis": false})
	// The stable has cavaletti in the hall and poles at the lunge.
	set(`{"locations": {"hall": {"equipment": ["cavaletti"]}, "lunge": {"equipment": ["poles"]}}}`)
	check("hall with cavaletti", available(seed.UserJan), map[string]bool{"Stangen": true, "Cavaletti": true, "Frei": true, "Longe mit Stangen": true, "Ausritt mit Hindernis": false})
	// A shared hall allows no set-up, but the arena (dressage is used there too) still has poles.
	set(`{"locations": {"hall": {"equipment": ["cavaletti"], "shared": true}, "lunge": {"equipment": ["poles"]}}}`)
	check("hall shared", available(seed.UserJan), map[string]bool{"Stangen": true, "Cavaletti": false, "Frei": true, "Longe mit Stangen": true, "Ausritt mit Hindernis": false})
	set(`{"locations": {"hall": {"equipment": ["cavaletti"], "shared": true}, "arena": {"shared": true}}}`)
	check("hall and arena shared", available(seed.UserJan), map[string]bool{"Stangen": false, "Cavaletti": false, "Frei": true, "Longe mit Stangen": false, "Ausritt mit Hindernis": false})
	// Riders and the owner may ask; members without a role and other stables may not.
	available(seed.UserMia)
	e.errCode(seed.UserSarah, http.MethodGet, path, "", http.StatusForbidden)
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/exercises?horse=00000000-0000-4000-8000-00000000ffff", "", http.StatusNotFound)
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/exercises?horse=nope", "", http.StatusNotFound)
	// Without a horse the field is absent.
	for _, x := range list(e.call(seed.UserSarah, http.MethodGet, "/api/v1/exercises", "", http.StatusOK)["exercises"]) {
		if _, has := obj(x)["available"]; has {
			t.Fatalf("available without a horse: %v", x)
		}
	}
	// A dressage horse uses the jumping library only on jumping days; for a jumping horse it is
	// used in the hall and arena too, so an exercise of it can find its jumps there.
	e.addExercise(exSpec{id: exID(6), library: "jumping", level: "beginner", title: "Sprung", tags: []string{"takt"}, equipment: []string{"jumps"}})
	balu := "/api/v1/exercises?horse=" + seed.HorseBalu
	sprung := func(user, path string) bool {
		for _, x := range list(e.call(user, http.MethodGet, path, "", http.StatusOK)["exercises"]) {
			if obj(x)["title"] == "Sprung" {
				return obj(x)["available"] == true
			}
		}
		t.Fatal("no jump exercise")
		return false
	}
	set(`{"locations": {"hall": {"equipment": []}, "arena": {"equipment": []}, "jumping": {"equipment": ["jumps"]}}}`)
	if !sprung(seed.UserJan, path) || !sprung(seed.UserJonas, balu) {
		t.Error("jumps at the jumping place are available")
	}
	set(`{"locations": {"hall": {"equipment": []}, "arena": {"equipment": []}, "jumping": {"equipment": ["jumps"], "shared": true}}}`)
	if sprung(seed.UserJan, path) || sprung(seed.UserJonas, balu) {
		t.Error("a shared jumping place has no set-up")
	}
	set(`{"locations": {"hall": {"equipment": ["jumps"]}, "jumping": {"equipment": [], "shared": true}}}`)
	if sprung(seed.UserJan, path) {
		t.Error("a dressage horse does not jump in the hall")
	}
	if !sprung(seed.UserJonas, balu) {
		t.Error("a jumping horse finds jumps in the hall")
	}
}
