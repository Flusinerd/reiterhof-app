// Package horses implements horses, riders (RB), the emergency card, extra emergency
// contacts and horse documents (JAN-10, JAN-49, JAN-53). See docs/domains/horses.md.
package horses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Register adds all horse routes. Everything requires a signed-in user with a stable.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	route := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, auth.RequireStable(fn))
	}
	route("GET /api/v1/members", h.listMembers)

	route("GET /api/v1/horses", h.list)
	route("POST /api/v1/horses", h.create)
	route("GET /api/v1/horses/{id}", h.get)
	route("PATCH /api/v1/horses/{id}", h.patch)
	route("PUT /api/v1/horses/{id}/riders/{userId}", h.putRider)
	route("DELETE /api/v1/horses/{id}/riders/{userId}", h.deleteRider)

	route("GET /api/v1/horses/{id}/emergency", h.emergency)
	route("POST /api/v1/horses/{id}/emergency-contacts", h.createContact)
	route("PATCH /api/v1/horses/{id}/emergency-contacts/{contactId}", h.patchContact)
	route("DELETE /api/v1/horses/{id}/emergency-contacts/{contactId}", h.deleteContact)

	h.registerDocuments(mux)
}

type handler struct{ deps httpx.Deps }

// ColorKeys are the accepted horse color keys (see docs/design-system.md).
var ColorKeys = []string{"green", "amber", "blue", "rose", "violet", "teal", "neutral"}

// Sexes are the accepted values of horses.sex.
var Sexes = []string{"mare", "gelding", "stallion"}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// Person is a user as embedded in horse responses.
type Person struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ColorKey *string `json:"color_key"`
}

// Rider is a person allowed to ride a horse with a positive rule list.
type Rider struct {
	UserID   string   `json:"user_id"`
	Name     string   `json:"name"`
	ColorKey *string  `json:"color_key"`
	Rules    []string `json:"rules"`
}

// Horse is the list/detail representation of a horse.
type Horse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Box        *string `json:"box"`
	Sex        *string `json:"sex"`
	BirthYear  *int    `json:"birth_year"`
	Breed      *string `json:"breed"`
	ColorKey   *string `json:"color_key"`
	WeightKG   *int    `json:"weight_kg"`
	HelperNote *string `json:"helper_note"`
	Owner      *Person `json:"owner"`
	Riders     []Rider `json:"riders"`
	// IsMine: the caller owns the horse. IRide: the caller is listed as rider.
	IsMine bool `json:"is_mine"`
	IRide  bool `json:"i_ride"`
	// CanManage: the caller may edit the horse (owner or admin).
	CanManage bool `json:"can_manage"`
	// MyRules are the caller's rider rules ([] if not a rider).
	MyRules []string `json:"my_rules"`
}

const horseColumns = `h.id, h.name, h.box, h.sex, h.birth_year, h.breed, h.color_key, h.weight_kg, h.helper_note,
	h.owner_id, o.name, o.avatar_color`

