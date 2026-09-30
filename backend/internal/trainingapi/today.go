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

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/stables"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/load"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
)

type recommendationOut struct {
	Activity       training.Activity `json:"activity"`
	Label          string            `json:"label"`
	Minutes        int               `json:"minutes"`
	Intensity      string            `json:"intensity"`
	IntensityLabel string            `json:"intensity_label"`
	Reason         string            `json:"reason"`
	Note           string            `json:"note,omitempty"`
	// Exercise is the library exercise to practise ("Ablauf"), if the activity has any.
	Exercise *exerciseOut `json:"exercise,omitempty"`
}

type hiddenOut struct {
	Activity training.Activity `json:"activity"`
	Label    string            `json:"label"`
	Reason   string            `json:"reason"`
}

// dotOut is one day of the 7-day status: trained, rest or nothing.
type dotOut struct {
	Date      string `json:"date"`
	Kind      string `json:"kind"`      // trained, rest, nothing
	Intensity string `json:"intensity"` // none, light, medium, intense (day load)
	IsToday   bool   `json:"is_today"`
}

type rehaOut struct {
	PlanID     string `json:"plan_id"`
	Phase      string `json:"phase"`
	PhaseIndex int    `json:"phase_index"` // 1-based
	Phases     int    `json:"phases"`
	Activity   string `json:"activity"`
	MinMinutes int    `json:"min_minutes"`
	MaxMinutes int    `json:"max_minutes"`
	Conditions string `json:"conditions,omitempty"`
}

type todayOut struct {
	HorseID         string              `json:"horse_id"`
	HorseName       string              `json:"horse_name"`
	Date            string              `json:"date"`
	HasProfile      bool                `json:"has_profile"`
	Status          training.Status     `json:"status"`
	Context         string              `json:"context"`
	Recommendations []recommendationOut `json:"recommendations"`
	Hidden          []hiddenOut         `json:"hidden"`
	Week            []dotOut            `json:"week"`
	Reha            *rehaOut            `json:"reha"`
	CanLog          bool                `json:"can_log"`
	CanEdit         bool                `json:"can_edit"`
}

