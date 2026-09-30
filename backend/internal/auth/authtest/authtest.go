// Package authtest helps domain packages test authenticated endpoints.
//
//	pool := dbtest.NewSeeded(t)
//	req := httptest.NewRequest("GET", "/api/v1/horses", nil)
//	authtest.Authorize(t, pool, req, seed.UserJan) // Authorization: Bearer <real session>
//	httpapi.NewHandler(httpapi.Deps{Pool: pool}).ServeHTTP(rec, req)
//
// For handlers called directly (without the middleware), use authtest.WithUser.
package authtest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
)

// WithUser returns ctx as if auth.Middleware had authenticated u.
func WithUser(ctx context.Context, u auth.User) context.Context {
	return auth.WithUser(ctx, u)
}

// Token creates a real session for the user and returns the bearer token. The session is
// valid relative to the wall clock; use TokenAt if the test fixes deps.Now.
func Token(t testing.TB, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	return TokenAt(t, pool, userID, time.Now())
}

// TokenAt is Token with an explicit "now" (must match the clock the handler under test uses).
func TokenAt(t testing.TB, pool *pgxpool.Pool, userID string, now time.Time) string {
	t.Helper()
	token, err := auth.CreateSession(context.Background(), pool, userID, "authtest", now)
	if err != nil {
		t.Fatalf("authtest: create session: %v", err)
	}
	return token
}

// Authorize sets the bearer token of a fresh session for userID on the request.
func Authorize(t testing.TB, pool *pgxpool.Pool, req *http.Request, userID string) {
	t.Helper()
	req.Header.Set("Authorization", "Bearer "+Token(t, pool, userID))
}
