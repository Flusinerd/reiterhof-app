// Package requests implements help requests to the whole stable (M4): create,
// list, accept ("Mach ich"), withdraw, done, cancel, recurring series, helper
// reminders, calendar export (ICS) and the optional thank-you counter.
//
// See docs/domains/requests.md.
package requests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
)

// Event describes a change for realtime listeners. Kind is one of created,
// updated, accepted, withdrawn, done, cancelled.
type Event struct {
	StableID  string
	RequestID string
	Kind      string
}

// Service holds the business logic; the HTTP handler and the scheduler jobs
// share it.
type Service struct {
	Pool   *pgxpool.Pool
	Notify httpx.Notifier
	Log    *slog.Logger
	Now    func() time.Time
	// Publish is the realtime extension point: it is called after every committed
	// change (best effort, must not block). Nil means no-op.
	Publish func(ctx context.Context, e Event)
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now()
	}
	return s.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log == nil {
		return slog.Default()
	}
	return s.Log
}

func (s *Service) publish(ctx context.Context, stableID, requestID, kind string) {
	if s.Publish != nil {
		s.Publish(ctx, Event{StableID: stableID, RequestID: requestID, Kind: kind})
	}
}

func (s *Service) notify(ctx context.Context, stableID string, users []string, kind, title, body string, data map[string]any) {
	if s.Notify == nil || len(users) == 0 {
		return
	}
	if err := s.Notify.NotifyUsers(context.WithoutCancel(ctx), stableID, users, kind, title, body, data); err != nil {
		s.log().Error("requests: notify", "kind", kind, "err", err)
	}
}

func pushData(id string) map[string]any {
	return map[string]any{"request_id": id, "screen": "/requests/" + id}
}

func (s *Service) tx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func internal(err error) error { return fmt.Errorf("requests: %w", err) }

// ---------------------------------------------------------------- reading

// ListFilter is the parsed query of GET /api/v1/requests.
type ListFilter = listFilter

// List returns requests of the user's stable and the number of open ones.
func (s *Service) List(ctx context.Context, u auth.User, f ListFilter) ([]Request, int, error) {
	loc := stableLocation(ctx, s.Pool, u.StableID)
	td := today(s.now(), loc)
	if f.Mine {
		f.CreatedBy = u.ID
	}
	if f.Assigned {
		f.AssignedTo = u.ID
	}
	out, err := s.list(ctx, s.Pool, u.StableID, u.ID, f, td)
	if err != nil {
		return nil, 0, internal(err)
	}
	if out == nil {
		out = []Request{}
	}
	var open int
	err = s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM requests
		WHERE stable_id = $1 AND status = 'open' AND COALESCE(date_end, date) >= $2::date`, u.StableID, td).Scan(&open)
	if err != nil {
		return nil, 0, internal(err)
	}
	return out, open, nil
}

// Get returns one request.
func (s *Service) Get(ctx context.Context, u auth.User, id string) (Request, error) {
	loc := stableLocation(ctx, s.Pool, u.StableID)
	r, err := s.get(ctx, s.Pool, u.StableID, u.ID, id, today(s.now(), loc))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, errMissing
	}
	if err != nil {
		return Request{}, internal(err)
	}
	return r, nil
}

// Calendar returns the upcoming requests the user helps with.
func (s *Service) Calendar(ctx context.Context, u auth.User, from, to string) ([]Request, error) {
	loc := stableLocation(ctx, s.Pool, u.StableID)
	td := today(s.now(), loc)
	if from == "" {
		from = td
	}
	out, err := s.list(ctx, s.Pool, u.StableID, u.ID, listFilter{
		Statuses: []string{StatusOpen, StatusAssigned}, AssignedTo: u.ID, From: from, To: to, Limit: 200,
	}, td)
	if err != nil {
		return nil, internal(err)
	}
	if out == nil {
		out = []Request{}
	}
	return out, nil
}

// ---------------------------------------------------------------- create

const insertRequest = `
INSERT INTO requests (stable_id, type, horse_id, created_by, date, date_end, time_from, time_to, location, description,
                      tasks, helpers_needed, recurring_rule, remind_helper_at, payload)
VALUES ($1, $2, $3::uuid, $4::uuid, $5::date, $6::date, $7::time, $8::time, NULLIF($9, ''), NULLIF($10, ''),
        $11::jsonb, $12, NULLIF($13, ''), $14, $15::jsonb)
RETURNING id::text`

func (s *Service) checkHorse(ctx context.Context, q querier, stableID string, horseID *string) error {
	if horseID == nil {
		return nil
	}
	var ok bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM horses WHERE id = $1::uuid AND stable_id = $2)`, *horseID, stableID).Scan(&ok); err != nil {
		return internal(err)
	}
	if !ok {
		return invalid("horse_id: horse not found in this stable")
	}
	return nil
}

