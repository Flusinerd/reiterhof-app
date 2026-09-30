package trainingapi_test

import (
	"context"
	"math"
	"net/http"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestQuickLogComputesLoadAndMarksSlot(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/sessions"

	// Jan planned hall for today.
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-25", `{"status":"planned","activity":"hall"}`, http.StatusOK)

	got := e.call(seed.UserJan, http.MethodPost, path, `{"activity":"hall","minutes":45,"canter_share":0.2}`, http.StatusCreated)
	s := obj(got["session"])
	want := 45 * (0.8 + 0.4*0.2) // load.Score
	if math.Abs(num(s["load"])-round1(want)) > 1e-9 || s["source"] != "quick" || s["intensity"] != "medium" || s["day"] != "2026-03-25" {
		t.Fatalf("session = %v (want load %.1f)", s, want)
	}
	if got["next_progression"] != nil {
		t.Fatalf("no progression without exercise: %v", got["next_progression"])
	}
	var stored float64
	var slotStatus string
	if err := e.pool.QueryRow(context.Background(), `SELECT load_score::float8 FROM sessions WHERE id = $1`, s["id"]).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if math.Abs(stored-want) > 1e-6 {
		t.Fatalf("stored load_score = %v, want %v", stored, want)
	}
	if err := e.pool.QueryRow(context.Background(), `SELECT status FROM week_slots WHERE horse_id = $1 AND day = '2026-03-25'`, luna).Scan(&slotStatus); err != nil || slotStatus != "done" {
		t.Fatalf("slot status = %q, err %v", slotStatus, err)
	}

	// A rider with the legacy "ride" rule may log; the slot is created when nobody planned the day.
	e.call(seed.UserMia, http.MethodPost, path, `{"activity":"walker","minutes":30,"started_at":"2026-03-24T17:00:00+01:00"}`, http.StatusCreated)
	var slotUser string
	if err := e.pool.QueryRow(context.Background(), `SELECT user_id::text FROM week_slots WHERE horse_id = $1 AND day = '2026-03-24'`, luna).Scan(&slotUser); err != nil || slotUser != seed.UserMia {
		t.Fatalf("slot user = %q, err %v", slotUser, err)
	}

	// Riders need log_sessions (or the legacy ride key); members never may.
	e.exec(`UPDATE horse_riders SET rules = '["groom"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)
	e.errCode(seed.UserMia, http.MethodPost, path, `{"activity":"hall","minutes":30}`, http.StatusForbidden)
	e.exec(`UPDATE horse_riders SET rules = '["log_sessions"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)
	e.call(seed.UserMia, http.MethodPost, path, `{"activity":"hall","minutes":30}`, http.StatusCreated)
	e.errCode(seed.UserSarah, http.MethodPost, path, `{"activity":"hall","minutes":30}`, http.StatusForbidden)
	e.errCode("", http.MethodPost, path, `{"activity":"hall","minutes":30}`, http.StatusUnauthorized)
	e.errCode(seed.UserAnna, http.MethodPost, path, `{"activity":"hall","minutes":30}`, http.StatusForbidden)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func TestSessionValidation(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/sessions"
	for name, body := range map[string]string{
		"activity rest":      `{"activity":"rest","minutes":30}`,
		"activity unknown":   `{"activity":"polo","minutes":30}`,
		"minutes zero":       `{"activity":"hall","minutes":0}`,
		"minutes too long":   `{"activity":"hall","minutes":601}`,
		"canter above one":   `{"activity":"hall","minutes":30,"canter_share":1.5}`,
		"gait sum":           `{"activity":"hall","minutes":30,"gait_shares":{"walk":0.6,"trot":0.6}}`,
		"gait unknown":       `{"activity":"hall","minutes":30,"gait_shares":{"gallop":0.5}}`,
		"rein":               `{"activity":"hall","minutes":30,"rein_changes":[{"rein":"up","minutes":3}]}`,
		"feel":               `{"activity":"hall","minutes":30,"feel":"happy"}`,
		"focus":              `{"activity":"hall","minutes":30,"focus_rating":4}`,
		"future":             `{"activity":"hall","minutes":30,"started_at":"2026-03-26T10:00:00+01:00"}`,
		"too old":            `{"activity":"hall","minutes":30,"started_at":"2026-01-01T10:00:00+01:00"}`,
		"unknown exercise":   `{"activity":"hall","minutes":30,"exercise_id":"00000000-0000-4000-8000-00000000ffff"}`,
		"malformed exercise": `{"activity":"hall","minutes":30,"exercise_id":"x"}`,
		"unknown field":      `{"activity":"hall","minutes":30,"load_score":1}`,
	} {
		if code := e.errCode(seed.UserJan, http.MethodPost, path, body, http.StatusBadRequest); code != "validation_failed" && code != "invalid_json" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM sessions`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("invalid requests stored %d sessions (err %v)", n, err)
	}
}

func TestFinishedSessionAndNextProgression(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/sessions"
	body := `{
	  "activity": "hall", "minutes": 50, "started_at": "2026-03-25T08:30:00+01:00",
	  "gait_shares": {"walk": 0.3, "trot": 0.5, "canter": 0.2},
	  "rein_changes": [{"rein":"left","minutes":20},{"rein":"right","minutes":22},{"rein":"left","minutes":8}],
	  "feel": "loose", "focus_rating": 3, "exercise_id": "` + exSchulterherein + `",
	  "note": "Schulterherein klappt im Schritt", "visible_to_rider": true
	}`
	got := e.call(seed.UserJan, http.MethodPost, path, body, http.StatusCreated)
	s := obj(got["session"])
	if s["source"] != "tracked" || s["feel"] != "loose" || num(s["focus_rating"]) != 3 || num(s["canter_share"]) != 0.2 || len(list(s["rein_changes"])) != 3 {
		t.Fatalf("session = %v", s)
	}
	if math.Abs(num(s["load"])-round1(50*(0.8+0.4*0.2))) > 1e-9 {
		t.Fatalf("load = %v", s["load"])
	}
	next := obj(got["next_progression"])
	if next["title"] != "Travers" || next["id"] != exTravers {
		t.Fatalf("next_progression = %v", got["next_progression"])
	}

	// rating below "Sitzt" suggests nothing; the last exercise of a chain has no follow-up
	got = e.call(seed.UserJan, http.MethodPost, path, `{"activity":"hall","minutes":30,"focus_rating":2,"exercise_id":"`+exSchulterherein+`"}`, http.StatusCreated)
	if got["next_progression"] != nil {
		t.Fatalf("focus 2: %v", got["next_progression"])
	}
	got = e.call(seed.UserJan, http.MethodPost, path, `{"activity":"hall","minutes":30,"focus_rating":3,"exercise_id":"00000000-0000-4000-8000-000000000812"}`, http.StatusCreated)
	if got["next_progression"] != nil {
		t.Fatalf("last exercise: %v", got["next_progression"])
	}

	// The suggestion for tomorrow skips exercises the horse has mastered (after this much
	// training only lunging is left, which gets exercises from the lunging library).
	e.call(seed.UserJan, http.MethodPost, path, `{"activity":"lunge","minutes":20,"focus_rating":3,"exercise_id":"`+exAufwaermen+`"}`, http.StatusCreated)
	e.rain(true, 5)
	today := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/today?minutes=45", "", http.StatusOK)
	var lunge map[string]any
	for _, r := range recs(today) {
		if r["activity"] == "lunge" {
			lunge = obj(r["exercise"])
		}
	}
	if lunge["title"] != "Handwechsel an der Longe" {
		t.Fatalf("exercise = %v", today["recommendations"])
	}
}

func TestSessionVisibility(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + luna + "/sessions"
	e.call(seed.UserJan, http.MethodPost, path, `{"activity":"hall","minutes":40,"started_at":"2026-03-24T17:00:00+01:00","visible_to_rider":true}`, http.StatusCreated)
	e.call(seed.UserJan, http.MethodPost, path, `{"activity":"arena","minutes":45,"started_at":"2026-03-23T17:00:00+01:00","visible_to_rider":false}`, http.StatusCreated)
	e.call(seed.UserMia, http.MethodPost, path, `{"activity":"walker","minutes":30,"started_at":"2026-03-22T17:00:00+01:00","visible_to_rider":false}`, http.StatusCreated)

	activities := func(user string) []string {
		var out []string
		for _, s := range list(e.call(user, http.MethodGet, path, "", http.StatusOK)["sessions"]) {
			out = append(out, str(obj(s)["activity"]))
		}
		return out
	}
	if got := activities(seed.UserJan); len(got) != 3 || got[0] != "hall" { // newest first
		t.Fatalf("owner sees %v", got)
	}
	if got := activities(seed.UserMia); len(got) != 2 || got[0] != "hall" || got[1] != "walker" {
		t.Fatalf("rider sees %v, want the visible one and her own hidden one", got)
	}
	e.errCode(seed.UserSarah, http.MethodGet, path, "", http.StatusForbidden)

	// limit and paging
	if got := list(e.call(seed.UserJan, http.MethodGet, path+"?limit=1", "", http.StatusOK)["sessions"]); len(got) != 1 {
		t.Fatalf("limit: %d", len(got))
	}
	if got := list(e.call(seed.UserJan, http.MethodGet, path+"?before=2026-03-23T20:00:00Z", "", http.StatusOK)["sessions"]); len(got) != 2 {
		t.Fatalf("before: %d", len(got))
	}
	e.errCode(seed.UserJan, http.MethodGet, path+"?limit=0", "", http.StatusBadRequest)

	// The week counts every session for done/load, but shows the rider no details of hidden ones.
	week := e.call(seed.UserMia, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-03-23", "", http.StatusOK)
	days := list(week["days"])
	mon := obj(days[0]) // 23rd: Jan's hidden arena session
	if mon["status"] != "done" || mon["user"] != nil || mon["activity"] != nil {
		t.Fatalf("hidden session in rider week: %v", mon)
	}
	if seg := obj(list(week["segments"])[0]); num(seg["sessions"]) != 1 || num(seg["load"]) == 0 {
		t.Fatalf("segment = %v", seg)
	}
	sun := obj(days[6]) // 29th, nothing
	if sun["status"] != "open" {
		t.Fatalf("sun = %v", sun)
	}
	ownerMon := obj(list(e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-03-23", "", http.StatusOK)["days"])[0])
	if ownerMon["activity"] != "arena" || obj(ownerMon["user"])["name"] != "Jan" {
		t.Fatalf("owner week monday = %v", ownerMon)
	}
}
