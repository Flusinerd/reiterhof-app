package weather

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/stables"
)

// WarningService is the warning job: for every stable with coordinates it fetches the
// active DWD warnings, stores them (ReplaceWarnings) and pushes new relevant ones
// (Warning.Notify) to all members of the stable, once per warning and member (claim row
// in reminders, source_table weather_warnings). A failed push releases the claims and
// is retried on the next run.
type WarningService struct {
	Pool    *pgxpool.Pool
	Fetcher WarningFetcher
	Notify  httpx.Notifier
	Log     *slog.Logger
	Now     func() time.Time
}

// sourceWarnings is reminders.source_table of the warning push.
const sourceWarnings = "weather_warnings"

// Refresh runs the job once. A failing stable does not stop the others; the errors are joined.
func (s *WarningService) Refresh(ctx context.Context) error {
	located, err := stables.ListLocated(ctx, s.Pool)
	if err != nil {
		return err
	}
	var errs []error
	for _, st := range located {
		if err := s.refreshStable(ctx, st); err != nil {
			errs = append(errs, fmt.Errorf("stable %s: %w", st.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (s *WarningService) refreshStable(ctx context.Context, st stables.Located) error {
	loc, err := time.LoadLocation(st.Timezone)
	if err != nil {
		loc = time.UTC
	}
	ws, err := s.Fetcher.FetchWarnings(ctx, st.Lat, st.Lng)
	if err != nil {
		return err
	}
	now := s.Now()
	active := ws[:0:0]
	for _, w := range ws {
		if w.Expires == nil || w.Expires.After(now) {
			active = append(active, w)
		}
	}
	store := &Store{Pool: s.Pool}
	if err := store.ReplaceWarnings(ctx, st.ID, active, now); err != nil {
		return err
	}
	pending, err := store.pendingWarnings(ctx, st.ID, now)
	if err != nil {
		return err
	}
	var errs []error
	for _, w := range pending {
		if w.Warning().Notify() {
			if err := s.send(ctx, st.ID, loc, w); err != nil {
				errs = append(errs, err)
				continue
			}
		}
		if err := store.markNotified(ctx, w.ID); err != nil {
			errs = append(errs, err)
		}
	}
	if len(active) > 0 {
		s.log().Info("weather warnings stored", "stable", st.ID, "count", len(active))
	}
	return errors.Join(errs...)
}

// WarningPush returns title and body of the push for a warning, times in loc.
func WarningPush(w StoredWarning, loc *time.Location) (title, body string) {
	title = "Wetterwarnung"
	if SeverityRank(w.Severity) >= SeverityRank(SeveritySevere) {
		title = "Unwetterwarnung"
	}
	return title, w.Headline + ": " + WarningSpan(w.Onset, w.Expires, loc) + "."
}

var weekdaysShort = []string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}

// WarningSpan is the German time range of a warning, e.g. "Sa 24.10., 14:00 bis 22:00 Uhr",
// "Sa 24.10., 23:00 bis So 25.10., 08:00 Uhr" or "ab Sa 24.10., 14:00 Uhr" without an end.
func WarningSpan(onset time.Time, expires *time.Time, loc *time.Location) string {
	day := func(t time.Time) string { return weekdaysShort[t.Weekday()] + " " + t.Format("02.01.") }
	on := onset.In(loc)
	if expires == nil {
		return "ab " + day(on) + ", " + on.Format("15:04") + " Uhr"
	}
	ex := expires.In(loc)
	if ex.Format("2006-01-02") == on.Format("2006-01-02") {
		return day(on) + ", " + on.Format("15:04") + " bis " + ex.Format("15:04") + " Uhr"
	}
	return day(on) + ", " + on.Format("15:04") + " bis " + day(ex) + ", " + ex.Format("15:04") + " Uhr"
}

func (s *WarningService) send(ctx context.Context, stableID string, loc *time.Location, w StoredWarning) error {
	title, body := WarningPush(w, loc)
	rows, err := s.Pool.Query(ctx, `SELECT id::text FROM users WHERE stable_id = $1 AND deleted_at IS NULL ORDER BY id`, stableID)
	if err != nil {
		return err
	}
	var members []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		members = append(members, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	now := s.Now()
	var claimed []string
	for _, uid := range members {
		tag, err := s.Pool.Exec(ctx, `INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (user_id, source_id) WHERE source_table = 'weather_warnings' DO NOTHING`,
			stableID, uid, push.KindWeatherWarning, title, body, now, sourceWarnings, w.ID, now)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 1 {
			claimed = append(claimed, uid)
		}
	}
	if len(claimed) == 0 {
		return nil
	}
	data := map[string]any{"warning_id": w.ID, "screen": "/blankets"}
	if err := s.Notify.NotifyUsers(ctx, stableID, claimed, push.KindWeatherWarning, title, body, data); err != nil {
		if _, derr := s.Pool.Exec(context.WithoutCancel(ctx), `DELETE FROM reminders
			WHERE source_table = 'weather_warnings' AND source_id = $1 AND user_id = ANY($2)`, w.ID, claimed); derr != nil {
			s.log().Error("release weather warning claims", "err", derr)
		}
		return err
	}
	return nil
}

func (s *WarningService) log() *slog.Logger {
	if s.Log == nil {
		return slog.Default()
	}
	return s.Log
}