// Create stores a request and notifies the stable (kind new_request, opt-in).
func (s *Service) Create(ctx context.Context, u auth.User, in Input) (Request, error) {
	loc := stableLocation(ctx, s.Pool, u.StableID)
	td := today(s.now(), loc)
	if err := in.normalize(td, true); err != nil {
		return Request{}, err
	}
	if err := s.checkHorse(ctx, s.Pool, u.StableID, in.HorseID); err != nil {
		return Request{}, err
	}
	if err := s.fillRules(ctx, s.Pool, u.StableID, &in); err != nil {
		return Request{}, err
	}
	tasks, _ := json.Marshal(in.Tasks)
	var id string
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, insertRequest, u.StableID, in.Type, in.HorseID, u.ID, in.Date, in.DateEnd,
			in.TimeFrom, in.TimeTo, in.Location, in.Description, string(tasks), in.HelpersNeeded, in.RecurringRule,
			in.RemindHelperAt, string(in.Payload)).Scan(&id); err != nil {
			return err
		}
		if in.RecurringRule != "" {
			_, err := tx.Exec(ctx, `UPDATE requests SET series_id = id WHERE id = $1::uuid`, id)
			return err
		}
		return nil
	})
	if err != nil {
		return Request{}, internal(err)
	}
	if in.RecurringRule != "" {
		if err := s.materializeSeries(ctx, s.Pool, u.StableID, id, td); err != nil {
			s.log().Error("requests: materialize new series", "err", err)
		}
	}
	r, err := s.Get(ctx, u, id)
	if err != nil {
		return Request{}, err
	}
	// Requests go to everyone in the stable, present or not.
	rows, err := s.Pool.Query(ctx, `SELECT id::text FROM users WHERE stable_id = $1 AND id <> $2::uuid`, u.StableID, u.ID)
	if err == nil {
		var ids []string
		for rows.Next() {
			var uid string
			if rows.Scan(&uid) == nil {
				ids = append(ids, uid)
			}
		}
		rows.Close()
		body := u.Name + " sucht Hilfe · " + whenText(r.Date, r.TimeFrom)
		if r.Location != "" {
			body += " · " + r.Location
		}
		s.notify(ctx, u.StableID, ids, push.KindNewRequest, r.title(), body, pushData(id))
	}
	s.publish(ctx, u.StableID, id, "created")
	return r, nil
}

// ---------------------------------------------------------------- update

func (s *Service) canManage(u auth.User, createdBy string) bool {
	return u.IsAdmin || u.ID == createdBy
}

func statusFor(helpers, needed int) string {
	if helpers >= needed {
		return StatusAssigned
	}
	return StatusOpen
}

