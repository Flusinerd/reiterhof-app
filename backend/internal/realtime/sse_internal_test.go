package realtime

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// The heartbeat comment keeps the stream alive; no database needed.
func TestHeartbeat(t *testing.T) {
	s := &stream{deps: httpx.Deps{Events: httpx.NopEvents{}}, heartbeat: 20 * time.Millisecond}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := authtest.WithUser(r.Context(), auth.User{ID: "u", StableID: "s"})
		s.serve(w, r.WithContext(ctx))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), ": keep-alive") {
			return
		}
	}
	t.Fatal("no heartbeat received")
}

func TestHeartbeatIntervalIs25s(t *testing.T) {
	if HeartbeatInterval != 25*time.Second {
		t.Fatal(HeartbeatInterval)
	}
}

func TestDispatchFiltersByStable(t *testing.T) {
	h := NewHub(nil, nil)
	a, cancelA := h.Subscribe("a")
	defer cancelA()
	b, cancelB := h.Subscribe("b")
	defer cancelB()
	h.dispatch(Event{StableID: "a", Type: "t"})
	select {
	case <-a:
	default:
		t.Error("a got nothing")
	}
	select {
	case e := <-b:
		t.Errorf("b got %+v", e)
	default:
	}
	// Resync (no stable) reaches everybody.
	h.dispatch(Event{Type: TypeResync})
	if len(a) != 1 || len(b) != 1 {
		t.Errorf("resync: a=%d b=%d", len(a), len(b))
	}
}
