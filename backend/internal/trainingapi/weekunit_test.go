package trainingapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// unitLibrary: a warm-up, a main exercise and a cool-down of the dressage library.
func unitLibrary(e *env) {
	e.clearLibrary()
	e.addExercise(exSpec{id: exID(1), library: "dressage", level: "beginner", title: "Aufwärmen im Schritt", phase: "warmup", tags: []string{"losgelassenheit"}, minutes: 10})
	e.addExercise(exSpec{id: exID(2), library: "dressage", level: "beginner", title: "Schlangenlinien", phase: "main", tags: []string{"losgelassenheit", "biegung"}, minutes: 12})
	e.addExercise(exSpec{id: exID(3), library: "dressage", level: "beginner", title: "Zügel aus der Hand", phase: "cooldown", tags: []string{"mobilitaet"}})
	e.addExercise(exSpec{id: exID(4), library: "dressage", level: "beginner", title: "Volte", phase: "main", tags: []string{"biegung"}})
}

func partOf(day map[string]any, part string) map[string]any { return obj(obj(day["unit"])[part]) }

func TestWeekSlotFocusKeyUnitAndEquipment(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	path := "/api/v1/horses/" + luna + "/week/2026-03-27" // the Friday of the test week
	thu := func(out map[string]any) map[string]any { return weekDays(out)[4] }

	body := `{"status":"planned","user_id":"","activity":"hall","focus_key":"losgelassenheit","equipment_ok":false,
		"unit":{"warmup":{"exercise_id":"` + exID(1) + `","minutes":15},"main":{"exercise_id":"` + exID(2) + `","minutes":27},"cooldown":{"minutes":8}}}`
	d := thu(e.call(seed.UserJan, http.MethodPut, path, body, http.StatusOK))
	if d["focus_key"] != "losgelassenheit" || d["focus_label"] != "Losgelassenheit" || d["activity"] != "hall" || d["equipment_ok"] != false {
		t.Fatalf("planned day = %v", d)
	}
	if w := partOf(d, "warmup"); obj(w["exercise"])["id"] != exID(1) || obj(w["exercise"])["title"] != "Aufwärmen im Schritt" || num(w["minutes"]) != 15 || num(w["exercise_minutes"]) != 10 {
		t.Fatalf("warm-up = %v", w)
	}
	// The main exercise of the unit is the slot's exercise (older clients read it from there); a
	// part without an exercise is free work with minutes only, no exercise_minutes without typical minutes.
	if m := partOf(d, "main"); obj(m["exercise"])["id"] != exID(2) || num(m["minutes"]) != 27 || num(m["exercise_minutes"]) != 12 {
		t.Fatalf("main = %v", m)
	}
	if obj(d["exercise"])["id"] != exID(2) {
		t.Fatalf("exercise = %v, want the main exercise", d["exercise"])
	}
	if c := partOf(d, "cooldown"); c["exercise"] != nil || num(c["minutes"]) != 8 || c["exercise_minutes"] != nil {
		t.Fatalf("cool-down = %v", c)
	}
	if d["equipment_note"] != "Heute ohne Aufbau geplant." {
		t.Fatalf("equipment_note = %v", d["equipment_note"])
	}

	// Mia takes the day with "Ich": everything stays.
	d = thu(e.call(seed.UserMia, http.MethodPut, path, `{}`, http.StatusOK))
	if obj(d["user"])["name"] != "Mia" || d["focus_key"] != "losgelassenheit" || d["equipment_ok"] != false || obj(partOf(d, "main")["exercise"])["id"] != exID(2) {
		t.Fatalf("claimed day = %v", d)
	}
	// Omitted and null keep focus key and unit; equipment_ok null is "the location's default".
	d = thu(e.call(seed.UserJan, http.MethodPut, path, `{"status":"planned","user_id":"","focus_key":null,"unit":null}`, http.StatusOK))
	if d["focus_key"] != "losgelassenheit" || d["unit"] == nil || d["equipment_ok"] != false {
		t.Fatalf("null keeps: %v", d)
	}
	d = thu(e.call(seed.UserJan, http.MethodPut, path, `{"status":"planned","user_id":"","equipment_ok":null}`, http.StatusOK))
	if d["equipment_ok"] != nil || d["equipment_note"] != nil || d["focus_key"] != "losgelassenheit" {
		t.Fatalf("equipment_ok null: %v", d)
	}
	d = thu(e.call(seed.UserJan, http.MethodPut, path, `{"status":"planned","user_id":"","equipment_ok":true}`, http.StatusOK))
	if d["equipment_ok"] != true || d["equipment_note"] != nil {
		t.Fatalf("equipment_ok true: %v", d)
	}

	// "" clears the focus key on its own.
	d = thu(e.call(seed.UserJan, http.MethodPut, path, `{"status":"planned","user_id":"","focus_key":""}`, http.StatusOK))
	if d["focus_key"] != nil || d["focus_label"] != nil || d["unit"] == nil {
		t.Fatalf("focus_key cleared: %v", d)
	}

	// An exercise sent alone becomes the main exercise of the stored unit; the minutes stay.
	d = thu(e.call(seed.UserJan, http.MethodPut, path, `{"status":"planned","user_id":"","exercise_id":"`+exID(4)+`"}`, http.StatusOK))
	if m := partOf(d, "main"); obj(m["exercise"])["id"] != exID(4) || num(m["minutes"]) != 27 || obj(d["exercise"])["id"] != exID(4) {
		t.Fatalf("exercise changed: main %v, exercise %v", m, d["exercise"])
	}
	if obj(partOf(d, "warmup")["exercise"])["id"] != exID(1) {
		t.Fatalf("the warm-up changed: %v", d["unit"])
	}
	d = thu(e.call(seed.UserJan, http.MethodPut, path, `{"status":"planned","user_id":"","exercise_id":""}`, http.StatusOK))
	if m := partOf(d, "main"); m["exercise"] != nil || num(m["minutes"]) != 27 || d["exercise"] != nil {
		t.Fatalf("exercise cleared: main %v, exercise %v", m, d["exercise"])
	}

	// A unit in the shape of the answer is accepted (exercise {id, title}); parts that are not sent
	// are gone; a part without minutes and exercise is left out.
	d = thu(e.call(seed.UserJan, http.MethodPut, path,
		`{"status":"planned","user_id":"","unit":{"warmup":{"exercise":{"id":"`+exID(1)+`","title":"x"},"minutes":12,"exercise_minutes":10,"why":"y","instead_of":null},"cooldown":{"minutes":0}}}`, http.StatusOK))
	u := obj(d["unit"])
	if len(u) != 1 || obj(partOf(d, "warmup")["exercise"])["id"] != exID(1) || num(partOf(d, "warmup")["minutes"]) != 12 || d["exercise"] != nil {
		t.Fatalf("unit = %v, exercise %v", d["unit"], d["exercise"])
	}
	// An empty object clears the plan and the main exercise.
	e.call(seed.UserJan, http.MethodPut, path, `{"status":"planned","user_id":"","unit":{"main":{"exercise_id":"`+exID(2)+`","minutes":20}}}`, http.StatusOK)
	d = thu(e.call(seed.UserJan, http.MethodPut, path, `{"status":"planned","user_id":"","unit":{}}`, http.StatusOK))
	if d["unit"] != nil || d["exercise"] != nil || d["activity"] != "hall" {
		t.Fatalf("unit cleared: %v", d)
	}
	// A rest day has no activity, focus key, unit or equipment.
	e.call(seed.UserJan, http.MethodPut, path, body, http.StatusOK)
	d = thu(e.call(seed.UserJan, http.MethodPut, path, `{"status":"rest"}`, http.StatusOK))
	if d["status"] != "rest" || d["focus_key"] != nil || d["unit"] != nil || d["equipment_ok"] != nil || d["equipment_note"] != nil {
		t.Fatalf("rest day = %v", d)
	}
	var key, unit *string
	var ok *bool
	if err := e.pool.QueryRow(context.Background(), `SELECT focus_key, unit_plan::text, equipment_ok FROM week_slots WHERE horse_id = $1 AND day = '2026-03-27'`, luna).Scan(&key, &unit, &ok); err != nil || key != nil || unit != nil || ok != nil {
		t.Fatalf("stored rest day: %v %v %v (%v)", key, unit, ok, err)
	}
}

