package reha

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// Rider rule keys (horse_riders.rules) that allow marking a day done; same as the training
// feature (log_sessions, and the legacy key ride of the first seed data).
const (
	ruleLogSessions = "log_sessions"
	ruleRide        = "ride"
)

const defaultTimezone = "Europe/Berlin"

// Register mounts the reha routes under /api/v1.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	route := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, auth.RequireStable(fn))
	}
	route("GET /api/v1/horses/{id}/reha", h.get)
	route("POST /api/v1/horses/{id}/reha-plans", h.create)
	route("PATCH /api/v1/reha-plans/{id}", h.patch)
	route("POST /api/v1/reha-plans/{id}/end", h.end)
	route("POST /api/v1/reha-plans/{id}/days/{date}/done", h.markDone)
	route("DELETE /api/v1/reha-plans/{id}/days/{date}/done", h.unmarkDone)
}

type handler struct {
	deps httpx.Deps
}

// access is what the caller may do with one horse's reha plan.
type access struct {
	user      auth.User
	horseID   string
	horseName string
	manage    bool // owner or admin
	rider     bool
	rules     []string
}

// full: owner, admin and riders see the whole plan; other members only today's rule.
func (a access) full() bool { return a.manage || a.rider }

// canMark: owner, admin, or a rider with log_sessions.
func (a access) canMark() bool {
	return a.manage || (a.rider && (slices.Contains(a.rules, ruleLogSessions) || slices.Contains(a.rules, ruleRide)))
}

var errNotFound = errors.New("reha: not found")

func (h *handler) resolve(ctx context.Context, user auth.User, horseID string) (access, error) {
	a := access{user: user, horseID: horseID}
	err := h.deps.Pool.QueryRow(ctx, `SELECT id::text, name FROM horses WHERE id::text = $1 AND stable_id = $2`,
		horseID, user.StableID).Scan(&a.horseID, &a.horseName)
	if errors.Is(err, pgx.ErrNoRows) {
		return access{}, errNotFound
	}
	if err != nil {
		return access{}, err
	}
	if a.manage, err = auth.CanManageHorse(ctx, h.deps.Pool, user, a.horseID); err != nil {
		return access{}, err
	}
	rules, isRider, err := auth.RiderRules(ctx, h.deps.Pool, user, a.horseID)
	if err != nil && !isRider {
		return access{}, err
	}
	a.rider, a.rules = isRider, rules
	return a, nil
}

// horse resolves the horse of the path, writing the error response itself.
func (h *handler) horse(w http.ResponseWriter, r *http.Request) (access, bool) {
	user, _ := auth.UserFrom(r.Context())
	a, err := h.resolve(r.Context(), user, r.PathValue("id"))
	switch {
	case errors.Is(err, errNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "horse not found")
		return access{}, false
	case err != nil:
		h.fail(w, r, err)
		return access{}, false
	}
	return a, true
}

