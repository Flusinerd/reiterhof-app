package push

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// FCMSendURL is the HTTP v1 endpoint; %s is the Firebase project ID.
	FCMSendURL = "https://fcm.googleapis.com/v1/projects/%s/messages:send"
	fcmScope   = "https://www.googleapis.com/auth/firebase.messaging"
	// fcmAndroidChannel is the notification channel the app creates (mobile/lib/push.ts).
	fcmAndroidChannel = "default"
)

// FCMClient sends to Firebase Cloud Messaging (HTTP v1) as a service account: an OAuth 2.0
// access token obtained with a signed JWT, cached until shortly before it expires.
type FCMClient struct {
	projectID   string
	clientEmail string
	key         *rsa.PrivateKey
	tokenURI    string

	// SendURL defaults to FCMSendURL with the project ID; tests point it at httptest.
	SendURL string
	// TokenURL overrides the token endpoint of the service account file (tests).
	TokenURL string
	// HTTPClient defaults to a client with a 15 s timeout.
	HTTPClient *http.Client
	// Now defaults to time.Now.
	Now func() time.Time
	// Parallel is the number of concurrent requests, default 8.
	Parallel int

	mu        sync.Mutex
	access    string
	expiresAt time.Time
}

// serviceAccount is the part of the JSON key file the client needs.
type serviceAccount struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

// NewFCMClient parses a service account JSON key file from the Firebase console
// (Project settings, Service accounts, Generate new private key).
func NewFCMClient(serviceAccountJSON []byte) (*FCMClient, error) {
	var sa serviceAccount
	if err := json.Unmarshal(serviceAccountJSON, &sa); err != nil {
		return nil, fmt.Errorf("push: FCM service account is not valid JSON: %w", err)
	}
	if sa.Type != "service_account" || sa.ProjectID == "" || sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, errors.New("push: FCM service account needs type=service_account, project_id, client_email and private_key")
	}
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return nil, errors.New("push: FCM private_key is not PEM encoded")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("push: invalid FCM private_key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("push: FCM private_key must be an RSA key")
	}
	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}
	return &FCMClient{projectID: sa.ProjectID, clientEmail: sa.ClientEmail, key: key, tokenURI: tokenURI}, nil
}

// ProjectID is the Firebase project the client sends for.
func (c *FCMClient) ProjectID() string { return c.projectID }

