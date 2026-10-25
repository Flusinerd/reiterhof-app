// Package health implements the due dates per horse (vaccination, farrier, deworming, dentist,
// physio, medication, vet) and their reminders (JAN-12). See docs/domains/horses.md.
package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

// EventChanged is the realtime event after every change to the health items of a horse
// (create, edit, delete, done, undo). Data: {horse_id}.
const EventChanged = "health.changed"

// Kinds are the accepted values of health_items.kind.
var Kinds = []string{"vaccination", "farrier", "deworming", "dentist", "physio", "medication", "vet"}

// IntervalUnits are the accepted values of health_items.interval_unit.
var IntervalUnits = []string{"day", "week", "month", "year"}

// maxIntervalCount is the largest interval per unit (about ten years).
var maxIntervalCount = map[string]int{"day": 3650, "week": 520, "month": 120, "year": 10}

// DefaultRemindDays are the days before the due date a new item reminds on, unless it says otherwise.
var DefaultRemindDays = []int{7, 1}

// SummaryKinds are the four tiles of the horse record, in display order.
var SummaryKinds = []string{"vaccination", "farrier", "deworming", "dentist"}

// Register adds the health routes.
//
//	GET    /api/v1/horses/{id}/health           items + four summary tiles (all members)
//	POST   /api/v1/horses/{id}/health-items     owner/admin
//	PATCH  /api/v1/health-items/{id}            owner/admin
//	DELETE /api/v1/health-items/{id}            owner/admin
//	POST   /api/v1/health-items/{id}/done       owner/admin, moves due_date forward (once per day)
//	DELETE /api/v1/health-items/{id}/done       owner/admin, undoes today's "erledigt"
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	route := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, auth.RequireStable(fn))
	}
	route("GET /api/v1/horses/{id}/health", h.list)
	route("POST /api/v1/horses/{id}/health-items", h.create)
	route("PATCH /api/v1/health-items/{id}", h.patch)
	route("DELETE /api/v1/health-items/{id}", h.delete)
	route("POST /api/v1/health-items/{id}/done", h.done)
	route("DELETE /api/v1/health-items/{id}/done", h.undoDone)
}

type handler struct{ deps httpx.Deps }

// Item is one entry of the health list.
type Item struct {
	ID      string  `json:"id"`
	HorseID string  `json:"horse_id"`
	Kind    string  `json:"kind"`
	Label   string  `json:"label"`
	DueDate *string `json:"due_date"` // YYYY-MM-DD, stable-local calendar day
	// IntervalCount and IntervalUnit (day, week, month, year) are the repeat interval; count is
	// null without one.
	IntervalCount *int   `json:"interval_count"`
	IntervalUnit  string `json:"interval_unit"`
	// IntervalDays is the interval in days for day and week intervals (null otherwise); kept for
	// app versions that only know days.
	IntervalDays *int    `json:"interval_days"`
	Note         *string `json:"note"`
	DailyTime    *string `json:"daily_time"` // HH:MM, for daily medication
	// RemindDays are the days before the due date a reminder goes out (0 = on the day), largest
	// first; empty means no due-date reminder. RemindTime is the stable-local HH:MM it goes out at.
	RemindDays []int  `json:"remind_days"`
	RemindTime string `json:"remind_time"`
	// DaysUntilDue is negative when overdue; null without a due date.
	DaysUntilDue *int `json:"days_until_due"`
	// LastDoneOn is the stable-local day (YYYY-MM-DD) of the last "erledigt"; null if there was none
	// or the due date was edited by hand since. While it is today, a second "erledigt" changes
	// nothing and DELETE .../done takes it back.
	LastDoneOn *string `json:"last_done_on"`
	// LastDoneBy is who marked the item done that day.
	LastDoneBy *Person `json:"last_done_by"`
	// Bundle is set by GET /horses/{id}/health for vet, vaccination and dentist items due within
	// 14 days when other horses have the same kind due about then (JAN-54); absent otherwise.
	Bundle *Bundle `json:"bundle,omitempty"`
}

