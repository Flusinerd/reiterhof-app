package trainingapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// TestTodayShowDayHasNoExercise: on the day of a show the show is the work; the light
// recommendation comes without a library exercise.
func TestTodayShowDayHasNoExercise(t *testing.T) {
	e := newEnv(t)
	body := strings.Replace(validProfile, "2026-03-28", "2026-03-25", 1) // show today
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/training-profile", body, http.StatusOK)
	got := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/today?minutes=30", "", http.StatusOK)
	r := recs(got)
	if len(r) == 0 {
		t.Fatalf("recommendations = %v", got["recommendations"])
	}
	for _, x := range r {
		if x["exercise"] != nil {
			t.Errorf("exercise on a show day: %v", x)
		}
	}
}

// TestWeekShowDay:a show day carries its start time and classes; a unit, focus or exercise
// planned for that day before the show was entered does not show, the claim stays, and no
// new unit can be planned on it.
func TestWeekShowDay(t *testing.T) {
	e := newEnv(t)
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status, focus, exercise_id)
	        VALUES ($1, $2, '2026-03-28', $3, 'hall', 'planned', 'Übergänge', '00000000-0000-4000-8000-000000000806')`,
		seed.StableB, luna, seed.UserMia)
	path := "/api/v1/horses/" + luna
	week := func() map[string]any {
		return weekDays(e.call(seed.UserJan, http.MethodGet, path+"/week?start=2026-03-23", "", http.StatusOK))[5]
	}
	if before := week(); before["activity"] != "hall" || before["exercise"] == nil || before["show"] != nil {
		t.Fatalf("saturday without show = %v", before)
	}

	profile := strings.Replace(validProfile, `"name": "Turnier", "classes": "Dressur L"`,
		`"name": "Turnier", "time": "08:30", "tests": [{"name": "A-Dressur", "time": "09:15"}, {"name": " L-Dressur "}]`, 1)
	got := e.call(seed.UserJan, http.MethodPut, path+"/training-profile", profile, http.StatusOK)
	if sh := obj(list(got["shows"])[0]); sh["time"] != "08:30" || len(list(sh["tests"])) != 2 || obj(list(sh["tests"])[1])["name"] != "L-Dressur" {
		t.Fatalf("shows = %v", got["shows"])
	}

	sat := week()
	sh := obj(sat["show"])
	tests := list(sh["tests"])
	if sh["name"] != "Turnier" || sh["time"] != "08:30" || len(tests) != 2 || obj(tests[0])["name"] != "A-Dressur" || obj(tests[0])["time"] != "09:15" {
		t.Fatalf("show = %v", sat["show"])
	}
	if sat["activity"] != nil || sat["focus"] != nil || sat["exercise"] != nil || sat["status"] != "planned" || obj(sat["user"])["name"] != "Mia" {
		t.Fatalf("saturday = %v", sat)
	}

	// No unit, focus or exercise on a show day; claiming it ("Ich") still works.
	for _, body := range []string{`{"activity":"hall"}`, `{"focus":"Takt"}`, `{"exercise_id":"00000000-0000-4000-8000-000000000806"}`} {
		e.errCode(seed.UserJan, http.MethodPut, path+"/week/2026-03-28", body, http.StatusBadRequest)
	}
	e.call(seed.UserJan, http.MethodPut, path+"/week/2026-03-28", `{}`, http.StatusOK)

	// Removing the show brings the earlier plan back.
	e.call(seed.UserJan, http.MethodPut, path+"/training-profile", strings.Replace(validProfile,
		`"shows": [{"date": "2026-03-28", "name": "Turnier", "classes": "Dressur L", "helper": "Anna"}]`, `"shows": []`, 1), http.StatusOK)
	if sat := week(); sat["activity"] != "hall" || sat["exercise"] == nil {
		t.Fatalf("saturday after removing the show = %v", sat)
	}
}