func (c *FCMClient) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *FCMClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// accessToken returns the cached OAuth token or fetches one with a signed assertion.
func (c *FCMClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.access != "" && now.Before(c.expiresAt.Add(-time.Minute)) {
		return c.access, nil
	}
	assertion, err := signJWT(
		map[string]any{"alg": "RS256", "typ": "JWT"},
		map[string]any{
			"iss":   c.clientEmail,
			"scope": fcmScope,
			"aud":   c.tokenURI,
			"iat":   now.Unix(),
			"exp":   now.Add(time.Hour).Unix(),
		},
		rs256(c.key),
	)
	if err != nil {
		return "", fmt.Errorf("fcm: sign assertion: %w", err)
	}
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	tokenURL := c.TokenURL
	if tokenURL == "" {
		tokenURL = c.tokenURI
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("fcm: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("fcm: token: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fcm: token: http %d: %.200s", resp.StatusCode, bytes.TrimSpace(raw))
	}
	var out tokenResponse
	if err := json.Unmarshal(raw, &out); err != nil || out.AccessToken == "" {
		return "", errors.New("fcm: token: invalid response")
	}
	ttl := time.Duration(out.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	c.access, c.expiresAt = out.AccessToken, now.Add(ttl)
	return c.access, nil
}

// dropAccessToken forgets the cached token (after a 401).
func (c *FCMClient) dropAccessToken(tok string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.access == tok {
		c.access = ""
	}
}

// fcmRequest is the HTTP v1 send body. The message is data-only: expo-notifications on
// Android builds the notification itself from `title`, `message`, the JSON in `body`
// (exposed as the notification's data) and `channelId`, exactly like it did for the
// Expo push service; a `notification` block would bypass that and lose the data.
type fcmRequest struct {
	Message fcmMessage `json:"message"`
}

type fcmMessage struct {
	Token   string            `json:"token"`
	Data    map[string]string `json:"data"`
	Android fcmAndroid        `json:"android"`
}

type fcmAndroid struct {
	Priority string `json:"priority"`
	TTL      string `json:"ttl,omitempty"`
}

// Send delivers all messages, up to Parallel at a time. Problems are returned as *SendError;
// tokens FCM reports as unregistered or malformed are listed in InvalidTokens.
func (c *FCMClient) Send(ctx context.Context, msgs []Message) error {
	return sendParallel(c.Parallel, msgs, func(m Message) (bool, error) {
		return c.sendOne(ctx, m)
	})
}

func (c *FCMClient) sendOne(ctx context.Context, m Message) (invalid bool, err error) {
	data, err := json.Marshal(m.Data)
	if err != nil {
		return false, fmt.Errorf("fcm: marshal data: %w", err)
	}
	priority := "normal"
	if m.Priority == PriorityHigh {
		priority = "high"
	}
	var ttl string
	if m.TTL > 0 {
		ttl = strconv.FormatInt(int64(m.TTL/time.Second), 10) + "s"
	}
	body, err := json.Marshal(fcmRequest{Message: fcmMessage{
		Token: m.To,
		Data: map[string]string{
			"title":     m.Title,
			"message":   m.Body,
			"body":      string(data),
			"channelId": fcmAndroidChannel,
		},
		Android: fcmAndroid{Priority: priority, TTL: ttl},
	}})
	if err != nil {
		return false, fmt.Errorf("fcm: marshal: %w", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		tok, err := c.accessToken(ctx)
		if err != nil {
			return false, err
		}
		retry, invalid, err := c.post(ctx, tok, body)
		if retry {
			c.dropAccessToken(tok)
			continue
		}
		return invalid, err
	}
	return false, errors.New("fcm: access token rejected twice")
}

type fcmError struct {
	Error struct {
		Code    int    `json:"code"`
		Status  string `json:"status"`
		Message string `json:"message"`
		Details []struct {
			Type      string `json:"@type"`
			ErrorCode string `json:"errorCode"`
		} `json:"details"`
	} `json:"error"`
}

func (c *FCMClient) post(ctx context.Context, tok string, body []byte) (retry, invalid bool, err error) {
	sendURL := c.SendURL
	if sendURL == "" {
		sendURL = fmt.Sprintf(FCMSendURL, url.PathEscape(c.projectID))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendURL, bytes.NewReader(body))
	if err != nil {
		return false, false, fmt.Errorf("fcm: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return false, false, fmt.Errorf("fcm: %w", unwrapURLError(err))
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
	if resp.StatusCode == http.StatusOK {
		return false, false, nil
	}
	var out fcmError
	_ = json.Unmarshal(raw, &out)
	code := ""
	for _, d := range out.Error.Details {
		if d.ErrorCode != "" {
			code = d.ErrorCode
		}
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return true, false, nil
	case code == "UNREGISTERED", resp.StatusCode == http.StatusNotFound,
		// INVALID_ARGUMENT covers a bad payload too; only a rejected token is "invalid".
		code == "INVALID_ARGUMENT" && strings.Contains(strings.ToLower(out.Error.Message), "registration token"):
		return false, true, nil
	}
	msg := fmt.Sprintf("fcm: http %d", resp.StatusCode)
	if out.Error.Status != "" {
		msg += " " + out.Error.Status
	}
	if code != "" {
		msg += " " + code
	}
	if out.Error.Message != "" {
		msg += fmt.Sprintf(": %.200s", out.Error.Message)
	}
	return false, false, errors.New(msg)
}
