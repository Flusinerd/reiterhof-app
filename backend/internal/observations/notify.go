package observations

import (
	"context"
	"sort"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
)

// categoryText is the German wording used in push texts.
var categoryText = map[string]string{
	"cough": "Husten", "lameness": "Lahmheit", "injury": "Verletzung", "not_eating": "Frisst nicht",
	"colic": "Kolik", "behavior": "Verhaltensänderung", "blanket_equipment": "Decke/Ausrüstung", "other": "Auffälligkeit",
}

// Recipients returns the users to notify about a new observation, without the reporter:
// always the owner and the riders of the horse; for an urgent report additionally everybody
// with an open presence visit, whatever their presence visibility (see
// docs/domains/observations.md for why that is fine). Sorted, without duplicates.
func Recipients(ctx context.Context, q queryer, stableID, horseID, reporterID string, urgent bool) ([]string, error) {
	set := map[string]bool{}
	add := func(sql string, args ...any) error {
		rows, err := q.Query(ctx, sql, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id *string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			if id != nil && *id != reporterID {
				set[*id] = true
			}
		}
		return rows.Err()
	}
	if err := add(`SELECT owner_id::text FROM horses WHERE id = $1 AND stable_id = $2`, horseID, stableID); err != nil {
		return nil, err
	}
	if err := add(`SELECT user_id::text FROM horse_riders WHERE horse_id = $1 AND stable_id = $2`, horseID, stableID); err != nil {
		return nil, err
	}
	if urgent {
		if err := add(`SELECT user_id::text FROM presence WHERE stable_id = $1 AND left_at IS NULL`, stableID); err != nil {
			return nil, err
		}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// notify pushes the new observation. Failures are logged; the report itself is already saved.
func (h *handler) notify(ctx context.Context, reporter auth.User, o Observation) {
	urgent := o.Urgency == UrgencyUrgent
	users, err := Recipients(ctx, h.deps.Pool, reporter.StableID, o.HorseID, reporter.ID, urgent)
	if err != nil {
		h.deps.Log.Error("observations: recipients", "err", err)
		return
	}
	if len(users) == 0 {
		return
	}
	what := "Auffälligkeit"
	if o.Category != nil {
		what = categoryText[*o.Category]
	}
	kind, title, body := push.KindObservation, "Auffälligkeit: "+o.HorseName, reporter.Name+" meldet: "+what+"."
	switch {
	case urgent:
		kind, title = push.KindUrgentObservation, "Dringend: "+o.HorseName
		body = reporter.Name + " meldet: " + what + ". Bitte sofort nach dem Pferd sehen."
	case o.Urgency == UrgencyCheck:
		body = reporter.Name + " meldet: " + what + ". Bitte ansehen."
	}
	data := map[string]any{"observation_id": o.ID, "horse_id": o.HorseID, "urgency": o.Urgency, "screen": "/observations/" + o.ID}
	if err := h.deps.Notify.NotifyUsers(context.WithoutCancel(ctx), reporter.StableID, users, kind, title, body, data); err != nil {
		h.deps.Log.Error("observations: notify", "err", err)
	}
}