// Update changes a request (creator or admin, only while open or assigned).
func (s *Service) Update(ctx context.Context, u auth.User, id string, p Patch) (Request, error) {
	if p.Scope == "series" {
		return s.updateSeries(ctx, u, id, p)
	}
	if p.Scope != "" && p.Scope != "one" {
		return Request{}, invalid("scope must be one or series")
	}
	loc := stableLocation(ctx, s.Pool, u.StableID)
	td := today(s.now(), loc)
	var notifyUsers []string
	var changedWhen bool
	var after Request
	err := s.tx(ctx, func(tx pgx.Tx) error {
		l, err := lock(ctx, tx, u.StableID, id)
		if errors.Is(err, errNotFound) {
			return errMissing
		}
		if err != nil {
			return err
		}
		if !s.canManage(u, l.CreatedBy) {
			return errForbidden
		}
		if l.Status != StatusOpen && l.Status != StatusAssigned {
			return errf(http.StatusConflict, "not_open", "the request is already %s", l.Status)
		}
		cur, err := s.get(ctx, tx, u.StableID, u.ID, id, td)
		if err != nil {
			return err
		}
		in := inputFromRequest(cur)
		if p.RecurringRule != nil && strings.TrimSpace(*p.RecurringRule) != cur.recurringRuleOrEmpty() {
			return invalid("recurring_rule can only be changed with scope series")
		}
		if err := p.apply(&in); err != nil {
			return invalid("%s", err.Error())
		}
		// A date that did not change may lie in the past (request already started).
		if err := in.normalize(td, in.Date != cur.Date); err != nil {
			return err
		}
		if err := s.checkHorse(ctx, tx, u.StableID, in.HorseID); err != nil {
			return err
		}
		if err := s.fillRules(ctx, tx, u.StableID, &in); err != nil {
			return err
		}
		if in.HelpersNeeded < l.Helpers {
			return errf(http.StatusConflict, "too_many_helpers", "%d helpers already joined; ask them to withdraw first", l.Helpers)
		}
		tasks, _ := json.Marshal(in.Tasks)
		_, err = tx.Exec(ctx, `
			UPDATE requests SET horse_id = $3::uuid, date = $4::date, date_end = $5::date, time_from = $6::time,
			       time_to = $7::time, location = NULLIF($8, ''), description = NULLIF($9, ''), tasks = $10::jsonb,
			       helpers_needed = $11, payload = $12::jsonb, remind_helper_at = $13, status = $14
			WHERE stable_id = $1 AND id = $2::uuid`,
			u.StableID, id, in.HorseID, in.Date, in.DateEnd, in.TimeFrom, in.TimeTo, in.Location, in.Description,
			string(tasks), in.HelpersNeeded, string(in.Payload), in.RemindHelperAt, statusFor(l.Helpers, in.HelpersNeeded))
		if err != nil {
			if isUniqueViolation(err) {
				return errf(http.StatusConflict, "conflict", "another request of this series already exists on that day")
			}
			return err
		}
		changedWhen = in.Date != cur.Date || strEq(in.TimeFrom, cur.TimeFrom) == false || in.Location != cur.Location ||
			!ptrEq(in.RemindHelperAt, cur.RemindHelperAt)
		if !ptrEq(in.RemindHelperAt, cur.RemindHelperAt) || in.Date != cur.Date {
			// New reminder time: allow the job to send again.
			if _, err := tx.Exec(ctx, `DELETE FROM reminders WHERE source_table = 'requests' AND source_id = $1::uuid AND kind = 'helper'`, id); err != nil {
				return err
			}
		}
		for _, h := range cur.Helpers {
			notifyUsers = append(notifyUsers, h.UserID)
		}
		after, err = s.get(ctx, tx, u.StableID, u.ID, id, td)
		return err
	})
	if err != nil {
		return Request{}, wrap(err)
	}
	if changedWhen {
		s.notify(ctx, u.StableID, notifyUsers, push.KindHelper, after.title(),
			"Geändert: "+whenText(after.Date, after.TimeFrom)+orEmpty(" · ", after.Location), pushData(id))
	}
	s.publish(ctx, u.StableID, id, "updated")
	return after, nil
}

func (r Request) recurringRuleOrEmpty() string {
	if r.RecurringRule == nil {
		return ""
	}
	return *r.RecurringRule
}

func strEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func ptrEq(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func orEmpty(prefix, s string) string {
	if s == "" {
		return ""
	}
	return prefix + s
}

// wrap keeps apiErrors and wraps everything else as an internal error.
func wrap(err error) error {
	if err == nil {
		return nil
	}
	var ae *apiError
	if errors.As(err, &ae) {
		return ae
	}
	return internal(err)
}

// ---------------------------------------------------------------- state transitions

// Accept adds the user as helper ("Mach ich" / "Ich komme mit"). It is
// idempotent, locks the row so that two people cannot take the last seat, and
// moves the request to assigned when it is full.
func (s *Service) Accept(ctx context.Context, u auth.User, id string) (Request, error) {
	loc := stableLocation(ctx, s.Pool, u.StableID)
	now := s.now()
	td := today(now, loc)
	var joined bool
	var after Request
	err := s.tx(ctx, func(tx pgx.Tx) error {
		l, err := lock(ctx, tx, u.StableID, id)
		if errors.Is(err, errNotFound) {
			return errMissing
		}
		if err != nil {
			return err
		}
		if l.CreatedBy == u.ID {
			return errf(http.StatusForbidden, "own_request", "you cannot accept your own request")
		}
		var already bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM request_assignees WHERE request_id = $1::uuid AND user_id = $2::uuid)`, id, u.ID).Scan(&already); err != nil {
			return err
		}
		if !already {
			if l.Status != StatusOpen && l.Status != StatusAssigned {
				return errf(http.StatusConflict, "not_open", "the request is already %s", l.Status)
			}
			if l.EndDate < td {
				return errf(http.StatusConflict, "expired", "the request lies in the past")
			}
			if l.Helpers >= l.HelpersNeeded {
				return errf(http.StatusConflict, "request_full", "all helpers have been found already")
			}
			if _, err := tx.Exec(ctx, `INSERT INTO request_assignees (stable_id, request_id, user_id) VALUES ($1, $2::uuid, $3::uuid)`, u.StableID, id, u.ID); err != nil {
				return err
			}
			joined = true
			remind := l.RemindHelperAt
			if remind == nil {
				// Default: the day before at 18:00, only if that lies ahead.
				if def, err := defaultReminder(l.Date, loc); err == nil && def.After(now) {
					remind = &def
					if _, err := tx.Exec(ctx, `UPDATE requests SET remind_helper_at = $2 WHERE id = $1::uuid`, id, def); err != nil {
						return err
					}
				}
			}
			if remind != nil && !remind.After(now) {
				// The reminder time has passed: do not remind someone who just said yes.
				if _, err := tx.Exec(ctx, `
					INSERT INTO reminders (stable_id, user_id, kind, title, due_at, source_table, source_id, sent_at)
					VALUES ($1, $2::uuid, 'helper', 'Anfrage angenommen', $3, 'requests', $4::uuid, $5)
					ON CONFLICT DO NOTHING`, u.StableID, u.ID, *remind, id, now); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(ctx, `UPDATE requests SET status = $2 WHERE id = $1::uuid`, id, statusFor(l.Helpers+1, l.HelpersNeeded)); err != nil {
				return err
			}
		}
		after, err = s.get(ctx, tx, u.StableID, u.ID, id, td)
		return err
	})
	if err != nil {
		return Request{}, wrap(err)
	}
	if joined {
		body := fmt.Sprintf("%s · %d/%d Helfer", whenText(after.Date, after.TimeFrom), after.HelpersCount, after.HelpersNeeded)
		s.notify(ctx, u.StableID, []string{after.CreatedBy}, push.KindHelper, u.Name+" hilft mit", after.title()+" · "+body, pushData(id))
		s.publish(ctx, u.StableID, id, "accepted")
	}
	return after, nil
}

// Withdraw removes the user as helper; a full request becomes open again.
func (s *Service) Withdraw(ctx context.Context, u auth.User, id string) (Request, error) {
	td := today(s.now(), stableLocation(ctx, s.Pool, u.StableID))
	var left bool
	var after Request
	err := s.tx(ctx, func(tx pgx.Tx) error {
		l, err := lock(ctx, tx, u.StableID, id)
		if errors.Is(err, errNotFound) {
			return errMissing
		}
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM request_assignees WHERE request_id = $1::uuid AND user_id = $2::uuid`, id, u.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			if l.Status == StatusDone || l.Status == StatusCancelled {
				return errf(http.StatusConflict, "not_open", "the request is already %s", l.Status)
			}
			left = true
			if _, err := tx.Exec(ctx, `UPDATE requests SET status = $2 WHERE id = $1::uuid`, id, statusFor(l.Helpers-1, l.HelpersNeeded)); err != nil {
				return err
			}
			// A later accept may remind again.
			if _, err := tx.Exec(ctx, `DELETE FROM reminders WHERE source_table = 'requests' AND source_id = $1::uuid AND user_id = $2::uuid AND kind = 'helper'`, id, u.ID); err != nil {
				return err
			}
		}
		after, err = s.get(ctx, tx, u.StableID, u.ID, id, td)
		return err
	})
	if err != nil {
		return Request{}, wrap(err)
	}
	if left {
		s.notify(ctx, u.StableID, []string{after.CreatedBy}, push.KindHelper, u.Name+" hilft nicht mehr",
			after.title()+" · "+whenText(after.Date, after.TimeFrom)+" · wieder offen", pushData(id))
		s.publish(ctx, u.StableID, id, "withdrawn")
	}
	return after, nil
}

