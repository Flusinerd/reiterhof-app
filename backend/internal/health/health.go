// Package health implements the due dates per horse (vaccination, farrier, deworming, dentist,
// physio, medication, vet) and their reminders (JAN-12). See docs/domains/horses.md.
package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Kinds are the accepted values of health_items.kind.
var Kinds = []string{"vaccination", "farrier", "deworming", "dentist", "physio", "medication", "vet"}

// SummaryKinds are the four tiles of the horse record, in display order.
var SummaryKinds = []string{"vaccination", "farrier", "deworming", "dentist"}

// Register adds the health routes.
//
//	GET    /api/v1/horses/{id}/health           items + four summary tiles (all members)
//	POST   /api/v1/horses/{id}/health-items     owner/admin
//	PATCH  /api/v1/health-items/{id}            owner/admin
//	DELETE /api/v1/health-items/{id}            owner/admin
//	POST   /api/v1/health-items/{id}/done       owner/admin, moves due_date forward
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
}

type handler struct{ deps httpx.Deps }

// Item is one entry of the health list.
type Item struct {
	ID           string  `json:"id"`
	HorseID      string  `json:"horse_id"`
	Kind         string  `json:"kind"`
	Label        string  `json:"label"`
	DueDate      *string `json:"due_date"` // YYYY-MM-DD, stable-local calendar day
	IntervalDays *int    `json:"interval_days"`
	Note         *string `json:"note"`
	DailyTime    *string `json:"daily_time"` // HH:MM, for daily medication
	// DaysUntilDue is negative when overdue; null without a due date.
	DaysUntilDue *int `json:"days_until_due"`
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

const itemColumns = `i.id, i.horse_id, i.kind, i.label, to_char(i.due_date, 'YYYY-MM-DD'), i.interval_days, i.note,
	to_char(i.daily_time, 'HH24:MI')`

func scanItem(row pgx.Row) (Item, error) {
	var it Item
	err := row.Scan(&it.ID, &it.HorseID, &it.Kind, &it.Label, &it.DueDate, &it.IntervalDays, &it.Note, &it.DailyTime)
	return it, err
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
	// IntervalDays is 1..3650; 0 clears it.
	IntervalDays *int    `json:"interval_days"`
	Note         *string `json:"note"`
	// DailyTime is HH:MM (stable-local) for daily reminders; "" clears it.
	DailyTime *string `json:"daily_time"`
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
	if in.IntervalDays != nil && (*in.IntervalDays < 0 || *in.IntervalDays > 3650) {
		return "interval_days must be between 1 and 3650"
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
	it, err := scanItem(h.deps.Pool.QueryRow(r.Context(), `WITH i AS (
			INSERT INTO health_items (stable_id, horse_id, kind, label, due_date, interval_days, note, daily_time)
			VALUES ($1, $2, $3, $4, NULLIF($5, '')::date, NULLIF($6::int, 0), NULLIF($7, ''), NULLIF($8, '')::time)
			RETURNING *)
		SELECT `+itemColumns+` FROM i`,
		user.StableID, horseID, *in.Kind, *in.Label, in.DueDate, in.IntervalDays, in.Note, in.DailyTime))
	if err != nil {
		h.fail(w, "create", err)
		return
	}
	h.respond(w, r, http.StatusCreated, it, user.StableID)
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
	if in.IntervalDays != nil {
		add("interval_days = NULLIF($%d::int, 0)", *in.IntervalDays)
	}
	if in.Note != nil {
		add("note = NULLIF($%d, '')", *in.Note)
	}
	if in.DailyTime != nil {
		add("daily_time = NULLIF($%d, '')::time", *in.DailyTime)
	}
	if len(sets) == 0 {
		sets = append(sets, "label = label")
	}
	it, err := scanItem(h.deps.Pool.QueryRow(r.Context(), `WITH i AS (
			UPDATE health_items SET `+strings.Join(sets, ", ")+` WHERE id = $1 AND stable_id = $2 RETURNING *)
		SELECT `+itemColumns+` FROM i`, args...))
	if err != nil {
		h.fail(w, "patch", err)
		return
	}
	h.respond(w, r, http.StatusOK, it, user.StableID)
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
	if _, err := h.deps.Pool.Exec(r.Context(), `DELETE FROM health_items WHERE id = $1 AND stable_id = $2`,
		r.PathValue("id"), user.StableID); err != nil {
		h.fail(w, "delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// NextDue computes the due date after an item was done ("erledigt"). With an interval
// the date moves forward by one interval from the previous due date, and by further
// intervals until it lies after today (so a long overdue item does not stay overdue).
// Without a previous due date the interval counts from today. Without an interval
// there is no follow-up and the result is nil (the due date is cleared).
func NextDue(prev *time.Time, intervalDays *int, today time.Time) *time.Time {
	if intervalDays == nil || *intervalDays <= 0 {
		return nil
	}
	step := *intervalDays
	next := today.AddDate(0, 0, step)
	if prev != nil {
		next = prev.AddDate(0, 0, step)
		for !next.After(today) {
			next = next.AddDate(0, 0, step)
		}
	}
	return &next
}

func (h *handler) done(w http.ResponseWriter, r *http.Request) {
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
	var (
		due      *time.Time
		interval *int
	)
	err = h.deps.Pool.QueryRow(r.Context(), `SELECT due_date, interval_days FROM health_items WHERE id = $1 AND stable_id = $2`,
		r.PathValue("id"), user.StableID).Scan(&due, &interval)
	if err != nil {
		h.fail(w, "load item", err)
		return
	}
	if due != nil {
		d := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, time.UTC)
		due = &d
	}
	next := NextDue(due, interval, today)
	var nextStr *string
	if next != nil {
		s := next.Format("2006-01-02")
		nextStr = &s
	}
	it, err := scanItem(h.deps.Pool.QueryRow(r.Context(), `WITH i AS (
			UPDATE health_items SET due_date = $3::date WHERE id = $1 AND stable_id = $2 RETURNING *)
		SELECT `+itemColumns+` FROM i`, r.PathValue("id"), user.StableID, nextStr))
	if err != nil {
		h.fail(w, "done", err)
		return
	}
	annotate(&it, today)
	httpx.WriteJSON(w, http.StatusOK, it)
}
