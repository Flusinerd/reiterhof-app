package horses

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

// ErrNotFound is returned by Delete for a horse that does not exist in the stable.
var ErrNotFound = errors.New("horses: horse not found")

// NameMatches reports whether typed is the horse's name: case-insensitive, surrounding
// white space ignored. An empty name never matches.
func NameMatches(name, typed string) bool {
	typed = strings.TrimSpace(typed)
	return typed != "" && strings.EqualFold(strings.TrimSpace(name), typed)
}

// DeleteDB is the part of the pool Delete needs.
type DeleteDB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Deleted describes a deleted horse.
type Deleted struct {
	ID   string
	Name string
	// FilesRemoved counts the stored files removed from disk after the commit; FilesKept the ones
	// another record still points to (or that could not be removed, which is logged).
	FilesRemoved, FilesKept int
}

// Delete permanently deletes a horse of the stable (no archive), shared by DELETE
// /api/v1/horses/{id} and "stallfunk-admin horse delete". Access checks and the name
// confirmation are the caller's job. ErrNotFound if the horse is not in the stable.
//
// One transaction: the foreign keys cascade (horse_riders, blankets, blanket_rules, blanket_states,
// blanket_overrides, observations, health_items, emergency_contacts, horse_documents, training_profiles,
// sessions and their gait_windows, week_slots, reha_plans and reha_days, rider_focuses, dressage_tests);
// requests keep existing with horse_id set NULL. The stored reminders (reminders has no foreign key to
// its source) of the horse's health items, reha plans and blanket weather changes are deleted
// explicitly, and horse.changed is published inside the transaction. After the commit every stored
// file that no record points to any more (photo, documents, observation and blanket images) is
// removed, best effort: a failure is logged and leaves an orphaned file, never an error.
func Delete(ctx context.Context, db DeleteDB, log *slog.Logger, stableID, horseID string) (Deleted, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return Deleted{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	out := Deleted{ID: horseID}
	var photo *string
	err = tx.QueryRow(ctx, `SELECT id::text, name, photo_path FROM horses WHERE id = $1 AND stable_id = $2 FOR UPDATE`,
		horseID, stableID).Scan(&out.ID, &out.Name, &photo)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return Deleted{}, ErrNotFound
	}
	if err != nil {
		return Deleted{}, err
	}

	// Files first: the rows that name them are gone after the delete.
	var paths []string
	if photo != nil {
		paths = append(paths, *photo)
	}
	for _, q := range []string{
		`SELECT file_path FROM horse_documents WHERE horse_id = $1`,
		`SELECT jsonb_array_elements_text(media) FROM observations WHERE horse_id = $1`,
		`SELECT photo_path FROM blankets WHERE horse_id = $1 AND photo_path IS NOT NULL`,
	} {
		rows, err := tx.Query(ctx, q, out.ID)
		if err != nil {
			return Deleted{}, err
		}
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err != nil {
				rows.Close()
				return Deleted{}, err
			}
			paths = append(paths, p)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return Deleted{}, err
		}
	}

	// Stored reminders point to their source without a foreign key; they would be left over.
	// Requests stay (horse_id becomes NULL), so their reminders stay too.
	for _, s := range []string{
		`DELETE FROM reminders WHERE source_table = 'health_items'
			AND source_id IN (SELECT id FROM health_items WHERE horse_id = $1)`,
		`DELETE FROM reminders WHERE source_table = 'reha_plans'
			AND source_id IN (SELECT id FROM reha_plans WHERE horse_id = $1)`,
		`DELETE FROM reminders WHERE source_table = 'blanket_weather' AND source_id = $1`,
		`DELETE FROM horses WHERE id = $1`,
	} {
		if _, err := tx.Exec(ctx, s, out.ID); err != nil {
			return Deleted{}, err
		}
	}
	if err := realtime.Publish(ctx, tx, stableID, EventChanged, map[string]any{"horse_id": out.ID}); err != nil {
		return Deleted{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Deleted{}, err
	}

	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		used, err := files.Referenced(ctx, db, stableID, p)
		if err != nil {
			log.WarnContext(ctx, "horses: file reference check after delete", "path", p, "err", err)
			out.FilesKept++
			continue
		}
		if used {
			out.FilesKept++
			continue
		}
		if err := files.Remove(stableID, p); err != nil {
			log.WarnContext(ctx, "horses: remove file of deleted horse", "path", p, "err", err)
			out.FilesKept++
			continue
		}
		out.FilesRemoved++
	}
	return out, nil
}

type deleteInput struct {
	ConfirmName string `json:"confirm_name"`
}

// deleteHorse handles DELETE /api/v1/horses/{id}: owner or admin, body {confirm_name}.
func (h *handler) deleteHorse(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.horseInStable(w, r)
	if !ok {
		return
	}
	manage, err := auth.CanManageHorse(r.Context(), h.deps.Pool, user, id)
	if err != nil {
		h.fail(w, "can manage", err)
		return
	}
	if !manage {
		forbidden(w, "only the owner or an admin may delete this horse")
		return
	}
	// An empty body is a missing confirmation (400 with the message), not a JSON error.
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, httpx.MaxBodyBytes))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	var in deleteInput
	if len(bytes.TrimSpace(raw)) > 0 {
		r.Body = io.NopCloser(bytes.NewReader(raw))
		if !httpx.ReadJSON(w, r, &in) {
			return
		}
	}
	var name string
	if err := h.deps.Pool.QueryRow(r.Context(), `SELECT name FROM horses WHERE id = $1 AND stable_id = $2`,
		id, user.StableID).Scan(&name); err != nil {
		h.fail(w, "horse name", err)
		return
	}
	if !NameMatches(name, in.ConfirmName) {
		invalid(w, "Der eingegebene Name stimmt nicht mit dem Namen des Pferdes überein.")
		return
	}
	if _, err := Delete(r.Context(), h.deps.Pool, h.deps.Log, user.StableID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			notFound(w)
			return
		}
		h.fail(w, "delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
