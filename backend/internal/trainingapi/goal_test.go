package trainingapi_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// goalLibrary: five intermediate lessons of the dressage library (L5 belongs to the stable), a
// chain L1 > L2 > L3 with prerequisites, a warm-up, a beginner and an advanced exercise.
func goalLibrary(e *env) {
	e.clearLibrary()
	// The follow-up has to exist: insert the chain from its end.
	e.addExercise(exSpec{id: exID(3), library: "dressage", level: "intermediate", title: "L3 Traversale", tags: []string{"seitengaenge"}, requires: []string{exID(2)}})
	e.addExercise(exSpec{id: exID(2), library: "dressage", level: "intermediate", title: "L2 Travers", tags: []string{"seitengaenge"}, requires: []string{exID(1)}, next: exID(3)})
	e.addExercise(exSpec{id: exID(1), library: "dressage", level: "intermediate", title: "L1 Schulterherein", tags: []string{"seitengaenge"}, next: exID(2)})
	e.addExercise(exSpec{id: exID(4), library: "dressage", level: "intermediate", title: "L4 Mitteltrab", tags: []string{"schwung"}})
	e.addExercise(exSpec{id: exID(5), library: "dressage", level: "intermediate", title: "L5 Eigene Lektion", tags: []string{"schwung"}, own: true})
	e.addExercise(exSpec{id: exID(6), library: "dressage", level: "intermediate", title: "L6 Aufwärmen", phase: "warmup", tags: []string{"takt"}})
	e.addExercise(exSpec{id: exID(7), library: "dressage", level: "beginner", title: "B1 Volte", tags: []string{"biegung"}})
	e.addExercise(exSpec{id: exID(8), library: "dressage", level: "advanced", title: "A1 Piaffe", tags: []string{"versammlung"}})
}

func refIDs(v any) []string {
	var ids []string
	for _, x := range list(v) {
		ids = append(ids, str(obj(x)["id"]))
	}
	return ids
}

func warningKeys(got map[string]any) []string {
	keys := []string{}
	for _, w := range list(got["warnings"]) {
		keys = append(keys, str(obj(w)["key"]))
	}
	return keys
}