// Person is a member as shown next to an item.
type Person struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Response is the body of GET /horses/{id}/health.
type Response struct {
	// Today is the stable-local date the days are counted from.
	Today string `json:"today"`
	// Items are ordered by due date (soonest first, undated last).
	Items []Item `json:"items"`
	// Summary has one entry per SummaryKinds: the item of that kind due first, or null.
	Summary   map[string]*Item `json:"summary"`
	CanManage bool             `json:"can_manage"`
}

const itemColumns = `i.id, i.horse_id, i.kind, i.label, to_char(i.due_date, 'YYYY-MM-DD'), i.interval_count,
	i.interval_unit, i.note, to_char(i.daily_time, 'HH24:MI'), i.remind_days, to_char(i.remind_time, 'HH24:MI'),
	to_char(i.last_done_on, 'YYYY-MM-DD'), i.last_done_by, (SELECT u.name FROM users u WHERE u.id = i.last_done_by)`

func scanItem(row pgx.Row) (Item, error) {
	var it Item
	var remind []int32
	var doneBy, doneByName *string
	err := row.Scan(&it.ID, &it.HorseID, &it.Kind, &it.Label, &it.DueDate, &it.IntervalCount, &it.IntervalUnit,
		&it.Note, &it.DailyTime, &remind, &it.RemindTime, &it.LastDoneOn, &doneBy, &doneByName)
	if err != nil {
		return it, err
	}
	if doneBy != nil {
		it.LastDoneBy = &Person{ID: *doneBy}
		if doneByName != nil {
			it.LastDoneBy.Name = *doneByName
		}
	}
	it.RemindDays = make([]int, len(remind))
	for i, d := range remind {
		it.RemindDays[i] = int(d)
	}
	if it.IntervalCount != nil {
		switch it.IntervalUnit {
		case "day":
			n := *it.IntervalCount
			it.IntervalDays = &n
		case "week":
			n := *it.IntervalCount * 7
			it.IntervalDays = &n
		}
	}
	return it, nil
}

