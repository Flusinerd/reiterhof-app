package push

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	testRSAOnce sync.Once
	testRSAKey  *rsa.PrivateKey
)

// testServiceAccount returns a service account JSON like the Firebase console exports,
// with a key generated once per test binary (RSA key generation is slow).
func testServiceAccount(t *testing.T, tokenURI string) ([]byte, *rsa.PrivateKey) {
	t.Helper()
	testRSAOnce.Do(func() {
		var err error
		testRSAKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
	})
	der, err := x509.MarshalPKCS8PrivateKey(testRSAKey)
	if err != nil {
		t.Fatal(err)
	}
	sa, _ := json.Marshal(map[string]string{
		"type":         "service_account",
		"project_id":   "stallfunk-test",
		"private_key":  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		"client_email": "fcm@stallfunk-test.iam.gserviceaccount.com",
		"token_uri":    tokenURI,
	})
	return sa, testRSAKey
}

type fcmCall struct {
	auth    string
	message map[string]any
}

// fakeFCM serves the token endpoint (/token) and the send endpoint (/send). Tokens
// containing "gone" are UNREGISTERED, "boom" is a 500, and the first send with the
// first access token is a 401 when expireFirst is set.
type fakeFCM struct {
	mu          sync.Mutex
	tokenCalls  []string // assertions
	sends       []fcmCall
	expireFirst bool
	pub         *rsa.PublicKey
	t           *testing.T
}

