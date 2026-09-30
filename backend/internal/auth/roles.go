package auth

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Role model (JAN-7). Roles are derived from data, not stored:
//
//   - member: any user with a stable (User.StableID != ""). Sees presence, the blanket list,
//     requests and emergency cards; may accept requests, blanket horses and report.
//   - rider (RB): listed in horse_riders for a horse. Positive rules list what the rider may
//     do. May log sessions, report observations and take week slots for that horse, but never
//     changes rules or profiles.
//   - owner: horses.owner_id = user. Full control of the own horse (profile, riders, rules).
//   - admin: User.IsAdmin. May do everything owners may, plus create invites.
//
// Every query filters by the user's stable_id, so ids from other stables never match.
// All helpers return false (never an error) for a user without a stable.

// Querier is what the role helpers need; *pgxpool.Pool and pgx.Tx satisfy it.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// HorseInStable reports whether the horse exists in the user's stable.
func HorseInStable(ctx context.Context, q Querier, u User, horseID string) (bool, error) {
	if u.StableID == "" {
		return false, nil
	}
	return exists(ctx, q, `SELECT EXISTS (SELECT 1 FROM horses WHERE id = $1 AND stable_id = $2)`,
		horseID, u.StableID)
}

// IsOwner reports whether the user owns the horse (in the user's stable).
func IsOwner(ctx context.Context, q Querier, u User, horseID string) (bool, error) {
	if u.StableID == "" {
		return false, nil
	}
	return exists(ctx, q, `SELECT EXISTS (SELECT 1 FROM horses WHERE id = $1 AND stable_id = $2 AND owner_id = $3)`,
		horseID, u.StableID, u.ID)
}

// CanManageHorse reports whether the user may change the horse's profile, riders and rules:
// the owner, or an admin of the horse's stable.
func CanManageHorse(ctx context.Context, q Querier, u User, horseID string) (bool, error) {
	if u.IsAdmin {
		return HorseInStable(ctx, q, u, horseID)
	}
	return IsOwner(ctx, q, u, horseID)
}

// IsRider reports whether the user is listed as rider (RB) of the horse.
// Owners are not automatically riders here; combine with IsOwner/CanManageHorse.
func IsRider(ctx context.Context, q Querier, u User, horseID string) (bool, error) {
	if u.StableID == "" {
		return false, nil
	}
	return exists(ctx, q, `SELECT EXISTS (SELECT 1 FROM horse_riders WHERE horse_id = $1 AND user_id = $2 AND stable_id = $3)`,
		horseID, u.ID, u.StableID)
}

// RiderRules returns the positive rule list (e.g. ["ride","groom"]) the user has for the
// horse, and whether the user is a rider of it at all.
func RiderRules(ctx context.Context, q Querier, u User, horseID string) (rules []string, isRider bool, err error) {
	if u.StableID == "" {
		return nil, false, nil
	}
	var raw []byte
	err = q.QueryRow(ctx, `SELECT rules FROM horse_riders WHERE horse_id = $1 AND user_id = $2 AND stable_id = $3`,
		horseID, u.ID, u.StableID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if isInvalidUUID(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if raw == nil {
		return nil, false, nil
	}
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, true, err
	}
	return rules, true, nil
}

func exists(ctx context.Context, q Querier, sql string, args ...any) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, sql, args...).Scan(&ok)
	if isInvalidUUID(err) {
		return false, nil
	}
	return ok, err
}

// isInvalidUUID detects "not a uuid" input (SQLSTATE 22P02), so a garbage id in a URL
// yields a plain "no" (404/403) rather than a 500.
func isInvalidUUID(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "22P02"
}
