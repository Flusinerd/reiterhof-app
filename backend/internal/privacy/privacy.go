package privacy

import (
	"errors"
	"net/http"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Register mounts the privacy routes. They need a signed-in user but no stable: a person
// may withdraw consents, export or delete the account before ever joining a stable.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	mux.Handle("GET /api/v1/me/consents", auth.RequireUser(http.HandlerFunc(h.listConsents)))
	mux.Handle("PUT /api/v1/me/consents/{kind}", auth.RequireUser(http.HandlerFunc(h.putConsent)))
	mux.Handle("GET /api/v1/me/export", auth.RequireUser(http.HandlerFunc(h.export)))
	mux.Handle("POST /api/v1/me/delete", auth.RequireUser(http.HandlerFunc(h.deleteAccount)))
}

type handler struct{ deps httpx.Deps }

func (h *handler) internal(w http.ResponseWriter, what string, err error) {
	h.deps.Log.Error("privacy: "+what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func (h *handler) export(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	doc, err := Export(r.Context(), h.deps.Pool, user.ID, h.deps.Now())
	if err != nil {
		h.internal(w, "export", err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="reiterhof-export-`+h.deps.Now().UTC().Format("2006-01-02")+`.json"`)
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, doc)
}

func (h *handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var in struct {
		Confirm bool `json:"confirm"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if !in.Confirm {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "confirm must be true")
		return
	}
	err := DeleteAccount(r.Context(), h.deps.Pool, user.ID, h.deps.Now())
	var owns *OwnsHorsesError
	switch {
	case errors.As(err, &owns):
		httpx.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":  httpx.ErrorDetail{Code: "owns_horses", Message: "transfer or delete your horses before deleting the account"},
			"horses": owns.Horses,
		})
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "user not found")
	case errors.Is(err, ErrLastAdmin):
		httpx.WriteError(w, http.StatusConflict, "last_admin", "the stable needs another admin before you can delete your account")
	case err != nil:
		h.internal(w, "delete account", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// months returns the instant n calendar months before t.
func months(t time.Time, n int) time.Time { return t.AddDate(0, -n, 0) }
