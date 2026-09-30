package requests

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// CompleteBlanketRequests marks the open or assigned requests of type "blanket" for the
// horse whose date range covers day (YYYY-MM-DD, the stable-local blanket day) as done and
// stores feedback (may be empty) as payload.feedback. It returns the ids of the requests
// it closed. The blankets package calls it in the transaction that records the day state
// (JAN-39); the caller publishes request.changed after commit.
func CompleteBlanketRequests(ctx context.Context, tx pgx.Tx, stableID, horseID, day, feedback string) ([]string, error) {
	rows, err := tx.Query(ctx, `
		UPDATE requests
		SET status = 'done',
		    payload = CASE WHEN $4 = '' THEN payload
		                   ELSE payload || jsonb_build_object('feedback', $4::text) END
		WHERE stable_id = $1 AND horse_id = $2 AND type = 'blanket'
		  AND status IN ('open', 'assigned')
		  AND $3::date BETWEEN date AND COALESCE(date_end, date)
		RETURNING id::text`, stableID, horseID, day, feedback)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
