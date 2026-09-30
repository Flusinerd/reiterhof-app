package requests_test

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
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push/pushtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/requests"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// env is a seeded database, the router and a fake clock (Wed 2026-09-30 10:00 Berlin).
type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
	svc  *requests.Service
	fake *push.Fake
	now  time.Time
	mu   sync.Mutex
}

func (e *env) clock() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.now
}

func (e *env) setNow(t time.Time) {
	e.mu.Lock()
	e.now = t
	e.mu.Unlock()
}

var berlin = func() *time.Location {
	l, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	return l
}()

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := dbtest.NewSeeded(t)
	e := &env{t: t, pool: pool, fake: &push.Fake{}, now: time.Date(2026, 9, 30, 10, 0, 0, 0, berlin)}
	deps := httpapi.Deps{Pool: pool, Now: e.clock, Notify: push.NewNotifier(pool, e.fake, nil)}
	e.h = httpapi.NewHandler(deps)
	e.svc = requests.NewService(httpapi.Deps{Pool: pool, Now: e.clock, Notify: deps.Notify})
	// Every seed user has one device so pushes can be observed (token "tok-<id>").
	for _, id := range []string{seed.UserJan, seed.UserAnna, seed.UserJonas, seed.UserTom, seed.UserSarah, seed.UserKai, seed.UserMia, seed.UserLea} {
		if err := push.RegisterToken(context.Background(), pool, seed.StableB, id, "tok-"+id, push.PlatformAndroid); err != nil {
			t.Fatal(err)
		}
		pushtest.GrantConsent(t, pool, id)
	}
	return e
}

