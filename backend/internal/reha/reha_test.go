package reha_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const (
	fanta = seed.HorseFanta
	luna  = seed.HorseLuna
)

var berlin = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	return loc
}()

// wed is the fixed "now" of the tests: Wednesday 2026-03-25 10:00 Berlin (CET).
var wed = time.Date(2026, 3, 25, 10, 0, 0, 0, berlin)

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
	now  time.Time
}

// newEnv creates a seeded database with a fixed clock and without the seed reha plan of Fanta
// (its dates are relative to the real day of seeding).
func newEnv(t *testing.T) *env {
	t.Helper()
	pool := dbtest.NewSeeded(t)
	for _, q := range []string{`DELETE FROM reha_days`, `DELETE FROM reha_plans`, `DELETE FROM sessions`, `DELETE FROM week_slots`, `DELETE FROM weather_snapshots`} {
		if _, err := pool.Exec(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	e := &env{t: t, pool: pool, now: wed}
	e.h = httpapi.NewHandler(httpapi.Deps{Pool: pool, Now: func() time.Time { return e.now }})
	return e
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("exec %q: %v", sql, err)
	}
}

func (e *env) count(sql string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatalf("count %q: %v", sql, err)
	}
	return n
}

// call sends a request as user ("" = anonymous), checks the status and decodes the body.
func (e *env) call(user, method, path string, body any, want int) map[string]any {
	e.t.Helper()
	var rd *strings.Reader
	if s, ok := body.(string); ok {
		rd = strings.NewReader(s)
	} else if body != nil {
		b, _ := json.Marshal(body)
		rd = strings.NewReader(string(b))
	} else {
		rd = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, rd)
	if user != "" {
		req.Header.Set("Authorization", "Bearer "+authtest.TokenAt(e.t, e.pool, user, e.now))
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Code != want {
		e.t.Fatalf("%s %s as %.4s: status = %d, want %d, body %s", method, path, user[max(0, len(user)-4):], rec.Code, want, rec.Body.String())
	}
	var out map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			e.t.Fatalf("%s %s: decode: %v (%s)", method, path, err, rec.Body.String())
		}
	}
	return out
}

func errCode(m map[string]any) string {
	return str(obj(m["error"])["code"])
}

func obj(v any) map[string]any { m, _ := v.(map[string]any); return m }
func list(v any) []any         { l, _ := v.([]any); return l }
func str(v any) string         { s, _ := v.(string); return s }
func num(v any) float64        { f, _ := v.(float64); return f }

// specPhases is the example of the specification.
func specPhases() []map[string]any {
	return []map[string]any{
		{"name": "Boxenruhe", "days": 5, "activity": "rest", "min_minutes": 0, "max_minutes": 0, "conditions": ""},
		{"name": "Schritt führen", "days": 9, "activity": "groundwork", "min_minutes": 10, "max_minutes": 20, "conditions": "nur Boden fest"},
		{"name": "Schritt reiten", "days": 14, "activity": "hall", "min_minutes": 20, "max_minutes": 40, "conditions": ""},
		{"name": "Trab aufbauen", "days": 14, "activity": "hall", "min_minutes": 2, "max_minutes": 15, "conditions": ""},
	}
}

// planBody starts on 2026-03-17: on the fixed day (03-25) the horse is on day 4 of "Schritt führen"
// (10 -> 20 min over 9 days), which allows 14 minutes.
func planBody(over map[string]any) map[string]any {
	b := map[string]any{
		"diagnosis": "Sehnenzerrung vorne links", "vet": "Dr. Berger", "start_date": "2026-03-17",
		"phases": specPhases(), "checkup_date": "2026-04-10", "abort_criteria": "Lahmheit, Wärme oder Schwellung: sofort abbrechen",
	}
	for k, v := range over {
		b[k] = v
	}
	return b
}

func (e *env) createPlan(user string, over map[string]any) map[string]any {
	e.t.Helper()
	return e.call(user, "POST", "/api/v1/horses/"+fanta+"/reha-plans", planBody(over), http.StatusCreated)
}

