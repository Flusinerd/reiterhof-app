package blankets_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/blankets"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestNightDay(t *testing.T) {
	cases := []struct {
		name string
		at   time.Time
		want string
	}{
		{"evening", berlinAt(2026, 9, 30, 19, 0), "2026-09-30"},
		{"just before midnight", berlinAt(2026, 9, 30, 23, 59), "2026-09-30"},
		{"after midnight belongs to the running night", berlinAt(2026, 10, 1, 0, 30), "2026-09-30"},
		{"just before the rollover", berlinAt(2026, 10, 1, 3, 59), "2026-09-30"},
		{"04:00 starts the coming night", berlinAt(2026, 10, 1, 4, 0), "2026-10-01"},
		{"morning shows the coming night", berlinAt(2026, 10, 1, 7, 0), "2026-10-01"},
		{"month change", berlinAt(2026, 11, 1, 2, 0), "2026-10-31"},
		{"utc input is converted", time.Date(2026, 9, 30, 22, 30, 0, 0, time.UTC), "2026-09-30"}, // 00:30 Berlin on 1 Oct
		{"dst end night", berlinAt(2026, 10, 25, 3, 0), "2026-10-24"},
	}
	for _, c := range cases {
		if got := blankets.NightDay(c.at, berlin); got != c.want {
			t.Errorf("%s: NightDay(%s) = %s, want %s", c.name, c.at, got, c.want)
		}
	}
}

func TestStateDay(t *testing.T) {
	cases := []struct {
		name   string
		at     time.Time
		action string
		want   string
	}{
		{"evening cover", berlinAt(2026, 9, 30, 19, 0), blankets.ActionCovered, "2026-09-30"},
		{"night uncover", berlinAt(2026, 10, 1, 1, 0), blankets.ActionUncovered, "2026-09-30"},
		{"morning uncover ends the past night", berlinAt(2026, 10, 1, 7, 0), blankets.ActionUncovered, "2026-09-30"},
		{"uncover at 12:30 (staff brings the horses in)", berlinAt(2026, 10, 1, 12, 30), blankets.ActionUncovered, "2026-09-30"},
		{"uncover just before 15:00", berlinAt(2026, 10, 1, 14, 59), blankets.ActionUncovered, "2026-09-30"},
		{"uncover from 15:00 on is for the coming night", berlinAt(2026, 10, 1, 15, 0), blankets.ActionUncovered, "2026-10-01"},
		{"morning cover is for the coming night", berlinAt(2026, 10, 1, 7, 0), blankets.ActionCovered, "2026-10-01"},
		{"morning check is for the coming night", berlinAt(2026, 10, 1, 7, 0), blankets.ActionChecked, "2026-10-01"},
	}
	for _, c := range cases {
		if got := blankets.StateDay(c.at, berlin, c.action); got != c.want {
			t.Errorf("%s: StateDay(%s, %s) = %s, want %s", c.name, c.at, c.action, got, c.want)
		}
	}
}

