package trainingapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/privacy"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/load"
)

const (
	maxSessionMinutes = 600
	// Sessions may start at most this far in the future (clock skew of the phone).
	futureSlack = 5 * time.Minute
	// Sessions may be logged for at most this many days back.
	maxBackdateDays = 14
)

// reinSegment is one stretch on one rein; rein_changes is the ordered list of segments,
// so the number of rein changes is len-1 and the shares follow from the minutes.
type reinSegment struct {
	Rein    string  `json:"rein"` // left, right
	Minutes float64 `json:"minutes"`
}

// Feel values ("Luna war ...") and their German labels.
var feelLabels = map[string]string{"fresh": "Frisch", "loose": "Locker", "tired": "Müde", "tense": "Klemmig"}

type sessionRow struct {
	ID         string
	UserID     string
	UserName   string
	Activity   training.Activity
	StartedAt  time.Time
	Minutes    int
	Gait       map[string]float64
	Rein       []reinSegment
	Feel       *string
	Focus      *int
	Note       *string
	Visible    bool
	Load       float64
	Source     string
	ExerciseID *string
	DistanceM  *int
}

func (s sessionRow) canter() float64 { return clamp01(s.Gait["canter"]) }

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// querySessions returns sessions with started_at in [from, to), oldest first. If
// onlyFor is set, sessions that are hidden from riders are skipped unless they are the
// user's own.
func (h *handler) querySessions(ctx context.Context, stableID, horseID string, from, to time.Time, onlyFor string, limit int) ([]sessionRow, error) {
	q := `
		SELECT s.id::text, s.user_id::text, u.name, s.activity, s.started_at, COALESCE(s.duration_min, 0),
		       s.gait_shares, s.rein_changes, s.feel, s.focus_rating, s.note, s.visible_to_rider,
		       COALESCE(s.load_score, 0)::float8, s.source, s.exercise_id::text, s.distance_m
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.stable_id = $1 AND s.horse_id = $2 AND s.started_at >= $3 AND s.started_at < $4
		  AND ($5 = '' OR s.visible_to_rider OR s.user_id::text = $5)
		ORDER BY s.started_at`
	if limit > 0 { // newest first, used by the list endpoint
		q = strings.Replace(q, "ORDER BY s.started_at", "ORDER BY s.started_at DESC, s.id LIMIT "+strconv.Itoa(limit), 1)
	}
	rows, err := h.deps.Pool.Query(ctx, q, stableID, horseID, from, to, onlyFor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sessionRow
	for rows.Next() {
		var s sessionRow
		var act string
		var gait, rein []byte
		if err := rows.Scan(&s.ID, &s.UserID, &s.UserName, &act, &s.StartedAt, &s.Minutes, &gait, &rein,
			&s.Feel, &s.Focus, &s.Note, &s.Visible, &s.Load, &s.Source, &s.ExerciseID, &s.DistanceM); err != nil {
			return nil, err
		}
		s.Activity = training.Activity(act)
		_ = json.Unmarshal(gait, &s.Gait)
		_ = json.Unmarshal(rein, &s.Rein)
		out = append(out, s)
	}
	return out, rows.Err()
}

// domainSessions converts rows to the pure type; days are the stable-local dates.
func domainSessions(rows []sessionRow, loc *time.Location) []training.Session {
	out := make([]training.Session, 0, len(rows))
	for _, s := range rows {
		out = append(out, training.Session{
			Day: training.Day(s.StartedAt.In(loc)), Activity: s.Activity,
			Minutes: s.Minutes, CanterShare: s.canter(), Load: s.Load,
		})
	}
	return out
}

// --- JSON ---------------------------------------------------------------------------------

type personOut struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ColorKey string `json:"color_key,omitempty"`
}

type sessionOut struct {
	ID             string             `json:"id"`
	HorseID        string             `json:"horse_id"`
	User           personOut          `json:"user"`
	Activity       training.Activity  `json:"activity"`
	Label          string             `json:"label"`
	StartedAt      time.Time          `json:"started_at"`
	Day            string             `json:"day"`
	Minutes        int                `json:"minutes"`
	CanterShare    float64            `json:"canter_share"`
	GaitShares     map[string]float64 `json:"gait_shares"`
	ReinChanges    []reinSegment      `json:"rein_changes"`
	Feel           *string            `json:"feel"`
	FocusRating    *int               `json:"focus_rating"`
	Note           *string            `json:"note"`
	VisibleToRider bool               `json:"visible_to_rider"`
	Load           float64            `json:"load"`
	Intensity      string             `json:"intensity"`
	Source         string             `json:"source"`
	ExerciseID     *string            `json:"exercise_id"`
	DistanceM      *int               `json:"distance_m"`
}

func intensityKey(i training.Intensity) string {
	switch i {
	case training.IntensityLight:
		return "light"
	case training.IntensityMedium:
		return "medium"
	case training.IntensityIntense:
		return "intense"
	}
	return "none"
}