func (h *handler) fail(w http.ResponseWriter, what string, err error) {
	h.deps.Log.Error("health: "+what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func invalid(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusBadRequest, "validation_failed", msg)
}

func isInvalidUUID(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "22P02"
}

// stableToday returns the current calendar day in the stable's time zone as YYYY-MM-DD.
func (h *handler) stableToday(ctx context.Context, stableID string) (time.Time, error) {
	var tz string
	if err := h.deps.Pool.QueryRow(ctx, `SELECT timezone FROM stables WHERE id = $1`, stableID).Scan(&tz); err != nil {
		return time.Time{}, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	y, m, d := h.deps.Now().In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
}

// daysUntil counts calendar days from today to the YYYY-MM-DD date.
func daysUntil(today time.Time, due string) (int, bool) {
	d, err := time.Parse("2006-01-02", due)
	if err != nil {
		return 0, false
	}
	return int(d.Sub(today).Hours() / 24), true
}

func annotate(it *Item, today time.Time) {
	if it.DueDate != nil {
		if n, ok := daysUntil(today, *it.DueDate); ok {
			it.DaysUntilDue = &n
		}
	}
}

// horse checks the horse is in the caller's stable (404) and, if write is set, that the
// caller is owner or admin (403).
func (h *handler) horse(w http.ResponseWriter, r *http.Request, horseID string, write bool) (auth.User, bool) {
	user, _ := auth.UserFrom(r.Context())
	in, err := auth.HorseInStable(r.Context(), h.deps.Pool, user, horseID)
	if err != nil {
		h.fail(w, "horse in stable", err)
		return user, false
	}
	if !in {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "horse not found")
		return user, false
	}
	if write {
		ok, err := auth.CanManageHorse(r.Context(), h.deps.Pool, user, horseID)
		if err != nil {
			h.fail(w, "can manage", err)
			return user, false
		}
		if !ok {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner or an admin may change health items")
			return user, false
		}
	}
	return user, true
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	horseID := r.PathValue("id")
	user, ok := h.horse(w, r, horseID, false)
	if !ok {
		return
	}
	today, err := h.stableToday(r.Context(), user.StableID)
	if err != nil {
		h.fail(w, "today", err)
		return
	}
	rows, err := h.deps.Pool.Query(r.Context(), `SELECT `+itemColumns+` FROM health_items i
		WHERE i.horse_id = $1 AND i.stable_id = $2
		ORDER BY i.due_date NULLS LAST, i.created_at, i.id`, horseID, user.StableID)
	if err != nil {
		h.fail(w, "list", err)
		return
	}
	defer rows.Close()
	resp := Response{Today: today.Format("2006-01-02"), Items: []Item{}, Summary: map[string]*Item{}}
	for _, k := range SummaryKinds {
		resp.Summary[k] = nil
	}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			h.fail(w, "scan", err)
			return
		}
		annotate(&it, today)
		resp.Items = append(resp.Items, it)
	}
	if err := rows.Err(); err != nil {
		h.fail(w, "list", err)
		return
	}
	if err := h.annotateBundles(r.Context(), user.StableID, horseID, today, resp.Items); err != nil {
		h.fail(w, "bundles", err)
		return
	}
	for i := range resp.Items {
		it := &resp.Items[i]
		if cur, tile := resp.Summary[it.Kind], contains(SummaryKinds, it.Kind); tile && cur == nil && it.DueDate != nil {
			cp := *it
			resp.Summary[it.Kind] = &cp
		}
	}
	resp.CanManage, err = auth.CanManageHorse(r.Context(), h.deps.Pool, user, horseID)
	if err != nil {
		h.fail(w, "can manage", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

var timeRe = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

type input struct {
	Kind  *string `json:"kind"`
	Label *string `json:"label"`
	// DueDate is YYYY-MM-DD; "" clears it.
	DueDate *string `json:"due_date"`
	// IntervalCount is the number of IntervalUnit (day, week, month, year) between two dates;
	// 0 clears the interval. IntervalUnit alone changes the unit of the current interval.
	IntervalCount *int    `json:"interval_count"`
	IntervalUnit  *string `json:"interval_unit"`
	// IntervalDays is the older form of the interval (1..3650 days, 0 clears it); it sets
	// interval_count with the unit day. Not together with interval_count.
	IntervalDays *int    `json:"interval_days"`
	Note         *string `json:"note"`
	// DailyTime is HH:MM (stable-local) for daily reminders; "" clears it.
	DailyTime *string `json:"daily_time"`
	// RemindDays are 0..365 days before the due date (at most 5, duplicates removed); [] turns
	// due-date reminders off.
	RemindDays *[]int `json:"remind_days"`
	// RemindTime is the stable-local HH:MM due-date reminders go out at.
	RemindTime *string `json:"remind_time"`
}

func (in *input) validate() string {
	if in.Kind != nil && !contains(Kinds, *in.Kind) {
		return "kind must be one of " + strings.Join(Kinds, ", ")
	}
	if in.Label != nil {
		l := strings.TrimSpace(*in.Label)
		if l == "" || utf8.RuneCountInString(l) > 120 {
			return "label must be 1 to 120 characters"
		}
		in.Label = &l
	}
	if in.DueDate != nil && *in.DueDate != "" {
		if _, err := time.Parse("2006-01-02", *in.DueDate); err != nil {
			return "due_date must be YYYY-MM-DD"
		}
	}
	if in.IntervalDays != nil {
		if in.IntervalCount != nil || in.IntervalUnit != nil {
			return "send either interval_days or interval_count and interval_unit"
		}
		if *in.IntervalDays < 0 || *in.IntervalDays > 3650 {
			return "interval_days must be between 1 and 3650"
		}
		day := "day"
		in.IntervalCount, in.IntervalUnit, in.IntervalDays = in.IntervalDays, &day, nil
	}
	if in.IntervalUnit != nil && !contains(IntervalUnits, *in.IntervalUnit) {
		return "interval_unit must be one of " + strings.Join(IntervalUnits, ", ")
	}
	if in.IntervalCount != nil {
		limit := maxIntervalCount["day"]
		if in.IntervalUnit != nil {
			limit = maxIntervalCount[*in.IntervalUnit]
		}
		if *in.IntervalCount < 0 || *in.IntervalCount > limit {
			return fmt.Sprintf("interval_count must be between 1 and %d", limit)
		}
	}
	if in.RemindDays != nil {
		days := slices.Clone(*in.RemindDays)
		slices.Sort(days)
		days = slices.Compact(days)
		if len(days) > 5 || (len(days) > 0 && (days[0] < 0 || days[len(days)-1] > 365)) {
			return "remind_days must be at most 5 values between 0 and 365"
		}
		slices.Reverse(days)
		in.RemindDays = &days
	}
	if in.RemindTime != nil && !timeRe.MatchString(*in.RemindTime) {
		return "remind_time must be HH:MM"
	}
	if in.Note != nil && utf8.RuneCountInString(*in.Note) > 1000 {
		return "note is too long"
	}
	if in.DailyTime != nil && *in.DailyTime != "" && !timeRe.MatchString(*in.DailyTime) {
		return "daily_time must be HH:MM"
	}
	return ""
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	horseID := r.PathValue("id")
	user, ok := h.horse(w, r, horseID, true)
	if !ok {
		return
	}
	var in input
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Kind == nil || in.Label == nil {
		invalid(w, "kind and label are required")
		return
	}
	if msg := in.validate(); msg != "" {
		invalid(w, msg)
		return
	}
	unit := "day"
	if in.IntervalUnit != nil {
		unit = *in.IntervalUnit
	}
	remindDays := DefaultRemindDays
	if in.RemindDays != nil {
		remindDays = *in.RemindDays
	}
	remindTime := "08:00"
	if in.RemindTime != nil {
		remindTime = *in.RemindTime
	}
	var it Item
	err := pgx.BeginFunc(r.Context(), h.deps.Pool, func(tx pgx.Tx) error {
		var err error
		it, err = scanItem(tx.QueryRow(r.Context(), `WITH i AS (
				INSERT INTO health_items (stable_id, horse_id, kind, label, due_date, interval_count, interval_unit, note,
					daily_time, remind_days, remind_time)
				VALUES ($1, $2, $3, $4, NULLIF($5, '')::date, NULLIF($6::int, 0), $7, NULLIF($8, ''), NULLIF($9, '')::time,
					$10, $11::time)
				RETURNING *)
			SELECT `+itemColumns+` FROM i`,
			user.StableID, horseID, *in.Kind, *in.Label, in.DueDate, in.IntervalCount, unit, in.Note, in.DailyTime,
			remindDays, remindTime))
		if err != nil {
			return err
		}
		return publish(r.Context(), tx, user.StableID, horseID)
	})
	if err != nil {
		h.fail(w, "create", err)
		return
	}
	h.respond(w, r, http.StatusCreated, it, user.StableID)
}

// publish sends health.changed inside the transaction, so it only goes out when the change commits.
func publish(ctx context.Context, tx pgx.Tx, stableID, horseID string) error {
	return realtime.Publish(ctx, tx, stableID, EventChanged, map[string]any{"horse_id": horseID})
}

// dismissReminders hides the stored reminders (the pushes listed in the reminder center) of an item
// for everybody: they describe a state that no longer holds once the item was done, moved or deleted.
// A computed item of the new state shows up on its own where it belongs.
func dismissReminders(ctx context.Context, tx pgx.Tx, itemID string, now time.Time) error {
	_, err := tx.Exec(ctx, `UPDATE reminders SET dismissed_at = $2
		WHERE source_table = 'health_items' AND source_id = $1 AND dismissed_at IS NULL`, itemID, now)
	return err
}

func (h *handler) respond(w http.ResponseWriter, r *http.Request, status int, it Item, stableID string) {
	if today, err := h.stableToday(r.Context(), stableID); err == nil {
		annotate(&it, today)
	}
	httpx.WriteJSON(w, status, it)
}

// itemHorse finds the horse of a health item in the caller's stable (404 otherwise).
func (h *handler) itemHorse(w http.ResponseWriter, r *http.Request) (horseID string, ok bool) {
	user, _ := auth.UserFrom(r.Context())
	err := h.deps.Pool.QueryRow(r.Context(), `SELECT horse_id FROM health_items WHERE id = $1 AND stable_id = $2`,
		r.PathValue("id"), user.StableID).Scan(&horseID)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "health item not found")
		return "", false
	}
	if err != nil {
		h.fail(w, "item horse", err)
		return "", false
	}
	return horseID, true
}