func TestBlanketCRUDAndPermissions(t *testing.T) {
	e := newEnv(t)
	lunaPath := "/api/v1/horses/" + seed.HorseLuna + "/blankets"

	// Everybody reads.
	var list struct{ Blankets []blanketJSON }
	e.do(seed.UserAnna, "GET", lunaPath, nil).status(t, 200).into(t, &list)
	if len(list.Blankets) != 3 || list.Blankets[0].Name != "Regendecke" || list.Blankets[1].FillG != 100 {
		t.Fatalf("Luna blankets = %+v", list.Blankets)
	}
	if list.Blankets[0].Location == nil || *list.Blankets[0].Location != "Haken 3" {
		t.Errorf("seed location missing: %+v", list.Blankets[0])
	}

	photo := seed.StableB + "/" + strings.Repeat("ab", 16) + ".jpg"
	body := m{"name": "Fleecedecke", "fill_g": 0, "color": "navy", "location": "Haken 11", "photo_path": photo}

	// Only owner or admin write; riders and other members do not.
	e.do(seed.UserMia, "POST", lunaPath, body).errCode(t, 403, "forbidden")
	e.do(seed.UserAnna, "POST", lunaPath, body).errCode(t, 403, "forbidden")
	e.do("", "POST", lunaPath, body).status(t, 401)

	var created blanketJSON
	e.do(seed.UserJan, "POST", lunaPath, body).status(t, 201).into(t, &created)
	if created.Name != "Fleecedecke" || created.PhotoPath == nil || *created.PhotoPath != photo ||
		created.PhotoURL == nil || *created.PhotoURL != "/api/v1/files/"+photo {
		t.Fatalf("created = %+v", created)
	}
	// An admin may write for any horse (Jan is admin, Balu belongs to Jonas), the owner too.
	e.do(seed.UserJan, "POST", "/api/v1/horses/"+seed.HorseBalu+"/blankets", m{"name": "Regendecke"}).status(t, 201)
	e.do(seed.UserJonas, "POST", "/api/v1/horses/"+seed.HorseBalu+"/blankets", m{"name": "Fleece", "fill_g": 50}).status(t, 201)

	// Validation.
	for name, b := range map[string]m{
		"no name":         {"fill_g": 100},
		"blank name":      {"name": "   "},
		"negative fill":   {"name": "x", "fill_g": -1},
		"huge fill":       {"name": "x", "fill_g": 5000},
		"long location":   {"name": "x", "location": strings.Repeat("h", 81)},
		"foreign photo":   {"name": "x", "photo_path": "../etc/passwd"},
		"other stable":    {"name": "x", "photo_path": "00000000-0000-4000-8000-0000000009ff/" + strings.Repeat("ab", 16) + ".jpg"},
		"unknown field":   {"name": "x", "weight": 1},
		"photo not a url": {"name": "x", "photo_path": "photo.jpg"},
	} {
		if r := e.do(seed.UserJan, "POST", lunaPath, b); r.Code != 400 {
			t.Errorf("%s: status %d, want 400: %s", name, r.Code, r.Body.String())
		}
	}

	// Patch: absent fields stay, "" clears.
	var patched blanketJSON
	e.do(seed.UserJan, "PATCH", lunaPath+"/"+created.ID, m{"fill_g": 30, "location": "", "photo_path": ""}).status(t, 200).into(t, &patched)
	if patched.FillG != 30 || patched.Location != nil || patched.PhotoPath != nil || patched.Name != "Fleecedecke" || patched.Color == nil {
		t.Errorf("patched = %+v", patched)
	}
	e.do(seed.UserMia, "PATCH", lunaPath+"/"+created.ID, m{"fill_g": 1}).errCode(t, 403, "forbidden")
	// A blanket of another horse is not addressable through this horse.
	e.do(seed.UserJan, "PATCH", lunaPath+"/"+seed.BlanketBalu150, m{"fill_g": 1}).errCode(t, 404, "not_found")
	e.do(seed.UserJan, "PATCH", lunaPath+"/not-a-uuid", m{"fill_g": 1}).errCode(t, 404, "not_found")

	// A blanket used by a rule cannot be deleted; a free one can.
	e.do(seed.UserJan, "DELETE", lunaPath+"/"+seed.BlanketLuna100, nil).errCode(t, 409, "in_use")
	e.do(seed.UserMia, "DELETE", lunaPath+"/"+created.ID, nil).errCode(t, 403, "forbidden")
	e.do(seed.UserJan, "DELETE", lunaPath+"/"+created.ID, nil).status(t, 204)
	e.do(seed.UserJan, "DELETE", lunaPath+"/"+created.ID, nil).errCode(t, 404, "not_found")
}

