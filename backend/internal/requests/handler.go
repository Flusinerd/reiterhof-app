package requests

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// DefaultPublish is the realtime extension point: when set before the router is
// built, every committed change is reported to it (see Event). Nil is a no-op.
var DefaultPublish func(ctx context.Context, e Event)

// NewService builds the service from the shared dependencies.
func NewService(deps httpx.Deps) *Service {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &Service{Pool: deps.Pool, Notify: deps.Notify, Log: deps.Log, Now: now, Publish: DefaultPublish}
}

// Register adds the /api/v1/requests routes (all behind auth.RequireStable).
func Register(mux *http.ServeMux, deps httpx.Deps) {
	RegisterService(mux, NewService(deps))
}

// RegisterService adds the routes for an existing service.
func RegisterService(mux *http.ServeMux, svc *Service) {
	h := &handler{svc: svc}
	route := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, auth.RequireStable(fn))
	}
	route("GET /api/v1/requests", h.list)
	route("POST /api/v1/requests", h.create)
	route("GET /api/v1/requests/options", h.options)
	route("GET /api/v1/requests/calendar", h.calendar)
	route("GET /api/v1/requests/calendar.ics", h.calendarICS)
	route("GET /api/v1/requests/notify-settings", h.getNotify)
	route("PATCH /api/v1/requests/notify-settings", h.setNotify)
	// {id} also serves "<id>.ics" (a path wildcard must be a whole segment).
	route("GET /api/v1/requests/{id}", h.get)
	route("PATCH /api/v1/requests/{id}", h.update)
	route("POST /api/v1/requests/{id}/accept", h.accept)
	route("POST /api/v1/requests/{id}/withdraw", h.withdraw)
	route("POST /api/v1/requests/{id}/done", h.done)
	route("POST /api/v1/requests/{id}/cancel", h.cancel)
	route("POST /api/v1/requests/{id}/thanks/{userId}", h.thank)
	route("GET /api/v1/me/thanks", h.thanks)
}

type handler struct{ svc *Service }

func (h *handler) fail(w http.ResponseWriter, err error) {
	var ae *apiError
	if errors.As(err, &ae) {
		httpx.WriteError(w, ae.Status, ae.Code, ae.Msg)
		return
	}
	h.svc.log().Error("requests", "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
}

func user(r *http.Request) auth.User {
	u, _ := auth.UserFrom(r.Context())
	return u
}

var validStatus = map[string]bool{StatusOpen: true, StatusAssigned: true, StatusDone: true, StatusCancelled: true}

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// list handles GET /api/v1/requests?status=open,assigned&type=&mine=true&assigned=true&horse_id=&from=&to=&limit=
// Without status, cancelled requests are left out.
func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f ListFilter
	if s := q.Get("status"); s != "" {
		for _, st := range strings.Split(s, ",") {
			if !validStatus[st] {
				httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "status must be open, assigned, done or cancelled")
				return
			}
			f.Statuses = append(f.Statuses, st)
		}
	} else {
		f.Statuses = []string{StatusOpen, StatusAssigned, StatusDone}
	}
	if t := q.Get("type"); t != "" {
		if !ValidType(t) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "unknown type")
			return
		}
		f.Type = t
	}
	if hid := q.Get("horse_id"); hid != "" {
		if !validUUID(hid) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "horse_id is not a valid id")
			return
		}
		f.HorseID = hid
	}
	f.Mine = q.Get("mine") == "true"
	f.Assigned = q.Get("assigned") == "true"
	for name, dst := range map[string]*string{"from": &f.From, "to": &f.To} {
		if v := q.Get(name); v != "" {
			if !validDate(v) {
				httpx.WriteError(w, http.StatusBadRequest, "validation_failed", name+" must be YYYY-MM-DD")
				return
			}
			*dst = v
		}
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "limit must be a positive number")
			return
		}
		f.Limit = n
	}
	// Finished lists read newest first.
	f.Desc = len(f.Statuses) > 0 && !contains(f.Statuses, StatusOpen) && !contains(f.Statuses, StatusAssigned)

	out, open, err := h.svc.List(r.Context(), user(r), f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"requests": out, "open_count": open})
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	req, err := h.svc.Create(r.Context(), user(r), in)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, req)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if base, ok := strings.CutSuffix(id, ".ics"); ok {
		h.eventICS(w, r, base)
		return
	}
	req, err := h.svc.Get(r.Context(), user(r), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, req)
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	var p Patch
	if !httpx.ReadJSON(w, r, &p) {
		return
	}
	req, err := h.svc.Update(r.Context(), user(r), r.PathValue("id"), p)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, req)
}

