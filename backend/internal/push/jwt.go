package push

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// signJWT builds a compact JWS (header.claims.signature, base64url without padding).
// sign receives the SHA-256 digest of the signing input and returns the raw signature.
func signJWT(header, claims map[string]any, sign func(digest []byte) ([]byte, error)) (string, error) {
	h, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	c, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	input := b64.EncodeToString(h) + "." + b64.EncodeToString(c)
	digest := sha256.Sum256([]byte(input))
	sig, err := sign(digest[:])
	if err != nil {
		return "", err
	}
	return input + "." + b64.EncodeToString(sig), nil
}

// es256 signs with a P-256 key in the JWS format (r || s, 32 bytes each). Used by APNs
// (and the VAPID header; see vapid.go).
func es256(key *ecdsa.PrivateKey) func(digest []byte) ([]byte, error) {
	return func(digest []byte) ([]byte, error) {
		r, s, err := ecdsa.Sign(rand.Reader, key, digest)
		if err != nil {
			return nil, err
		}
		out := make([]byte, 64)
		r.FillBytes(out[:32])
		s.FillBytes(out[32:])
		return out, nil
	}
}

// rs256 signs with an RSA key (PKCS #1 v1.5, SHA-256). Used for Google's OAuth 2.0
// service account flow.
func rs256(key *rsa.PrivateKey) func(digest []byte) ([]byte, error) {
	return func(digest []byte) ([]byte, error) {
		sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest)
		if err != nil {
			return nil, fmt.Errorf("rs256: %w", err)
		}
		return sig, nil
	}
}