// plan resolves the plan of the path and the caller's access to its horse.
func (h *handler) plan(w http.ResponseWriter, r *http.Request) (*Plan, access, bool) {
	user, _ := auth.UserFrom(r.Context())
	p, err := PlanByID(r.Context(), h.deps.Pool, user.StableID, r.PathValue("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "reha plan not found")
		return nil, access{}, false
	}
	if err != nil {
		h.fail(w, r, err)
		return nil, access{}, false
	}
	a, err := h.resolve(r.Context(), user, p.HorseID)
	if err != nil {
		h.fail(w, r, err)
		return nil, access{}, false
	}
	return p, a, true
}

func (h *handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	h.deps.Log.ErrorContext(r.Context(), "reha request failed", "path", r.URL.Path, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func invalid(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusBadRequest, "validation_failed", msg)
}

func forbidden(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusForbidden, "forbidden", msg)
}

// location returns the stable's timezone (Europe/Berlin if unset or unknown).
func (h *handler) location(ctx context.Context, stableID string) *time.Location {
	var tz string
	_ = h.deps.Pool.QueryRow(ctx, `SELECT timezone FROM stables WHERE id = $1`, stableID).Scan(&tz)
	return LoadLocation(tz)
}

// LoadLocation loads a timezone name, falling back to Europe/Berlin and then UTC.
func LoadLocation(tz string) *time.Location {
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

func (h *handler) today(ctx context.Context, stableID string) (time.Time, *time.Location) {
	loc := h.location(ctx, stableID)
	return training.Day(h.deps.Now().In(loc)), loc
}

// get: GET /api/v1/horses/{id}/reha
func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	a, ok := h.horse(w, r)
	if !ok {
		return
	}
	h.respond(w, r, http.StatusOK, a)
}

func (h *handler) respond(w http.ResponseWriter, r *http.Request, status int, a access) {
	today, loc := h.today(r.Context(), a.user.StableID)
	v, err := h.buildView(r.Context(), a, today, loc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, status, v)
}

// --- write side ---------------------------------------------------------------------------

type createBody struct {
	Diagnosis     string  `json:"diagnosis"`
	Vet           string  `json:"vet"`
	StartDate     string  `json:"start_date"`
	Phases        []Phase `json:"phases"`
	CheckupDate   string  `json:"checkup_date"`
	AbortCriteria string  `json:"abort_criteria"`
	ObservationID *string `json:"observation_id"`
}

// planInput is the validated state of a plan that is stored.
type planInput struct {
	Diagnosis     string
	Vet           string
	Start         time.Time
	Phases        []Phase
	Checkup       *time.Time
	AbortCriteria string
	ObservationID *string
}

func parseDate(s string) (time.Time, bool) {
	d, err := time.Parse(DateLayout, s)
	return d, err == nil
}

// validate trims and checks the plan; the message is English.
func (in *planInput) validate(today time.Time) string {
	in.Diagnosis = strings.TrimSpace(in.Diagnosis)
	in.Vet = strings.TrimSpace(in.Vet)
	in.AbortCriteria = strings.TrimSpace(in.AbortCriteria)
	switch {
	case in.Diagnosis == "" || len([]rune(in.Diagnosis)) > 200:
		return "diagnosis is required (max 200 characters)"
	case len([]rune(in.Vet)) > 120:
		return "vet: max 120 characters"
	case len([]rune(in.AbortCriteria)) > 1000:
		return "abort_criteria: max 1000 characters"
	case in.Start.Year() < 2000 || in.Start.After(today.AddDate(1, 0, 0)):
		return "start_date must lie within the next year and after 1999"
	}
	phases, err := ValidatePhases(in.Phases)
	if err != nil {
		return err.Error()
	}
	in.Phases = phases
	if in.Checkup != nil && in.Checkup.Before(in.Start) {
		return "checkup_date must not be before start_date"
	}
	return ""
}

// checkObservation makes sure the observation belongs to the horse (and the stable).
func (h *handler) checkObservation(ctx context.Context, q DB, stableID, horseID string, id *string) (bool, error) {
	if id == nil {
		return true, nil
	}
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM observations WHERE id::text = $1 AND stable_id = $2 AND horse_id = $3::uuid)`,
		*id, stableID, horseID).Scan(&ok)
	return ok, err
}

// create: POST /api/v1/horses/{id}/reha-plans
func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	a, ok := h.horse(w, r)
	if !ok {
		return
	}
	if !a.manage {
		forbidden(w, "only the owner or an admin can create a reha plan")
		return
	}
	var body createBody
	if !httpx.ReadJSON(w, r, &body) {
		return
	}
	ctx := r.Context()
	today, _ := h.today(ctx, a.user.StableID)
	in := planInput{Diagnosis: body.Diagnosis, Vet: body.Vet, Phases: body.Phases, AbortCriteria: body.AbortCriteria, ObservationID: blankToNil(body.ObservationID)}
	var okDate bool
	if in.Start, okDate = parseDate(body.StartDate); !okDate {
		invalid(w, "start_date must be YYYY-MM-DD")
		return
	}
	if body.CheckupDate != "" {
		d, okc := parseDate(body.CheckupDate)
		if !okc {
			invalid(w, "checkup_date must be YYYY-MM-DD")
			return
		}
		in.Checkup = &d
	}
	if msg := in.validate(today); msg != "" {
		invalid(w, msg)
		return
	}
	if found, err := h.checkObservation(ctx, h.deps.Pool, a.user.StableID, a.horseID, in.ObservationID); err != nil {
		h.fail(w, r, err)
		return
	} else if !found {
		invalid(w, "observation_id: observation not found for this horse")
		return
	}

	err := h.tx(ctx, func(tx pgx.Tx) error {
		// The horse row serialises concurrent creates for the same horse.
		if _, err := tx.Exec(ctx, `SELECT 1 FROM horses WHERE id = $1::uuid AND stable_id = $2 FOR UPDATE`, a.horseID, a.user.StableID); err != nil {
			return err
		}
		// Only one plan is active per horse: the previous one ends now.
		if _, err := tx.Exec(ctx, `UPDATE reha_plans SET active = false, ended_at = $3
			WHERE stable_id = $1 AND horse_id = $2::uuid AND active`, a.user.StableID, a.horseID, h.deps.Now()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO reha_plans (stable_id, horse_id, diagnosis, vet, start_date, phases, checkup_date,
				abort_criteria, observation_id, active, created_by)
			VALUES ($1, $2::uuid, $3, NULLIF($4, ''), $5, $6::jsonb, $7, NULLIF($8, ''), $9::uuid, true, $10::uuid)`,
			a.user.StableID, a.horseID, in.Diagnosis, in.Vet, in.Start, PhasesJSON(in.Phases), in.Checkup,
			in.AbortCriteria, in.ObservationID, a.user.ID); err != nil {
			return err
		}
		// The horse is in rehab now (the profile row is created if the horse has none yet).
		if _, err := tx.Exec(ctx, `INSERT INTO training_profiles (stable_id, horse_id, status) VALUES ($1, $2::uuid, 'reha')
			ON CONFLICT (horse_id) DO UPDATE SET status = 'reha'`, a.user.StableID, a.horseID); err != nil {
			return err
		}
		return realtime.Publish(ctx, tx, a.user.StableID, "reha.changed", map[string]any{"horse_id": a.horseID})
	})
	if isUnique(err) {
		httpx.WriteError(w, http.StatusConflict, "conflict", "the horse already has an active reha plan")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.respond(w, r, http.StatusCreated, a)
}

