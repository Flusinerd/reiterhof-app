// Package observations implements the reports about a horse ("Auffälligkeiten", JAN-50 to
// JAN-52): a member reports cough, lameness, colic and so on, optionally with photos, and an
// urgent report immediately alerts everybody who can help. See docs/domains/observations.md.
package observations

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/horses"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
)

// EventChanged is published (with the observation and horse id) whenever an observation is
// created or its status changes.
const EventChanged = "observation.changed"

// Urgency and status values (observations.urgency / observations.status).
const (
	UrgencyInfo   = "info"
	UrgencyCheck  = "check"
	UrgencyUrgent = "urgent"

	StatusWatch = "watch"
	StatusDone  = "done"
)

// Accepted values.
var (
	Categories = []string{"cough", "lameness", "injury", "not_eating", "colic", "behavior", "blanket_equipment", "other"}
	BodyParts  = []string{"front_left", "front_right", "hind_left", "hind_right", "head", "back", "belly", "other"}
	Urgencies  = []string{UrgencyInfo, UrgencyCheck, UrgencyUrgent}
	Statuses   = []string{StatusWatch, StatusDone}
)

const (
	maxDescription = 2000
	maxMedia       = 6
	defaultLimit   = 50
	maxLimit       = 100
)

// Register adds the observation routes (all behind auth.RequireStable).
//
//	POST  /api/v1/observations               report (every member); urgent also returns the emergency card
//	GET   /api/v1/horses/{id}/observations   list of a horse, newest first (?status=watch|done&limit=)
//	GET   /api/v1/observations/{id}          one observation
//	PATCH /api/v1/observations/{id}          {status}: owner, admin or reporter
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	route := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, auth.RequireStable(fn))
	}
	route("POST /api/v1/observations", h.create)
	route("GET /api/v1/horses/{id}/observations", h.list)
	route("GET /api/v1/observations/{id}", h.get)
	route("PATCH /api/v1/observations/{id}", h.patch)
}

type handler struct{ deps httpx.Deps }

// Person is the reporter as shown to other members.
type Person struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ColorKey *string `json:"color_key"`
}

// Media is one attached photo.
type Media struct {
	// Path is the stored path (files API); URL loads it (needs the session token).
	Path        string `json:"path"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
}

// Observation is the API shape of a report.
type Observation struct {
	ID          string    `json:"id"`
	HorseID     string    `json:"horse_id"`
	HorseName   string    `json:"horse_name"`
	Reporter    Person    `json:"reporter"`
	Category    *string   `json:"category"`
	BodyPart    *string   `json:"body_part"`
	Description *string   `json:"description"`
	Media       []Media   `json:"media"`
	Urgency     string    `json:"urgency"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	// CanChange: the caller may set the status (owner of the horse, admin or reporter).
	CanChange bool `json:"can_change"`
	// RehaPlanID is the reha plan created from this observation (the active one, else the newest),
	// null if there is none. The app hides "In Reha-Plan umwandeln" and links to the plan instead.
	RehaPlanID *string `json:"reha_plan_id"`
}

// CreateResponse is the body of POST /observations.
type CreateResponse struct {
	Observation Observation `json:"observation"`
	// Emergency is the emergency card of the horse for an urgent report (the app opens it
	// immediately) and null otherwise.
	Emergency *horses.EmergencyCard `json:"emergency"`
}

// $2 = caller is admin, $3 = caller id (for can_change).
const selectObservation = `SELECT o.id, o.horse_id, h.name, o.reported_by, u.name, u.avatar_color,
		o.category, o.body_part, o.description, o.media, o.urgency, o.status, o.created_at,
		($2::boolean OR h.owner_id = $3 OR o.reported_by = $3),
		(SELECT rp.id::text FROM reha_plans rp
		  WHERE rp.observation_id = o.id AND rp.stable_id = o.stable_id
		  ORDER BY rp.active DESC, rp.created_at DESC LIMIT 1)
	FROM observations o
	JOIN horses h ON h.id = o.horse_id AND h.stable_id = o.stable_id
	JOIN users u ON u.id = o.reported_by`