// today: GET /api/v1/horses/{id}/today?minutes=45
func (h *handler) today(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorize(w, r)
	if !ok {
		return
	}
	minutes := 0
	if s := r.URL.Query().Get("minutes"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || n > 300 {
			invalid(w, "minutes must be between 0 and 300")
			return
		}
		minutes = n
	}
	ctx := r.Context()
	stableID := a.user.StableID
	loc := h.location(ctx, stableID)
	today := training.Day(h.deps.Now().In(loc))

	prof, err := h.loadProfile(ctx, stableID, a.horseID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	// Last 14 days including today; the recommender ignores older ones.
	rows, err := h.querySessions(ctx, stableID, a.horseID,
		localMidnight(today.AddDate(0, 0, -13), loc), localMidnight(today.AddDate(0, 0, 1), loc), "", 0)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	sessions := domainSessions(rows, loc)

	var wx *recommend.Weather
	snap, err := (&weather.Store{Pool: h.deps.Pool}).Latest(ctx, stableID, today)
	switch {
	case err == nil:
		wx = &recommend.Weather{Rain: snap.WillRain, TempC: snap.NightMinC}
	case !errors.Is(err, weather.ErrNotFound):
		h.fail(w, r, err)
		return
	}
	ground, err := stables.GetGroundCondition(ctx, h.deps.Pool, stableID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	plan, err := h.activeRehaPhase(ctx, stableID, a.horseID, today)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	in := recommend.Input{
		Today: today, Profile: prof.domain(a.horseName), Recent: sessions,
		Weather: wx, Ground: recommend.Ground(ground.Condition), AvailableMinutes: minutes,
		Role: recommend.RoleOwner, RiderName: a.user.Name,
	}
	if !a.manage {
		in.Role = recommend.RoleRider
		in.Rider = prof.riderRules(a.user.ID, a.rules)
	}
	if plan != nil {
		in.Reha = plan.phase
	}
	res := recommend.Recommend(in)

	out := todayOut{
		HorseID: a.horseID, HorseName: a.horseName, Date: today.Format(dateLayout),
		HasProfile: prof.exists, Status: prof.status, CanLog: a.canLog(), CanEdit: a.manage,
		Recommendations: []recommendationOut{}, Hidden: []hiddenOut{},
	}
	if plan != nil {
		out.Reha = &plan.out
	}
	for _, rec := range res.Recommendations {
		ro := recommendationOut{
			Activity: rec.Activity, Label: rec.Activity.GermanName(), Minutes: rec.Minutes,
			Intensity: intensityKey(rec.Intensity), IntensityLabel: rec.Intensity.Label(),
			Reason: rec.Reason, Note: rec.Note,
		}
		if ro.Exercise, err = h.suggestExercise(ctx, stableID, a.horseID, exerciseDiscipline(rec.Activity, prof.discipline)); err != nil {
			h.fail(w, r, err)
			return
		}
		out.Recommendations = append(out.Recommendations, ro)
	}
	for _, hd := range res.Hidden {
		out.Hidden = append(out.Hidden, hiddenOut{Activity: hd.Activity, Label: hd.Activity.GermanName(), Reason: hd.Reason})
	}
	slots, err := h.slotsBetween(ctx, stableID, a.horseID, today.AddDate(0, 0, -6), today)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out.Week = weekDots(today, sessions, prof, slots)
	out.Context = contextLine(contextInput{
		Weather: wx, Ground: ground.Condition, Today: today, Shows: in.Profile.Shows, Reha: out.Reha,
	})
	httpx.WriteJSON(w, http.StatusOK, out)
}

// weekDots builds the last seven days (today last): trained when a session exists, rest
// for a planned rest day or the mandatory rest day after a show, otherwise nothing.
func weekDots(today time.Time, sessions []training.Session, prof storedProfile, slots map[string]slotRow) []dotOut {
	dots := make([]dotOut, 0, 7)
	shows := prof.domain("").Shows
	for i := 6; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		d := dotOut{Date: day.Format(dateLayout), Kind: "nothing", IsToday: i == 0}
		dayLoad := load.DayLoad(sessions, day)
		d.Intensity = intensityKey(load.Classify(dayLoad))
		switch {
		case dayLoad > 0 || slots[d.Date].Status == "done":
			d.Kind = "trained"
		case slots[d.Date].Status == "rest" || restAfterShow(prof.rhythm, shows, day):
			d.Kind = "rest"
		}
		dots = append(dots, d)
	}
	return dots
}

// restAfterShow reports whether day is the mandatory rest day after a show.
func restAfterShow(r training.Rhythm, shows []training.Show, day time.Time) bool {
	if !r.RestAfterShow {
		return false
	}
	for _, s := range shows {
		if training.DaysBetween(s.Date, day) == 1 {
			return true
		}
	}
	return false
}

type contextInput struct {
	Weather *recommend.Weather
	Ground  string
	Today   time.Time
	Shows   []training.Show
	Reha    *rehaOut
}

// contextLine is the short line under the status card, e.g. "Regen, 6 °C · Turnier in 3 Tagen".
// The temperature is the forecast night minimum of the latest weather snapshot.
func contextLine(in contextInput) string {
	var parts []string
	if in.Weather != nil {
		kind := "Trocken"
		if in.Weather.Rain {
			kind = "Regen"
		}
		parts = append(parts, fmt.Sprintf("%s, %d °C", kind, int(math.Round(in.Weather.TempC))))
	}
	switch in.Ground {
	case stables.GroundWet:
		parts = append(parts, "Boden nass")
	case stables.GroundMuddy:
		parts = append(parts, "Boden matschig")
	case stables.GroundFrozen:
		parts = append(parts, "Boden gefroren")
	}
	best, name := -1, ""
	for _, s := range in.Shows {
		if d := training.DaysBetween(in.Today, s.Date); d >= 0 && d <= 14 && (best < 0 || d < best) {
			best, name = d, s.Name
		}
	}
	switch {
	case best == 0:
		parts = append(parts, name+" heute")
	case best == 1:
		parts = append(parts, name+" morgen")
	case best > 1:
		parts = append(parts, fmt.Sprintf("%s in %d Tagen", name, best))
	}
	if in.Reha != nil {
		parts = append(parts, fmt.Sprintf("Reha: %s", in.Reha.Phase))
	}
	return strings.Join(parts, " · ")
}

// --- reha ---------------------------------------------------------------------------------

// rehaPhaseIn is one entry of reha_plans.phases. Duration is days, or weeks x 7.
type rehaPhaseIn struct {
	Name         string `json:"name"`
	Days         int    `json:"days"`
	DurationDays int    `json:"duration_days"`
	Weeks        int    `json:"weeks"`
	Activity     string `json:"activity"`
	MinMinutes   int    `json:"min_minutes"`
	MaxMinutes   int    `json:"max_minutes"`
	Conditions   string `json:"conditions"`
}

func (p rehaPhaseIn) length() int {
	switch {
	case p.Days > 0:
		return p.Days
	case p.DurationDays > 0:
		return p.DurationDays
	}
	return p.Weeks * 7
}

type activeReha struct {
	phase *recommend.RehaPhase
	out   rehaOut
}

// activeRehaPhase returns the phase of the horse's active reha plan that covers today: the
// phases run one after the other starting at start_date. It returns nil without an active
// plan, before the start, after the last phase, or when the phase has no valid activity.
func (h *handler) activeRehaPhase(ctx context.Context, stableID, horseID string, today time.Time) (*activeReha, error) {
	var (
		id     string
		start  time.Time
		phases []byte
	)
	err := h.deps.Pool.QueryRow(ctx, `
		SELECT id::text, start_date, phases FROM reha_plans
		WHERE stable_id = $1 AND horse_id = $2 AND active`, stableID, horseID).Scan(&id, &start, &phases)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var list []rehaPhaseIn
	if err := json.Unmarshal(phases, &list); err != nil {
		return nil, nil
	}
	p, idx := currentPhase(list, training.DaysBetween(start, today))
	if p == nil || !training.Activity(p.Activity).Valid() {
		return nil, nil
	}
	name := p.Name
	if name == "" {
		name = fmt.Sprintf("Phase %d", idx+1)
	}
	minM, maxM := p.MinMinutes, p.MaxMinutes
	if maxM <= 0 {
		maxM = 20
	}
	if minM <= 0 || minM > maxM {
		minM = maxM
	}
	act := training.Activity(p.Activity)
	return &activeReha{
		phase: &recommend.RehaPhase{Name: name, Activity: act, MinMinutes: minM, MaxMinutes: maxM, Conditions: p.Conditions},
		out: rehaOut{PlanID: id, Phase: name, PhaseIndex: idx + 1, Phases: len(list), Activity: p.Activity,
			MinMinutes: minM, MaxMinutes: maxM, Conditions: p.Conditions},
	}, nil
}

// currentPhase finds the phase that covers the given number of days since the plan start.
func currentPhase(phases []rehaPhaseIn, elapsedDays int) (*rehaPhaseIn, int) {
	if elapsedDays < 0 {
		return nil, 0
	}
	end := 0
	for i := range phases {
		end += phases[i].length()
		if elapsedDays < end {
			return &phases[i], i
		}
	}
	return nil, 0
}