func (e *env) profileStatus(horse string) string {
	e.t.Helper()
	var s string
	if err := e.pool.QueryRow(context.Background(), `SELECT status FROM training_profiles WHERE horse_id = $1`, horse).Scan(&s); err != nil {
		e.t.Fatal(err)
	}
	return s
}

func TestCreateAndView(t *testing.T) {
	e := newEnv(t)
	empty := e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusOK)
	if empty["has_active_plan"] != false || empty["plan"] != nil || empty["today"] != nil || empty["view"] != "full" {
		t.Fatalf("empty view = %v", empty)
	}

	v := e.createPlan(seed.UserAnna, nil)
	if v["has_active_plan"] != true || v["view"] != "full" || v["can_edit"] != true || v["can_mark_done"] != true || v["horse_name"] != "Fanta" {
		t.Fatalf("view = %v", v)
	}
	plan := obj(v["plan"])
	if plan["diagnosis"] != "Sehnenzerrung vorne links" || plan["vet"] != "Dr. Berger" || plan["start_date"] != "2026-03-17" ||
		plan["end_date"] != "2026-04-27" || plan["checkup_date"] != "2026-04-10" || num(plan["checkup_in_days"]) != 16 ||
		plan["state"] != "running" || num(plan["current_phase"]) != 2 || num(plan["day_index"]) != 9 || num(plan["total_days"]) != 42 ||
		!strings.Contains(str(plan["abort_criteria"]), "abbrechen") || plan["active"] != true {
		t.Fatalf("plan = %v", plan)
	}
	phases := list(plan["phases"])
	if len(phases) != 4 {
		t.Fatalf("phases = %v", phases)
	}
	want := [][3]string{
		{"2026-03-17", "2026-03-21", "past"}, {"2026-03-22", "2026-03-30", "current"},
		{"2026-03-31", "2026-04-13", "upcoming"}, {"2026-04-14", "2026-04-27", "upcoming"},
	}
	for i, p := range phases {
		p := obj(p)
		if p["start_date"] != want[i][0] || p["end_date"] != want[i][1] || p["status"] != want[i][2] {
			t.Errorf("phase %d = %v, want %v", i, p, want[i])
		}
	}
	if obj(phases[0])["rest"] != true || obj(phases[1])["activity_label"] != "Bodenarbeit" {
		t.Errorf("phase flags = %v %v", phases[0], phases[1])
	}

	today := obj(v["today"])
	if today["phase"] != "Schritt führen" || num(today["phase_index"]) != 2 || num(today["day_in_phase"]) != 4 || num(today["days_in_phase"]) != 9 ||
		today["activity"] != "groundwork" || num(today["minutes"]) != 14 || today["conditions"] != "nur Boden fest" ||
		today["text"] != "Reha: Schritt führen 14 min, nur Boden fest" || today["done"] != false {
		t.Fatalf("today = %v", today)
	}
	if e.profileStatus(fanta) != "reha" {
		t.Errorf("profile status = %s, want reha", e.profileStatus(fanta))
	}
	if len(list(v["history"])) != 0 {
		t.Errorf("history = %v", v["history"])
	}
}

func TestSinglePlanAndHistory(t *testing.T) {
	e := newEnv(t)
	first := obj(e.createPlan(seed.UserAnna, nil)["plan"])
	second := obj(e.createPlan(seed.UserJan, map[string]any{"diagnosis": "Hufabszess", "start_date": "2026-03-25"})["plan"]) // admin may
	if first["id"] == second["id"] || second["diagnosis"] != "Hufabszess" {
		t.Fatalf("plans = %v %v", first, second)
	}
	if n := e.count(`SELECT count(*) FROM reha_plans WHERE horse_id = $1 AND active`, fanta); n != 1 {
		t.Fatalf("active plans = %d, want 1", n)
	}
	v := e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusOK)
	hist := list(v["history"])
	if len(hist) != 1 || obj(hist[0])["id"] != first["id"] || obj(hist[0])["active"] != false || obj(hist[0])["ended_on"] != "2026-03-25" {
		t.Fatalf("history = %v", hist)
	}
	if obj(v["plan"])["id"] != second["id"] || e.profileStatus(fanta) != "reha" {
		t.Errorf("active plan = %v, status %s", v["plan"], e.profileStatus(fanta))
	}
	// The database enforces it as well.
	_, err := e.pool.Exec(context.Background(), `INSERT INTO reha_plans (stable_id, horse_id, diagnosis, start_date, active) VALUES ($1, $2, 'x', '2026-03-01', true)`, seed.StableB, fanta)
	if err == nil {
		t.Error("a second active plan must violate the unique index")
	}
}

