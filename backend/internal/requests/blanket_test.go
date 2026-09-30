package requests_test

import (
	"context"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/requests"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestCompleteBlanketRequests(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	today := e.create(seed.UserJan, m{"type": "blanket", "horse_id": seed.HorseLuna, "date": "2026-09-30", "payload": m{"wish": "x"}})
	later := e.create(seed.UserJan, m{"type": "blanket", "horse_id": seed.HorseLuna, "date": "2026-10-05", "payload": m{}})
	other := e.create(seed.UserJonas, m{"type": "blanket", "horse_id": seed.HorseBalu, "date": "2026-09-30", "payload": m{}})

	complete := func(horse, day, feedback string) []string {
		t.Helper()
		tx, err := e.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		ids, err := requests.CompleteBlanketRequests(ctx, tx, seed.StableB, horse, day, feedback)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		return ids
	}
	if ids := complete(seed.HorseLuna, "2026-09-30", ""); len(ids) != 1 || ids[0] != today.ID {
		t.Fatalf("closed = %v, want [%s]", ids, today.ID)
	}
	var status, wish string
	var hasFeedback bool
	if err := e.pool.QueryRow(ctx, `SELECT status, payload->>'wish', payload ? 'feedback' FROM requests WHERE id = $1`, today.ID).Scan(&status, &wish, &hasFeedback); err != nil {
		t.Fatal(err)
	}
	if status != "done" || wish != "x" || hasFeedback {
		t.Errorf("empty feedback: status %s wish %q feedback %v", status, wish, hasFeedback)
	}
	// Already done: nothing more to close.
	if ids := complete(seed.HorseLuna, "2026-09-30", "again"); len(ids) != 0 {
		t.Errorf("closed twice: %v", ids)
	}
	// The later request is closed on its own day with feedback; the other horse stays open.
	if ids := complete(seed.HorseLuna, "2026-10-05", "Eingedeckt"); len(ids) != 1 || ids[0] != later.ID {
		t.Fatalf("closed = %v", ids)
	}
	var feedback string
	if err := e.pool.QueryRow(ctx, `SELECT payload->>'feedback' FROM requests WHERE id = $1`, later.ID).Scan(&feedback); err != nil || feedback != "Eingedeckt" {
		t.Errorf("feedback = %q, %v", feedback, err)
	}
	if err := e.pool.QueryRow(ctx, `SELECT status FROM requests WHERE id = $1`, other.ID).Scan(&status); err != nil || status != "open" {
		t.Errorf("other horse: %s, %v", status, err)
	}
	if ids := complete("00000000-0000-4000-8000-0000000009ff", "2026-09-30", ""); len(ids) != 0 {
		t.Errorf("unknown horse closed %v", ids)
	}
}