type patchBody struct {
	Diagnosis     *string  `json:"diagnosis"`
	Vet           *string  `json:"vet"`
	StartDate     *string  `json:"start_date"`
	Phases        *[]Phase `json:"phases"`
	CheckupDate   *string  `json:"checkup_date"` // "" clears
	AbortCriteria *string  `json:"abort_criteria"`
}

// patch: PATCH /api/v1/reha-plans/{id}. Absent fields stay; "" clears vet, checkup_date and
// abort_criteria. Only the active plan can be changed.
func (h *handler) patch(w http.ResponseWriter, r *http.Request) {
	p, a, ok := h.plan(w, r)
	if !ok {
		return
	}
	if !a.manage {
		forbidden(w, "only the owner or an admin can change a reha plan")
		return
	}
	var body patchBody
	if !httpx.ReadJSON(w, r, &body) {
		return
	}
	ctx := r.Context()
	today, _ := h.today(ctx, a.user.StableID)

	err := h.tx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT 1 FROM horses WHERE id = $1::uuid AND stable_id = $2 FOR UPDATE`, a.horseID, a.user.StableID); err != nil {
			return err
		}
		cur, err := PlanByID(ctx, tx, a.user.StableID, p.ID)
		if err != nil {
			return err
		}
		if !cur.Active {
			return &apiError{http.StatusConflict, "not_active", "the reha plan has ended and can no longer be changed"}
		}
		in := planInput{Diagnosis: cur.Diagnosis, Vet: cur.Vet, Start: cur.Start, Phases: cur.Phases, Checkup: cur.CheckupDate, AbortCriteria: cur.AbortCriteria}
		if body.Diagnosis != nil {
			in.Diagnosis = *body.Diagnosis
		}
		if body.Vet != nil {
			in.Vet = *body.Vet
		}
		if body.AbortCriteria != nil {
			in.AbortCriteria = *body.AbortCriteria
		}
		if body.Phases != nil {
			in.Phases = *body.Phases
		}
		if body.StartDate != nil {
			d, okd := parseDate(*body.StartDate)
			if !okd {
				return &apiError{http.StatusBadRequest, "validation_failed", "start_date must be YYYY-MM-DD"}
			}
			in.Start = d
		}
		if body.CheckupDate != nil {
			in.Checkup = nil
			if *body.CheckupDate != "" {
				d, okd := parseDate(*body.CheckupDate)
				if !okd {
					return &apiError{http.StatusBadRequest, "validation_failed", "checkup_date must be YYYY-MM-DD"}
				}
				in.Checkup = &d
			}
		}
		// A start date that did not change may lie further back than a new plan may.
		checkToday := today
		if body.StartDate == nil {
			checkToday = in.Start
		}
		if msg := in.validate(checkToday); msg != "" {
			return &apiError{http.StatusBadRequest, "validation_failed", msg}
		}
		_, err = tx.Exec(ctx, `UPDATE reha_plans SET diagnosis = $3, vet = NULLIF($4, ''), start_date = $5, phases = $6::jsonb,
				checkup_date = $7, abort_criteria = NULLIF($8, '')
			WHERE stable_id = $1 AND id = $2::uuid`,
			a.user.StableID, cur.ID, in.Diagnosis, in.Vet, in.Start, PhasesJSON(in.Phases), in.Checkup, in.AbortCriteria)
		if err != nil {
			return err
		}
		return realtime.Publish(ctx, tx, a.user.StableID, "reha.changed", map[string]any{"horse_id": a.horseID})
	})
	if h.failAPI(w, r, err) {
		return
	}
	h.respond(w, r, http.StatusOK, a)
}

// end: POST /api/v1/reha-plans/{id}/end. Idempotent. The training status goes back to fit,
// but only if it is still reha: an owner who switched the horse to pause in the meantime keeps
// that choice.
func (h *handler) end(w http.ResponseWriter, r *http.Request) {
	p, a, ok := h.plan(w, r)
	if !ok {
		return
	}
	if !a.manage {
		forbidden(w, "only the owner or an admin can end a reha plan")
		return
	}
	ctx := r.Context()
	err := h.tx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT 1 FROM horses WHERE id = $1::uuid AND stable_id = $2 FOR UPDATE`, a.horseID, a.user.StableID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE reha_plans SET active = false, ended_at = $3
			WHERE stable_id = $1 AND id = $2::uuid AND active`, a.user.StableID, p.ID, h.deps.Now())
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE training_profiles SET status = 'fit'
			WHERE stable_id = $1 AND horse_id = $2::uuid AND status = 'reha'`, a.user.StableID, a.horseID); err != nil {
			return err
		}
		return realtime.Publish(ctx, tx, a.user.StableID, "reha.changed", map[string]any{"horse_id": a.horseID})
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.respond(w, r, http.StatusOK, a)
}

