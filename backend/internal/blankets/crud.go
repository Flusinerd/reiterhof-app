package blankets

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

const blanketCols = `id::text, horse_id::text, name, fill_g, color, location, photo_path`

func scanBlanket(row pgx.Row) (Blanket, error) {
	var b Blanket
	if err := row.Scan(&b.ID, &b.HorseID, &b.Name, &b.FillG, &b.Color, &b.Location, &b.PhotoPath); err != nil {
		return Blanket{}, err
	}
	b.PhotoURL = photoURL(b.PhotoPath)
	return b, nil
}

// blanketInput is the body of POST and PATCH. Text fields: "" clears (except name).
type blanketInput struct {
	Name      *string `json:"name"`
	FillG     *int    `json:"fill_g"`
	Color     *string `json:"color"`
	Location  *string `json:"location"`
	PhotoPath *string `json:"photo_path"`
}

// validate checks the fields that are present. It returns an error message or "".
func (in blanketInput) validate(stableID string, create bool) string {
	if create && (in.Name == nil || nonEmpty(in.Name) == "") {
		return "name is required"
	}
	if in.Name != nil {
		if s, ok := text(in.Name, maxNameLen); !ok || s == "" {
			return "name must be 1 to 80 characters"
		}
	}
	if in.FillG != nil && (*in.FillG < 0 || *in.FillG > maxFillG) {
		return "fill_g must be between 0 and 2000"
	}
	if _, ok := text(in.Color, maxColorLen); !ok {
		return "color is too long"
	}
	if _, ok := text(in.Location, maxLocationLen); !ok {
		return "location is too long (80 characters)"
	}
	if in.PhotoPath != nil && *in.PhotoPath != "" && !files.Belongs(stableID, *in.PhotoPath) {
		return "photo_path is not an uploaded file of this stable"
	}
	return ""
}

func nonEmpty(s *string) string {
	v, _ := text(s, 1<<20)
	return v
}

// nullable turns "" into NULL for optional text columns.
func nullable(s *string) *string {
	v, _ := text(s, 1<<20)
	if v == "" {
		return nil
	}
	return &v
}

func (h *handler) listBlankets(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, false)
	if !ok {
		return
	}
	rows, err := h.deps.Pool.Query(r.Context(), `SELECT `+blanketCols+` FROM blankets
		WHERE stable_id = $1 AND horse_id = $2 ORDER BY fill_g, name, id`, user.StableID, horseID)
	if err != nil {
		h.internal(w, "list blankets", err)
		return
	}
	defer rows.Close()
	out := []Blanket{}
	for rows.Next() {
		b, err := scanBlanket(rows)
		if err != nil {
			h.internal(w, "scan blanket", err)
			return
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		h.internal(w, "list blankets", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"blankets": out})
}

func (h *handler) createBlanket(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, true)
	if !ok {
		return
	}
	var in blanketInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if msg := in.validate(user.StableID, true); msg != "" {
		invalid(w, msg)
		return
	}
	fill := 0
	if in.FillG != nil {
		fill = *in.FillG
	}
	b, err := scanBlanket(h.deps.Pool.QueryRow(r.Context(), `
		INSERT INTO blankets (stable_id, horse_id, name, fill_g, color, location, photo_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+blanketCols,
		user.StableID, horseID, nonEmpty(in.Name), fill, nullable(in.Color), nullable(in.Location), nullable(in.PhotoPath)))
	if err != nil {
		h.internal(w, "create blanket", err)
		return
	}
	h.svc.hint(r.Context(), user.StableID, EventPlanChanged, map[string]any{"horse_id": horseID})
	httpx.WriteJSON(w, http.StatusCreated, b)
}

func (h *handler) patchBlanket(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, true)
	if !ok {
		return
	}
	var in blanketInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if msg := in.validate(user.StableID, false); msg != "" {
		invalid(w, msg)
		return
	}
	// Absent fields keep their value; "" clears color, location and photo.
	b, err := scanBlanket(h.deps.Pool.QueryRow(r.Context(), `
		UPDATE blankets SET
			name       = COALESCE($4, name),
			fill_g     = COALESCE($5, fill_g),
			color      = CASE WHEN $6::boolean THEN $7 ELSE color END,
			location   = CASE WHEN $8::boolean THEN $9 ELSE location END,
			photo_path = CASE WHEN $10::boolean THEN $11 ELSE photo_path END
		WHERE id::text = $1 AND stable_id = $2 AND horse_id = $3
		RETURNING `+blanketCols,
		r.PathValue("blanketId"), user.StableID, horseID,
		nullable(in.Name), in.FillG,
		in.Color != nil, nullable(in.Color),
		in.Location != nil, nullable(in.Location),
		in.PhotoPath != nil, nullable(in.PhotoPath)))
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "blanket not found")
		return
	}
	if err != nil {
		h.internal(w, "update blanket", err)
		return
	}
	h.svc.hint(r.Context(), user.StableID, EventPlanChanged, map[string]any{"horse_id": horseID})
	httpx.WriteJSON(w, http.StatusOK, b)
}

// deleteBlanket removes a blanket unless a rule still uses it: the foreign key would
// turn such a rule into "no blanket", which silently changes its meaning. The photo file
// is kept (the path might be shared with another record); orphaned uploads are harmless.
func (h *handler) deleteBlanket(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, true)
	if !ok {
		return
	}
	id := r.PathValue("blanketId")
	var inUse bool
	err := h.deps.Pool.QueryRow(r.Context(), `SELECT EXISTS (
		SELECT 1 FROM blanket_rules WHERE stable_id = $1 AND horse_id = $2 AND blanket_id::text = $3)`,
		user.StableID, horseID, id).Scan(&inUse)
	if err != nil {
		h.internal(w, "check blanket use", err)
		return
	}
	if inUse {
		httpx.WriteError(w, http.StatusConflict, "in_use", "the blanket is used by a rule; change the rule first")
		return
	}
	tag, err := h.deps.Pool.Exec(r.Context(), `DELETE FROM blankets WHERE id::text = $1 AND stable_id = $2 AND horse_id = $3`,
		id, user.StableID, horseID)
	if err != nil {
		h.internal(w, "delete blanket", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "blanket not found")
		return
	}
	h.svc.hint(r.Context(), user.StableID, EventPlanChanged, map[string]any{"horse_id": horseID})
	w.WriteHeader(http.StatusNoContent)
}