// loadHorses returns the horses of the user's stable (all, or only horseID) with owner,
// riders and the caller's flags, ordered by name.
func (h *handler) loadHorses(ctx context.Context, user auth.User, horseID string) ([]Horse, error) {
	q := `SELECT ` + horseColumns + ` FROM horses h LEFT JOIN users o ON o.id = h.owner_id
		WHERE h.stable_id = $1`
	args := []any{user.StableID}
	if horseID != "" {
		q += ` AND h.id = $2`
		args = append(args, horseID)
	}
	rows, err := h.deps.Pool.Query(ctx, q+` ORDER BY lower(h.name), h.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	horses := []Horse{}
	index := map[string]int{}
	for rows.Next() {
		var (
			hr        Horse
			ownerID   *string
			ownerName *string
			ownerCol  *string
		)
		if err := rows.Scan(&hr.ID, &hr.Name, &hr.Box, &hr.Sex, &hr.BirthYear, &hr.Breed, &hr.ColorKey,
			&hr.WeightKG, &hr.HelperNote, &ownerID, &ownerName, &ownerCol); err != nil {
			return nil, err
		}
		if ownerID != nil && ownerName != nil {
			hr.Owner = &Person{ID: *ownerID, Name: *ownerName, ColorKey: ownerCol}
		}
		hr.Riders = []Rider{}
		hr.MyRules = []string{}
		hr.IsMine = ownerID != nil && *ownerID == user.ID
		hr.CanManage = hr.IsMine || user.IsAdmin
		index[hr.ID] = len(horses)
		horses = append(horses, hr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rq := `SELECT r.horse_id, r.user_id, u.name, u.avatar_color, r.rules
		FROM horse_riders r JOIN users u ON u.id = r.user_id WHERE r.stable_id = $1`
	rargs := []any{user.StableID}
	if horseID != "" {
		rq += ` AND r.horse_id = $2`
		rargs = append(rargs, horseID)
	}
	rrows, err := h.deps.Pool.Query(ctx, rq+` ORDER BY lower(u.name), u.id`, rargs...)
	if err != nil {
		return nil, err
	}
	defer rrows.Close()
	for rrows.Next() {
		var (
			hid string
			r   Rider
			raw []byte
		)
		if err := rrows.Scan(&hid, &r.UserID, &r.Name, &r.ColorKey, &raw); err != nil {
			return nil, err
		}
		r.Rules = []string{}
		if err := json.Unmarshal(raw, &r.Rules); err != nil {
			return nil, fmt.Errorf("rules of rider %s: %w", r.UserID, err)
		}
		i, ok := index[hid]
		if !ok {
			continue
		}
		horses[i].Riders = append(horses[i].Riders, r)
		if r.UserID == user.ID {
			horses[i].IRide = true
			horses[i].MyRules = r.Rules
		}
	}
	return horses, rrows.Err()
}

func (h *handler) fail(w http.ResponseWriter, what string, err error) {
	h.deps.Log.Error("horses: "+what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func notFound(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusNotFound, "not_found", "horse not found")
}

func forbidden(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusForbidden, "forbidden", msg)
}

func invalid(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusBadRequest, "validation_failed", msg)
}

// horseInStable answers 404 itself and returns false if the horse is not in the caller's stable.
func (h *handler) horseInStable(w http.ResponseWriter, r *http.Request) (auth.User, string, bool) {
	user, _ := auth.UserFrom(r.Context())
	id := r.PathValue("id")
	ok, err := auth.HorseInStable(r.Context(), h.deps.Pool, user, id)
	if err != nil {
		h.fail(w, "horse in stable", err)
		return user, id, false
	}
	if !ok {
		notFound(w)
		return user, id, false
	}
	return user, id, true
}

// canManage is horseInStable plus owner-or-admin (403 otherwise).
func (h *handler) canManage(w http.ResponseWriter, r *http.Request) (auth.User, string, bool) {
	user, id, ok := h.horseInStable(w, r)
	if !ok {
		return user, id, false
	}
	manage, err := auth.CanManageHorse(r.Context(), h.deps.Pool, user, id)
	if err != nil {
		h.fail(w, "can manage", err)
		return user, id, false
	}
	if !manage {
		forbidden(w, "only the owner or an admin may change this horse")
		return user, id, false
	}
	return user, id, true
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	horses, err := h.loadHorses(r.Context(), user, "")
	if err != nil {
		h.fail(w, "list", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, horses)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.horseInStable(w, r)
	if !ok {
		return
	}
	h.writeHorse(w, r, user, id, http.StatusOK)
}

func (h *handler) writeHorse(w http.ResponseWriter, r *http.Request, user auth.User, id string, status int) {
	horses, err := h.loadHorses(r.Context(), user, id)
	if err != nil {
		h.fail(w, "load horse", err)
		return
	}
	if len(horses) != 1 {
		notFound(w)
		return
	}
	httpx.WriteJSON(w, status, horses[0])
}

// horseInput is the body of POST and PATCH. Absent fields are left alone (PATCH) or
// empty (POST). For text fields "" clears the value, for numbers 0 does.
type horseInput struct {
	Name                *string `json:"name"`
	Box                 *string `json:"box"`
	Sex                 *string `json:"sex"`
	BirthYear           *int    `json:"birth_year"`
	Breed               *string `json:"breed"`
	ColorKey            *string `json:"color_key"`
	WeightKG            *int    `json:"weight_kg"`
	HelperNote          *string `json:"helper_note"`
	EmergencyNote       *string `json:"emergency_note"`
	EmergencyMedication *string `json:"emergency_medication"`
	PermanentMedication *string `json:"permanent_medication"`
	Allergies           *string `json:"allergies"`
	Insurance           *string `json:"insurance"`
	VetName             *string `json:"vet_name"`
	VetPhone            *string `json:"vet_phone"`
	// OwnerID may only be set by admins.
	OwnerID *string `json:"owner_id"`
}

type textField struct {
	column string
	value  *string
	max    int
	label  string
}

func (in *horseInput) textFields() []textField {
	return []textField{
		{"box", in.Box, 20, "box"},
		{"breed", in.Breed, 80, "breed"},
		{"helper_note", in.HelperNote, 2000, "helper_note"},
		{"emergency_note", in.EmergencyNote, 2000, "emergency_note"},
		{"emergency_medication", in.EmergencyMedication, 2000, "emergency_medication"},
		{"permanent_medication", in.PermanentMedication, 2000, "permanent_medication"},
		{"allergies", in.Allergies, 1000, "allergies"},
		{"insurance", in.Insurance, 500, "insurance"},
		{"vet_name", in.VetName, 120, "vet_name"},
		{"vet_phone", in.VetPhone, 40, "vet_phone"},
	}
}

// validate checks all present fields and returns a message for the first problem.
func (in *horseInput) validate(thisYear int) string {
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" || utf8.RuneCountInString(n) > 80 {
			return "name must be 1 to 80 characters"
		}
		in.Name = &n
	}
	for _, f := range in.textFields() {
		if f.value != nil && utf8.RuneCountInString(*f.value) > f.max {
			return f.label + " is too long"
		}
	}
	if in.Sex != nil && *in.Sex != "" && !contains(Sexes, *in.Sex) {
		return "sex must be one of mare, gelding, stallion"
	}
	if in.ColorKey != nil && *in.ColorKey != "" && !contains(ColorKeys, *in.ColorKey) {
		return "color_key must be one of " + strings.Join(ColorKeys, ", ")
	}
	if in.BirthYear != nil && *in.BirthYear != 0 && (*in.BirthYear < 1970 || *in.BirthYear > thisYear) {
		return fmt.Sprintf("birth_year must be between 1970 and %d", thisYear)
	}
	if in.WeightKG != nil && *in.WeightKG != 0 && (*in.WeightKG < 20 || *in.WeightKG > 1500) {
		return "weight_kg must be between 20 and 1500"
	}
	return ""
}

// userInStable reports whether the user exists in the stable.
func (h *handler) userInStable(ctx context.Context, stableID, userID string) (bool, error) {
	var ok bool
	err := h.deps.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND stable_id = $2)`,
		userID, stableID).Scan(&ok)
	if isInvalidUUID(err) {
		return false, nil
	}
	return ok, err
}

