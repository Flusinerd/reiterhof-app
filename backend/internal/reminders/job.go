package reminders

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/scheduler"
)

// TrainingPlanUntil is the stable-local hour after which the evening plan is no longer sent
// (a missed run is skipped, not sent late at night).
const TrainingPlanUntil = 22

// sourceTrainingPlan is reminders.source_table of the training plan claims
// (source_id = stable, due_at = 19:00 stable-local of the evening; unique index in migration 0120).
const sourceTrainingPlan = "training_plan"

// Service runs the background job of this package.
type Service struct {
	Pool   *pgxpool.Pool
	Notify httpx.Notifier
	Log    *slog.Logger
	Now    func() time.Time
}

// NewService builds the job runner from the API dependencies.
func NewService(deps httpx.Deps) *Service {
	return &Service{Pool: deps.Pool, Notify: deps.Notify, Log: deps.Log, Now: deps.Now}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// Jobs returns the scheduler jobs for cmd/api.
func (s *Service) Jobs() []scheduler.Job {
	return []scheduler.Job{
		{Name: "training-plan-reminders", Schedule: scheduler.Every(15 * time.Minute), RunOnStart: true, Run: s.RunTrainingPlan},
	}
}

// RunTrainingPlan sends the training plan of the next day to everybody who has planned
// week slots for tomorrow ("Trainingsplan am Vorabend", kind training_plan). From 19:00
// stable-local time until 22:00, once per user and evening: the run claims (user, kind,
// stable, evening) with a reminders row and only pushes when the claim is new; a failed push
// releases it so the next run retries. Users who switched the kind off are skipped (no row,
// so the reminder center does not list a push that was never meant for them).
func (s *Service) RunTrainingPlan(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT id::text, timezone FROM stables`)
	if err != nil {
		return err
	}
	type stableRow struct{ id, tz string }
	var stables []stableRow
	for rows.Next() {
		var st stableRow
		if err := rows.Scan(&st.id, &st.tz); err != nil {
			rows.Close()
			return err
		}
		stables = append(stables, st)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var errs []error
	for _, st := range stables {
		loc, err := time.LoadLocation(st.tz)
		if err != nil {
			loc = time.UTC
		}
		now := s.now().In(loc)
		if now.Hour() < TrainingPlanHour || now.Hour() >= TrainingPlanUntil {
			continue
		}
		if err := s.trainingPlanStable(ctx, st.id, now); err != nil {
			s.log().Error("reminders: training plan", "stable", st.id, "err", err)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Service) trainingPlanStable(ctx context.Context, stableID string, now time.Time) error {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	evening := time.Date(now.Year(), now.Month(), now.Day(), TrainingPlanHour, 0, 0, 0, loc)
	tomorrow := today.AddDate(0, 0, 1)

	rows, err := s.Pool.Query(ctx, `
		SELECT ws.user_id::text, h.name, COALESCE(ws.activity, '')
		FROM week_slots ws
		JOIN horses h ON h.id = ws.horse_id
		JOIN users u ON u.id = ws.user_id AND u.stable_id = ws.stable_id
		WHERE ws.stable_id = $1 AND ws.day = $2::date AND ws.status = 'planned' AND ws.user_id IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM reminder_settings rs
		                  WHERE rs.user_id = ws.user_id AND rs.kind = $3 AND NOT rs.enabled)
		ORDER BY ws.user_id, h.name`, stableID, pgDate(tomorrow), push.KindTrainingPlan)
	if err != nil {
		return err
	}
	var order []string
	byUser := map[string][]slotText{}
	for rows.Next() {
		var uid string
		var st slotText
		if err := rows.Scan(&uid, &st.horse, &st.activity); err != nil {
			rows.Close()
			return err
		}
		if _, ok := byUser[uid]; !ok {
			order = append(order, uid)
		}
		byUser[uid] = append(byUser[uid], st)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	title := trainingTitle(tomorrow, today)
	var errs []error
	for _, uid := range order {
		body := trainingBody(byUser[uid])
		tag, err := s.Pool.Exec(ctx, `
			INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at)
			VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $1::uuid, $8)
			ON CONFLICT (user_id, kind, source_id, due_at) WHERE source_table = 'training_plan' DO NOTHING`,
			stableID, uid, push.KindTrainingPlan, title, body, evening, sourceTrainingPlan, s.now())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if tag.RowsAffected() == 0 {
			continue
		}
		if err := s.Notify.NotifyUsers(ctx, stableID, []string{uid}, push.KindTrainingPlan, title, body,
			map[string]any{"screen": screenWeek}); err != nil {
			if _, derr := s.Pool.Exec(context.WithoutCancel(ctx), `
				DELETE FROM reminders WHERE user_id = $1::uuid AND kind = $2 AND source_table = $3
				  AND source_id = $4::uuid AND due_at = $5`,
				uid, push.KindTrainingPlan, sourceTrainingPlan, stableID, evening); derr != nil {
				s.log().Error("reminders: release training plan claim", "err", derr)
			}
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
