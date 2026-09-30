package trainingapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// Disciplines accepted in the profile (also used by the recommender).
var disciplines = []string{"dressage", "jumping", "eventing", "leisure", "western", "young_horse"}

// showDTO is a show as stored in training_profiles.shows: the pure training.Show plus
// optional info for the week view.
type showDTO struct {
	Date    string `json:"date"` // YYYY-MM-DD
	Name    string `json:"name"`
	Classes string `json:"classes,omitempty"`
	Helper  string `json:"helper,omitempty"`
}

// rbRule is one entry of training_profiles.rb_rules (key: rider user id).
type rbRule struct {
	AllowedActivities []training.Activity `json:"allowed_activities"`
	MaxIntensity      string              `json:"max_intensity"` // any, light, medium, intense
	MayHackAlone      bool                `json:"may_hack_alone"`
	MayRideShows      bool                `json:"may_ride_shows"`
}

// intensityKeys maps the API names of rider max intensity to the domain type.
var intensityKeys = map[string]training.Intensity{
	"any": training.IntensityNone, "light": training.IntensityLight,
	"medium": training.IntensityMedium, "intense": training.IntensityIntense,
}

// storedProfile is a training_profiles row decoded for use.
type storedProfile struct {
	exists     bool
	discipline string
	level      string
	allowed    []training.AllowedActivity
	shows      []showDTO
	seasonEnd  *time.Time
	rhythm     training.Rhythm
	rbRules    map[string]rbRule
	status     training.Status
}

func emptyProfile() storedProfile {
	return storedProfile{rhythm: training.DefaultRhythm(), status: training.StatusFit, rbRules: map[string]rbRule{}}
}

