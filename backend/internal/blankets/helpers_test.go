package blankets_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/blankets"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push/pushtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

var berlin = func() *time.Location {
	l, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	return l
}()

// env is a seeded database, the router and a fake clock (Wed 2026-09-30 19:00 Berlin).
type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
	svc  *blankets.Service
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

// berlinAt builds a Berlin wall-clock time.
func berlinAt(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, berlin)
}

var allUsers = []string{seed.UserJan, seed.UserAnna, seed.UserJonas, seed.UserTom, seed.UserSarah, seed.UserKai, seed.UserMia, seed.UserLea}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := dbtest.NewSeeded(t)
	e := &env{t: t, pool: pool, fake: &push.Fake{}, now: berlinAt(2026, 9, 30, 19, 0)}
	deps := httpapi.Deps{Pool: pool, Now: e.clock, Notify: push.NewNotifier(pool, e.fake, nil)}
	e.h = httpapi.NewHandler(deps)
	e.svc = blankets.NewService(deps)
	// Every seed user has one device so pushes can be observed (token "tok-<id>").
	for _, id := range allUsers {
		if err := push.RegisterToken(context.Background(), pool, seed.StableB, id, "tok-"+id, push.PlatformAndroid); err != nil {
			t.Fatal(err)
		}
		pushtest.GrantConsent(t, pool, id)
	}
	return e
}

type resp struct{ *httptest.ResponseRecorder }

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
	return resp{rec}
}

func (r resp) status(t *testing.T, want int) resp {
	t.Helper()
	if r.Code != want {
		t.Fatalf("status = %d, want %d, body: %s", r.Code, want, r.Body.String())
	}
	return r
}

func (r resp) errCode(t *testing.T, status int, code string) {
	t.Helper()
	r.status(t, status)
	var b struct{ Error struct{ Code string } }
	if err := json.Unmarshal(r.Body.Bytes(), &b); err != nil || b.Error.Code != code {
		t.Fatalf("error code = %q (%v), want %q, body: %s", b.Error.Code, err, code, r.Body.String())
	}
}

func (r resp) into(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body.Bytes(), v); err != nil {
		t.Fatalf("decode: %v: %s", err, r.Body.String())
	}
}

type m = map[string]any

// snapshot stores a weather snapshot for the night of day.
func (e *env) snapshot(day string, fetched time.Time, minC float64, rain bool) {
	e.t.Helper()
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO weather_snapshots (stable_id, fetched_at, valid_for, night_min_c, rain_probability, rain_mm, wind_kmh, will_rain, raw)
		VALUES ($1, $2, $3::date, $4, 60, 1.5, 12, $5, '{}')`, seed.StableB, fetched, day, minC, rain)
	if err != nil {
		e.t.Fatal(err)
	}
}

// exec runs SQL against the test database.
func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatal(err)
	}
}

// present opens a presence visit for the user.
func (e *env) present(user string, since time.Time) {
	e.t.Helper()
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at) VALUES ($1, $2, $3)`, seed.StableB, user, since)
}

func (e *env) leave(user string, at time.Time) {
	e.t.Helper()
	e.exec(`UPDATE presence SET left_at = $2 WHERE user_id = $1 AND left_at IS NULL`, user, at)
}

// setState records a state row directly (all horses done etc.).
func (e *env) setState(horse, day, action string, changedAt time.Time) {
	e.t.Helper()
	e.exec(`INSERT INTO blanket_states (stable_id, horse_id, day, action, changed_at, changed_by) VALUES ($1, $2, $3::date, $4, $5, $6)`,
		seed.StableB, horse, day, action, changedAt, seed.UserMia)
}

func (e *env) allHorses() []string {
	return []string{seed.HorseLuna, seed.HorseFanta, seed.HorseBalu, seed.HorseCookie, seed.HorseMerlin, seed.HorsePepe, seed.HorseNala}
}

func sentTo(fake *push.Fake, userID string) []push.Message {
	var out []push.Message
	for _, msg := range fake.Sent() {
		if msg.To == "tok-"+userID {
			out = append(out, msg)
		}
	}
	return out
}

func recipients(fake *push.Fake) []string {
	var out []string
	for _, msg := range fake.Sent() {
		out = append(out, strings.TrimPrefix(msg.To, "tok-"))
	}
	return out
}

// Decoded response shapes (only the fields the tests look at).
type blanketJSON struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	FillG     int     `json:"fill_g"`
	Color     *string `json:"color"`
	Location  *string `json:"location"`
	PhotoPath *string `json:"photo_path"`
	PhotoURL  *string `json:"photo_url"`
}

type recJSON struct {
	Status    string       `json:"status"`
	RuleIndex *int         `json:"rule_index"`
	Blanket   *blanketJSON `json:"blanket"`
	Note      string       `json:"note"`
}

type stateJSON struct {
	ID          string  `json:"id"`
	Day         string  `json:"day"`
	Action      string  `json:"action"`
	CoveredWith *string `json:"covered_with"`
	ChangedBy   *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"changed_by"`
}

type todayJSON struct {
	Day          string `json:"day"`
	ReminderTime string `json:"reminder_time"`
	Weather      *struct {
		NightMinC float64 `json:"night_min_c"`
		WillRain  bool    `json:"will_rain"`
	} `json:"weather"`
	Progress struct {
		Done  int `json:"done"`
		Total int `json:"total"`
	} `json:"progress"`
	Horses []struct {
		Horse struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"horse"`
		Recommendation recJSON    `json:"recommendation"`
		State          *stateJSON `json:"state"`
		Done           bool       `json:"done"`
	} `json:"horses"`
}

func (e *env) today(user string) todayJSON {
	e.t.Helper()
	var out todayJSON
	e.do(user, "GET", "/api/v1/blankets/today", nil).status(e.t, 200).into(e.t, &out)
	return out
}
