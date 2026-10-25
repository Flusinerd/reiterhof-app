package trainingapi

import (
	"context"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
)

// Periodisation and return to work need more history than the 14 days "Was heute?" looks at: the
// block walks the weeks since cycle_start, the return to work looks for a break in the last 60 days.

// restartLookbackDays is how far back the sessions are read to find a break.
const restartLookbackDays = 60

// sessionDays returns the stable-local day of every session of the horse from the given day on
// (one entry per session, oldest first).
func (h *handler) sessionDays(ctx context.Context, stableID, horseID string, from time.Time, loc *time.Location) ([]time.Time, error) {
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT (started_at AT TIME ZONE $4)::date FROM sessions
		WHERE stable_id = $1 AND horse_id = $2 AND started_at >= $3 ORDER BY started_at`,
		stableID, horseID, localMidnight(from, loc), loc.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var days []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		days = append(days, d)
	}
	return days, rows.Err()
}

// blockOf returns the place of the week starting on the Monday start in the horse's four-week
// blocks; the zero Block without a cycle_start.
func (h *handler) blockOf(ctx context.Context, stableID, horseID string, prof storedProfile, start, today time.Time, loc *time.Location) (recommend.Block, error) {
	if prof.cycleStart == nil {
		return recommend.Block{}, nil
	}
	days, err := h.sessionDays(ctx, stableID, horseID, *prof.cycleStart, loc)
	if err != nil {
		return recommend.Block{}, err
	}
	return recommend.BlockFor(prof.cycleStart, start, today, days, prof.rhythm.SessionsMin), nil
}

// restartAnchor returns the first day of the horse's return to work (nil when it is not
// returning from a break): see recommend.RestartAnchor.
func (h *handler) restartAnchor(ctx context.Context, stableID, horseID string, today time.Time, loc *time.Location) (*time.Time, error) {
	from := today.AddDate(0, 0, -restartLookbackDays)
	days, err := h.sessionDays(ctx, stableID, horseID, from, loc)
	if err != nil {
		return nil, err
	}
	// The last session before the window shows whether the horse had a break at its start.
	var last *time.Time
	if err := h.deps.Pool.QueryRow(ctx, `
		SELECT max((started_at AT TIME ZONE $3)::date) FROM sessions
		WHERE stable_id = $1 AND horse_id = $2 AND started_at < $4`,
		stableID, horseID, loc.String(), localMidnight(from, loc)).Scan(&last); err != nil {
		return nil, err
	}
	if last != nil {
		days = append(days, *last)
	}
	return recommend.RestartAnchor(days, today), nil
}
