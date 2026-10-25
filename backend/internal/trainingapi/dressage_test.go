package trainingapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/dressage"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const (
	testsPath = "/api/v1/horses/" + luna + "/dressage-tests"
	reqPath   = "/api/v1/dressage/requirements"
)

func TestDressageCatalog(t *testing.T) {
	e := newEnv(t)
	c := e.call(seed.UserMia, http.MethodGet, "/api/v1/dressage/catalog", "", http.StatusOK)
	if arenas := list(c["arenas"]); len(arenas) != 2 || len(list(obj(arenas[0])["letters"])) < 10 {
		t.Fatalf("arenas = %v", c["arenas"])
	}
	classes := list(c["classes"])
	if len(classes) != len(dressage.Classes) || obj(classes[0])["key"] != "E" || obj(classes[0])["label"] != "Klasse E" {
		t.Fatalf("classes = %v", classes)
	}
	var unsourced int
	for _, m := range list(c["movements"]) {
		mv := obj(m)
		if str(mv["source"]) == "" || str(mv["label"]) == "" {
			t.Fatalf("movement without label or source: %v", mv)
		}
		if mv["lowest_class"] == nil || mv["lowest_class"] == "" {
			unsourced++
			if mv["class_note"] != "Klasse nicht belegt" {
				t.Fatalf("unsourced movement without note: %v", mv)
			}
		}
	}
	if unsourced == 0 {
		t.Fatal("expected movements without a sourced class")
	}
	if len(list(c["requirement_sets"])) != 0 {
		t.Fatalf("requirement sets = %v", c["requirement_sets"])
	}
	// No required movements and no time window of an official class are shipped.
	raw, _ := json.Marshal(c)
	if strings.Contains(string(raw), "min_seconds") {
		t.Fatal("catalog ships a time window")
	}
	e.errCode("", http.MethodGet, "/api/v1/dressage/catalog", "", http.StatusUnauthorized)
}

