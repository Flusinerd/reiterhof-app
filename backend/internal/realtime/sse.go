package realtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// HeartbeatInterval is how often an SSE comment keeps proxies and mobile networks from
// closing an idle stream.
const HeartbeatInterval = 25 * time.Second

// Register mounts GET /api/v1/events.
//
// Authentication: the normal bearer header, or, because EventSource implementations in
// React Native usually cannot set headers, the query parameter ?access_token=<token>
// (accepted for this path only, see auth.BearerToken). Tradeoff: URLs end up in access
// logs and proxies; the token is the same long-lived session token. Prefer the header
// (the mobile client uses fetch streaming and does), and make sure the reverse proxy does
// not log query strings for /api/v1/events.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	s := &stream{deps: deps, heartbeat: HeartbeatInterval}
	mux.Handle("GET /api/v1/events", auth.RequireStable(http.HandlerFunc(s.serve)))
}

type stream struct {
	deps      httpx.Deps
	heartbeat time.Duration
}

// serve writes text/event-stream: `event: <type>` + `data: <json>` per event and a
// `: keep-alive` comment every HeartbeatInterval. The optional query `types=a,b` filters.
func (s *stream) serve(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	rc := http.NewResponseController(w)
	// The server may have a write timeout; a stream must outlive it.
	_ = rc.SetWriteDeadline(time.Time{})

	var only map[string]bool
	if t := r.URL.Query().Get("types"); t != "" {
		only = map[string]bool{}
		for _, v := range strings.Split(t, ",") {
			only[strings.TrimSpace(v)] = true
		}
	}

	events, cancel := s.deps.Events.Subscribe(user.StableID)
	defer cancel()

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no") // nginx: do not buffer
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, "retry: 3000\n: connected\n\n"); err != nil || rc.Flush() != nil {
		return
	}

	tick := time.NewTicker(s.heartbeat)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil || rc.Flush() != nil {
				return
			}
		case e, ok := <-events:
			if !ok {
				return // dropped as a slow client or hub stopped: the client reconnects
			}
			if only != nil && !only[e.Type] && e.Type != TypeResync {
				continue
			}
			data, err := json.Marshal(e)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.Type, data); err != nil || rc.Flush() != nil {
				return
			}
		}
	}
}