func TestProfileProgressAndGoal(t *testing.T) {
	e := newEnv(t)
	goalLibrary(e)
	path := "/api/v1/horses/" + luna + "/training-profile"
	level := func(l string) string { return strings.Replace(validProfile, `"level": "L"`, `"level": "`+l+`"`, 1) }
	with := func(l, extra string) string {
		return strings.Replace(level(l), `"status": "fit"`, `"status": "fit", `+extra, 1)
	}

	// A beginner is on the way to the class above: the lessons of the intermediate level.
	got := e.call(seed.UserJan, http.MethodPut, path, level("A"), http.StatusOK)
	p := obj(got["progress"])
	if p["level"] != "beginner" || p["goal_level"] != "intermediate" || p["goal_until"] != nil || num(p["mastered"]) != 0 || num(p["total"]) != 5 || p["ready"] != false {
		t.Fatalf("progress = %v", p)
	}
	if p["text"] != "Auf dem Weg zu L: 0 von 5 Lektionen gefestigt." {
		t.Fatalf("text = %q", p["text"])
	}
	// next: the lessons whose prerequisites are mastered, along the progression (chains first).
	if !reflect.DeepEqual(refIDs(p["next"]), []string{exID(1), exID(4), exID(5)}) {
		t.Fatalf("next = %v", p["next"])
	}
	if first := obj(list(p["next"])[0]); first["title"] != "L1 Schulterherein" {
		t.Fatalf("next[0] = %v", first)
	}
	// The stored goal is null: the default applies; riders see the progress too.
	if got["goal_level"] != nil || got["goal_until"] != nil {
		t.Fatalf("stored goal = %v / %v", got["goal_level"], got["goal_until"])
	}
	if mia := obj(e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK)["progress"]); mia["text"] != p["text"] {
		t.Fatalf("rider progress = %v", mia)
	}

	// Two lessons mastered (rated "Sitzt"), one only improved: the next ones open up.
	e.rate(luna, exID(1), 3, "2026-03-20", "hall")
	e.rate(luna, exID(4), 3, "2026-03-21", "hall")
	e.rate(luna, exID(5), 2, "2026-03-22", "hall")
	got = e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK)
	p = obj(got["progress"])
	if num(p["mastered"]) != 2 || p["text"] != "Auf dem Weg zu L: 2 von 5 Lektionen gefestigt." || !reflect.DeepEqual(refIDs(p["next"]), []string{exID(5), exID(2)}) {
		t.Fatalf("progress = %v", p)
	}
	if !reflect.DeepEqual(warningKeys(got), []string{}) {
		t.Fatalf("warnings = %v", got["warnings"])
	}

	// A goal date: "Ziel L bis 15.04.: noch 3 Lektionen."; a date that is over is only stored.
	got = e.call(seed.UserJan, http.MethodPut, path, with("A", `"goal_until": "2026-04-15"`), http.StatusOK)
	p = obj(got["progress"])
	if p["text"] != "Ziel L bis 15.04.: noch 3 Lektionen." || p["goal_until"] != "2026-04-15" || got["goal_until"] != "2026-04-15" || got["goal_level"] != nil {
		t.Fatalf("goal date: %v / %v", p, got["goal_until"])
	}
	got = e.call(seed.UserJan, http.MethodPut, path, with("A", `"goal_until": "2026-03-01"`), http.StatusOK)
	if p := obj(got["progress"]); p["text"] != "Auf dem Weg zu L: 2 von 5 Lektionen gefestigt." || p["goal_until"] != "2026-03-01" {
		t.Fatalf("goal date over: %v", p)
	}
	// The goal stays when a request does not send it; null clears it.
	got = e.call(seed.UserJan, http.MethodPut, path, level("A"), http.StatusOK)
	if got["goal_until"] != "2026-03-01" {
		t.Fatalf("goal_until must be kept: %v", got["goal_until"])
	}
	got = e.call(seed.UserJan, http.MethodPut, path, with("A", `"goal_until": null`), http.StatusOK)
	if got["goal_until"] != nil || obj(got["progress"])["goal_until"] != nil {
		t.Fatalf("goal_until cleared: %v", got["goal_until"])
	}

	// 80 % mastered (4 of 5): ready, with the hint to adjust the level.
	e.rate(luna, exID(2), 3, "2026-03-23", "hall")
	e.rate(luna, exID(5), 3, "2026-03-24", "hall")
	got = e.call(seed.UserJan, http.MethodGet, path, "", http.StatusOK)
	p = obj(got["progress"])
	if p["ready"] != true || num(p["mastered"]) != 4 || p["text"] != "Luna ist bereit für L: 4 von 5 Lektionen gefestigt." || !reflect.DeepEqual(refIDs(p["next"]), []string{exID(3)}) {
		t.Fatalf("ready progress = %v", p)
	}
	w := list(got["warnings"])
	if len(w) != 1 || obj(w[0])["key"] != "level_up" || obj(w[0])["text"] != "Luna hat 4 von 5 L-Lektionen gefestigt – Einstufung auf L anpassen?" {
		t.Fatalf("warnings = %v", got["warnings"])
	}
	// The riders see the hint too (it is no secret of the owner's); with the new level it is gone.
	if mia := e.call(seed.UserMia, http.MethodGet, path, "", http.StatusOK); !reflect.DeepEqual(warningKeys(mia), []string{"level_up"}) {
		t.Fatalf("rider warnings = %v", mia["warnings"])
	}
	got = e.call(seed.UserJan, http.MethodPut, path, level("L"), http.StatusOK)
	if got["progress"] != nil && obj(got["progress"])["goal_level"] != "advanced" {
		t.Fatalf("progress at level L = %v", got["progress"])
	}
	if !reflect.DeepEqual(warningKeys(got), []string{}) {
		t.Fatalf("warnings after the level changed = %v", got["warnings"])
	}
}

