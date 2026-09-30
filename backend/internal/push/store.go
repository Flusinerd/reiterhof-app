package push

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Platforms accepted by push_tokens.platform.
const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
)

// ErrUnknownUser is returned by RegisterToken when userID is not a user of stableID.
var ErrUnknownUser = errors.New("push: user not found in stable")

// RegisterToken stores an Expo push token for a user. A token belongs to one
// device, so an existing row with the same token is re-assigned (device handed
// to another user or stable).
func RegisterToken(ctx context.Context, pool *pgxpool.Pool, stableID, userID, token, platform string) error {
	if token == "" {
		return fmt.Errorf("push: empty token")
	}
	if platform != PlatformIOS && platform != PlatformAndroid {
		return fmt.Errorf("push: invalid platform %q", platform)
	}
	tag, err := pool.Exec(ctx, `
		INSERT INTO push_tokens (stable_id, user_id, token, platform)
		SELECT u.stable_id, u.id, $3, $4 FROM users u WHERE u.id = $2 AND u.stable_id = $1
		ON CONFLICT (token) DO UPDATE
		SET stable_id = EXCLUDED.stable_id, user_id = EXCLUDED.user_id, platform = EXCLUDED.platform`,
		stableID, userID, token, platform)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUnknownUser
	}
	return nil
}

// DeleteToken removes a user's token. It reports whether a row was deleted.
func DeleteToken(ctx context.Context, pool *pgxpool.Pool, stableID, userID, token string) (bool, error) {
	tag, err := pool.Exec(ctx,
		`DELETE FROM push_tokens WHERE stable_id = $1 AND user_id = $2 AND token = $3`,
		stableID, userID, token)
	return tag.RowsAffected() > 0, err
}
