// Package realtimetest helps domain tests assert the realtime events a request publishes.
//
//	w := realtimetest.Watch(t, pool, seed.StableB)
//	// ... call the API ...
//	data := w.Next("horse.changed") // {"horse_id": ...}
//	w.Quiet()                       // nothing else was published
package realtimetest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

// Watcher receives the events of one stable.
type Watcher struct {
	t  testing.TB
	ch <-chan realtime.Event
}

// Watch starts a hub on the pool (stopped when the test ends) and subscribes to the stable. Events
// published through the same database reach it, whichever handler published them.
func Watch(t testing.TB, pool *pgxpool.Pool, stableID string) *Watcher {
	t.Helper()
	hub := realtime.NewHub(pool, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { hub.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case <-hub.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("realtimetest: hub did not start listening")
	}
	ch, unsubscribe := hub.Subscribe(stableID)
	t.Cleanup(unsubscribe)
	return &Watcher{t: t, ch: ch}
}

// Next waits for the next event, which must be of the given type, and returns its data.
func (w *Watcher) Next(eventType string) map[string]any {
	w.t.Helper()
	select {
	case ev, ok := <-w.ch:
		if !ok {
			w.t.Fatalf("realtimetest: stream closed while waiting for %s", eventType)
		}
		if ev.Type != eventType {
			w.t.Fatalf("realtimetest: event %s %s, want %s", ev.Type, ev.Data, eventType)
		}
		var data map[string]any
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			w.t.Fatalf("realtimetest: data %s: %v", ev.Data, err)
		}
		return data
	case <-time.After(5 * time.Second):
		w.t.Fatalf("realtimetest: no %s event", eventType)
		return nil
	}
}

// Quiet fails if an event arrives within a short grace period.
func (w *Watcher) Quiet() {
	w.t.Helper()
	select {
	case ev := <-w.ch:
		w.t.Fatalf("realtimetest: unexpected event %s %s", ev.Type, ev.Data)
	case <-time.After(300 * time.Millisecond):
	}
}
