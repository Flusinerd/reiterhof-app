package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Notifier sends notifications to users of a stable, honoring reminder_settings and
// the push consent. Devices are native tokens (Sender: APNs and FCM) and, when a
// WebSender is set, browser subscriptions of the PWA.
type Notifier struct {
	pool   *pgxpool.Pool
	sender Sender
	web    WebSender
	log    *slog.Logger
}

// NewNotifier creates a Notifier. log may be nil.
func NewNotifier(pool *pgxpool.Pool, sender Sender, log *slog.Logger) *Notifier {
	if log == nil {
		log = slog.Default()
	}
	return &Notifier{pool: pool, sender: sender, log: log}
}

// WithWeb enables Web Push delivery and returns the notifier. Without it, web
// subscriptions are skipped (with a log line).
func (n *Notifier) WithWeb(web WebSender) *Notifier {
	n.web = web
	return n
}

// recipientFilter is the WHERE clause both device tables share (alias t, both have
// stable_id and user_id): the users in the stable ($1, $2) with a current `push` consent
// who did not switch the kind ($3) off in reminder_settings; for opt-in kinds ($4) only
// users who switched it on. Native tokens and web subscriptions must use this one filter,
// so consent and opt-outs cannot diverge between the two paths.
const recipientFilter = `
		t.stable_id = $1
		  AND t.user_id = ANY($2::uuid[])
		  AND EXISTS (
		      SELECT 1 FROM consents c
		      WHERE c.user_id = t.user_id AND c.kind = 'push' AND c.revoked_at IS NULL)
		  AND CASE WHEN $4::boolean
		      THEN EXISTS (
		          SELECT 1 FROM reminder_settings s
		          WHERE s.stable_id = t.stable_id AND s.user_id = t.user_id
		            AND s.kind = $3 AND s.enabled)
		      ELSE NOT EXISTS (
		          SELECT 1 FROM reminder_settings s
		          WHERE s.stable_id = t.stable_id AND s.user_id = t.user_id
		            AND s.kind = $3 AND NOT s.enabled)
		      END`

// NotifyUsers pushes a notification of the given kind to all devices of userIDs
// within stableID. Only users with a current `push` consent (consents row, not
// revoked; a user who never consented gets nothing) are notified. Users who disabled
// the kind in reminder_settings are skipped (no row means enabled), except for opt-in
// kinds (see OptIn): those are only sent to users with a reminder_settings row that
// enables them. Tokens APNs or FCM report as invalid and subscriptions the push service
// reports as gone (404/410) are deleted. Delivery problems other than invalid tokens
// are returned; a failing web delivery does not stop the native one and vice versa.
func (n *Notifier) NotifyUsers(ctx context.Context, stableID string, userIDs []string, kind, title, body string, data map[string]any) error {
	if !ValidKind(kind) {
		return fmt.Errorf("push: unknown kind %q", kind)
	}
	if len(userIDs) == 0 {
		return nil
	}
	d := make(map[string]any, len(data)+1)
	for k, v := range data {
		d[k] = v
	}
	d["kind"] = kind

	nativeErr := n.notifyNative(ctx, stableID, userIDs, kind, title, body, d)
	webErr := n.notifyWeb(ctx, stableID, userIDs, kind, title, body, d)
	return errors.Join(nativeErr, webErr)
}

type device struct {
	token, platform string
}

func (n *Notifier) notifyNative(ctx context.Context, stableID string, userIDs []string, kind, title, body string, data map[string]any) error {
	rows, err := n.pool.Query(ctx, `
		SELECT t.token, t.platform
		FROM push_tokens t
		WHERE `+recipientFilter+`
		ORDER BY t.created_at, t.token`, stableID, userIDs, kind, OptIn(kind))
	if err != nil {
		return fmt.Errorf("push: load tokens: %w", err)
	}
	devices, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (device, error) {
		var d device
		err := row.Scan(&d.token, &d.platform)
		return d, err
	})
	if err != nil {
		return fmt.Errorf("push: load tokens: %w", err)
	}
	if len(devices) == 0 {
		return nil
	}
	msgs := make([]Message, 0, len(devices))
	for _, d := range devices {
		// Every message gets its own copy of the data map.
		msgs = append(msgs, Message{
			To: d.token, Platform: d.platform, Title: title, Body: body, Data: cloneData(data),
			Sound: "default", Priority: priority(kind), TTL: webTTL(kind),
		})
	}

	err = n.sender.Send(ctx, msgs)
	var serr *SendError
	if !errors.As(err, &serr) {
		return err
	}
	for _, tok := range serr.InvalidTokens {
		if _, derr := n.pool.Exec(ctx,
			`DELETE FROM push_tokens WHERE stable_id = $1 AND token = $2`, stableID, tok); derr != nil {
			n.log.Error("push: delete invalid token", "err", derr)
		}
	}
	if len(serr.Failures) == 0 {
		return nil
	}
	return &SendError{Failures: serr.Failures}
}

