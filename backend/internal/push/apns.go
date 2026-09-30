package push

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// APNs hosts (HTTP/2). Development builds run against the sandbox.
const (
	APNSProductionURL = "https://api.push.apple.com"
	APNSSandboxURL    = "https://api.sandbox.push.apple.com"

	// apnsTokenLifetime is how long one provider token (JWT) is reused. Apple accepts
	// tokens for an hour and asks for at most one refresh per 20 minutes.
	apnsTokenLifetime = 50 * time.Minute
)

// apnsTokenRe is the shape of a device token as expo-notifications reports it on iOS.
var apnsTokenRe = regexp.MustCompile(`^[0-9a-fA-F]{32,}$`)

// APNSClient sends to the Apple Push Notification service with token-based
// authentication (an ES256 key from the developer account, the key ID and the team ID).
type APNSClient struct {
	key    *ecdsa.PrivateKey
	keyID  string
	teamID string
	// topic is the app's bundle identifier.
	topic string

	// BaseURL defaults to APNSProductionURL; tests point it at httptest.
	BaseURL string
	// HTTPClient defaults to NewAPNSHTTPClient(). APNs requires HTTP/2.
	HTTPClient *http.Client
	// Now defaults to time.Now.
	Now func() time.Time
	// Parallel is the number of concurrent requests, default 8.
	Parallel int

	mu      sync.Mutex
	token   string
	tokenAt time.Time
}

// NewAPNSClient parses the PEM encoded .p8 key from the developer account. topic is the
// bundle identifier of the app.
func NewAPNSClient(keyPEM []byte, keyID, teamID, topic string) (*APNSClient, error) {
	if keyID == "" || teamID == "" || topic == "" {
		return nil, errors.New("push: APNs key ID, team ID and topic are required")
	}
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("push: APNs key is not PEM encoded")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("push: invalid APNs key: %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("push: APNs key must be an EC (P-256) key")
	}
	return &APNSClient{key: key, keyID: keyID, teamID: teamID, topic: topic}, nil
}

// NewAPNSHTTPClient returns an HTTP client for APNs: 15 s timeout and HTTP/2, which the
// service requires.
func NewAPNSHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{ForceAttemptHTTP2: true, TLSHandshakeTimeout: 10 * time.Second},
	}
}

func (c *APNSClient) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// providerToken returns the cached JWT or a fresh one (ES256, kid = key ID, iss = team ID).
func (c *APNSClient) providerToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.token != "" && now.Sub(c.tokenAt) < apnsTokenLifetime {
		return c.token, nil
	}
	tok, err := signJWT(
		map[string]any{"alg": "ES256", "kid": c.keyID},
		map[string]any{"iss": c.teamID, "iat": now.Unix()},
		es256(c.key),
	)
	if err != nil {
		return "", fmt.Errorf("apns: sign provider token: %w", err)
	}
	c.token, c.tokenAt = tok, now
	return tok, nil
}

// dropToken forgets the cached JWT (after ExpiredProviderToken or a clock jump).
func (c *APNSClient) dropToken(tok string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == tok {
		c.token = ""
	}
}

// apnsPayload is what the device receives. expo-notifications reads the notification's
// title and body from aps.alert and exposes the custom `body` object as the data of the
// notification (mobile: notification.request.content.data).
type apnsPayload struct {
	APS  apnsAPS        `json:"aps"`
	Body map[string]any `json:"body,omitempty"`
}

type apnsAPS struct {
	Alert apnsAlert `json:"alert"`
	Sound string    `json:"sound,omitempty"`
}

type apnsAlert struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body"`
}

// Send delivers all messages, up to Parallel at a time. Problems are returned as *SendError;
// tokens Apple reports as gone or malformed are listed in InvalidTokens.
func (c *APNSClient) Send(ctx context.Context, msgs []Message) error {
	return sendParallel(c.Parallel, msgs, func(m Message) (bool, error) {
		return c.sendOne(ctx, m)
	})
}

// sendOne posts one notification. invalid reports a token that will never work again.
func (c *APNSClient) sendOne(ctx context.Context, m Message) (invalid bool, err error) {
	if !apnsTokenRe.MatchString(m.To) {
		return true, nil
	}
	payload, err := json.Marshal(apnsPayload{
		APS:  apnsAPS{Alert: apnsAlert{Title: m.Title, Body: m.Body}, Sound: m.Sound},
		Body: m.Data,
	})
	if err != nil {
		return false, fmt.Errorf("apns: marshal: %w", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		tok, err := c.providerToken()
		if err != nil {
			return false, err
		}
		retry, invalid, err := c.post(ctx, m, tok, payload)
		if retry {
			c.dropToken(tok)
			continue
		}
		return invalid, err
	}
	return false, errors.New("apns: provider token rejected twice")
}

type apnsResponse struct {
	Reason string `json:"reason"`
}

// post sends one request. retry asks for a fresh provider token and one more try.
func (c *APNSClient) post(ctx context.Context, m Message, tok string, payload []byte) (retry, invalid bool, err error) {
	base := c.BaseURL
	if base == "" {
		base = APNSProductionURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/3/device/"+m.To, bytes.NewReader(payload))
	if err != nil {
		return false, false, fmt.Errorf("apns: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer "+tok)
	req.Header.Set("apns-topic", c.topic)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "5")
	if m.Priority == PriorityHigh {
		req.Header.Set("apns-priority", "10")
	}
	if m.TTL > 0 {
		req.Header.Set("apns-expiration", strconv.FormatInt(c.now().Add(m.TTL).Unix(), 10))
	}
	hc := c.HTTPClient
	if hc == nil {
		hc = NewAPNSHTTPClient()
	}
	resp, err := hc.Do(req)
	if err != nil {
		// The URL holds the device token; keep it out of the error text.
		return false, false, fmt.Errorf("apns: %w", unwrapURLError(err))
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if resp.StatusCode == http.StatusOK {
		return false, false, nil
	}
	var out apnsResponse
	_ = json.Unmarshal(raw, &out)
	switch {
	case resp.StatusCode == http.StatusGone,
		out.Reason == "BadDeviceToken", out.Reason == "Unregistered", out.Reason == "DeviceTokenNotForTopic":
		return false, true, nil
	case out.Reason == "ExpiredProviderToken":
		return true, false, nil
	}
	msg := fmt.Sprintf("apns: http %d", resp.StatusCode)
	if out.Reason != "" {
		msg += " " + out.Reason
	}
	return false, false, errors.New(msg)
}

// sendParallel runs send for every message, parallel at a time, and collects the
// results into a *SendError (nil when everything went through).
func sendParallel(parallel int, msgs []Message, send func(Message) (invalid bool, err error)) error {
	if parallel <= 0 {
		parallel = 8
	}
	var (
		mu   sync.Mutex
		serr SendError
		wg   sync.WaitGroup
		sem  = make(chan struct{}, parallel)
	)
	for _, m := range msgs {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() { <-sem; wg.Done() }()
			invalid, err := send(m)
			mu.Lock()
			defer mu.Unlock()
			if invalid {
				serr.InvalidTokens = append(serr.InvalidTokens, m.To)
			}
			if err != nil {
				serr.Failures = append(serr.Failures, err.Error())
			}
		}()
	}
	wg.Wait()
	if len(serr.InvalidTokens) == 0 && len(serr.Failures) == 0 {
		return nil
	}
	return &serr
}
