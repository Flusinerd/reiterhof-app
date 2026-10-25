package trainingapi_test

import (
	"net/http"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime/realtimetest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
	"github.com/Flusinerd/reiterhof-app/backend/internal/trainingapi"
)

func TestTrainingChangedEvents(t *testing.T) {
	e := newEnv(t)
	w := realtimetest.Watch(t, e.pool, seed.StableB)
	expect := func(what, horseID string) {
		t.Helper()
		data := w.Next(trainingapi.EventChanged)
		if data["horse_id"] != horseID || len(data) != 1 {
			t.Fatalf("%s: data = %v, want only horse_id %s", what, data, horseID)
		}
	}
	horse := "/api/v1/horses/" + luna

	e.call(seed.UserJan, http.MethodPut, horse+"/week/2026-03-26", `{"status":"planned","activity":"hall"}`, http.StatusOK)
	expect("plan a day", luna)
	e.call(seed.UserMia, http.MethodPut, horse+"/week/2026-03-27", `{"status":"planned"}`, http.StatusOK) // a rider takes a free day
	expect("take a day", luna)
	e.call(seed.UserJan, http.MethodPut, horse+"/week/2026-03-26", `{"status":"open"}`, http.StatusOK)
	expect("release a day", luna)
	e.call(seed.UserMia, http.MethodPost, horse+"/sessions", `{"activity":"hall","minutes":45}`, http.StatusCreated)
	expect("log a session", luna)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+fanta+"/training-profile", validProfile, http.StatusOK)
	expect("save the profile", fanta)

	// Refused, invalid or read-only requests publish nothing.
	e.errCode(seed.UserSarah, http.MethodPut, horse+"/week/2026-03-28", `{"status":"planned"}`, http.StatusForbidden)
	e.errCode(seed.UserSarah, http.MethodPost, horse+"/sessions", `{"activity":"hall","minutes":30}`, http.StatusForbidden)
	e.errCode(seed.UserJan, http.MethodPost, horse+"/sessions", `{"activity":"hall","minutes":0}`, http.StatusBadRequest)
	e.call(seed.UserJan, http.MethodGet, horse+"/week", "", http.StatusOK)
	e.call(seed.UserJan, http.MethodPost, horse+"/week/plan", "", http.StatusOK) // proposes only, stores nothing
	w.Quiet()
}
