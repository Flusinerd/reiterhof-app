package realtime_test

import (
	"bufio"
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
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const otherStable = "00000000-0000-4000-8000-0000000009b1"

type server struct {
	pool *pgxpool.Pool
	hub  *realtime.Hub
	srv  *httptest.Server
}

func newServer(t *testing.T) *server {
	t.Helper()
	pool := dbtest.NewSeeded(t)
	hub := realtime.NewHub(pool, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { hub.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case <-hub.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("hub did not start listening")
	}
	srv := httptest.NewServer(httpapi.NewHandler(httpapi.Deps{Pool: pool, Events: hub}))
	t.Cleanup(srv.Close)
	return &server{pool: pool, hub: hub, srv: srv}
}

// sse is a connected event stream; lines are delivered on lines.
type sse struct {
	resp  *http.Response
	lines chan string
}

func (s *server) connect(t *testing.T, path, token string, header bool) *sse {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, _ := http.NewRequestWithContext(ctx, "GET", s.srv.URL+path, nil)
	if header {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	c := &sse{resp: resp, lines: make(chan string, 100)}
	go func() {
		defer close(c.lines)
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			c.lines <- sc.Text()
		}
	}()
	return c
}

// waitFor returns true when a line with the given prefix arrives in time.
func (c *sse) waitFor(prefix string, d time.Duration) (string, bool) {
	deadline := time.After(d)
	for {
		select {
		case l, ok := <-c.lines:
			if !ok {
				return "", false
			}
			if strings.HasPrefix(l, prefix) {
				return l, true
			}
		case <-deadline:
			return "", false
		}
	}
}

func TestSSEDeliversPublishedEventToOwnStableOnly(t *testing.T) {
	s := newServer(t)
	if _, err := s.pool.Exec(context.Background(), `INSERT INTO stables (id, name) VALUES ($1, 'Other')`, otherStable); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(context.Background(), `INSERT INTO users (id, stable_id, name, email) VALUES ('00000000-0000-4000-8000-0000000009b2', $1, 'Olga', 'olga@example.org')`, otherStable); err != nil {
		t.Fatal(err)
	}
	mine := s.connect(t, "/api/v1/events", authtest.Token(t, s.pool, seed.UserAnna), true)
	foreign := s.connect(t, "/api/v1/events", authtest.Token(t, s.pool, "00000000-0000-4000-8000-0000000009b2"), true)
	for _, c := range []*sse{mine, foreign} {
		if c.resp.StatusCode != 200 || c.resp.Header.Get("Content-Type") != "text/event-stream" {
			t.Fatalf("status %d, content type %q", c.resp.StatusCode, c.resp.Header.Get("Content-Type"))
		}
		if _, ok := c.waitFor(": connected", 2*time.Second); !ok {
			t.Fatal("no initial comment")
		}
	}

	if err := realtime.Publish(context.Background(), s.pool, seed.StableB, "request.changed", map[string]any{"id": 7}); err != nil {
		t.Fatal(err)
	}
	if l, ok := mine.waitFor("event: request.changed", 3*time.Second); !ok {
		t.Fatal("event not delivered")
	} else if l != "event: request.changed" {
		t.Fatal(l)
	}
	data, ok := mine.waitFor("data: ", time.Second)
	if !ok {
		t.Fatal("no data line")
	}
	var e realtime.Event
	if err := json.Unmarshal([]byte(strings.TrimPrefix(data, "data: ")), &e); err != nil {
		t.Fatal(err)
	}
	if e.StableID != seed.StableB || e.Type != "request.changed" || string(e.Data) != `{"id":7}` {
		t.Errorf("event = %+v", e)
	}
	if l, ok := foreign.waitFor("event:", 300*time.Millisecond); ok {
		t.Errorf("event of another stable leaked: %s", l)
	}
}

func TestSSEAuth(t *testing.T) {
	s := newServer(t)
	if c := s.connect(t, "/api/v1/events", "", false); c.resp.StatusCode != 401 {
		t.Errorf("no token: status %d, want 401", c.resp.StatusCode)
	}
	token := authtest.Token(t, s.pool, seed.UserAnna)
	if c := s.connect(t, "/api/v1/events", token, true); c.resp.StatusCode != 200 {
		t.Errorf("bearer header: status %d, want 200", c.resp.StatusCode)
	}
	// A session token in the query string is never accepted (it would end up in logs and histories).
	if c := s.connect(t, "/api/v1/events?access_token="+token, "", false); c.resp.StatusCode != 401 {
		t.Errorf("access_token query: status %d, want 401", c.resp.StatusCode)
	}
	if c := s.connect(t, "/api/v1/events?access_token=bogus", "", false); c.resp.StatusCode != 401 {
		t.Errorf("bad token: status %d, want 401", c.resp.StatusCode)
	}
	req, _ := http.NewRequest("GET", s.srv.URL+"/api/v1/presence?access_token="+token, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("access_token on /presence: status %d, want 401", resp.StatusCode)
	}
}

func TestSSETypesFilter(t *testing.T) {
	s := newServer(t)
	c := s.connect(t, "/api/v1/events?types=b.changed", authtest.Token(t, s.pool, seed.UserAnna), true)
	c.waitFor(": connected", 2*time.Second)
	_ = realtime.Publish(context.Background(), s.pool, seed.StableB, "a.changed", nil)
	_ = realtime.Publish(context.Background(), s.pool, seed.StableB, "b.changed", nil)
	if l, ok := c.waitFor("event:", 3*time.Second); !ok || l != "event: b.changed" {
		t.Errorf("first event = %q, %v; want b.changed", l, ok)
	}
}

func TestPresenceCheckInPublishesEvent(t *testing.T) {
	s := newServer(t)
	c := s.connect(t, "/api/v1/events", authtest.Token(t, s.pool, seed.UserTom), true)
	c.waitFor(": connected", 2*time.Second)

	req, _ := http.NewRequest("POST", s.srv.URL+"/api/v1/presence/check-in", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+authtest.Token(t, s.pool, seed.UserAnna))
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("check-in: %v %v", err, resp)
	}
	resp.Body.Close()
	if _, ok := c.waitFor("event: "+presence.EventChanged, 3*time.Second); !ok {
		t.Fatal("presence.changed not delivered to the other user")
	}
}

func TestPublishInRolledBackTransactionIsNotDelivered(t *testing.T) {
	s := newServer(t)
	events, cancel := s.hub.Subscribe(seed.StableB)
	defer cancel()

	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := realtime.Publish(context.Background(), tx, seed.StableB, "x.changed", nil); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback(context.Background())
	select {
	case e := <-events:
		t.Fatalf("got %+v from a rolled back transaction", e)
	case <-time.After(300 * time.Millisecond):
	}

	tx, _ = s.pool.Begin(context.Background())
	_ = realtime.Publish(context.Background(), tx, seed.StableB, "x.changed", nil)
	_ = tx.Commit(context.Background())
	select {
	case e := <-events:
		if e.Type != "x.changed" {
			t.Fatalf("event = %+v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("committed event not delivered")
	}
}

func TestSlowClientIsDropped(t *testing.T) {
	s := newServer(t)
	slow, cancelSlow := s.hub.Subscribe(seed.StableB)
	defer cancelSlow()
	fast, cancelFast := s.hub.Subscribe(seed.StableB)
	defer cancelFast()

	got := make(chan int, 1)
	go func() {
		n := 0
		for range fast {
			n++
		}
		got <- n
	}()
	// Nobody reads `slow`; publish more than its buffer holds.
	// Paced, so that the reading client keeps up while the other overflows its buffer.
	for i := 0; i < 50; i++ {
		if err := realtime.Publish(context.Background(), s.pool, seed.StableB, "n.changed", nil); err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	deadline := time.After(5 * time.Second)
	for s.hub.Clients() != 1 {
		select {
		case <-deadline:
			t.Fatalf("clients = %d, want the slow one dropped", s.hub.Clients())
		case <-time.After(20 * time.Millisecond):
		}
	}
	// The slow channel is closed after its buffered events.
	n := 0
	for range slow {
		n++
	}
	if n == 0 || n >= 50 {
		t.Errorf("slow client got %d events", n)
	}
	cancelFast()
	if n := <-got; n == 0 {
		t.Error("fast client got nothing")
	}
}

func TestPublishValidation(t *testing.T) {
	s := newServer(t)
	if err := realtime.Publish(context.Background(), s.pool, "", "x", nil); err == nil {
		t.Error("empty stable must fail")
	}
	if err := realtime.Publish(context.Background(), s.pool, seed.StableB, "x", strings.Repeat("a", 9000)); err == nil {
		t.Error("oversized payload must fail")
	}
}
