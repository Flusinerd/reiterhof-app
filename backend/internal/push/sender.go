package push

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Message is one push notification for one device token.
type Message struct {
	To    string         `json:"to"`
	Title string         `json:"title,omitempty"`
	Body  string         `json:"body"`
	Data  map[string]any `json:"data,omitempty"`
	Sound string         `json:"sound,omitempty"`
}

// Sender delivers messages. Implementations: Client (Expo) and Fake (tests).
type Sender interface {
	// Send delivers all messages. If some devices are gone or some tickets
	// failed, it still processes everything and returns a *SendError.
	Send(ctx context.Context, msgs []Message) error
}

// SendError reports per-message problems of an otherwise processed batch.
type SendError struct {
	// InvalidTokens are tokens Expo reported as DeviceNotRegistered; callers
	// should delete them.
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
	// Invalid holds tokens that Send reports as DeviceNotRegistered.
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
