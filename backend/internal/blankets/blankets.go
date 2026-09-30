// Package blankets implements the blanket feature (M3): blankets per horse with photo,
// location and fill weight, the ordered blanket rules an owner maintains, tonight's
// recommendation, the daily blanket state (covered / uncovered / checked) with history,
// the "Decken heute" overview, the last-person reminder and the weather change push.
//
// The recommendation itself is the pure logic in internal/blanketplan. "Day" means the
// blanket night that starts on that stable-local date, see NightDay. Rules and
// endpoints are documented in docs/domains/blankets.md.
package blankets

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

// Realtime event types (data carries ids only).
const (
	// EventStateChanged is published after a day state was recorded: {horse_id, day}.
	EventStateChanged = "blanket_state.changed"
	// EventPlanChanged is published after blankets or rules of a horse changed: {horse_id}.
	EventPlanChanged = "blanket_plan.changed"
)

// Day states (blanket_states.action).
const (
	ActionCovered   = "covered"
	ActionUncovered = "uncovered"
	ActionChecked   = "checked"
)

// Limits.
const (
	maxRules       = 30
	maxNameLen     = 80
	maxColorLen    = 40
	maxLocationLen = 80
	maxNoteLen     = 200
	maxFillG       = 2000
	tempLimit      = 60.0
	rainLimitMM    = 500.0
	defaultDays    = 14
	maxDays        = 90
)

// Register mounts the blanket routes. All of them need a signed-in user with a stable.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{svc: NewService(deps), deps: deps}
	route := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, auth.RequireStable(fn))
	}
	route("GET /api/v1/blankets/today", h.today)

	route("GET /api/v1/horses/{id}/blankets", h.listBlankets)
	route("POST /api/v1/horses/{id}/blankets", h.createBlanket)
	route("PATCH /api/v1/horses/{id}/blankets/{blanketId}", h.patchBlanket)
	route("DELETE /api/v1/horses/{id}/blankets/{blanketId}", h.deleteBlanket)

	route("GET /api/v1/horses/{id}/blanket-rules", h.getRules)
	route("PUT /api/v1/horses/{id}/blanket-rules", h.putRules)
	route("GET /api/v1/horses/{id}/blanket-plan", h.plan)
	route("PUT /api/v1/horses/{id}/cover-window", h.putCoverWindow)

	route("POST /api/v1/horses/{id}/blanket-state", h.setState)
	route("GET /api/v1/horses/{id}/blanket-states", h.history)
}

// Service holds the logic shared by the HTTP handlers, the jobs and the presence hook.
type Service struct {
	Pool   *pgxpool.Pool
	Notify httpx.Notifier
	Log    *slog.Logger
	Now    func() time.Time
}

// NewService builds a Service from the shared dependencies.
func NewService(deps httpx.Deps) *Service {
	s := &Service{Pool: deps.Pool, Notify: deps.Notify, Log: deps.Log, Now: deps.Now}
	if s.Notify == nil {
		s.Notify = httpx.NopNotifier{}
	}
	return s
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

type handler struct {
	svc  *Service
	deps httpx.Deps
}

func (h *handler) internal(w http.ResponseWriter, what string, err error) {
	h.svc.log().Error("blankets: "+what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func invalid(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusBadRequest, "validation_failed", msg)
}

// access resolves the user and the horse of the request. The horse must be in the
// user's stable (404 otherwise); with manage the user must also be owner or admin (403).
// It writes the error response itself and returns ok=false.
func (h *handler) access(w http.ResponseWriter, r *http.Request, manage bool) (user auth.User, horseID string, ok bool) {
	user, _ = auth.UserFrom(r.Context())
	horseID = r.PathValue("id")
	in, err := auth.HorseInStable(r.Context(), h.deps.Pool, user, horseID)
	if err != nil {
		h.internal(w, "horse lookup", err)
		return user, "", false
	}
	if !in {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "horse not found")
		return user, "", false
	}
	if manage {
		can, err := auth.CanManageHorse(r.Context(), h.deps.Pool, user, horseID)
		if err != nil {
			h.internal(w, "role lookup", err)
			return user, "", false
		}
		if !can {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner or an admin may change blankets and rules")
			return user, "", false
		}
	}
	return user, horseID, true
}

// text validates an optional text field: trimmed, at most max characters.
func text(v *string, max int) (string, bool) {
	if v == nil {
		return "", true
	}
	s := strings.TrimSpace(*v)
	return s, utf8.RuneCountInString(s) <= max
}

func validTemp(v *float64) bool {
	return v == nil || (!math.IsNaN(*v) && !math.IsInf(*v, 0) && *v >= -tempLimit && *v <= tempLimit)
}

func validRainMM(v *float64) bool {
	return v == nil || (!math.IsNaN(*v) && !math.IsInf(*v, 0) && *v >= 0 && *v <= rainLimitMM)
}

var errNotFound = errors.New("not found")

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// hint sends a realtime event outside a transaction; failures are only logged because
// the data is already committed.
func (s *Service) hint(ctx context.Context, stableID, typ string, data map[string]any) {
	if err := realtime.Publish(ctx, s.Pool, stableID, typ, data); err != nil {
		s.log().Warn("blankets: publish event", "type", typ, "err", err)
	}
}

func photoURL(path *string) *string {
	if path == nil || *path == "" {
		return nil
	}
	u := files.URLFor(*path)
	return &u
}
