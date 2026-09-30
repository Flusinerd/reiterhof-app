package push

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Message priorities (Message.Priority). High wakes the device even in battery saving
// modes; only urgent kinds use it.
const (
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)

// Message is one push notification for one device token.
type Message struct {
	// To is the native device token: the APNs token (hex) on iOS, the FCM registration
	// token on Android.
	To string
	// Platform is PlatformIOS or PlatformAndroid and picks the service.
	Platform string
	Title    string
	Body     string
	// Data reaches the app as the notification's data (screen, kind, ids).
	Data map[string]any
	// Sound is the iOS sound name ("default"); Android plays the channel's sound.
	Sound string
	// Priority is PriorityNormal (default) or PriorityHigh.
	Priority string
	// TTL is how long the service keeps the message for an offline device; 0 means the
	// service default.
	TTL time.Duration
}

// Sender delivers messages. Implementations: Client (APNs and FCM), APNSClient,
// FCMClient and Fake (tests).
type Sender interface {
	// Send delivers all messages. If some devices are gone or some tickets
	// failed, it still processes everything and returns a *SendError.
	Send(ctx context.Context, msgs []Message) error
}

// SendError reports per-message problems of an otherwise processed batch.
type SendError struct {
	// InvalidTokens are tokens the service reported as gone or malformed (APNs
	// BadDeviceToken/Unregistered, FCM UNREGISTERED); callers should delete them.
	InvalidTokens []string
	// Failures are descriptions of all other failed tickets or batches.
	Failures []string
}

func (e *SendError) Error() string {
	var parts []string
	if len(e.InvalidTokens) > 0 {
		parts = append(parts, fmt.Sprintf("%d invalid token(s)", len(e.InvalidTokens)))
	}
	parts = append(parts, e.Failures...)
	return "push: " + strings.Join(parts, "; ")
}

// Fake is an in-memory Sender for tests of other packages.
type Fake struct {
	mu   sync.Mutex
	sent []Message
	// Invalid holds tokens that Send reports as invalid.
	Invalid map[string]bool
	// Err, if set, is returned by Send instead of delivering.
	Err error
}

// Send records the messages.
func (f *Fake) Send(_ context.Context, msgs []Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Err != nil {
		return f.Err
	}
	var bad []string
	for _, m := range msgs {
		if f.Invalid[m.To] {
			bad = append(bad, m.To)
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
func (f *Fake) Sent() []Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Message(nil), f.sent...)
}

// Reset clears the recorded messages.
func (f *Fake) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = nil
}
