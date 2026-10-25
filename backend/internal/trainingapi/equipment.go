package trainingapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// Stable equipment (docs/domains/training-planning.md, "Equipment and alternatives"): which
// facility equipment (poles, cavaletti, jumps, obstacles) each location of the stable has and
// whether the location is shared. The planner only plans exercises whose equipment is there.

type equipmentKeysOut struct {
	Facility []string `json:"facility"`
	Personal []string `json:"personal"`
}

// equipmentOut is the answer of GET and PUT /stables/training-equipment. Set is false while the
// stable has not saved its equipment; Locations always has every location (the defaults fill the
// missing ones).
type equipmentOut struct {
	Set       bool                                    `json:"set"`
	Locations map[training.Activity]training.Location `json:"locations"`
	Keys      equipmentKeysOut                        `json:"keys"`
	// CanEdit is true for admins.
	CanEdit bool `json:"can_edit"`
}

// cleanEquipment keeps the known keys of a list once each, in order.
func cleanEquipment(keys []string) []string {
	out := []string{}
	for _, k := range keys {
		if training.ValidEquipment(k) && !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// loadEquipment reads the equipment of the stable's locations as it is stored (a location the
// stable never saved is missing, StableEquipment.Of fills in its default) and whether the stable
// has saved it at all.
func (h *handler) loadEquipment(ctx context.Context, stableID string) (training.StableEquipment, bool, error) {
	var raw []byte
	var set bool
	err := h.deps.Pool.QueryRow(ctx, `SELECT training_equipment, training_equipment <> '{}'::jsonb FROM stables WHERE id = $1`, stableID).
		Scan(&raw, &set)
	if errors.Is(err, pgx.ErrNoRows) {
		return training.StableEquipment{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var stored map[string]training.Location
	_ = json.Unmarshal(raw, &stored)
	cfg := training.StableEquipment{}
	for name, loc := range stored {
		if a := training.Activity(name); training.IsLocation(a) {
			loc.Equipment = cleanEquipment(loc.Equipment)
			cfg[a] = loc
		}
	}
	return cfg, set, nil
}

func (h *handler) equipmentOut(ctx context.Context, stableID string, canEdit bool) (equipmentOut, error) {
	cfg, set, err := h.loadEquipment(ctx, stableID)
	if err != nil {
		return equipmentOut{}, err
	}
	out := equipmentOut{
		Set: set, Locations: map[training.Activity]training.Location{}, CanEdit: canEdit,
		Keys: equipmentKeysOut{Facility: slices.Clone(training.FacilityEquipment), Personal: slices.Clone(training.PersonalEquipment)},
	}
	for _, a := range training.Locations {
		out.Locations[a] = cfg.Of(a)
	}
	return out, nil
}

// getEquipment: GET /api/v1/stables/training-equipment (every member).
func (h *handler) getEquipment(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	out, err := h.equipmentOut(r.Context(), user.StableID, user.IsAdmin)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

type locationIn struct {
	// Equipment nil takes the location's default.
	Equipment *[]string `json:"equipment"`
	Shared    bool      `json:"shared"`
}

// putEquipment: PUT /api/v1/stables/training-equipment {"locations": {"hall": {"equipment":
// ["poles"], "shared": false}, ...}} (admin only). It replaces the stored equipment: a location
// that is not sent has its default again, {"locations": {}} resets all of them. An unknown
// location or key is a 400.
func (h *handler) putEquipment(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var in struct {
		Locations *map[string]locationIn `json:"locations"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Locations == nil {
		invalid(w, "locations is required")
		return
	}
	stored := map[string]training.Location{}
	for name, l := range *in.Locations {
		a := training.Activity(name)
		if !training.IsLocation(a) {
			invalid(w, fmt.Sprintf("unknown location %q (one of %v)", name, training.Locations))
			return
		}
		loc := training.Location{Shared: l.Shared, Equipment: slices.Clone(training.DefaultLocation(a).Equipment)}
		if l.Equipment != nil {
			loc.Equipment = []string{}
			for _, k := range *l.Equipment {
				if !training.ValidEquipment(k) {
					invalid(w, fmt.Sprintf("unknown equipment %q", k))
					return
				}
				if !slices.Contains(loc.Equipment, k) {
					loc.Equipment = append(loc.Equipment, k)
				}
			}
		}
		stored[name] = loc
	}
	raw, _ := json.Marshal(stored)
	if _, err := h.deps.Pool.Exec(r.Context(), `UPDATE stables SET training_equipment = $2 WHERE id = $1`, user.StableID, raw); err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.equipmentOut(r.Context(), user.StableID, true)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
