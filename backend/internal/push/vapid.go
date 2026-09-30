package push

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// Environment variables of the VAPID identity (RFC 8292). Without keys, web push is off.
const (
	EnvVAPIDPublicKey  = "REITERHOF_VAPID_PUBLIC_KEY"
	EnvVAPIDPrivateKey = "REITERHOF_VAPID_PRIVATE_KEY"
	EnvVAPIDSubject    = "REITERHOF_VAPID_SUBJECT"
)

// VAPID is the application server identity: a P-256 key pair and a contact subject
// (a mailto: or https: URI the push services can use to reach the operator).
type VAPID struct {
	key     *ecdsa.PrivateKey
	public  string // base64url of the 65 byte uncompressed point
	subject string
}

// ErrVAPIDNotConfigured is returned by VAPIDFromEnv when no key is set.
var ErrVAPIDNotConfigured = errors.New("push: VAPID keys not configured")

// b64 is the unpadded base64url encoding used by Web Push and JWT.
var b64 = base64.RawURLEncoding

// decodeB64 decodes base64url with or without padding.
func decodeB64(s string) ([]byte, error) {
	return b64.DecodeString(strings.TrimRight(s, "="))
}

// GenerateVAPIDKeys returns a fresh key pair as base64url strings (public: 65 byte
// uncompressed point, private: 32 byte scalar).
func GenerateVAPIDKeys() (public, private string, err error) {
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return b64.EncodeToString(k.PublicKey().Bytes()), b64.EncodeToString(k.Bytes()), nil
}

// NewVAPID builds the identity from base64url keys. The public key must belong to the
// private key (a mismatch means a copy-paste error that would make every push fail).
func NewVAPID(publicKey, privateKey, subject string) (*VAPID, error) {
	if !strings.HasPrefix(subject, "mailto:") && !strings.HasPrefix(subject, "https://") {
		return nil, fmt.Errorf("push: %s must start with mailto: or https://", EnvVAPIDSubject)
	}
	d, err := decodeB64(privateKey)
	if err != nil || len(d) != 32 {
		return nil, errors.New("push: VAPID private key must be 32 bytes, base64url")
	}
	key, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), d)
	if err != nil {
		return nil, fmt.Errorf("push: invalid VAPID private key: %w", err)
	}
	ecdhKey, err := key.ECDH()
	if err != nil {
		return nil, err
	}
	derived := ecdhKey.PublicKey().Bytes()
	given, err := decodeB64(publicKey)
	if err != nil || subtle.ConstantTimeCompare(given, derived) != 1 {
		return nil, errors.New("push: VAPID public key does not match the private key")
	}
	return &VAPID{key: key, public: b64.EncodeToString(derived), subject: subject}, nil
}

// VAPIDFromEnv reads the identity from the environment. It returns
// ErrVAPIDNotConfigured when neither key is set, and another error for a broken setup.
func VAPIDFromEnv() (*VAPID, error) {
	pub, priv := os.Getenv(EnvVAPIDPublicKey), os.Getenv(EnvVAPIDPrivateKey)
	if pub == "" && priv == "" {
		return nil, ErrVAPIDNotConfigured
	}
	if pub == "" || priv == "" {
		return nil, fmt.Errorf("push: set both %s and %s", EnvVAPIDPublicKey, EnvVAPIDPrivateKey)
	}
	return NewVAPID(pub, priv, os.Getenv(EnvVAPIDSubject))
}

// PublicKey is the applicationServerKey for PushManager.subscribe (base64url).
func (v *VAPID) PublicKey() string { return v.public }

// authorization returns the Authorization header value for a push service endpoint:
// `vapid t=<jwt>, k=<public key>` (RFC 8292).
func (v *VAPID) authorization(endpoint string, now time.Time) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("push: invalid endpoint %q", endpoint)
	}
	claims, err := json.Marshal(map[string]any{
		"aud": u.Scheme + "://" + u.Host,
		"exp": now.Add(12 * time.Hour).Unix(), // the maximum is 24 h
		"sub": v.subject,
	})
	if err != nil {
		return "", err
	}
	signingInput := b64.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`)) + "." + b64.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, v.key, digest[:])
	if err != nil {
		return "", err
	}
	sig := make([]byte, 64) // JWS ES256: fixed-width r || s
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return "vapid t=" + signingInput + "." + b64.EncodeToString(sig) + ", k=" + v.public, nil
}
