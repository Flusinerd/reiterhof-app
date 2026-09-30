// Package pushtest has test helpers for code that sends pushes.
package pushtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GrantConsent records a current `push` consent for the user, which push.Notifier
// requires before it sends anything to the user's devices.
func GrantConsent(t testing.TB, pool *pgxpool.Pool, userID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO consents (user_id, stable_id, kind, version, granted_at)
		SELECT id, stable_id, 'push', 'test', now() FROM users WHERE id = $1
		ON CONFLICT (user_id, kind) DO UPDATE SET revoked_at = NULL`, userID)
	if err != nil {
		t.Fatalf("pushtest: grant push consent: %v", err)
	}
}
