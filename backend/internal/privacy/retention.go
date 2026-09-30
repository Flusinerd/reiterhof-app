package privacy

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/scheduler"
)

// Retention periods, documented in docs/domains/privacy.md and docs/legal/datenschutz.md.
const (
	// PresenceMonths is how long finished presence visits are kept.
	PresenceMonths = 12
	// TrackMonths is how long the raw GPS track and the raw gait windows of a training
	// session are kept; the session with its duration, distance and gait shares stays.
	TrackMonths = 12
	// ReminderMonths is how long delivered reminders are kept.
	ReminderMonths = 12
)

// PruneResult counts what Prune removed.
type PruneResult struct {
	PresenceVisits int64
	Tracks         int64
	GaitWindows    int64
	Reminders      int64
	LoginTokens    int64
	AuthSessions   int64
}

// Prune deletes data past its retention period. Idempotent; safe to run at any time.
func Prune(ctx context.Context, pool *pgxpool.Pool, now time.Time) (PruneResult, error) {
	var res PruneResult
	steps := []struct {
		n    *int64
		sql  string
		args []any
	}{
		{&res.PresenceVisits, `DELETE FROM presence WHERE left_at IS NOT NULL AND left_at < $1`, []any{months(now, PresenceMonths)}},
		{&res.Tracks, `UPDATE sessions SET track = NULL WHERE track IS NOT NULL AND started_at < $1`, []any{months(now, TrackMonths)}},
		{&res.GaitWindows, `DELETE FROM gait_windows g USING sessions s WHERE s.id = g.session_id AND s.started_at < $1`, []any{months(now, TrackMonths)}},
		{&res.Reminders, `DELETE FROM reminders WHERE due_at < $1`, []any{months(now, ReminderMonths)}},
		{&res.LoginTokens, `DELETE FROM login_tokens WHERE expires_at < $1`, []any{now}},
		{&res.AuthSessions, `DELETE FROM auth_sessions WHERE expires_at < $1`, []any{now}},
	}
	for _, s := range steps {
		tag, err := pool.Exec(ctx, s.sql, s.args...)
		if err != nil {
			return res, err
		}
		*s.n = tag.RowsAffected()
	}
	return res, nil
}

// Job is the scheduler job that runs Prune daily at 03:30 Europe/Berlin.
func Job(pool *pgxpool.Pool, log *slog.Logger, now func() time.Time) scheduler.Job {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		loc = time.UTC
	}
	return scheduler.Job{
		Name:     "privacy-retention",
		Schedule: scheduler.DailyAt(3, 30, loc),
		Run: func(ctx context.Context) error {
			res, err := Prune(ctx, pool, now())
			if err == nil {
				log.Info("retention run", "presence_visits", res.PresenceVisits, "tracks", res.Tracks,
					"reminders", res.Reminders, "login_tokens", res.LoginTokens, "auth_sessions", res.AuthSessions)
			}
			return err
		},
	}
}
