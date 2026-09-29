package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
)

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(Deps{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != `{"status":"ok"}` {
		t.Errorf("body = %q", body)
	}
}

func TestUnknownRouteIs404(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(Deps{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestReadyzWithoutDatabase(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(Deps{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"code":"database_unavailable"`) {
		t.Errorf("body = %q", body)
	}
}

func TestReadyzWithDatabase(t *testing.T) {
	pool := dbtest.New(t)
	rec := httptest.NewRecorder()
	NewHandler(Deps{Pool: pool}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
