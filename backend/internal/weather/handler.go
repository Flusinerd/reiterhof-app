package weather

import (
	"net/http"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Register adds the weather routes.
//
//	GET /api/v1/weather/warnings   active DWD warnings for the stable (all members)
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	mux.Handle("GET /api/v1/weather/warnings", auth.RequireStable(http.HandlerFunc(h.warnings)))
}

type handler struct{ deps httpx.Deps }

// WarningsResponse is the body of GET /weather/warnings.
type WarningsResponse struct {
	// Warnings are the active ones, most severe first; empty when there are none, the
	// stable has no coordinates or the warning job is off.
	Warnings []StoredWarning `json:"warnings"`
}

func (h *handler) warnings(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	ws, err := (&Store{Pool: h.deps.Pool}).ActiveWarnings(r.Context(), user.StableID, h.deps.Now())
	if err != nil {
		h.deps.Log.Error("weather: warnings", "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, WarningsResponse{Warnings: ws})
}
