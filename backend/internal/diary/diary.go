// Package diary implements the horse diary ("Tagebuch" of the horse page): one reverse-chronological
// list that merges the training sessions and the observations of a horse. It owns no table, it
// reads `sessions` and `observations`. See docs/domains/horses.md.
//
// Health items keep no history (marking one done moves its due date), so there is nothing to merge
// for them.
package diary

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

const (
	defaultLimit    = 30
	maxLimit        = 100
	defaultTimezone = "Europe/Berlin"
)

// Register adds GET /api/v1/horses/{id}/diary (every member of the stable).
func Register(mux *http.ServeMux, deps httpx.Deps) {
	h := &handler{deps: deps}
	mux.Handle("GET /api/v1/horses/{id}/diary", auth.RequireStable(http.HandlerFunc(h.get)))
}

type handler struct{ deps httpx.Deps }

// Person is who wrote the entry.
type Person struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ColorKey *string `json:"color_key"`
}

// Session is the part of an entry of type "session".
type Session struct {
	Activity string  `json:"activity"`
	Label    string  `json:"label"`
	Minutes  int     `json:"minutes"`
	Note     *string `json:"note"`
	Feel     *string `json:"feel"`
	// ExerciseTitle is the library exercise that was practised, if one was linked.
	ExerciseTitle *string `json:"exercise_title"`
}

// Observation is the part of an entry of type "observation".
type Observation struct {
	Category    *string `json:"category"`
	BodyPart    *string `json:"body_part"`
	Description *string `json:"description"`
	Urgency     string  `json:"urgency"`
	Status      string  `json:"status"`
	// MediaURL is the first photo (files API, needs the session token), null without photos.
	MediaURL   *string `json:"media_url"`
	MediaCount int     `json:"media_count"`
}

// Entry is one line of the diary. Exactly one of Session and Observation is set, matching Type.
type Entry struct {
	// Type is "session" or "observation".
	Type string    `json:"type"`
	ID   string    `json:"id"`
	At   time.Time `json:"at"`
	// Day is the calendar day of At in the stable's time zone (YYYY-MM-DD).
	Day         string       `json:"day"`
	User        Person       `json:"user"`
	Session     *Session     `json:"session"`
	Observation *Observation `json:"observation"`
}

// Page is the response: newest first. NextBefore is the `before` value of the next page, null
// when the diary ends here.
type Page struct {
	Items      []Entry `json:"items"`
	NextBefore *string `json:"next_before"`
}

