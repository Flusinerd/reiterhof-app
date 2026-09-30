package devices_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func webSubscriptionJSON(t *testing.T, endpoint string) string {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatal(err)
	}
	enc := base64.RawURLEncoding.EncodeToString
	return fmt.Sprintf(`{"endpoint":%q,"expirationTime":null,"keys":{"p256dh":%q,"auth":%q}}`,
		endpoint, enc(priv.PublicKey().Bytes()), enc(auth))
}

func TestWebPushNotConfigured(t *testing.T) {
	t.Setenv(push.EnvVAPIDPublicKey, "")
	t.Setenv(push.EnvVAPIDPrivateKey, "")
	pool := dbtest.NewSeeded(t)
	h := httpapi.NewHandler(httpapi.Deps{Pool: pool})
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/push/web/public-key", ""},
		{http.MethodPost, "/api/v1/me/web-push-subscriptions", webSubscriptionJSON(t, "https://push.example.com/a")},
	} {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		authtest.Authorize(t, pool, req, seed.UserJan)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"not_configured"`) {
			t.Errorf("%s %s: %d %s, want 503 not_configured", c.method, c.path, rec.Code, rec.Body)
		}
	}
}

func TestWebPushSubscriptionEndpoints(t *testing.T) {
	pub, priv, err := push.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(push.EnvVAPIDPublicKey, pub)
	t.Setenv(push.EnvVAPIDPrivateKey, priv)
	t.Setenv(push.EnvVAPIDSubject, "mailto:admin@example.org")
	pool := dbtest.NewSeeded(t)
	h := httpapi.NewHandler(httpapi.Deps{Pool: pool})
	do := func(method, path, body string, authorize bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("User-Agent", "TestSafari/1")
		if authorize {
			authtest.Authorize(t, pool, req, seed.UserJan)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	count := func() int {
		var n int
		if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM web_push_subscriptions WHERE user_id = $1`, seed.UserJan).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	const path = "/api/v1/me/web-push-subscriptions"

	if rec := do(http.MethodGet, "/api/v1/push/web/public-key", "", false); rec.Code != http.StatusUnauthorized {
		t.Errorf("public key without auth: %d, want 401", rec.Code)
	}
	rec := do(http.MethodGet, "/api/v1/push/web/public-key", "", true)
	var key struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &key); err != nil || rec.Code != 200 || key.PublicKey != pub {
		t.Fatalf("public key: %d %s", rec.Code, rec.Body)
	}

	good := webSubscriptionJSON(t, "https://web.push.apple.com/QAbc")
	if rec := do(http.MethodPost, path, good, false); rec.Code != http.StatusUnauthorized {
		t.Errorf("register without auth: %d, want 401", rec.Code)
	}
	for name, body := range map[string]string{
		"http endpoint": webSubscriptionJSON(t, "http://web.push.apple.com/x"),
		"no keys":       `{"endpoint":"https://web.push.apple.com/x"}`,
		"short auth":    `{"endpoint":"https://web.push.apple.com/x","keys":{"p256dh":"BAAA","auth":"AAAA"}}`,
		"unknown field": `{"endpoint":"https://web.push.apple.com/x","keys":{},"extra":1}`,
	} {
		if rec := do(http.MethodPost, path, body, true); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", name, rec.Code)
		}
	}
	for range 2 { // idempotent
		if rec := do(http.MethodPost, path, good, true); rec.Code != http.StatusNoContent {
			t.Fatalf("register: %d %s", rec.Code, rec.Body)
		}
	}
	if n := count(); n != 1 {
		t.Fatalf("subscriptions = %d, want 1", n)
	}
	var ua string
	_ = pool.QueryRow(context.Background(), `SELECT user_agent FROM web_push_subscriptions`).Scan(&ua)
	if ua != "TestSafari/1" {
		t.Errorf("user agent = %q", ua)
	}
	if rec := do(http.MethodDelete, path, good, true); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if n := count(); n != 0 {
		t.Fatalf("subscriptions after delete = %d", n)
	}
}