func TestBlanketsAreStableScoped(t *testing.T) {
	e := newEnv(t)
	const stableA = "00000000-0000-4000-8000-0000000009a1"
	const userA = "00000000-0000-4000-8000-0000000009a2"
	e.exec(`INSERT INTO stables (id, name) VALUES ($1, 'Anderer Stall')`, stableA)
	e.exec(`INSERT INTO users (id, stable_id, name, email, is_admin) VALUES ($1, $2, 'Fremd', 'fremd@example.org', true)`, userA, stableA)

	for _, req := range []struct{ method, path string }{
		{"GET", "/api/v1/horses/" + seed.HorseLuna + "/blankets"},
		{"POST", "/api/v1/horses/" + seed.HorseLuna + "/blankets"},
		{"GET", "/api/v1/horses/" + seed.HorseLuna + "/blanket-rules"},
		{"PUT", "/api/v1/horses/" + seed.HorseLuna + "/blanket-rules"},
		{"GET", "/api/v1/horses/" + seed.HorseLuna + "/blanket-plan"},
		{"POST", "/api/v1/horses/" + seed.HorseLuna + "/blanket-state"},
		{"GET", "/api/v1/horses/" + seed.HorseLuna + "/blanket-states"},
	} {
		body := any(nil)
		if req.method != "GET" {
			body = m{"name": "x", "rules": []m{}, "action": "checked"}
		}
		e.do(userA, req.method, req.path, body).errCode(t, 404, "not_found")
	}
	// The other stable sees only its own (empty) day.
	var today todayJSON
	e.do(userA, "GET", "/api/v1/blankets/today", nil).status(t, 200).into(t, &today)
	if today.Progress.Total != 0 || len(today.Horses) != 0 {
		t.Errorf("foreign today = %+v", today)
	}
}

func TestRulesValidationAndReplace(t *testing.T) {
	e := newEnv(t)
	path := "/api/v1/horses/" + seed.HorseLuna + "/blanket-rules"

	var got struct {
		Rules []struct {
			ID        string   `json:"id"`
			Position  int      `json:"position"`
			TempMin   *float64 `json:"temp_min"`
			TempMax   *float64 `json:"temp_max"`
			Rain      *bool    `json:"rain"`
			BlanketID *string  `json:"blanket_id"`
			Note      string   `json:"note"`
		}
	}
	e.do(seed.UserMia, "GET", path, nil).status(t, 200).into(t, &got)
	if len(got.Rules) != 5 || got.Rules[0].Position != 1 || got.Rules[3].BlanketID == nil || *got.Rules[3].BlanketID != seed.BlanketLuna100 {
		t.Fatalf("seed rules = %+v", got.Rules)
	}

	good := m{"temp_min": 0, "temp_max": 5, "blanket_id": seed.BlanketLuna100, "note": "Wunsch"}
	e.do(seed.UserMia, "PUT", path, m{"rules": []m{good}}).errCode(t, 403, "forbidden")
	e.do(seed.UserAnna, "PUT", path, m{"rules": []m{good}}).errCode(t, 403, "forbidden")

	bad := map[string]any{
		"missing rules":     m{},
		"null rules":        m{"rules": nil},
		"min above max":     m{"rules": []m{{"temp_min": 5, "temp_max": 0}}},
		"min equals max":    m{"rules": []m{{"temp_min": 5, "temp_max": 5}}},
		"temp out of range": m{"rules": []m{{"temp_min": -100}}},
		"foreign blanket":   m{"rules": []m{{"blanket_id": seed.BlanketBalu150}}},
		"unknown blanket":   m{"rules": []m{{"blanket_id": "00000000-0000-4000-8000-0000000009ff"}}},
		"blanket garbage":   m{"rules": []m{{"blanket_id": "abc"}}},
		"long note":         m{"rules": []m{{"note": strings.Repeat("n", 201)}}},
		"unknown field":     m{"rules": []m{{"temp": 5}}},
	}
	tooMany := make([]m, 31)
	for i := range tooMany {
		tooMany[i] = m{"note": "x"}
	}
	bad["too many"] = m{"rules": tooMany}
	for name, body := range bad {
		if r := e.do(seed.UserJan, "PUT", path, body); r.Code != 400 {
			t.Errorf("%s: status %d, want 400: %s", name, r.Code, r.Body.String())
		}
	}
	// A rejected request changes nothing.
	e.do(seed.UserMia, "GET", path, nil).status(t, 200).into(t, &got)
	if len(got.Rules) != 5 {
		t.Fatalf("rules after rejected PUTs = %d", len(got.Rules))
	}

	// Full replace: the array order is the priority, positions are 1-based.
	rules := []m{
		{"temp_max": 0, "blanket_id": seed.BlanketLuna200, "note": "Sehr kalt"},
		{"temp_min": 0, "temp_max": 8, "rain": true, "blanket_id": seed.BlanketLunaRain},
		{"note": "Keine Decke"},
	}
	e.do(seed.UserJan, "PUT", path, m{"rules": rules}).status(t, 200).into(t, &got)
	if len(got.Rules) != 3 || got.Rules[0].Position != 1 || got.Rules[2].Position != 3 || got.Rules[0].TempMax == nil ||
		*got.Rules[0].TempMax != 0 || got.Rules[1].Rain == nil || !*got.Rules[1].Rain || got.Rules[2].BlanketID != nil || got.Rules[2].Note != "Keine Decke" {
		t.Fatalf("replaced = %+v", got.Rules)
	}
	// The owner of another horse (Jonas, Balu) may not change Luna's rules; an admin may.
	e.do(seed.UserJonas, "PUT", path, m{"rules": []m{}}).errCode(t, 403, "forbidden")
	// Now Luna 100 g is unused, so it can be deleted.
	e.do(seed.UserJan, "DELETE", "/api/v1/horses/"+seed.HorseLuna+"/blankets/"+seed.BlanketLuna100, nil).status(t, 204)
	// Empty list removes all rules.
	e.do(seed.UserJan, "PUT", path, m{"rules": []m{}}).status(t, 200).into(t, &got)
	if len(got.Rules) != 0 {
		t.Fatalf("rules after clearing = %+v", got.Rules)
	}
	var plan struct {
		Recommendation recJSON `json:"recommendation"`
	}
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 6, 0), 3, true)
	e.do(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseLuna+"/blanket-plan", nil).status(t, 200).into(t, &plan)
	if plan.Recommendation.Status != "no_rule" {
		t.Errorf("status without rules = %q", plan.Recommendation.Status)
	}
}