func (h *handler) fail(w http.ResponseWriter, what string, err error) {
	h.deps.Log.Error("diary: "+what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func invalid(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusBadRequest, "validation_failed", msg)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	ctx := r.Context()
	horseID := r.PathValue("id")
	ok, err := auth.HorseInStable(ctx, h.deps.Pool, user, horseID)
	if err != nil {
		h.fail(w, "horse in stable", err)
		return
	}
	if !ok {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "horse not found")
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
	before := h.deps.Now().Add(5 * time.Minute)
	if v := r.URL.Query().Get("before"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			invalid(w, "before must be an RFC 3339 timestamp")
			return
		}
		before = t
	}

	// Observations are visible to every member of the stable (like GET /horses/{id}/observations).
	// Training sessions follow the training rule: owner and admin see all, a rider the sessions
	// marked visible and their own, other members none.
	sessionsFor, seeSessions, err := h.sessionAccess(ctx, user, horseID)
	if err != nil {
		h.fail(w, "access", err)
		return
	}

	// Each source gives at most limit+1 entries, so the merge knows whether more exist.
	var items []Entry
	if seeSessions {
		s, err := h.sessions(ctx, user.StableID, horseID, before, sessionsFor, limit+1)
		if err != nil {
			h.fail(w, "sessions", err)
			return
		}
		items = append(items, s...)
	}
	o, err := h.observations(ctx, user.StableID, horseID, before, limit+1)
	if err != nil {
		h.fail(w, "observations", err)
		return
	}
	items = append(items, o...)
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].At.Equal(items[j].At) {
			return items[i].At.After(items[j].At)
		}
		return items[i].ID < items[j].ID
	})

	page := Page{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		next := page.Items[limit-1].At.UTC().Format(time.RFC3339Nano)
		page.NextBefore = &next
	}
	if page.Items == nil {
		page.Items = []Entry{}
	}
	loc := h.location(ctx, user.StableID)
	for i := range page.Items {
		page.Items[i].Day = page.Items[i].At.In(loc).Format("2006-01-02")
		page.Items[i].At = page.Items[i].At.UTC()
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

// sessionAccess returns whether the caller may see training sessions of the horse and, if so, the
// user id that restricts them to visible ones ("" = all).
func (h *handler) sessionAccess(ctx context.Context, user auth.User, horseID string) (onlyFor string, ok bool, err error) {
	manage, err := auth.CanManageHorse(ctx, h.deps.Pool, user, horseID)
	if err != nil {
		return "", false, err
	}
	if manage {
		return "", true, nil
	}
	rider, err := auth.IsRider(ctx, h.deps.Pool, user, horseID)
	if err != nil {
		return "", false, err
	}
	return user.ID, rider, nil
}

func (h *handler) sessions(ctx context.Context, stableID, horseID string, before time.Time, onlyFor string, n int) ([]Entry, error) {
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT s.id::text, s.started_at, u.id::text, u.name, u.avatar_color, s.activity, COALESCE(s.duration_min, 0),
		       s.note, s.feel, e.title
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		LEFT JOIN exercises e ON e.id = s.exercise_id
		WHERE s.stable_id = $1 AND s.horse_id = $2 AND s.started_at < $3
		  AND ($4 = '' OR s.visible_to_rider OR s.user_id::text = $4)
		ORDER BY s.started_at DESC, s.id LIMIT $5`, stableID, horseID, before, onlyFor, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		e := Entry{Type: "session"}
		s := &Session{}
		if err := rows.Scan(&e.ID, &e.At, &e.User.ID, &e.User.Name, &e.User.ColorKey, &s.Activity, &s.Minutes,
			&s.Note, &s.Feel, &s.ExerciseTitle); err != nil {
			return nil, err
		}
		s.Label = training.Activity(s.Activity).GermanName()
		e.Session = s
		out = append(out, e)
	}
	return out, rows.Err()
}

func (h *handler) observations(ctx context.Context, stableID, horseID string, before time.Time, n int) ([]Entry, error) {
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT o.id::text, o.created_at, u.id::text, u.name, u.avatar_color, o.category, o.body_part, o.description,
		       o.urgency, o.status, o.media
		FROM observations o
		JOIN users u ON u.id = o.reported_by
		WHERE o.stable_id = $1 AND o.horse_id = $2 AND o.created_at < $3
		ORDER BY o.created_at DESC, o.id LIMIT $4`, stableID, horseID, before, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		e := Entry{Type: "observation"}
		o := &Observation{}
		var media []string
		if err := rows.Scan(&e.ID, &e.At, &e.User.ID, &e.User.Name, &e.User.ColorKey, &o.Category, &o.BodyPart,
			&o.Description, &o.Urgency, &o.Status, &media); err != nil {
			return nil, err
		}
		o.MediaCount = len(media)
		if len(media) > 0 {
			url := files.URLFor(media[0])
			o.MediaURL = &url
		}
		e.Observation = o
		out = append(out, e)
	}
	return out, rows.Err()
}

// location returns the stable's time zone (Europe/Berlin if unset or unknown).
func (h *handler) location(ctx context.Context, stableID string) *time.Location {
	var tz string
	_ = h.deps.Pool.QueryRow(ctx, `SELECT timezone FROM stables WHERE id = $1`, stableID).Scan(&tz)
	if tz == "" {
		tz = defaultTimezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation(defaultTimezone)
	}
	if loc == nil {
		loc = time.UTC
	}
	return loc
}
