package weather

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// StoredWarning is a weather_warnings row.
type StoredWarning struct {
	ID          string     `json:"id"`
	Event       string     `json:"event"`
	Severity    string     `json:"severity"`
	Headline    string     `json:"headline"`
	Description string     `json:"description"`
	Instruction string     `json:"instruction"`
	Onset       time.Time  `json:"onset"`
	Expires     *time.Time `json:"expires"`
	FetchedAt   time.Time  `json:"fetched_at"`
	Notified    bool       `json:"-"`
}

// Warning returns the fetched form of the row (for Notify).
func (s StoredWarning) Warning() Warning {
	return Warning{Event: s.Event, Severity: s.Severity, Headline: s.Headline, Description: s.Description,
		Instruction: s.Instruction, Onset: s.Onset, Expires: s.Expires}
}

// ReplaceWarnings makes the stored warnings of the stable the given ones: rows of
// warnings that are gone are deleted, known ones (same DWD identifier) updated, new ones
// inserted. When the DWD updates a warning it sends it with a new identifier; a new row
// counts as already notified when the stable had a notified warning of the same event and
// at least the same severity, so an update does not push again (an escalation does).
func (s *Store) ReplaceWarnings(ctx context.Context, stableID string, ws []Warning, fetchedAt time.Time) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("weather warnings: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	notifiedRank := map[string]int{}
	rows, err := tx.Query(ctx, `SELECT event, severity FROM weather_warnings WHERE stable_id = $1 AND notified`, stableID)
	if err != nil {
		return fmt.Errorf("weather warnings: load: %w", err)
	}
	for rows.Next() {
		var ev, sev string
		if err := rows.Scan(&ev, &sev); err != nil {
			rows.Close()
			return fmt.Errorf("weather warnings: scan: %w", err)
		}
		if r := SeverityRank(sev); r > notifiedRank[ev] {
			notifiedRank[ev] = r
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("weather warnings: load: %w", err)
	}

	ids := make([]string, 0, len(ws))
	for _, w := range ws {
		ids = append(ids, w.DWDID)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM weather_warnings WHERE stable_id = $1 AND NOT (dwd_id = ANY($2))`, stableID, ids); err != nil {
		return fmt.Errorf("weather warnings: delete gone: %w", err)
	}
	for _, w := range ws {
		inherit := notifiedRank[w.Event] >= SeverityRank(w.Severity)
		if _, err := tx.Exec(ctx, `
			INSERT INTO weather_warnings
				(stable_id, dwd_id, event, severity, headline, description, instruction, onset, expires, fetched_at, notified)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (stable_id, dwd_id) DO UPDATE SET
				event = EXCLUDED.event, severity = EXCLUDED.severity, headline = EXCLUDED.headline,
				description = EXCLUDED.description, instruction = EXCLUDED.instruction,
				onset = EXCLUDED.onset, expires = EXCLUDED.expires, fetched_at = EXCLUDED.fetched_at`,
			stableID, w.DWDID, w.Event, w.Severity, w.Headline, w.Description, w.Instruction,
			w.Onset, w.Expires, fetchedAt, inherit); err != nil {
			return fmt.Errorf("weather warnings: save: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("weather warnings: commit: %w", err)
	}
	return nil
}

const warningColumns = `id::text, event, severity, headline, description, instruction, onset, expires, fetched_at, notified`

func scanWarnings(rows pgx.Rows) ([]StoredWarning, error) {
	defer rows.Close()
	out := []StoredWarning{}
	for rows.Next() {
		var w StoredWarning
		if err := rows.Scan(&w.ID, &w.Event, &w.Severity, &w.Headline, &w.Description, &w.Instruction,
			&w.Onset, &w.Expires, &w.FetchedAt, &w.Notified); err != nil {
			return nil, fmt.Errorf("weather warnings: scan: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ActiveWarnings returns the stored warnings of the stable that have not expired at now,
// the most severe first, then by onset.
func (s *Store) ActiveWarnings(ctx context.Context, stableID string, now time.Time) ([]StoredWarning, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+warningColumns+` FROM weather_warnings
		WHERE stable_id = $1 AND (expires IS NULL OR expires > $2)
		ORDER BY CASE severity WHEN 'extreme' THEN 4 WHEN 'severe' THEN 3 WHEN 'moderate' THEN 2 ELSE 1 END DESC,
		         onset, id`, stableID, now)
	if err != nil {
		return nil, fmt.Errorf("weather warnings: active: %w", err)
	}
	return scanWarnings(rows)
}

// pendingWarnings are the warnings of the stable whose push is not done yet.
func (s *Store) pendingWarnings(ctx context.Context, stableID string, now time.Time) ([]StoredWarning, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+warningColumns+` FROM weather_warnings
		WHERE stable_id = $1 AND NOT notified AND (expires IS NULL OR expires > $2)
		ORDER BY onset, id`, stableID, now)
	if err != nil {
		return nil, fmt.Errorf("weather warnings: pending: %w", err)
	}
	return scanWarnings(rows)
}

func (s *Store) markNotified(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE weather_warnings SET notified = true WHERE id = $1`, id)
	return err
}