func TestPlanRecommendation(t *testing.T) {
	e := newEnv(t)
	planPath := func(id string) string { return "/api/v1/horses/" + id + "/blanket-plan" }
	type planJSON struct {
		Horse   struct{ Name string }
		Day     string
		Weather *struct {
			NightMinC float64 `json:"night_min_c"`
			WillRain  bool    `json:"will_rain"`
		}
		Recommendation recJSON
		Rules          []struct{ Position int }
		Blankets       []blanketJSON
		HelperNote     *string `json:"helper_note"`
		State          *stateJSON
		CanManage      bool `json:"can_manage"`
	}
	get := func(user, horse string) planJSON {
		var p planJSON
		e.do(user, "GET", planPath(horse), nil).status(t, 200).into(t, &p)
		return p
	}

	// No forecast yet: graceful, no error.
	p := get(seed.UserMia, seed.HorseLuna)
	if p.Weather != nil || p.Recommendation.Status != "no_weather" || p.Recommendation.Blanket != nil || len(p.Rules) != 5 || len(p.Blankets) != 3 {
		t.Fatalf("plan without weather = %+v", p)
	}
	if p.HelperNote == nil || !strings.Contains(*p.HelperNote, "Heu") {
		t.Errorf("helper note = %v", p.HelperNote)
	}
	if p.CanManage {
		t.Error("a rider must not manage")
	}
	if !get(seed.UserJan, seed.HorseLuna).CanManage {
		t.Error("owner must manage")
	}

	// Night 3 °C with rain: Luna gets "Decke 100 g" by rule 4, Balu no blanket by rule 2.
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 6, 0), 3, true)
	p = get(seed.UserMia, seed.HorseLuna)
	if p.Weather == nil || p.Weather.NightMinC != 3 || !p.Weather.WillRain || p.Day != "2026-09-30" {
		t.Fatalf("weather = %+v", p.Weather)
	}
	r := p.Recommendation
	if r.Status != "blanket" || r.RuleIndex == nil || *r.RuleIndex != 3 || r.Blanket == nil || r.Blanket.ID != seed.BlanketLuna100 ||
		r.Blanket.Location == nil || *r.Blanket.Location != "Haken 4" || r.Note != "Bei Regen zusätzlich Regendecke" {
		t.Errorf("Luna at 3 °C rain = %+v", r)
	}
	b := get(seed.UserMia, seed.HorseBalu).Recommendation
	if b.Status != "none" || b.RuleIndex == nil || *b.RuleIndex != 1 || b.Blanket != nil || b.Note != "Keine Decke" {
		t.Errorf("Balu at 3 °C = %+v", b)
	}

	// The newest snapshot of the day wins.
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 12, 0), 8, true)
	r = get(seed.UserMia, seed.HorseLuna).Recommendation
	if r.Status != "blanket" || r.Blanket.ID != seed.BlanketLunaRain || *r.RuleIndex != 1 {
		t.Errorf("Luna at 8 °C rain = %+v", r)
	}
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 13, 0), 8, false)
	if r = get(seed.UserMia, seed.HorseLuna).Recommendation; r.Status != "none" || *r.RuleIndex != 2 {
		t.Errorf("Luna at 8 °C dry = %+v", r)
	}
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 14, 0), -1, false)
	if r = get(seed.UserMia, seed.HorseLuna).Recommendation; r.Blanket == nil || r.Blanket.ID != seed.BlanketLuna200 {
		t.Errorf("Luna at -1 °C = %+v", r)
	}
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 15, 0), -6, false)
	if r = get(seed.UserMia, seed.HorseBalu).Recommendation; r.Blanket == nil || r.Blanket.ID != seed.BlanketBalu150 {
		t.Errorf("Balu at -6 °C = %+v", r)
	}
	// Snapshots of other days are ignored.
	e.snapshot("2026-10-01", berlinAt(2026, 9, 30, 16, 0), 20, false)
	if r = get(seed.UserMia, seed.HorseBalu).Recommendation; r.Blanket == nil {
		t.Errorf("tomorrow's snapshot leaked into tonight: %+v", r)
	}
}

