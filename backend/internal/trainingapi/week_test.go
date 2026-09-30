package trainingapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func weekDays(out map[string]any) []map[string]any {
	var d []map[string]any
	for _, x := range list(out["days"]) {
		d = append(d, obj(x))
	}
	return d
}

// insertSession stores a session at an exact instant (the API refuses the future).
func (e *env) insertSession(horse, user, activity string, at time.Time, minutes int, loadScore float64, visible bool) {
	e.t.Helper()
	e.exec(`INSERT INTO sessions (stable_id, horse_id, user_id, activity, started_at, duration_min, load_score, visible_to_rider)
	        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, seed.StableB, horse, user, activity, at, minutes, loadScore, visible)
}

func TestWeekAggregation(t *testing.T) {
	e := newEnv(t)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/training-profile", validProfile, http.StatusOK) // show Sat 28th
	e.insertSession(luna, seed.UserJan, "hall", time.Date(2026, 3, 23, 17, 0, 0, 0, berlin), 45, 39.6, true)
	e.insertSession(luna, seed.UserMia, "walker", time.Date(2026, 3, 23, 18, 0, 0, 0, berlin), 30, 9, true) // second session on Monday
	e.insertSession(luna, seed.UserMia, "arena", time.Date(2026, 3, 24, 17, 0, 0, 0, berlin), 70, 70, true)
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status) VALUES ($1, $2, '2026-03-26', $3, 'hack', 'planned')`, seed.StableB, luna, seed.UserMia)

	// any day of the week works as start; the week is Monday to Sunday
	got := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-03-27", "", http.StatusOK)
	if got["start"] != "2026-03-23" || got["end"] != "2026-03-29" {
		t.Fatalf("range = %v .. %v", got["start"], got["end"])
	}
	d := weekDays(got)
	if len(d) != 7 || len(list(got["segments"])) != 7 {
		t.Fatalf("days = %d, segments = %d", len(d), len(list(got["segments"])))
	}
	status := func() string {
		var s []string
		for _, x := range d {
			s = append(s, str(x["status"]))
		}
		return strings.Join(s, ",")
	}
	// Mon/Tue done, Wed today, Thu planned by Mia, Fri open, Sat show day (open), Sun: rest after the show
	if want := "done,done,today,planned,open,open,rest"; status() != want {
		t.Fatalf("statuses = %s, want %s", status(), want)
	}
	if d[6]["rest_reason"] != "after_show" {
		t.Fatalf("sunday = %v", d[6])
	}
	if sh := obj(d[5]["show"]); sh["name"] != "Turnier" || sh["classes"] != "Dressur L" || sh["helper"] != "Anna" {
		t.Fatalf("show = %v", d[5]["show"])
	}
	if d[3]["activity"] != "hack" || obj(d[3]["user"])["name"] != "Mia" || d[3]["is_me"] != false {
		t.Fatalf("thursday = %v", d[3])
	}
	if d[1]["minutes"] != float64(70) || obj(d[1]["user"])["name"] != "Mia" {
		t.Fatalf("tuesday = %v", d[1])
	}
	segs := list(got["segments"])
	if obj(segs[0])["level"] != "medium" || obj(segs[1])["level"] != "intense" || obj(segs[2])["level"] != "none" {
		t.Fatalf("segments = %v", segs)
	}
	if mon := d[0]; mon["activity"] != "hall" || obj(mon["user"])["name"] != "Jan" || num(mon["minutes"]) != 75 {
		t.Fatalf("two sessions on one day: the first names the row, minutes add up: %v", mon)
	}
	if num(got["sessions"]) != 3 || !strings.Contains(str(got["assessment"]), "3 von 5 Einheiten") {
		t.Fatalf("sessions = %v, assessment = %q", got["sessions"], got["assessment"])
	}

	// default week is the current one; a bad date is rejected; members without role are not allowed
	if e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week", "", http.StatusOK)["start"] != "2026-03-23" {
		t.Fatal("default week")
	}
	e.errCode(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=morgen", "", http.StatusBadRequest)
	e.errCode(seed.UserSarah, http.MethodGet, "/api/v1/horses/"+luna+"/week", "", http.StatusForbidden)
	// riders may read the week
	e.call(seed.UserMia, http.MethodGet, "/api/v1/horses/"+luna+"/week", "", http.StatusOK)
}