func (h *handler) accept(w http.ResponseWriter, r *http.Request) {
	req, err := h.svc.Accept(r.Context(), user(r), r.PathValue("id"))
	h.respond(w, req, err)
}

func (h *handler) withdraw(w http.ResponseWriter, r *http.Request) {
	req, err := h.svc.Withdraw(r.Context(), user(r), r.PathValue("id"))
	h.respond(w, req, err)
}

func (h *handler) done(w http.ResponseWriter, r *http.Request) {
	req, err := h.svc.Done(r.Context(), user(r), r.PathValue("id"))
	h.respond(w, req, err)
}

// cancel takes an optional body {"scope": "one" | "series"}.
func (h *handler) cancel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Scope string `json:"scope"`
	}
	if r.ContentLength != 0 && !httpx.ReadJSON(w, r, &body) {
		return
	}
	req, err := h.svc.Cancel(r.Context(), user(r), r.PathValue("id"), body.Scope)
	h.respond(w, req, err)
}

func (h *handler) respond(w http.ResponseWriter, req Request, err error) {
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, req)
}

func (h *handler) thank(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Thank(r.Context(), user(r), r.PathValue("id"), r.PathValue("userId")); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) thanks(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.ThanksCount(r.Context(), user(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"count": n})
}

func (h *handler) options(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.Options(r.Context(), user(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

func (h *handler) calendarRange(w http.ResponseWriter, r *http.Request) ([]Request, bool) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	for _, v := range []string{from, to} {
		if v != "" && !validDate(v) {
			httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "from and to must be YYYY-MM-DD")
			return nil, false
		}
	}
	out, err := h.svc.Calendar(r.Context(), user(r), from, to)
	if err != nil {
		h.fail(w, err)
		return nil, false
	}
	return out, true
}

// calendar handles GET /api/v1/requests/calendar: my accepted requests, upcoming first.
func (h *handler) calendar(w http.ResponseWriter, r *http.Request) {
	out, ok := h.calendarRange(w, r)
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"requests": out})
}

func (h *handler) writeICS(w http.ResponseWriter, u auth.User, reqs []Request, name string) {
	loc := stableLocation(context.Background(), h.svc.Pool, u.StableID)
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="`+name+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(BuildICS(reqs, loc, h.svc.now())))
}

// calendarICS handles GET /api/v1/requests/calendar.ics.
func (h *handler) calendarICS(w http.ResponseWriter, r *http.Request) {
	out, ok := h.calendarRange(w, r)
	if !ok {
		return
	}
	h.writeICS(w, user(r), out, "reiterhof-anfragen.ics")
}

// eventICS handles GET /api/v1/requests/{id}.ics for any request of the stable.
func (h *handler) eventICS(w http.ResponseWriter, r *http.Request, id string) {
	req, err := h.svc.Get(r.Context(), user(r), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeICS(w, user(r), []Request{req}, "anfrage-"+req.ID+".ics")
}

func (h *handler) getNotify(w http.ResponseWriter, r *http.Request) {
	on, err := h.svc.NewRequestPush(r.Context(), user(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"new_request": on})
}

func (h *handler) setNotify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		NewRequest *bool `json:"new_request"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.NewRequest == nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "new_request (boolean) is required")
		return
	}
	if err := h.svc.SetNewRequestPush(r.Context(), user(r), *in.NewRequest); err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"new_request": *in.NewRequest})
}
