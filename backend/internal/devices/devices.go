// Package devices lets signed-in users register the native push tokens (APNs, FCM) and
// the Web Push subscriptions of their devices.
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
	// Web push needs the VAPID key pair; without it the web endpoints answer 503 not_configured.
	vapid, err := push.VAPIDFromEnv()
	if err != nil && !errors.Is(err, push.ErrVAPIDNotConfigured) {
		deps.Log.Error("web push disabled: invalid VAPID configuration", "err", err)
	}
	h.vapid = vapid
	mux.Handle("POST /api/v1/me/push-tokens", auth.RequireStable(http.HandlerFunc(h.register)))
	mux.Handle("DELETE /api/v1/me/push-tokens", auth.RequireStable(http.HandlerFunc(h.unregister)))
	mux.Handle("GET /api/v1/push/web/public-key", auth.RequireStable(http.HandlerFunc(h.webPublicKey)))
	mux.Handle("POST /api/v1/me/web-push-subscriptions", auth.RequireStable(http.HandlerFunc(h.registerWeb)))
	mux.Handle("DELETE /api/v1/me/web-push-subscriptions", auth.RequireStable(http.HandlerFunc(h.unregisterWeb)))
}

type handler struct {
	deps  httpx.Deps
	vapid *push.VAPID // nil: web push is not configured
}

type tokenBody struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

func (h handler) register(w http.ResponseWriter, r *http.Request) {
	var in tokenBody
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if !push.ValidToken(in.Token) || (in.Platform != push.PlatformIOS && in.Platform != push.PlatformAndroid) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_token", "a native device token and platform (ios, android) are required")
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

// subscriptionBody is a browser PushSubscription as PushSubscription.toJSON() returns it.
type subscriptionBody struct {
	Endpoint       string `json:"endpoint"`
	ExpirationTime *int64 `json:"expirationTime"`
	Keys           struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (b subscriptionBody) subscription() push.WebSubscription {
	return push.WebSubscription{Endpoint: b.Endpoint, P256dh: b.Keys.P256dh, Auth: b.Keys.Auth}
}

func (h handler) notConfigured(w http.ResponseWriter) bool {
	if h.vapid != nil {
		return false
	}
	httpx.WriteError(w, http.StatusServiceUnavailable, "not_configured", "web push is not configured on this server")
	return true
}

// webPublicKey returns the VAPID public key (the applicationServerKey for PushManager.subscribe).
func (h handler) webPublicKey(w http.ResponseWriter, _ *http.Request) {
	if h.notConfigured(w) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"public_key": h.vapid.PublicKey()})
}

func (h handler) registerWeb(w http.ResponseWriter, r *http.Request) {
	if h.notConfigured(w) {
		return
	}
	var in subscriptionBody
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	sub, err := push.ValidateWebSubscription(in.subscription())
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_subscription", "endpoint (https) and keys.p256dh, keys.auth are required")
		return
	}
	user, _ := auth.UserFrom(r.Context())
	err = push.RegisterWebSubscription(r.Context(), h.deps.Pool, user.StableID, user.ID, sub, r.UserAgent())
	if errors.Is(err, push.ErrUnknownUser) {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "user is not part of the stable")
		return
	}
	if err != nil {
		h.deps.Log.Error("register web push subscription", "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "could not register the subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// unregisterWeb works without VAPID keys too, so a subscription can always be removed.
func (h handler) unregisterWeb(w http.ResponseWriter, r *http.Request) {
	var in subscriptionBody
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Endpoint == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_subscription", "endpoint is required")
		return
	}
	user, _ := auth.UserFrom(r.Context())
	if _, err := push.DeleteWebSubscription(r.Context(), h.deps.Pool, user.StableID, user.ID, in.Endpoint); err != nil {
		h.deps.Log.Error("delete web push subscription", "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "could not delete the subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
