package health

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
)

const (
	// DueReminderHour is the stable-local hour from which due-date reminders go out.
	DueReminderHour = 8
	// MedicationWindow is how long after daily_time a missed medication reminder is
	// still sent (e.g. after a restart). Later ones are skipped, not sent late.
	MedicationWindow = 3 * time.Hour
)

// DueOffsets are the days before the due date on which owner and riders are notified.
var DueOffsets = []int{7, 1}

// Reminders sends the health reminders (JAN-12).
//
//   - kind health_due: for items with a due date and no daily_time, 7 days and 1 day
//     before the due date, from 08:00 stable-local time on.
//   - kind medication: for items with a daily_time, every day from that time on
//     (stable-local), for up to MedicationWindow.
//
// Recipients are the owner and all riders of the horse. Every reminder is claimed by a
// row in reminders (unique per user, kind, item and scheduled time; see migration 0040),
// so running the job any number of times sends each reminder once. If the push fails,
// the claim is released and the next run tries again. Run it often (every 15 minutes):
// medication times are per item.
type Reminders struct {
	Pool   *pgxpool.Pool
	Notify httpx.Notifier
	Log    *slog.Logger
	Now    func() time.Time
}

// Run processes all stables. It keeps going after a failure in one stable and returns the first error.
func (s *Reminders) Run(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT id, timezone FROM stables`)
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
		loc, err := time.LoadLocation(st.tz)
		if err != nil {
			loc = time.UTC
		}
		if err := s.runStable(ctx, st.id, loc); err != nil {
			if s.Log != nil {
				s.Log.Error("health reminders failed", "stable", st.id, "err", err)
			}
			if first == nil {
				first = err
			}
		}
	}
	return first
}

type dueItem struct {
	id, horseID, horseName, kind, label string
	dueDate                             time.Time
	dailyTime                           *string
}

func (s *Reminders) runStable(ctx context.Context, stableID string, loc *time.Location) error {
	now := s.Now().In(loc)
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	rows, err := s.Pool.Query(ctx, `SELECT i.id, i.horse_id, h.name, i.kind, i.label, i.due_date, to_char(i.daily_time, 'HH24:MI')
		FROM health_items i JOIN horses h ON h.id = i.horse_id
		WHERE i.stable_id = $1 AND (i.due_date IS NOT NULL OR i.daily_time IS NOT NULL)`, stableID)
	if err != nil {
		return err
	}
	var items []dueItem
	for rows.Next() {
		var it dueItem
		var due *time.Time
		if err := rows.Scan(&it.id, &it.horseID, &it.horseName, &it.kind, &it.label, &due, &it.dailyTime); err != nil {
			rows.Close()
			return err
		}
		if due != nil {
			it.dueDate = time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, time.UTC)
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var first error
	for _, it := range items {
		var err error
		switch {
		case it.dailyTime != nil:
			err = s.medication(ctx, stableID, loc, now, it)
		case now.Hour() >= DueReminderHour:
			err = s.dueDate(ctx, stableID, loc, today, it)
		}
		if err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s *Reminders) dueDate(ctx context.Context, stableID string, loc *time.Location, today time.Time, it dueItem) error {
	if it.dueDate.IsZero() {
		return nil
	}
	days := int(it.dueDate.Sub(today).Hours() / 24)
	if !containsInt(DueOffsets, days) {
		return nil
	}
	// The scheduled time is what makes the claim unique: the same due date and offset is
	// the same reminder, a new due date (after "erledigt") is a new one.
	scheduled := time.Date(it.dueDate.Year(), it.dueDate.Month(), it.dueDate.Day()-days, DueReminderHour, 0, 0, 0, loc)
	when := "morgen"
	if days > 1 {
		when = fmt.Sprintf("in %d Tagen", days)
	}
	title := "Fällig: " + it.horseName
	body := fmt.Sprintf("%s: %s (%s).", it.label, when, it.dueDate.Format("02.01.2006"))
	return s.send(ctx, stableID, it, push.KindHealthDue, scheduled, title, body)
}

func (s *Reminders) medication(ctx context.Context, stableID string, loc *time.Location, now time.Time, it dueItem) error {
	var hh, mm int
	if _, err := fmt.Sscanf(*it.dailyTime, "%d:%d", &hh, &mm); err != nil {
		return nil
	}
	y, m, d := now.Date()
	scheduled := time.Date(y, m, d, hh, mm, 0, 0, loc)
	if now.Before(scheduled) || now.After(scheduled.Add(MedicationWindow)) {
		return nil
	}
	// A course with an end date (due_date) stops after that day.
	if !it.dueDate.IsZero() && time.Date(y, m, d, 0, 0, 0, 0, time.UTC).After(it.dueDate) {
		return nil
	}
	title := "Medikament: " + it.horseName
	body := fmt.Sprintf("%s, %s Uhr.", it.label, *it.dailyTime)
	return s.send(ctx, stableID, it, push.KindMedication, scheduled, title, body)
}

// send claims the reminder for every recipient that has not had it yet and notifies them.
func (s *Reminders) send(ctx context.Context, stableID string, it dueItem, kind string, scheduled time.Time, title, body string) error {
	recipients, err := s.recipients(ctx, stableID, it.horseID)
	if err != nil {
		return err
	}
	var claimed []string
	for _, uid := range recipients {
		tag, err := s.Pool.Exec(ctx, `INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'health_items', $7, $8)
			ON CONFLICT (user_id, kind, source_id, due_at) WHERE source_table = 'health_items' DO NOTHING`,
			stableID, uid, kind, title, body, scheduled, it.id, s.Now())
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
	data := map[string]any{"horse_id": it.horseID, "item_id": it.id, "route": "/horses/" + it.horseID + "/health", "screen": "/horses/" + it.horseID + "/health"}
	if err := s.Notify.NotifyUsers(ctx, stableID, claimed, kind, title, body, data); err != nil {
		// Release the claims so the next run retries.
		if _, derr := s.Pool.Exec(context.WithoutCancel(ctx), `DELETE FROM reminders
			WHERE source_table = 'health_items' AND source_id = $1 AND kind = $2 AND due_at = $3 AND user_id = ANY($4)`,
			it.id, kind, scheduled, claimed); derr != nil && s.Log != nil {
			s.Log.Error("release health reminder claims", "err", derr)
		}
		return err
	}
	return nil
}

// recipients are the owner and the riders of the horse.
func (s *Reminders) recipients(ctx context.Context, stableID, horseID string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT owner_id FROM horses WHERE id = $1 AND stable_id = $2 AND owner_id IS NOT NULL
		UNION SELECT user_id FROM horse_riders WHERE horse_id = $1 AND stable_id = $2`, horseID, stableID)
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
