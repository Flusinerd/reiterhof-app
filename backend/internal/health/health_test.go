package health_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/health"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const (
	otherStable = "00000000-0000-4000-8000-0000000009a1"
	otherUser   = "00000000-0000-4000-8000-0000000009a2"
)

// fixed "now": 2026-10-01 07:00 UTC = 09:00 in Europe/Berlin (CEST).
var fixedNow = time.Date(2026, 10, 1, 7, 0, 0, 0, time.UTC)

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
	now  time.Time
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()
	for _, q := range []string{
		// Start from a clean slate; the seed has items relative to the real date.
		`DELETE FROM health_items`,
		`INSERT INTO stables (id, name) VALUES ('` + otherStable + `', 'Anderer Stall')`,
		`INSERT INTO users (id, stable_id, name, email) VALUES ('` + otherUser + `', '` + otherStable + `', 'Fremd', 'fremd@example.org')`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	e := &env{t: t, pool: pool, now: fixedNow}
	e.h = httpapi.NewHandler(httpapi.Deps{Pool: pool, Now: func() time.Time { return e.now }})
	return e
}

func (e *env) call(user, method, path string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		req.Header.Set("Authorization", "Bearer "+authtest.TokenAt(e.t, e.pool, user, e.now))
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func (e *env) expect(rec *httptest.ResponseRecorder, want int) {
	e.t.Helper()
	if rec.Code != want {
		e.t.Fatalf("status = %d, want %d: %s", rec.Code, want, rec.Body)
	}
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func (e *env) addItem(horse, kind, label, due string, interval int, daily string) string {
	e.t.Helper()
	var id string
	err := e.pool.QueryRow(context.Background(), `INSERT INTO health_items (stable_id, horse_id, kind, label, due_date, interval_days, daily_time)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::date, NULLIF($6, 0), NULLIF($7, '')::time) RETURNING id`,
		seed.StableB, horse, kind, label, due, interval, daily).Scan(&id)
	if err != nil {
		e.t.Fatal(err)
	}
	return id
}

func TestNextDue(t *testing.T) {
	d := func(s string) *time.Time {
		v, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatal(err)
		}
		return &v
	}
	iv := func(n int) *int { return &n }
	today := *d("2026-10-01")
	cases := []struct {
		name     string
		prev     *time.Time
		interval *int
		want     string
	}{
		{"on time", d("2026-10-03"), iv(42), "2026-11-14"},
		{"due today", d("2026-10-01"), iv(30), "2026-10-31"},
		{"overdue by less than one interval", d("2026-09-10"), iv(42), "2026-10-22"},
		{"overdue by several intervals", d("2026-05-01"), iv(90), "2026-10-28"},
		{"no previous date counts from today", nil, iv(28), "2026-10-29"},
		{"no interval clears", d("2026-10-03"), nil, ""},
	}
	for _, c := range cases {
		got := health.NextDue(c.prev, c.interval, today)
		switch {
		case c.want == "" && got != nil:
			t.Errorf("%s: got %v, want nil", c.name, got)
		case c.want != "" && (got == nil || got.Format("2006-01-02") != c.want):
			t.Errorf("%s: got %v, want %s", c.name, got, c.want)
		}
	}
}

func TestListSummaryAndPermissions(t *testing.T) {
	e := setup(t)
	e.addItem(seed.HorseLuna, "vaccination", "Influenza", "2026-12-01", 365, "")
	e.addItem(seed.HorseLuna, "vaccination", "Tetanus", "2026-11-01", 365, "") // sooner: shown in the tile
	e.addItem(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-07", 42, "")
	e.addItem(seed.HorseLuna, "deworming", "Wurmkur", "2026-09-20", 90, "") // overdue
	e.addItem(seed.HorseLuna, "physio", "Physio", "", 0, "")
	path := "/api/v1/horses/" + seed.HorseLuna + "/health"

	// Every member reads.
	for _, u := range []string{seed.UserJan, seed.UserMia, seed.UserTom} {
		e.expect(e.call(u, "GET", path, nil), 200)
	}
	rec := e.call(seed.UserTom, "GET", path, nil)
	resp := decode[health.Response](t, rec)
	if resp.Today != "2026-10-01" || len(resp.Items) != 5 || resp.CanManage {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.Items[0].Label != "Wurmkur" || resp.Items[4].Label != "Physio" {
		t.Errorf("order = %v ... %v", resp.Items[0].Label, resp.Items[4].Label)
	}
	tile := func(kind string) *health.Item { return resp.Summary[kind] }
	if v := tile("vaccination"); v == nil || v.Label != "Tetanus" || *v.DaysUntilDue != 31 {
		t.Errorf("vaccination tile = %+v", v)
	}
	if f := tile("farrier"); f == nil || *f.DaysUntilDue != 6 {
		t.Errorf("farrier tile = %+v", f)
	}
	if d := tile("deworming"); d == nil || *d.DaysUntilDue != -11 {
		t.Errorf("deworming tile = %+v", d)
	}
	if v, ok := resp.Summary["dentist"]; !ok || v != nil {
		t.Errorf("dentist tile = %+v (present %v), want explicit null", v, ok)
	}
	if _, ok := resp.Summary["physio"]; ok {
		t.Error("physio must not be a tile")
	}
	if !decode[health.Response](t, e.call(seed.UserJan, "GET", path, nil)).CanManage {
		t.Error("owner should manage")
	}

	e.expect(e.call(otherUser, "GET", path, nil), 404)
	e.expect(e.call("", "GET", path, nil), 401)
	e.expect(e.call(seed.UserJan, "GET", "/api/v1/horses/nope/health", nil), 404)
}

func TestCRUDPermissions(t *testing.T) {
	e := setup(t)
	create := "/api/v1/horses/" + seed.HorseFanta + "/health-items" // Anna's horse, Lea rides it
	body := map[string]any{"kind": "farrier", "label": "Hufschmied", "due_date": "2026-10-10", "interval_days": 42}

	e.expect(e.call(seed.UserTom, "POST", create, body), 403)
	e.expect(e.call(seed.UserLea, "POST", create, body), 403)
	e.expect(e.call(otherUser, "POST", create, body), 404)
	e.expect(e.call("", "POST", create, body), 401)
	rec := e.call(seed.UserAnna, "POST", create, body)
	e.expect(rec, 201)
	it := decode[health.Item](t, rec)
	if it.ID == "" || it.Kind != "farrier" || *it.DueDate != "2026-10-10" || *it.IntervalDays != 42 || *it.DaysUntilDue != 9 {
		t.Fatalf("created = %+v", it)
	}
	e.expect(e.call(seed.UserJan, "POST", create, map[string]any{"kind": "medication", "label": "Pergolid", "daily_time": "08:30"}), 201) // admin

	// Validation.
	for name, b := range map[string]map[string]any{
		"no kind":      {"label": "x"},
		"bad kind":     {"kind": "massage", "label": "x"},
		"empty label":  {"kind": "vet", "label": " "},
		"bad date":     {"kind": "vet", "label": "x", "due_date": "01.10.2026"},
		"bad interval": {"kind": "vet", "label": "x", "interval_days": -3},
		"bad time":     {"kind": "medication", "label": "x", "daily_time": "25:00"},
	} {
		if rec := e.call(seed.UserAnna, "POST", create, b); rec.Code != 400 {
			t.Errorf("%s: status = %d %s", name, rec.Code, rec.Body)
		}
	}

	item := "/api/v1/health-items/" + it.ID
	e.expect(e.call(seed.UserTom, "PATCH", item, map[string]any{"label": "x"}), 403)
	e.expect(e.call(seed.UserLea, "PATCH", item, map[string]any{"label": "x"}), 403)
	e.expect(e.call(otherUser, "PATCH", item, map[string]any{"label": "x"}), 404)
	e.expect(e.call(seed.UserAnna, "PATCH", "/api/v1/health-items/nope", map[string]any{"label": "x"}), 404)
	rec = e.call(seed.UserAnna, "PATCH", item, map[string]any{"label": "Hufschmied Kruse", "due_date": "", "note": "Vormittags", "interval_days": 0})
	e.expect(rec, 200)
	it = decode[health.Item](t, rec)
	if it.Label != "Hufschmied Kruse" || it.DueDate != nil || it.IntervalDays != nil || *it.Note != "Vormittags" || it.Kind != "farrier" {
		t.Errorf("patched = %+v", it)
	}
	e.expect(e.call(seed.UserAnna, "PATCH", item, map[string]any{}), 200)

	e.expect(e.call(seed.UserTom, "DELETE", item, nil), 403)
	e.expect(e.call(otherUser, "DELETE", item, nil), 404)
	e.expect(e.call(seed.UserAnna, "DELETE", item, nil), 204)
	e.expect(e.call(seed.UserAnna, "DELETE", item, nil), 404)
}

func TestDone(t *testing.T) {
	e := setup(t)
	farrier := e.addItem(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-07", 42, "")
	overdue := e.addItem(seed.HorseLuna, "deworming", "Wurmkur", "2026-06-01", 90, "")
	oneOff := e.addItem(seed.HorseLuna, "vet", "Kontrolle", "2026-10-05", 0, "")

	e.expect(e.call(seed.UserMia, "POST", "/api/v1/health-items/"+farrier+"/done", nil), 403)
	e.expect(e.call(seed.UserTom, "POST", "/api/v1/health-items/"+farrier+"/done", nil), 403)
	e.expect(e.call(otherUser, "POST", "/api/v1/health-items/"+farrier+"/done", nil), 404)
	e.expect(e.call(seed.UserJan, "POST", "/api/v1/health-items/nope/done", nil), 404)

	rec := e.call(seed.UserJan, "POST", "/api/v1/health-items/"+farrier+"/done", nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); *it.DueDate != "2026-11-18" || *it.DaysUntilDue != 48 {
		t.Errorf("farrier after done = %+v", it)
	}
	rec = e.call(seed.UserJan, "POST", "/api/v1/health-items/"+overdue+"/done", nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); *it.DueDate != "2026-11-28" {
		t.Errorf("overdue after done = %v", *it.DueDate)
	}
	rec = e.call(seed.UserJan, "POST", "/api/v1/health-items/"+oneOff+"/done", nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); it.DueDate != nil {
		t.Errorf("one-off after done = %+v", it)
	}
}

// recorder is a Notifier that records calls and can fail on demand.
type recorder struct {
	mu    sync.Mutex
	calls []call
	fail  bool
}

type call struct {
	stableID string
	users    []string
	kind     string
	title    string
	body     string
	data     map[string]any
}

func (r *recorder) NotifyUsers(_ context.Context, stableID string, userIDs []string, kind, title, body string, data map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail {
		return errors.New("push down")
	}
	r.calls = append(r.calls, call{stableID, userIDs, kind, title, body, data})
	return nil
}

func (r *recorder) take() []call {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.calls
	r.calls = nil
	return c
}

func newJob(e *env, rec *recorder) *health.Reminders {
	return &health.Reminders{Pool: e.pool, Notify: rec, Now: func() time.Time { return e.now }}
}

func has(users []string, id string) bool {
	for _, u := range users {
		if u == id {
			return true
		}
	}
	return false
}

func TestDueReminders(t *testing.T) {
	e := setup(t)
	rec := &recorder{}
	job := newJob(e, rec)
	ctx := context.Background()

	e.addItem(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-08", 42, "")     // in 7 days
	e.addItem(seed.HorseLuna, "vaccination", "Influenza", "2026-10-02", 365, "") // tomorrow
	e.addItem(seed.HorseLuna, "dentist", "Zahnarzt", "2026-10-05", 365, "")      // in 4 days: nothing
	e.addItem(seed.HorseLuna, "deworming", "Wurmkur", "2026-09-25", 90, "")      // overdue: nothing
	e.addItem(seed.HorseBalu, "farrier", "Hufschmied", "2026-10-08", 56, "")     // Balu: owner Jonas, no riders

	if err := job.Run(ctx); err != nil {
		t.Fatal(err)
	}
	calls := rec.take()
	if len(calls) != 3 {
		t.Fatalf("calls = %+v", calls)
	}
	var sawWeek, sawTomorrow, sawBalu bool
	for _, c := range calls {
		if c.kind != push.KindHealthDue || c.stableID != seed.StableB {
			t.Errorf("call = %+v", c)
		}
		switch {
		case c.data["horse_id"] == seed.HorseBalu:
			sawBalu = true
			if len(c.users) != 1 || c.users[0] != seed.UserJonas {
				t.Errorf("Balu recipients = %v", c.users)
			}
		case c.title == "Fällig: Luna" && c.body == "Hufschmied ist in 7 Tagen fällig (08.10.2026).":
			sawWeek = true
		case c.title == "Fällig: Luna" && c.body == "Influenza ist morgen fällig (02.10.2026).":
			sawTomorrow = true
		default:
			t.Errorf("unexpected %q %q", c.title, c.body)
		}
		if c.data["horse_id"] == seed.HorseLuna {
			// Owner Jan and rider Mia, nobody else.
			if len(c.users) != 2 || !has(c.users, seed.UserJan) || !has(c.users, seed.UserMia) {
				t.Errorf("Luna recipients = %v", c.users)
			}
		}
	}
	if !sawWeek || !sawTomorrow || !sawBalu {
		t.Errorf("missing reminder: week=%v tomorrow=%v balu=%v", sawWeek, sawTomorrow, sawBalu)
	}

	// Idempotent: same day, later runs and a run on the next tick send nothing.
	for _, add := range []time.Duration{0, 15 * time.Minute, 5 * time.Hour} {
		e.now = fixedNow.Add(add)
		if err := job.Run(ctx); err != nil {
			t.Fatal(err)
		}
		if c := rec.take(); len(c) != 0 {
			t.Fatalf("+%v: duplicate calls %+v", add, c)
		}
	}
	var n int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM reminders WHERE kind = 'health_due' AND sent_at IS NOT NULL`).Scan(&n); err != nil || n != 5 {
		t.Errorf("reminder rows = %d %v, want 5 (2 users x 2 items + 1 owner)", n, err)
	}

	// The next day: farrier in 6 days, influenza due today, dentist in 3 days: nothing to send.
	e.now = fixedNow.Add(24 * time.Hour)
	if err := job.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if c := rec.take(); len(c) != 0 {
		t.Fatalf("next day: %+v", c)
	}
	// On Oct 4 the dentist item (Oct 5) is due tomorrow.
	e.now = fixedNow.Add(3 * 24 * time.Hour)
	if err := job.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if c := rec.take(); len(c) != 1 || c[0].body != "Zahnarzt ist morgen fällig (05.10.2026)." {
		t.Fatalf("dentist tomorrow: %+v", c)
	}
}

func TestDueReminderWaitsForMorningAndUsesStableTimezone(t *testing.T) {
	e := setup(t)
	rec := &recorder{}
	job := newJob(e, rec)
	e.addItem(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-08", 42, "")

	// 05:30 UTC = 07:30 Berlin: too early.
	e.now = time.Date(2026, 10, 1, 5, 30, 0, 0, time.UTC)
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c := rec.take(); len(c) != 0 {
		t.Fatalf("before 08:00 local: %+v", c)
	}
	// 06:00 UTC = 08:00 Berlin.
	e.now = time.Date(2026, 10, 1, 6, 0, 0, 0, time.UTC)
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c := rec.take(); len(c) != 1 {
		t.Fatalf("at 08:00 local: %+v", c)
	}
	// 23:30 UTC on Sep 30 is already Oct 1 01:30 in Berlin, but before 08:00: the local day counts.
	e.now = time.Date(2026, 9, 30, 23, 30, 0, 0, time.UTC)
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c := rec.take(); len(c) != 0 {
		t.Fatalf("night: %+v", c)
	}
}

func TestDoneCreatesNewReminderCycle(t *testing.T) {
	e := setup(t)
	rec := &recorder{}
	job := newJob(e, rec)
	id := e.addItem(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-08", 7, "")
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(rec.take()) != 1 {
		t.Fatal("first reminder missing")
	}
	// Done today: next due 2026-10-15, i.e. in 14 days. Seven days later the next reminder goes out.
	e.expect(e.call(seed.UserJan, "POST", "/api/v1/health-items/"+id+"/done", nil), 200)
	e.now = fixedNow.Add(7 * 24 * time.Hour)
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c := rec.take(); len(c) != 1 || c[0].body != "Hufschmied ist in 7 Tagen fällig (15.10.2026)." {
		t.Fatalf("second cycle: %+v", c)
	}
}

func TestPushFailureIsRetried(t *testing.T) {
	e := setup(t)
	rec := &recorder{fail: true}
	job := newJob(e, rec)
	e.addItem(seed.HorseBalu, "farrier", "Hufschmied", "2026-10-08", 56, "")

	if err := job.Run(context.Background()); err == nil {
		t.Fatal("expected the push error")
	}
	rec.fail = false
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c := rec.take(); len(c) != 1 {
		t.Fatalf("retry: %+v", c)
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c := rec.take(); len(c) != 0 {
		t.Fatalf("after success: %+v", c)
	}
}

func TestMedicationReminders(t *testing.T) {
	e := setup(t)
	rec := &recorder{}
	job := newJob(e, rec)
	ctx := context.Background()
	e.addItem(seed.HorseFanta, "medication", "Pergolid 1 mg", "", 0, "08:00")
	e.addItem(seed.HorseFanta, "medication", "Kur", "2026-09-30", 0, "09:00") // course ended yesterday

	run := func(at time.Time) []call {
		t.Helper()
		e.now = at
		if err := job.Run(ctx); err != nil {
			t.Fatal(err)
		}
		return rec.take()
	}
	berlin := func(day, h, m int) time.Time {
		return time.Date(2026, 10, day, h, m, 0, 0, time.FixedZone("CEST", 2*3600)).UTC()
	}

	if c := run(berlin(1, 7, 59)); len(c) != 0 {
		t.Fatalf("before daily_time: %+v", c)
	}
	c := run(berlin(1, 8, 0))
	if len(c) != 1 || c[0].kind != push.KindMedication || c[0].title != "Medikament: Fanta" || c[0].body != "Pergolid 1 mg, 08:00 Uhr." {
		t.Fatalf("at daily_time: %+v", c)
	}
	// Owner Anna and rider Lea.
	if len(c[0].users) != 2 || !has(c[0].users, seed.UserAnna) || !has(c[0].users, seed.UserLea) {
		t.Errorf("recipients = %v", c[0].users)
	}
	if c := run(berlin(1, 8, 15)); len(c) != 0 {
		t.Fatalf("duplicate: %+v", c)
	}
	if c := run(berlin(1, 20, 0)); len(c) != 0 {
		t.Fatalf("evening: %+v", c)
	}
	// Next day again.
	if c := run(berlin(2, 8, 5)); len(c) != 1 {
		t.Fatalf("next day: %+v", c)
	}
	// Missed by more than the window (server was down): skipped, not sent late.
	if c := run(berlin(3, 12, 0)); len(c) != 0 {
		t.Fatalf("late: %+v", c)
	}
}

func TestSeedHealthData(t *testing.T) {
	// The demo data shows a mix of statuses for Luna (used by the manual checks in docs/domains/horses.md).
	pool := dbtest.NewSeeded(t)
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM health_items WHERE horse_id = $1`, seed.HorseLuna).Scan(&n); err != nil || n < 4 {
		t.Errorf("Luna items = %d %v", n, err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM emergency_contacts WHERE horse_id = $1`, seed.HorseLuna).Scan(&n); err != nil || n < 2 {
		t.Errorf("Luna contacts = %d %v", n, err)
	}
}
