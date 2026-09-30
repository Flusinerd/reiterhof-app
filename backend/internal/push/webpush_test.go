package push

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- test receiver (the browser side of RFC 8291) -----------------------------------------

// receiver plays the user agent: it owns the P-256 key and the auth secret of a subscription.
type receiver struct {
	priv *ecdh.PrivateKey
	auth []byte
}

func newReceiver(t testing.TB) receiver {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		t.Fatal(err)
	}
	return receiver{priv: priv, auth: auth}
}

func (r receiver) subscription(endpoint string) WebSubscription {
	return WebSubscription{
		Endpoint: endpoint,
		P256dh:   b64.EncodeToString(r.priv.PublicKey().Bytes()),
		Auth:     b64.EncodeToString(r.auth),
	}
}

// hkdf32 is an independent HKDF-SHA256 (RFC 5869) so the receiver does not share code with
// the implementation under test.
func hkdfSHA256(salt, ikm, info []byte, n int) []byte {
	ext := func(key, msg []byte) []byte {
		m := hmacSHA256(key, msg)
		return m
	}
	prk := ext(salt, ikm)
	var out, prev []byte
	for i := byte(1); len(out) < n; i++ {
		prev = hmacSHA256(prk, append(append(append([]byte(nil), prev...), info...), i))
		out = append(out, prev...)
	}
	return out[:n]
}

func hmacSHA256(key, msg []byte) []byte {
	const block = 64
	if len(key) > block {
		s := sha256.Sum256(key)
		key = s[:]
	}
	k := make([]byte, block)
	copy(k, key)
	ipad, opad := make([]byte, block), make([]byte, block)
	for i := range k {
		ipad[i], opad[i] = k[i]^0x36, k[i]^0x5c
	}
	inner := sha256.Sum256(append(ipad, msg...))
	outer := sha256.Sum256(append(opad, inner[:]...))
	return outer[:]
}

// decrypt reverses encryptWebPush for a single-record body.
func (r receiver) decrypt(t testing.TB, body []byte) []byte {
	t.Helper()
	if len(body) < webHeaderLen+17 {
		t.Fatalf("body too short: %d", len(body))
	}
	salt := body[:16]
	if rs := binary.BigEndian.Uint32(body[16:20]); rs != 4096 {
		t.Fatalf("record size %d", rs)
	}
	idlen := int(body[20])
	if idlen != 65 {
		t.Fatalf("keyid length %d", idlen)
	}
	asPublic := body[21 : 21+idlen]
	ciphertext := body[21+idlen:]
	asKey, err := ecdh.P256().NewPublicKey(asPublic)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := r.priv.ECDH(asKey)
	if err != nil {
		t.Fatal(err)
	}
	uaPublic := r.priv.PublicKey().Bytes()
	keyInfo := append(append([]byte("WebPush: info\x00"), uaPublic...), asPublic...)
	ikm := hkdfSHA256(r.auth, secret, keyInfo, 32)
	cek := hkdfSHA256(salt, ikm, []byte("Content-Encoding: aes128gcm\x00"), 16)
	nonce := hkdfSHA256(salt, ikm, []byte("Content-Encoding: nonce\x00"), 12)
	block, err := aes.NewCipher(cek)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	// Strip padding: zeros, then the 0x02 delimiter of the last record.
	i := len(plain) - 1
	for i >= 0 && plain[i] == 0 {
		i--
	}
	if i < 0 || plain[i] != 0x02 {
		t.Fatalf("missing padding delimiter")
	}
	return plain[:i]
}

// --- RFC 8291 -------------------------------------------------------------------------------

