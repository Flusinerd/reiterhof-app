package trainingapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// focusOut is an entry of the focus catalog (GET /training/focuses).
type focusOut struct {
	Key         string              `json:"key"`
	Label       string              `json:"label"`
	Group       string              `json:"group"`
	Phase       int                 `json:"phase"`
	Description string              `json:"description"`
	Signs       string              `json:"signs"`
	Fits        []training.Activity `json:"fits"`
}

// listFocuses: GET /api/v1/training/focuses, the static focus catalog.
func (h *handler) listFocuses(w http.ResponseWriter, _ *http.Request) {
	out := make([]focusOut, 0, len(training.Focuses))
	for _, f := range training.Focuses {
		out = append(out, focusOut{
			Key: f.Key, Label: f.Label, Group: f.Group, Phase: f.Phase, Description: f.Description, Signs: f.Signs,
			Fits: slices.Clone(f.Fits),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"focuses": out})
}

// focusStored is a focus of the horse as stored in training_profiles.focuses.
type focusStored struct {
	Key   string `json:"key"`
	Until string `json:"until,omitempty"` // YYYY-MM-DD
}

// focusChoiceOut is a focus of the horse in the profile response: until is null for none.
type focusChoiceOut struct {
	Key   string  `json:"key"`
	Until *string `json:"until"`
}

func focusChoicesOut(list []focusStored) []focusChoiceOut {
	out := make([]focusChoiceOut, 0, len(list))
	for _, f := range list {
		c := focusChoiceOut{Key: f.Key}
		if f.Until != "" {
			until := f.Until
			c.Until = &until
		}
		out = append(out, c)
	}
	return out
}

// warningOut is a hint to the owner about the profile: key is a focus key or "level_up".
type warningOut struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

// focusKeys validates a list of focus keys: at most max, known and distinct.
func focusKeys(keys []string, max int) error {
	if len(keys) > max {
		return fmt.Errorf("at most %d focuses", max)
	}
	for i, k := range keys {
		if !training.ValidFocusKey(k) {
			return fmt.Errorf("unknown focus %q", k)
		}
		if slices.Contains(keys[:i], k) {
			return fmt.Errorf("focus %q is listed twice", k)
		}
	}
	return nil
}

// --- my focus -----------------------------------------------------------------------------

// myFocusOut is the answer of /horses/{id}/my-focus. CatalogPhase is the highest phase of the
// focus catalog the horse's phase allows (1, 2 or 3; a focus with a higher phase gets a warning
// in the app), Phase the horse's phase itself.
type myFocusOut struct {
	Keys         []string            `json:"keys"`
	CatalogPhase int                 `json:"catalog_phase"`
	Phase        training.HorsePhase `json:"phase"`
}

// authorizeFocus resolves the horse and checks that the caller may keep an own focus: the owner,
// admins and riders with take_week_slots or log_sessions.
func (h *handler) authorizeFocus(w http.ResponseWriter, r *http.Request) (access, bool) {
	a, ok := h.authorize(w, r)
	if !ok {
		return a, false
	}
	if !a.canLog() && !a.canTakeSlots() {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "you may not set a focus for this horse")
		return a, false
	}
	return a, true
}

func (h *handler) myFocusOut(ctx context.Context, a access) (myFocusOut, error) {
	out := myFocusOut{Keys: []string{}}
	var keys []string
	err := h.deps.Pool.QueryRow(ctx, `SELECT keys FROM rider_focuses WHERE horse_id = $1 AND user_id = $2`,
		a.horseID, a.user.ID).Scan(&keys)
	if err != nil && !isNoRows(err) {
		return out, err
	}
	out.Keys = nonNil(keys)
	phase, err := h.horsePhase(ctx, a)
	if err != nil {
		return out, err
	}
	out.Phase, out.CatalogPhase = phase, phase.Rank()
	return out, nil
}