func TestDayBoundary(t *testing.T) {
	e := newEnv(t)
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 6, 0), 3, true)
	e.snapshot("2026-10-01", berlinAt(2026, 9, 30, 6, 0), 8, false)
	luna := "/api/v1/horses/" + seed.HorseLuna + "/blanket-state"
	var res struct{ State stateJSON }

	// 00:30: still the night of 30 September, the state counts for that day.
	e.at(berlinAt(2026, 10, 1, 0, 30))
	e.do(seed.UserMia, "POST", luna, m{"action": "covered"}).status(t, 200).into(t, &res)
	if res.State.Day != "2026-09-30" {
		t.Fatalf("state day = %s, want 2026-09-30", res.State.Day)
	}
	// 07:00: the overview shows the coming night with its forecast, everything open.
	e.at(berlinAt(2026, 10, 1, 7, 0))
	td := e.today(seed.UserMia)
	if td.Day != "2026-10-01" || td.Progress.Done != 0 || td.Weather == nil || td.Weather.NightMinC != 8 {
		t.Fatalf("today at 07:00 = %+v", td)
	}
	// Taking the blanket off in the morning ends the past night; Luna stays open for tonight.
	e.do(seed.UserMia, "POST", luna, m{"action": "uncovered"}).status(t, 200).into(t, &res)
	if res.State.Day != "2026-09-30" {
		t.Fatalf("morning uncover day = %s, want 2026-09-30", res.State.Day)
	}
	if td = e.today(seed.UserMia); td.Progress.Done != 0 {
		t.Fatalf("morning uncover marked tonight done: %+v", td)
	}
	// Covering in the morning already counts for the coming night.
	e.do(seed.UserMia, "POST", luna, m{"action": "covered"}).status(t, 200).into(t, &res)
	if res.State.Day != "2026-10-01" {
		t.Fatalf("morning cover day = %s, want 2026-10-01", res.State.Day)
	}
	// The history keeps both nights.
	var hist struct {
		Today  string
		States []stateJSON
	}
	e.do(seed.UserMia, "GET", "/api/v1/horses/"+seed.HorseLuna+"/blanket-states", nil).status(t, 200).into(t, &hist)
	if hist.Today != "2026-10-01" || len(hist.States) != 3 || hist.States[0].Day != "2026-10-01" || hist.States[2].Day != "2026-09-30" {
		t.Fatalf("history = %+v", hist)
	}
}

