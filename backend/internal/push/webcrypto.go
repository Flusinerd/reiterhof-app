package push

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

// Message encryption for Web Push, RFC 8291 with the aes128gcm content coding of RFC 8188.
// Standard library only.

const (
	// webRecordSize is the record size announced in the header. A push message is
	// always a single record, so it is also the upper bound of the encrypted body.
	webRecordSize = 4096
	// webHeaderLen is salt (16) + record size (4) + key id length (1) + sender key (65).
	webHeaderLen = 16 + 4 + 1 + 65
	// MaxWebPayload is the largest plaintext one message may carry: the record minus
	// header, the padding delimiter (1) and the GCM tag (16). Push services accept 4096
	// bytes of body, so this is the limit for every message.
	MaxWebPayload = webRecordSize - webHeaderLen - 1 - 16
)

// ErrPayloadTooLarge is returned when the plaintext exceeds MaxWebPayload.
var ErrPayloadTooLarge = errors.New("push: web push payload too large")

// encryptWebPush encrypts plaintext for the subscription keys (uaPublic: the 65 byte
// uncompressed P-256 point "p256dh", authSecret: the 16 byte "auth") and returns the
// complete aes128gcm request body. asPriv is the sender's ephemeral key and salt the
// 16 random bytes; both are parameters so tests can use the vector of RFC 8291 appendix A.
func encryptWebPush(uaPublic, authSecret, plaintext []byte, asPriv *ecdh.PrivateKey, salt []byte) ([]byte, error) {
	if len(plaintext) > MaxWebPayload {
		return nil, ErrPayloadTooLarge
	}
	if len(authSecret) != 16 || len(salt) != 16 {
		return nil, errors.New("push: auth secret and salt must be 16 bytes")
	}
	uaKey, err := ecdh.P256().NewPublicKey(uaPublic)
	if err != nil {
		return nil, fmt.Errorf("push: invalid p256dh key: %w", err)
	}
	secret, err := asPriv.ECDH(uaKey)
	if err != nil {
		return nil, fmt.Errorf("push: ecdh: %w", err)
	}
	asPublic := asPriv.PublicKey().Bytes()

	// IKM = HKDF(salt = auth, ikm = ecdh, info = "WebPush: info" 0x00 ua_public as_public, 32)
	keyInfo := append(append([]byte("WebPush: info\x00"), uaPublic...), asPublic...)
	prkKey, err := hkdf.Extract(sha256.New, secret, authSecret)
	if err != nil {
		return nil, err
	}
	ikm, err := hkdf.Expand(sha256.New, prkKey, string(keyInfo), 32)
	if err != nil {
		return nil, err
	}
	// CEK and nonce from a second HKDF keyed with the random salt (RFC 8188).
	prk, err := hkdf.Extract(sha256.New, ikm, salt)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	body := make([]byte, 0, webHeaderLen+len(plaintext)+1+gcm.Overhead())
	body = append(body, salt...)
	body = binary.BigEndian.AppendUint32(body, webRecordSize)
	body = append(body, byte(len(asPublic)))
	body = append(body, asPublic...)
	record := append(append([]byte(nil), plaintext...), 0x02) // 0x02: last record delimiter, no padding
	return gcm.Seal(body, nonce, record, nil), nil
}

// newWebPushBody encrypts with a fresh ephemeral key and random salt.
func newWebPushBody(uaPublic, authSecret, plaintext []byte) ([]byte, error) {
	asPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return encryptWebPush(uaPublic, authSecret, plaintext, asPriv, salt)
}
