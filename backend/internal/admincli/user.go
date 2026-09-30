package admincli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Flusinerd/reiterhof-app/backend/internal/privacy"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

func runUserList(ctx context.Context, e *Env, args []string) error {
	fs := e.flags("user list")
	stable := fs.String("stable", "", "only members of this stable (id or name)")
	if _, err := parse(fs, args, 0, 0); err != nil {
		return err
	}
	stableID, err := e.optionalStable(ctx, *stable)
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	rows, err := pool.Query(ctx, userSelect+` WHERE u.deleted_at IS NULL AND ($1 = '' OR u.stable_id::text = $1)
		ORDER BY s.name NULLS LAST, u.is_admin DESC, lower(u.name), u.created_at`, stableID)
	if err != nil {
		return err
	}
	users, err := pgx.CollectRows(rows, pgx.RowToStructByPos[userRow])
	if err != nil {
		return err
	}
	if e.JSON {
		if users == nil {
			users = []userRow{}
		}
		return e.writeJSON(users)
	}
	table := make([][]string, 0, len(users))
	for _, u := range users {
		table = append(table, []string{u.ID, u.Name, u.Email, u.stableLabel(), yesNo(u.IsAdmin), fmtTime(u.CreatedAt)})
	}
	writeTable(e.Out, []string{"ID", "NAME", "EMAIL", "STABLE", "ADMIN", "CREATED (UTC)"}, table)
	return nil
}

func runUserCreate(ctx context.Context, e *Env, args []string) error {
	fs := e.flags("user create")
	email := fs.String("email", "", "e-mail address (required, stored lower case)")
	name := fs.String("name", "", "display name (required)")
	stable := fs.String("stable", "", "stable id or name (default: the only stable)")
	admin := fs.Bool("admin", false, "make the user an admin of the stable")
	if _, err := parse(fs, args, 0, 0); err != nil {
		return err
	}
	if *email == "" || strings.TrimSpace(*name) == "" {
		return usagef("--email and --name are required")
	}
	addr, err := validEmail(*email)
	if err != nil {
		return err
	}
	displayName := strings.TrimSpace(*name)
	if utf8.RuneCountInString(displayName) > 100 {
		return usagef("--name is longer than 100 characters")
	}
	st, err := e.resolveStable(ctx, *stable)
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	var id string
	err = pool.QueryRow(ctx, `INSERT INTO users (stable_id, name, email, is_admin) VALUES ($1, $2, $3, $4) RETURNING id::text`,
		st.ID, displayName, addr, *admin).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation on users_email_key
		existing, ferr := findUser(ctx, pool, addr)
		if ferr == nil {
			return fmt.Errorf("a user with %s already exists (%s, stable %s); use user move / promote instead", addr, existing.ID, existing.stableLabel())
		}
		return fmt.Errorf("a user with %s already exists", addr)
	}
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	if e.JSON {
		return e.writeJSON(map[string]any{"id": id, "email": addr, "name": displayName, "stable_id": st.ID, "is_admin": *admin})
	}
	fmt.Fprintln(e.Out, id)
	role := "member"
	if *admin {
		role = "admin"
	}
	fmt.Fprintf(e.Err, "created %s %s <%s> in stable %q; they can sign in with a magic link to this address\n", role, displayName, addr, st.Name)
	return nil
}

func runUserPromote(ctx context.Context, e *Env, args []string) error {
	pos, err := parse(e.flags("user promote"), args, 1, 1)
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	u, err := findUser(ctx, pool, pos[0])
	if err != nil {
		return err
	}
	if u.StableID == nil {
		return fmt.Errorf("%s is not in a stable; move them into one first (user move)", u.label())
	}
	if u.IsAdmin {
		fmt.Fprintf(e.Out, "%s is already an admin of %s\n", u.label(), u.stableLabel())
		return nil
	}
	if _, err := pool.Exec(ctx, `UPDATE users SET is_admin = true WHERE id = $1`, u.ID); err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "%s is now an admin of %s\n", u.label(), u.stableLabel())
	return nil
}