func TestDressageGeneratePreviewSaveEditDelete(t *testing.T) {
	e := newEnv(t)
	body := `{"kind":"practice","class":"A","arena":"20x40","length":"normal","seed":7}`

	// A preview is not stored and the same seed gives the same plan.
	a := e.call(seed.UserJan, http.MethodPost, testsPath, body, http.StatusOK)
	b := e.call(seed.UserJan, http.MethodPost, testsPath, body, http.StatusOK)
	if a["id"] != nil || a["saved"] != false || a["title"] != "Übungsaufgabe A · 20×40" || num(a["seed"]) != 7 {
		t.Fatalf("preview = %v", a)
	}
	ja, _ := json.Marshal(a["plan"])
	jb, _ := json.Marshal(b["plan"])
	if string(ja) != string(jb) {
		t.Fatal("same seed, different plan")
	}
	plan := obj(a["plan"])
	lines := list(plan["lines"])
	if len(lines) < 10 || num(plan["seconds"]) < 280 {
		t.Fatalf("plan seconds %v, %d lines", plan["seconds"], len(lines))
	}
	first := obj(lines[0])
	if !strings.HasPrefix(str(first["text"]), "Einreiten") || len(list(first["path"])) < 2 || str(first["letters"]) == "" {
		t.Fatalf("first line = %v", first)
	}
	for _, k := range []string{"reins", "gaits", "music", "checks", "requirements"} {
		if plan[k] == nil {
			t.Errorf("plan lacks %s", k)
		}
	}
	e.call(seed.UserJan, http.MethodGet, testsPath, "", http.StatusOK)
	if got := list(e.call(seed.UserJan, http.MethodGet, testsPath, "", http.StatusOK)["tests"]); len(got) != 0 {
		t.Fatalf("a preview was stored: %v", got)
	}

	// Saving stores it and answers 201.
	saved := e.call(seed.UserJan, http.MethodPost, testsPath, `{"kind":"practice","class":"A","arena":"20x40","seed":7,"save":true,"title":"Zuhause"}`, http.StatusCreated)
	id := str(saved["id"])
	if id == "" || saved["saved"] != true || saved["title"] != "Zuhause" {
		t.Fatalf("saved = %v", saved)
	}
	tests := list(e.call(seed.UserJan, http.MethodGet, testsPath, "", http.StatusOK)["tests"])
	if len(tests) != 1 || obj(tests[0])["id"] != id || obj(tests[0])["class"] != "A" || num(obj(tests[0])["seconds"]) < 280 {
		t.Fatalf("list = %v", tests)
	}
	one := e.call(seed.UserJan, http.MethodGet, "/api/v1/dressage-tests/"+id, "", http.StatusOK)
	if len(list(obj(one["plan"])["lines"])) != len(lines) {
		t.Fatal("stored plan differs")
	}

	// A newer test is listed first.
	e.call(seed.UserJan, http.MethodPost, testsPath, `{"kind":"practice","class":"E","arena":"20x60","length":"short","seed":1,"save":true}`, http.StatusCreated)
	tests = list(e.call(seed.UserJan, http.MethodGet, testsPath, "", http.StatusOK)["tests"])
	if len(tests) != 2 || obj(tests[0])["class"] != "E" {
		t.Fatalf("order = %v", tests)
	}

	// PUT: title only, then the same lines again.
	put := "/api/v1/dressage-tests/" + id
	got := e.call(seed.UserJan, http.MethodPut, put, `{"title":"Neuer Titel"}`, http.StatusOK)
	if got["title"] != "Neuer Titel" {
		t.Fatalf("put title = %v", got["title"])
	}
	whole, _ := json.Marshal(map[string]any{"plan": obj(one["plan"])})
	e.call(seed.UserJan, http.MethodPut, put, string(whole), http.StatusOK)

	// A rule breaks: a line removed from the middle no longer chains. 400 with the German reasons.
	// (a circle starts and ends at one letter, so the line removed has to lead somewhere else).
	cut := 5
	for i, l := range list(obj(one["plan"])["lines"]) {
		if i >= 3 && obj(obj(l)["start"])["node"] != obj(obj(l)["end"])["node"] {
			cut = i
			break
		}
	}
	edited := removeLine(t, one, cut)
	out := e.call(seed.UserJan, http.MethodPut, put, edited, http.StatusBadRequest)
	if obj(out["error"])["code"] != "rules_broken" {
		t.Fatalf("error = %v", out["error"])
	}
	reasons := list(out["reasons"])
	if len(reasons) == 0 || !strings.Contains(str(reasons[0]), "Zeile") {
		t.Fatalf("reasons = %v", reasons)
	}
	// Nothing was stored.
	again := e.call(seed.UserJan, http.MethodGet, put, "", http.StatusOK)
	if len(list(obj(again["plan"])["lines"])) != len(lines) || again["title"] != "Neuer Titel" {
		t.Fatal("a broken plan was stored")
	}
	// An unknown figure and a malformed body.
	e.call(seed.UserJan, http.MethodPut, put, `{"plan":{"lines":[{"option":"x","gait":"trot","tempo":"working"}]}}`, http.StatusBadRequest)
	e.errCode(seed.UserJan, http.MethodPut, put, `{"title":""}`, http.StatusBadRequest)
	e.errCode(seed.UserJan, http.MethodPut, put, `{"nope":1}`, http.StatusBadRequest)

	// Delete.
	e.do(seed.UserJan, http.MethodDelete, put, "")
	e.errCode(seed.UserJan, http.MethodGet, put, "", http.StatusNotFound)
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/dressage-tests/not-an-id", "", http.StatusNotFound)
}

