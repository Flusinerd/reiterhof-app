package requests

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/scheduler"
)

// Jobs returns the background jobs of this package for cmd/api:
// the helper reminder (every 5 minutes) and the daily materialisation of
// recurring requests (03:30 Europe/Berlin, also once at start).
func (s *Service) Jobs() []scheduler.Job {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		loc = time.UTC
	}
	return []scheduler.Job{
		{Name: "request-helper-reminders", Schedule: scheduler.Every(5 * time.Minute), Run: s.SendHelperReminders},
		{Name: "request-recurring", Schedule: scheduler.DailyAt(3, 30, loc), RunOnStart: true, Run: s.MaterializeAll},
	}
}

type dueReminder struct {
	requestID string
	stableID  string
	helper    string
	remindAt  time.Time
}

// SendHelperReminders sends kind "helper" pushes to the helpers of requests whose
// remind_helper_at has passed. Each (request, helper) pair is claimed by inserting a
// row into reminders (unique index), so a reminder is never sent twice, even with
// overlapping runs.
func (s *Service) SendHelperReminders(ctx context.Context) error {
	now := s.now()
	rows, err := s.Pool.Query(ctx, `
		SELECT r.id::text, r.stable_id::text, a.user_id::text, r.remind_helper_at
		FROM requests r
		JOIN stables st ON st.id = r.stable_id
		JOIN request_assignees a ON a.request_id = r.id
		WHERE r.status IN ('open', 'assigned')
		  AND r.remind_helper_at IS NOT NULL AND r.remind_helper_at <= $1
		  AND ($1::timestamptz AT TIME ZONE st.timezone)::date <= COALESCE(r.date_end, r.date)
		  AND NOT EXISTS (SELECT 1 FROM reminders m
		                  WHERE m.source_table = 'requests' AND m.source_id = r.id AND m.user_id = a.user_id AND m.kind = 'helper')
		ORDER BY r.id, a.created_at`, now)
	if err != nil {
		return err
	}
	var due []dueReminder
	for rows.Next() {
		var d dueReminder
		if err := rows.Scan(&d.requestID, &d.stableID, &d.helper, &d.remindAt); err != nil {
			rows.Close()
			return err
		}
		due = append(due, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for i := 0; i < len(due); {
		j := i
		for j < len(due) && due[j].requestID == due[i].requestID {
			j++
		}
		group := due[i:j]
		i = j

		r, err := s.get(ctx, s.Pool, group[0].stableID, "", group[0].requestID, "")
		if err != nil {
			s.log().Error("requests: reminder load", "request", group[0].requestID, "err", err)
			continue
		}
		title, body := reminderText(r)
		var claimed []string
		for _, d := range group {
			tag, err := s.Pool.Exec(ctx, `
				INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at)
				VALUES ($1, $2::uuid, $3, $4, $5, $6, 'requests', $7::uuid, $8)
				ON CONFLICT DO NOTHING`, d.stableID, d.helper, push.KindHelper, title, body, d.remindAt, d.requestID, now)
			if err != nil {
				s.log().Error("requests: claim reminder", "request", d.requestID, "err", err)
				continue
			}
			if tag.RowsAffected() == 1 {
				claimed = append(claimed, d.helper)
			}
		}
		s.notify(ctx, group[0].stableID, claimed, push.KindHelper, title, body, pushData(r.ID))
	}
	return nil
}

// reminderText builds the German push title and body: when, where, checklist.
func reminderText(r Request) (title, body string) {
	title = "Erinnerung: " + r.title()
	parts := []string{whenText(r.Date, r.TimeFrom)}
	if r.Location != "" {
		parts = append(parts, r.Location)
	}
	if r.Type == TypeRideShare {
		var p RideSharePayload
		if json.Unmarshal(r.Payload, &p) == nil && p.Destination != "" {
			parts = append(parts, "Ziel: "+p.Destination)
		}
	}
	body = strings.Join(parts, " · ")
	if list := r.checklist(); len(list) > 0 {
		body += ". Checkliste: " + strings.Join(list, ", ")
	}
	return title, body
}
