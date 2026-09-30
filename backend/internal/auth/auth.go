// Package auth implements authentication (magic link, Sign in with Google/Apple),
// sessions, the current-user middleware and the role checks used by all domain packages.
//
// Contract for other packages (see docs/architecture.md, "Authentication"):
//
//	user, ok := auth.UserFrom(r.Context())          // set by auth.Middleware
//	mux.Handle("GET /api/v1/x", auth.RequireStable(h)) // 401 without session, 403 without stable
//
// Identifiers are UUIDs in text form (string), like everywhere else in the backend.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

const (
	// SessionTTL is the sliding lifetime of a session.
	SessionTTL = 90 * 24 * time.Hour
	// touchInterval limits how often a session's last_seen_at/expires_at are written.
	touchInterval = time.Hour
)

// User is the authenticated user as seen by domain handlers.
type User struct {
	ID string // UUID in text form
	// StableID is the UUID of the user's stable, or "" if the user has not joined one yet.
	// Handlers behind RequireStable can rely on it being set.
	StableID string
	IsAdmin  bool
	Name     string
}

// DB is the subset of pgx used by this package; *pgxpool.Pool and pgx.Tx both satisfy it.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ctxKey int

const (
	userKey ctxKey = iota
	sessionKey
)

// UserFrom returns the authenticated user of the request context.
func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey).(User)
	return u, ok
}

// WithUser returns a context carrying u. Middleware uses it; tests use it through authtest.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// Middleware wraps the whole mux. It reads "Authorization: Bearer <token>", loads the
// session's user and stores it in the context. It never rejects a request itself (public
// routes stay public, a stale token on /healthz is harmless); use RequireUser or
// RequireStable on protected routes. Sessions slide: 90 days from the last use.
func Middleware(deps httpx.Deps) func(http.Handler) http.Handler {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := BearerToken(r)
			if token == "" || deps.Pool == nil {
				next.ServeHTTP(w, r)
				return
			}
			u, sessionID, err := lookupSession(r.Context(), deps.Pool, token, now())
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				next.ServeHTTP(w, r)
				return
			case err != nil:
				if deps.Log != nil {
					deps.Log.Error("auth: load session", "err", err)
				}
				httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
				return
			}
			ctx := context.WithValue(WithUser(r.Context(), u), sessionKey, sessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireUser answers 401 unauthorized unless the request carries a valid session.
func RequireUser(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFrom(r.Context()); !ok {
			unauthorized(w)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// RequireStable answers 401 without a session and 403 no_stable if the user has not joined a
// stable. Domain handlers use it; behind it User.StableID is always set.
func RequireStable(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFrom(r.Context())
		if !ok {
			unauthorized(w)
			return
		}
		if u.StableID == "" {
			httpx.WriteError(w, http.StatusForbidden, "no_stable", "join a stable first")
			return
		}
		h.ServeHTTP(w, r)
	})
}

// RequireAdmin is RequireStable plus User.IsAdmin (403 forbidden otherwise).
func RequireAdmin(h http.Handler) http.Handler {
	return RequireStable(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, _ := UserFrom(r.Context()); !u.IsAdmin {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "admin only")
			return
		}
		h.ServeHTTP(w, r)
	}))
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "sign in required")
}

// BearerToken extracts the token of an "Authorization: Bearer" header ("" if absent). The
// session token is never read from the URL: query strings end up in histories, proxies and
// share sheets. File downloads that cannot send headers use short-lived links instead
// (files.DownloadLink), the realtime stream sends the header through fetch streaming.
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// AgeConfirmed reports whether the user may give consents and join a stable: they stated to
// be 16 or older, or a parent confirmed through the e-mailed link (Art. 8 GDPR; agegate.go).
func AgeConfirmed(ctx context.Context, q DB, userID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT age_confirmed_at IS NOT NULL OR parental_consent_at IS NOT NULL
		FROM users WHERE id = $1`, userID).Scan(&ok)
	return ok, err
}

// UserByID loads a user without a session, for download links (files.DownloadLink). Deleted
// (anonymised) accounts are not found.
func UserByID(ctx context.Context, q DB, id string) (User, error) {
	var (
		u        User
		stableID *string
	)
	err := q.QueryRow(ctx, `SELECT id, stable_id, is_admin, name FROM users WHERE id = $1 AND deleted_at IS NULL`, id).
		Scan(&u.ID, &stableID, &u.IsAdmin, &u.Name)
	if err != nil {
		return User{}, err
	}
	if stableID != nil {
		u.StableID = *stableID
	}
	return u, nil
}

// newToken returns a random URL-safe token (256 bit) and its SHA-256 hash for storage.
func newToken() (token string, hash []byte, err error) {
	var b [32]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", nil, err
	}
	token = base64.RawURLEncoding.EncodeToString(b[:])
	return token, hashToken(token), nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// CreateSession stores a new session for the user and returns the opaque bearer token.
func CreateSession(ctx context.Context, q DB, userID, userAgent string, now time.Time) (string, error) {
	token, hash, err := newToken()
	if err != nil {
		return "", err
	}
	if len(userAgent) > 256 {
		userAgent = userAgent[:256]
	}
	_, err = q.Exec(ctx, `INSERT INTO auth_sessions (token_hash, user_id, user_agent, created_at, last_seen_at, expires_at)
		VALUES ($1, $2, $3, $4, $4, $5)`, hash, userID, userAgent, now, now.Add(SessionTTL))
	if err != nil {
		return "", err
	}
	return token, nil
}

func lookupSession(ctx context.Context, q DB, token string, now time.Time) (User, string, error) {
	var (
		u        User
		stableID *string
		id       string
		lastSeen time.Time
	)
	err := q.QueryRow(ctx, `SELECT u.id, u.stable_id, u.is_admin, u.name, s.id, s.last_seen_at
		FROM auth_sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > $2`, hashToken(token), now).
		Scan(&u.ID, &stableID, &u.IsAdmin, &u.Name, &id, &lastSeen)
	if err != nil {
		return User{}, "", err
	}
	if stableID != nil {
		u.StableID = *stableID
	}
	if now.Sub(lastSeen) > touchInterval {
		// Best effort: a failed slide must not fail the request.
		_, _ = q.Exec(ctx, `UPDATE auth_sessions SET last_seen_at = $2, expires_at = $3 WHERE id = $1`,
			id, now, now.Add(SessionTTL))
	}
	return u, id, nil
}
