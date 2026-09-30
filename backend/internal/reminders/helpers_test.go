package reminders_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push/pushtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/reminders"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

var berlin = func() *time.Location {
	l, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	return l
}()

func berlinAt(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, berlin)
}

// env is a seeded database whose date-dependent example data is removed (it is relative to
// the real clock), a router and a fake clock (Wed 2026-09-30 19:00 Berlin).
type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
	svc  *reminders.Service
	fake *push.Fake
	mu   sync.Mutex
	now  time.Time
}

func (e *env) clock() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.now
}

func (e *env) at(t time.Time) {
	e.mu.Lock()
	e.now = t
	e.mu.Unlock()
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := dbtest.NewSeeded(t)
	e := &env{t: t, pool: pool, fake: &push.Fake{}, now: berlinAt(2026, 9, 30, 19, 0)}
	for _, table := range []string{"reminders", "health_items", "reha_plans", "requests", "week_slots", "blanket_states", "presence"} {
		e.exec(`DELETE FROM ` + table)
	}
	deps := httpapi.Deps{Pool: pool, Now: e.clock, Notify: push.NewNotifier(pool, e.fake, nil)}
	e.h = httpapi.NewHandler(deps)
	e.svc = reminders.NewService(deps)
	return e
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("%v\n%s", err, sql)
	}
}

func (e *env) scan(sql string, args ...any) string {
	e.t.Helper()
	var s string
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		e.t.Fatalf("%v\n%s", err, sql)
	}
	return s
}

func (e *env) count(sql string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatalf("%v\n%s", err, sql)
	}
	return n
}

// device gives the user a push token and the push consent.
func (e *env) device(userID string) {
	e.t.Helper()
	if err := push.RegisterToken(context.Background(), e.pool, seed.StableB, userID, "tok-"+userID, push.PlatformAndroid); err != nil {
		e.t.Fatal(err)
	}
	pushtest.GrantConsent(e.t, e.pool, userID)
}

type resp struct {
	*httptest.ResponseRecorder
	t *testing.T
}

func (e *env) do(user, method, path string, body any) resp {
	e.t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		req.Header.Set("Authorization", "Bearer "+authtest.TokenAt(e.t, e.pool, user, e.clock()))
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return resp{rec, e.t}
}

func (r resp) status(want int) resp {
	r.t.Helper()
	if r.Code != want {
		r.t.Fatalf("status = %d, want %d, body: %s", r.Code, want, r.Body.String())
	}
	return r
}

func (r resp) errCode(status int, code string) {
	r.t.Helper()
	r.status(status)
	var b struct{ Error struct{ Code string } }
	if err := json.Unmarshal(r.Body.Bytes(), &b); err != nil || b.Error.Code != code {
		r.t.Fatalf("error code = %q (%v), want %q, body: %s", b.Error.Code, err, code, r.Body.String())
	}
}

func (r resp) into(v any) {
	r.t.Helper()
	if err := json.Unmarshal(r.Body.Bytes(), v); err != nil {
		r.t.Fatalf("decode: %v: %s", err, r.Body.String())
	}
}

type m = map[string]any

// list fetches GET /reminders and returns the titles of every group, plus the response.
func (e *env) list(user, query string) (reminders.Response, map[string][]reminders.Item) {
	e.t.Helper()
	var out reminders.Response
	e.do(user, "GET", "/api/v1/reminders"+query, nil).status(200).into(&out)
	groups := map[string][]reminders.Item{}
	for _, g := range out.Groups {
		groups[g.Key] = g.Items
	}
	return out, groups
}

func titles(items []reminders.Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Title
	}
	return out
}

func find(items []reminders.Item, title string) (reminders.Item, bool) {
	for _, it := range items {
		if it.Title == title {
			return it, true
		}
	}
	return reminders.Item{}, false
}
