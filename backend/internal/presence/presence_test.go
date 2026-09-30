package presence_test

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
	"github.com/Flusinerd/reiterhof-app/backend/internal/presence"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const (
	stableX = "00000000-0000-4000-8000-0000000009a1"
	userX   = "00000000-0000-4000-8000-0000000009a2"
)

// clock is 2026-09-30 20:00 in Berlin (18:00 UTC, summer time).
var clock = time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC)

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
}

func newEnv(t *testing.T) *env {
	pool := dbtest.NewSeeded(t)
	return &env{t: t, pool: pool, h: httpapi.NewHandler(httpapi.Deps{Pool: pool, Now: func() time.Time { return clock }})}
}

func (e *env) do(userID, method, path, body string) *httptest.ResponseRecorder {
	e.t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		req.Header.Set("Authorization", "Bearer "+authtest.TokenAt(e.t, e.pool, userID, clock))
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func (e *env) overview(userID string) presence.Overview {
	e.t.Helper()
	rec := e.do(userID, "GET", "/api/v1/presence", "")
	if rec.Code != 200 {
		e.t.Fatalf("GET /presence = %d %s", rec.Code, rec.Body)
	}
	var o presence.Overview
	if err := json.Unmarshal(rec.Body.Bytes(), &o); err != nil {
		e.t.Fatal(err)
	}
	return o
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
}

func (e *env) setVisibility(userID, vis string) {
	e.exec(`UPDATE users SET presence_visibility = $2 WHERE id = $1`, userID, vis)
}

// visit inserts a finished visit.
func (e *env) visit(userID string, arrived, left time.Time) {
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at, left_at) VALUES ($1, $2, $3, $4)`,
		seed.StableB, userID, arrived, left)
}

func (e *env) open(userID string, arrived time.Time) {
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at) VALUES ($1, $2, $3)`, seed.StableB, userID, arrived)
}

func hereIDs(o presence.Overview) map[string]presence.Here {
	m := map[string]presence.Here{}
	for _, h := range o.Here {
		m[h.UserID] = h
	}
	return m
}

func recentIDs(o presence.Overview) map[string]presence.Recent {
	m := map[string]presence.Recent{}
	for _, r := range o.Recent {
		m[r.UserID] = r
	}
	return m
}

func TestRequiresSession(t *testing.T) {
	e := newEnv(t)
	for _, r := range [][2]string{{"POST", "/api/v1/presence/check-in"}, {"POST", "/api/v1/presence/check-out"}, {"GET", "/api/v1/presence"}} {
		if rec := e.do("", r[0], r[1], ""); rec.Code != 401 {
			t.Errorf("%s %s without session = %d, want 401", r[0], r[1], rec.Code)
		}
	}
}

