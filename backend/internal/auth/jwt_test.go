package auth_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
)

// testIdP is a fake identity provider: an RSA key, a JWKS endpoint and a token signer.
type testIdP struct {
	key      *rsa.PrivateKey
	srv      *httptest.Server
	fetches  atomic.Int32
	issuer   string
	audience string
	now      time.Time
}

func newTestIdP(t testing.TB, issuer, audience string, now time.Time) *testIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	p := &testIdP{key: key, issuer: issuer, audience: audience, now: now}
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		p.fetches.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(p.srv.Close)
	return p
}

func (p *testIdP) verifier() *auth.Verifier {
	return &auth.Verifier{JWKSURL: p.srv.URL, Issuers: []string{p.issuer}, Audiences: []string{p.audience}, Now: func() time.Time { return p.now }}
}

// sign builds a compact RS256 JWT with the payload merged over sensible defaults.
func (p *testIdP) sign(t testing.TB, key *rsa.PrivateKey, override map[string]any) string {
	t.Helper()
	payload := map[string]any{
		"iss": p.issuer, "aud": p.audience, "sub": "sub-1", "exp": p.now.Add(time.Hour).Unix(),
		"email": "Rider@Example.org", "email_verified": true,
	}
	for k, v := range override {
		if v == nil {
			delete(payload, k)
		} else {
			payload[k] = v
		}
	}
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	signing := enc(map[string]string{"alg": "RS256", "kid": "k1", "typ": "JWT"}) + "." + enc(payload)
	digest := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifier(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	idp := newTestIdP(t, "https://accounts.google.com", "client-1", now)
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		token   func() string
		wantErr bool
	}{
		{"valid", func() string { return idp.sign(t, idp.key, nil) }, false},
		{"valid with aud array", func() string { return idp.sign(t, idp.key, map[string]any{"aud": []string{"x", "client-1"}}) }, false},
		{"bad signature", func() string { return idp.sign(t, other, nil) }, true},
		{"wrong audience", func() string { return idp.sign(t, idp.key, map[string]any{"aud": "other-client"}) }, true},
		{"wrong issuer", func() string { return idp.sign(t, idp.key, map[string]any{"iss": "https://evil.example"}) }, true},
		{"expired", func() string { return idp.sign(t, idp.key, map[string]any{"exp": now.Add(-time.Hour).Unix()}) }, true},
		{"no exp", func() string { return idp.sign(t, idp.key, map[string]any{"exp": nil}) }, true},
		{"no subject", func() string { return idp.sign(t, idp.key, map[string]any{"sub": ""}) }, true},
		{"garbage", func() string { return "not-a-jwt" }, true},
		{"alg none", func() string {
			enc := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
			return enc(`{"alg":"none"}`) + "." + enc(`{"iss":"https://accounts.google.com","aud":"client-1","sub":"x","exp":9999999999}`) + "."
		}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := idp.verifier().Verify(context.Background(), tc.token())
			if tc.wantErr {
				if !errors.Is(err, auth.ErrInvalidToken) {
					t.Fatalf("want auth.ErrInvalidToken, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if c.Subject != "sub-1" || c.Email != "rider@example.org" || !c.EmailVerified {
				t.Fatalf("claims = %+v", c)
			}
		})
	}
}

func TestVerifierAppleStringEmailVerified(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	idp := newTestIdP(t, "https://appleid.apple.com", "org.example.app", now)
	c, err := idp.verifier().Verify(context.Background(), idp.sign(t, idp.key, map[string]any{"email_verified": "true"}))
	if err != nil || !c.EmailVerified {
		t.Fatalf("claims = %+v, err = %v", c, err)
	}
}

func TestVerifierCachesJWKS(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	idp := newTestIdP(t, "https://accounts.google.com", "client-1", now)
	v := idp.verifier()
	for range 3 {
		if _, err := v.Verify(context.Background(), idp.sign(t, idp.key, nil)); err != nil {
			t.Fatal(err)
		}
	}
	if n := idp.fetches.Load(); n != 1 {
		t.Fatalf("JWKS fetched %d times, want 1", n)
	}
	// After the cache expired it is refreshed.
	idp.now = now.Add(2 * time.Hour)
	if _, err := v.Verify(context.Background(), idp.sign(t, idp.key, map[string]any{"exp": idp.now.Add(time.Hour).Unix()})); err != nil {
		t.Fatal(err)
	}
	if n := idp.fetches.Load(); n != 2 {
		t.Fatalf("JWKS fetched %d times, want 2", n)
	}
}

func TestVerifierNoAudiencesRejectsEverything(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	idp := newTestIdP(t, "https://accounts.google.com", "client-1", now)
	v := idp.verifier()
	v.Audiences = nil
	if _, err := v.Verify(context.Background(), idp.sign(t, idp.key, nil)); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("got %v", err)
	}
}