// day resolves the plan and the {date} of a done route and checks that marking is allowed.
func (h *handler) dayRoute(w http.ResponseWriter, r *http.Request) (*Plan, access, time.Time, bool) {
	p, a, ok := h.plan(w, r)
	if !ok {
		return nil, access{}, time.Time{}, false
	}
	if !a.canMark() {
		forbidden(w, "only the owner, admins and riders with log_sessions can mark a day")
		return nil, access{}, time.Time{}, false
	}
	day, okd := parseDate(r.PathValue("date"))
	if !okd {
		invalid(w, "date must be YYYY-MM-DD")
		return nil, access{}, time.Time{}, false
	}
	if !p.Active {
		httpx.WriteError(w, http.StatusConflict, "not_active", "the reha plan has ended")
		return nil, access{}, time.Time{}, false
	}
	return p, a, day, true
}

// markDone: POST /api/v1/reha-plans/{id}/days/{date}/done ("Heute erledigt"). Idempotent; the
// first person who marked the day stays recorded.
func (h *handler) markDone(w http.ResponseWriter, r *http.Request) {
	p, a, day, ok := h.dayRoute(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	today, _ := h.today(ctx, a.user.StableID)
	if day.After(today) {
		invalid(w, "date must not be in the future")
		return
	}
	if _, in := Locate(p.Start, p.Phases, day); !in {
		invalid(w, "date lies outside the phases of the plan")
		return
	}
	err := h.tx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO reha_days (stable_id, reha_plan_id, day, done_by) VALUES ($1, $2::uuid, $3, $4::uuid)
			ON CONFLICT (reha_plan_id, day) DO NOTHING`, a.user.StableID, p.ID, day, a.user.ID); err != nil {
			return err
		}
		return realtime.Publish(ctx, tx, a.user.StableID, "reha.changed", map[string]any{"horse_id": a.horseID})
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.respond(w, r, http.StatusOK, a)
}

// unmarkDone: DELETE /api/v1/reha-plans/{id}/days/{date}/done. The owner, admins and the person
// who marked the day may take it back.
func (h *handler) unmarkDone(w http.ResponseWriter, r *http.Request) {
	p, a, day, ok := h.dayRoute(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	tag, err := h.deps.Pool.Exec(ctx, `DELETE FROM reha_days WHERE stable_id = $1 AND reha_plan_id = $2::uuid AND day = $3
		AND ($4 OR done_by = $5::uuid)`, a.user.StableID, p.ID, day, a.manage, a.user.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		_ = h.deps.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM reha_days WHERE reha_plan_id = $1::uuid AND day = $2)`, p.ID, day).Scan(&exists)
		if exists {
			forbidden(w, "only the owner or the person who marked the day can undo it")
			return
		}
	} else {
		_ = realtime.Publish(ctx, h.deps.Pool, a.user.StableID, "reha.changed", map[string]any{"horse_id": a.horseID})
	}
	h.respond(w, r, http.StatusOK, a)
}

// --- helpers ------------------------------------------------------------------------------

type apiError struct {
	status int
	code   string
	msg    string
}

func (e *apiError) Error() string { return e.code + ": " + e.msg }

// failAPI writes err (an apiError or an internal error) and reports whether there was one.
func (h *handler) failAPI(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	var ae *apiError
	switch {
	case errors.As(err, &ae):
		httpx.WriteError(w, ae.status, ae.code, ae.msg)
	case isUnique(err):
		httpx.WriteError(w, http.StatusConflict, "conflict", "the horse already has an active reha plan")
	default:
		h.fail(w, r, err)
	}
	return true
}

func (h *handler) tx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := h.deps.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isUnique(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}

func blankToNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}