func mustB64(t testing.TB, s string) []byte {
	t.Helper()
	b, err := decodeB64(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The example of RFC 8291 appendix A.
const (
	rfcPlaintext = "When I grow up, I want to be a watermelon"
	rfcASPrivate = "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"
	rfcUAPrivate = "q1dXpw3UpT5VOmu_cf_v6ih07Aems3njxI-JWgLcM94"
	rfcUAPublic  = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	rfcAuth      = "BTBZMqHH6r4Tts7J_aSIgg"
	rfcSalt      = "DGv6ra1nlYgDCS1FRnbzlw"
	rfcBody      = "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
)

func TestEncryptKnownAnswerRFC8291(t *testing.T) {
	asPriv, err := ecdh.P256().NewPrivateKey(mustB64(t, rfcASPrivate))
	if err != nil {
		t.Fatal(err)
	}
	got, err := encryptWebPush(mustB64(t, rfcUAPublic), mustB64(t, rfcAuth), []byte(rfcPlaintext), asPriv, mustB64(t, rfcSalt))
	if err != nil {
		t.Fatal(err)
	}
	if enc := b64.EncodeToString(got); enc != rfcBody {
		t.Errorf("body mismatch\n got %s\nwant %s", enc, rfcBody)
	}
	// And the test receiver reads the RFC's own ciphertext.
	uaPriv, err := ecdh.P256().NewPrivateKey(mustB64(t, rfcUAPrivate))
	if err != nil {
		t.Fatal(err)
	}
	r := receiver{priv: uaPriv, auth: mustB64(t, rfcAuth)}
	if plain := r.decrypt(t, mustB64(t, rfcBody)); string(plain) != rfcPlaintext {
		t.Errorf("receiver decrypted %q", plain)
	}
}

func TestEncryptRoundTrip(t *testing.T) {
	r := newReceiver(t)
	for _, size := range []int{0, 1, 100, MaxWebPayload} {
		plain := bytes.Repeat([]byte("x"), size)
		body, err := newWebPushBody(r.priv.PublicKey().Bytes(), r.auth, plain)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if len(body) > 4096 {
			t.Errorf("size %d: body is %d bytes, push services accept 4096", size, len(body))
		}
		if got := r.decrypt(t, body); !bytes.Equal(got, plain) {
			t.Errorf("size %d: round trip mismatch", size)
		}
	}
	if _, err := newWebPushBody(r.priv.PublicKey().Bytes(), r.auth, make([]byte, MaxWebPayload+1)); !errors.Is(err, ErrPayloadTooLarge) {
		t.Errorf("oversized payload: err = %v", err)
	}
	// A second message uses a fresh key and salt.
	a, _ := newWebPushBody(r.priv.PublicKey().Bytes(), r.auth, []byte("hi"))
	b, _ := newWebPushBody(r.priv.PublicKey().Bytes(), r.auth, []byte("hi"))
	if bytes.Equal(a, b) {
		t.Error("two encryptions of the same plaintext are identical")
	}
}

// --- VAPID (RFC 8292) ---------------------------------------------------------------------

func newTestVAPID(t testing.TB) *VAPID {
	t.Helper()
	pub, priv, err := GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	v, err := NewVAPID(pub, priv, "mailto:admin@example.org")
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// verifyVAPID checks an Authorization header the way a push service does.
func verifyVAPID(t testing.TB, header, wantAud string, now time.Time) (claims map[string]any, key string) {
	t.Helper()
	rest, ok := strings.CutPrefix(header, "vapid ")
	if !ok {
		t.Fatalf("authorization %q is not the vapid scheme", header)
	}
	var jwt string
	for _, part := range strings.Split(rest, ", ") {
		if v, ok := strings.CutPrefix(part, "t="); ok {
			jwt = v
		}
		if v, ok := strings.CutPrefix(part, "k="); ok {
			key = v
		}
	}
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 || key == "" {
		t.Fatalf("malformed vapid header %q", header)
	}
	if h := string(mustB64(t, parts[0])); h != `{"typ":"JWT","alg":"ES256"}` {
		t.Errorf("jwt header %s", h)
	}
	if err := json.Unmarshal(mustB64(t, parts[1]), &claims); err != nil {
		t.Fatal(err)
	}
	if claims["aud"] != wantAud {
		t.Errorf("aud = %v, want %s", claims["aud"], wantAud)
	}
	exp, _ := claims["exp"].(float64)
	if d := time.Unix(int64(exp), 0).Sub(now); d <= 0 || d > 24*time.Hour {
		t.Errorf("exp is %v from now, want within 24h", d)
	}
	if claims["sub"] != "mailto:admin@example.org" {
		t.Errorf("sub = %v", claims["sub"])
	}
	sig := mustB64(t, parts[2])
	if len(sig) != 64 {
		t.Fatalf("signature is %d bytes, want 64 (r||s)", len(sig))
	}
	pub := mustB64(t, key)
	x, y := elliptic.Unmarshal(elliptic.P256(), pub) //nolint:staticcheck // test only
	if x == nil {
		t.Fatal("k is not a P-256 point")
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(&ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, digest[:],
		new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		t.Error("JWT signature does not verify")
	}
	return claims, key
}

func TestVAPIDAuthorization(t *testing.T) {
	v := newTestVAPID(t)
	now := time.Now()
	h, err := v.authorization("https://push.example.com:443/wpush/v2/abc", now)
	if err != nil {
		t.Fatal(err)
	}
	_, key := verifyVAPID(t, h, "https://push.example.com:443", now)
	if key != v.PublicKey() {
		t.Errorf("k = %s, want %s", key, v.PublicKey())
	}
}

func TestNewVAPIDValidation(t *testing.T) {
	pub, priv, _ := GenerateVAPIDKeys()
	otherPub, _, _ := GenerateVAPIDKeys()
	for name, tc := range map[string]struct{ pub, priv, sub string }{
		"mismatch":    {otherPub, priv, "mailto:a@b.de"},
		"bad subject": {pub, priv, "admin@example.org"},
		"short key":   {pub, "abc", "mailto:a@b.de"},
	} {
		if _, err := NewVAPID(tc.pub, tc.priv, tc.sub); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	t.Setenv(EnvVAPIDPublicKey, "")
	t.Setenv(EnvVAPIDPrivateKey, "")
	if _, err := VAPIDFromEnv(); !errors.Is(err, ErrVAPIDNotConfigured) {
		t.Errorf("empty env: %v", err)
	}
	t.Setenv(EnvVAPIDPublicKey, pub)
	if _, err := VAPIDFromEnv(); err == nil || errors.Is(err, ErrVAPIDNotConfigured) {
		t.Errorf("only public key: %v", err)
	}
	t.Setenv(EnvVAPIDPrivateKey, priv)
	t.Setenv(EnvVAPIDSubject, "mailto:a@b.de")
	if v, err := VAPIDFromEnv(); err != nil || v.PublicKey() != pub {
		t.Errorf("valid env: %v", err)
	}
}

// --- WebClient against a fake push service ------------------------------------------------

type pushRequest struct {
	header http.Header
	body   []byte
	path   string
}

// pushService is an httptest push service answering with status per path.
func pushService(t *testing.T, status map[string]int) (*httptest.Server, func() []pushRequest) {
	t.Helper()
	var mu sync.Mutex
	var got []pushRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(r.Body)
		mu.Lock()
		got = append(got, pushRequest{header: r.Header.Clone(), body: body.Bytes(), path: r.URL.Path})
		mu.Unlock()
		st, ok := status[r.URL.Path]
		if !ok {
			st = http.StatusCreated
		}
		if st == http.StatusTooManyRequests {
			w.Header().Set("Retry-After", "30")
		}
		w.WriteHeader(st)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []pushRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]pushRequest(nil), got...)
	}
}

func TestWebClientSend(t *testing.T) {
	srv, requests := pushService(t, nil)
	r := newReceiver(t)
	v := newTestVAPID(t)
	now := time.Unix(1_800_000_000, 0)
	c := &WebClient{VAPID: v, HTTPClient: srv.Client(), Now: func() time.Time { return now }}
	payload := []byte(`{"title":"Hallo","body":"Welt","data":{"screen":"/blankets"}}`)
	err := c.SendWeb(context.Background(), []WebMessage{{
		Sub: r.subscription(srv.URL + "/sub/1"), Payload: payload, TTL: 90 * time.Minute, Urgency: "high", Topic: "t1",
	}})
	if err != nil {
		t.Fatal(err)
	}
	reqs := requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests", len(reqs))
	}
	q := reqs[0]
	for k, want := range map[string]string{
		"Content-Encoding": "aes128gcm", "Content-Type": "application/octet-stream",
		"Ttl": "5400", "Urgency": "high", "Topic": "t1",
	} {
		if got := q.header.Get(k); got != want {
			t.Errorf("header %s = %q, want %q", k, got, want)
		}
	}
	verifyVAPID(t, q.header.Get("Authorization"), srv.URL, now)
	if got := r.decrypt(t, q.body); !bytes.Equal(got, payload) {
		t.Errorf("decrypted %q", got)
	}
}

func TestWebClientStatusHandling(t *testing.T) {
	srv, requests := pushService(t, map[string]int{
		"/gone": 410, "/missing": 404, "/big": 413, "/limited": 429, "/broken": 502, "/forbidden": 403,
	})
	r := newReceiver(t)
	c := &WebClient{VAPID: newTestVAPID(t), HTTPClient: srv.Client()}
	var msgs []WebMessage
	for _, p := range []string{"/ok", "/gone", "/missing", "/big", "/limited", "/broken", "/forbidden"} {
		msgs = append(msgs, WebMessage{Sub: r.subscription(srv.URL + p), Payload: []byte("{}")})
	}
	err := c.SendWeb(context.Background(), msgs)
	var serr *SendError
	if !errors.As(err, &serr) {
		t.Fatalf("err = %v, want *SendError", err)
	}
	if len(requests()) != len(msgs) {
		t.Errorf("%d requests, want %d (failures must not stop the rest)", len(requests()), len(msgs))
	}
	gone := map[string]bool{}
	for _, e := range serr.InvalidTokens {
		gone[strings.TrimPrefix(e, srv.URL)] = true
	}
	if len(gone) != 2 || !gone["/gone"] || !gone["/missing"] {
		t.Errorf("gone endpoints = %v, want /gone and /missing", gone)
	}
	if len(serr.Failures) != 4 {
		t.Fatalf("failures = %v, want 4 (413, 429, 502, 403)", serr.Failures)
	}
	all := strings.Join(serr.Failures, "\n")
	for _, want := range []string{"http 413", "http 429", "retry-after 30", "http 502", "http 403"} {
		if !strings.Contains(all, want) {
			t.Errorf("failures lack %q:\n%s", want, all)
		}
	}
	if strings.Contains(all, "/limited") || strings.Contains(all, "/broken") {
		t.Errorf("failure text leaks the endpoint path:\n%s", all)
	}
}

func TestWebClientRefusesPrivateAddresses(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("the guarded client connected to a loopback server")
	}))
	defer srv.Close()
	r := newReceiver(t)
	c := NewWebClient(newTestVAPID(t))
	err := c.SendWeb(context.Background(), []WebMessage{{Sub: r.subscription(srv.URL + "/x"), Payload: []byte("{}")}})
	var serr *SendError
	if !errors.As(err, &serr) || len(serr.Failures) != 1 || len(serr.InvalidTokens) != 0 {
		t.Fatalf("err = %v, want one failure", err)
	}
}