// A week with a DST change still has seven days, and sessions around midnight land on
// the stable-local calendar day.
func TestWeekDST(t *testing.T) {
	cases := []struct {
		name  string
		start string
		early time.Time // just after local midnight on Sunday
		late  time.Time // just before local midnight on Sunday
		sunHr int       // hours in that Sunday
	}{
		{"spring (23 h Sunday)", "2026-03-23", time.Date(2026, 3, 29, 0, 30, 0, 0, berlin), time.Date(2026, 3, 29, 23, 30, 0, 0, berlin), 23},
		{"autumn (25 h Sunday)", "2026-10-19", time.Date(2026, 10, 25, 0, 30, 0, 0, berlin), time.Date(2026, 10, 25, 23, 30, 0, 0, berlin), 25},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			startOfSun := time.Date(c.early.Year(), c.early.Month(), c.early.Day(), 0, 0, 0, 0, berlin)
			if h := startOfSun.AddDate(0, 0, 1).Sub(startOfSun) / time.Hour; int(h) != c.sunHr {
				t.Fatalf("test setup: Sunday has %d h, want %d", h, c.sunHr)
			}
			// Saturday late evening (UTC already Sunday? no: still Saturday), Sunday early and late.
			sat := time.Date(c.early.Year(), c.early.Month(), c.early.Day()-1, 23, 30, 0, 0, berlin)
			nextMon := time.Date(c.early.Year(), c.early.Month(), c.early.Day()+1, 0, 30, 0, 0, berlin)
			e.insertSession(luna, seed.UserJan, "hall", sat, 30, 24, true)
			e.insertSession(luna, seed.UserJan, "hall", c.early, 30, 24, true)
			e.insertSession(luna, seed.UserJan, "hack", c.late, 60, 40, true)
			e.insertSession(luna, seed.UserJan, "hall", nextMon, 30, 24, true) // next week

			got := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start="+c.start, "", http.StatusOK)
			d := weekDays(got)
			if len(d) != 7 || d[0]["date"] != c.start || d[6]["weekday"] != float64(6) {
				t.Fatalf("days = %v", d)
			}
			segs := list(got["segments"])
			counts := ""
			for _, s := range segs {
				counts += string('0' + rune(int(num(obj(s)["sessions"]))))
			}
			if counts != "0000012" {
				t.Fatalf("sessions per day = %s, want 0000012 (Sat 1, Sun 2)", counts)
			}
			if num(obj(segs[6])["load"]) != 64 || obj(segs[6])["level"] != "intense" {
				t.Fatalf("sunday segment = %v", segs[6])
			}
			if num(got["sessions"]) != 3 {
				t.Fatalf("week sessions = %v, want 3 (next Monday excluded)", got["sessions"])
			}
			next := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start="+d[6]["date"].(string)+"", "", http.StatusOK)
			if next["start"] != c.start {
				t.Fatalf("Sunday as start must give its own week, got %v", next["start"])
			}
		})
	}
}

