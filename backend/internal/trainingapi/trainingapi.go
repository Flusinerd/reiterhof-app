// Package trainingapi is the HTTP and storage layer of the training feature (M6):
// training profile (JAN-55), "Was heute?" (JAN-57), exercise library (JAN-58), week view
// (JAN-59), sessions (JAN-60, JAN-61) and the week plan (JAN-89).
//
// The rules live in the pure packages training, training/load and training/recommend; this
// package only reads and writes the database, maps roles to rules and shapes the JSON.
// See docs/domains/training.md for the API and the rider rule mapping.
package trainingapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Rider rule keys of horse_riders.rules (positive list) that the training feature reads.
const (
	RuleLogSessions       = "log_sessions"
	RuleReportObservation = "report_observations"
	RuleTakeWeekSlots     = "take_week_slots"
	RuleHackAlone         = "hack_alone"
	RuleShows             = "shows"
	// ruleRide is the legacy key of the first seed data (["ride","groom"]); it implies
	// log_sessions and take_week_slots.
	ruleRide = "ride"
)

const defaultTimezone = "Europe/Berlin"

type handler struct {
	deps httpx.Deps
}

// Register mounts the training routes under /api/v1.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	route := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, auth.RequireStable(fn))
	}
	route("GET /api/v1/training/horses", h.listHorses)
	route("GET /api/v1/horses/{id}/training-profile", h.getProfile)
	route("PUT /api/v1/horses/{id}/training-profile", h.putProfile)
	route("GET /api/v1/horses/{id}/today", h.today)
	route("POST /api/v1/horses/{id}/sessions", h.createSession)
	route("GET /api/v1/horses/{id}/sessions", h.listSessions)
	route("GET /api/v1/horses/{id}/week", h.getWeek)
	route("PUT /api/v1/horses/{id}/week/{day}", h.putWeekDay)
	route("POST /api/v1/horses/{id}/week/plan", h.planWeek)
	route("GET /api/v1/exercises", h.listExercises)
	route("GET /api/v1/exercises/{id}", h.getExercise)
}

// access is what the requesting user may do with one horse's training data.
type access struct {
	user      auth.User
	horseID   string
	horseName string
	colorKey  string
	manage    bool // owner or admin
	rider     bool
	rules     []string // horse_riders.rules of the user, empty for non-riders
}

func (a access) has(keys ...string) bool {
	for _, k := range keys {
		if slices.Contains(a.rules, k) {
			return true
		}
	}
	return false
}

// canRead: owner, admin and riders may see profile, today, week and sessions.
func (a access) canRead() bool { return a.manage || a.rider }

// canLog: owner, admin, or a rider with log_sessions.
func (a access) canLog() bool {
	return a.manage || (a.rider && a.has(RuleLogSessions, ruleRide))
}

// canTakeSlots: owner, admin, or a rider with take_week_slots.
func (a access) canTakeSlots() bool {
	return a.manage || (a.rider && a.has(RuleTakeWeekSlots, ruleRide))
}

// authorize resolves the horse of the path and the caller's role. It writes the error
// response itself (404 for a horse outside the stable, 403 for members without a role).
func (h *handler) authorize(w http.ResponseWriter, r *http.Request) (access, bool) {
	user, _ := auth.UserFrom(r.Context())
	a, err := h.resolve(r.Context(), user, r.PathValue("id"))
	switch {
	case errors.Is(err, errNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "horse not found")
		return access{}, false
	case err != nil:
		h.fail(w, r, err)
		return access{}, false
	case !a.canRead():
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner, admins and riders can see the training of this horse")
		return access{}, false
	}
	return a, true
}

var errNotFound = errors.New("trainingapi: not found")

func (h *handler) resolve(ctx context.Context, user auth.User, horseID string) (access, error) {
	a := access{user: user, horseID: horseID}
	var color *string
	err := h.deps.Pool.QueryRow(ctx, `SELECT name, color_key FROM horses WHERE id = $1 AND stable_id = $2`,
		horseID, user.StableID).Scan(&a.horseName, &color)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return access{}, errNotFound
	}
	if err != nil {
		return access{}, err
	}
	if color != nil {
		a.colorKey = *color
	}
	if a.manage, err = auth.CanManageHorse(ctx, h.deps.Pool, user, horseID); err != nil {
		return access{}, err
	}
	rules, isRider, err := auth.RiderRules(ctx, h.deps.Pool, user, horseID)
	if err != nil && !isRider {
		return access{}, err
	}
	// A rider whose rule list cannot be read has no rules (fail closed).
	a.rider, a.rules = isRider, rules
	return a, nil
}

func isInvalidUUID(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "22P02"
}

func (h *handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	h.deps.Log.ErrorContext(r.Context(), "training request failed", "path", r.URL.Path, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func invalid(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusBadRequest, "validation_failed", msg)
}

// location returns the stable's timezone (Europe/Berlin if unset or unknown).
func (h *handler) location(ctx context.Context, stableID string) *time.Location {
	var tz string
	_ = h.deps.Pool.QueryRow(ctx, `SELECT timezone FROM stables WHERE id = $1`, stableID).Scan(&tz)
	if tz == "" {
		tz = defaultTimezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation(defaultTimezone)
	}
	if loc == nil {
		loc = time.UTC
	}
	return loc
}

// localMidnight is 00:00 of the calendar day of day (a UTC-normalised date) in loc.
func localMidnight(day time.Time, loc *time.Location) time.Time {
	y, m, d := day.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// mondayOf returns the Monday of the week containing day (UTC-normalised dates).
func mondayOf(day time.Time) time.Time {
	return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
}

const dateLayout = "2006-01-02"
