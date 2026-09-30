package push

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// WebSubscription is the part of a browser PushSubscription needed to send: the push
// service URL and the keys of the browser (both base64url, as the browser reports them).
type WebSubscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// WebMessage is one Web Push message for one subscription.
type WebMessage struct {
	Sub WebSubscription
	// Payload is the plaintext (JSON for the service worker); at most MaxWebPayload bytes.
	Payload []byte
	// TTL is how long the push service keeps the message for an offline device; 0 means 24 h.
	TTL time.Duration
	// Urgency is the RFC 8030 header: very-low, low, normal or high; empty means normal.
	Urgency string
	// Topic optionally replaces an undelivered message with the same topic (max 32 chars).
	Topic string
}

// WebSender delivers Web Push messages. Implementations: WebClient and FakeWeb (tests).
// Like Sender.Send it processes everything and reports problems as a *SendError, where
// InvalidTokens holds the endpoints the push service reported as gone (404/410).
type WebSender interface {
	SendWeb(ctx context.Context, msgs []WebMessage) error
}

const defaultWebTTL = 24 * time.Hour

// WebClient is a WebSender that posts to the push services (RFC 8030) with VAPID (RFC 8292).
type WebClient struct {
	VAPID *VAPID
	// HTTPClient defaults to NewWebHTTPClient(). Tests pass their own.
	HTTPClient *http.Client
	// Now defaults to time.Now.
	Now func() time.Time
	// Parallel is the number of concurrent requests, default 8.
	Parallel int
}

// NewWebClient returns a WebClient with the SSRF-guarded HTTP client.
func NewWebClient(v *VAPID) *WebClient {
	return &WebClient{VAPID: v, HTTPClient: NewWebHTTPClient()}
}

// NewWebHTTPClient returns an HTTP client for push services: 15 s timeout, no redirects,
// and it refuses to connect to loopback, private or link-local addresses, because the
// endpoint URL is chosen by the (signed-in) client.
func NewWebHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
				ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
				return errors.New("push: refusing to connect to a non-public address")
			}
			return nil
		},
	}
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{DialContext: dialer.DialContext, TLSHandshakeTimeout: 10 * time.Second},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// SendWeb delivers all messages, up to Parallel at a time. Problems are returned as *SendError.
func (c *WebClient) SendWeb(ctx context.Context, msgs []WebMessage) error {
	parallel := c.Parallel
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
			gone, err := c.sendOne(ctx, m)
			mu.Lock()
			defer mu.Unlock()
			if gone {
				serr.InvalidTokens = append(serr.InvalidTokens, m.Sub.Endpoint)
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

// sendOne posts one message. gone reports a subscription the push service dropped.
func (c *WebClient) sendOne(ctx context.Context, m WebMessage) (gone bool, err error) {
	// Never put the endpoint (a secret capability URL) into error text; use its host.
	host := endpointHost(m.Sub.Endpoint)
	uaPublic, err := decodeB64(m.Sub.P256dh)
	if err != nil {
		return false, fmt.Errorf("web push %s: invalid p256dh: %w", host, err)
	}
	authSecret, err := decodeB64(m.Sub.Auth)
	if err != nil {
		return false, fmt.Errorf("web push %s: invalid auth: %w", host, err)
	}
	body, err := newWebPushBody(uaPublic, authSecret, m.Payload)
	if err != nil {
		return false, fmt.Errorf("web push %s: %w", host, err)
	}
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	authz, err := c.VAPID.authorization(m.Sub.Endpoint, now())
	if err != nil {
		return false, fmt.Errorf("web push %s: %w", host, err)
	}
	ttl := m.TTL
	if ttl <= 0 {
		ttl = defaultWebTTL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.Sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("web push %s: %w", host, err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Authorization", authz)
	req.Header.Set("TTL", strconv.FormatInt(int64(ttl/time.Second), 10))
	if m.Urgency != "" {
		req.Header.Set("Urgency", m.Urgency)
	}
	if m.Topic != "" {
		req.Header.Set("Topic", m.Topic)
	}
	hc := c.HTTPClient
	if hc == nil {
		hc = NewWebHTTPClient()
	}
	resp, err := hc.Do(req)
	if err != nil {
		return false, fmt.Errorf("web push %s: %w", host, unwrapURLError(err))
	}
	defer resp.Body.Close()
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return false, nil
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return true, nil
	default:
		// 413 (too large), 429 (rate limited, see Retry-After), 401/403 (VAPID rejected), 5xx.
		msg := fmt.Sprintf("web push %s: http %d: %s", host, resp.StatusCode, bytes.TrimSpace(snippet))
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			msg += " (retry-after " + ra + ")"
		}
		return false, errors.New(msg)
	}
}

func endpointHost(endpoint string) string {
	if h, err := hostOf(endpoint); err == nil {
		return h
	}
	return "?"
}

// unwrapURLError drops the URL (which contains the secret endpoint) from a client error.
func unwrapURLError(err error) error {
	var ue interface{ Unwrap() error }
	if errors.As(err, &ue) && ue.Unwrap() != nil {
		return ue.Unwrap()
	}
	return err
}

// FakeWeb is an in-memory WebSender for tests of other packages.
type FakeWeb struct {
	mu   sync.Mutex
	sent []WebMessage
	// Gone holds endpoints that SendWeb reports as expired (404/410).
	Gone map[string]bool
	// Err, if set, is returned by SendWeb instead of delivering.
	Err error
}

// SendWeb records the messages.
func (f *FakeWeb) SendWeb(_ context.Context, msgs []WebMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Err != nil {
		return f.Err
	}
	var bad []string
	for _, m := range msgs {
		if f.Gone[m.Sub.Endpoint] {
			bad = append(bad, m.Sub.Endpoint)
			continue
		}
		f.sent = append(f.sent, m)
	}
	if len(bad) > 0 {
		return &SendError{InvalidTokens: bad}
	}
	return nil
}

// Sent returns a copy of all successfully "delivered" messages.
func (f *FakeWeb) Sent() []WebMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]WebMessage(nil), f.sent...)
}

// Reset clears the recorded messages.
func (f *FakeWeb) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = nil
}
