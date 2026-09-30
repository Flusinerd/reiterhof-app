// Package privacy implements consents, data subject rights (export, account deletion) and
// the retention job. Rules and decisions are documented in docs/domains/privacy.md.
package privacy

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// TextVersion identifies the current wording of the privacy text and the consent
// explanations (docs/legal/datenschutz.md, mobile/lib/legal). A grant records it. Change it
// together with the texts; then users have to confirm again ("up_to_date": false).
const TextVersion = "2026-09-30"

// Consent kinds (consents.kind).
const (
	KindLocationGeofence = "location_geofence"
	KindLocationTracking = "location_tracking"
	// KindMaps covers loading map tiles from Apple, Google or OpenFreeMap, which see the
	// viewer's IP address and the map area.
	KindMaps            = "maps"
	KindPresenceSharing = "presence_sharing"
	KindPhotos          = "photos"
	KindPush            = "push"
	// KindAITraining lets the training week plan send the owner's horse data (without names)
	// to the language model of Mistral AI (JAN-89). The owner's grant covers their horses.
	KindAITraining = "ai_training"
)

// Kinds lists all consent kinds in display order.
func Kinds() []string {
	return []string{KindLocationGeofence, KindLocationTracking, KindMaps, KindPresenceSharing, KindPhotos, KindPush, KindAITraining}
}

// ErrAgeUnconfirmed is returned when a consent is granted before the person confirmed being
// 16 or older or a parent consented (Art. 8 GDPR; see auth, "age confirmation").
var ErrAgeUnconfirmed = errors.New("privacy: age not confirmed")

// ValidKind reports whether kind is a known consent kind.
func ValidKind(kind string) bool {
	for _, k := range Kinds() {
		if k == kind {
			return true
		}
	}
	return false
}

// DB is the subset of pgx used here; *pgxpool.Pool and pgx.Tx satisfy it.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Consent is the state of one consent kind of a user.
type Consent struct {
	Kind           string     `json:"kind"`
	Granted        bool       `json:"granted"`
	Version        *string    `json:"version"`
	GrantedAt      *time.Time `json:"granted_at"`
	RevokedAt      *time.Time `json:"revoked_at"`
	CurrentVersion string     `json:"current_version"`
	// UpToDate is true when the grant was given for the current text version.
	UpToDate bool `json:"up_to_date"`
}

// Has reports whether the user currently has a grant for kind (not revoked). The version
// is not compared: a text update asks users again in the app but does not lock them out.
func Has(ctx context.Context, q DB, userID, kind string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM consents
		WHERE user_id = $1 AND kind = $2 AND revoked_at IS NULL)`, userID, kind).Scan(&ok)
	return ok, err
}

// List returns all kinds (never granted ones with Granted=false), in display order.
func List(ctx context.Context, q DB, userID string) ([]Consent, error) {
	rows, err := q.Query(ctx, `SELECT kind, version, granted_at, revoked_at FROM consents WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byKind := map[string]Consent{}
	for rows.Next() {
		var c Consent
		var version string
		var granted time.Time
		if err := rows.Scan(&c.Kind, &version, &granted, &c.RevokedAt); err != nil {
			return nil, err
		}
		c.Version, c.GrantedAt = &version, &granted
		c.Granted = c.RevokedAt == nil
		c.UpToDate = c.Granted && version == TextVersion
		byKind[c.Kind] = c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]Consent, 0, len(Kinds()))
	for _, k := range Kinds() {
		c, ok := byKind[k]
		if !ok {
			c = Consent{Kind: k}
		}
		c.CurrentVersion = TextVersion
		out = append(out, c)
	}
	return out, nil
}

// Set grants or revokes a consent and applies the server-side consequences of a revocation:
//   - push: the user's push tokens are deleted (no notification can reach the devices),
//   - presence_sharing: the presence visibility is set to "hidden".
//
// Revoking something that was never granted creates no row and changes nothing else.
func Set(ctx context.Context, db interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}, userID, kind string, granted bool, now time.Time) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if granted {
		// A minor without parental consent cannot consent (the app never gets here, the
		// server checks anyway).
		var ok bool
		if ok, err = auth.AgeConfirmed(ctx, tx, userID); err != nil {
			return err
		}
		if !ok {
			return ErrAgeUnconfirmed
		}
		_, err = tx.Exec(ctx, `INSERT INTO consents (user_id, stable_id, kind, version, granted_at)
			SELECT u.id, u.stable_id, $2, $3, $4 FROM users u WHERE u.id = $1
			ON CONFLICT (user_id, kind) DO UPDATE
			SET version = EXCLUDED.version, granted_at = EXCLUDED.granted_at, revoked_at = NULL,
			    stable_id = EXCLUDED.stable_id`, userID, kind, TextVersion, now)
	} else {
		_, err = tx.Exec(ctx, `UPDATE consents SET revoked_at = $3
			WHERE user_id = $1 AND kind = $2 AND revoked_at IS NULL`, userID, kind, now)
		if err == nil {
			switch kind {
			case KindPush:
				_, err = tx.Exec(ctx, `DELETE FROM push_tokens WHERE user_id = $1`, userID)
				if err == nil {
					_, err = tx.Exec(ctx, `DELETE FROM web_push_subscriptions WHERE user_id = $1`, userID)
				}
			case KindPresenceSharing:
				_, err = tx.Exec(ctx, `UPDATE users SET presence_visibility = 'hidden' WHERE id = $1`, userID)
			}
		}
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (h *handler) listConsents(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	items, err := List(r.Context(), h.deps.Pool, user.ID)
	if err != nil {
		h.internal(w, "list consents", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"current_version": TextVersion, "items": items})
}

func (h *handler) putConsent(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	kind := r.PathValue("kind")
	if !ValidKind(kind) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "unknown consent kind")
		return
	}
	var in struct {
		Granted *bool  `json:"granted"`
		Version string `json:"version"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Granted == nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "granted (true or false) is required")
		return
	}
	// A grant refers to the text the person has seen; a stale app must not grant a newer text.
	if *in.Granted && in.Version != "" && in.Version != TextVersion {
		httpx.WriteError(w, http.StatusConflict, "version_mismatch", "the privacy text has changed, please update the app")
		return
	}
	err := Set(r.Context(), h.deps.Pool, user.ID, kind, *in.Granted, h.deps.Now())
	if errors.Is(err, ErrAgeUnconfirmed) {
		httpx.WriteError(w, http.StatusForbidden, "age_unconfirmed", "confirm your age (or have a parent consent) before granting consents")
		return
	}
	if err != nil {
		h.internal(w, "set consent", err)
		return
	}
	items, err := List(r.Context(), h.deps.Pool, user.ID)
	if err != nil {
		h.internal(w, "list consents", err)
		return
	}
	for _, c := range items {
		if c.Kind == kind {
			httpx.WriteJSON(w, http.StatusOK, c)
			return
		}
	}
	h.internal(w, "consent missing", errors.New(kind))
}