func TestWeekSlots(t *testing.T) {
	e := newEnv(t)
	day := func(d string) string { return "/api/v1/horses/" + luna + "/week/" + d }
	dayRow := func(out map[string]any, i int) map[string]any { return weekDays(out)[i] }

	// A rider takes a free day ("Ich").
	got := e.call(seed.UserMia, http.MethodPut, day("2026-03-26"), `{"activity":"hall"}`, http.StatusOK)
	thu := dayRow(got, 3)
	if thu["status"] != "planned" || thu["is_me"] != true || thu["activity"] != "hall" || thu["can_take"] != true {
		t.Fatalf("thursday = %v", thu)
	}
	// The owner sees who; the owner can also take a day and assign others.
	e.call(seed.UserJan, http.MethodPut, day("2026-03-27"), `{"status":"planned","user_id":"`+seed.UserMia+`"}`, http.StatusOK)
	got = e.call(seed.UserJan, http.MethodPut, day("2026-03-28"), `{}`, http.StatusOK)
	if sat := dayRow(got, 5); obj(sat["user"])["name"] != "Jan" || sat["status"] != "planned" {
		t.Fatalf("saturday = %v", sat)
	}

	// A rider cannot take a day somebody else has, change rules, plan rest or assign others.
	e.call(seed.UserJan, http.MethodPut, day("2026-03-29"), `{"user_id":"`+seed.UserJan+`"}`, http.StatusOK)
	if code := e.errCode(seed.UserMia, http.MethodPut, day("2026-03-29"), `{}`, http.StatusConflict); code != "conflict" {
		t.Fatalf("code = %q", code)
	}
	e.errCode(seed.UserMia, http.MethodPut, day("2026-03-30"), `{"status":"rest"}`, http.StatusForbidden)
	e.errCode(seed.UserMia, http.MethodPut, day("2026-03-30"), `{"user_id":"`+seed.UserJan+`"}`, http.StatusForbidden)
	e.errCode(seed.UserMia, http.MethodPut, day("2026-03-24"), `{}`, http.StatusBadRequest) // over
	e.errCode(seed.UserMia, http.MethodPut, day("2026-03-30"), `{"status":"sleeping"}`, http.StatusBadRequest)
	e.errCode(seed.UserMia, http.MethodPut, day("gestern"), `{}`, http.StatusBadRequest)
	e.errCode(seed.UserMia, http.MethodPut, day("2026-03-30"), `{"activity":"rest"}`, http.StatusBadRequest)
	// Members without a role, and riders without take_week_slots, may not.
	e.errCode(seed.UserSarah, http.MethodPut, day("2026-03-30"), `{}`, http.StatusForbidden)
	e.exec(`UPDATE horse_riders SET rules = '["log_sessions"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)
	e.errCode(seed.UserMia, http.MethodPut, day("2026-03-30"), `{}`, http.StatusForbidden)
	e.exec(`UPDATE horse_riders SET rules = '["take_week_slots"]' WHERE horse_id = $1 AND user_id = $2`, luna, seed.UserMia)

	// The owner may plan a rest day, assign only stable members, and release days; a rider
	// can release her own claim but not the owner's.
	e.call(seed.UserJan, http.MethodPut, day("2026-03-30"), `{"status":"rest"}`, http.StatusOK)
	e.errCode(seed.UserJan, http.MethodPut, day("2026-03-31"), `{"user_id":"00000000-0000-4000-8000-00000000ffff"}`, http.StatusBadRequest)
	got = e.call(seed.UserMia, http.MethodPut, day("2026-03-26"), `{"status":"open"}`, http.StatusOK)
	if dayRow(got, 3)["status"] != "open" {
		t.Fatalf("released thursday = %v", dayRow(got, 3))
	}
	e.errCode(seed.UserMia, http.MethodPut, day("2026-03-29"), `{"status":"open"}`, http.StatusConflict)
	e.errCode(seed.UserMia, http.MethodPut, day("2026-03-26"), `{"status":"open"}`, http.StatusConflict) // nothing to release

	// A done day is fixed for riders; the owner can still edit.
	e.call(seed.UserJan, http.MethodPost, "/api/v1/horses/"+luna+"/sessions", `{"activity":"hall","minutes":30}`, http.StatusCreated)
	e.errCode(seed.UserMia, http.MethodPut, day("2026-03-25"), `{}`, http.StatusConflict)
	got = e.call(seed.UserMia, http.MethodGet, "/api/v1/horses/"+luna+"/week", "", http.StatusOK)
	if wedRow := dayRow(got, 2); wedRow["status"] != "done" || wedRow["can_take"] != false {
		t.Fatalf("wednesday = %v", wedRow)
	}
}
