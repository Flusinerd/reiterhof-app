// Package httpx holds the dependencies and helpers shared by all HTTP handlers.
//
// Domain packages import httpx (never httpapi, which imports them) to avoid
// import cycles. httpapi re-exports Deps as httpapi.Deps.
package httpx

import (
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