func TestCheckInOutIsIdempotent(t *testing.T) {
	e := newEnv(t)

	first := e.do(seed.UserAnna, "POST", "/api/v1/presence/check-in", "") // empty body = manual
	if first.Code != 200 {
		t.Fatalf("check-in = %d %s", first.Code, first.Body)
	}
	var a, b struct{ Visit *presence.Visit }
	_ = json.Unmarshal(first.Body.Bytes(), &a)
	if a.Visit == nil || a.Visit.Source != "manual" || !a.Visit.ArrivedAt.Equal(clock) || a.Visit.LeftAt != nil {
		t.Fatalf("visit = %+v", a.Visit)
	}

	// Second check-in (even with another source) returns the same open visit.
	second := e.do(seed.UserAnna, "POST", "/api/v1/presence/check-in", `{"source":"geofence"}`)
	_ = json.Unmarshal(second.Body.Bytes(), &b)
	if b.Visit == nil || b.Visit.ID != a.Visit.ID || b.Visit.Source != "manual" {
		t.Fatalf("second check-in = %+v, want the first visit", b.Visit)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM presence WHERE user_id = $1`, seed.UserAnna).Scan(&n)
	if n != 1 {
		t.Fatalf("%d rows, want 1", n)
	}

	out := e.do(seed.UserAnna, "POST", "/api/v1/presence/check-out", "")
	var c struct{ Visit *presence.Visit }
	_ = json.Unmarshal(out.Body.Bytes(), &c)
	if out.Code != 200 || c.Visit == nil || c.Visit.ID != a.Visit.ID || c.Visit.LeftAt == nil {
		t.Fatalf("check-out = %d %s", out.Code, out.Body)
	}

	// Check-out without an open visit is a no-op.
	again := e.do(seed.UserAnna, "POST", "/api/v1/presence/check-out", "")
	if again.Code != 200 || strings.TrimSpace(again.Body.String()) != `{"visit":null}` {
		t.Fatalf("second check-out = %d %s", again.Code, again.Body)
	}

	// After leaving, a new visit can start.
	third := e.do(seed.UserAnna, "POST", "/api/v1/presence/check-in", `{"source":"geofence"}`)
	var d struct{ Visit *presence.Visit }
	_ = json.Unmarshal(third.Body.Bytes(), &d)
	if d.Visit == nil || d.Visit.ID == a.Visit.ID || d.Visit.Source != "geofence" {
		t.Fatalf("new visit = %+v", d.Visit)
	}
}

func TestCheckInValidatesSource(t *testing.T) {
	e := newEnv(t)
	if rec := e.do(seed.UserAnna, "POST", "/api/v1/presence/check-in", `{"source":"telepathy"}`); rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if rec := e.do(seed.UserAnna, "POST", "/api/v1/presence/check-in", `{"nope":1}`); rec.Code != 400 {
		t.Fatalf("unknown field: status = %d, want 400", rec.Code)
	}
}

func TestOverviewShowsOwnDataAndOthers(t *testing.T) {
	e := newEnv(t)
	e.do(seed.UserAnna, "POST", "/api/v1/presence/check-in", "")
	e.open(seed.UserTom, clock.Add(-30*time.Minute))
	e.visit(seed.UserKai, clock.Add(-26*time.Hour), clock.Add(-24*time.Hour))
	e.visit(seed.UserMia, clock.Add(-2*time.Hour), clock.Add(-1*time.Hour)) // earlier today

	o := e.overview(seed.UserAnna)
	if o.Me.OpenVisit == nil || o.Me.Visibility != "all" {
		t.Fatalf("me = %+v", o.Me)
	}
	here := hereIDs(o)
	if _, self := here[seed.UserAnna]; self || len(here) != 1 || here[seed.UserTom].Since == nil {
		t.Fatalf("here = %+v (self must not be listed, Tom must have a time)", o.Here)
	}
	if !here[seed.UserTom].Since.Equal(clock.Add(-30 * time.Minute)) {
		t.Errorf("since = %v", here[seed.UserTom].Since)
	}
	recent := recentIDs(o)
	if len(recent) != 2 {
		t.Fatalf("recent = %+v", o.Recent)
	}
	if r := recent[seed.UserMia]; !r.Today || r.LastSeenDate != "2026-09-30" || r.LastSeenAt == nil {
		t.Errorf("Mia = %+v", r)
	}
	if r := recent[seed.UserKai]; r.Today || r.LastSeenDate != "2026-09-29" {
		t.Errorf("Kai = %+v", r)
	}
	if o.Recent[0].UserID != seed.UserMia {
		t.Errorf("recent must be newest first: %+v", o.Recent)
	}
	// Tom is here, so he is not "zuletzt gesehen"; nobody who was never here is listed.
	if _, ok := recent[seed.UserTom]; ok {
		t.Error("a person who is here must not be in recent")
	}

	// After checking out, Anna's last visit is reported.
	e.do(seed.UserAnna, "POST", "/api/v1/presence/check-out", "")
	o = e.overview(seed.UserAnna)
	if o.Me.OpenVisit != nil || o.Me.LastVisit == nil || o.Me.LastVisit.LeftAt == nil {
		t.Fatalf("me after check-out = %+v", o.Me)
	}
	if got := hereIDs(e.overview(seed.UserTom)); got[seed.UserAnna].UserID != "" {
		t.Error("Anna checked out but is still listed")
	}
}

func TestVisibilityAll(t *testing.T) {
	e := newEnv(t)
	e.setVisibility(seed.UserAnna, "all")
	e.open(seed.UserAnna, clock.Add(-time.Hour))
	e.visit(seed.UserKai, clock.Add(-3*time.Hour), clock.Add(-2*time.Hour))
	e.setVisibility(seed.UserKai, "all")

	o := e.overview(seed.UserTom)
	if h := hereIDs(o)[seed.UserAnna]; h.Since == nil {
		t.Errorf("all: since missing: %+v", h)
	}
	if r := recentIDs(o)[seed.UserKai]; r.LastSeenAt == nil || r.LastSeenDate == "" {
		t.Errorf("all: recent = %+v", r)
	}
}

func TestVisibilityOnlyDay(t *testing.T) {
	e := newEnv(t)
	e.setVisibility(seed.UserAnna, "only_day")
	e.setVisibility(seed.UserKai, "only_day")
	e.open(seed.UserAnna, clock.Add(-time.Hour))
	e.visit(seed.UserKai, clock.Add(-3*time.Hour), clock.Add(-2*time.Hour))

	for _, viewer := range []string{seed.UserTom, seed.UserJan /* admin: no exception */} {
		o := e.overview(viewer)
		h, ok := hereIDs(o)[seed.UserAnna]
		if !ok || h.Since != nil {
			t.Errorf("only_day here for %s = %+v, ok=%v (present without a time expected)", viewer, h, ok)
		}
		r, ok := recentIDs(o)[seed.UserKai]
		if !ok || r.LastSeenAt != nil || r.LastSeenDate != "2026-09-30" || !r.Today {
			t.Errorf("only_day recent for %s = %+v, ok=%v (date without a time expected)", viewer, r, ok)
		}
	}

	// The raw JSON must not contain the time at all.
	body := e.do(seed.UserTom, "GET", "/api/v1/presence", "").Body.String()
	for _, leak := range []string{clock.Add(-time.Hour).Format(time.RFC3339), clock.Add(-2 * time.Hour).Format(time.RFC3339)} {
		if strings.Contains(body, leak) {
			t.Errorf("body leaks %s: %s", leak, body)
		}
	}

	// The person sees their own full data.
	o := e.overview(seed.UserAnna)
	if o.Me.OpenVisit == nil || !o.Me.OpenVisit.ArrivedAt.Equal(clock.Add(-time.Hour)) || o.Me.Visibility != "only_day" {
		t.Errorf("own data = %+v", o.Me)
	}
}

func TestVisibilityHidden(t *testing.T) {
	e := newEnv(t)
	e.setVisibility(seed.UserAnna, "hidden")
	e.setVisibility(seed.UserKai, "hidden")
	e.open(seed.UserAnna, clock.Add(-time.Hour))
	e.visit(seed.UserKai, clock.Add(-3*time.Hour), clock.Add(-2*time.Hour))
	e.visit(seed.UserAnna, clock.Add(-30*time.Hour), clock.Add(-29*time.Hour))

	for _, viewer := range []string{seed.UserTom, seed.UserJan /* admin: no exception */} {
		body := e.do(viewer, "GET", "/api/v1/presence", "").Body.String()
		o := e.overview(viewer)
		if len(o.Here) != 0 || len(o.Recent) != 0 {
			t.Errorf("viewer %s sees hidden people: %s", viewer, body)
		}
		if strings.Contains(body, "Anna") || strings.Contains(body, "Kai") {
			t.Errorf("viewer %s: names leak: %s", viewer, body)
		}
	}

	// Hidden people still see themselves, and everyone else.
	e.open(seed.UserTom, clock.Add(-10*time.Minute))
	o := e.overview(seed.UserAnna)
	if o.Me.OpenVisit == nil || o.Me.Visibility != "hidden" || o.Me.LastVisit == nil {
		t.Errorf("own data = %+v", o.Me)
	}
	if _, ok := hereIDs(o)[seed.UserTom]; !ok {
		t.Errorf("hidden people can see others: %+v", o.Here)
	}
}

func TestVisibilityChangeAppliesImmediately(t *testing.T) {
	e := newEnv(t)
	e.open(seed.UserAnna, clock.Add(-time.Hour))
	if _, ok := hereIDs(e.overview(seed.UserTom))[seed.UserAnna]; !ok {
		t.Fatal("Anna should be visible by default")
	}
	if rec := e.do(seed.UserAnna, "PATCH", "/api/v1/me", `{"presence_visibility":"hidden"}`); rec.Code != 200 {
		t.Fatalf("PATCH /me = %d %s", rec.Code, rec.Body)
	}
	if _, ok := hereIDs(e.overview(seed.UserTom))[seed.UserAnna]; ok {
		t.Fatal("Anna is hidden now")
	}
}

func TestCrossStableIsolation(t *testing.T) {
	e := newEnv(t)
	e.exec(`INSERT INTO stables (id, name) VALUES ($1, 'Other Stable')`, stableX)
	e.exec(`INSERT INTO users (id, stable_id, name, email) VALUES ($1, $2, 'Xaver', 'xaver@example.org')`, userX, stableX)

	e.open(seed.UserAnna, clock.Add(-time.Hour))
	if rec := e.do(userX, "POST", "/api/v1/presence/check-in", ""); rec.Code != 200 {
		t.Fatalf("check-in X = %d %s", rec.Code, rec.Body)
	}
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at, left_at) VALUES ($1, $2, $3, $4)`,
		stableX, userX, clock.Add(-50*time.Hour), clock.Add(-49*time.Hour))

	if o := e.overview(seed.UserTom); len(hereIDs(o)) != 1 || hereIDs(o)[userX].UserID != "" || len(o.Recent) != 0 {
		t.Errorf("stable B sees %+v", o)
	}
	o := e.overview(userX)
	if len(o.Here) != 0 || len(o.Recent) != 0 || o.Me.OpenVisit == nil {
		t.Errorf("stable X sees %+v", o)
	}

	// Checking out only touches the own visit.
	e.do(userX, "POST", "/api/v1/presence/check-out", "")
	if _, ok := hereIDs(e.overview(seed.UserTom))[seed.UserAnna]; !ok {
		t.Error("Anna's visit was affected by X")
	}
}

