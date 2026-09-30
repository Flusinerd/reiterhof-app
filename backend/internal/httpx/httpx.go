// Package httpx holds the dependencies and helpers shared by all HTTP handlers.
//
// Domain packages import httpx (never httpapi, which imports them) to avoid
// import cycles. httpapi re-exports Deps as httpapi.Deps.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
)

// MaxBodyBytes is the size limit applied by ReadJSON.
const MaxBodyBytes = 1 << 20 // 1 MiB

// Deps are the dependencies handed to every domain package.
type Deps struct {
	Pool   *pgxpool.Pool
	Config config.Config
	Log    *slog.Logger
	// Now returns the current time. Use it instead of time.Now so tests can
	// control the clock.
	Now func() time.Time
	// Notify sends push notifications. httpapi.NewHandler replaces nil with a
	// no-op, so handlers can always call it. Tests can pass
	// push.NewNotifier(pool, &push.Fake{}, nil) to inspect what was sent.
	Notify Notifier
	// Events is the subscribe side of the realtime hub (SSE). httpapi.NewHandler
	// replaces nil with a no-op; cmd/api wires the real *realtime.Hub. Publishing
	// does not need it: use realtime.Publish(ctx, q, ...) inside your transaction.
	Events EventBus
}

// Event is one realtime message for the members of a stable.
type Event struct {
	StableID string          `json:"stable_id"`
	Type     string          `json:"type"`
	Data     json.RawMessage `json:"data"`
}

// EventBus hands out event streams; *realtime.Hub implements it.
type EventBus interface {
	// Subscribe returns a channel with the events of one stable and a cancel
	// function (idempotent). The channel is closed when the subscriber is too
	// slow (buffer full) or the hub stops; cancel must still be called.
	Subscribe(stableID string) (events <-chan Event, cancel func())
}

// NopEvents never delivers an event.
type NopEvents struct{}

// Subscribe implements EventBus.
func (NopEvents) Subscribe(string) (<-chan Event, func()) { return make(chan Event), func() {} }

// Notifier delivers push notifications to users of a stable; *push.Notifier
// implements it. kind must be one of the push.Kind* constants.
type Notifier interface {
	NotifyUsers(ctx context.Context, stableID string, userIDs []string, kind, title, body string, data map[string]any) error
}

// NopNotifier discards all notifications.
type NopNotifier struct{}

// NotifyUsers implements Notifier.
func (NopNotifier) NotifyUsers(context.Context, string, []string, string, string, string, map[string]any) error {
	return nil
}

// ErrorBody is the JSON error envelope: {"error":{"code":"...","message":"..."}}.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is the payload of ErrorBody. Code is a stable machine-readable
// identifier (snake_case), Message a human-readable description.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes v as JSON with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes the standard error envelope.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// ReadJSON decodes the request body into v. It limits the body to
// MaxBodyBytes, rejects unknown fields and trailing data. On failure it
// writes a 400 (or 413) error response itself and returns false; the caller
// must then return immediately.
func ReadJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()
	err := dec.Decode(v)
	if err == nil {
		// A second value (or garbage) after the first is an error; io.EOF is expected.
		if _, err2 := dec.Token(); !errors.Is(err2, io.EOF) {
			err = errors.New("unexpected data after JSON body")
		}
	}
	if err == nil {
		return true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		WriteError(w, http.StatusRequestEntityTooLarge, "body_too_large", fmt.Sprintf("request body exceeds %d bytes", tooLarge.Limit))
		return false
	}
	WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body: "+err.Error())
	return false
}
