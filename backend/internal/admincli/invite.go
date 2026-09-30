package admincli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
)

// Same bounds as POST /api/v1/stables/invites.
const (
	defaultInviteDays = 7
	defaultInviteUses = 10
)

func runInviteCreate(ctx context.Context, e *Env, args []string) error {
	fs := e.flags("invite create")
	stable := fs.String("stable", "", "stable id or name (default: the only stable)")
	days := fs.Int("days", defaultInviteDays, "validity in days (1..90)")
	uses := fs.Int("max-uses", defaultInviteUses, "how often the code can be redeemed (1..100)")
	if _, err := parse(fs, args, 0, 0); err != nil {
		return err
	}
	if *days < 1 || *days > 90 || *uses < 1 || *uses > 100 {
		return usagef("--days must be 1..90 and --max-uses 1..100")
	}
	st, err := e.resolveStable(ctx, *stable)
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	expires := e.now().Add(time.Duration(*days) * 24 * time.Hour)
	code, err := auth.CreateInvite(ctx, pool, st.ID, nil, expires, *uses)
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	if e.JSON {
		return e.writeJSON(map[string]any{
			"code": auth.FormatInviteCode(code), "stable_id": st.ID, "expires_at": expires, "max_uses": *uses,
		})
	}
	fmt.Fprintln(e.Out, auth.FormatInviteCode(code))
	fmt.Fprintf(e.Err, "invite for stable %q, valid until %s UTC, %d uses\n", st.Name, fmtTime(expires), *uses)
	return nil
}

type inviteRow struct {
	Code      string     `json:"code"`
	StableID  string     `json:"stable_id"`
	Stable    string     `json:"stable"`
	CreatedBy *string    `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at"`
	Uses      int        `json:"uses"`
	MaxUses   *int       `json:"max_uses"`
	CreatedAt time.Time  `json:"created_at"`
}

func (i inviteRow) status(now time.Time) string {
	switch {
	case i.ExpiresAt != nil && !i.ExpiresAt.After(now):
		return "expired"
	case i.MaxUses != nil && i.Uses >= *i.MaxUses:
		return "used up"
	}
	return "active"
}

func runInviteList(ctx context.Context, e *Env, args []string) error {
	fs := e.flags("invite list")
	stable := fs.String("stable", "", "only invites of this stable (id or name)")
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
	rows, err := pool.Query(ctx, `SELECT i.code, i.stable_id::text, s.name, u.name, i.expires_at, i.uses, i.max_uses, i.created_at
		FROM stable_invites i JOIN stables s ON s.id = i.stable_id LEFT JOIN users u ON u.id = i.created_by
		WHERE $1 = '' OR i.stable_id::text = $1
		ORDER BY i.created_at DESC`, stableID)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[inviteRow])
	if err != nil {
		return err
	}
	now := e.now()
	if e.JSON {
		type out struct {
			inviteRow
			Code   string `json:"code"`
			Status string `json:"status"`
		}
		res := make([]out, 0, len(list))
		for _, i := range list {
			res = append(res, out{inviteRow: i, Code: auth.FormatInviteCode(i.Code), Status: i.status(now)})
		}
		return e.writeJSON(res)
	}
	table := make([][]string, 0, len(list))
	for _, i := range list {
		maxUses := "unlimited"
		if i.MaxUses != nil {
			maxUses = strconv.Itoa(*i.MaxUses)
		}
		table = append(table, []string{auth.FormatInviteCode(i.Code), i.Stable, orDash(deref(i.CreatedBy)),
			fmtTimePtr(i.ExpiresAt), fmt.Sprintf("%d/%s", i.Uses, maxUses), i.status(now)})
	}
	writeTable(e.Out, []string{"CODE", "STABLE", "CREATED BY", "EXPIRES (UTC)", "USES", "STATUS"}, table)
	return nil
}
