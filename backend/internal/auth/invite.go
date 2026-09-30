package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateInvite stores a new invite code for the stable and returns it (without the display
// dash, see FormatInviteCode). createdBy may be nil (created by an operator, not a member).
// It retries on the (very unlikely) code collision.
func CreateInvite(ctx context.Context, pool *pgxpool.Pool, stableID string, createdBy *string, expires time.Time, maxUses int) (string, error) {
	for range 5 {
		code, err := NewInviteCode()
		if err != nil {
			return "", err
		}
		tag, err := pool.Exec(ctx, `INSERT INTO stable_invites (stable_id, code, created_by, expires_at, max_uses)
			VALUES ($1, $2, $3, $4, $5) ON CONFLICT (code) DO NOTHING`, stableID, code, createdBy, expires, maxUses)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 1 {
			return code, nil
		}
	}
	return "", errors.New("could not find a free invite code")
}