func TestStateFlowProgressAndHistory(t *testing.T) {
	e := newEnv(t)
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 6, 0), 3, true)
	luna := "/api/v1/horses/" + seed.HorseLuna + "/blanket-state"

	td := e.today(seed.UserMia)
	if td.Progress.Done != 0 || td.Progress.Total != 7 || td.ReminderTime != "20:30" || td.Weather == nil || td.Weather.NightMinC != 3 {
		t.Fatalf("initial today = %+v", td)
	}
	if td.Horses[0].Horse.Name != "Balu" { // open horses are sorted by name
		t.Errorf("first horse = %s", td.Horses[0].Horse.Name)
	}

	// Any member may set the state (Mia rides Luna, Lea does not).
	var res struct {
		State  stateJSON
		Closed []string `json:"closed_requests"`
	}
	e.do(seed.UserLea, "POST", luna, m{"action": "covered"}).status(t, 200).into(t, &res)
	// covered_with defaults to the recommended blanket (Luna 100 g at 3 °C with rain).
	if res.State.CoveredWith == nil || *res.State.CoveredWith != seed.BlanketLuna100 || res.State.ChangedBy == nil || res.State.ChangedBy.Name != "Lea" {
		t.Fatalf("state = %+v", res.State)
	}
	firstID := res.State.ID
	// The same tap again changes nothing.
	e.do(seed.UserMia, "POST", luna, m{"action": "covered"}).status(t, 200).into(t, &res)
	if res.State.ID != firstID {
		t.Errorf("repeated state created a new row")
	}
	// Another blanket is a new state.
	e.do(seed.UserMia, "POST", luna, m{"action": "covered", "covered_with": seed.BlanketLuna200}).status(t, 200).into(t, &res)
	if res.State.ID == firstID || *res.State.CoveredWith != seed.BlanketLuna200 {
		t.Errorf("second state = %+v", res.State)
	}
	e.do(seed.UserTom, "POST", "/api/v1/horses/"+seed.HorseBalu+"/blanket-state", m{"action": "checked"}).status(t, 200)

	td = e.today(seed.UserMia)
	if td.Progress.Done != 2 || td.Progress.Total != 7 {
		t.Fatalf("progress = %+v", td.Progress)
	}
	// Open horses first, the done ones last (by name).
	names := []string{}
	for _, h := range td.Horses {
		names = append(names, h.Horse.Name)
	}
	if got := strings.Join(names, ","); got != "Cookie,Fanta,Merlin,Nala,Pepe,Balu,Luna" {
		t.Errorf("order = %s", got)
	}
	last := td.Horses[6]
	if !last.Done || last.State == nil || last.State.Action != "covered" || *last.State.CoveredWith != seed.BlanketLuna200 {
		t.Errorf("Luna entry = %+v", last)
	}

	// Uncovering in the morning is another state of the same night.
	e.do(seed.UserMia, "POST", luna, m{"action": "uncovered"}).status(t, 200)
	var hist struct{ States []stateJSON }
	e.do(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseLuna+"/blanket-states?days=14", nil).status(t, 200).into(t, &hist)
	if len(hist.States) != 3 || hist.States[0].Action != "uncovered" || hist.States[2].Action != "covered" {
		t.Fatalf("history = %+v", hist.States)
	}
	// Older days show up, the window is bounded by `days`.
	e.setState(seed.HorseLuna, "2026-09-25", "covered", berlinAt(2026, 9, 25, 20, 0))
	e.setState(seed.HorseLuna, "2026-09-01", "covered", berlinAt(2026, 9, 1, 20, 0))
	e.do(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseLuna+"/blanket-states", nil).status(t, 200).into(t, &hist)
	if len(hist.States) != 4 || hist.States[3].Day != "2026-09-25" {
		t.Errorf("default window = %+v", hist.States)
	}
	e.do(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseLuna+"/blanket-states?days=1", nil).status(t, 200).into(t, &hist)
	if len(hist.States) != 3 {
		t.Errorf("days=1 = %d states", len(hist.States))
	}
	for _, d := range []string{"0", "91", "x"} {
		e.do(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseLuna+"/blanket-states?days="+d, nil).errCode(t, 400, "validation_failed")
	}

	// Validation.
	for name, body := range map[string]m{
		"bad action":            {"action": "wash"},
		"no action":             {},
		"blanket with uncover":  {"action": "uncovered", "covered_with": seed.BlanketLuna100},
		"blanket with check":    {"action": "checked", "covered_with": seed.BlanketLuna100},
		"blanket of the others": {"action": "covered", "covered_with": seed.BlanketBalu150},
		"unknown blanket":       {"action": "covered", "covered_with": "00000000-0000-4000-8000-0000000009ff"},
	} {
		if r := e.do(seed.UserMia, "POST", luna, body); r.Code != 400 {
			t.Errorf("%s: status %d, want 400: %s", name, r.Code, r.Body.String())
		}
	}
	e.do(seed.UserMia, "POST", "/api/v1/horses/00000000-0000-4000-8000-0000000009ff/blanket-state", m{"action": "checked"}).errCode(t, 404, "not_found")

	// Covered while the plan recommends no blanket stores no blanket.
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseBalu+"/blanket-state", m{"action": "covered"}).status(t, 200).into(t, &res)
	if res.State.CoveredWith != nil {
		t.Errorf("Balu covered_with = %v", res.State.CoveredWith)
	}
}

