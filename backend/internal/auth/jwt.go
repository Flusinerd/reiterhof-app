package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Well-known OIDC endpoints.
const (
	GoogleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"
	AppleJWKSURL  = "https://appleid.apple.com/auth/keys"
)

// Claims are the ID token claims this service uses.
type Claims struct {
	Subject       string
	Email         string // lower-cased
	EmailVerified bool
	Name          string
}

// ErrInvalidToken is returned for every ID token that fails verification.
var ErrInvalidToken = errors.New("invalid id token")

// Verifier verifies RS256 ID tokens against a provider's JWKS.
// It uses the standard library only; JWKS responses are cached and refreshed when a
// token names an unknown key id (at most once a minute).
type Verifier struct {
	JWKSURL   string
	Issuers   []string // accepted "iss" values
	Audiences []string // accepted "aud" values (client IDs)
	HTTP      *http.Client
	Now       func() time.Time // injectable clock, default time.Now

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

const (
	jwksMaxAge     = time.Hour
	jwksMinRefresh = time.Minute
	clockLeeway    = time.Minute
)

// NewGoogleVerifier accepts Google ID tokens issued for one of the client IDs.
func NewGoogleVerifier(clientIDs []string) *Verifier {
	return &Verifier{JWKSURL: GoogleJWKSURL, Audiences: clientIDs,
		Issuers: []string{"https://accounts.google.com", "accounts.google.com"}}
}

// NewAppleVerifier accepts Apple ID tokens issued for one of the client IDs (bundle ID).
func NewAppleVerifier(clientIDs []string) *Verifier {
	return &Verifier{JWKSURL: AppleJWKSURL, Audiences: clientIDs, Issuers: []string{"https://appleid.apple.com"}}
}

// Verify checks signature (RS256), iss, aud, exp and returns the claims.
func (v *Verifier) Verify(ctx context.Context, idToken string) (Claims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return Claims{}, fmt.Errorf("%w: malformed", ErrInvalidToken)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := decodeSegment(parts[0], &header); err != nil {
		return Claims{}, fmt.Errorf("%w: header: %v", ErrInvalidToken, err)
	}
	// Only RS256 is accepted (never "none" or HMAC variants).
	if header.Alg != "RS256" {
		return Claims{}, fmt.Errorf("%w: unsupported alg %q", ErrInvalidToken, header.Alg)
	}
	key, err := v.key(ctx, header.Kid)
	if err != nil {
		return Claims{}, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, fmt.Errorf("%w: signature encoding", ErrInvalidToken)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig); err != nil {
		return Claims{}, fmt.Errorf("%w: bad signature", ErrInvalidToken)
	}

	var p struct {
		Iss           string          `json:"iss"`
		Aud           json.RawMessage `json:"aud"`
		Exp           float64         `json:"exp"`
		Sub           string          `json:"sub"`
		Email         string          `json:"email"`
		EmailVerified json.RawMessage `json:"email_verified"` // bool (Google) or "true" string (Apple)
		Name          string          `json:"name"`
	}
	if err := decodeSegment(parts[1], &p); err != nil {
		return Claims{}, fmt.Errorf("%w: payload: %v", ErrInvalidToken, err)
	}
	if !contains(v.Issuers, p.Iss) {
		return Claims{}, fmt.Errorf("%w: issuer %q", ErrInvalidToken, p.Iss)
	}
	if !audienceMatches(p.Aud, v.Audiences) {
		return Claims{}, fmt.Errorf("%w: audience", ErrInvalidToken)
	}
	now := time.Now
	if v.Now != nil {
		now = v.Now
	}
	if p.Exp == 0 || !now().Before(time.Unix(int64(p.Exp), 0).Add(clockLeeway)) {
		return Claims{}, fmt.Errorf("%w: expired", ErrInvalidToken)
	}
	if p.Sub == "" {
		return Claims{}, fmt.Errorf("%w: no subject", ErrInvalidToken)
	}
	verified := string(p.EmailVerified) == "true" || string(p.EmailVerified) == `"true"`
	return Claims{Subject: p.Sub, Email: strings.ToLower(strings.TrimSpace(p.Email)), EmailVerified: verified, Name: p.Name}, nil
}

func decodeSegment(seg string, v any) error {
	raw, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// audienceMatches handles "aud" as a string or an array. An empty allow-list matches nothing.
func audienceMatches(raw json.RawMessage, allowed []string) bool {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return contains(allowed, one)
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		for _, a := range many {
			if contains(allowed, a) {
				return true
			}
		}
	}
	return false
}

func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	now := time.Now
	if v.Now != nil {
		now = v.Now
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if k, ok := v.keys[kid]; ok && now().Sub(v.fetchedAt) < jwksMaxAge {
		return k, nil
	}
	if v.keys == nil || now().Sub(v.fetchedAt) >= jwksMinRefresh {
		keys, err := v.fetch(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch jwks: %w", err)
		}
		v.keys, v.fetchedAt = keys, now()
	}
	if k, ok := v.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("%w: unknown key id", ErrInvalidToken)
}

func (v *Verifier) fetch(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	client := v.HTTP
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.JWKSURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&set); err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(k.N)
		e, err2 := base64.RawURLEncoding.DecodeString(k.E)
		if err1 != nil || err2 != nil || len(e) == 0 || len(e) > 4 {
			continue
		}
		exp := 0
		for _, b := range e {
			exp = exp<<8 | int(b)
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exp}
	}
	return keys, nil
}