// loadProfile reads the profile of a horse; a horse without a row gets the defaults
// (exists=false). The rhythm is decoded on top of training.DefaultRhythm() so that the
// mandatory rest day after a show stays on unless the stored value says otherwise.
func (h *handler) loadProfile(ctx context.Context, stableID, horseID string) (storedProfile, error) {
	p := emptyProfile()
	var (
		disc, level                    *string
		allowed, shows, rhythm, rbJSON []byte
		seasonEnd                      *time.Time
		status                         string
	)
	err := h.deps.Pool.QueryRow(ctx, `
		SELECT discipline, level, allowed_activities, shows, season_end, rhythm, rb_rules, status
		FROM training_profiles WHERE stable_id = $1 AND horse_id = $2`, stableID, horseID).
		Scan(&disc, &level, &allowed, &shows, &seasonEnd, &rhythm, &rbJSON, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	p.exists = true
	if disc != nil {
		p.discipline = *disc
	}
	if level != nil {
		p.level = *level
	}
	p.seasonEnd = seasonEnd
	p.status = training.Status(status)
	_ = json.Unmarshal(allowed, &p.allowed)
	_ = json.Unmarshal(shows, &p.shows)
	_ = json.Unmarshal(rhythm, &p.rhythm) // starts from DefaultRhythm, missing keys keep the default
	if err := json.Unmarshal(rbJSON, &p.rbRules); err != nil || p.rbRules == nil {
		p.rbRules = map[string]rbRule{}
	}
	for i := range p.shows {
		if len(p.shows[i].Date) > 10 { // tolerate RFC 3339 written by other code
			p.shows[i].Date = p.shows[i].Date[:10]
		}
	}
	return p, nil
}

// domain converts the stored profile to the pure type used by the recommender.
func (p storedProfile) domain(horseName string) training.Profile {
	out := training.Profile{
		HorseName: horseName, Discipline: p.discipline, Level: p.level,
		Allowed: p.allowed, Rhythm: p.rhythm, Status: p.status, SeasonEnd: p.seasonEnd,
	}
	for _, s := range p.shows {
		if d, err := time.Parse(dateLayout, s.Date); err == nil {
			out.Shows = append(out.Shows, training.Show{Date: d, Name: s.Name})
		}
	}
	return out
}

// riderRules maps the stored per-rider rules and the rider's rule keys to
// training.RiderRules. A rider without an entry in rb_rules gets nil, which the
// recommender treats as "sees nothing" (fail closed).
//
//   - allowed_activities, max_intensity: from rb_rules[userID]
//   - MayHackAlone: rb_rules.may_hack_alone or the rule key hack_alone
//   - MayRideShows: rb_rules.may_ride_shows or the rule key shows
func (p storedProfile) riderRules(userID string, keys []string) *training.RiderRules {
	e, ok := p.rbRules[userID]
	if !ok {
		return nil
	}
	r := &training.RiderRules{
		MaxIntensity: intensityKeys[e.MaxIntensity],
		MayHackAlone: e.MayHackAlone || slices.Contains(keys, RuleHackAlone),
		MayRideShows: e.MayRideShows || slices.Contains(keys, RuleShows),
	}
	for _, a := range e.AllowedActivities {
		if a.Valid() {
			r.AllowedActivities = append(r.AllowedActivities, a)
		}
	}
	return r
}

// --- HTTP ---------------------------------------------------------------------------------

type rhythmOut struct {
	SessionsMin   int  `json:"sessions_min"`
	SessionsMax   int  `json:"sessions_max"`
	RestDaysMin   int  `json:"rest_days_min"`
	RestDaysMax   int  `json:"rest_days_max"`
	MaxMinutes    int  `json:"max_minutes"`
	RestAfterShow bool `json:"rest_after_show"`
}

type riderRuleOut struct {
	UserID            string              `json:"user_id"`
	Name              string              `json:"name"`
	AllowedActivities []training.Activity `json:"allowed_activities"`
	MaxIntensity      string              `json:"max_intensity"`
	MayHackAlone      bool                `json:"may_hack_alone"`
	MayRideShows      bool                `json:"may_ride_shows"`
}

type profileOut struct {
	HorseID           string                     `json:"horse_id"`
	HorseName         string                     `json:"horse_name"`
	Exists            bool                       `json:"exists"`
	CanEdit           bool                       `json:"can_edit"`
	Discipline        string                     `json:"discipline"`
	Level             string                     `json:"level"`
	AllowedActivities []training.AllowedActivity `json:"allowed_activities"`
	Shows             []showDTO                  `json:"shows"`
	SeasonEnd         *string                    `json:"season_end"`
	Rhythm            rhythmOut                  `json:"rhythm"`
	Status            training.Status            `json:"status"`
	// RiderRules has one entry per rider of the horse for owners and admins, and only the
	// caller's own entry for riders.
	RiderRules []riderRuleOut `json:"rider_rules"`
}

func (h *handler) getProfile(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorize(w, r)
	if !ok {
		return
	}
	out, err := h.profileOut(r.Context(), a)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *handler) profileOut(ctx context.Context, a access) (profileOut, error) {
	p, err := h.loadProfile(ctx, a.user.StableID, a.horseID)
	if err != nil {
		return profileOut{}, err
	}
	out := profileOut{
		HorseID: a.horseID, HorseName: a.horseName, Exists: p.exists, CanEdit: a.manage,
		Discipline: p.discipline, Level: p.level,
		AllowedActivities: nonNil(p.allowed), Shows: nonNil(p.shows),
		Rhythm: rhythmOut(p.rhythm), Status: p.status, RiderRules: []riderRuleOut{},
	}
	if p.seasonEnd != nil {
		s := p.seasonEnd.Format(dateLayout)
		out.SeasonEnd = &s
	}
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT hr.user_id::text, u.name FROM horse_riders hr JOIN users u ON u.id = hr.user_id
		WHERE hr.stable_id = $1 AND hr.horse_id = $2 ORDER BY u.name, hr.user_id`, a.user.StableID, a.horseID)
	if err != nil {
		return profileOut{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return profileOut{}, err
		}
		if !a.manage && id != a.user.ID {
			continue
		}
		e := p.rbRules[id]
		max := e.MaxIntensity
		if _, known := intensityKeys[max]; !known {
			max = "any"
		}
		out.RiderRules = append(out.RiderRules, riderRuleOut{
			UserID: id, Name: name, AllowedActivities: nonNil(e.AllowedActivities), MaxIntensity: max,
			MayHackAlone: e.MayHackAlone, MayRideShows: e.MayRideShows,
		})
	}
	return out, rows.Err()
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

type rhythmIn struct {
	SessionsMin   *int  `json:"sessions_min"`
	SessionsMax   *int  `json:"sessions_max"`
	RestDaysMin   *int  `json:"rest_days_min"`
	RestDaysMax   *int  `json:"rest_days_max"`
	MaxMinutes    *int  `json:"max_minutes"`
	RestAfterShow *bool `json:"rest_after_show"`
}

type riderRuleIn struct {
	UserID            string              `json:"user_id"`
	AllowedActivities []training.Activity `json:"allowed_activities"`
	MaxIntensity      string              `json:"max_intensity"`
	MayHackAlone      bool                `json:"may_hack_alone"`
	MayRideShows      bool                `json:"may_ride_shows"`
}

type profileIn struct {
	Discipline        string                     `json:"discipline"`
	Level             string                     `json:"level"`
	AllowedActivities []training.AllowedActivity `json:"allowed_activities"`
	Shows             []showDTO                  `json:"shows"`
	SeasonEnd         *string                    `json:"season_end"`
	Rhythm            *rhythmIn                  `json:"rhythm"`
	Status            string                     `json:"status"`
	// RiderRules nil keeps the stored rules; an empty list removes them all.
	RiderRules []riderRuleIn `json:"rider_rules"`
}

func (h *handler) putProfile(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorize(w, r)
	if !ok {
		return
	}
	if !a.manage {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner can change the training profile")
		return
	}
	var in profileIn
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	ctx := r.Context()
	riders, err := h.riderIDs(ctx, a)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	v, err := validateProfile(in, riders)
	if err != nil {
		invalid(w, err.Error())
		return
	}
	var rb []byte // nil keeps the stored rules
	if v.rbRules != nil {
		rb, _ = json.Marshal(v.rbRules)
	}
	allowed, _ := json.Marshal(v.allowed)
	shows, _ := json.Marshal(v.shows)
	rhythm, _ := json.Marshal(v.rhythm)
	_, err = h.deps.Pool.Exec(ctx, `
		INSERT INTO training_profiles
			(stable_id, horse_id, discipline, level, allowed_activities, shows, season_end, rhythm, rb_rules, status)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, COALESCE($9::jsonb, '{}'), $10)
		ON CONFLICT (horse_id) DO UPDATE SET
			discipline = EXCLUDED.discipline, level = EXCLUDED.level,
			allowed_activities = EXCLUDED.allowed_activities, shows = EXCLUDED.shows,
			season_end = EXCLUDED.season_end, rhythm = EXCLUDED.rhythm,
			rb_rules = COALESCE($9::jsonb, training_profiles.rb_rules), status = EXCLUDED.status`,
		a.user.StableID, a.horseID, in.Discipline, v.level, allowed, shows, v.seasonEnd, rhythm, rb, string(v.status))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.profileOut(ctx, a)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// riderIDs returns the user ids of the horse's riders.
func (h *handler) riderIDs(ctx context.Context, a access) ([]string, error) {
	rows, err := h.deps.Pool.Query(ctx, `SELECT user_id::text FROM horse_riders WHERE stable_id = $1 AND horse_id = $2`,
		a.user.StableID, a.horseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type validProfile struct {
	level     string
	allowed   []training.AllowedActivity
	shows     []showDTO
	seasonEnd *time.Time
	rhythm    training.Rhythm
	status    training.Status
	rbRules   map[string]rbRule // nil: keep
}

func validateProfile(in profileIn, riders []string) (validProfile, error) {
	var v validProfile
	if !slices.Contains(disciplines, in.Discipline) {
		return v, fmt.Errorf("discipline must be one of %s", strings.Join(disciplines, ", "))
	}
	v.level = strings.TrimSpace(in.Level)
	if len([]rune(v.level)) > 40 {
		return v, errors.New("level is too long (max 40 characters)")
	}

	seen := map[training.Activity]bool{}
	for _, e := range in.AllowedActivities {
		if !e.Activity.Valid() {
			return v, fmt.Errorf("unknown activity %q", e.Activity)
		}
		if seen[e.Activity] {
			return v, fmt.Errorf("activity %q is listed twice", e.Activity)
		}
		seen[e.Activity] = true
		switch e.Mode {
		case training.ModeOn, training.ModeOff:
			e.Note = ""
		case training.ModeConditional:
			e.Note = strings.TrimSpace(e.Note)
			if e.Note == "" {
				return v, fmt.Errorf("activity %q is conditional and needs a note", e.Activity)
			}
			if len([]rune(e.Note)) > 200 {
				return v, errors.New("note is too long (max 200 characters)")
			}
		default:
			return v, fmt.Errorf("mode of %q must be on, off or conditional", e.Activity)
		}
		v.allowed = append(v.allowed, e)
	}
	if v.allowed == nil {
		v.allowed = []training.AllowedActivity{}
	}

	if len(in.Shows) > 50 {
		return v, errors.New("too many shows (max 50)")
	}
	v.shows = []showDTO{}
	for _, s := range in.Shows {
		if _, err := time.Parse(dateLayout, s.Date); err != nil {
			return v, fmt.Errorf("show date %q must be YYYY-MM-DD", s.Date)
		}
		s.Name = strings.TrimSpace(s.Name)
		if s.Name == "" || len([]rune(s.Name)) > 80 {
			return v, errors.New("show name is required (max 80 characters)")
		}
		if len([]rune(s.Classes)) > 200 || len([]rune(s.Helper)) > 200 {
			return v, errors.New("show classes and helper are too long (max 200 characters)")
		}
		v.shows = append(v.shows, s)
	}
	slices.SortStableFunc(v.shows, func(x, y showDTO) int { return strings.Compare(x.Date, y.Date) })

	if in.SeasonEnd != nil && *in.SeasonEnd != "" {
		d, err := time.Parse(dateLayout, *in.SeasonEnd)
		if err != nil {
			return v, errors.New("season_end must be YYYY-MM-DD")
		}
		v.seasonEnd = &d
	}

	v.rhythm = training.DefaultRhythm()
	if r := in.Rhythm; r != nil {
		set := func(dst *int, src *int) {
			if src != nil {
				*dst = *src
			}
		}
		set(&v.rhythm.SessionsMin, r.SessionsMin)
		set(&v.rhythm.SessionsMax, r.SessionsMax)
		set(&v.rhythm.RestDaysMin, r.RestDaysMin)
		set(&v.rhythm.RestDaysMax, r.RestDaysMax)
		set(&v.rhythm.MaxMinutes, r.MaxMinutes)
		if r.RestAfterShow != nil {
			v.rhythm.RestAfterShow = *r.RestAfterShow
		}
	}
	rh := v.rhythm
	switch {
	case rh.SessionsMin < 1 || rh.SessionsMax > 7 || rh.SessionsMin > rh.SessionsMax:
		return v, errors.New("sessions per week must satisfy 1 <= min <= max <= 7")
	case rh.RestDaysMin < 0 || rh.RestDaysMax > 6 || rh.RestDaysMin > rh.RestDaysMax:
		return v, errors.New("rest days per week must satisfy 0 <= min <= max <= 6")
	case rh.SessionsMin+rh.RestDaysMin > 7:
		return v, errors.New("sessions and rest days do not fit into a week")
	case rh.MaxMinutes < 0 || rh.MaxMinutes > 240:
		return v, errors.New("max_minutes must be between 0 and 240")
	}

	switch training.Status(in.Status) {
	case "":
		v.status = training.StatusFit
	case training.StatusFit, training.StatusReha, training.StatusPause:
		v.status = training.Status(in.Status)
	default:
		return v, errors.New("status must be fit, reha or pause")
	}

	if in.RiderRules != nil {
		v.rbRules = map[string]rbRule{}
		for _, rr := range in.RiderRules {
			if !slices.Contains(riders, rr.UserID) {
				return v, fmt.Errorf("user %q is not a rider of this horse", rr.UserID)
			}
			if _, dup := v.rbRules[rr.UserID]; dup {
				return v, errors.New("rider listed twice")
			}
			max := rr.MaxIntensity
			if max == "" {
				max = "any"
			}
			if _, known := intensityKeys[max]; !known {
				return v, errors.New("max_intensity must be any, light, medium or intense")
			}
			acts := []training.Activity{}
			for _, act := range rr.AllowedActivities {
				if !act.Valid() {
					return v, fmt.Errorf("unknown activity %q", act)
				}
				if !slices.Contains(acts, act) {
					acts = append(acts, act)
				}
			}
			v.rbRules[rr.UserID] = rbRule{AllowedActivities: acts, MaxIntensity: max, MayHackAlone: rr.MayHackAlone, MayRideShows: rr.MayRideShows}
		}
	}
	return v, nil
}
