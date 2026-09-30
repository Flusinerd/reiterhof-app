package blanketplan

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadRules loads the rules of one horse ordered by position. The stable id
// scopes the query, so a horse of another stable yields no rules.
func LoadRules(ctx context.Context, pool *pgxpool.Pool, stableID, horseID string) ([]Rule, error) {
	rows, err := pool.Query(ctx, `
		SELECT position, temp_min::float8, temp_max::float8, rain, blanket_id::text, COALESCE(note, '')
		FROM blanket_rules
		WHERE stable_id = $1 AND horse_id = $2
		ORDER BY position`, stableID, horseID)
	if err != nil {
		return nil, fmt.Errorf("blanketplan: query rules: %w", err)
	}
	defer rows.Close()
	var rules []Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.Position, &r.TempMin, &r.TempMax, &r.Rain, &r.BlanketID, &r.Note); err != nil {
			return nil, fmt.Errorf("blanketplan: scan rule: %w", err)
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("blanketplan: rules: %w", err)
	}
	return rules, nil
}