func isInvalidUUID(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "22P02"
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var in horseInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Name == nil {
		invalid(w, "name is required")
		return
	}
	if msg := in.validate(h.deps.Now().Year()); msg != "" {
		invalid(w, msg)
		return
	}
	owner := user.ID
	if in.OwnerID != nil && *in.OwnerID != user.ID {
		if !user.IsAdmin {
			forbidden(w, "only admins may create a horse for someone else")
			return
		}
		ok, err := h.userInStable(r.Context(), user.StableID, *in.OwnerID)
		if err != nil {
			h.fail(w, "owner lookup", err)
			return
		}
		if !ok {
			invalid(w, "owner_id is not a member of the stable")
			return
		}
		owner = *in.OwnerID
	}

	var id string
	err := h.deps.Pool.QueryRow(r.Context(), `INSERT INTO horses
		(stable_id, name, box, sex, birth_year, breed, color_key, weight_kg, helper_note, emergency_note,
		 emergency_medication, permanent_medication, allergies, insurance, vet_name, vet_phone, owner_id)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5::int, 0), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8::int, 0),
		        NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''),
		        NULLIF($15, ''), NULLIF($16, ''), $17)
		RETURNING id`,
		user.StableID, *in.Name, in.Box, in.Sex, in.BirthYear, in.Breed, in.ColorKey, in.WeightKG, in.HelperNote,
		in.EmergencyNote, in.EmergencyMedication, in.PermanentMedication, in.Allergies, in.Insurance,
		in.VetName, in.VetPhone, owner).Scan(&id)
	if err != nil {
		h.fail(w, "create", err)
		return
	}
	h.writeHorse(w, r, user, id, http.StatusCreated)
}

