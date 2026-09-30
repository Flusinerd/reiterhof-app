package trainingapi

import (
	"net/http"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

type horseOut struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ColorKey string `json:"color_key"`
	// Role is owner or rider (owner wins if both apply).
	Role string `json:"role"`
	// ProfileStatus is fit, reha or pause; empty when the horse has no training profile yet.
	ProfileStatus string `json:"profile_status"`
	HasProfile    bool   `json:"has_profile"`
}

// listHorses: GET /api/v1/training/horses, the horse switcher of the training tab: the
// horses the user owns or rides, with the profile status.
func (h *handler) listHorses(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	rows, err := h.deps.Pool.Query(r.Context(), `
		SELECT hs.id::text, hs.name, COALESCE(hs.color_key, ''),
		       CASE WHEN hs.owner_id = $2 THEN 'owner' ELSE 'rider' END,
		       COALESCE(tp.status, '')
		FROM horses hs
		LEFT JOIN training_profiles tp ON tp.horse_id = hs.id
		WHERE hs.stable_id = $1
		  AND (hs.owner_id = $2 OR EXISTS (SELECT 1 FROM horse_riders hr WHERE hr.horse_id = hs.id AND hr.user_id = $2))
		ORDER BY (hs.owner_id = $2) DESC, hs.name, hs.id`, user.StableID, user.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer rows.Close()
	out := []horseOut{}
	for rows.Next() {
		var o horseOut
		if err := rows.Scan(&o.ID, &o.Name, &o.ColorKey, &o.Role, &o.ProfileStatus); err != nil {
			h.fail(w, r, err)
			return
		}
		o.HasProfile = o.ProfileStatus != ""
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"horses": out})
}