// getMyFocus: GET /api/v1/horses/{id}/my-focus.
func (h *handler) getMyFocus(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorizeFocus(w, r)
	if !ok {
		return
	}
	out, err := h.myFocusOut(r.Context(), a)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// putMyFocus: PUT /api/v1/horses/{id}/my-focus {"keys": [...]}, at most two known, distinct keys;
// an empty list clears the focus.
func (h *handler) putMyFocus(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorizeFocus(w, r)
	if !ok {
		return
	}
	var in struct {
		Keys []string `json:"keys"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if err := focusKeys(in.Keys, training.MaxRiderFocuses); err != nil {
		invalid(w, "keys: "+err.Error())
		return
	}
	var err error
	if len(in.Keys) == 0 {
		_, err = h.deps.Pool.Exec(r.Context(), `DELETE FROM rider_focuses WHERE horse_id = $1 AND user_id = $2`, a.horseID, a.user.ID)
	} else {
		_, err = h.deps.Pool.Exec(r.Context(), `
			INSERT INTO rider_focuses (stable_id, horse_id, user_id, keys, updated_at) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (horse_id, user_id) DO UPDATE SET keys = EXCLUDED.keys, updated_at = EXCLUDED.updated_at`,
			a.user.StableID, a.horseID, a.user.ID, in.Keys, h.deps.Now())
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.myFocusOut(r.Context(), a)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// deleteMyFocus: DELETE /api/v1/horses/{id}/my-focus removes the own focus (204, also when none
// was set).
func (h *handler) deleteMyFocus(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorizeFocus(w, r)
	if !ok {
		return
	}
	if _, err := h.deps.Pool.Exec(r.Context(), `DELETE FROM rider_focuses WHERE horse_id = $1 AND user_id = $2`, a.horseID, a.user.ID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// riderFocuses returns the own focuses of every person of the horse that set some, by user id.
func (h *handler) riderFocuses(ctx context.Context, horseID string) (map[string][]string, error) {
	rows, err := h.deps.Pool.Query(ctx, `SELECT user_id::text, keys FROM rider_focuses WHERE horse_id = $1`, horseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var id string
		var keys []string
		if err := rows.Scan(&id, &keys); err != nil {
			return nil, err
		}
		// Only keys the catalog still knows count.
		var known []string
		for _, k := range keys {
			if training.ValidFocusKey(k) {
				known = append(known, k)
			}
		}
		if len(known) > 0 {
			out[id] = known
		}
	}
	return out, rows.Err()
}

// --- horse phase and age -----------------------------------------------------------------------

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// ageOf is the age of a horse born in birthYear, 0 when unknown or not plausible.
func ageOf(birthYear *int, today time.Time) int {
	if birthYear != nil {
		if age := today.Year() - *birthYear; age >= 0 && age <= 45 {
			return age
		}
	}
	return 0
}

// horseAge reads the horse's age in years (0 = unknown).
func (h *handler) horseAge(ctx context.Context, horseID string, today time.Time) (int, error) {
	var birthYear *int
	if err := h.deps.Pool.QueryRow(ctx, `SELECT birth_year FROM horses WHERE id = $1`, horseID).Scan(&birthYear); err != nil {
		return 0, err
	}
	return ageOf(birthYear, today), nil
}

// horsePhase is the phase of the horse's training: its level class, age and status decide.
func (h *handler) horsePhase(ctx context.Context, a access) (training.HorsePhase, error) {
	p, err := h.loadProfile(ctx, a.user.StableID, a.horseID)
	if err != nil {
		return "", err
	}
	today := training.Day(h.deps.Now().In(h.location(ctx, a.user.StableID)))
	age, err := h.horseAge(ctx, a.horseID, today)
	if err != nil {
		return "", err
	}
	return training.Phase(training.LevelClass(p.level), age, p.status), nil
}

// parseFocusJSON decodes training_profiles.focuses; entries with an unknown key are dropped (the
// catalog may have lost a key since it was saved).
func parseFocusJSON(raw []byte) []focusStored {
	var list []focusStored
	_ = json.Unmarshal(raw, &list)
	out := list[:0]
	for _, f := range list {
		if training.ValidFocusKey(f.Key) {
			if len(f.Until) > 10 {
				f.Until = f.Until[:10]
			}
			out = append(out, f)
		}
	}
	return out
}