func (s sessionRow) out(horseID string, loc *time.Location) sessionOut {
	gait := s.Gait
	if gait == nil {
		gait = map[string]float64{}
	}
	return sessionOut{
		ID: s.ID, HorseID: horseID, User: personOut{ID: s.UserID, Name: s.UserName},
		Activity: s.Activity, Label: s.Activity.GermanName(),
		StartedAt: s.StartedAt.UTC(), Day: s.StartedAt.In(loc).Format(dateLayout),
		Minutes: s.Minutes, CanterShare: s.canter(), GaitShares: gait, ReinChanges: nonNil(s.Rein),
		Feel: s.Feel, FocusRating: s.Focus, Note: s.Note, VisibleToRider: s.Visible,
		Load: round1(s.Load), Intensity: intensityKey(load.Classify(s.Load)), Source: s.Source,
		ExerciseID: s.ExerciseID, DistanceM: s.DistanceM,
	}
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// --- create -------------------------------------------------------------------------------

type sessionIn struct {
	Activity       string             `json:"activity"`
	Minutes        int                `json:"minutes"`
	CanterShare    *float64           `json:"canter_share"`
	StartedAt      *time.Time         `json:"started_at"`
	GaitShares     map[string]float64 `json:"gait_shares"`
	ReinChanges    []reinSegment      `json:"rein_changes"`
	Feel           string             `json:"feel"`
	FocusRating    *int               `json:"focus_rating"`
	ExerciseID     string             `json:"exercise_id"`
	Note           string             `json:"note"`
	VisibleToRider *bool              `json:"visible_to_rider"`
	DistanceM      *int               `json:"distance_m"`
	Track          []trackPoint       `json:"track"`
	GaitWindows    []gaitWindow       `json:"gait_windows"`
}

type progressionOut struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Discipline string `json:"discipline"`
	Level      string `json:"level"`
}

type createSessionOut struct {
	Session sessionOut `json:"session"`
	// NextProgression is set when the focus rating was "Sitzt" (3) for a library exercise
	// that has a follow-up exercise.
	NextProgression *progressionOut `json:"next_progression"`
}

