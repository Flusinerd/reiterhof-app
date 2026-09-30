package trainingapi_test

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

// Seed ids of the exercise library (see internal/seed/training.go).
const (
	exUebergaenge    = "00000000-0000-4000-8000-000000000808"
	exSchulterherein = "00000000-0000-4000-8000-000000000810"
	exTravers        = "00000000-0000-4000-8000-000000000811"
)

// wed is the fixed "now" of most tests: Wednesday 2026-03-25 10:00 Berlin (CET).
var berlin = mustLoc("Europe/Berlin")
var wed = time.Date(2026, 3, 25, 10, 0, 0, 0, berlin)

func mustLoc(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
	now  time.Time
}

// newEnv creates a seeded database with a fixed clock. Sessions and week slots of the seed
// (relative to the real day of seeding) are removed so tests start from an empty history.
func newEnv(t *testing.T) *env {
	t.Helper()
	return newEnvAt(t, wed)
}

func newEnvAt(t *testing.T, now time.Time) *env {
	t.Helper()
	pool := dbtest.NewSeeded(t)
	for _, q := range []string{`DELETE FROM sessions`, `DELETE FROM week_slots`, `DELETE FROM weather_snapshots`} {
		if _, err := pool.Exec(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	h := httpapi.NewHandler(httpapi.Deps{Pool: pool, Now: func() time.Time { return now }})
	return &env{t: t, pool: pool, h: h, now: now}
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("exec %q: %v", sql, err)
	}
}

// do sends a request as the given user ("" = anonymous) and returns the recorder.
func (e *env) do(user, method, path, body string) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != "" {
		req.Header.Set("Authorization", "Bearer "+authtest.TokenAt(e.t, e.pool, user, e.now))
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

// call sends the request, checks the status and decodes the JSON body into a map.
func (e *env) call(user, method, path, body string, want int) map[string]any {
	e.t.Helper()
	rec := e.do(user, method, path, body)
	if rec.Code != want {
		e.t.Fatalf("%s %s as %s: status = %d, want %d, body %s", method, path, short(user), rec.Code, want, rec.Body.String())
	}
	var out map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			e.t.Fatalf("%s %s: decode: %v (%s)", method, path, err, rec.Body.String())
		}
	}
	return out
}

func (e *env) errCode(user, method, path, body string, want int) string {
	e.t.Helper()
	out := e.call(user, method, path, body, want)
	if errObj, ok := out["error"].(map[string]any); ok {
		code, _ := errObj["code"].(string)
		return code
	}
	return ""
}

func short(user string) string {
	if user == "" {
		return "anonymous"
	}
	return user[len(user)-3:]
}

func list(v any) []any {
	l, _ := v.([]any)
	return l
}

func obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func num(v any) float64 {
	f, _ := v.(float64)
	return f
}

const (
	luna  = seed.HorseLuna
	fanta = seed.HorseFanta
)
