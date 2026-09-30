package devices_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestRegisterAndDeletePushToken(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	h := httpapi.NewHandler(httpapi.Deps{Pool: pool})
	do := func(method, body string, authorize bool) int {
		req := httptest.NewRequest(method, "/api/v1/me/push-tokens", strings.NewReader(body))
		if authorize {
			authtest.Authorize(t, pool, req, seed.UserJan)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	count := func() int {
		var n int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM push_tokens WHERE user_id = $1`, seed.UserJan).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if got := do(http.MethodPost, `{"token":"ExponentPushToken[a]","platform":"ios"}`, false); got != http.StatusUnauthorized {
		t.Fatalf("without token: status = %d, want 401", got)
	}
	if got := do(http.MethodPost, `{"token":"ExponentPushToken[a]","platform":"web"}`, true); got != http.StatusBadRequest {
		t.Fatalf("bad platform: status = %d, want 400", got)
	}
	for range 2 { // registering twice is idempotent
		if got := do(http.MethodPost, `{"token":"ExponentPushToken[a]","platform":"ios"}`, true); got != http.StatusNoContent {
			t.Fatalf("register: status = %d, want 204", got)
		}
	}
	if n := count(); n != 1 {
		t.Fatalf("tokens = %d, want 1", n)
	}
	if got := do(http.MethodDelete, `{"token":"ExponentPushToken[a]"}`, true); got != http.StatusNoContent {
		t.Fatalf("delete: status = %d, want 204", got)
	}
	if n := count(); n != 0 {
		t.Fatalf("tokens after delete = %d, want 0", n)
	}
}
