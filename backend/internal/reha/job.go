package reha

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

const (
	// CheckupReminderHour is the stable-local hour from which the checkup reminders go out
	// ("the morning of the checkup").
	CheckupReminderHour = 8
)

// CheckupOffsets are the days before checkup_date on which owner and riders are notified
// (0 = the morning of the day itself).
var CheckupOffsets = []int{2, 0}

// Reminders sends the vet checkup reminders (JAN-66, kind reha_checkup) for active plans with
// a checkup_date: 2 days before and on the morning of the day, from 08:00 stable-local time on.
//
// Recipients are the owner and all riders of the horse. Every reminder is claimed by a row in
// reminders (unique per user, kind, plan and scheduled time; migration 0100), so running the job
// any number of times sends each reminder once. If the push fails the claim is released and the
// next run tries again. A new checkup_date is a new scheduled time and therefore reminds again.
type Reminders struct {
	Pool   *pgxpool.Pool
	Notify httpx.Notifier
	Log    *slog.Logger
	Now    func() time.Time
}

// Run processes all stables; it keeps going after a failure and returns the first error.
func (s *Reminders) Run(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT id::text, timezone FROM stables`)
	if err != nil {
		return err
	}
	type stable struct{ id, tz string }
	var stables []stable
	for rows.Next() {
		var st stable
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
	var first error
	for _, st := range stables {
		if err := s.runStable(ctx, st.id, LoadLocation(st.tz)); err != nil {
			if s.Log != nil {
				s.Log.Error("reha reminders failed", "stable", st.id, "err", err)
			}
			if first == nil {
				first = err
			}
		}
	}
	return first
}

type checkup struct {
	planID, horseID, horseName, vet string
	date                            time.Time
}

func (s *Reminders) runStable(ctx context.Context, stableID string, loc *time.Location) error {
	now := s.Now().In(loc)
	if now.Hour() < CheckupReminderHour {
		return nil
	}
	today := training.Day(now)
	rows, err := s.Pool.Query(ctx, `SELECT p.id::text, p.horse_id::text, h.name, COALESCE(p.vet, ''), p.checkup_date
		FROM reha_plans p JOIN horses h ON h.id = p.horse_id
		WHERE p.stable_id = $1 AND p.active AND p.checkup_date IS NOT NULL`, stableID)
	if err != nil {
		return err
	}
	var list []checkup
	for rows.Next() {
		var c checkup
		if err := rows.Scan(&c.planID, &c.horseID, &c.horseName, &c.vet, &c.date); err != nil {
			rows.Close()
			return err
		}
		c.date = training.Day(c.date)
		list = append(list, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var first error
	for _, c := range list {
		days := training.DaysBetween(today, c.date)
		if !containsInt(CheckupOffsets, days) {
			continue
		}
		if err := s.send(ctx, stableID, loc, c, days); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s *Reminders) send(ctx context.Context, stableID string, loc *time.Location, c checkup, days int) error {
	// The scheduled time identifies the reminder: the same checkup date and offset is the same
	// reminder, a moved checkup date is a new one.
	scheduled := time.Date(c.date.Year(), c.date.Month(), c.date.Day()-days, CheckupReminderHour, 0, 0, 0, loc)
	when := "heute"
	if days > 0 {
		when = fmt.Sprintf("in %d Tagen", days)
	}
	title := "Reha-Kontrolle: " + c.horseName
	body := fmt.Sprintf("Kontrolltermin %s (%s).", when, c.date.Format("02.01.2006"))
	if c.vet != "" {
		body = fmt.Sprintf("Kontrolltermin bei %s %s (%s).", c.vet, when, c.date.Format("02.01.2006"))
	}

	recipients, err := s.recipients(ctx, stableID, c.horseID)
	if err != nil {
		return err
	}
	var claimed []string
	for _, uid := range recipients {
		tag, err := s.Pool.Exec(ctx, `INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at)
			VALUES ($1, $2::uuid, $3, $4, $5, $6, 'reha_plans', $7::uuid, $8)
			ON CONFLICT (user_id, kind, source_id, due_at) WHERE source_table = 'reha_plans' DO NOTHING`,
			stableID, uid, push.KindRehaCheckup, title, body, scheduled, c.planID, s.Now())
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
	data := map[string]any{"horse_id": c.horseID, "plan_id": c.planID, "route": "/horses/" + c.horseID + "/reha", "screen": "/horses/" + c.horseID + "/reha"}
	if err := s.Notify.NotifyUsers(ctx, stableID, claimed, push.KindRehaCheckup, title, body, data); err != nil {
		// Release the claims so that the next run tries again.
		if _, derr := s.Pool.Exec(context.WithoutCancel(ctx), `DELETE FROM reminders
			WHERE source_table = 'reha_plans' AND source_id = $1::uuid AND kind = $2 AND due_at = $3 AND user_id = ANY($4::uuid[])`,
			c.planID, push.KindRehaCheckup, scheduled, claimed); derr != nil && s.Log != nil {
			s.Log.Error("release reha reminder claims", "err", derr)
		}
		return err
	}
	return nil
}

// recipients are the owner and the riders of the horse.
func (s *Reminders) recipients(ctx context.Context, stableID, horseID string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT owner_id::text FROM horses WHERE id = $1::uuid AND stable_id = $2 AND owner_id IS NOT NULL
		UNION SELECT user_id::text FROM horse_riders WHERE horse_id = $1::uuid AND stable_id = $2`, horseID, stableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