func TestValidateWebSubscription(t *testing.T) {
	r := newReceiver(t)
	good := r.subscription("https://web.push.apple.com/QAbc123")
	if got, err := ValidateWebSubscription(good); err != nil || got != good {
		t.Fatalf("valid subscription: %v %v", got, err)
	}
	padded := good
	padded.Auth += "=="
	if got, err := ValidateWebSubscription(padded); err != nil || got.Auth != good.Auth {
		t.Errorf("padded auth: %v %v", got, err)
	}
	mutate := func(f func(s *WebSubscription)) WebSubscription {
		s := good
		f(&s)
		return s
	}
	for name, s := range map[string]WebSubscription{
		"http":       mutate(func(s *WebSubscription) { s.Endpoint = "http://web.push.apple.com/x" }),
		"empty":      mutate(func(s *WebSubscription) { s.Endpoint = "" }),
		"ip literal": mutate(func(s *WebSubscription) { s.Endpoint = "https://10.0.0.1/x" }),
		"localhost":  mutate(func(s *WebSubscription) { s.Endpoint = "https://localhost/x" }),
		"port":       mutate(func(s *WebSubscription) { s.Endpoint = "https://push.example.com:8443/x" }),
		"userinfo":   mutate(func(s *WebSubscription) { s.Endpoint = "https://a:b@push.example.com/x" }),
		"long":       mutate(func(s *WebSubscription) { s.Endpoint = "https://push.example.com/" + strings.Repeat("a", 3000) }),
		"p256dh len": mutate(func(s *WebSubscription) { s.P256dh = b64.EncodeToString([]byte("short")) }),
		"p256dh pt":  mutate(func(s *WebSubscription) { s.P256dh = b64.EncodeToString(append([]byte{4}, make([]byte, 64)...)) }),
		"auth len":   mutate(func(s *WebSubscription) { s.Auth = b64.EncodeToString(make([]byte, 8)) }),
		"auth b64":   mutate(func(s *WebSubscription) { s.Auth = "***" }),
	} {
		if _, err := ValidateWebSubscription(s); !errors.Is(err, ErrInvalidSubscription) {
			t.Errorf("%s: err = %v, want ErrInvalidSubscription", name, err)
		}
	}
}
