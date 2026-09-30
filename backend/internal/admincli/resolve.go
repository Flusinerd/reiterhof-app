package admincli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool { return uuidRe.MatchString(s) }

func (e *Env) writeJSON(v any) error {
	enc := json.NewEncoder(e.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// --- stables -----------------------------------------------------------------------

type stableRef struct {
	ID   string
	Name string
}

// resolveStable finds the stable named by --stable (UUID or exact name, case-insensitive).
// Without a value it returns the only stable, or an error when there are none or several.
func (e *Env) resolveStable(ctx context.Context, ref string) (stableRef, error) {
	pool, err := e.pool(ctx)
	if err != nil {
		return stableRef{}, err
	}
	ref = strings.TrimSpace(ref)
	if ref != "" {
		var rows pgx.Rows
		if isUUID(ref) {
			rows, err = pool.Query(ctx, `SELECT id::text, name FROM stables WHERE id = $1`, ref)
		} else {
			rows, err = pool.Query(ctx, `SELECT id::text, name FROM stables WHERE lower(name) = lower($1) ORDER BY created_at`, ref)
		}
		if err != nil {
			return stableRef{}, err
		}
		found, err := pgx.CollectRows(rows, pgx.RowToStructByPos[stableRef])
		if err != nil {
			return stableRef{}, err
		}
		switch len(found) {
		case 0:
			return stableRef{}, fmt.Errorf("no stable %q (see: stallfunk-admin stable list)", ref)
		case 1:
			return found[0], nil
		}
		return stableRef{}, fmt.Errorf("several stables are named %q, use the id:\n%s", ref, describeStables(found))
	}
	rows, err := pool.Query(ctx, `SELECT id::text, name FROM stables ORDER BY created_at, id`)
	if err != nil {
		return stableRef{}, err
	}
	all, err := pgx.CollectRows(rows, pgx.RowToStructByPos[stableRef])
	if err != nil {
		return stableRef{}, err
	}
	switch len(all) {
	case 0:
		return stableRef{}, errors.New("there is no stable yet; create one with: stallfunk-admin stable create --name ...")
	case 1:
		return all[0], nil
	}
	return stableRef{}, fmt.Errorf("several stables exist, pass --stable ID:\n%s", describeStables(all))
}

func describeStables(s []stableRef) string {
	var b strings.Builder
	for _, st := range s {
		fmt.Fprintf(&b, "  %s  %s\n", st.ID, st.Name)
	}
	return strings.TrimRight(b.String(), "\n")
}

// optionalStable is like resolveStable for list commands: an empty ref means "all
// stables" and returns "".
func (e *Env) optionalStable(ctx context.Context, ref string) (string, error) {
	if strings.TrimSpace(ref) == "" {
		return "", nil
	}
	st, err := e.resolveStable(ctx, ref)
	return st.ID, err
}

// --- users -------------------------------------------------------------------------

// userRow is a member (not an anonymised, deleted account).
type userRow struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	StableID   *string   `json:"stable_id"`
	StableName *string   `json:"stable_name"`
	IsAdmin    bool      `json:"is_admin"`
	CreatedAt  time.Time `json:"created_at"`
}

const userSelect = `SELECT u.id::text, u.name, u.email, u.stable_id::text, s.name, u.is_admin, u.created_at
	FROM users u LEFT JOIN stables s ON s.id = u.stable_id`

func (u userRow) label() string { return fmt.Sprintf("%s <%s>", u.Name, u.Email) }

func (u userRow) stableLabel() string {
	if u.StableName == nil {
		return "-"
	}
	return *u.StableName
}

// queryer is what the lookups need; *pgxpool.Pool and pgx.Tx satisfy it.
type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// findUser looks a user up by UUID or e-mail address (case-insensitive).
func findUser(ctx context.Context, q queryer, ref string) (userRow, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return userRow{}, usagef("missing user (email or id)")
	}
	var row pgx.Row
	if isUUID(ref) {
		row = q.QueryRow(ctx, userSelect+` WHERE u.id = $1 AND u.deleted_at IS NULL`, ref)
	} else {
		row = q.QueryRow(ctx, userSelect+` WHERE u.email = lower($1) AND u.deleted_at IS NULL`, ref)
	}
	var u userRow
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.StableID, &u.StableName, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return userRow{}, fmt.Errorf("no user %q (see: stallfunk-admin user list)", ref)
	}
	return u, err
}

// otherAdmins counts the admins of the stable except the given user.
func otherAdmins(ctx context.Context, q queryer, stableID, userID string) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(*) FROM users
		WHERE stable_id = $1 AND is_admin AND deleted_at IS NULL AND id <> $2`, stableID, userID).Scan(&n)
	return n, err
}

// validEmail checks and lower-cases an address the way the API does at sign-in.
func validEmail(in string) (string, error) {
	email, ok := auth.NormalizeEmail(in)
	if !ok {
		return "", usagef("%q is not a valid e-mail address (plain address, no display name)", in)
	}
	return email, nil
}
