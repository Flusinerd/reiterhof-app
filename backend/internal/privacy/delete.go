package privacy

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

// DeletedName is the display name of an anonymised account.
const DeletedName = "Gelöschtes Mitglied"

// ErrLastAdmin is returned when the user is the only admin of their stable.
var ErrLastAdmin = errors.New("privacy: user is the last admin of the stable")

// OwnedHorse names a horse that blocks the deletion.
type OwnedHorse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OwnsHorsesError is returned when the user still owns horses.
type OwnsHorsesError struct{ Horses []OwnedHorse }

func (e *OwnsHorsesError) Error() string { return "privacy: user still owns horses" }

// ErrNotFound is returned for an unknown or already deleted user.
var ErrNotFound = errors.New("privacy: user not found")

// DeleteAccount deletes the account of userID (Art. 17 GDPR). Decisions (see
// docs/domains/privacy.md):
//
//   - The deletion is refused while the user owns horses (OwnsHorsesError) or is the only
//     admin of the stable (ErrLastAdmin). An owner transfers or removes the horses first.
//   - The users row is anonymised, not removed: requests, training sessions and similar
//     records of a horse reference it. Name, email, phone, colour, stable membership and
//     admin flag are cleared; the row keeps only the id and DeletedName.
//   - Deleted: sessions, sign-in identities, pending login tokens for the email, push tokens,
//     reminder settings, reminders, presence visits, rider roles, helper assignments,
//     consents, GPS tracks and raw gait windows of training sessions, and uploaded photos of reported observations.
//   - Kept, now attributed to DeletedName: reported observations (text), training sessions
//     (without track), requests (open ones are cancelled), week slots, documents.
func DeleteAccount(ctx context.Context, db interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}, userID string, now time.Time) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		email    string
		stableID *string
		isAdmin  bool
		deleted  *time.Time
	)
	err = tx.QueryRow(ctx, `SELECT email, stable_id::text, is_admin, deleted_at FROM users WHERE id = $1 FOR UPDATE`, userID).
		Scan(&email, &stableID, &isAdmin, &deleted)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && deleted != nil) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `SELECT id::text, name FROM horses WHERE owner_id = $1 ORDER BY name`, userID)
	if err != nil {
		return err
	}
	var owned []OwnedHorse
	for rows.Next() {
		var h OwnedHorse
		if err := rows.Scan(&h.ID, &h.Name); err != nil {
			rows.Close()
			return err
		}
		owned = append(owned, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(owned) > 0 {
		return &OwnsHorsesError{Horses: owned}
	}
	if isAdmin && stableID != nil {
		var others bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users
			WHERE stable_id = $1 AND is_admin AND id <> $2 AND deleted_at IS NULL)`, *stableID, userID).Scan(&others); err != nil {
			return err
		}
		if !others {
			return ErrLastAdmin
		}
	}

	// Photos of reported observations may show people and carry location metadata (EXIF);
	// remove them from disk after the commit. The observation text stays.
	mediaPaths, err := observationMediaPaths(ctx, tx, userID)
	if err != nil {
		return err
	}

	// Requests the person helped with go back to "open" if nobody else is left.
	if _, err := tx.Exec(ctx, `UPDATE requests SET status = 'open'
		WHERE status = 'assigned'
		  AND id IN (SELECT request_id FROM request_assignees WHERE user_id = $1)
		  AND NOT EXISTS (SELECT 1 FROM request_assignees a WHERE a.request_id = requests.id AND a.user_id <> $1)`, userID); err != nil {
		return err
	}
	statements := []string{
		`DELETE FROM request_assignees WHERE user_id = $1`,
		`UPDATE requests SET status = 'cancelled' WHERE created_by = $1 AND status IN ('open', 'assigned')`,
		`DELETE FROM horse_riders WHERE user_id = $1`,
		`DELETE FROM presence WHERE user_id = $1`,
		`UPDATE sessions SET track = NULL WHERE user_id = $1`,
		`DELETE FROM gait_windows g USING sessions s WHERE s.id = g.session_id AND s.user_id = $1`,
		`UPDATE observations SET media = '[]'::jsonb WHERE reported_by = $1`,
		`UPDATE stable_invites SET created_by = NULL WHERE created_by = $1`,
		`DELETE FROM reminders WHERE user_id = $1`,
		`DELETE FROM reminder_settings WHERE user_id = $1`,
		`DELETE FROM push_tokens WHERE user_id = $1`,
		`DELETE FROM web_push_subscriptions WHERE user_id = $1`,
		`DELETE FROM consents WHERE user_id = $1`,
		`DELETE FROM auth_sessions WHERE user_id = $1`,
		`DELETE FROM auth_identities WHERE user_id = $1`,
	}
	for _, s := range statements {
		if _, err := tx.Exec(ctx, s, userID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM login_tokens WHERE email = $1`, email); err != nil {
		return err
	}
	// The placeholder address is unique per id, unroutable (.invalid) and lower case.
	if _, err := tx.Exec(ctx, `UPDATE users SET name = $2, email = 'deleted-' || id::text || '@deleted.invalid',
		phone = NULL, avatar_color = NULL, stable_id = NULL, is_admin = false,
		presence_visibility = 'hidden', deleted_at = $3 WHERE id = $1`, userID, DeletedName, now); err != nil {
		return err
	}
	if stableID != nil {
		if err := realtime.Publish(ctx, tx, *stableID, "presence.changed", nil); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// After the commit: a failed file removal must not undo the deletion; leftovers are
	// harmless because no row references them any more.
	for _, p := range mediaPaths {
		stable, _, ok := strings.Cut(p, "/")
		if !ok {
			continue
		}
		_ = files.Remove(stable, p)
	}
	return nil
}
