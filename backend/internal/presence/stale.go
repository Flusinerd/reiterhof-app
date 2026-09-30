package presence

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/scheduler"
)

// CloseStale closes visits that have been open for longer than maxAge (people forget to
// check out and the phone may miss the geofence exit). The visit ends at
// arrived_at + maxAge, not "now", so that "zuletzt gesehen" is not inflated by hours nobody
// witnessed. It publishes presence.changed for every affected stable and returns the number
// of closed visits.
func CloseStale(ctx context.Context, pool *pgxpool.Pool, now time.Time, maxAge time.Duration) (int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `UPDATE presence SET left_at = arrived_at + $2::interval
		WHERE left_at IS NULL AND arrived_at < $1::timestamptz - $2::interval
		RETURNING stable_id`, now, maxAge)
	if err != nil {
		return 0, err
	}
	stables := map[string]bool{}
	n := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		stables[id] = true
		n++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for id := range stables {
		if err := realtime.Publish(ctx, tx, id, EventChanged, nil); err != nil {
			return 0, err
		}
	}
	return n, tx.Commit(ctx)
}

// StaleJob is the scheduler job that runs CloseStale every 15 minutes (StaleAfter = 12 h).
func StaleJob(pool *pgxpool.Pool, log *slog.Logger, now func() time.Time) scheduler.Job {
	return scheduler.Job{
		Name:       "presence-close-stale",
		Schedule:   scheduler.Every(15 * time.Minute),
		RunOnStart: true,
		Run: func(ctx context.Context) error {
			n, err := CloseStale(ctx, pool, now(), StaleAfter)
			if err == nil && n > 0 {
				log.Info("closed stale visits", "count", n)
			}
			return err
		},
	}
}