func TestProfileGoalLevelRules(t *testing.T) {
	e := newEnv(t)
	goalLibrary(e)
	path := "/api/v1/horses/" + luna + "/training-profile"
	level := func(l, extra string) string {
		return strings.Replace(strings.Replace(validProfile, `"level": "L"`, `"level": "`+l+`"`, 1), `"status": "fit"`, `"status": "fit"`+extra, 1)
	}
	put := func(l, extra string) map[string]any {
		return e.call(seed.UserJan, http.MethodPut, path, level(l, extra), http.StatusOK)
	}

	// An advanced horse has no class above it; an unknown level has no default goal either.
	if got := put("M", ""); got["progress"] != nil {
		t.Fatalf("advanced horse: %v", got["progress"])
	}
	if got := put("", ""); got["progress"] != nil || got["phase"] != "gewoehnung" {
		t.Fatalf("unknown level: %v / %v", got["progress"], got["phase"])
	}
	// A stored goal counts when it is above the level; the same goal as the level is ignored.
	got := put("A", `, "goal_level": "advanced"`)
	if p := obj(got["progress"]); p["goal_level"] != "advanced" || num(p["total"]) != 1 || p["text"] != "Auf dem Weg zu M: 0 von 1 Lektion gefestigt." || got["goal_level"] != "advanced" {
		t.Fatalf("goal advanced: %v / %v", got["progress"], got["goal_level"])
	}
	if got := put("", `, "goal_level": "intermediate"`); obj(got["progress"])["goal_level"] != "intermediate" || num(obj(got["progress"])["total"]) != 5 {
		t.Fatalf("unknown level with a goal: %v", got["progress"])
	}
	if got := put("L", `, "goal_level": "intermediate"`); obj(got["progress"])["goal_level"] != "advanced" || got["goal_level"] != "intermediate" {
		t.Fatalf("a goal that is not above the level is ignored: %v / %v", got["progress"], got["goal_level"])
	}
	if got := put("M", `, "goal_level": "intermediate"`); got["progress"] != nil {
		t.Fatalf("advanced horse with a stale goal: %v", got["progress"])
	}
	// "" and null remove the stored goal; not sending it keeps it.
	put("A", `, "goal_level": "advanced"`)
	if got := put("A", ""); got["goal_level"] != "advanced" {
		t.Fatalf("goal_level must be kept: %v", got["goal_level"])
	}
	if got := put("A", `, "goal_level": ""`); got["goal_level"] != nil || obj(got["progress"])["goal_level"] != "intermediate" {
		t.Fatalf("goal_level cleared with an empty string: %v", got)
	}
	put("A", `, "goal_level": "advanced"`)
	if got := put("A", `, "goal_level": null`); got["goal_level"] != nil {
		t.Fatalf("goal_level cleared with null: %v", got["goal_level"])
	}
	// A goal level without lessons in the home libraries gives no progress.
	e.exec(`DELETE FROM exercises WHERE level = 'advanced'`)
	if got := put("A", `, "goal_level": "advanced"`); got["progress"] != nil {
		t.Fatalf("no lessons: %v", got["progress"])
	}
	// The goal level has to be intermediate or advanced; a rider may not set it.
	e.errCode(seed.UserJan, http.MethodPut, path, level("A", `, "goal_level": "beginner"`), http.StatusBadRequest)
	e.errCode(seed.UserMia, http.MethodPut, path, level("A", `, "goal_level": "advanced"`), http.StatusForbidden)
}

// A jumping horse is trained in the jumping and dressage libraries; the lessons of both count.
func TestProgressHomeLibraries(t *testing.T) {
	e := newEnv(t)
	goalLibrary(e)
	e.addExercise(exSpec{id: exID(20), library: "jumping", level: "intermediate", title: "J1 Reihe", tags: []string{"springgymnastik"}})
	e.addExercise(exSpec{id: exID(21), library: "jumping", level: "intermediate", title: "J2 Oxer", tags: []string{"springgymnastik"}, requires: []string{exID(20)}})
	e.addExercise(exSpec{id: exID(22), library: "cavaletti", level: "intermediate", title: "C1 Stangen", tags: []string{"takt"}})
	e.addExercise(exSpec{id: exID(23), library: "western", level: "intermediate", title: "W1 Stopp", tags: []string{"takt"}})
	balu := "/api/v1/horses/" + seed.HorseBalu + "/training-profile"
	body := func(discipline string) string {
		return strings.Replace(strings.Replace(validProfile, `"discipline": "dressage"`, `"discipline": "`+discipline+`"`, 1), `"level": "L"`, `"level": "A"`, 1)
	}
	total := func(path, discipline string) float64 {
		got := e.call(seed.UserJonas, http.MethodPut, path, body(discipline), http.StatusOK)
		return num(obj(got["progress"])["total"])
	}
	if n := total(balu, "jumping"); n != 7 { // 5 dressage + 2 jumping
		t.Errorf("jumping horse: %v lessons, want 7", n)
	}
	if n := total(balu, "dressage"); n != 5 {
		t.Errorf("dressage horse: %v lessons, want 5", n)
	}
	if n := total(balu, "western"); n != 1 {
		t.Errorf("western horse: %v lessons, want 1 (western only)", n)
	}
	if n := total(balu, "leisure"); n != 5 {
		t.Errorf("leisure horse: %v lessons, want 5 (dressage)", n)
	}
	// A horse without a profile has no progress and starts in the Gewöhnungsphase.
	e.exec(`DELETE FROM training_profiles WHERE horse_id = $1`, seed.HorseNala)
	got := e.call(seed.UserAnna, http.MethodGet, "/api/v1/horses/"+seed.HorseNala+"/training-profile", "", http.StatusOK)
	if got["exists"] != false || got["progress"] != nil || got["phase"] != "gewoehnung" || len(list(got["focuses"])) != 0 {
		t.Fatalf("profile without row = %v", got)
	}
}