// Done marks the request as done (creator or helper).
func (s *Service) Done(ctx context.Context, u auth.User, id string) (Request, error) {
	td := today(s.now(), stableLocation(ctx, s.Pool, u.StableID))
	var changed bool
	var after Request
	err := s.tx(ctx, func(tx pgx.Tx) error {
		l, err := lock(ctx, tx, u.StableID, id)
		if errors.Is(err, errNotFound) {
			return errMissing
		}
		if err != nil {
			return err
		}
		var helper bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM request_assignees WHERE request_id = $1::uuid AND user_id = $2::uuid)`, id, u.ID).Scan(&helper); err != nil {
			return err
		}
		if l.CreatedBy != u.ID && !helper {
			return errf(http.StatusForbidden, "forbidden", "only the creator or a helper can mark the request as done")
		}
		switch l.Status {
		case StatusDone:
		case StatusCancelled:
			return errf(http.StatusConflict, "not_open", "the request was cancelled")
		default:
			if _, err := tx.Exec(ctx, `UPDATE requests SET status = 'done' WHERE id = $1::uuid`, id); err != nil {
				return err
			}
			changed = true
		}
		after, err = s.get(ctx, tx, u.StableID, u.ID, id, td)
		return err
	})
	if err != nil {
		return Request{}, wrap(err)
	}
	if changed {
		s.publish(ctx, u.StableID, id, "done")
	}
	return after, nil
}

// Cancel cancels one request or, with scope "series", all upcoming ones of its
// series and ends the recurrence. Creator or admin. Helpers get a push.
func (s *Service) Cancel(ctx context.Context, u auth.User, id, scope string) (Request, error) {
	if scope != "" && scope != "one" && scope != "series" {
		return Request{}, invalid("scope must be one or series")
	}
	loc := stableLocation(ctx, s.Pool, u.StableID)
	td := today(s.now(), loc)
	type cancelled struct {
		id      string
		helpers []string
	}
	var done []cancelled
	var after Request
	err := s.tx(ctx, func(tx pgx.Tx) error {
		l, err := lock(ctx, tx, u.StableID, id)
		if errors.Is(err, errNotFound) {
			return errMissing
		}
		if err != nil {
			return err
		}
		if !s.canManage(u, l.CreatedBy) {
			return errForbidden
		}
		ids := []string{}
		if l.Status == StatusOpen || l.Status == StatusAssigned {
			ids = append(ids, id)
		} else if l.Status == StatusDone {
			return errf(http.StatusConflict, "not_open", "the request is already done")
		}
		if scope == "series" && l.SeriesID != nil {
			rows, err := tx.Query(ctx, `
				SELECT id::text FROM requests
				WHERE stable_id = $1 AND series_id = $2::uuid AND id <> $3::uuid AND status IN ('open', 'assigned')
				  AND COALESCE(date_end, date) >= $4::date
				ORDER BY date FOR UPDATE`, u.StableID, *l.SeriesID, id, td)
			if err != nil {
				return err
			}
			for rows.Next() {
				var other string
				if err := rows.Scan(&other); err != nil {
					rows.Close()
					return err
				}
				ids = append(ids, other)
			}
			rows.Close()
			if _, err := tx.Exec(ctx, `UPDATE requests SET recurring_rule = NULL WHERE stable_id = $1 AND series_id = $2::uuid`, u.StableID, *l.SeriesID); err != nil {
				return err
			}
		}
		for _, rid := range ids {
			hrows, err := tx.Query(ctx, `SELECT user_id::text FROM request_assignees WHERE request_id = $1::uuid`, rid)
			if err != nil {
				return err
			}
			c := cancelled{id: rid}
			for hrows.Next() {
				var uid string
				if err := hrows.Scan(&uid); err != nil {
					hrows.Close()
					return err
				}
				c.helpers = append(c.helpers, uid)
			}
			hrows.Close()
			if _, err := tx.Exec(ctx, `UPDATE requests SET status = 'cancelled' WHERE id = $1::uuid`, rid); err != nil {
				return err
			}
			done = append(done, c)
		}
		after, err = s.get(ctx, tx, u.StableID, u.ID, id, td)
		return err
	})
	if err != nil {
		return Request{}, wrap(err)
	}
	for _, c := range done {
		r, err := s.get(ctx, s.Pool, u.StableID, u.ID, c.id, td)
		if err == nil {
			s.notify(ctx, u.StableID, c.helpers, push.KindHelper, "Anfrage abgesagt",
				r.title()+" · "+whenText(r.Date, r.TimeFrom), pushData(c.id))
		}
		s.publish(ctx, u.StableID, c.id, "cancelled")
	}
	return after, nil
}

// ---------------------------------------------------------------- series

func (s *Service) updateSeries(ctx context.Context, u auth.User, id string, p Patch) (Request, error) {
	if p.HorseID != nil || p.Date != nil || p.DateEnd != nil || p.RemindHelperAt != nil {
		return Request{}, invalid("scope series only changes time, location, description, tasks, helpers_needed, payload and recurring_rule")
	}
	loc := stableLocation(ctx, s.Pool, u.StableID)
	td := today(s.now(), loc)
	var head string
	var after Request
	var touched []string
	err := s.tx(ctx, func(tx pgx.Tx) error {
		l, err := lock(ctx, tx, u.StableID, id)
		if errors.Is(err, errNotFound) {
			return errMissing
		}
		if err != nil {
			return err
		}
		if !s.canManage(u, l.CreatedBy) {
			return errForbidden
		}
		if l.SeriesID == nil {
			return invalid("the request does not belong to a series")
		}
		head = *l.SeriesID
		tmpl, err := s.get(ctx, tx, u.StableID, u.ID, head, td)
		if err != nil {
			return err
		}
		in := inputFromRequest(tmpl)
		oldRule := in.RecurringRule
		if err := p.apply(&in); err != nil {
			return invalid("%s", err.Error())
		}
		if err := in.normalize(td, false); err != nil {
			return err
		}
		tasks, _ := json.Marshal(in.Tasks)
		rows, err := tx.Query(ctx, `
			SELECT r.id::text, r.status, to_char(COALESCE(r.date_end, r.date), 'YYYY-MM-DD'),
			       (SELECT count(*) FROM request_assignees a WHERE a.request_id = r.id)
			FROM requests r WHERE r.stable_id = $1 AND r.series_id = $2::uuid ORDER BY r.date FOR UPDATE`, u.StableID, head)
		if err != nil {
			return err
		}
		type row struct {
			id, status, end string
			helpers         int
		}
		var all []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.status, &r.end, &r.helpers); err != nil {
				rows.Close()
				return err
			}
			all = append(all, r)
		}
		rows.Close()
		for _, r := range all {
			active := (r.status == StatusOpen || r.status == StatusAssigned) && r.end >= td
			if r.id != head && !active {
				continue
			}
			needed := in.HelpersNeeded
			status := r.status
			if active {
				if r.helpers > needed {
					// Keep the larger crowd of an occurrence that already has more helpers.
					needed = r.helpers
				}
				status = statusFor(r.helpers, needed)
			}
			if _, err := tx.Exec(ctx, `
				UPDATE requests SET time_from = $2::time, time_to = $3::time, location = NULLIF($4, ''),
				       description = NULLIF($5, ''), tasks = $6::jsonb, helpers_needed = $7, payload = $8::jsonb,
				       status = $9, recurring_rule = NULLIF($10, '')
				WHERE id = $1::uuid`, r.id, in.TimeFrom, in.TimeTo, in.Location, in.Description, string(tasks),
				needed, string(in.Payload), status, in.RecurringRule); err != nil {
				return err
			}
			touched = append(touched, r.id)
		}
		// All rows of the series carry the rule (also past and done ones).
		if _, err := tx.Exec(ctx, `UPDATE requests SET recurring_rule = NULLIF($3, '') WHERE stable_id = $1 AND series_id = $2::uuid`,
			u.StableID, head, in.RecurringRule); err != nil {
			return err
		}
		if in.RecurringRule != oldRule && in.RecurringRule != "" {
			// New rhythm: drop upcoming occurrences nobody signed up for; the job recreates them.
			if _, err := tx.Exec(ctx, `
				DELETE FROM requests r
				WHERE r.stable_id = $1 AND r.series_id = $2::uuid AND r.id <> r.series_id AND r.status = 'open'
				  AND r.date >= $3::date AND NOT EXISTS (SELECT 1 FROM request_assignees a WHERE a.request_id = r.id)`,
				u.StableID, head, td); err != nil {
				return err
			}
		}
		after, err = s.get(ctx, tx, u.StableID, u.ID, id, td)
		return err
	})
	if err != nil {
		return Request{}, wrap(err)
	}
	if err := s.materializeSeries(ctx, s.Pool, u.StableID, head, td); err != nil {
		s.log().Error("requests: materialize series", "err", err)
	}
	// A rhythm change may have removed this (unassigned) occurrence; return the template then.
	if r, err := s.get(ctx, s.Pool, u.StableID, u.ID, id, td); err == nil {
		after = r
	} else if r, err := s.get(ctx, s.Pool, u.StableID, u.ID, head, td); err == nil {
		after = r
	}
	for _, t := range touched {
		s.publish(ctx, u.StableID, t, "updated")
	}
	return after, nil
}

// materializeSeries creates the occurrences of a series for today..today+14 days
// (idempotent, unique on series_id + date). Push notifications are not sent for
// materialized occurrences.
func (s *Service) materializeSeries(ctx context.Context, q querier, stableID, headID, todayStr string) error {
	var date, rule string
	err := q.QueryRow(ctx, `
		SELECT to_char(date, 'YYYY-MM-DD'), COALESCE(recurring_rule, '')
		FROM requests WHERE stable_id = $1 AND id = $2::uuid AND id = series_id`, stableID, headID).Scan(&date, &rule)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if rule == "" {
		return nil
	}
	r, err := ParseRule(rule)
	if err != nil {
		return err
	}
	start, err := time.Parse("2006-01-02", date)
	if err != nil {
		return err
	}
	from, err := time.Parse("2006-01-02", todayStr)
	if err != nil {
		return err
	}
	for _, d := range r.Occurrences(start, from, from.AddDate(0, 0, materializeDays)) {
		_, err := q.Exec(ctx, `
			INSERT INTO requests (stable_id, type, horse_id, created_by, date, date_end, time_from, time_to, location,
			                      description, tasks, helpers_needed, recurring_rule, payload, series_id)
			SELECT stable_id, type, horse_id, created_by, $2::date,
			       CASE WHEN date_end IS NULL THEN NULL ELSE $2::date + (date_end - date) END,
			       time_from, time_to, location, description, tasks, helpers_needed, recurring_rule, payload, series_id
			FROM requests WHERE id = $1::uuid
			ON CONFLICT (series_id, date) WHERE series_id IS NOT NULL DO NOTHING`, headID, d.Format("2006-01-02"))
		if err != nil {
			return err
		}
	}
	return nil
}

// materializeDays is how far ahead occurrences of a recurring request are created.
const materializeDays = 14

// MaterializeAll creates upcoming occurrences for every recurring series (daily job).
func (s *Service) MaterializeAll(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT stable_id::text, id::text FROM requests WHERE id = series_id AND recurring_rule IS NOT NULL`)
	if err != nil {
		return err
	}
	type head struct{ stable, id string }
	var heads []head
	for rows.Next() {
		var h head
		if err := rows.Scan(&h.stable, &h.id); err != nil {
			rows.Close()
			return err
		}
		heads = append(heads, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	days := map[string]string{}
	var firstErr error
	for _, h := range heads {
		td, ok := days[h.stable]
		if !ok {
			td = today(s.now(), stableLocation(ctx, s.Pool, h.stable))
			days[h.stable] = td
		}
		if err := s.materializeSeries(ctx, s.Pool, h.stable, h.id, td); err != nil {
			s.log().Error("requests: materialize", "series", h.id, "err", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// ---------------------------------------------------------------- thanks

// Thank marks a helper as thanked (creator only, idempotent, no push).
func (s *Service) Thank(ctx context.Context, u auth.User, id, helperID string) error {
	if !validUUID(helperID) {
		return errf(http.StatusNotFound, "not_found", "helper not found")
	}
	return wrap(s.tx(ctx, func(tx pgx.Tx) error {
		l, err := lock(ctx, tx, u.StableID, id)
		if errors.Is(err, errNotFound) {
			return errMissing
		}
		if err != nil {
			return err
		}
		if l.CreatedBy != u.ID {
			return errf(http.StatusForbidden, "forbidden", "only the creator can say thanks")
		}
		if l.Status == StatusCancelled {
			return errf(http.StatusConflict, "not_open", "the request was cancelled")
		}
		tag, err := tx.Exec(ctx, `UPDATE request_assignees SET thanked = true WHERE stable_id = $1 AND request_id = $2::uuid AND user_id = $3::uuid`,
			u.StableID, id, helperID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errf(http.StatusNotFound, "not_found", "helper not found")
		}
		return nil
	}))
}

// ThanksCount is how often the user was thanked. It is a private number for the
// user; there is no ranking.
func (s *Service) ThanksCount(ctx context.Context, u auth.User) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM request_assignees WHERE stable_id = $1 AND user_id = $2::uuid AND thanked`, u.StableID, u.ID).Scan(&n)
	if err != nil {
		return 0, internal(err)
	}
	return n, nil
}

// ---------------------------------------------------------------- options and settings

// Options lists what the "new request" form needs to fill its pickers.
type Options struct {
	Horses  []OptionHorse  `json:"horses"`
	Members []OptionMember `json:"members"`
}

// OptionHorse is a selectable horse.
type OptionHorse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ColorKey *string `json:"color_key"`
}

// OptionMember is a member of the stable.
type OptionMember struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	AvatarColor *string `json:"avatar_color"`
}

// Options returns the horses and members of the stable.
func (s *Service) Options(ctx context.Context, u auth.User) (Options, error) {
	out := Options{Horses: []OptionHorse{}, Members: []OptionMember{}}
	rows, err := s.Pool.Query(ctx, `SELECT id::text, name, color_key FROM horses WHERE stable_id = $1 ORDER BY name`, u.StableID)
	if err != nil {
		return out, internal(err)
	}
	for rows.Next() {
		var h OptionHorse
		if err := rows.Scan(&h.ID, &h.Name, &h.ColorKey); err != nil {
			rows.Close()
			return out, internal(err)
		}
		out.Horses = append(out.Horses, h)
	}
	rows.Close()
	rows, err = s.Pool.Query(ctx, `SELECT id::text, name, avatar_color FROM users WHERE stable_id = $1 ORDER BY name`, u.StableID)
	if err != nil {
		return out, internal(err)
	}
	for rows.Next() {
		var m OptionMember
		if err := rows.Scan(&m.ID, &m.Name, &m.AvatarColor); err != nil {
			rows.Close()
			return out, internal(err)
		}
		out.Members = append(out.Members, m)
	}
	rows.Close()
	return out, nil
}

// NewRequestPush reports whether the user opted in to pushes for new requests.
func (s *Service) NewRequestPush(ctx context.Context, u auth.User) (bool, error) {
	var on bool
	err := s.Pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT enabled FROM reminder_settings WHERE user_id = $1::uuid AND kind = $2), false)`,
		u.ID, push.KindNewRequest).Scan(&on)
	return on, err
}

// SetNewRequestPush switches the opt-in for pushes about new requests.
func (s *Service) SetNewRequestPush(ctx context.Context, u auth.User, on bool) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1, $2::uuid, $3, $4)
		ON CONFLICT (user_id, kind) DO UPDATE SET enabled = EXCLUDED.enabled`, u.StableID, u.ID, push.KindNewRequest, on)
	return err
}
