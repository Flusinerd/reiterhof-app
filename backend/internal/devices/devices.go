// Package devices lets signed-in users register the Expo push tokens of their devices.
package devices

import (
	"errors"
	"net/http"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
)

// Register adds the push token routes.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := handler{deps: deps}
	mux.Handle("POST /api/v1/me/push-tokens", auth.RequireStable(http.HandlerFunc(h.register)))
	mux.Handle("DELETE /api/v1/me/push-tokens", auth.RequireStable(http.HandlerFunc(h.unregister)))
}

type handler struct{ deps httpx.Deps }

type tokenBody struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

func (h handler) register(w http.ResponseWriter, r *http.Request) {
	var in tokenBody
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Token == "" || (in.Platform != push.PlatformIOS && in.Platform != push.PlatformAndroid) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_token", "token and platform (ios, android) are required")
		return
	}
	user, _ := auth.UserFrom(r.Context())
	err := push.RegisterToken(r.Context(), h.deps.Pool, user.StableID, user.ID, in.Token, in.Platform)
	if errors.Is(err, push.ErrUnknownUser) {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "user is not part of the stable")
		return
	}
	if err != nil {
		h.deps.Log.Error("register push token", "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "could not register the token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h handler) unregister(w http.ResponseWriter, r *http.Request) {
	var in tokenBody
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	user, _ := auth.UserFrom(r.Context())
	if _, err := push.DeleteToken(r.Context(), h.deps.Pool, user.StableID, user.ID, in.Token); err != nil {
		h.deps.Log.Error("delete push token", "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "could not delete the token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
