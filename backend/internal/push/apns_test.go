package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// testAPNSKey returns a fresh P-256 key as the .p8 PEM Apple hands out, plus the key.
func testAPNSKey(t *testing.T) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), key
}

// verifyES256 checks a compact JWT against the public key and returns header and claims.
func verifyES256(t *testing.T, tok string, pub *ecdsa.PublicKey) (header, claims map[string]any) {
	t.Helper()
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("jwt has %d parts", len(parts))
	}
	sig, err := b64.DecodeString(parts[2])
	if err != nil || len(sig) != 64 {
		t.Fatalf("signature: %v, %d bytes", err, len(sig))
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(pub, digest[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		t.Fatal("jwt signature does not verify")
	}
	for i, dst := range []*map[string]any{&header, &claims} {
		raw, err := b64.DecodeString(parts[i])
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatal(err)
		}
	}
	return header, claims
}

type apnsCall struct {
	token, auth, priority, expiration string
	payload                           map[string]any
}

// fakeAPNS records requests and answers by device token: "…bad…" is BadDeviceToken,
// "…gone…" is 410 Unregistered, the first request of "…expired…" is ExpiredProviderToken.
type fakeAPNS struct {
	mu    sync.Mutex
	calls []apnsCall
	proto int
	topic string
}

func (f *fakeAPNS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/3/device/")
	var payload map[string]any
	_ = json.NewDecoder(r.Body).Decode(&payload)
	f.mu.Lock()
	f.proto = r.ProtoMajor
	f.topic = r.Header.Get("apns-topic")
	f.calls = append(f.calls, apnsCall{
		token: token, auth: r.Header.Get("Authorization"), priority: r.Header.Get("apns-priority"),
		expiration: r.Header.Get("apns-expiration"), payload: payload,
	})
	seen := 0
	for _, c := range f.calls {
		if c.token == token {
			seen++
		}
	}
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case strings.Contains(token, "bad"):
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"reason":"BadDeviceToken"}`))
	case strings.Contains(token, "9095"):
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`{"reason":"Unregistered"}`))
	case strings.Contains(token, "e0f1") && seen == 1:
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"reason":"ExpiredProviderToken"}`))
	case strings.Contains(token, "5e5e"):
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"reason":"Shutdown"}`))
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func newAPNSTest(t *testing.T) (*APNSClient, *fakeAPNS, *ecdsa.PrivateKey) {
	t.Helper()
	pemKey, key := testAPNSKey(t)
	f := &fakeAPNS{}
	srv := httptest.NewUnstartedServer(f)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)
	c, err := NewAPNSClient(pemKey, "KEY1234567", "TEAM123456", "de.example.app")
	if err != nil {
		t.Fatal(err)
	}
	c.BaseURL = srv.URL
	c.HTTPClient = srv.Client()
	return c, f, key
}

const (
	tokOK      = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	tokBad     = "bad0bad0bad0bad0bad0bad0bad0bad0bad0bad0bad0bad0bad0bad0bad0bad0"
	tokGone    = "9095909590959095909590959095909590959095909590959095909590959095"
	tokExpired = "e0f1e0f1e0f1e0f1e0f1e0f1e0f1e0f1e0f1e0f1e0f1e0f1e0f1e0f1e0f1e0f1"
	tokDown    = "5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e"
)

func TestAPNSSendsHTTP2WithProviderTokenAndPayload(t *testing.T) {
	c, f, key := newAPNSTest(t)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	c.Now = func() time.Time { return now }

	err := c.Send(context.Background(), []Message{{
		To: tokOK, Platform: PlatformIOS, Title: "Hallo", Body: "Balu braucht dich", Sound: "default",
		Data: map[string]any{"kind": KindLastPerson, "screen": "/blankets"}, Priority: PriorityHigh, TTL: 4 * time.Hour,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if f.proto != 2 {
		t.Errorf("HTTP/%d, want HTTP/2", f.proto)
	}
	if f.topic != "de.example.app" {
		t.Errorf("apns-topic = %q", f.topic)
	}
	if len(f.calls) != 1 {
		t.Fatalf("calls = %d", len(f.calls))
	}
	call := f.calls[0]
	if call.token != tokOK || call.priority != "10" || call.expiration != "1790784000" {
		t.Errorf("call = %+v", call)
	}
	if !strings.HasPrefix(call.auth, "bearer ") {
		t.Fatalf("Authorization = %q", call.auth)
	}
	header, claims := verifyES256(t, strings.TrimPrefix(call.auth, "bearer "), &key.PublicKey)
	if header["alg"] != "ES256" || header["kid"] != "KEY1234567" || claims["iss"] != "TEAM123456" || claims["iat"] != float64(now.Unix()) {
		t.Errorf("header = %v, claims = %v", header, claims)
	}
	aps := call.payload["aps"].(map[string]any)
	alert := aps["alert"].(map[string]any)
	if alert["title"] != "Hallo" || alert["body"] != "Balu braucht dich" || aps["sound"] != "default" {
		t.Errorf("aps = %v", aps)
	}
	// expo-notifications on iOS exposes the top-level "body" object as the notification data.
	data := call.payload["body"].(map[string]any)
	if data["kind"] != KindLastPerson || data["screen"] != "/blankets" {
		t.Errorf("data = %v", data)
	}
}

func TestAPNSNormalPriorityAndNoExpiration(t *testing.T) {
	c, f, _ := newAPNSTest(t)
	if err := c.Send(context.Background(), []Message{{To: tokOK, Platform: PlatformIOS, Body: "x"}}); err != nil {
		t.Fatal(err)
	}
	if f.calls[0].priority != "5" || f.calls[0].expiration != "" {
		t.Errorf("call = %+v", f.calls[0])
	}
}

func TestAPNSReportsInvalidTokensAndFailures(t *testing.T) {
	c, f, _ := newAPNSTest(t)
	err := c.Send(context.Background(), []Message{
		{To: tokOK, Platform: PlatformIOS, Body: "a"},
		{To: tokBad, Platform: PlatformIOS, Body: "b"},
		{To: tokGone, Platform: PlatformIOS, Body: "c"},
		{To: "ExponentPushToken[x]", Platform: PlatformIOS, Body: "d"},
		{To: tokDown, Platform: PlatformIOS, Body: "e"},
	})
	var serr *SendError
	if !errors.As(err, &serr) {
		t.Fatalf("err = %v", err)
	}
	invalid := map[string]bool{}
	for _, tok := range serr.InvalidTokens {
		invalid[tok] = true
	}
	if len(invalid) != 3 || !invalid[tokBad] || !invalid[tokGone] || !invalid["ExponentPushToken[x]"] {
		t.Errorf("InvalidTokens = %v", serr.InvalidTokens)
	}
	if len(serr.Failures) != 1 || !strings.Contains(serr.Failures[0], "http 503 Shutdown") || strings.Contains(serr.Failures[0], tokDown) {
		t.Errorf("Failures = %v", serr.Failures)
	}
	if len(f.calls) != 4 { // the malformed token never reaches Apple
		t.Errorf("calls = %d, want 4", len(f.calls))
	}
}

func TestAPNSRefreshesExpiredProviderToken(t *testing.T) {
	c, f, _ := newAPNSTest(t)
	if err := c.Send(context.Background(), []Message{{To: tokExpired, Platform: PlatformIOS, Body: "x"}}); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 2 || f.calls[0].auth == f.calls[1].auth {
		t.Fatalf("calls = %+v, want a retry with a fresh token", f.calls)
	}
}

func TestAPNSCachesProviderTokenForFiftyMinutes(t *testing.T) {
	c, f, _ := newAPNSTest(t)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	c.Now = func() time.Time { return now }
	send := func() {
		t.Helper()
		if err := c.Send(context.Background(), []Message{{To: tokOK, Platform: PlatformIOS, Body: "x"}}); err != nil {
			t.Fatal(err)
		}
	}
	send()
	now = now.Add(30 * time.Minute)
	send()
	now = now.Add(30 * time.Minute)
	send()
	if f.calls[0].auth != f.calls[1].auth {
		t.Error("token not reused within 50 minutes")
	}
	if f.calls[1].auth == f.calls[2].auth {
		t.Error("token not refreshed after 50 minutes")
	}
}

func TestNewAPNSClientRejectsBadInput(t *testing.T) {
	pemKey, _ := testAPNSKey(t)
	if _, err := NewAPNSClient(pemKey, "", "TEAM", "topic"); err == nil {
		t.Error("want error for missing key ID")
	}
	if _, err := NewAPNSClient([]byte("not pem"), "K", "T", "topic"); err == nil {
		t.Error("want error for non-PEM key")
	}
	if _, err := NewAPNSClient(pemKey, "K", "T", "topic"); err != nil {
		t.Error(err)
	}
}
