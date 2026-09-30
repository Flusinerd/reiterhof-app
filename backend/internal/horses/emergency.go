package horses

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Contact is an extra emergency contact of a horse (emergency_contacts).
type Contact struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// EmergencyCard is the emergency card of a horse (JAN-49). Every member of the stable may read it.
type EmergencyCard struct {
	HorseID             string    `json:"horse_id"`
	HorseName           string    `json:"horse_name"`
	Box                 *string   `json:"box"`
	WeightKG            *int      `json:"weight_kg"`
	Owner               *Owner    `json:"owner"`
	VetName             *string   `json:"vet_name"`
	VetPhone            *string   `json:"vet_phone"`
	EmergencyNote       *string   `json:"emergency_note"`
	EmergencyMedication *string   `json:"emergency_medication"`
	PermanentMedication *string   `json:"permanent_medication"`
	Allergies           *string   `json:"allergies"`
	Insurance           *string   `json:"insurance"`
	Contacts            []Contact `json:"contacts"`
	// CanManage: the caller may edit the card (owner or admin).
	CanManage bool `json:"can_manage"`
}

// Owner is the owner as shown on the emergency card, including the phone number
// (users.phone; the only place other members see it).
type Owner struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Phone *string `json:"phone"`
}

func (h *handler) emergency(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.horseInStable(w, r)
	if !ok {
		return
	}
	card, err := LoadEmergencyCard(r.Context(), h.deps.Pool, user.StableID, id, user.ID, user.IsAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		notFound(w)
		return
	}
	if err != nil {
		h.fail(w, "emergency", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, card)
}

// LoadEmergencyCard reads the emergency card of a horse of the stable (pgx.ErrNoRows if the
// horse does not exist there). userID and isAdmin only decide CanManage. The observations
// feature uses it to hand the card back right after an urgent report.
func LoadEmergencyCard(ctx context.Context, pool *pgxpool.Pool, stableID, horseID, userID string, isAdmin bool) (EmergencyCard, error) {
	var (
		card      EmergencyCard
		ownerID   *string
		ownerName *string
		ownerTel  *string
	)
	err := pool.QueryRow(ctx, `SELECT h.id, h.name, h.box, h.weight_kg, h.vet_name, h.vet_phone,
			h.emergency_note, h.emergency_medication, h.permanent_medication, h.allergies, h.insurance,
			h.owner_id, o.name, o.phone
		FROM horses h LEFT JOIN users o ON o.id = h.owner_id
		WHERE h.id = $1 AND h.stable_id = $2`, horseID, stableID).
		Scan(&card.HorseID, &card.HorseName, &card.Box, &card.WeightKG, &card.VetName, &card.VetPhone,
			&card.EmergencyNote, &card.EmergencyMedication, &card.PermanentMedication, &card.Allergies,
			&card.Insurance, &ownerID, &ownerName, &ownerTel)
	if err != nil {
		return EmergencyCard{}, err
	}
	if ownerID != nil && ownerName != nil {
		card.Owner = &Owner{ID: *ownerID, Name: *ownerName, Phone: ownerTel}
	}
	card.CanManage = isAdmin || (ownerID != nil && *ownerID == userID)

	rows, err := pool.Query(ctx, `SELECT id, label, name, phone FROM emergency_contacts
		WHERE horse_id = $1 AND stable_id = $2 ORDER BY created_at, id`, horseID, stableID)
	if err != nil {
		return EmergencyCard{}, err
	}
	defer rows.Close()
	card.Contacts = []Contact{}
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.Label, &c.Name, &c.Phone); err != nil {
			return EmergencyCard{}, err
		}
		card.Contacts = append(card.Contacts, c)
	}
	return card, rows.Err()
}

type contactInput struct {
	Label *string `json:"label"`
	Name  *string `json:"name"`
	Phone *string `json:"phone"`
}

func (in *contactInput) validate(requireAll bool) string {
	for _, f := range []struct {
		v    **string
		name string
		max  int
	}{{&in.Label, "label", 60}, {&in.Name, "name", 120}, {&in.Phone, "phone", 40}} {
		if *f.v == nil {
			if requireAll {
				return f.name + " is required"
			}
			continue
		}
		t := strings.TrimSpace(**f.v)
		if t == "" || utf8.RuneCountInString(t) > f.max {
			return f.name + " must be 1 to " + strconv.Itoa(f.max) + " characters"
		}
		*f.v = &t
	}
	return ""
}

func (h *handler) createContact(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.canManage(w, r)
	if !ok {
		return
	}
	var in contactInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if msg := in.validate(true); msg != "" {
		invalid(w, msg)
		return
	}
	c := Contact{Label: *in.Label, Name: *in.Name, Phone: *in.Phone}
	err := h.deps.Pool.QueryRow(r.Context(), `INSERT INTO emergency_contacts (stable_id, horse_id, label, name, phone)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`, user.StableID, id, c.Label, c.Name, c.Phone).Scan(&c.ID)
	if err != nil {
		h.fail(w, "create contact", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *handler) patchContact(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.canManage(w, r)
	if !ok {
		return
	}
	var in contactInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if msg := in.validate(false); msg != "" {
		invalid(w, msg)
		return
	}
	var c Contact
	err := h.deps.Pool.QueryRow(r.Context(), `UPDATE emergency_contacts SET
			label = COALESCE($4, label), name = COALESCE($5, name), phone = COALESCE($6, phone)
		WHERE id = $1 AND horse_id = $2 AND stable_id = $3
		RETURNING id, label, name, phone`,
		r.PathValue("contactId"), id, user.StableID, in.Label, in.Name, in.Phone).
		Scan(&c.ID, &c.Label, &c.Name, &c.Phone)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "contact not found")
		return
	}
	if err != nil {
		h.fail(w, "patch contact", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *handler) deleteContact(w http.ResponseWriter, r *http.Request) {
	user, id, ok := h.canManage(w, r)
	if !ok {
		return
	}
	tag, err := h.deps.Pool.Exec(r.Context(), `DELETE FROM emergency_contacts WHERE id = $1 AND horse_id = $2 AND stable_id = $3`,
		r.PathValue("contactId"), id, user.StableID)
	if isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "contact not found")
		return
	}
	if err != nil {
		h.fail(w, "delete contact", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "contact not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
