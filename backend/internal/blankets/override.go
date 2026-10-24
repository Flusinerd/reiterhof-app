package blankets

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// maxOverrideNights limits a manual choice with an end; longer ones are "until reset" (nights null).
const maxOverrideNights = 30

// Override is a manual blanket choice of the owner or an admin that replaces the rule
// recommendation from DayFrom on, until DayUntil (inclusive) or, without it, until reset.
type Override struct {
	// BlanketID is nil for "no blanket".
	BlanketID *string   `json:"blanket_id"`
	Blanket   *Blanket  `json:"blanket"`
	DayFrom   string    `json:"day_from"`
	DayUntil  *string   `json:"day_until"`
	Note      string    `json:"note"`
	SetBy     *Person   `json:"set_by"`
	SetAt     time.Time `json:"set_at"`
}

// applyOverride turns the rule recommendation into the effective one. Without an override the
// rule answer is returned unchanged. With one the result is "blanket" or "none" with Manual
// set, no rule index and the note of the override.
func applyOverride(rule Recommendation, o *Override) Recommendation {
	if o == nil {
		return rule
	}
	rec := Recommendation{Status: StatusNone, Note: o.Note, Manual: true}
	if o.Blanket != nil {
		b := *o.Blanket
		rec.Status, rec.Blanket = StatusBlanket, &b
	}
	return rec
}

// overrideSQL selects the override columns; $1 is the stable, $2 the horse or NULL, $3 the day.
// Only overrides that are active on the day are returned.
const overrideSQL = `
	SELECT o.horse_id::text, o.blanket_id::text, to_char(o.day_from, 'YYYY-MM-DD'), to_char(o.day_until, 'YYYY-MM-DD'),
	       COALESCE(o.note, ''), o.set_by::text, u.name, o.set_at
	FROM blanket_overrides o LEFT JOIN users u ON u.id = o.set_by
	WHERE o.stable_id = $1 AND ($2::uuid IS NULL OR o.horse_id = $2::uuid)
	  AND o.day_from <= $3::date AND (o.day_until IS NULL OR o.day_until >= $3::date)`

func scanOverride(row pgx.Row) (horseID string, o Override, err error) {
	var by, byName *string
	if err := row.Scan(&horseID, &o.BlanketID, &o.DayFrom, &o.DayUntil, &o.Note, &by, &byName, &o.SetAt); err != nil {
		return "", Override{}, err
	}
	if by != nil {
		name := ""
		if byName != nil {
			name = *byName
		}
		o.SetBy = &Person{ID: *by, Name: name}
	}
	return horseID, o, nil
}

// resolveBlanket fills o.Blanket from the horse's blankets.
func (o *Override) resolveBlanket(blankets []Blanket) {
	if o.BlanketID == nil {
		return
	}
	for i := range blankets {
		if blankets[i].ID == *o.BlanketID {
			b := blankets[i]
			o.Blanket = &b
		}
	}
}

type overrideInput struct {
	// BlanketID is required as a key: null means "no blanket".
	BlanketID *string `json:"blanket_id"`
	// Nights is 1..30 starting with tonight; null or absent means until reset.
	Nights *int    `json:"nights"`
	Note   *string `json:"note"`
}

// putOverride handles PUT /horses/{id}/blanket-override (owner or admin): the manual blanket
// choice from tonight's blanket day on. It replaces an earlier choice of the horse.
func (h *handler) putOverride(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, true)
	if !ok {
		return
	}
	var in overrideInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Nights != nil && (*in.Nights < 1 || *in.Nights > maxOverrideNights) {
		invalid(w, "nights must be between 1 and 30 (null = until reset)")
		return
	}
	note, okNote := text(in.Note, maxNoteLen)
	if !okNote {
		invalid(w, "note is too long")
		return
	}
	if in.BlanketID != nil {
		var exists bool
		// Compared as text, so a malformed id is simply "not a blanket of this horse".
		err := h.deps.Pool.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM blankets WHERE id::text = $1 AND horse_id = $2 AND stable_id = $3)`,
			*in.BlanketID, horseID, user.StableID).Scan(&exists)
		if err != nil {
			h.internal(w, "blanket lookup", err)
			return
		}
		if !exists {
			invalid(w, "blanket_id is not a blanket of this horse")
			return
		}
	}
	loc, _, err := h.svc.stableInfo(r.Context(), h.deps.Pool, user.StableID)
	if err != nil {
		h.internal(w, "stable", err)
		return
	}
	day := NightDay(h.svc.now(), loc)
	var until *string
	if in.Nights != nil {
		d, _ := time.Parse("2006-01-02", day)
		u := d.AddDate(0, 0, *in.Nights-1).Format("2006-01-02")
		until = &u
	}
	_, err = h.deps.Pool.Exec(r.Context(), `
		INSERT INTO blanket_overrides (horse_id, stable_id, blanket_id, day_from, day_until, note, set_by, set_at)
		VALUES ($1, $2, $3::uuid, $4::date, $5::date, NULLIF($6, ''), $7, $8)
		ON CONFLICT (horse_id) DO UPDATE SET blanket_id = EXCLUDED.blanket_id, day_from = EXCLUDED.day_from,
			day_until = EXCLUDED.day_until, note = EXCLUDED.note, set_by = EXCLUDED.set_by, set_at = EXCLUDED.set_at`,
		horseID, user.StableID, in.BlanketID, day, until, note, user.ID, h.svc.now())
	if err != nil {
		h.internal(w, "save override", err)
		return
	}
	n, err := h.svc.loadNight(r.Context(), user.StableID, horseID)
	if err != nil || len(n.Horses) != 1 || n.Horses[0].Override == nil {
		h.internal(w, "load override", errors.Join(err, errors.New("override not loaded")))
		return
	}
	h.svc.hint(r.Context(), user.StableID, EventPlanChanged, map[string]any{"horse_id": horseID})
	hn := n.Horses[0]
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"override": hn.Override, "recommendation": hn.Rec,
		"rule_recommendation": hn.RuleRec})
}

// deleteOverride handles DELETE /horses/{id}/blanket-override (owner or admin): back to the
// rules. Idempotent, 204.
func (h *handler) deleteOverride(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, true)
	if !ok {
		return
	}
	tag, err := h.deps.Pool.Exec(r.Context(), `DELETE FROM blanket_overrides WHERE horse_id = $1 AND stable_id = $2`,
		horseID, user.StableID)
	if err != nil {
		h.internal(w, "delete override", err)
		return
	}
	if tag.RowsAffected() > 0 {
		h.svc.hint(r.Context(), user.StableID, EventPlanChanged, map[string]any{"horse_id": horseID})
	}
	w.WriteHeader(http.StatusNoContent)
}

// loadOverrides reads the overrides active on day, keyed by horse.
func (s *Service) loadOverrides(ctx context.Context, stableID string, only *string, day string) (map[string]Override, error) {
	rows, err := s.Pool.Query(ctx, overrideSQL, stableID, only, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Override{}
	for rows.Next() {
		hid, o, err := scanOverride(rows)
		if err != nil {
			return nil, err
		}
		out[hid] = o
	}
	return out, rows.Err()
}