func (f *fakeFCM) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.URL.Path {
	case "/token":
		_ = r.ParseForm()
		if r.PostForm.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			http.Error(w, "grant_type", http.StatusBadRequest)
			return
		}
		assertion := r.PostForm.Get("assertion")
		f.tokenCalls = append(f.tokenCalls, assertion)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at-` + string(rune('0'+len(f.tokenCalls))) + `","expires_in":3599,"token_type":"Bearer"}`))
	case "/send":
		var body struct {
			Message map[string]any `json:"message"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		call := fcmCall{auth: r.Header.Get("Authorization"), message: body.Message}
		f.sends = append(f.sends, call)
		w.Header().Set("Content-Type", "application/json")
		token, _ := body.Message["token"].(string)
		switch {
		case f.expireFirst && call.auth == "Bearer at-1":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":401,"status":"UNAUTHENTICATED","message":"Request had invalid authentication credentials."}}`))
		case strings.Contains(token, "gone"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":404,"message":"Requested entity was not found.","status":"NOT_FOUND","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`))
		case strings.Contains(token, "malformed"):
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"code":400,"message":"The registration token is not a valid FCM registration token","status":"INVALID_ARGUMENT","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"INVALID_ARGUMENT"}]}}`))
		case strings.Contains(token, "boom"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"code":500,"message":"Internal error encountered.","status":"INTERNAL"}}`))
		default:
			_, _ = w.Write([]byte(`{"name":"projects/stallfunk-test/messages/0:1"}`))
		}
	default:
		http.NotFound(w, r)
	}
}

func newFCMTest(t *testing.T) (*FCMClient, *fakeFCM) {
	t.Helper()
	f := &fakeFCM{t: t}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	sa, key := testServiceAccount(t, srv.URL+"/token")
	f.pub = &key.PublicKey
	c, err := NewFCMClient(sa)
	if err != nil {
		t.Fatal(err)
	}
	c.SendURL = srv.URL + "/send"
	c.HTTPClient = srv.Client()
	return c, f
}

func TestFCMFetchesAccessTokenAndSendsDataMessage(t *testing.T) {
	c, f := newFCMTest(t)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	c.Now = func() time.Time { return now }
	if c.ProjectID() != "stallfunk-test" {
		t.Errorf("ProjectID = %q", c.ProjectID())
	}

	err := c.Send(context.Background(), []Message{{
		To: "dLx-3:APA91b_Q", Platform: PlatformAndroid, Title: "Hallo", Body: "Balu braucht dich",
		Data: map[string]any{"kind": KindLastPerson, "screen": "/blankets"}, Priority: PriorityHigh, TTL: 4 * time.Hour,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.tokenCalls) != 1 {
		t.Fatalf("token calls = %d", len(f.tokenCalls))
	}
	// The assertion is an RS256 JWT for the service account and the messaging scope.
	parts := strings.Split(f.tokenCalls[0], ".")
	if len(parts) != 3 {
		t.Fatalf("assertion has %d parts", len(parts))
	}
	sig, err := b64.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(f.pub, crypto.SHA256, digest[:], sig); err != nil {
		t.Fatalf("assertion signature: %v", err)
	}
	var claims map[string]any
	rawClaims, _ := b64.DecodeString(parts[1])
	_ = json.Unmarshal(rawClaims, &claims)
	if claims["iss"] != "fcm@stallfunk-test.iam.gserviceaccount.com" || claims["scope"] != fcmScope ||
		claims["aud"] != c.tokenURI || claims["exp"] != float64(now.Add(time.Hour).Unix()) {
		t.Errorf("claims = %v", claims)
	}

	if len(f.sends) != 1 {
		t.Fatalf("sends = %d", len(f.sends))
	}
	send := f.sends[0]
	if send.auth != "Bearer at-1" || send.message["token"] != "dLx-3:APA91b_Q" {
		t.Errorf("send = %+v", send)
	}
	if _, hasNotification := send.message["notification"]; hasNotification {
		t.Error("message must be data-only so expo-notifications builds the notification")
	}
	data := send.message["data"].(map[string]any)
	if data["title"] != "Hallo" || data["message"] != "Balu braucht dich" || data["channelId"] != "default" {
		t.Errorf("data = %v", data)
	}
	var inner map[string]any
	if err := json.Unmarshal([]byte(data["body"].(string)), &inner); err != nil || inner["kind"] != KindLastPerson || inner["screen"] != "/blankets" {
		t.Errorf("body = %v (%v)", data["body"], err)
	}
	android := send.message["android"].(map[string]any)
	if android["priority"] != "high" || android["ttl"] != "14400s" {
		t.Errorf("android = %v", android)
	}
}

func TestFCMReusesAccessTokenAndRetriesAfter401(t *testing.T) {
	c, f := newFCMTest(t)
	f.expireFirst = true
	err := c.Send(context.Background(), []Message{
		{To: "a1", Platform: PlatformAndroid, Body: "x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.tokenCalls) != 2 || len(f.sends) != 2 || f.sends[1].auth != "Bearer at-2" {
		t.Fatalf("token calls = %d, sends = %+v", len(f.tokenCalls), f.sends)
	}
	// The second token is cached for the next send.
	if err := c.Send(context.Background(), []Message{{To: "a2", Platform: PlatformAndroid, Body: "y"}}); err != nil {
		t.Fatal(err)
	}
	if len(f.tokenCalls) != 2 || f.sends[2].auth != "Bearer at-2" {
		t.Errorf("token calls = %d, sends = %+v", len(f.tokenCalls), f.sends)
	}
}

func TestFCMReportsInvalidTokensAndFailures(t *testing.T) {
	c, f := newFCMTest(t)
	err := c.Send(context.Background(), []Message{
		{To: "ok1", Platform: PlatformAndroid, Body: "a"},
		{To: "gone1", Platform: PlatformAndroid, Body: "b"},
		{To: "malformed1", Platform: PlatformAndroid, Body: "c"},
		{To: "boom1", Platform: PlatformAndroid, Body: "d"},
	})
	var serr *SendError
	if !errors.As(err, &serr) {
		t.Fatalf("err = %v", err)
	}
	invalid := map[string]bool{}
	for _, tok := range serr.InvalidTokens {
		invalid[tok] = true
	}
	if len(invalid) != 2 || !invalid["gone1"] || !invalid["malformed1"] {
		t.Errorf("InvalidTokens = %v", serr.InvalidTokens)
	}
	if len(serr.Failures) != 1 || !strings.Contains(serr.Failures[0], "http 500 INTERNAL") || strings.Contains(serr.Failures[0], "boom1") {
		t.Errorf("Failures = %v", serr.Failures)
	}
	if len(f.sends) != 4 {
		t.Errorf("sends = %d", len(f.sends))
	}
}

func TestFCMNormalPriorityWithoutTTL(t *testing.T) {
	c, f := newFCMTest(t)
	if err := c.Send(context.Background(), []Message{{To: "ok", Platform: PlatformAndroid, Body: "x"}}); err != nil {
		t.Fatal(err)
	}
	android := f.sends[0].message["android"].(map[string]any)
	if android["priority"] != "normal" || android["ttl"] != nil {
		t.Errorf("android = %v", android)
	}
}

func TestNewFCMClientRejectsBadInput(t *testing.T) {
	if _, err := NewFCMClient([]byte(`{"type":"user"}`)); err == nil {
		t.Error("want error for a non service account")
	}
	if _, err := NewFCMClient([]byte(`nope`)); err == nil {
		t.Error("want error for invalid JSON")
	}
	sa, _ := testServiceAccount(t, "")
	c, err := NewFCMClient(sa)
	if err != nil {
		t.Fatal(err)
	}
	if c.tokenURI != "https://oauth2.googleapis.com/token" {
		t.Errorf("tokenURI = %q, want the Google default", c.tokenURI)
	}
}
