// Package realtime delivers small "something changed" events to the apps of a stable.
//
// Publishing goes through Postgres NOTIFY (channel Channel), so it works across
// processes and can run inside a transaction: the event is only sent when the
// transaction commits. Every API process runs one Hub that LISTENs on the channel and
// fans events out to its SSE clients (GET /api/v1/events).
//
// Events are hints, not data: keep Data tiny and free of anything a member of the stable
// may not see (e.g. presence visibility); clients react by refetching over the normal API.
package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Channel is the Postgres NOTIFY channel.
const Channel = "reiterhof_events"

// TypeResync is sent by the hub to all subscribers after it lost and re-established its
// database connection: events may have been missed, refetch everything.
const TypeResync = "resync"

// maxPayload is Postgres' NOTIFY limit (8000 bytes) with some headroom.
const maxPayload = 7900

// Event is the wire format inside NOTIFY and SSE.
type Event = httpx.Event

// Execer is what Publish needs; *pgxpool.Pool, pgx.Tx and *pgx.Conn all satisfy it.
type Execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Publish sends an event to all clients of the stable. Call it with your pgx.Tx to tie
// delivery to the commit. data is JSON-marshalled (nil becomes {}); the payload must stay
// small (< 8 kB in total).
func Publish(ctx context.Context, q Execer, stableID, eventType string, data any) error {
	if stableID == "" || eventType == "" {
		return errors.New("realtime: stable id and event type are required")
	}
	raw := json.RawMessage(`{}`)
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("realtime: marshal data: %w", err)
		}
		raw = b
	}
	payload, err := json.Marshal(Event{StableID: stableID, Type: eventType, Data: raw})
	if err != nil {
		return fmt.Errorf("realtime: marshal event: %w", err)
	}
	if len(payload) > maxPayload {
		return fmt.Errorf("realtime: event payload of %d bytes is too large", len(payload))
	}
	if _, err := q.Exec(ctx, `SELECT pg_notify($1, $2)`, Channel, string(payload)); err != nil {
		return fmt.Errorf("realtime: notify: %w", err)
	}
	return nil
}

// bufferSize is the per-client queue; a client that falls this far behind is dropped
// (it reconnects and refetches).
const bufferSize = 32

type subscriber struct {
	stableID string
	ch       chan Event
}

// Hub listens on Channel and distributes events to subscribers. Create it with NewHub and
// start Run once per process.
type Hub struct {
	pool *pgxpool.Pool
	log  *slog.Logger

	mu    sync.Mutex
	subs  map[*subscriber]struct{}
	ready chan struct{}
	once  sync.Once
}

// NewHub creates a hub; call Run to start listening.
func NewHub(pool *pgxpool.Pool, log *slog.Logger) *Hub {
	if log == nil {
		log = slog.Default()
	}
	return &Hub{pool: pool, log: log, subs: map[*subscriber]struct{}{}, ready: make(chan struct{})}
}

// Ready is closed once the hub listens for the first time (useful in tests).
func (h *Hub) Ready() <-chan struct{} { return h.ready }

// Subscribe implements httpx.EventBus.
func (h *Hub) Subscribe(stableID string) (<-chan Event, func()) {
	s := &subscriber{stableID: stableID, ch: make(chan Event, bufferSize)}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s.ch, func() { h.remove(s) }
}

func (h *Hub) remove(s *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[s]; ok {
		delete(h.subs, s)
		close(s.ch)
	}
}

// Clients returns the number of connected subscribers.
func (h *Hub) Clients() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// dispatch delivers e to the subscribers of its stable ("" = everybody). Subscribers whose
// buffer is full are dropped: their channel is closed, the SSE handler ends the stream.
func (h *Hub) dispatch(e Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		if e.StableID != "" && s.stableID != e.StableID {
			continue
		}
		select {
		case s.ch <- e:
		default:
			h.log.Warn("realtime: dropping slow client", "stable_id", s.stableID)
			delete(h.subs, s)
			close(s.ch)
		}
	}
}

// Run listens until ctx is cancelled, reconnecting with backoff (1 s up to 30 s).
func (h *Hub) Run(ctx context.Context) {
	backoff := time.Second
	first := true
	for ctx.Err() == nil {
		started, err := h.listen(ctx, first)
		if ctx.Err() != nil {
			return
		}
		if started {
			first = false
			backoff = time.Second
		}
		h.log.Warn("realtime: listener stopped, reconnecting", "err", err, "in", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

// listen runs one LISTEN session; started reports whether LISTEN succeeded.
func (h *Hub) listen(ctx context.Context, first bool) (started bool, err error) {
	pc, err := h.pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	// A listening connection is never returned to the pool.
	conn := pc.Hijack()
	defer func() {
		cctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = conn.Close(cctx)
	}()
	if _, err := conn.Exec(ctx, "LISTEN "+Channel); err != nil {
		return false, err
	}
	h.once.Do(func() { close(h.ready) })
	if !first {
		h.dispatch(Event{Type: TypeResync, Data: json.RawMessage(`{}`)})
	}
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return true, err
		}
		var e Event
		if err := json.Unmarshal([]byte(n.Payload), &e); err != nil || e.StableID == "" || e.Type == "" {
			h.log.Warn("realtime: ignoring malformed notification", "err", err)
			continue
		}
		h.dispatch(e)
	}
}