func (h *handler) patch(w http.ResponseWriter, r *http.Request) {
	horseID, ok := h.itemHorse(w, r)
	if !ok {
		return
	}
	user, ok := h.horse(w, r, horseID, true)
	if !ok {
		return
	}
	var in input
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		invalid(w, msg)
		return
	}
	var sets []string
	args := []any{r.PathValue("id"), user.StableID}
	add := func(expr string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf(expr, len(args)))
	}
	if in.Kind != nil {
		add("kind = $%d", *in.Kind)
	}
	if in.Label != nil {
		add("label = $%d", *in.Label)
	}
	if in.DueDate != nil {
		add("due_date = NULLIF($%d, '')::date", *in.DueDate)
	}
	if in.IntervalCount != nil {
		add("interval_count = NULLIF($%d::int, 0)", *in.IntervalCount)
	}
	if in.IntervalUnit != nil {
		add("interval_unit = $%d", *in.IntervalUnit)
	}
	if in.Note != nil {
		add("note = NULLIF($%d, '')", *in.Note)
	}
	if in.DailyTime != nil {
		add("daily_time = NULLIF($%d, '')::time", *in.DailyTime)
	}
	if in.RemindDays != nil {
		add("remind_days = $%d", *in.RemindDays)
	}
	if in.RemindTime != nil {
		add("remind_time = $%d::time", *in.RemindTime)
	}
	var it Item
	err := pgx.BeginFunc(r.Context(), h.deps.Pool, func(tx pgx.Tx) error {
		cur, err := lockSchedule(r.Context(), tx, r.PathValue("id"), user.StableID)
		if err != nil {
			return err
		}
		dueMoved, otherChange := cur.changedBy(&in)
		if dueMoved {
			// A hand-edited due date is a fresh start: today's "erledigt" and its undo no longer apply.
			sets = append(sets, "last_done_on = NULL", "last_done_at = NULL", "last_done_by = NULL", "previous_due_date = NULL")
		}
		if len(sets) == 0 {
			sets = append(sets, "label = label")
		}
		it, err = scanItem(tx.QueryRow(r.Context(), `WITH i AS (
				UPDATE health_items SET `+strings.Join(sets, ", ")+` WHERE id = $1 AND stable_id = $2 RETURNING *)
			SELECT `+itemColumns+` FROM i`, args...))
		if err != nil {
			return err
		}
		if dueMoved || otherChange {
			if err := dismissReminders(r.Context(), tx, it.ID, h.deps.Now()); err != nil {
				return err
			}
		}
		return publish(r.Context(), tx, user.StableID, horseID)
	})
	if errors.Is(err, pgx.ErrNoRows) { // deleted while this request ran
		httpx.WriteError(w, http.StatusNotFound, "not_found", "health item not found")
		return
	}
	if err != nil {
		h.fail(w, "patch", err)
		return
	}
	h.respond(w, r, http.StatusOK, it, user.StableID)
}