func TestUsualArrivalHint(t *testing.T) {
	e := newEnv(t)
	day := func(daysAgo int, hour, min int) time.Time {
		// Local (Berlin, UTC+2 in summer) wall clock time on that day.
		return time.Date(2026, 9, 30-daysAgo, hour-2, min, 0, 0, time.UTC)
	}
	// Mia: 5 visits, median 19:10 -> 19. Lea: only 3 visits. Kai: 5 visits but only_day.
	for i, hm := range [][2]int{{18, 50}, {19, 10}, {19, 20}, {19, 0}, {21, 30}} {
		e.visit(seed.UserMia, day(i+1, hm[0], hm[1]), day(i+1, hm[0]+1, hm[1]))
		e.visit(seed.UserKai, day(i+1, hm[0], hm[1]), day(i+1, hm[0]+1, hm[1]))
	}
	for i := 0; i < 3; i++ {
		e.visit(seed.UserLea, day(i+1, 8, 0), day(i+1, 9, 0))
	}
	// Old visits (older than eight weeks) do not count: Jonas has 4, but 3 of them are old.
	e.visit(seed.UserJonas, day(1, 7, 0), day(1, 8, 0))
	for i := 0; i < 3; i++ {
		e.visit(seed.UserJonas, day(70+i, 7, 0), day(70+i, 8, 0))
	}
	e.setVisibility(seed.UserKai, "only_day")

	r := recentIDs(e.overview(seed.UserTom))
	if h := r[seed.UserMia].UsualArrivalHour; h == nil || *h != 19 {
		t.Errorf("Mia hint = %v, want 19", h)
	}
	if r[seed.UserLea].UsualArrivalHour != nil {
		t.Error("Lea has < 4 visits, no hint expected")
	}
	if r[seed.UserKai].UsualArrivalHour != nil {
		t.Error("only_day people get no hint")
	}
	if r[seed.UserJonas].UsualArrivalHour != nil {
		t.Error("visits older than 8 weeks must not count")
	}
}

