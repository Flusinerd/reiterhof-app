package push

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Notifier sends notifications to users of a stable, honoring reminder_settings and
// the push consent.
type Notifier struct {
	pool   *pgxpool.Pool
	sender Sender
	log    *slog.Logger
}

// NewNotifier creates a Notifier. log may be nil.
func NewNotifier(pool *pgxpool.Pool, sender Sender, log *slog.Logger) *Notifier {
	if log == nil {
		log = slog.Default()
	}
	return &Notifier{pool: pool, sender: sender, log: log}
}

// NotifyUsers pushes a notification of the given kind to all devices of userIDs
// within stableID. Only users with a current `push` consent (consents row, not
// revoked; a user who never consented gets nothing) are notified. Users who disabled
// the kind in reminder_settings are skipped (no row means enabled), except for opt-in
// kinds (see OptIn): those are only sent to users with a reminder_settings row that
// enables them. Tokens Expo reports as invalid are deleted. Delivery problems other
// than invalid tokens are returned.
func (n *Notifier) NotifyUsers(ctx context.Context, stableID string, userIDs []string, kind, title, body string, data map[string]any) error {
	if !ValidKind(kind) {
		return fmt.Errorf("push: unknown kind %q", kind)
	}
	if len(userIDs) == 0 {
		return nil
	}
	rows, err := n.pool.Query(ctx, `
		SELECT t.token
		FROM push_tokens t
		WHERE t.stable_id = $1
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
		      END
		ORDER BY t.created_at, t.token`, stableID, userIDs, kind, OptIn(kind))
	if err != nil {
		return fmt.Errorf("push: load tokens: %w", err)
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return fmt.Errorf("push: scan token: %w", err)
		}
		d := make(map[string]any, len(data)+1)
		for k, v := range data {
			d[k] = v
		}
		d["kind"] = kind
		msgs = append(msgs, Message{To: token, Title: title, Body: body, Data: d, Sound: "default"})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("push: load tokens: %w", err)
	}
	rows.Close()
	if len(msgs) == 0 {
		return nil
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