func (h *handler) createSession(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorize(w, r)
	if !ok {
		return
	}
	if !a.canLog() {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "you may not log sessions for this horse")
		return
	}
	var in sessionIn
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	ctx := r.Context()
	loc := h.location(ctx, a.user.StableID)
	now := h.deps.Now()
	v, err := validateSession(in, now)
	if err != nil {
		invalid(w, err.Error())
		return
	}
	// A GPS track is location data: it needs the location tracking consent (JAN-19). Indoor
	// sessions (gait windows, rein changes) carry no coordinates and need nothing.
	if len(in.Track) > 0 {
		granted, err := privacy.Has(ctx, h.deps.Pool, a.user.ID, privacy.KindLocationTracking)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		if !granted {
			httpx.WriteError(w, http.StatusForbidden, "consent_required", "GPS tracking needs the location tracking consent")
			return
		}
	}
	if v.exerciseID != "" {
		ok, err := h.exerciseVisible(ctx, a.user.StableID, v.exerciseID)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		if !ok {
			invalid(w, "exercise_id is not an exercise of the library")
			return
		}
	}

	score := load.Score(in.Minutes, training.Activity(in.Activity), v.canter)
	gait, _ := json.Marshal(v.gait)
	var track []byte // nil = SQL NULL: the session has no track
	if len(in.Track) > 0 {
		track, _ = json.Marshal(in.Track)
	}
	rein, _ := json.Marshal(nonNil(in.ReinChanges))
	day := training.Day(v.startedAt.In(loc))

	tx, err := h.deps.Pool.Begin(ctx)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO sessions (stable_id, horse_id, user_id, activity, started_at, duration_min, gait_shares,
		                      rein_changes, feel, focus_rating, note, visible_to_rider, load_score, source,
		                      exercise_id, distance_m, track)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), $10, NULLIF($11, ''), $12, $13, $14, NULLIF($15, '')::uuid, $16, $17)
		RETURNING id::text`,
		a.user.StableID, a.horseID, a.user.ID, in.Activity, v.startedAt, in.Minutes, gait, rein, in.Feel,
		in.FocusRating, in.Note, v.visible, score, v.source, v.exerciseID, in.DistanceM, track).Scan(&id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if len(in.GaitWindows) > 0 {
		windows, _ := json.Marshal(in.GaitWindows)
		if _, err := tx.Exec(ctx, `
			INSERT INTO gait_windows (session_id, stable_id, window_count, windows)
			VALUES ($1, $2, $3, $4)`, id, a.user.StableID, len(in.GaitWindows), windows); err != nil {
			h.fail(w, r, err)
			return
		}
	}
	// The matching week slot is done now (created if nobody had planned the day).
	if _, err := tx.Exec(ctx, `
		INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status)
		VALUES ($1, $2, $3, $4, $5, 'done')
		ON CONFLICT (horse_id, day) DO UPDATE SET
			status = 'done',
			user_id = COALESCE(week_slots.user_id, EXCLUDED.user_id),
			activity = COALESCE(week_slots.activity, EXCLUDED.activity)`,
		a.user.StableID, a.horseID, day, a.user.ID, in.Activity); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		h.fail(w, r, err)
		return
	}

	rows, err := h.querySessions(ctx, a.user.StableID, a.horseID, v.startedAt.Add(-time.Second), v.startedAt.Add(time.Second), "", 0)
	var created sessionRow
	for _, s := range rows {
		if s.ID == id {
			created = s
		}
	}
	if err != nil || created.ID == "" {
		h.fail(w, r, fmt.Errorf("reload session %s: %v", id, err))
		return
	}
	out := createSessionOut{Session: created.out(a.horseID, loc)}
	if in.FocusRating != nil && *in.FocusRating == 3 && v.exerciseID != "" {
		if out.NextProgression, err = h.nextProgression(ctx, a.user.StableID, v.exerciseID); err != nil {
			h.fail(w, r, err)
			return
		}
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

type validSession struct {
	startedAt  time.Time
	canter     float64
	gait       map[string]float64
	visible    bool
	source     string
	exerciseID string
}

func validateSession(in sessionIn, now time.Time) (validSession, error) {
	v := validSession{visible: true, source: "quick", gait: map[string]float64{}}
	if !training.Activity(in.Activity).Valid() {
		return v, errors.New("activity must be one of hall, arena, hack, lunge, jumping, groundwork, walker")
	}
	if in.Minutes < 1 || in.Minutes > maxSessionMinutes {
		return v, fmt.Errorf("minutes must be between 1 and %d", maxSessionMinutes)
	}
	v.startedAt = now
	if in.StartedAt != nil {
		v.startedAt = *in.StartedAt
		if v.startedAt.After(now.Add(futureSlack)) {
			return v, errors.New("started_at must not be in the future")
		}
		if v.startedAt.Before(now.AddDate(0, 0, -maxBackdateDays)) {
			return v, fmt.Errorf("started_at must be within the last %d days", maxBackdateDays)
		}
	}

	if in.CanterShare != nil {
		if *in.CanterShare < 0 || *in.CanterShare > 1 {
			return v, errors.New("canter_share must be between 0 and 1")
		}
		v.canter = *in.CanterShare
		v.gait["canter"] = v.canter
	}
	if in.GaitShares != nil {
		sum := 0.0
		for g, share := range in.GaitShares {
			switch g {
			case "halt", "walk", "trot", "canter":
			default:
				return v, fmt.Errorf("unknown gait %q", g)
			}
			if share < 0 || share > 1 || math.IsNaN(share) {
				return v, errors.New("gait shares must be between 0 and 1")
			}
			sum += share
		}
		if sum > 1.001 {
			return v, errors.New("gait shares must not add up to more than 1")
		}
		v.gait = in.GaitShares
		v.canter = clamp01(in.GaitShares["canter"])
		v.source = "tracked"
	}
	for _, seg := range in.ReinChanges {
		if seg.Rein != "left" && seg.Rein != "right" {
			return v, errors.New("rein must be left or right")
		}
		if seg.Minutes < 0 || seg.Minutes > maxSessionMinutes || math.IsNaN(seg.Minutes) {
			return v, errors.New("rein minutes are out of range")
		}
	}
	if len(in.ReinChanges) > 200 {
		return v, errors.New("too many rein segments")
	}
	if in.ReinChanges != nil || in.DistanceM != nil || len(in.Track) > 0 || len(in.GaitWindows) > 0 {
		v.source = "tracked"
	}
	if err := validateTrack(in.Track); err != nil {
		return v, err
	}
	if err := validateGaitWindows(in.GaitWindows); err != nil {
		return v, err
	}
	if in.DistanceM != nil && (*in.DistanceM < 0 || *in.DistanceM > 500_000) {
		return v, errors.New("distance_m is out of range")
	}
	if _, ok := feelLabels[in.Feel]; in.Feel != "" && !ok {
		return v, errors.New("feel must be fresh, loose, tired or tense")
	}
	if in.FocusRating != nil && (*in.FocusRating < 1 || *in.FocusRating > 3) {
		return v, errors.New("focus_rating must be 1, 2 or 3")
	}
	if len([]rune(in.Note)) > 2000 {
		return v, errors.New("note is too long (max 2000 characters)")
	}
	if in.VisibleToRider != nil {
		v.visible = *in.VisibleToRider
	}
	v.exerciseID = in.ExerciseID
	return v, nil
}

// --- list ---------------------------------------------------------------------------------

func (h *handler) listSessions(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorize(w, r)
	if !ok {
		return
	}
	limit := 30
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 100 {
			invalid(w, "limit must be between 1 and 100")
			return
		}
		limit = n
	}
	before := h.deps.Now().Add(futureSlack)
	if s := r.URL.Query().Get("before"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			invalid(w, "before must be an RFC 3339 timestamp")
			return
		}
		before = t
	}
	onlyFor := ""
	if !a.manage { // riders see sessions marked visible and their own
		onlyFor = a.user.ID
	}
	ctx := r.Context()
	rows, err := h.querySessions(ctx, a.user.StableID, a.horseID, time.Unix(0, 0), before, onlyFor, limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	loc := h.location(ctx, a.user.StableID)
	out := make([]sessionOut, 0, len(rows))
	for _, s := range rows {
		out = append(out, s.out(a.horseID, loc))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sessions": out})
}
