package push

import (
	"context"
	"crypto/ecdh"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxEndpointLen = 2048
	// MaxWebSubscriptionsPerUser bounds the browsers one user can register; the oldest go first.
	MaxWebSubscriptionsPerUser = 10
	maxUserAgentLen            = 300
)

// ErrInvalidSubscription is returned for a PushSubscription that fails validation.
var ErrInvalidSubscription = errors.New("push: invalid web push subscription")

// hostOf returns the host name of an https URL.
func hostOf(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}

// ValidateWebSubscription checks a PushSubscription received from a client and returns it
// with the keys in unpadded base64url. The endpoint must be a plain https URL to a named
// host (no IP literal, no credentials, no odd port), p256dh a valid uncompressed P-256
// point and auth 16 bytes. The sender additionally refuses non-public addresses when
// connecting (NewWebHTTPClient).
func ValidateWebSubscription(sub WebSubscription) (WebSubscription, error) {
	bad := func(format string, args ...any) (WebSubscription, error) {
		return WebSubscription{}, fmt.Errorf("%w: %s", ErrInvalidSubscription, fmt.Sprintf(format, args...))
	}
	if sub.Endpoint == "" || len(sub.Endpoint) > maxEndpointLen {
		return bad("endpoint missing or too long")
	}
	u, err := url.Parse(sub.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return bad("endpoint must be an https URL")
	}
	host := u.Hostname()
	if net.ParseIP(host) != nil || !strings.Contains(host, ".") || strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".localhost") {
		return bad("endpoint must use a public host name")
	}
	if p := u.Port(); p != "" && p != "443" {
		return bad("endpoint port must be 443")
	}
	p256dh, err := decodeB64(sub.P256dh)
	if err != nil {
		return bad("p256dh is not base64url")
	}
	if _, err := ecdh.P256().NewPublicKey(p256dh); err != nil {
		return bad("p256dh is not an uncompressed P-256 point")
	}
	auth, err := decodeB64(sub.Auth)
	if err != nil || len(auth) != 16 {
		return bad("auth must be 16 bytes, base64url")
	}
	return WebSubscription{
		Endpoint: sub.Endpoint,
		P256dh:   b64.EncodeToString(p256dh),
		Auth:     b64.EncodeToString(auth),
	}, nil
}

// RegisterWebSubscription stores a browser subscription for a user (already validated with
// ValidateWebSubscription). Like RegisterToken it upserts by endpoint, so a browser handed to
// another user is re-assigned, and the user must belong to the stable (else ErrUnknownUser).
// A user keeps at most MaxWebSubscriptionsPerUser subscriptions; the oldest are dropped.
func RegisterWebSubscription(ctx context.Context, pool *pgxpool.Pool, stableID, userID string, sub WebSubscription, userAgent string) error {
	if len(userAgent) > maxUserAgentLen {
		userAgent = userAgent[:maxUserAgentLen]
	}
	userAgent = strings.ToValidUTF8(userAgent, "")
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		INSERT INTO web_push_subscriptions (stable_id, user_id, endpoint, p256dh, auth, user_agent)
		SELECT u.stable_id, u.id, $3, $4, $5, $6 FROM users u WHERE u.id = $2 AND u.stable_id = $1
		ON CONFLICT (endpoint) DO UPDATE
		SET stable_id = EXCLUDED.stable_id, user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh,
		    auth = EXCLUDED.auth, user_agent = EXCLUDED.user_agent, created_at = now()`,
		stableID, userID, sub.Endpoint, sub.P256dh, sub.Auth, userAgent)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUnknownUser
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM web_push_subscriptions
		WHERE id IN (SELECT id FROM web_push_subscriptions WHERE user_id = $1
		             ORDER BY created_at DESC, id OFFSET $2)`, userID, MaxWebSubscriptionsPerUser); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteWebSubscription removes a user's subscription. It reports whether a row was deleted.
func DeleteWebSubscription(ctx context.Context, pool *pgxpool.Pool, stableID, userID, endpoint string) (bool, error) {
	tag, err := pool.Exec(ctx,
		`DELETE FROM web_push_subscriptions WHERE stable_id = $1 AND user_id = $2 AND endpoint = $3`,
		stableID, userID, endpoint)
	return tag.RowsAffected() > 0, err
}