// schedule is what a stored reminder of an item is about: when it is due, repeats and reminds.
type schedule struct {
	due        string // YYYY-MM-DD, "" = none
	count      int    // 0 = no interval
	unit       string
	daily      string // HH:MM, "" = none
	remindDays []int
	remindTime string
}

// lockSchedule locks the item row for the rest of the transaction and reads its schedule.
func lockSchedule(ctx context.Context, tx pgx.Tx, id, stableID string) (schedule, error) {
	var s schedule
	var days []int32
	err := tx.QueryRow(ctx, `SELECT COALESCE(to_char(due_date, 'YYYY-MM-DD'), ''), COALESCE(interval_count, 0), interval_unit,
			COALESCE(to_char(daily_time, 'HH24:MI'), ''), remind_days, to_char(remind_time, 'HH24:MI')
		FROM health_items WHERE id = $1 AND stable_id = $2 FOR UPDATE`, id, stableID).
		Scan(&s.due, &s.count, &s.unit, &s.daily, &days, &s.remindTime)
	for _, d := range days {
		s.remindDays = append(s.remindDays, int(d))
	}
	return s, err
}

// changedBy tells whether a PATCH moves the due date and whether it changes anything else a stored
// reminder was about (interval, daily time, reminder days or time). Sending the current value again
// is no change, so editing only the label keeps today's "erledigt" and the reminders.
func (s schedule) changedBy(in *input) (dueMoved, other bool) {
	dueMoved = in.DueDate != nil && *in.DueDate != s.due
	other = (in.IntervalCount != nil && *in.IntervalCount != s.count) ||
		(in.IntervalUnit != nil && *in.IntervalUnit != s.unit) ||
		(in.DailyTime != nil && *in.DailyTime != s.daily) ||
		(in.RemindDays != nil && !slices.Equal(*in.RemindDays, s.remindDays)) ||
		(in.RemindTime != nil && *in.RemindTime != s.remindTime)
	return dueMoved, other
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	horseID, ok := h.itemHorse(w, r)
	if !ok {
		return
	}
	user, ok := h.horse(w, r, horseID, true)
	if !ok {
		return
	}
	err := pgx.BeginFunc(r.Context(), h.deps.Pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(), `DELETE FROM health_items WHERE id = $1 AND stable_id = $2`,
			r.PathValue("id"), user.StableID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows // deleted while this request ran
		}
		// The reminder rows have no foreign key to the item; without this they would stay behind
		// in the reminder center as entries without a target.
		if err := dismissReminders(r.Context(), tx, r.PathValue("id"), h.deps.Now()); err != nil {
			return err
		}
		return publish(r.Context(), tx, user.StableID, horseID)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "health item not found")
		return
	}
	if err != nil {
		h.fail(w, "delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Interval is the repeat interval of a health item: Count days, weeks, months or years.
type Interval struct {
	Count int
	Unit  string
}

// After returns the date n intervals after from. Months and years keep the day of the month
// and fall back to the last day of a shorter month (31 Jan + 1 month = 28/29 Feb), always counted
// from from itself, so a date never drifts (31 Jan + 2 months = 31 Mar).
func (iv Interval) After(from time.Time, n int) time.Time {
	switch iv.Unit {
	case "week":
		return from.AddDate(0, 0, 7*iv.Count*n)
	case "month", "year":
		months := iv.Count * n
		if iv.Unit == "year" {
			months *= 12
		}
		first := time.Date(from.Year(), from.Month()+time.Month(months), 1, 0, 0, 0, 0, from.Location())
		last := first.AddDate(0, 1, -1).Day()
		return first.AddDate(0, 0, min(from.Day(), last)-1)
	default:
		return from.AddDate(0, 0, iv.Count*n)
	}
}

// NextDue computes the due date after an item was done ("erledigt"). With an interval
// the date moves forward by one interval from the previous due date, and by further
// intervals until it lies after today (so a long overdue item does not stay overdue).
// Without a previous due date the interval counts from today. Without an interval
// there is no follow-up and the result is nil (the due date is cleared).
func NextDue(prev *time.Time, iv *Interval, today time.Time) *time.Time {
	if iv == nil || iv.Count <= 0 {
		return nil
	}
	if prev == nil {
		next := iv.After(today, 1)
		return &next
	}
	next := iv.After(*prev, 1)
	for n := 2; !next.After(today); n++ {
		next = iv.After(*prev, n)
	}
	return &next
}

// DueAfterDone is the due date of an item once it was done today. An item that repeats follows
// NextDue. For a medication course the due date is the last day of the course: "done" ends the
// course today (an end date further away moves up to today, one in the past or none stays) instead
// of clearing the date, which would let the daily reminders run on forever.
func DueAfterDone(due *time.Time, iv *Interval, course bool, today time.Time) *time.Time {
	if !course {
		return NextDue(due, iv, today)
	}
	if due != nil && due.After(today) {
		return &today
	}
	return due
}

var (
	errDueDateChanged = errors.New("due date changed")
	errUndoExpired    = errors.New("undo expired")
)

// doneInput is the optional body of POST /health-items/{id}/done.
type doneInput struct {
	// DueDate is the due date the app showed ("" = none). If the item has another one by now,
	// somebody else changed it and nothing is done: 409 due_date_changed.
	DueDate *string `json:"due_date"`
}

// doneState is what "erledigt" and its undo need to know about a locked item.
type doneState struct {
	due        *time.Time // UTC midnight
	interval   *Interval
	course     bool    // medication: due date = last day of the course
	lastDoneOn *string // YYYY-MM-DD
	lastDoneAt *time.Time
}

func (s doneState) dueString() string {
	if s.due == nil {
		return ""
	}
	return s.due.Format("2006-01-02")
}

// lockForDone locks the item row for the rest of the transaction and reads its state.
func lockForDone(ctx context.Context, tx pgx.Tx, id, stableID string) (doneState, error) {
	var (
		s     doneState
		count *int
		unit  string
	)
	err := tx.QueryRow(ctx, `SELECT due_date, interval_count, interval_unit, kind = 'medication' OR daily_time IS NOT NULL,
			to_char(last_done_on, 'YYYY-MM-DD'), last_done_at
		FROM health_items WHERE id = $1 AND stable_id = $2 FOR UPDATE`, id, stableID).
		Scan(&s.due, &count, &unit, &s.course, &s.lastDoneOn, &s.lastDoneAt)
	if err != nil {
		return s, err
	}
	if s.due != nil {
		d := time.Date(s.due.Year(), s.due.Month(), s.due.Day(), 0, 0, 0, 0, time.UTC)
		s.due = &d
	}
	if count != nil {
		s.interval = &Interval{Count: *count, Unit: unit}
	}
	return s, nil
}

func loadItem(ctx context.Context, tx pgx.Tx, id, stableID string) (Item, error) {
	return scanItem(tx.QueryRow(ctx, `SELECT `+itemColumns+` FROM health_items i WHERE i.id = $1 AND i.stable_id = $2`, id, stableID))
}

// done handles POST /health-items/{id}/done. It is idempotent per stable-local day: the first call
// moves the due date (and remembers the old one for the undo), every further call that day answers
// with the unchanged item, so a second tap, a retry or a second phone cannot push the date another
// interval out.
func (h *handler) done(w http.ResponseWriter, r *http.Request) {
	horseID, ok := h.itemHorse(w, r)
	if !ok {
		return
	}
	user, ok := h.horse(w, r, horseID, true)
	if !ok {
		return
	}
	var in doneInput
	if r.ContentLength != 0 && !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.DueDate != nil && *in.DueDate != "" {
		if _, err := time.Parse("2006-01-02", *in.DueDate); err != nil {
			invalid(w, "due_date must be YYYY-MM-DD")
			return
		}
	}
	today, err := h.stableToday(r.Context(), user.StableID)
	if err != nil {
		h.fail(w, "today", err)
		return
	}
	day := today.Format("2006-01-02")
	id := r.PathValue("id")

	var it Item
	err = pgx.BeginFunc(r.Context(), h.deps.Pool, func(tx pgx.Tx) error {
		cur, err := lockForDone(r.Context(), tx, id, user.StableID)
		if err != nil {
			return err
		}
		if cur.lastDoneOn != nil && *cur.lastDoneOn == day {
			it, err = loadItem(r.Context(), tx, id, user.StableID)
			return err
		}
		// Checked after the repeat above: the second tap of the same app also carries the old date.
		if in.DueDate != nil && *in.DueDate != cur.dueString() {
			return errDueDateChanged
		}
		var next *string
		if n := DueAfterDone(cur.due, cur.interval, cur.course, today); n != nil {
			s := n.Format("2006-01-02")
			next = &s
		}
		// One timestamp for the item and the reminder rows it dismisses: the undo finds them by it.
		now := h.deps.Now()
		it, err = scanItem(tx.QueryRow(r.Context(), `WITH i AS (
				UPDATE health_items SET previous_due_date = due_date, due_date = $3::date, last_done_on = $4::date,
					last_done_at = $5, last_done_by = $6
				WHERE id = $1 AND stable_id = $2 RETURNING *)
			SELECT `+itemColumns+` FROM i`, id, user.StableID, next, day, now, user.ID))
		if err != nil {
			return err
		}
		if err := dismissReminders(r.Context(), tx, id, now); err != nil {
			return err
		}
		return publish(r.Context(), tx, user.StableID, horseID)
	})
	switch {
	case errors.Is(err, errDueDateChanged):
		httpx.WriteError(w, http.StatusConflict, "due_date_changed", "the due date changed since the item was loaded")
	case errors.Is(err, pgx.ErrNoRows): // deleted while this request ran
		httpx.WriteError(w, http.StatusNotFound, "not_found", "health item not found")
	case err != nil:
		h.fail(w, "done", err)
	default:
		annotate(&it, today)
		httpx.WriteJSON(w, http.StatusOK, it)
	}
}

// undoDone handles DELETE /health-items/{id}/done: it takes back today's "erledigt" and restores the
// due date from before. Nothing to undo (never done, or the date was edited by hand since) answers
// with the unchanged item, so a repeated undo is fine; an "erledigt" of an earlier day stays: 409
// undo_expired.
func (h *handler) undoDone(w http.ResponseWriter, r *http.Request) {
	horseID, ok := h.itemHorse(w, r)
	if !ok {
		return
	}
	user, ok := h.horse(w, r, horseID, true)
	if !ok {
		return
	}
	today, err := h.stableToday(r.Context(), user.StableID)
	if err != nil {
		h.fail(w, "today", err)
		return
	}
	day := today.Format("2006-01-02")
	id := r.PathValue("id")

	var it Item
	err = pgx.BeginFunc(r.Context(), h.deps.Pool, func(tx pgx.Tx) error {
		cur, err := lockForDone(r.Context(), tx, id, user.StableID)
		if err != nil {
			return err
		}
		if cur.lastDoneOn == nil {
			it, err = loadItem(r.Context(), tx, id, user.StableID)
			return err
		}
		if *cur.lastDoneOn != day {
			return errUndoExpired
		}
		it, err = scanItem(tx.QueryRow(r.Context(), `WITH i AS (
				UPDATE health_items SET due_date = previous_due_date, previous_due_date = NULL, last_done_on = NULL,
					last_done_at = NULL, last_done_by = NULL
				WHERE id = $1 AND stable_id = $2 RETURNING *)
			SELECT `+itemColumns+` FROM i`, id, user.StableID))
		if err != nil {
			return err
		}
		if cur.lastDoneAt != nil {
			// The reminders the "erledigt" hid are true again; ones dismissed at another time stay dismissed.
			if _, err := tx.Exec(r.Context(), `UPDATE reminders SET dismissed_at = NULL
				WHERE source_table = 'health_items' AND source_id = $1 AND dismissed_at = $2`, id, *cur.lastDoneAt); err != nil {
				return err
			}
		}
		return publish(r.Context(), tx, user.StableID, horseID)
	})
	switch {
	case errors.Is(err, errUndoExpired):
		httpx.WriteError(w, http.StatusConflict, "undo_expired", "only today's \"erledigt\" can be undone")
	case errors.Is(err, pgx.ErrNoRows): // deleted while this request ran
		httpx.WriteError(w, http.StatusNotFound, "not_found", "health item not found")
	case err != nil:
		h.fail(w, "undo done", err)
	default:
		annotate(&it, today)
		httpx.WriteJSON(w, http.StatusOK, it)
	}
}