// webPayload is what the service worker (mobile/public/sw.js) receives.
type webPayload struct {
	Title string         `json:"title"`
	Body  string         `json:"body"`
	Data  map[string]any `json:"data"`
}

func (n *Notifier) notifyWeb(ctx context.Context, stableID string, userIDs []string, kind, title, body string, data map[string]any) error {
	rows, err := n.pool.Query(ctx, `
		SELECT t.endpoint, t.p256dh, t.auth
		FROM web_push_subscriptions t
		WHERE `+recipientFilter+`
		ORDER BY t.created_at, t.endpoint`, stableID, userIDs, kind, OptIn(kind))
	if err != nil {
		return fmt.Errorf("push: load web subscriptions: %w", err)
	}
	subs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (WebSubscription, error) {
		var s WebSubscription
		err := row.Scan(&s.Endpoint, &s.P256dh, &s.Auth)
		return s, err
	})
	if err != nil {
		return fmt.Errorf("push: load web subscriptions: %w", err)
	}
	if len(subs) == 0 {
		return nil
	}
	if n.web == nil {
		n.log.Info("push: web push is not configured, skipping web subscriptions", "count", len(subs), "kind", kind)
		return nil
	}
	payload, err := json.Marshal(webPayload{Title: title, Body: body, Data: data})
	if err != nil {
		return fmt.Errorf("push: marshal web payload: %w", err)
	}
	if len(payload) > MaxWebPayload {
		return fmt.Errorf("push: web payload of kind %q: %w", kind, ErrPayloadTooLarge)
	}
	msgs := make([]WebMessage, 0, len(subs))
	for _, s := range subs {
		msgs = append(msgs, WebMessage{Sub: s, Payload: payload, TTL: webTTL(kind), Urgency: webUrgency(kind)})
	}

	err = n.web.SendWeb(ctx, msgs)
	var serr *SendError
	if !errors.As(err, &serr) {
		return err
	}
	for _, endpoint := range serr.InvalidTokens {
		if _, derr := n.pool.Exec(ctx,
			`DELETE FROM web_push_subscriptions WHERE stable_id = $1 AND endpoint = $2`, stableID, endpoint); derr != nil {
			n.log.Error("push: delete expired web subscription", "err", derr)
		}
	}
	if len(serr.Failures) == 0 {
		return nil
	}
	return &SendError{Failures: serr.Failures}
}

func cloneData(data map[string]any) map[string]any {
	out := make(map[string]any, len(data))
	for k, v := range data {
		out[k] = v
	}
	return out
}

// priority is PriorityHigh for the kinds that should wake the device even in battery
// saving mode: urgent alarms and the "last person" reminder. Web Push uses the same
// split as its RFC 8030 urgency (webUrgency).
func priority(kind string) string {
	if kind == KindUrgentObservation || kind == KindLastPerson {
		return PriorityHigh
	}
	return PriorityNormal
}

// webUrgency is the RFC 8030 urgency, see priority.
func webUrgency(kind string) string {
	if priority(kind) == PriorityHigh {
		return "high"
	}
	return "normal"
}

// webTTL is how long a push service keeps a message for an offline device: reminders
// that are wrong after a few hours expire sooner.
func webTTL(kind string) time.Duration {
	switch kind {
	case KindLastPerson:
		return 4 * time.Hour
	case KindWeatherChange:
		return 6 * time.Hour
	}
	return defaultWebTTL
}