// queryer is what pgxpool.Pool and pgx.Tx share.
type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func scanObservation(row pgx.Row) (Observation, error) {
	var (
		o     Observation
		media []string
	)
	err := row.Scan(&o.ID, &o.HorseID, &o.HorseName, &o.Reporter.ID, &o.Reporter.Name, &o.Reporter.ColorKey,
		&o.Category, &o.BodyPart, &o.Description, &media, &o.Urgency, &o.Status, &o.CreatedAt, &o.CanChange, &o.RehaPlanID)
	if err != nil {
		return o, err
	}
	o.Media = make([]Media, 0, len(media))
	for _, p := range media {
		ct, _ := files.ContentTypeOf(p)
		o.Media = append(o.Media, Media{Path: p, URL: files.URLFor(p), ContentType: ct})
	}
	return o, nil
}

// load reads one observation of the stable as seen by user.
func load(ctx context.Context, q queryer, user auth.User, id string) (Observation, error) {
	return scanObservation(q.QueryRow(ctx, selectObservation+` WHERE o.stable_id = $1 AND o.id = $4`,
		user.StableID, user.IsAdmin, user.ID, id))
}

func (h *handler) fail(w http.ResponseWriter, what string, err error) {
	h.deps.Log.Error("observations: "+what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func invalid(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusBadRequest, "validation_failed", msg)
}

func notFound(w http.ResponseWriter, what string) {
	httpx.WriteError(w, http.StatusNotFound, "not_found", what+" not found")
}

func isInvalidUUID(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "22P02"
}

type createInput struct {
	HorseID     string   `json:"horse_id"`
	Category    string   `json:"category"`
	BodyPart    string   `json:"body_part"`
	Description string   `json:"description"`
	Media       []string `json:"media"`
	Urgency     string   `json:"urgency"`
}

// validate normalizes the input and returns a message for the first problem.
func (in *createInput) validate(stableID string) string {
	if in.HorseID == "" {
		return "horse_id is required"
	}
	if !slices.Contains(Categories, in.Category) {
		return "category must be one of " + strings.Join(Categories, ", ")
	}
	if in.BodyPart != "" && !slices.Contains(BodyParts, in.BodyPart) {
		return "body_part must be one of " + strings.Join(BodyParts, ", ")
	}
	in.Description = strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(in.Description) > maxDescription {
		return "description must be at most " + strconv.Itoa(maxDescription) + " characters"
	}
	if in.Urgency == "" {
		in.Urgency = UrgencyInfo
	}
	if !slices.Contains(Urgencies, in.Urgency) {
		return "urgency must be one of " + strings.Join(Urgencies, ", ")
	}
	if len(in.Media) > maxMedia {
		return "at most " + strconv.Itoa(maxMedia) + " photos per report"
	}
	seen := make(map[string]bool, len(in.Media))
	for _, p := range in.Media {
		ct, ok := files.ContentTypeOf(p)
		if !files.Belongs(stableID, p) || !ok || !strings.HasPrefix(ct, "image/") {
			return "media must be paths of photos uploaded to this stable"
		}
		if seen[p] {
			return "media contains a path twice"
		}
		seen[p] = true
	}
	return ""
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var in createInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if msg := in.validate(user.StableID); msg != "" {
		invalid(w, msg)
		return
	}
	// Every member may report for every horse of the stable (spec: member "darf melden").
	ok, err := auth.HorseInStable(r.Context(), h.deps.Pool, user, in.HorseID)
	if err != nil {
		h.fail(w, "horse in stable", err)
		return
	}
	if !ok {
		notFound(w, "horse")
		return
	}
	media := in.Media
	if media == nil {
		media = []string{}
	}

	tx, err := h.deps.Pool.Begin(r.Context())
	if err != nil {
		h.fail(w, "begin", err)
		return
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(r.Context())) }()
	var id string
	err = tx.QueryRow(r.Context(), `INSERT INTO observations (stable_id, horse_id, reported_by, category, body_part, description, media, urgency)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8) RETURNING id`,
		user.StableID, in.HorseID, user.ID, in.Category, in.BodyPart, in.Description, media, in.Urgency).Scan(&id)
	if err != nil {
		h.fail(w, "insert", err)
		return
	}
	if err := realtime.Publish(r.Context(), tx, user.StableID, EventChanged, map[string]any{"id": id, "horse_id": in.HorseID}); err != nil {
		h.fail(w, "publish", err)
		return
	}
	obs, err := load(r.Context(), tx, user, id)
	if err != nil {
		h.fail(w, "load", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		h.fail(w, "commit", err)
		return
	}

	resp := CreateResponse{Observation: obs}
	if obs.Urgency == UrgencyUrgent {
		card, err := horses.LoadEmergencyCard(r.Context(), h.deps.Pool, user.StableID, obs.HorseID, user.ID, user.IsAdmin)
		if err != nil {
			h.fail(w, "emergency card", err)
			return
		}
		resp.Emergency = &card
	}
	h.notify(r.Context(), user, obs)
	httpx.WriteJSON(w, http.StatusCreated, resp)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	horseID := r.PathValue("id")
	ok, err := auth.HorseInStable(r.Context(), h.deps.Pool, user, horseID)
	if err != nil {
		h.fail(w, "horse in stable", err)
		return
	}
	if !ok {
		notFound(w, "horse")
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && !slices.Contains(Statuses, status) {
		invalid(w, "status must be one of "+strings.Join(Statuses, ", "))
		return
	}
	limit := defaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxLimit {
			invalid(w, "limit must be between 1 and "+strconv.Itoa(maxLimit))
			return
		}
		limit = n
	}
	rows, err := h.deps.Pool.Query(r.Context(), selectObservation+`
		WHERE o.stable_id = $1 AND o.horse_id = $4 AND ($5 = '' OR o.status = $5)
		ORDER BY o.created_at DESC, o.id DESC LIMIT $6`,
		user.StableID, user.IsAdmin, user.ID, horseID, status, limit)
	if err != nil {
		h.fail(w, "list", err)
		return
	}
	defer rows.Close()
	out := []Observation{}
	for rows.Next() {
		o, err := scanObservation(rows)
		if err != nil {
			h.fail(w, "scan", err)
			return
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		h.fail(w, "list", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	o, err := load(r.Context(), h.deps.Pool, user, r.PathValue("id"))
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		notFound(w, "observation")
		return
	}
	if err != nil {
		h.fail(w, "get", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

type patchInput struct {
	Status *string `json:"status"`
}

func (h *handler) patch(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var in patchInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Status == nil || !slices.Contains(Statuses, *in.Status) {
		invalid(w, "status must be one of "+strings.Join(Statuses, ", "))
		return
	}
	cur, err := load(r.Context(), h.deps.Pool, user, r.PathValue("id"))
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		notFound(w, "observation")
		return
	}
	if err != nil {
		h.fail(w, "load", err)
		return
	}
	if !cur.CanChange {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner, an admin or the reporter may change the status")
		return
	}
	tx, err := h.deps.Pool.Begin(r.Context())
	if err != nil {
		h.fail(w, "begin", err)
		return
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(r.Context())) }()
	if _, err := tx.Exec(r.Context(), `UPDATE observations SET status = $3 WHERE id = $1 AND stable_id = $2`,
		cur.ID, user.StableID, *in.Status); err != nil {
		h.fail(w, "update", err)
		return
	}
	if err := realtime.Publish(r.Context(), tx, user.StableID, EventChanged, map[string]any{"id": cur.ID, "horse_id": cur.HorseID}); err != nil {
		h.fail(w, "publish", err)
		return
	}
	o, err := load(r.Context(), tx, user, cur.ID)
	if err != nil {
		h.fail(w, "reload", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		h.fail(w, "commit", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}