func TestWeekSlotUnitValidation(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	path := "/api/v1/horses/" + luna + "/week/2026-03-27"
	e.exec(`INSERT INTO stables (id, name) VALUES ('00000000-0000-4000-8000-0000000001f9', 'Fremder Stall')`)
	e.exec(`INSERT INTO exercises (id, stable_id, discipline, level, title) VALUES ($1, '00000000-0000-4000-8000-0000000001f9', 'dressage', 'beginner', 'Fremde Übung')`, exID(90))
	for name, body := range map[string]string{
		"unknown focus_key":    `{"user_id":"","focus_key":"tempo"}`,
		"focus_key case":       `{"user_id":"","focus_key":"Takt"}`,
		"focus_key number":     `{"user_id":"","focus_key":3}`,
		"unknown exercise":     `{"user_id":"","unit":{"main":{"exercise_id":"00000000-0000-4000-8000-00000000ffff","minutes":10}}}`,
		"exercise of a stable": `{"user_id":"","unit":{"main":{"exercise_id":"` + exID(90) + `","minutes":10}}}`,
		"malformed exercise":   `{"user_id":"","unit":{"main":{"exercise_id":"x","minutes":10}}}`,
		"negative minutes":     `{"user_id":"","unit":{"main":{"minutes":-1}}}`,
		"too many minutes":     `{"user_id":"","unit":{"main":{"minutes":241}}}`,
		"unit too long":        `{"user_id":"","unit":{"warmup":{"minutes":240},"main":{"minutes":240},"cooldown":{"minutes":240}}}`,
		"exercise mismatch":    `{"user_id":"","exercise_id":"` + exID(4) + `","unit":{"main":{"exercise_id":"` + exID(2) + `","minutes":10}}}`,
		"exercise, no main":    `{"user_id":"","exercise_id":"` + exID(4) + `","unit":{"warmup":{"minutes":10}}}`,
		"unit not an object":   `{"user_id":"","unit":[]}`,
		"unknown part":         `{"user_id":"","unit":{"stretch":{"minutes":5}}}`,
		"unknown field":        `{"user_id":"","unit":{"main":{"minutes":5,"rest":true}}}`,
		"minutes not a number": `{"user_id":"","unit":{"main":{"minutes":"viel"}}}`,
		"equipment_ok string":  `{"user_id":"","equipment_ok":"yes"}`,
	} {
		if code := e.errCode(seed.UserJan, http.MethodPut, path, body, http.StatusBadRequest); code != "validation_failed" && code != "invalid_json" {
			t.Errorf("%s: code = %q", name, code)
		}
	}
	if e.count(`SELECT count(*) FROM week_slots WHERE focus_key IS NOT NULL OR unit_plan IS NOT NULL`) != 0 {
		t.Fatal("a refused request stored a plan")
	}
	// exercise_id and the main part agree: accepted.
	e.call(seed.UserJan, http.MethodPut, path, `{"user_id":"","exercise_id":"`+exID(2)+`","unit":{"main":{"exercise_id":"`+exID(2)+`","minutes":10}}}`, http.StatusOK)
	// Riders may send them too, for a day they take.
	e.call(seed.UserMia, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-28", `{"activity":"hall","focus_key":"takt","unit":{"main":{"minutes":30}}}`, http.StatusOK)
}

// A show day has no unit: focus key and unit are refused and not shown, like activity, focus and exercise.
func TestWeekShowDayHasNoFocusKeyOrUnit(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/training-profile", validProfile, http.StatusOK) // show on Sat 28th
	sat := "/api/v1/horses/" + luna + "/week/2026-03-28"
	for _, body := range []string{`{"focus_key":"takt"}`, `{"unit":{"main":{"minutes":20}}}`, `{"activity":"hall","equipment_ok":true}`} {
		e.errCode(seed.UserJan, http.MethodPut, sat, body, http.StatusBadRequest)
	}
	// An empty unit and claiming the day are fine.
	e.call(seed.UserJan, http.MethodPut, sat, `{"unit":{}}`, http.StatusOK)
	// A plan stored before the show was entered is hidden and comes back without the show.
	e.exec(`UPDATE week_slots SET focus_key = 'takt', unit_plan = '{"main": {"exercise_id": null, "minutes": 30}}', equipment_ok = true WHERE horse_id = $1 AND day = '2026-03-28'`, luna)
	d := weekDays(e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-03-23", "", http.StatusOK))[5]
	if d["focus_key"] != nil || d["focus_label"] != nil || d["unit"] != nil || d["equipment_ok"] != nil || d["equipment_note"] != nil {
		t.Fatalf("show day = %v", d)
	}
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/training-profile", strings.Replace(validProfile,
		`"shows": [{"date": "2026-03-28", "name": "Turnier", "classes": "Dressur L", "helper": "Anna"}]`, `"shows": []`, 1), http.StatusOK)
	d = weekDays(e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-03-23", "", http.StatusOK))[5]
	if d["focus_key"] != "takt" || num(partOf(d, "main")["minutes"]) != 30 {
		t.Fatalf("show removed = %v", d)
	}
}

// What a rider may not see of a day (a hidden session) is not leaked through the plan fields either.
func TestWeekPlanFieldsStayHiddenFromRiders(t *testing.T) {
	e := newEnv(t)
	unitLibrary(e)
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status, focus_key, unit_plan, equipment_ok)
	        VALUES ($1, $2, '2026-03-24', $3, 'hall', 'planned', 'takt', '{"main": {"exercise_id": null, "minutes": 30}}', true)`, seed.StableB, luna, seed.UserJan)
	e.insertSession(luna, seed.UserJan, "hall", time.Date(2026, 3, 24, 17, 0, 0, 0, berlin), 45, 36, false)

	owner := weekDays(e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-03-23", "", http.StatusOK))[1]
	rider := weekDays(e.call(seed.UserMia, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-03-23", "", http.StatusOK))[1]
	if owner["focus_key"] != "takt" || owner["unit"] == nil {
		t.Fatalf("owner sees %v", owner)
	}
	if rider["status"] != "done" || rider["focus_key"] != nil || rider["focus_label"] != nil || rider["unit"] != nil || rider["equipment_ok"] != nil || rider["equipment_note"] != nil {
		t.Fatalf("rider sees %v", rider)
	}
}

func TestWeekEquipmentNote(t *testing.T) {
	e := newEnv(t)
	e.exec(`UPDATE stables SET training_equipment = '{"hall": {"equipment": ["poles"], "shared": true}, "arena": {"equipment": ["poles"], "shared": false}}'`)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-26", `{"status":"planned","user_id":"","activity":"hall"}`, http.StatusOK)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-27", `{"status":"planned","user_id":"","activity":"arena"}`, http.StatusOK)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-29", `{"status":"planned","user_id":"","activity":"hack"}`, http.StatusOK)
	days := func() []map[string]any {
		return weekDays(e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-03-23", "", http.StatusOK))
	}
	d := days()
	// A shared hall is planned without set-up, the arena is not shared, a hack has no location.
	if d[3]["equipment_note"] != "Halle geteilt: ohne Aufbau geplant." || d[4]["equipment_note"] != nil || d[6]["equipment_note"] != nil {
		t.Fatalf("notes = %v / %v / %v", d[3]["equipment_note"], d[4]["equipment_note"], d[6]["equipment_note"])
	}
	// The day says set-up is possible (agreed) or not possible.
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-26", `{"status":"planned","user_id":"","equipment_ok":true}`, http.StatusOK)
	e.call(seed.UserJan, http.MethodPut, "/api/v1/horses/"+luna+"/week/2026-03-27", `{"status":"planned","user_id":"","equipment_ok":false}`, http.StatusOK)
	d = days()
	if d[3]["equipment_note"] != nil || d[3]["equipment_ok"] != true || d[4]["equipment_note"] != "Heute ohne Aufbau geplant." || d[4]["equipment_ok"] != false {
		t.Fatalf("with overrides: %v / %v", d[3], d[4])
	}
	// A done day has no note.
	e.call(seed.UserJan, http.MethodPost, "/api/v1/horses/"+luna+"/sessions", `{"activity":"arena","minutes":30,"started_at":"2026-03-24T17:00:00+01:00"}`, http.StatusCreated)
	e.exec(`UPDATE week_slots SET equipment_ok = false WHERE horse_id = $1 AND day = '2026-03-24'`, luna)
	if d := days(); d[1]["status"] != "done" || d[1]["equipment_note"] != nil {
		t.Fatalf("done day = %v", d[1])
	}
}

func TestWeekBlock(t *testing.T) {
	e := newEnv(t)
	week := func(start string) map[string]any {
		return obj(e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start="+start, "", http.StatusOK)["block"])
	}
	// No cycle_start (profiles from before the focuses), no block.
	if got := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week", "", http.StatusOK); got["block"] != nil {
		t.Fatalf("block without cycle_start = %v", got["block"])
	}
	// The seed's profile has a minimum of 4 sessions a week. Four weeks from 2026-02-23 with four
	// sessions each: week 1, 2, 3 and the Entlastungswoche, then the next block.
	e.exec(`UPDATE training_profiles SET cycle_start = '2026-02-23' WHERE horse_id = $1`, luna)
	for w := 0; w < 4; w++ {
		for s := 0; s < 4; s++ {
			day := time.Date(2026, 2, 23+7*w+s, 17, 0, 0, 0, berlin)
			e.insertSession(luna, seed.UserJan, "hall", day, 40, 32, true)
		}
	}
	for start, want := range map[string][2]any{
		"2026-02-23": {1.0, false}, "2026-03-02": {2.0, false}, "2026-03-09": {3.0, false}, "2026-03-16": {4.0, true}, "2026-03-23": {1.0, false},
	} {
		if b := week(start); b["week"] != want[0] || b["deload"] != want[1] {
			t.Errorf("week %s: block %v, want %v", start, b, want)
		}
	}
	// The weeks before the start have no block.
	if got := e.call(seed.UserJan, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-02-16", "", http.StatusOK); got["block"] != nil {
		t.Errorf("week before the cycle start: %v", got["block"])
	}
	// A week with fewer sessions than the minimum (week 2, 03-02) starts the block again: the week
	// after it is week 1, the weeks after that count on from there.
	e.exec(`DELETE FROM sessions WHERE started_at >= '2026-03-02' AND started_at < '2026-03-05'`)
	for start, want := range map[string]float64{"2026-03-02": 2, "2026-03-09": 1, "2026-03-16": 2, "2026-03-23": 3} {
		if b := week(start); b["week"] != want {
			t.Errorf("after a weak week, %s: block %v, want week %v", start, b, want)
		}
	}
	// Riders see it too.
	got := e.call(seed.UserMia, http.MethodGet, "/api/v1/horses/"+luna+"/week?start=2026-03-23", "", http.StatusOK)
	if b := obj(got["block"]); b["week"] != 3.0 {
		t.Errorf("rider block = %v", got["block"])
	}
}
