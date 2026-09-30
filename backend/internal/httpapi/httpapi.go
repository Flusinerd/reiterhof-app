// Package httpapi assembles the HTTP router of the cloud API.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/devices"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/horses"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Deps are the dependencies handed to every domain package (see httpx.Deps).
type Deps = httpx.Deps

// registrations lists the Register function of every domain package.
// Add one line per domain, e.g. horses.Register.
var registrations = []func(mux *http.ServeMux, deps Deps){
	auth.Register,
	devices.Register,
	files.Register,
	horses.Register,
}

// NewHandler builds the router. It uses net/http only, no framework.
func NewHandler(deps Deps) http.Handler {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	if deps.Notify == nil {
		deps.Notify = httpx.NopNotifier{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", readyz(deps))
	for _, register := range registrations {
		register(mux, deps)
	}
	// auth.Middleware puts the current user into the request context (never rejects).
	return auth.Middleware(deps)(mux)
}

// healthz reports that the process is up; it does not touch the database.
func healthz(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz reports whether the API can serve requests, i.e. the database answers.
func readyz(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if deps.Pool == nil || deps.Pool.Ping(ctx) != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "database_unavailable", "database is not reachable")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