func (h *handler) patch(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.canManage(w, r)
	if !ok {
		return
	}
	var in horseInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		invalid(w, "name must not be empty")
		return
	}
	if msg := in.validate(h.deps.Now().Year()); msg != "" {
		invalid(w, msg)
		return
	}
	if in.OwnerID != nil {
		if !user.IsAdmin {
			forbidden(w, "only admins may change the owner")
			return
		}
		ok, err := h.userInStable(r.Context(), user.StableID, *in.OwnerID)
		if err != nil {
			h.fail(w, "owner lookup", err)
			return
		}
		if !ok {
			invalid(w, "owner_id is not a member of the stable")
			return
		}
	}

	var sets []string
	args := []any{id, user.StableID}
	add := func(expr string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf(expr, len(args)))
	}
	if in.Name != nil {
		add("name = $%d", *in.Name)
	}
	if in.Sex != nil {
		add("sex = NULLIF($%d, '')", *in.Sex)
	}
	if in.ColorKey != nil {
		add("color_key = NULLIF($%d, '')", *in.ColorKey)
	}
	if in.BirthYear != nil {
		add("birth_year = NULLIF($%d::int, 0)", *in.BirthYear)
	}
	if in.WeightKG != nil {
		add("weight_kg = NULLIF($%d::int, 0)", *in.WeightKG)
	}
	if in.OwnerID != nil {
		add("owner_id = $%d", *in.OwnerID)
	}
	for _, f := range in.textFields() {
		if f.value != nil {
			add(f.column+" = NULLIF($%d, '')", *f.value)
		}
	}
	if len(sets) > 0 {
		_, err := h.deps.Pool.Exec(r.Context(),
			`UPDATE horses SET `+strings.Join(sets, ", ")+` WHERE id = $1 AND stable_id = $2`, args...)
		if err != nil {
			h.fail(w, "patch", err)
			return
		}
	}
	h.writeHorse(w, r, user, id, http.StatusOK)
}

func (h *handler) putRider(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.canManage(w, r)
	if !ok {
		return
	}
	var in struct {
		Rules *[]string `json:"rules"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	rules := DefaultRiderRules()
	if in.Rules != nil {
		var err error
		if rules, err = NormalizeRules(*in.Rules); err != nil {
			invalid(w, err.Error()+"; allowed: "+strings.Join(Rules(), ", "))
			return
		}
	}
	riderID := r.PathValue("userId")
	var name string
	var color *string
	err := h.deps.Pool.QueryRow(r.Context(), `SELECT name, avatar_color FROM users WHERE id = $1 AND stable_id = $2`,
		riderID, user.StableID).Scan(&name, &color)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if err != nil {
		h.fail(w, "rider lookup", err)
		return
	}
	isOwner, err := auth.IsOwner(r.Context(), h.deps.Pool, auth.User{ID: riderID, StableID: user.StableID}, horseID)
	if err != nil {
		h.fail(w, "owner check", err)
		return
	}
	if isOwner {
		invalid(w, "the owner does not need a rider entry")
		return
	}
	raw, _ := json.Marshal(rules)
	_, err = h.deps.Pool.Exec(r.Context(), `INSERT INTO horse_riders (stable_id, horse_id, user_id, rules)
		VALUES ($1, $2, $3, $4) ON CONFLICT (horse_id, user_id) DO UPDATE SET rules = EXCLUDED.rules`,
		user.StableID, horseID, riderID, raw)
	if err != nil {
		h.fail(w, "put rider", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, Rider{UserID: riderID, Name: name, ColorKey: color, Rules: rules})
}

func (h *handler) deleteRider(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.canManage(w, r)
	if !ok {
		return
	}
	tag, err := h.deps.Pool.Exec(r.Context(), `DELETE FROM horse_riders WHERE horse_id = $1 AND user_id = $2 AND stable_id = $3`,
		horseID, r.PathValue("userId"), user.StableID)
	if isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "rider not found")
		return
	}
	if err != nil {
		h.fail(w, "delete rider", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "rider not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Member is a person of the stable, for pickers.
type Member struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ColorKey *string `json:"color_key"`
	IsMe     bool    `json:"is_me"`
}

func (h *handler) listMembers(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	rows, err := h.deps.Pool.Query(r.Context(),
		`SELECT id, name, avatar_color FROM users WHERE stable_id = $1 ORDER BY lower(name), id`, user.StableID)
	if err != nil {
		h.fail(w, "members", err)
		return
	}
	defer rows.Close()
	members := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Name, &m.ColorKey); err != nil {
			h.fail(w, "members scan", err)
			return
		}
		m.IsMe = m.ID == user.ID
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		h.fail(w, "members", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, members)
}
