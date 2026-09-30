// Package presence tracks who is at the stable ("Bin da" / "Ich gehe", geofence).
//
// A visit is a row in `presence`; at most one visit per user is open (left_at IS NULL),
// enforced by a unique partial index. Rules are documented in docs/domains/presence.md.
package presence

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

// EventChanged is published (no data) whenever a visit opens or closes.
const EventChanged = "presence.changed"

const (
	// SourceManual and SourceGeofence are the allowed values of presence.source.
	SourceManual   = "manual"
	SourceGeofence = "geofence"

	// StaleAfter is the longest a visit may stay open; see CloseStale.
	StaleAfter = 12 * time.Hour

	// recentWindow limits "zuletzt gesehen" to people seen within this time.
	recentWindow = 60 * 24 * time.Hour
	// recentLimit caps the list.
	recentLimit = 50
	// hintWindow and hintMinVisits define the "usually arrives around" hint.
	hintWindow    = 8 * 7 * 24 * time.Hour
	hintMinVisits = 4
)

// Register mounts the presence routes.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	mux.Handle("POST /api/v1/presence/check-in", auth.RequireStable(http.HandlerFunc(h.checkIn)))
	mux.Handle("POST /api/v1/presence/check-out", auth.RequireStable(http.HandlerFunc(h.checkOut)))
	mux.Handle("GET /api/v1/presence", auth.RequireStable(http.HandlerFunc(h.overview)))
}

type handler struct{ deps httpx.Deps }

// Visit is one stay of the current user.
type Visit struct {
	ID        string     `json:"id"`
	ArrivedAt time.Time  `json:"arrived_at"`
	LeftAt    *time.Time `json:"left_at"`
	Source    string     `json:"source"`
}

type visitResponse struct {
	Visit *Visit `json:"visit"`
}

func (h *handler) internal(w http.ResponseWriter, what string, err error) {
	h.deps.Log.Error("presence: "+what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

// checkIn is idempotent: an already open visit is returned unchanged.
func (h *handler) checkIn(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	in := struct {
		Source string `json:"source"`
	}{Source: SourceManual}
	if r.ContentLength != 0 && !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Source == "" {
		in.Source = SourceManual
	}
	if in.Source != SourceManual && in.Source != SourceGeofence {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "source must be manual or geofence")
		return
	}
	v, _, err := CheckIn(r.Context(), h.deps.Pool, user.StableID, user.ID, in.Source, h.deps.Now())
	if err != nil {
		h.internal(w, "check-in", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, visitResponse{Visit: v})
}

// checkOut closes the open visit; without one it does nothing (visit: null).
func (h *handler) checkOut(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	v, err := CheckOut(r.Context(), h.deps.Pool, user.StableID, user.ID, h.deps.Now())
	if err != nil {
		h.internal(w, "check-out", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, visitResponse{Visit: v})
}

type txStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// CheckIn opens a visit for the user, or returns the already open one (created=false).
// It publishes presence.changed in the same transaction when a visit was opened.
func CheckIn(ctx context.Context, db txStarter, stableID, userID, source string, now time.Time) (v *Visit, created bool, err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	v = &Visit{}
	err = tx.QueryRow(ctx, `INSERT INTO presence (stable_id, user_id, arrived_at, source)
		SELECT $1, u.id, $3, $4 FROM users u WHERE u.id = $2 AND u.stable_id = $1
		ON CONFLICT (user_id) WHERE left_at IS NULL DO NOTHING
		RETURNING id, arrived_at, left_at, source`, stableID, userID, now, source).
		Scan(&v.ID, &v.ArrivedAt, &v.LeftAt, &v.Source)
	switch {
	case err == nil:
		created = true
		if err := realtime.Publish(ctx, tx, stableID, EventChanged, nil); err != nil {
			return nil, false, err
		}
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `SELECT id, arrived_at, left_at, source FROM presence
			WHERE user_id = $1 AND stable_id = $2 AND left_at IS NULL`, userID, stableID).
			Scan(&v.ID, &v.ArrivedAt, &v.LeftAt, &v.Source)
		if err != nil {
			return nil, false, err
		}
	default:
		return nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	return v, created, nil
}

// CheckOut closes the user's open visit and returns it, or nil if none was open.
func CheckOut(ctx context.Context, db txStarter, stableID, userID string, now time.Time) (*Visit, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	v := &Visit{}
	err = tx.QueryRow(ctx, `UPDATE presence SET left_at = GREATEST(arrived_at, $3)
		WHERE user_id = $1 AND stable_id = $2 AND left_at IS NULL
		RETURNING id, arrived_at, left_at, source`, userID, stableID, now).
		Scan(&v.ID, &v.ArrivedAt, &v.LeftAt, &v.Source)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := realtime.Publish(ctx, tx, stableID, EventChanged, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return v, nil
}