func runUserDemote(ctx context.Context, e *Env, args []string) error {
	pos, err := parse(e.flags("user demote"), args, 1, 1)
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	u, err := findUser(ctx, pool, pos[0])
	if err != nil {
		return err
	}
	if !u.IsAdmin {
		fmt.Fprintf(e.Out, "%s is not an admin\n", u.label())
		return nil
	}
	// Check before asking; the check is repeated under lock below.
	if others, err := otherAdmins(ctx, pool, *u.StableID, u.ID); err != nil {
		return err
	} else if others == 0 {
		return fmt.Errorf("refusing to demote %s: they are the last admin of %s (promote someone else first)", u.label(), u.stableLabel())
	}
	if err := e.confirm(fmt.Sprintf("Remove admin rights of %s in stable %s?", u.label(), u.stableLabel())); err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Lock the admins of the stable so that two demotions cannot remove the last two.
	if _, err := tx.Exec(ctx, `SELECT id FROM users WHERE stable_id = $1 AND is_admin FOR UPDATE`, *u.StableID); err != nil {
		return err
	}
	others, err := otherAdmins(ctx, tx, *u.StableID, u.ID)
	if err != nil {
		return err
	}
	if others == 0 {
		return fmt.Errorf("refusing to demote %s: they are the last admin of %s (promote someone else first)", u.label(), u.stableLabel())
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET is_admin = false WHERE id = $1`, u.ID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "%s is no longer an admin of %s\n", u.label(), u.stableLabel())
	return nil
}

// runUserMove puts a member into another stable. A user without a stable (signed in but
// never joined) is simply assigned. A user who belongs to a stable loses everything that is
// scoped to it (rider roles, presence, reminders, helper assignments, open requests) and
// the admin flag; moving is refused while they own horses or are the last admin.
func runUserMove(ctx context.Context, e *Env, args []string) error {
	fs := e.flags("user move")
	stable := fs.String("stable", "", "target stable id or name (default: the only stable)")
	pos, err := parse(fs, args, 1, 1)
	if err != nil {
		return err
	}
	target, err := e.resolveStable(ctx, *stable)
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	u, err := findUser(ctx, pool, pos[0])
	if err != nil {
		return err
	}
	if u.StableID != nil && *u.StableID == target.ID {
		fmt.Fprintf(e.Out, "%s is already in %s\n", u.label(), target.Name)
		return nil
	}
	if u.StableID != nil {
		if err := e.confirm(fmt.Sprintf("Move %s from %s to %s? Their rider roles, presence, reminders and open requests in %s are removed.",
			u.label(), u.stableLabel(), target.Name, u.stableLabel())); err != nil {
			return err
		}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Re-read under lock: a concurrent change must not slip past the checks.
	if _, err := tx.Exec(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, u.ID); err != nil {
		return err
	}
	if u.StableID != nil {
		if err := detachFromStable(ctx, tx, u); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET stable_id = $2, is_admin = false WHERE id = $1`, u.ID, target.ID); err != nil {
		return err
	}
	if u.StableID != nil {
		if err := realtime.Publish(ctx, tx, *u.StableID, "presence.changed", nil); err != nil {
			return err
		}
	}
	if err := realtime.Publish(ctx, tx, target.ID, "presence.changed", nil); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "moved %s to %s\n", u.label(), target.Name)
	return nil
}

// detachFromStable refuses the move for owners and last admins and removes the user's
// stable-scoped rows (mirrors what privacy.DeleteAccount does for them).
func detachFromStable(ctx context.Context, tx pgx.Tx, u userRow) error {
	rows, err := tx.Query(ctx, `SELECT name FROM horses WHERE owner_id = $1 ORDER BY name`, u.ID)
	if err != nil {
		return err
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if len(names) > 0 {
		return fmt.Errorf("refusing to move %s: they own horses (%s); transfer them first: stallfunk-admin horse transfer NAME --to EMAIL",
			u.label(), strings.Join(names, ", "))
	}
	if u.IsAdmin {
		others, err := otherAdmins(ctx, tx, *u.StableID, u.ID)
		if err != nil {
			return err
		}
		if others == 0 {
			return fmt.Errorf("refusing to move %s: they are the last admin of %s (promote someone else first)", u.label(), u.stableLabel())
		}
	}
	statements := []string{
		// Requests the person helped with go back to "open" if nobody else is left.
		`UPDATE requests SET status = 'open' WHERE status = 'assigned'
			AND id IN (SELECT request_id FROM request_assignees WHERE user_id = $1)
			AND NOT EXISTS (SELECT 1 FROM request_assignees a WHERE a.request_id = requests.id AND a.user_id <> $1)`,
		`DELETE FROM request_assignees WHERE user_id = $1`,
		`UPDATE requests SET status = 'cancelled' WHERE created_by = $1 AND status IN ('open', 'assigned')`,
		`DELETE FROM horse_riders WHERE user_id = $1`,
		`DELETE FROM presence WHERE user_id = $1`,
		`DELETE FROM reminders WHERE user_id = $1`,
		`DELETE FROM reminder_settings WHERE user_id = $1`,
	}
	for _, s := range statements {
		if _, err := tx.Exec(ctx, s, u.ID); err != nil {
			return err
		}
	}
	return nil
}

func runUserDelete(ctx context.Context, e *Env, args []string) error {
	pos, err := parse(e.flags("user delete"), args, 1, 1)
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	u, err := findUser(ctx, pool, pos[0])
	if err != nil {
		return err
	}
	if err := e.confirm(fmt.Sprintf("Delete the account of %s (stable %s)? The account is anonymised and cannot be restored.", u.label(), u.stableLabel())); err != nil {
		return err
	}
	err = privacy.DeleteAccount(ctx, pool, u.ID, e.now())
	var owns *privacy.OwnsHorsesError
	switch {
	case errors.As(err, &owns):
		var b strings.Builder
		fmt.Fprintf(&b, "cannot delete %s: owns_horses. Transfer or remove these horses first (stallfunk-admin horse transfer ID --to EMAIL):", u.label())
		for _, h := range owns.Horses {
			fmt.Fprintf(&b, "\n  %s  %s", h.ID, h.Name)
		}
		return errors.New(b.String())
	case errors.Is(err, privacy.ErrLastAdmin):
		return fmt.Errorf("cannot delete %s: last_admin. They are the only admin of %s; promote someone else first (stallfunk-admin user promote EMAIL)", u.label(), u.stableLabel())
	case errors.Is(err, privacy.ErrNotFound):
		return fmt.Errorf("user %s does not exist or was already deleted", u.label())
	case err != nil:
		return fmt.Errorf("delete account: %w", err)
	}
	fmt.Fprintf(e.Out, "deleted account of %s (%s)\n", u.label(), u.ID)
	return nil
}