type resp struct {
	*httptest.ResponseRecorder
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

func (r resp) req(t *testing.T) requests.Request {
	t.Helper()
	var out requests.Request
	if err := json.Unmarshal(r.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode request: %v: %s", err, r.Body.String())
	}
	return out
}

func (r resp) list(t *testing.T) ([]requests.Request, int) {
	t.Helper()
	var out struct {
		Requests  []requests.Request `json:"requests"`
		OpenCount int                `json:"open_count"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode list: %v: %s", err, r.Body.String())
	}
	return out.Requests, out.OpenCount
}

func ids(rs []requests.Request) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.ID
	}
	return out
}

type m = map[string]any

func showHelperBody(needed int) m {
	return m{
		"type": "show_helper", "horse_id": seed.HorseFanta, "date": "2026-10-02", "time_from": "08:30",
		"location": "Reitanlage Haltern", "helpers_needed": needed,
		"payload": m{
			"show_name": "Herbstturnier", "ride_along": true,
			"classes": []m{{"name": "E-Dressur", "time": "09:30"}},
			"tasks":   []string{"hold_horse", "film"},
		},
	}
}

func (e *env) create(user string, body m) requests.Request {
	e.t.Helper()
	return e.do(user, "POST", "/api/v1/requests", body).status(e.t, 201).req(e.t)
}

func sentTo(fake *push.Fake, userID string) []push.Message {
	var out []push.Message
	for _, m := range fake.Sent() {
		if m.To == "tok-"+userID {
			out = append(out, m)
		}
	}
	return out
}

func TestAcceptFlowShowHelper(t *testing.T) {
	e := newEnv(t)
	r := e.create(seed.UserAnna, showHelperBody(2))
	if r.Status != "open" || r.HelpersNeeded != 2 || r.CreatorName != "Anna" || !r.IsCreator {
		t.Fatalf("created = %+v", r)
	}
	if len(r.Tasks) != 2 || r.Tasks[0] != "hold_horse" {
		t.Errorf("tasks = %v", r.Tasks)
	}
	path := "/api/v1/requests/" + r.ID

	// The creator cannot accept.
	e.do(seed.UserAnna, "POST", path+"/accept", nil).errCode(t, 403, "own_request")

	got := e.do(seed.UserJan, "POST", path+"/accept", nil).status(t, 200).req(t)
	if got.Status != "open" || got.HelpersCount != 1 || !got.IsHelper || got.SpotsLeft != 1 || got.CanAccept {
		t.Errorf("after jan = %+v", got)
	}
	// Idempotent.
	got = e.do(seed.UserJan, "POST", path+"/accept", nil).status(t, 200).req(t)
	if got.HelpersCount != 1 {
		t.Errorf("second accept added a helper: %+v", got)
	}
	got = e.do(seed.UserTom, "POST", path+"/accept", nil).status(t, 200).req(t)
	if got.Status != "assigned" || got.HelpersCount != 2 {
		t.Errorf("after tom = %+v", got)
	}
	// Full.
	e.do(seed.UserKai, "POST", path+"/accept", nil).errCode(t, 409, "request_full")
	// Tom is idempotent even though it is full now.
	e.do(seed.UserTom, "POST", path+"/accept", nil).status(t, 200)

	// Kai sees it as not acceptable.
	kai := e.do(seed.UserKai, "GET", path, nil).status(t, 200).req(t)
	if kai.CanAccept || kai.IsHelper || kai.IsCreator || len(kai.Helpers) != 2 || kai.Helpers[0].Name != "Jan" {
		t.Errorf("kai view = %+v", kai)
	}

	// Anna is told about both helpers with kind helper.
	msgs := sentTo(e.fake, seed.UserAnna)
	if len(msgs) != 2 || msgs[0].Data["kind"] != "helper" || !strings.Contains(msgs[0].Title, "Jan hilft mit") || msgs[0].Data["request_id"] != r.ID {
		t.Errorf("anna pushes = %+v", msgs)
	}

	// Withdraw: back to open, then Kai takes the seat.
	got = e.do(seed.UserTom, "POST", path+"/withdraw", nil).status(t, 200).req(t)
	if got.Status != "open" || got.HelpersCount != 1 {
		t.Errorf("after withdraw = %+v", got)
	}
	e.do(seed.UserTom, "POST", path+"/withdraw", nil).status(t, 200) // idempotent
	got = e.do(seed.UserKai, "POST", path+"/accept", nil).status(t, 200).req(t)
	if got.Status != "assigned" {
		t.Errorf("after kai = %+v", got)
	}

	// Done by a helper; afterwards accept and withdraw are refused.
	got = e.do(seed.UserJan, "POST", path+"/done", nil).status(t, 200).req(t)
	if got.Status != "done" {
		t.Errorf("done = %+v", got)
	}
	e.do(seed.UserJan, "POST", path+"/withdraw", nil).errCode(t, 409, "not_open")
	e.do(seed.UserMia, "POST", path+"/accept", nil).errCode(t, 409, "not_open")
	e.do(seed.UserJan, "POST", path+"/done", nil).status(t, 200) // idempotent
}

func TestAcceptRaceForLastSeat(t *testing.T) {
	e := newEnv(t)
	r := e.create(seed.UserAnna, showHelperBody(1))
	racers := []string{seed.UserJan, seed.UserTom, seed.UserKai, seed.UserMia, seed.UserLea, seed.UserJonas, seed.UserSarah}
	codes := make([]int, len(racers))
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i, u := range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			codes[i] = e.do(u, "POST", "/api/v1/requests/"+r.ID+"/accept", nil).Code
		}()
	}
	close(start)
	wg.Wait()
	ok, full := 0, 0
	for _, c := range codes {
		switch c {
		case 200:
			ok++
		case 409:
			full++
		default:
			t.Errorf("unexpected status %d", c)
		}
	}
	if ok != 1 || full != len(racers)-1 {
		t.Fatalf("winners = %d, rejected = %d (codes %v)", ok, full, codes)
	}
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM request_assignees WHERE request_id = $1`, r.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("assignees = %d (%v)", n, err)
	}
}

func TestRideShareSeats(t *testing.T) {
	e := newEnv(t)
	r := e.create(seed.UserMia, m{
		"type": "ride_share", "date": "2026-10-03", "location": "Hof Ahlers", "helpers_needed": 9,
		"payload": m{"destination": "Turnier Haltern", "departure_time": "07:15", "seats_free": 2},
	})
	// One seat is one helper slot; the departure time becomes the start time.
	if r.HelpersNeeded != 2 || r.TimeFrom == nil || *r.TimeFrom != "07:15" {
		t.Fatalf("ride share = %+v", r)
	}
	e.do(seed.UserJan, "POST", "/api/v1/requests/"+r.ID+"/accept", nil).status(t, 200)
	e.do(seed.UserTom, "POST", "/api/v1/requests/"+r.ID+"/accept", nil).status(t, 200)
	e.do(seed.UserKai, "POST", "/api/v1/requests/"+r.ID+"/accept", nil).errCode(t, 409, "request_full")
	// Fewer seats than passengers is refused; more seats reopens the request.
	e.do(seed.UserMia, "PATCH", "/api/v1/requests/"+r.ID, m{"payload": m{"destination": "Turnier Haltern", "departure_time": "07:15", "seats_free": 1}}).errCode(t, 409, "too_many_helpers")
	got := e.do(seed.UserMia, "PATCH", "/api/v1/requests/"+r.ID, m{"payload": m{"destination": "Turnier Haltern", "departure_time": "07:15", "seats_free": 3}}).status(t, 200).req(t)
	if got.Status != "open" || got.HelpersNeeded != 3 {
		t.Errorf("after more seats = %+v", got)
	}
}

func TestPermissions(t *testing.T) {
	e := newEnv(t)
	r := e.create(seed.UserAnna, m{"type": "other", "date": "2026-10-02", "description": "Heu holen", "helpers_needed": 2})
	path := "/api/v1/requests/" + r.ID

	// Unauthenticated and stable-less callers.
	e.do("", "GET", "/api/v1/requests", nil).status(t, 401)
	e.do(seed.UserKai, "PATCH", path, m{"location": "x"}).errCode(t, 403, "forbidden")
	e.do(seed.UserKai, "POST", path+"/cancel", nil).errCode(t, 403, "forbidden")
	e.do(seed.UserKai, "POST", path+"/done", nil).errCode(t, 403, "forbidden")
	e.do(seed.UserAnna, "POST", path+"/thanks/"+seed.UserKai, nil).errCode(t, 404, "not_found") // Kai is no helper

	e.do(seed.UserJan, "POST", path+"/accept", nil).status(t, 200)
	// Only the creator may say thanks.
	e.do(seed.UserJan, "POST", path+"/thanks/"+seed.UserJan, nil).errCode(t, 403, "forbidden")
	e.do(seed.UserAnna, "POST", path+"/thanks/"+seed.UserJan, nil).status(t, 204)
	e.do(seed.UserAnna, "POST", path+"/thanks/"+seed.UserJan, nil).status(t, 204) // idempotent

	// Creator updates; type cannot change.
	got := e.do(seed.UserAnna, "PATCH", path, m{"location": "Scheune"}).status(t, 200).req(t)
	if got.Location != "Scheune" {
		t.Errorf("location = %q", got.Location)
	}
	e.do(seed.UserAnna, "PATCH", path, m{"type": "exercise"}).errCode(t, 400, "invalid_json")
	e.do(seed.UserAnna, "PATCH", path, m{"helpers_needed": 0}).status(t, 400)
	e.do(seed.UserAnna, "PATCH", path, m{"date": "2026-09-01"}).errCode(t, 400, "validation_failed")

	// An admin (Jan) may cancel someone else's request; the helper hears about it.
	e.fake.Reset()
	e.do(seed.UserTom, "POST", path+"/accept", nil).status(t, 200)
	e.fake.Reset()
	got = e.do(seed.UserJan, "POST", path+"/cancel", nil).status(t, 200).req(t)
	if got.Status != "cancelled" {
		t.Errorf("cancel = %+v", got)
	}
	if msgs := sentTo(e.fake, seed.UserTom); len(msgs) != 1 || !strings.HasPrefix(msgs[0].Title, "Anfrage abgesagt") || msgs[0].Data["kind"] != "helper" {
		t.Errorf("tom pushes = %+v", msgs)
	}
	e.do(seed.UserAnna, "PATCH", path, m{"location": "x"}).errCode(t, 409, "not_open")
	e.do(seed.UserKai, "POST", path+"/accept", nil).errCode(t, 409, "not_open")
	e.do(seed.UserAnna, "POST", path+"/done", nil).errCode(t, 409, "not_open")
	e.do(seed.UserAnna, "POST", path+"/cancel", nil).status(t, 200) // idempotent

	// Thank-you counter is private to the helper.
	var c struct{ Count int }
	if err := json.Unmarshal(e.do(seed.UserJan, "GET", "/api/v1/me/thanks", nil).status(t, 200).Body.Bytes(), &c); err != nil || c.Count != 1 {
		t.Errorf("jan thanks = %+v (%v)", c, err)
	}
	if err := json.Unmarshal(e.do(seed.UserTom, "GET", "/api/v1/me/thanks", nil).status(t, 200).Body.Bytes(), &c); err != nil || c.Count != 0 {
		t.Errorf("tom thanks = %+v (%v)", c, err)
	}
}

func TestCreateValidation(t *testing.T) {
	e := newEnv(t)
	cases := map[string]m{
		"unknown type":       {"type": "party", "date": "2026-10-02"},
		"no date":            {"type": "other"},
		"past date":          {"type": "other", "date": "2026-09-29"},
		"end before start":   {"type": "other", "date": "2026-10-02", "date_end": "2026-10-01"},
		"bad time":           {"type": "other", "date": "2026-10-02", "time_from": "8 Uhr"},
		"time_to too early":  {"type": "other", "date": "2026-10-02", "time_from": "10:00", "time_to": "09:00"},
		"other with payload": {"type": "other", "date": "2026-10-02", "payload": m{"x": 1}},
		"exercise no horse":  {"type": "exercise", "date": "2026-10-02", "payload": m{"mode": "lunge"}},
		"exercise bad mode":  {"type": "exercise", "date": "2026-10-02", "horse_id": seed.HorseLuna, "payload": m{"mode": "fly"}},
		"show no name":       {"type": "show_helper", "date": "2026-10-02", "payload": m{"tasks": []string{"film"}}},
		"show bad task":      {"type": "show_helper", "date": "2026-10-02", "payload": m{"show_name": "x", "tasks": []string{"nap"}}},
		"ride no seats":      {"type": "ride_share", "date": "2026-10-02", "payload": m{"destination": "x", "departure_time": "07:00"}},
		"unknown horse":      {"type": "other", "date": "2026-10-02", "horse_id": "00000000-0000-4000-8000-0000000009ff"},
		"malformed horse":    {"type": "other", "date": "2026-10-02", "horse_id": "luna"},
		"too many helpers":   {"type": "other", "date": "2026-10-02", "helpers_needed": 50},
		"bad rule":           {"type": "other", "date": "2026-10-02", "recurring_rule": "FREQ=MONTHLY"},
		"rule with range":    {"type": "other", "date": "2026-10-02", "date_end": "2026-10-03", "recurring_rule": "FREQ=DAILY"},
		"appointment bad":    {"type": "appointment_companion", "date": "2026-10-02", "horse_id": seed.HorseLuna, "payload": m{"with": "baker"}},
		"feed bad":           {"type": "feed_or_turnout", "date": "2026-10-02", "horse_id": seed.HorseLuna, "payload": m{"what": "nap"}},
	}
	for name, body := range cases {
		r := e.do(seed.UserAnna, "POST", "/api/v1/requests", body)
		if r.Code != 400 {
			t.Errorf("%s: status %d, body %s", name, r.Code, r.Body.String())
		}
	}
	e.do(seed.UserAnna, "POST", "/api/v1/requests", m{"type": "other", "date": "2026-10-02", "bogus": 1}).errCode(t, 400, "invalid_json")

	// Valid ones of the remaining types, including a date range and the generic blanket type.
	e.create(seed.UserAnna, m{"type": "exercise", "date": "2026-10-02", "horse_id": seed.HorseLuna, "tasks": []string{" Trense putzen ", ""}, "payload": m{"mode": "lunge", "rules_note": "nur Schritt"}})
	f := e.create(seed.UserAnna, m{"type": "feed_or_turnout", "date": "2026-10-05", "date_end": "2026-10-09", "horse_id": seed.HorseFanta, "payload": m{"what": "turnout"}})
	if f.DateEnd == nil || *f.DateEnd != "2026-10-09" {
		t.Errorf("range = %+v", f)
	}
	e.create(seed.UserAnna, m{"type": "appointment_companion", "date": "2026-10-06", "time_from": "14:00", "horse_id": seed.HorseBalu, "payload": m{"with": "farrier", "note": "Eisen neu"}})
	e.create(seed.UserAnna, m{"type": "blanket", "date": "2026-10-06", "horse_id": seed.HorseBalu, "payload": m{"blanket": "Decke 150 g"}})
	x := e.create(seed.UserAnna, m{"type": "exercise", "date": "2026-10-07", "horse_id": seed.HorseLuna, "tasks": []string{" Trense putzen ", ""}, "payload": m{"mode": "ride"}})
	if len(x.Tasks) != 1 || x.Tasks[0] != "Trense putzen" {
		t.Errorf("tasks = %v", x.Tasks)
	}
}

func TestListFiltersAndCounts(t *testing.T) {
	e := newEnv(t)
	a := e.create(seed.UserAnna, m{"type": "other", "date": "2026-10-04", "description": "A"})
	b := e.create(seed.UserJan, m{"type": "exercise", "date": "2026-10-02", "horse_id": seed.HorseLuna, "payload": m{"mode": "lunge"}})
	c := e.create(seed.UserJan, m{"type": "other", "date": "2026-10-03", "time_from": "09:00"})
	e.do(seed.UserTom, "POST", "/api/v1/requests/"+a.ID+"/accept", nil).status(t, 200)
	e.do(seed.UserTom, "POST", "/api/v1/requests/"+c.ID+"/accept", nil).status(t, 200)
	e.do(seed.UserJan, "POST", "/api/v1/requests/"+c.ID+"/done", nil).status(t, 200)
	d := e.create(seed.UserJan, m{"type": "other", "date": "2026-10-06"})
	e.do(seed.UserJan, "POST", "/api/v1/requests/"+d.ID+"/cancel", nil).status(t, 200)

	get := func(user, q string) ([]string, int) {
		l, open := e.do(user, "GET", "/api/v1/requests"+q, nil).status(t, 200).list(t)
		return ids(l), open
	}
	eq := func(name string, got, want []string) {
		t.Helper()
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	// Default: everything but cancelled, by date. open_count counts requests still looking for help
	// (a is full, c is done, d is cancelled).
	got, open := get(seed.UserKai, "")
	eq("default", got, []string{b.ID, c.ID, a.ID})
	if open != 1 {
		t.Errorf("open_count = %d, want 1", open)
	}
	got, _ = get(seed.UserKai, "?status=open")
	eq("open", got, []string{b.ID})
	got, _ = get(seed.UserKai, "?status=open,assigned")
	eq("open,assigned", got, []string{b.ID, a.ID})
	got, _ = get(seed.UserKai, "?status=done")
	eq("done", got, []string{c.ID})
	got, _ = get(seed.UserKai, "?status=cancelled")
	eq("cancelled", got, []string{d.ID})
	got, _ = get(seed.UserJan, "?mine=true")
	eq("mine", got, []string{b.ID, c.ID})
	got, _ = get(seed.UserTom, "?assigned=true")
	eq("assigned", got, []string{c.ID, a.ID})
	got, _ = get(seed.UserKai, "?type=exercise")
	eq("type", got, []string{b.ID})
	got, _ = get(seed.UserKai, "?from=2026-10-03&to=2026-10-03")
	eq("range", got, []string{c.ID})
	got, _ = get(seed.UserKai, "?horse_id="+seed.HorseLuna)
	eq("horse", got, []string{b.ID})
	got, _ = get(seed.UserKai, "?limit=1")
	eq("limit", got, []string{b.ID})
	for _, q := range []string{"?status=weird", "?type=weird", "?from=yesterday", "?limit=0", "?horse_id=x"} {
		e.do(seed.UserKai, "GET", "/api/v1/requests"+q, nil).status(t, 400)
	}
}

func TestCrossStableIsolation(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	const stable2, outsider, horse2 = "00000000-0000-4000-8000-000000000199", "00000000-0000-4000-8000-000000000299", "00000000-0000-4000-8000-000000000399"
	for _, sql := range []string{
		`INSERT INTO stables (id, name) VALUES ('` + stable2 + `', 'Stall C')`,
		`INSERT INTO users (id, stable_id, name, email) VALUES ('` + outsider + `', '` + stable2 + `', 'Otto', 'otto@example.org')`,
		`INSERT INTO horses (id, stable_id, name) VALUES ('` + horse2 + `', '` + stable2 + `', 'Blitz')`,
	} {
		if _, err := e.pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	r := e.create(seed.UserAnna, showHelperBody(2))
	path := "/api/v1/requests/" + r.ID

	e.do(outsider, "GET", path, nil).errCode(t, 404, "not_found")
	e.do(outsider, "GET", path+".ics", nil).errCode(t, 404, "not_found")
	for _, p := range []string{"/accept", "/withdraw", "/done", "/cancel"} {
		e.do(outsider, "POST", path+p, nil).errCode(t, 404, "not_found")
	}
	e.do(outsider, "PATCH", path, m{"location": "x"}).errCode(t, 404, "not_found")
	e.do(outsider, "POST", path+"/thanks/"+seed.UserJan, nil).errCode(t, 404, "not_found")
	if l, open := e.do(outsider, "GET", "/api/v1/requests", nil).status(t, 200).list(t); len(l) != 0 || open != 0 {
		t.Errorf("outsider sees %d requests (open %d)", len(l), open)
	}
	// A horse of another stable cannot be referenced.
	e.do(seed.UserAnna, "POST", "/api/v1/requests", m{"type": "other", "date": "2026-10-02", "horse_id": horse2}).errCode(t, 400, "validation_failed")
	// Options only list the own stable.
	var o requests.Options
	if err := json.Unmarshal(e.do(outsider, "GET", "/api/v1/requests/options", nil).status(t, 200).Body.Bytes(), &o); err != nil {
		t.Fatal(err)
	}
	if len(o.Horses) != 1 || o.Horses[0].Name != "Blitz" || len(o.Members) != 1 {
		t.Errorf("options = %+v", o)
	}
	// New requests never push across stables.
	e.fake.Reset()
	e.create(outsider, m{"type": "other", "date": "2026-10-02"})
	if len(e.fake.Sent()) != 0 {
		t.Errorf("cross stable pushes: %+v", e.fake.Sent())
	}
}

func TestNewRequestPushIsOptIn(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	// Tom opts in via the API, Jan via SQL and then out again, Kai never did.
	e.do(seed.UserTom, "PATCH", "/api/v1/requests/notify-settings", m{"new_request": true}).status(t, 200)
	if _, err := e.pool.Exec(ctx, `INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1, $2, 'new_request', false)`, seed.StableB, seed.UserJan); err != nil {
		t.Fatal(err)
	}
	e.do(seed.UserAnna, "PATCH", "/api/v1/requests/notify-settings", m{"new_request": true}).status(t, 200) // the creator never gets her own
	var s map[string]bool
	if err := json.Unmarshal(e.do(seed.UserTom, "GET", "/api/v1/requests/notify-settings", nil).status(t, 200).Body.Bytes(), &s); err != nil || !s["new_request"] {
		t.Errorf("tom settings = %v (%v)", s, err)
	}
	if err := json.Unmarshal(e.do(seed.UserKai, "GET", "/api/v1/requests/notify-settings", nil).status(t, 200).Body.Bytes(), &s); err != nil || s["new_request"] {
		t.Errorf("kai settings = %v (%v)", s, err)
	}
	e.do(seed.UserKai, "PATCH", "/api/v1/requests/notify-settings", m{}).status(t, 400)

	r := e.create(seed.UserAnna, showHelperBody(1))
	sent := e.fake.Sent()
	if len(sent) != 1 || sent[0].To != "tok-"+seed.UserTom {
		t.Fatalf("pushes = %+v, want only tom", sent)
	}
	if sent[0].Data["kind"] != "new_request" || sent[0].Data["request_id"] != r.ID ||
		!strings.HasPrefix(sent[0].Title, "Neue Anfrage: Turniertrottel") || !strings.Contains(sent[0].Body, "Anna sucht Hilfe · Fr, 2. Okt · 08:30") {
		t.Errorf("push = %+v", sent[0])
	}
}

func TestRecurringSeries(t *testing.T) {
	e := newEnv(t)
	body := m{"type": "feed_or_turnout", "date": "2026-09-30", "time_from": "17:00", "horse_id": seed.HorseFanta,
		"location": "Koppel 2", "recurring_rule": "freq=weekly;byday=we", "payload": m{"what": "bring_in"}}
	head := e.create(seed.UserAnna, body)
	if head.SeriesID == nil || *head.SeriesID != head.ID || head.RecurringRule == nil || *head.RecurringRule != "FREQ=WEEKLY;BYDAY=WE" {
		t.Fatalf("head = %+v", head)
	}
	dates := func() string {
		rows, err := e.pool.Query(context.Background(),
			`SELECT to_char(date, 'YYYY-MM-DD') || ':' || status FROM requests WHERE series_id = $1 ORDER BY date`, head.ID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			_ = rows.Scan(&s)
			out = append(out, s)
		}
		return strings.Join(out, " ")
	}
	if got, want := dates(), "2026-09-30:open 2026-10-07:open 2026-10-14:open"; got != want {
		t.Fatalf("occurrences = %q, want %q", got, want)
	}
	// No pushes for materialised occurrences (only the creation itself notifies).
	// The job is idempotent and extends the window as time passes.
	if err := e.svc.MaterializeAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got, want := dates(), "2026-09-30:open 2026-10-07:open 2026-10-14:open"; got != want {
		t.Fatalf("after rerun = %q", got)
	}
	e.setNow(time.Date(2026, 10, 8, 3, 30, 0, 0, berlin))
	if err := e.svc.MaterializeAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got, want := dates(), "2026-09-30:open 2026-10-07:open 2026-10-14:open 2026-10-21:open"; got != want {
		t.Fatalf("after a week = %q, want %q", got, want)
	}
	var second requests.Request
	{
		l, _ := e.do(seed.UserKai, "GET", "/api/v1/requests?from=2026-10-14&to=2026-10-14", nil).status(t, 200).list(t)
		if len(l) != 1 {
			t.Fatalf("oct 14 = %v", l)
		}
		second = l[0]
	}
	if second.Location != "Koppel 2" || second.TimeFrom == nil || *second.TimeFrom != "17:00" || second.SeriesID == nil || *second.SeriesID != head.ID || second.CreatedBy != seed.UserAnna {
		t.Errorf("occurrence = %+v", second)
	}

	// Edit the series: new time and place reach the upcoming occurrences, a helper keeps the seat.
	e.do(seed.UserJan, "POST", "/api/v1/requests/"+second.ID+"/accept", nil).status(t, 200)
	e.do(seed.UserAnna, "PATCH", "/api/v1/requests/"+second.ID, m{"scope": "series", "time_from": "16:30", "location": "Koppel 3"}).status(t, 200)
	var loc, tm string
	if err := e.pool.QueryRow(context.Background(), `SELECT location, to_char(time_from, 'HH24:MI') FROM requests WHERE date = '2026-10-21' AND series_id = $1`, head.ID).Scan(&loc, &tm); err != nil || loc != "Koppel 3" || tm != "16:30" {
		t.Errorf("oct 21 = %q %q (%v)", loc, tm, err)
	}
	// A single-request edit does not touch the others, and dates/rules need the right scope.
	e.do(seed.UserAnna, "PATCH", "/api/v1/requests/"+second.ID, m{"location": "Stall"}).status(t, 200)
	e.do(seed.UserAnna, "PATCH", "/api/v1/requests/"+second.ID, m{"recurring_rule": "FREQ=DAILY"}).errCode(t, 400, "validation_failed")
	e.do(seed.UserAnna, "PATCH", "/api/v1/requests/"+second.ID, m{"scope": "series", "date": "2026-10-15"}).errCode(t, 400, "validation_failed")
	if err := e.pool.QueryRow(context.Background(), `SELECT location FROM requests WHERE date = '2026-10-21' AND series_id = $1`, head.ID).Scan(&loc); err != nil || loc != "Koppel 3" {
		t.Errorf("oct 21 location = %q (%v)", loc, err)
	}
	// Moving one occurrence onto another one of the series is a conflict.
	e.do(seed.UserAnna, "PATCH", "/api/v1/requests/"+second.ID, m{"date": "2026-10-21"}).errCode(t, 409, "conflict")

	// Switching to a daily rhythm drops unassigned upcoming occurrences and recreates them daily; Jan's stays.
	e.do(seed.UserAnna, "PATCH", "/api/v1/requests/"+second.ID, m{"scope": "series", "recurring_rule": "FREQ=DAILY;UNTIL=20261011"}).status(t, 200)
	if got, want := dates(), "2026-09-30:open 2026-10-07:open 2026-10-08:open 2026-10-09:open 2026-10-10:open 2026-10-11:open 2026-10-14:assigned"; got != want {
		t.Errorf("after daily = %q, want %q", got, want)
	}

	// Cancel the series from an occurrence: upcoming ones are cancelled, the rule ends, Jan hears about it.
	e.fake.Reset()
	e.do(seed.UserAnna, "POST", "/api/v1/requests/"+second.ID+"/cancel", m{"scope": "series"}).status(t, 200)
	// Today is 2026-10-08: earlier occurrences keep their status, the rest is cancelled.
	if got, want := dates(), "2026-09-30:open 2026-10-07:open 2026-10-08:cancelled 2026-10-09:cancelled 2026-10-10:cancelled 2026-10-11:cancelled 2026-10-14:cancelled"; got != want {
		t.Errorf("after cancel = %q, want %q", got, want)
	}
	if msgs := sentTo(e.fake, seed.UserJan); len(msgs) != 1 {
		t.Errorf("jan pushes = %+v", msgs)
	}
	var rules int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM requests WHERE series_id = $1 AND recurring_rule IS NOT NULL`, head.ID).Scan(&rules)
	if rules != 0 {
		t.Errorf("%d rows still carry the rule", rules)
	}
	if err := e.svc.MaterializeAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM requests WHERE series_id = $1`, head.ID).Scan(&n)
	if n != 7 {
		t.Errorf("a cancelled series grew to %d rows", n)
	}
}

func TestHelperReminders(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	r := e.create(seed.UserAnna, showHelperBody(2)) // Fri 2026-10-02 08:30
	path := "/api/v1/requests/" + r.ID
	accepted := e.do(seed.UserJan, "POST", path+"/accept", nil).status(t, 200).req(t)

	// Default: the day before at 18:00 Berlin (= 16:00 UTC).
	want := time.Date(2026, 10, 1, 18, 0, 0, 0, berlin)
	if accepted.RemindHelperAt == nil || !accepted.RemindHelperAt.Equal(want) {
		t.Fatalf("remind_helper_at = %v, want %v", accepted.RemindHelperAt, want)
	}
	e.fake.Reset()
	e.setNow(want.Add(-time.Minute))
	if err := e.svc.SendHelperReminders(ctx); err != nil {
		t.Fatal(err)
	}
	if len(e.fake.Sent()) != 0 {
		t.Fatalf("early reminder: %+v", e.fake.Sent())
	}
	e.setNow(want.Add(2 * time.Minute))
	if err := e.svc.SendHelperReminders(ctx); err != nil {
		t.Fatal(err)
	}
	msgs := sentTo(e.fake, seed.UserJan)
	if len(msgs) != 1 || len(e.fake.Sent()) != 1 {
		t.Fatalf("reminders = %+v", e.fake.Sent())
	}
	m0 := msgs[0]
	if m0.Data["kind"] != "helper" || !strings.HasPrefix(m0.Title, "Erinnerung: Turniertrottel · Herbstturnier: Fanta") ||
		!strings.Contains(m0.Body, "Fr, 2. Okt · 08:30 · Reitanlage Haltern") ||
		!strings.Contains(m0.Body, "Checkliste: Pferd halten, Filmen") {
		t.Errorf("reminder = %+v", m0)
	}
	// Marked as sent: no duplicates, also not with a later helper who joins after the reminder time.
	e.fake.Reset()
	if err := e.svc.SendHelperReminders(ctx); err != nil {
		t.Fatal(err)
	}
	e.do(seed.UserTom, "POST", path+"/accept", nil).status(t, 200)
	if err := e.svc.SendHelperReminders(ctx); err != nil {
		t.Fatal(err)
	}
	for _, msg := range e.fake.Sent() {
		if strings.HasPrefix(msg.Title, "Erinnerung") {
			t.Errorf("duplicate or late reminder: %+v", msg)
		}
	}
	var sentRows int
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM reminders WHERE source_table = 'requests' AND source_id = $1 AND kind = 'helper' AND sent_at IS NOT NULL`, r.ID).Scan(&sentRows)
	if sentRows != 2 {
		t.Errorf("reminder rows = %d", sentRows)
	}

	// A creator-chosen time wins over the default; changing it re-arms the reminder.
	e.setNow(time.Date(2026, 9, 30, 10, 0, 0, 0, berlin))
	r2 := e.create(seed.UserAnna, m{"type": "other", "date": "2026-10-05", "remind_helper_at": "2026-10-05T05:00:00Z"})
	got := e.do(seed.UserKai, "POST", "/api/v1/requests/"+r2.ID+"/accept", nil).status(t, 200).req(t)
	if got.RemindHelperAt == nil || !got.RemindHelperAt.Equal(time.Date(2026, 10, 5, 5, 0, 0, 0, time.UTC)) {
		t.Fatalf("custom reminder = %v", got.RemindHelperAt)
	}
	e.fake.Reset()
	e.setNow(time.Date(2026, 10, 5, 7, 5, 0, 0, berlin))
	if err := e.svc.SendHelperReminders(ctx); err != nil {
		t.Fatal(err)
	}
	if len(sentTo(e.fake, seed.UserKai)) != 1 {
		t.Fatalf("custom reminder not sent: %+v", e.fake.Sent())
	}

	// Cancelled requests and finished days send nothing.
	e.setNow(time.Date(2026, 9, 30, 10, 0, 0, 0, berlin))
	r3 := e.create(seed.UserAnna, m{"type": "other", "date": "2026-10-10"})
	e.do(seed.UserLea, "POST", "/api/v1/requests/"+r3.ID+"/accept", nil).status(t, 200)
	e.do(seed.UserAnna, "POST", "/api/v1/requests/"+r3.ID+"/cancel", nil).status(t, 200)
	e.fake.Reset()
	e.setNow(time.Date(2026, 10, 9, 19, 0, 0, 0, berlin))
	if err := e.svc.SendHelperReminders(ctx); err != nil {
		t.Fatal(err)
	}
	if len(sentTo(e.fake, seed.UserLea)) != 0 {
		t.Errorf("reminder for a cancelled request: %+v", e.fake.Sent())
	}

	// Accepting after the default time passed sets no reminder at all.
	e.setNow(time.Date(2026, 10, 11, 20, 0, 0, 0, berlin))
	r4 := e.create(seed.UserAnna, m{"type": "other", "date": "2026-10-12"})
	got = e.do(seed.UserKai, "POST", "/api/v1/requests/"+r4.ID+"/accept", nil).status(t, 200).req(t)
	if got.RemindHelperAt != nil {
		t.Errorf("late accept has reminder %v", got.RemindHelperAt)
	}
}

func TestCalendarAndICS(t *testing.T) {
	e := newEnv(t)
	long := strings.Repeat("Trense, Sattel; Gamaschen ", 6)
	timed := e.create(seed.UserAnna, m{
		"type": "exercise", "date": "2026-10-02", "time_from": "17:00", "time_to": "18:00", "horse_id": seed.HorseLuna,
		"location": "Halle, Bahn 2", "description": "Bitte locker\nlongieren; keine Sprünge\\" + long,
		"tasks": []string{"Abschwitzdecke", "Hufe auskratzen"}, "payload": m{"mode": "lunge"},
	})
	allDay := e.create(seed.UserAnna, m{"type": "feed_or_turnout", "date": "2026-10-05", "date_end": "2026-10-07", "horse_id": seed.HorseFanta, "payload": m{"what": "feed"}})
	other := e.create(seed.UserAnna, m{"type": "other", "date": "2026-10-09"})
	for _, r := range []requests.Request{allDay, timed} {
		e.do(seed.UserJan, "POST", "/api/v1/requests/"+r.ID+"/accept", nil).status(t, 200)
	}

	// Calendar: only my accepted requests, in date order.
	l := e.do(seed.UserJan, "GET", "/api/v1/requests/calendar", nil).status(t, 200)
	var cal struct{ Requests []requests.Request }
	if err := json.Unmarshal(l.Body.Bytes(), &cal); err != nil || strings.Join(ids(cal.Requests), ",") != timed.ID+","+allDay.ID {
		t.Fatalf("calendar = %s (%v)", l.Body.String(), err)
	}
	if got := e.do(seed.UserKai, "GET", "/api/v1/requests/calendar", nil).status(t, 200).Body.String(); !strings.Contains(got, `"requests":[]`) {
		t.Errorf("kai calendar = %s", got)
	}
	_ = other

	// ICS of the calendar.
	rec := e.do(seed.UserJan, "GET", "/api/v1/requests/calendar.ics", nil).status(t, 200)
	if ct := rec.Header().Get("Content-Type"); ct != "text/calendar; charset=utf-8" {
		t.Errorf("content type = %q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"BEGIN:VCALENDAR\r\nVERSION:2.0\r\n", "PRODID:-//Reiterhof//Anfragen//DE\r\n", "BEGIN:VTIMEZONE\r\nTZID:Europe/Berlin\r\n",
		"DTSTART;TZID=Europe/Berlin:20261002T170000\r\n", "DTEND;TZID=Europe/Berlin:20261002T180000\r\n",
		"UID:" + timed.ID + "@reiterhof.app\r\n", "SUMMARY:Bewegen: Luna\r\n", `LOCATION:Halle\, Bahn 2` + "\r\n",
		"DTSTART;VALUE=DATE:20261005\r\n", "DTEND;VALUE=DATE:20261008\r\n", // inclusive end 7th -> exclusive 8th
		"STATUS:CONFIRMED\r\n", "END:VCALENDAR\r\n",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("ICS lacks %q\n%s", want, body)
		}
	}
	if strings.Count(body, "BEGIN:VEVENT") != 2 || strings.Contains(body, other.ID) {
		t.Errorf("ICS events wrong:\n%s", body)
	}
	assertICSWellFormed(t, body)
	unfolded := strings.ReplaceAll(body, "\r\n ", "")
	if !strings.Contains(unfolded, `DESCRIPTION:Bitte locker\nlongieren\; keine Sprünge\\Trense\, Sattel\; Gamaschen`) ||
		!strings.Contains(unfolded, `Checkliste: Abschwitzdecke\, Hufe auskratzen`) || !strings.Contains(unfolded, `Helfer: Jan`) {
		t.Errorf("DESCRIPTION wrong:\n%s", unfolded)
	}

	// Single event, visible to every member (also to those who do not help).
	one := e.do(seed.UserKai, "GET", "/api/v1/requests/"+timed.ID+".ics", nil).status(t, 200)
	if strings.Count(one.Body.String(), "BEGIN:VEVENT") != 1 || !strings.Contains(one.Header().Get("Content-Disposition"), timed.ID+".ics") {
		t.Errorf("single ICS: %s / %v", one.Body.String(), one.Header())
	}
	e.do(seed.UserKai, "GET", "/api/v1/requests/00000000-0000-4000-8000-0000000009ff.ics", nil).errCode(t, 404, "not_found")
	e.do(seed.UserKai, "GET", "/api/v1/requests/calendar.ics?from=x", nil).status(t, 400)
}

// assertICSWellFormed checks line endings, folding and BEGIN/END balance.
func assertICSWellFormed(t *testing.T, body string) {
	t.Helper()
	if strings.Contains(strings.ReplaceAll(body, "\r\n", ""), "\n") || !strings.HasSuffix(body, "\r\n") {
		t.Error("ICS must use CRLF line endings only")
	}
	depth := 0
	for _, line := range strings.Split(strings.TrimSuffix(body, "\r\n"), "\r\n") {
		if len(line) > 75 {
			t.Errorf("line longer than 75 octets: %q", line)
		}
		switch {
		case strings.HasPrefix(line, "BEGIN:"):
			depth++
		case strings.HasPrefix(line, "END:"):
			depth--
		}
	}
	if depth != 0 {
		t.Errorf("unbalanced BEGIN/END: %d", depth)
	}
}
