// Package reminders is the reminder center and the notification settings (M8, JAN-69,
// JAN-70, JAN-18): the list of a user's reminders, the switches per push kind, the
// stable's reminder time and the evening push of the training plan. Details and
// decisions are documented in docs/domains/reminders.md.
package reminders

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Register adds the reminder center, the notification settings and the reminder time.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	route := func(pattern string, fn http.HandlerFunc) { mux.Handle(pattern, auth.RequireStable(fn)) }
	route("GET /api/v1/reminders", h.list)
	route("POST /api/v1/reminders/{id}/dismiss", h.dismiss)
	route("GET /api/v1/settings/notifications", h.getNotifications)
	route("PUT /api/v1/settings/notifications/{kind}", h.putNotification)
	route("GET /api/v1/stables/reminder-time", h.getReminderTime)
	mux.Handle("PUT /api/v1/stables/reminder-time", auth.RequireAdmin(http.HandlerFunc(h.putReminderTime)))
}

type handler struct{ deps httpx.Deps }

func (h *handler) now() time.Time {
	if h.deps.Now != nil {
		return h.deps.Now()
	}
	return time.Now()
}

func (h *handler) log() *slog.Logger {
	if h.deps.Log != nil {
		return h.deps.Log
	}
	return slog.Default()
}

func (h *handler) internal(w http.ResponseWriter, what string, err error) {
	h.log().Error("reminders: "+what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
}

func user(r *http.Request) auth.User {
	u, _ := auth.UserFrom(r.Context())
	return u
}
