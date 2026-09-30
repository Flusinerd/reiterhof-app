package blankets

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

type ruleInput struct {
	TempMin   *float64 `json:"temp_min"`
	TempMax   *float64 `json:"temp_max"`
	Rain      *bool    `json:"rain"`
	BlanketID *string  `json:"blanket_id"`
	Note      *string  `json:"note"`
}

type rulesInput struct {
	Rules []ruleInput `json:"rules"`
}

func loadRules(ctx context.Context, q querier, stableID, horseID string) ([]Rule, error) {
	rows, err := q.Query(ctx, `
		SELECT id::text, position, temp_min::float8, temp_max::float8, rain, blanket_id::text, COALESCE(note, '')
		FROM blanket_rules WHERE stable_id = $1 AND horse_id = $2 ORDER BY position`, stableID, horseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Rule{}
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.Position, &r.TempMin, &r.TempMax, &r.Rain, &r.BlanketID, &r.Note); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (h *handler) getRules(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, false)
	if !ok {
		return
	}
	rules, err := loadRules(r.Context(), h.deps.Pool, user.StableID, horseID)
	if err != nil {
		h.internal(w, "load rules", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

// putRules replaces the whole rule list. The order of the array is the priority (the
// first matching rule wins); positions are stored 1-based.
func (h *handler) putRules(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, true)
	if !ok {
		return
	}
	var in rulesInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Rules == nil {
		invalid(w, "rules is required (use [] to remove all rules)")
		return
	}
	if len(in.Rules) > maxRules {
		invalid(w, fmt.Sprintf("at most %d rules", maxRules))
		return
	}

	own := map[string]bool{}
	rows, err := h.deps.Pool.Query(r.Context(), `SELECT id::text FROM blankets WHERE stable_id = $1 AND horse_id = $2`, user.StableID, horseID)
	if err != nil {
		h.internal(w, "load blankets", err)
		return
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			h.internal(w, "scan blanket", err)
			return
		}
		own[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		h.internal(w, "load blankets", err)
		return
	}

	for i, rule := range in.Rules {
		if !validTemp(rule.TempMin) || !validTemp(rule.TempMax) {
			invalid(w, fmt.Sprintf("rules[%d]: temperatures must be between -60 and 60", i))
			return
		}
		if rule.TempMin != nil && rule.TempMax != nil && *rule.TempMin >= *rule.TempMax {
			invalid(w, fmt.Sprintf("rules[%d]: temp_min must be lower than temp_max", i))
			return
		}
		if _, ok := text(rule.Note, maxNoteLen); !ok {
			invalid(w, fmt.Sprintf("rules[%d]: note is too long (%d characters)", i, maxNoteLen))
			return
		}
		if rule.BlanketID != nil && !own[*rule.BlanketID] {
			invalid(w, fmt.Sprintf("rules[%d]: blanket_id is not a blanket of this horse", i))
			return
		}
	}

	var saved []Rule
	err = h.svc.tx(r.Context(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `DELETE FROM blanket_rules WHERE stable_id = $1 AND horse_id = $2`, user.StableID, horseID); err != nil {
			return err
		}
		for i, rule := range in.Rules {
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO blanket_rules (stable_id, horse_id, position, temp_min, temp_max, rain, blanket_id, note)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				user.StableID, horseID, i+1, rule.TempMin, rule.TempMax, rule.Rain, rule.BlanketID, nullable(rule.Note)); err != nil {
				return err
			}
		}
		var err error
		saved, err = loadRules(r.Context(), tx, user.StableID, horseID)
		if err != nil {
			return err
		}
		return realtime.Publish(r.Context(), tx, user.StableID, EventPlanChanged, map[string]any{"horse_id": horseID})
	})
	if err != nil {
		h.internal(w, "replace rules", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"rules": saved})
}

// tx runs fn in a transaction that commits when fn returns nil.
func (s *Service) tx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