func TestValidation(t *testing.T) {
	e := newEnv(t)
	url := "/api/v1/horses/" + fanta + "/reha-plans"
	phase := func(over map[string]any) []map[string]any {
		p := map[string]any{"name": "Schritt", "days": 5, "activity": "walker", "min_minutes": 10, "max_minutes": 20, "conditions": ""}
		for k, v := range over {
			p[k] = v
		}
		return []map[string]any{p}
	}
	bad := map[string]map[string]any{
		"no diagnosis":     {"diagnosis": "  "},
		"long diagnosis":   {"diagnosis": strings.Repeat("x", 201)},
		"bad start":        {"start_date": "25.03.2026"},
		"far future":       {"start_date": "2028-01-01"},
		"bad checkup":      {"checkup_date": "morgen"},
		"checkup < start":  {"checkup_date": "2026-03-01"},
		"long abort":       {"abort_criteria": strings.Repeat("x", 1001)},
		"no phases":        {"phases": []any{}},
		"no name":          {"phases": phase(map[string]any{"name": ""})},
		"zero days":        {"phases": phase(map[string]any{"days": 0})},
		"negative days":    {"phases": phase(map[string]any{"days": -3})},
		"unknown activity": {"phases": phase(map[string]any{"activity": "dancing"})},
		"min > max":        {"phases": phase(map[string]any{"min_minutes": 30})},
		"zero minutes":     {"phases": phase(map[string]any{"min_minutes": 0})},
		"rest with mins":   {"phases": phase(map[string]any{"activity": "rest"})},
		"unknown field":    {"phases": phase(map[string]any{"weeks": 2})},
		"foreign obs":      {"observation_id": "00000000-0000-4000-8000-00000000ffff"},
		"malformed obs":    {"observation_id": "not-a-uuid"},
	}
	for name, over := range bad {
		out := e.call(seed.UserAnna, "POST", url, planBody(over), http.StatusBadRequest)
		if c := errCode(out); c != "validation_failed" && c != "invalid_json" {
			t.Errorf("%s: code %q", name, c)
		}
	}
	e.call(seed.UserAnna, "POST", url, "{not json", http.StatusBadRequest)
	if n := e.count(`SELECT count(*) FROM reha_plans`); n != 0 {
		t.Fatalf("rejected requests must not create plans, have %d", n)
	}

	// An observation of the horse is accepted, one of another horse is not.
	var mine, other string
	for horse, dst := range map[string]*string{fanta: &mine, luna: &other} {
		if err := e.pool.QueryRow(context.Background(), `INSERT INTO observations (stable_id, horse_id, reported_by, description) VALUES ($1, $2, $3, 'Schwellung') RETURNING id::text`,
			seed.StableB, horse, seed.UserLea).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	e.call(seed.UserAnna, "POST", url, planBody(map[string]any{"observation_id": other}), http.StatusBadRequest)
	v := e.call(seed.UserAnna, "POST", url, planBody(map[string]any{"observation_id": mine}), http.StatusCreated)
	if obj(v["plan"])["observation_id"] != mine {
		t.Errorf("observation_id = %v", obj(v["plan"])["observation_id"])
	}
	// Optional fields may be left out, unknown horses are 404.
	e.call(seed.UserAnna, "POST", url, map[string]any{"diagnosis": "x", "start_date": "2026-03-25", "phases": phase(nil)}, http.StatusCreated)
	e.call(seed.UserAnna, "POST", "/api/v1/horses/00000000-0000-4000-8000-00000000ffff/reha-plans", planBody(nil), http.StatusNotFound)
}

func TestPermissions(t *testing.T) {
	e := newEnv(t)
	url := "/api/v1/horses/" + fanta + "/reha-plans"
	e.call("", "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusUnauthorized)
	e.call("", "POST", url, planBody(nil), http.StatusUnauthorized)

	// Only the owner and admins create.
	for _, u := range []string{seed.UserLea, seed.UserSarah, seed.UserMia} {
		e.call(u, "POST", url, planBody(nil), http.StatusForbidden)
	}
	v := e.createPlan(seed.UserAnna, nil)
	planID := str(obj(v["plan"])["id"])
	base := "/api/v1/reha-plans/" + planID

	for _, u := range []string{seed.UserLea, seed.UserSarah} {
		e.call(u, "PATCH", base, map[string]any{"diagnosis": "x"}, http.StatusForbidden)
		e.call(u, "POST", base+"/end", nil, http.StatusForbidden)
	}
	// Jan is admin: may change and, later, end.
	e.call(seed.UserJan, "PATCH", base, map[string]any{"diagnosis": "Neu"}, http.StatusOK)

	// The rider Lea sees everything and may mark the day (legacy rule "ride").
	rv := e.call(seed.UserLea, "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusOK)
	if rv["view"] != "full" || rv["can_edit"] != false || rv["can_mark_done"] != true || obj(rv["plan"])["diagnosis"] != "Neu" {
		t.Fatalf("rider view = %v", rv)
	}
	// A rider without log_sessions can look but not mark.
	e.exec(`UPDATE horse_riders SET rules = '["report_observations"]' WHERE horse_id = $1 AND user_id = $2`, fanta, seed.UserLea)
	rv = e.call(seed.UserLea, "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusOK)
	if rv["can_mark_done"] != false {
		t.Errorf("rider without log_sessions must not mark: %v", rv)
	}
	e.call(seed.UserLea, "POST", base+"/days/2026-03-25/done", nil, http.StatusForbidden)
	e.exec(`UPDATE horse_riders SET rules = '["log_sessions"]' WHERE horse_id = $1 AND user_id = $2`, fanta, seed.UserLea)
	e.call(seed.UserLea, "POST", base+"/days/2026-03-25/done", nil, http.StatusOK)

	// Other members see today's rule only: no diagnosis, vet, dates or history, cannot mark.
	mv := e.call(seed.UserSarah, "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusOK)
	if mv["view"] != "today" || mv["plan"] != nil || mv["can_edit"] != false || mv["can_mark_done"] != false || mv["has_active_plan"] != true {
		t.Fatalf("member view = %v", mv)
	}
	today := obj(mv["today"])
	if today["text"] != "Reha: Schritt führen 14 min, nur Boden fest" || num(today["minutes"]) != 14 || today["done"] != true {
		t.Errorf("member today = %v", today)
	}
	raw, _ := json.Marshal(mv)
	for _, secret := range []string{"Sehnenzerrung", "Neu", "Berger", "2026-04-10", "abbrechen"} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("member view leaks %q: %s", secret, raw)
		}
	}
	e.call(seed.UserSarah, "POST", base+"/days/2026-03-25/done", nil, http.StatusForbidden)
	e.call(seed.UserSarah, "DELETE", base+"/days/2026-03-25/done", nil, http.StatusForbidden)

	// Another stable sees nothing.
	e.exec(`INSERT INTO stables (id, name) VALUES ('00000000-0000-4000-8000-0000000000a1', 'Fremd')`)
	e.exec(`INSERT INTO users (id, stable_id, name, email) VALUES ('00000000-0000-4000-8000-0000000000a2', '00000000-0000-4000-8000-0000000000a1', 'Fremd', 'fremd@example.org')`)
	const outsider = "00000000-0000-4000-8000-0000000000a2"
	e.call(outsider, "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusNotFound)
	e.call(outsider, "POST", url, planBody(nil), http.StatusNotFound)
	e.call(outsider, "PATCH", base, map[string]any{"diagnosis": "x"}, http.StatusNotFound)
	e.call(outsider, "POST", base+"/end", nil, http.StatusNotFound)
	e.call(outsider, "POST", base+"/days/2026-03-25/done", nil, http.StatusNotFound)
	e.call(seed.UserAnna, "PATCH", "/api/v1/reha-plans/nope", map[string]any{"diagnosis": "x"}, http.StatusNotFound)
	// A user without a stable is turned away by the stable check.
	e.exec(`INSERT INTO users (id, name, email) VALUES ('00000000-0000-4000-8000-0000000000a3', 'Ohne', 'ohne@example.org')`)
	e.call("00000000-0000-4000-8000-0000000000a3", "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusForbidden)
}

func TestPatchAndEnd(t *testing.T) {
	e := newEnv(t)
	v := e.createPlan(seed.UserAnna, nil)
	base := "/api/v1/reha-plans/" + str(obj(v["plan"])["id"])

	// Partial update: phases replaced, vet and checkup cleared, others untouched.
	out := e.call(seed.UserAnna, "PATCH", base, map[string]any{
		"vet": "", "checkup_date": "", "start_date": "2026-03-24",
		"phases": []map[string]any{{"name": "Nur Schritt", "days": 10, "activity": "walker", "min_minutes": 15, "max_minutes": 15, "conditions": ""}},
	}, http.StatusOK)
	plan := obj(out["plan"])
	if plan["vet"] != "" || plan["checkup_date"] != nil || plan["diagnosis"] != "Sehnenzerrung vorne links" || plan["end_date"] != "2026-04-02" ||
		len(list(plan["phases"])) != 1 || num(plan["day_index"]) != 2 {
		t.Fatalf("patched plan = %v", plan)
	}
	// An untouched old start date stays valid even when it lies far in the past.
	e.exec(`UPDATE reha_plans SET start_date = '2020-01-01'`)
	e.call(seed.UserAnna, "PATCH", base, map[string]any{"abort_criteria": "neu"}, http.StatusOK)
	e.exec(`UPDATE reha_plans SET start_date = '2026-03-24'`)

	for name, body := range map[string]map[string]any{
		"empty diagnosis": {"diagnosis": ""}, "bad phases": {"phases": []map[string]any{{"name": "x", "days": 0, "activity": "walker", "min_minutes": 1, "max_minutes": 2}}},
		"bad start": {"start_date": "x"}, "bad checkup": {"checkup_date": "x"}, "checkup before start": {"checkup_date": "2026-01-01"},
	} {
		if c := errCode(e.call(seed.UserAnna, "PATCH", base, body, http.StatusBadRequest)); c != "validation_failed" {
			t.Errorf("%s: code %q", name, c)
		}
	}
	e.call(seed.UserAnna, "PATCH", base, map[string]any{"unknown": 1}, http.StatusBadRequest)

	// Ending sets the status back to fit and is idempotent.
	ended := e.call(seed.UserAnna, "POST", base+"/end", nil, http.StatusOK)
	if ended["has_active_plan"] != false || ended["plan"] != nil || ended["today"] != nil || len(list(ended["history"])) != 1 {
		t.Fatalf("ended view = %v", ended)
	}
	if e.profileStatus(fanta) != "fit" {
		t.Errorf("status after end = %s, want fit", e.profileStatus(fanta))
	}
	e.call(seed.UserAnna, "POST", base+"/end", nil, http.StatusOK)
	if c := errCode(e.call(seed.UserAnna, "PATCH", base, map[string]any{"diagnosis": "x"}, http.StatusConflict)); c != "not_active" {
		t.Errorf("patch after end: %q", c)
	}
	if c := errCode(e.call(seed.UserAnna, "POST", base+"/days/2026-03-25/done", nil, http.StatusConflict)); c != "not_active" {
		t.Errorf("done after end: %q", c)
	}

	// A status the owner changed meanwhile survives the end of the plan.
	v = e.createPlan(seed.UserAnna, nil)
	base = "/api/v1/reha-plans/" + str(obj(v["plan"])["id"])
	e.exec(`UPDATE training_profiles SET status = 'pause' WHERE horse_id = $1`, fanta)
	e.call(seed.UserAnna, "POST", base+"/end", nil, http.StatusOK)
	if e.profileStatus(fanta) != "pause" {
		t.Errorf("status after end = %s, want pause (changed meanwhile)", e.profileStatus(fanta))
	}

	// A horse without training profile gets one with status reha.
	e.exec(`DELETE FROM training_profiles WHERE horse_id = $1`, fanta)
	e.createPlan(seed.UserAnna, nil)
	if e.profileStatus(fanta) != "reha" {
		t.Errorf("new profile status = %s, want reha", e.profileStatus(fanta))
	}
}

func TestMarkDone(t *testing.T) {
	e := newEnv(t)
	v := e.createPlan(seed.UserAnna, nil)
	base := "/api/v1/reha-plans/" + str(obj(v["plan"])["id"])

	out := e.call(seed.UserLea, "POST", base+"/days/2026-03-25/done", nil, http.StatusOK)
	today := obj(out["today"])
	if today["done"] != true || today["done_by"] != "Lea" {
		t.Fatalf("today after done = %v", today)
	}
	// Idempotent: a second tap by someone else keeps the first person.
	out = e.call(seed.UserAnna, "POST", base+"/days/2026-03-25/done", nil, http.StatusOK)
	if obj(out["today"])["done_by"] != "Lea" || e.count(`SELECT count(*) FROM reha_days`) != 1 {
		t.Fatalf("second tap = %v", out["today"])
	}
	if got := list(obj(out["plan"])["done_days"]); len(got) != 1 || got[0] != "2026-03-25" {
		t.Errorf("done_days = %v", got)
	}

	// Yesterday can be caught up, the future and days outside the plan cannot.
	e.call(seed.UserAnna, "POST", base+"/days/2026-03-24/done", nil, http.StatusOK)
	e.call(seed.UserAnna, "POST", base+"/days/2026-03-26/done", nil, http.StatusBadRequest)
	e.call(seed.UserAnna, "POST", base+"/days/2026-03-10/done", nil, http.StatusBadRequest)
	e.call(seed.UserAnna, "POST", base+"/days/gestern/done", nil, http.StatusBadRequest)

	// Undo: the owner may always, a rider only her own mark.
	e.call(seed.UserLea, "DELETE", base+"/days/2026-03-24/done", nil, http.StatusForbidden)
	e.call(seed.UserAnna, "DELETE", base+"/days/2026-03-25/done", nil, http.StatusOK)
	out = e.call(seed.UserLea, "POST", base+"/days/2026-03-25/done", nil, http.StatusOK)
	out = e.call(seed.UserLea, "DELETE", base+"/days/2026-03-25/done", nil, http.StatusOK)
	if obj(out["today"])["done"] != false || e.count(`SELECT count(*) FROM reha_days`) != 1 {
		t.Fatalf("after undo = %v", out["today"])
	}
	// Undoing a day nobody marked is a no-op.
	e.call(seed.UserAnna, "DELETE", base+"/days/2026-03-25/done", nil, http.StatusOK)
}

func TestIntegrationTodayAndWeek(t *testing.T) {
	e := newEnv(t)
	v := e.createPlan(seed.UserAnna, nil)
	base := "/api/v1/reha-plans/" + str(obj(v["plan"])["id"])

	// "Was heute?" shows only the allowed unit, with the ramp minutes.
	today := e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/today?minutes=60", nil, http.StatusOK)
	recs := list(today["recommendations"])
	if len(recs) != 1 {
		t.Fatalf("recommendations = %v", recs)
	}
	r := obj(recs[0])
	if r["activity"] != "groundwork" || num(r["minutes"]) != 14 || r["note"] != "nur Boden fest" || !strings.Contains(str(r["reason"]), "Schritt führen") {
		t.Fatalf("recommendation = %v", r)
	}
	rh := obj(today["reha"])
	if num(rh["minutes"]) != 14 || num(rh["min_minutes"]) != 10 || num(rh["max_minutes"]) != 20 || rh["done"] != false || rh["rest"] != false ||
		num(rh["phase_index"]) != 2 || num(rh["phases"]) != 4 || today["status"] != "reha" {
		t.Fatalf("reha = %v", rh)
	}
	// Less available time shortens the unit but never below the phase minimum.
	short := list(e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/today?minutes=12", nil, http.StatusOK)["recommendations"])
	if num(obj(short[0])["minutes"]) != 12 {
		t.Errorf("12 minutes available: %v", short[0])
	}
	tiny := list(e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/today?minutes=5", nil, http.StatusOK)["recommendations"])
	if num(obj(tiny[0])["minutes"]) != 10 {
		t.Errorf("5 minutes available: %v", tiny[0])
	}
	e.call(seed.UserAnna, "POST", base+"/days/2026-03-25/done", nil, http.StatusOK)
	if obj(e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/today", nil, http.StatusOK)["reha"])["done"] != true {
		t.Error("today must report the done mark")
	}

	// The week shows the reha entry of every day of the plan: planned minutes and done.
	week := e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/week?start=2026-03-23", nil, http.StatusOK)
	days := list(week["days"])
	type expect struct {
		phase   string
		minutes float64
		rest    bool
		done    bool
	}
	// Mon 03-23 .. Sun 03-29: 03-21 was the last box rest day, 03-22 is day 1 of "Schritt führen".
	exp := []expect{
		{"Schritt führen", 11, false, false}, {"Schritt führen", 13, false, false}, {"Schritt führen", 14, false, true},
		{"Schritt führen", 15, false, false}, {"Schritt führen", 16, false, false}, {"Schritt führen", 18, false, false},
		{"Schritt führen", 19, false, false},
	}
	// 03-23 is day 2 of the phase: 10 + 10/8 = 11.25 -> 11.
	for i, x := range exp {
		rh := obj(obj(days[i])["reha"])
		if rh == nil || rh["phase"] != x.phase || num(rh["minutes"]) != x.minutes || rh["rest"] != x.rest || rh["done"] != x.done {
			t.Errorf("week day %d reha = %v, want %+v", i, rh, x)
		}
	}
	next := list(e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/week?start=2026-03-30", nil, http.StatusOK)["days"])
	if a, b := obj(obj(next[0])["reha"]), obj(obj(next[1])["reha"]); num(a["minutes"]) != 20 || a["phase"] != "Schritt führen" || b["phase"] != "Schritt reiten" || num(b["minutes"]) != 20 {
		t.Errorf("phase change in the week: %v %v", a, b)
	}
	box := list(e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/week?start=2026-03-16", nil, http.StatusOK)["days"])
	if b := obj(obj(box[2])["reha"]); b["rest"] != true || b["phase"] != "Boxenruhe" || num(b["minutes"]) != 0 {
		t.Errorf("box rest day = %v", b)
	}
	if obj(box[0])["reha"] != nil {
		t.Errorf("the day before the plan must have no reha entry: %v", obj(box[0])["reha"])
	}
	if lunaWeek := list(e.call(seed.UserJan, "GET", "/api/v1/horses/"+luna+"/week", nil, http.StatusOK)["days"]); obj(lunaWeek[0])["reha"] != nil {
		t.Error("a horse without plan has no reha entries")
	}

	// A rest phase: "Was heute?" says so and offers nothing to do.
	e.call(seed.UserAnna, "PATCH", base, map[string]any{"start_date": "2026-03-22"}, http.StatusOK)
	rest := list(e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/today", nil, http.StatusOK)["recommendations"])
	if len(rest) != 1 || obj(rest[0])["activity"] != "rest" || !strings.Contains(str(obj(rest[0])["reason"]), "Boxenruhe") {
		t.Errorf("box rest recommendation = %v", rest)
	}
	// A plan that starts in the future or has finished gives no unit; the status stays reha (light only).
	e.call(seed.UserAnna, "PATCH", base, map[string]any{"start_date": "2026-03-30"}, http.StatusOK)
	tv := e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusOK)
	if tv["today"] != nil || obj(tv["plan"])["state"] != "upcoming" || num(obj(tv["plan"])["current_phase"]) != 0 {
		t.Errorf("upcoming plan = %v", tv)
	}
	e.call(seed.UserAnna, "PATCH", base, map[string]any{"start_date": "2026-01-01"}, http.StatusOK)
	tv = e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusOK)
	if tv["today"] != nil || obj(tv["plan"])["state"] != "finished" {
		t.Errorf("finished plan = %v", tv)
	}
}

func TestExerciseRequestsCarryTheRule(t *testing.T) {
	e := newEnv(t)
	create := func(user, horse, date string, note string) map[string]any {
		payload := map[string]any{"mode": "lunge"}
		if note != "" {
			payload["rules_note"] = note
		}
		out := e.call(user, "POST", "/api/v1/requests", map[string]any{"type": "exercise", "horse_id": horse, "date": date, "payload": payload}, http.StatusCreated)
		return out
	}
	note := func(r map[string]any) string { return str(obj(r["payload"])["rules_note"]) }

	// Without a plan the note is what the requester wrote.
	if got := note(create(seed.UserAnna, fanta, "2026-03-25", "nur Schritt")); got != "nur Schritt" {
		t.Errorf("without plan: %q", got)
	}
	e.createPlan(seed.UserAnna, nil)

	r := create(seed.UserAnna, fanta, "2026-03-25", "Trense putzen")
	if got, want := note(r), "Reha: Schritt führen 14 min, nur Boden fest\nTrense putzen"; got != want {
		t.Errorf("rules_note = %q, want %q", got, want)
	}
	// The rule follows the request's date: 03-24 is one day earlier in the ramp, 03-22 the first day.
	if got := note(create(seed.UserAnna, fanta, "2026-03-27", "")); got != "Reha: Schritt führen 16 min, nur Boden fest" {
		t.Errorf("03-27: %q", got)
	}
	// Any member may ask for help; the rule is added server-side.
	if got := note(create(seed.UserSarah, fanta, "2026-03-25", "")); !strings.HasPrefix(got, "Reha: Schritt führen 14 min") {
		t.Errorf("member request: %q", got)
	}
	// After the plan there is no unit; a rest day says so.
	if got := note(create(seed.UserAnna, fanta, "2026-06-01", "")); got != "Reha-Plan: keine Einheit an diesem Tag" {
		t.Errorf("after the plan: %q", got)
	}
	if got := note(create(seed.UserAnna, fanta, "2026-03-25", "Reha: erfunden")); strings.Contains(got, "erfunden") {
		t.Errorf("a typed Reha line must not survive: %q", got)
	}
	// Other horses are unaffected.
	if got := note(create(seed.UserJan, luna, "2026-03-25", "")); got != "" {
		t.Errorf("horse without plan: %q", got)
	}
	// Other request types are unaffected.
	feed := e.call(seed.UserAnna, "POST", "/api/v1/requests", map[string]any{"type": "feed_or_turnout", "horse_id": fanta, "date": "2026-03-25", "payload": map[string]any{"what": "feed"}}, http.StatusCreated)
	if strings.Contains(string(mustJSON(feed["payload"])), "Reha") {
		t.Errorf("feed payload = %v", feed["payload"])
	}

	// Editing refreshes the rule instead of duplicating it.
	id := str(r["id"])
	patched := e.call(seed.UserAnna, "PATCH", "/api/v1/requests/"+id, map[string]any{"date": "2026-03-26"}, http.StatusOK)
	if got, want := note(patched), "Reha: Schritt führen 15 min, nur Boden fest\nTrense putzen"; got != want {
		t.Errorf("after moving the date: %q, want %q", got, want)
	}
	patched = e.call(seed.UserAnna, "PATCH", "/api/v1/requests/"+id, map[string]any{"description": "neu"}, http.StatusOK)
	if strings.Count(note(patched), "Reha:") != 1 {
		t.Errorf("rule duplicated: %q", note(patched))
	}
	// When the plan ends, the next edit drops the line and keeps the requester's text.
	planID := e.call(seed.UserAnna, "GET", "/api/v1/horses/"+fanta+"/reha", nil, http.StatusOK)
	e.call(seed.UserAnna, "POST", "/api/v1/reha-plans/"+str(obj(planID["plan"])["id"])+"/end", nil, http.StatusOK)
	patched = e.call(seed.UserAnna, "PATCH", "/api/v1/requests/"+id, map[string]any{"description": "nochmal"}, http.StatusOK)
	if got := note(patched); got != "Trense putzen" {
		t.Errorf("after the plan ended: %q", got)
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