func TestCloseStale(t *testing.T) {
	e := newEnv(t)
	e.open(seed.UserAnna, clock.Add(-13*time.Hour)) // stale
	e.open(seed.UserTom, clock.Add(-11*time.Hour))  // still fine
	e.open(seed.UserKai, clock.Add(-12*time.Hour))  // exactly 12 h: not older than the limit
	e.visit(seed.UserMia, clock.Add(-40*time.Hour), clock.Add(-39*time.Hour))

	n, err := presence.CloseStale(context.Background(), e.pool, clock, presence.StaleAfter)
	if err != nil || n != 1 {
		t.Fatalf("CloseStale = %d, %v; want 1", n, err)
	}
	var left time.Time
	if err := e.pool.QueryRow(context.Background(), `SELECT left_at FROM presence WHERE user_id = $1`, seed.UserAnna).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if want := clock.Add(-13*time.Hour + presence.StaleAfter); !left.Equal(want) {
		t.Errorf("left_at = %v, want arrival + 12 h = %v", left, want)
	}
	here := hereIDs(e.overview(seed.UserJan))
	if len(here) != 2 || here[seed.UserTom].UserID == "" || here[seed.UserKai].UserID == "" {
		t.Errorf("here = %+v", here)
	}
	// Running it again finds nothing.
	if n, err := presence.CloseStale(context.Background(), e.pool, clock, presence.StaleAfter); err != nil || n != 0 {
		t.Errorf("second run = %d, %v", n, err)
	}
}

func TestMeExposesGeofenceData(t *testing.T) {
	e := newEnv(t)
	rec := e.do(seed.UserAnna, "GET", "/api/v1/me", "")
	var me struct {
		Stable struct {
			Lat             *float64 `json:"lat"`
			Lng             *float64 `json:"lng"`
			GeofenceRadiusM int      `json:"geofence_radius_m"`
		} `json:"stable"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.Stable.Lat == nil || *me.Stable.Lat != 51.66 || me.Stable.Lng == nil || *me.Stable.Lng != 6.96 || me.Stable.GeofenceRadiusM != 150 {
		t.Errorf("stable = %+v", me.Stable)
	}
}
