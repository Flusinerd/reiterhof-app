package admincli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type horseRow struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Box       *string `json:"box"`
	OwnerID   *string `json:"owner_id"`
	OwnerName *string `json:"owner_name"`
	StableID  string  `json:"stable_id"`
	Stable    string  `json:"stable"`
}

const horseSelect = `SELECT h.id::text, h.name, h.box, h.owner_id::text, o.name, h.stable_id::text, s.name
	FROM horses h JOIN stables s ON s.id = h.stable_id LEFT JOIN users o ON o.id = h.owner_id`

func runHorseList(ctx context.Context, e *Env, args []string) error {
	fs := e.flags("horse list")
	stable := fs.String("stable", "", "only horses of this stable (id or name)")
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
	rows, err := pool.Query(ctx, horseSelect+` WHERE $1 = '' OR h.stable_id::text = $1 ORDER BY s.name, lower(h.name)`, stableID)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[horseRow])
	if err != nil {
		return err
	}
	if e.JSON {
		if list == nil {
			list = []horseRow{}
		}
		return e.writeJSON(list)
	}
	table := make([][]string, 0, len(list))
	for _, h := range list {
		table = append(table, []string{h.ID, h.Name, orDash(deref(h.Box)), orDash(deref(h.OwnerName)), h.Stable})
	}
	writeTable(e.Out, []string{"ID", "NAME", "BOX", "OWNER", "STABLE"}, table)
	return nil
}

// findHorse looks a horse up by UUID or by name (case-insensitive, optionally within one stable).
func findHorse(ctx context.Context, q queryer, ref, stableID string) (horseRow, error) {
	ref = strings.TrimSpace(ref)
	var rows pgx.Rows
	var err error
	if isUUID(ref) {
		rows, err = q.Query(ctx, horseSelect+` WHERE h.id = $1`, ref)
	} else {
		rows, err = q.Query(ctx, horseSelect+` WHERE lower(h.name) = lower($1) AND ($2 = '' OR h.stable_id::text = $2) ORDER BY s.name`, ref, stableID)
	}
	if err != nil {
		return horseRow{}, err
	}
	found, err := pgx.CollectRows(rows, pgx.RowToStructByPos[horseRow])
	if err != nil {
		return horseRow{}, err
	}
	switch len(found) {
	case 0:
		return horseRow{}, fmt.Errorf("no horse %q (see: stallfunk-admin horse list)", ref)
	case 1:
		return found[0], nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "several horses are named %q, use the id:", ref)
	for _, h := range found {
		fmt.Fprintf(&b, "\n  %s  %s (stable %s, owner %s)", h.ID, h.Name, h.Stable, orDash(deref(h.OwnerName)))
	}
	return horseRow{}, errors.New(b.String())
}

func runHorseTransfer(ctx context.Context, e *Env, args []string) error {
	fs := e.flags("horse transfer")
	to := fs.String("to", "", "new owner (e-mail or id, must be in the horse's stable)")
	stable := fs.String("stable", "", "narrow a horse name down to this stable (id or name)")
	pos, err := parse(fs, args, 1, 1)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*to) == "" {
		return usagef("--to is required (e-mail or id of the new owner)")
	}
	stableID, err := e.optionalStable(ctx, *stable)
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	h, err := findHorse(ctx, pool, pos[0], stableID)
	if err != nil {
		return err
	}
	newOwner, err := findUser(ctx, pool, *to)
	if err != nil {
		return err
	}
	if newOwner.StableID == nil || *newOwner.StableID != h.StableID {
		return fmt.Errorf("%s is in stable %s, but %s belongs to %s: horses can only be transferred within a stable (user move first)",
			newOwner.label(), newOwner.stableLabel(), h.Name, h.Stable)
	}
	if h.OwnerID != nil && *h.OwnerID == newOwner.ID {
		fmt.Fprintf(e.Out, "%s already belongs to %s\n", h.Name, newOwner.label())
		return nil
	}
	if err := e.confirm(fmt.Sprintf("Transfer horse %s from %s to %s?", h.Name, orDash(deref(h.OwnerName)), newOwner.label())); err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, `UPDATE horses SET owner_id = $2 WHERE id = $1 AND stable_id = $3`, h.ID, newOwner.ID, h.StableID)
	if err != nil {
		return fmt.Errorf("transfer horse: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return errors.New("transfer horse: the horse disappeared")
	}
	fmt.Fprintf(e.Out, "%s now belongs to %s\n", h.Name, newOwner.label())
	return nil
}