// removeLine returns a PUT body with line n (0-based) of the stored test left out.
func removeLine(t *testing.T, test map[string]any, n int) string {
	t.Helper()
	plan := obj(test["plan"])
	lines := list(plan["lines"])
	plan["lines"] = append(append([]any{}, lines[:n]...), lines[n+1:]...)
	b, err := json.Marshal(map[string]any{"plan": plan})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDressageAccess(t *testing.T) {
	e := newEnv(t)
	body := `{"kind":"practice","class":"E","arena":"20x40","seed":3,"save":true}`

	// The owner (Jan, admin) and a rider with the legacy ride rule may generate; so may a rider with log_sessions.
	jans := str(e.call(seed.UserJan, http.MethodPost, testsPath, body, http.StatusCreated)["id"])
	mias := str(e.call(seed.UserMia, http.MethodPost, testsPath, body, http.StatusCreated)["id"])
	if got := list(e.call(seed.UserMia, http.MethodGet, testsPath, "", http.StatusOK)["tests"]); len(got) != 2 {
		t.Fatalf("rider sees %d tests", len(got))
	}
	e.call(seed.UserMia, http.MethodGet, "/api/v1/dressage-tests/"+jans, "", http.StatusOK)
	// A rider deletes only the own tests; the owner deletes all.
	e.errCode(seed.UserMia, http.MethodDelete, "/api/v1/dressage-tests/"+jans, "", http.StatusForbidden)
	if rec := e.do(seed.UserMia, http.MethodDelete, "/api/v1/dressage-tests/"+mias, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("rider deletes own test: %d", rec.Code)
	}
	// Riders need log_sessions (or ride).
	e.exec(`UPDATE horse_riders SET rules = '["groom"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)
	e.errCode(seed.UserMia, http.MethodPost, testsPath, body, http.StatusForbidden)
	e.errCode(seed.UserMia, http.MethodGet, testsPath, "", http.StatusForbidden)
	e.errCode(seed.UserMia, http.MethodGet, "/api/v1/dressage-tests/"+jans, "", http.StatusForbidden)
	e.exec(`UPDATE horse_riders SET rules = '["log_sessions"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)
	e.call(seed.UserMia, http.MethodPost, testsPath, body, http.StatusCreated)
	// Strangers: members without a role, other owners, no session.
	for _, u := range []string{seed.UserSarah, seed.UserAnna} {
		e.errCode(u, http.MethodPost, testsPath, body, http.StatusForbidden)
		e.errCode(u, http.MethodGet, testsPath, "", http.StatusForbidden)
		e.errCode(u, http.MethodGet, "/api/v1/dressage-tests/"+jans, "", http.StatusForbidden)
		e.errCode(u, http.MethodPut, "/api/v1/dressage-tests/"+jans, `{"title":"x"}`, http.StatusForbidden)
		e.errCode(u, http.MethodDelete, "/api/v1/dressage-tests/"+jans, "", http.StatusForbidden)
	}
	e.errCode("", http.MethodPost, testsPath, body, http.StatusUnauthorized)
	e.errCode("", http.MethodGet, "/api/v1/dressage-tests/"+jans, "", http.StatusUnauthorized)
	e.errCode(seed.UserJan, http.MethodPost, "/api/v1/horses/00000000-0000-4000-8000-00000000ffff/dressage-tests", body, http.StatusNotFound)
	// A test of another stable is not found.
	e.exec(`INSERT INTO stables (id, name) VALUES ('00000000-0000-4000-8000-000000000199', 'Anderer Stall')`)
	e.exec(`UPDATE dressage_tests SET stable_id = '00000000-0000-4000-8000-000000000199' WHERE id = $1`, jans)
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/dressage-tests/"+jans, "", http.StatusNotFound)
}

func TestDressageGenerateValidation(t *testing.T) {
	e := newEnv(t)
	for name, body := range map[string]string{
		"kind":                 `{"kind":"x","class":"E","arena":"20x40"}`,
		"class":                `{"kind":"practice","class":"Z","arena":"20x40"}`,
		"practice class above": `{"kind":"practice","class":"M**","arena":"20x40"}`,
		"arena":                `{"kind":"practice","class":"E","arena":"20x80"}`,
		"length":               `{"kind":"practice","class":"E","arena":"20x40","length":"lang"}`,
		"focus":                `{"kind":"practice","class":"E","arena":"20x40","focus_key":"nope"}`,
		"unknown field":        `{"kind":"practice","class":"E","arena":"20x40","x":1}`,
		"both requirements":    `{"kind":"freestyle","class":"A","arena":"20x40","requirements_id":"x","requirements":{"movements":[]}}`,
		"unknown set":          `{"kind":"freestyle","class":"A","arena":"20x40","requirements_id":"00000000-0000-4000-8000-00000000ffff"}`,
		"inline movement":      `{"kind":"freestyle","class":"A","arena":"20x40","requirements":{"movements":[{"key":"x","count":1}]}}`,
		"title":                `{"kind":"practice","class":"E","arena":"20x40","title":"` + strings.Repeat("x", 121) + `"}`,
	} {
		if code := e.errCode(seed.UserJan, http.MethodPost, testsPath, body, http.StatusBadRequest); code == "" {
			t.Errorf("%s: no error code", name)
		}
	}
	// With a focus the practice test is made as well.
	e.call(seed.UserJan, http.MethodPost, testsPath, `{"kind":"practice","class":"L","arena":"20x60","focus_key":"seitengaenge","seed":2}`, http.StatusOK)
}

func TestDressageRequirementSets(t *testing.T) {
	e := newEnv(t)
	set := `{"class":"L*","label":"Meine Pflichtlektionen","min_seconds":200,"max_seconds":400,"movements":[
		{"key":"mittelschritt","count":1,"min_meters":30},{"key":"volte_8_galopp","count":1,"rein":"both"},{"key":"mitteltrab","count":1}]}`
	created := e.call(seed.UserMia, http.MethodPost, reqPath, set, http.StatusCreated)
	id := str(created["id"])
	if id == "" || created["label"] != "Meine Pflichtlektionen" || num(created["min_seconds"]) != 200 || len(list(created["movements"])) != 3 {
		t.Fatalf("created = %v", created)
	}
	// Every member reads the sets of the stable; the catalog lists them.
	if got := list(e.call(seed.UserSarah, http.MethodGet, reqPath, "", http.StatusOK)["requirements"]); len(got) != 1 {
		t.Fatalf("list = %v", got)
	}
	if got := list(e.call(seed.UserSarah, http.MethodGet, "/api/v1/dressage/catalog", "", http.StatusOK)["requirement_sets"]); len(got) != 1 {
		t.Fatalf("catalog sets = %v", got)
	}
	e.call(seed.UserSarah, http.MethodGet, reqPath+"/"+id, "", http.StatusOK)
	// Owners, admins and riders with log_sessions write; a groom does not, nor does a member without a horse.
	e.call(seed.UserAnna, http.MethodPost, reqPath, set, http.StatusCreated)
	e.exec(`UPDATE horse_riders SET rules = '["groom"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)
	e.errCode(seed.UserMia, http.MethodPost, reqPath, set, http.StatusForbidden)
	e.errCode(seed.UserMia, http.MethodPut, reqPath+"/"+id, set, http.StatusForbidden)
	e.errCode(seed.UserMia, http.MethodDelete, reqPath+"/"+id, "", http.StatusForbidden)
	e.exec(`UPDATE horse_riders SET rules = '["log_sessions"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)
	e.call(seed.UserMia, http.MethodPut, reqPath+"/"+id, strings.Replace(set, "Meine", "Geänderte", 1), http.StatusOK)
	e.errCode("", http.MethodGet, reqPath, "", http.StatusUnauthorized)
	// Validation with the German reasons.
	for name, body := range map[string]string{
		"label":    `{"class":"A","label":" ","movements":[]}`,
		"movement": `{"class":"A","label":"x","movements":[{"key":"gibt-es-nicht","count":1}]}`,
		"count":    `{"class":"A","label":"x","movements":[{"key":"arbeitstrab","count":0}]}`,
		"time":     `{"class":"A","label":"x","movements":[],"min_seconds":300,"max_seconds":200}`,
		"class":    `{"class":"Q","label":"x","movements":[]}`,
	} {
		out := e.call(seed.UserJan, http.MethodPost, reqPath, body, http.StatusBadRequest)
		if name != "label" && len(list(out["reasons"])) == 0 {
			t.Errorf("%s: no reasons: %v", name, out)
		}
	}

	// A freestyle from the saved set, and from an inline set; the set is copied into the test.
	fs := `{"kind":"freestyle","class":"L*","arena":"20x40","seed":4,"save":true,"requirements_id":"` + id + `"}`
	test := e.call(seed.UserJan, http.MethodPost, testsPath, fs, http.StatusCreated)
	plan := obj(test["plan"])
	if test["title"] != "Kür L* · 20×40" || obj(test["requirements"])["label"] != "Geänderte Pflichtlektionen" || len(list(plan["requirements"])) != 4 {
		t.Fatalf("freestyle = %v", test)
	}
	for _, r := range list(plan["requirements"]) {
		if obj(r)["met"] != true {
			t.Errorf("requirement not met: %v", r)
		}
	}
	e.call(seed.UserJan, http.MethodPost, testsPath, `{"kind":"freestyle","class":"L*","arena":"20x40","seed":4,"requirements":{"label":"Einmal","movements":[{"key":"arbeitstrab","count":1}],"min_seconds":150,"max_seconds":400}}`, http.StatusOK)
	e.errCode(seed.UserJan, http.MethodPost, testsPath, `{"kind":"freestyle","class":"A","arena":"20x40","requirements_id":"`+id+`"}`, http.StatusBadRequest)
	// Editing a freestyle keeps its requirements: removing the lines of a required movement breaks the rule.
	stored := e.call(seed.UserJan, http.MethodGet, "/api/v1/dressage-tests/"+str(test["id"]), "", http.StatusOK)
	whole, _ := json.Marshal(map[string]any{"plan": obj(stored["plan"])})
	e.call(seed.UserJan, http.MethodPut, "/api/v1/dressage-tests/"+str(test["id"]), string(whole), http.StatusOK)
	// The set can be deleted; the stored test keeps its copy.
	if rec := e.do(seed.UserJan, http.MethodDelete, reqPath+"/"+id, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete set: %d", rec.Code)
	}
	e.errCode(seed.UserJan, http.MethodGet, reqPath+"/"+id, "", http.StatusNotFound)
	kept := e.call(seed.UserJan, http.MethodGet, "/api/v1/dressage-tests/"+str(test["id"]), "", http.StatusOK)
	if obj(kept["requirements"])["label"] != "Geänderte Pflichtlektionen" {
		t.Fatal("the stored test lost its requirement set")
	}
}

func TestDressageFromSessionsOnlyMasteredAndNext(t *testing.T) {
	e := newEnv(t)
	const volte = "00000000-0000-4000-8000-000000000805"
	// Luna mastered the volte (a session rated 3).
	e.call(seed.UserJan, http.MethodPost, "/api/v1/horses/"+luna+"/sessions", `{"activity":"hall","minutes":30,"exercise_id":"`+volte+`","focus_rating":3}`, http.StatusCreated)
	profile := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/training-profile", "", http.StatusOK)
	next := map[string]bool{}
	if prog := obj(profile["progress"]); prog != nil {
		for _, n := range list(prog["next"]) {
			next[str(obj(n)["id"])] = true
		}
	}
	for seedN := 1; seedN <= 4; seedN++ {
		out := e.call(seed.UserJan, http.MethodPost, testsPath, `{"kind":"practice","class":"L","arena":"20x40","seed":`+string(rune('0'+seedN))+`}`, http.StatusOK)
		nextUsed := map[string]bool{}
		for _, l := range list(obj(out["plan"])["lines"]) {
			for _, k := range list(obj(l)["movements"]) {
				m, _ := dressage.MovementByKey(str(k))
				if m.Basis || len(m.ExerciseIDs) == 0 {
					continue
				}
				switch {
				case contains(m.ExerciseIDs, volte):
				case anyIn(m.ExerciseIDs, next):
					nextUsed[m.Key] = true
				default:
					t.Fatalf("seed %d: %s is neither mastered nor next", seedN, m.Key)
				}
			}
		}
		if len(nextUsed) > 2 {
			t.Fatalf("seed %d: next movements %v", seedN, nextUsed)
		}
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func anyIn(s []string, set map[string]bool) bool {
	for _, x := range s {
		if set[x] {
			return true
		}
	}
	return false
}

func TestDressageMovementExercisesExist(t *testing.T) {
	e := newEnv(t)
	for _, m := range dressage.Movements {
		for _, id := range m.ExerciseIDs {
			var n int
			if err := e.pool.QueryRow(t.Context(), `SELECT count(*) FROM exercises WHERE id = $1 AND discipline = 'dressage'`, id).Scan(&n); err != nil || n != 1 {
				t.Errorf("%s: exercise %s is not in the dressage library (%v)", m.Key, id, err)
			}
		}
	}
}
