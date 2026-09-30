package trainingapi_test

import (
	"net/http"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func exerciseTitles(out map[string]any) []string {
	var t []string
	for _, x := range list(out["exercises"]) {
		t = append(t, str(obj(x)["title"]))
	}
	return t
}

func TestExerciseLibrary(t *testing.T) {
	e := newEnv(t)
	e.call("", http.MethodGet, "/api/v1/exercises", "", http.StatusUnauthorized)

	all := e.call(seed.UserMia, http.MethodGet, "/api/v1/exercises", "", http.StatusOK) // any stable member
	if n := len(list(all["exercises"])); n != 11 {
		t.Fatalf("library has %d exercises: %v", n, exerciseTitles(all))
	}
	dressage := exerciseTitles(e.call(seed.UserSarah, http.MethodGet, "/api/v1/exercises?discipline=dressage", "", http.StatusOK))
	want := []string{"Übergänge", "Zirkel verkleinern und vergrößern", "Schulterherein", "Travers", "Traversale"}
	if len(dressage) != len(want) {
		t.Fatalf("dressage = %v", dressage)
	}
	for i := range want { // easiest first
		if dressage[i] != want[i] {
			t.Fatalf("dressage = %v, want %v", dressage, want)
		}
	}
	if got := exerciseTitles(e.call(seed.UserJan, http.MethodGet, "/api/v1/exercises?discipline=jumping&level=beginner", "", http.StatusOK)); len(got) != 2 {
		t.Fatalf("jumping beginner = %v", got)
	}
	if got := exerciseTitles(e.call(seed.UserJan, http.MethodGet, "/api/v1/exercises?tag=seitengaenge", "", http.StatusOK)); len(got) != 3 {
		t.Fatalf("tag seitengaenge = %v", got)
	}
	if got := exerciseTitles(e.call(seed.UserJan, http.MethodGet, "/api/v1/exercises?discipline=polo", "", http.StatusOK)); len(got) != 0 {
		t.Fatalf("unknown discipline = %v", got)
	}

	// detail: steps and the follow-up exercise
	d := e.call(seed.UserMia, http.MethodGet, "/api/v1/exercises/"+exSchulterherein, "", http.StatusOK)
	if d["title"] != "Schulterherein" || len(list(d["steps"])) != 4 || d["next_exercise_id"] != exTravers || obj(d["next"])["title"] != "Travers" {
		t.Fatalf("detail = %v", d)
	}
	if d["global"] != true {
		t.Fatalf("global = %v", d["global"])
	}
	e.errCode(seed.UserMia, http.MethodGet, "/api/v1/exercises/00000000-0000-4000-8000-00000000ffff", "", http.StatusNotFound)
	e.errCode(seed.UserMia, http.MethodGet, "/api/v1/exercises/xyz", "", http.StatusNotFound)
}

func TestExerciseStableScope(t *testing.T) {
	e := newEnv(t)
	const other, otherUser, mine = "00000000-0000-4000-8000-0000000001f2", "00000000-0000-4000-8000-0000000002f2", "00000000-0000-4000-8000-000000000f01"
	e.exec(`INSERT INTO stables (id, name) VALUES ($1, 'Fremder Stall')`, other)
	e.exec(`INSERT INTO users (id, stable_id, name, email) VALUES ($1, $2, 'Fremd', 'fremd2@example.org')`, otherUser, other)
	e.exec(`INSERT INTO exercises (id, stable_id, discipline, level, title, steps) VALUES ($1, $2, 'dressage', 'beginner', 'Eigene Übung', '["Schritt 1"]')`, mine, other)

	// visible to the stable that owns it, invisible to everybody else
	if got := exerciseTitles(e.call(otherUser, http.MethodGet, "/api/v1/exercises?discipline=dressage", "", http.StatusOK)); len(got) != 6 {
		t.Fatalf("own stable sees %v", got)
	}
	if got := exerciseTitles(e.call(seed.UserJan, http.MethodGet, "/api/v1/exercises?discipline=dressage", "", http.StatusOK)); len(got) != 5 {
		t.Fatalf("other stable sees %v", got)
	}
	e.call(otherUser, http.MethodGet, "/api/v1/exercises/"+mine, "", http.StatusOK)
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/exercises/"+mine, "", http.StatusNotFound)
	e.errCode(seed.UserJan, http.MethodPost, "/api/v1/horses/"+luna+"/sessions", `{"activity":"hall","minutes":30,"exercise_id":"`+mine+`"}`, http.StatusBadRequest)
}