func TestStateChangedEventIsPublished(t *testing.T) {
	e := newEnv(t)
	hub := realtime.NewHub(e.pool, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { hub.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case <-hub.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("hub did not start")
	}
	events, unsubscribe := hub.Subscribe(seed.StableB)
	defer unsubscribe()

	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state", m{"action": "checked"}).status(t, 200)
	select {
	case ev := <-events:
		if ev.Type != blankets.EventStateChanged || !strings.Contains(string(ev.Data), seed.HorseLuna) || !strings.Contains(string(ev.Data), "2026-09-30") {
			t.Errorf("event = %s %s", ev.Type, ev.Data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("blanket_state.changed not delivered")
	}

	// A repeated identical tap still notifies (harmless) but a rejected request does not.
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state", m{"action": "nope"}).status(t, 400)
	select {
	case ev := <-events:
		if ev.Type == blankets.EventStateChanged && !strings.Contains(string(ev.Data), seed.HorseLuna) {
			t.Errorf("unexpected event %s", ev.Data)
		}
	case <-time.After(300 * time.Millisecond):
	}
}

func TestRuleChangesPublishPlanEvent(t *testing.T) {
	e := newEnv(t)
	hub := realtime.NewHub(e.pool, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { hub.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	<-hub.Ready()
	events, unsubscribe := hub.Subscribe(seed.StableB)
	defer unsubscribe()
	e.do(seed.UserJan, "PUT", "/api/v1/horses/"+seed.HorseLuna+"/blanket-rules", m{"rules": []m{{"note": "x"}}}).status(t, 200)
	select {
	case ev := <-events:
		if ev.Type != blankets.EventPlanChanged {
			t.Errorf("event = %s", ev.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("blanket_plan.changed not delivered")
	}
}

func TestStateClosesBlanketRequests(t *testing.T) {
	e := newEnv(t)
	create := func(user string, body m) string {
		var out struct{ ID string }
		e.do(user, "POST", "/api/v1/requests", body).status(t, 201).into(t, &out)
		return out.ID
	}
	lunaToday := create(seed.UserJan, m{"type": "blanket", "horse_id": seed.HorseLuna, "date": "2026-09-30", "payload": m{"wish": "Bitte Regendecke"}})
	lunaRange := create(seed.UserJan, m{"type": "blanket", "horse_id": seed.HorseLuna, "date": "2026-09-30", "date_end": "2026-10-02", "payload": m{}})
	lunaLater := create(seed.UserJan, m{"type": "blanket", "horse_id": seed.HorseLuna, "date": "2026-10-03", "payload": m{}})
	baluToday := create(seed.UserJonas, m{"type": "blanket", "horse_id": seed.HorseBalu, "date": "2026-09-30", "payload": m{}})
	lunaOther := create(seed.UserJan, m{"type": "other", "horse_id": seed.HorseLuna, "date": "2026-09-30", "payload": m{}})
	lunaCancelled := create(seed.UserJan, m{"type": "blanket", "horse_id": seed.HorseLuna, "date": "2026-09-30", "payload": m{}})
	e.do(seed.UserJan, "POST", "/api/v1/requests/"+lunaCancelled+"/cancel", nil).status(t, 200)
	// An assigned one is closed as well.
	e.do(seed.UserTom, "POST", "/api/v1/requests/"+lunaToday+"/accept", nil).status(t, 200)

	status := func(id string) (string, string) {
		var st, fb string
		if err := e.pool.QueryRow(context.Background(), `SELECT status, COALESCE(payload->>'feedback', '') FROM requests WHERE id = $1`, id).Scan(&st, &fb); err != nil {
			t.Fatal(err)
		}
		return st, fb
	}
	if st, _ := status(lunaToday); st != "assigned" {
		t.Fatalf("precondition: %s", st)
	}

	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 6, 0), 3, true)
	var res struct {
		Closed []string `json:"closed_requests"`
	}
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state", m{"action": "covered"}).status(t, 200).into(t, &res)
	if len(res.Closed) != 2 {
		t.Errorf("closed = %v", res.Closed)
	}
	if st, fb := status(lunaToday); st != "done" || fb != "Eingedeckt mit Decke 100 g (Mia)" {
		t.Errorf("Luna today: %s %q", st, fb)
	}
	if st, _ := status(lunaRange); st != "done" {
		t.Errorf("range request covering today: %s", st)
	}
	for name, id := range map[string]string{"future": lunaLater, "other horse": baluToday, "other type": lunaOther} {
		if st, _ := status(id); st != "open" {
			t.Errorf("%s request changed to %s", name, st)
		}
	}
	if st, _ := status(lunaCancelled); st != "cancelled" {
		t.Errorf("cancelled request changed to %s", st)
	}
	// The payload keeps the original content.
	var wish string
	if err := e.pool.QueryRow(context.Background(), `SELECT payload->>'wish' FROM requests WHERE id = $1`, lunaToday).Scan(&wish); err != nil || wish != "Bitte Regendecke" {
		t.Errorf("payload wish = %q, %v", wish, err)
	}
	// A second tap closes nothing more, a checked Balu closes Balu's request.
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state", m{"action": "covered"}).status(t, 200).into(t, &res)
	if len(res.Closed) != 0 {
		t.Errorf("second tap closed %v", res.Closed)
	}
	e.do(seed.UserTom, "POST", "/api/v1/horses/"+seed.HorseBalu+"/blanket-state", m{"action": "checked"}).status(t, 200)
	if st, fb := status(baluToday); st != "done" || fb != "Keine Decke nötig (Tom)" {
		t.Errorf("Balu today: %s %q", st, fb)
	}
}
