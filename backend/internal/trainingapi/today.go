package trainingapi

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/reha"
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
	// Exercise is the main exercise of the unit to practise ("Ablauf"), if the unit has one.
	Exercise *exerciseOut `json:"exercise,omitempty"`
	// Focus is the focus of the unit with the reason for it, Unit its three parts (Lösen,
	// Arbeiten, Entspannen); both are null for the walker and on a show day.
	Focus *unitFocusOut `json:"focus"`
	Unit  *unitOut      `json:"unit"`
	// EquipmentOK is the day's override of the set-up (null: the location's default),
	// EquipmentNote says in German that the unit is planned without set-up.
	EquipmentOK   *bool   `json:"equipment_ok"`
	EquipmentNote *string `json:"equipment_note"`
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
	// Minutes is today's allowed duration (ramp between min and max), 0 for a rest phase.
	Minutes int    `json:"minutes"`
	Rest    bool   `json:"rest"`
	Done    bool   `json:"done"` // "Heute erledigt" was tapped
	Text    string `json:"text"`
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
	// Phase is the horse's phase in the training scale, RestartWeek the week of its return to work
	// (0 when it is not returning), Block the week's place in the four-week blocks (null without
	// cycle_start) and Progress how far the horse is on the way to its goal level (null without).
	Phase       training.HorsePhase `json:"phase"`
	RestartWeek int                 `json:"restart_week"`
	Block       *blockOut           `json:"block"`
	Progress    *progressBrief      `json:"progress"`
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
	domain := prof.domain(a.horseName)
	pc, err := h.loadPlanCtx(ctx, a, prof, domain, mondayOf(today), today, loc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	// From the Monday three weeks before this week's up to today: the recommender scores the last
	// 14 days, the load limit of the week structure needs the two weeks before (JAN-93) and the
	// Entlastungswoche three.
	from := mondayOf(today).AddDate(0, 0, -historyDays)
	rows, err := h.querySessions(ctx, stableID, a.horseID, localMidnight(from, loc), localMidnight(today.AddDate(0, 0, 1), loc), "", 0)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	slots, err := h.slotsBetween(ctx, stableID, a.horseID, from, today)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	sessions := domainSessions(rows, loc)
	attachSlots(sessions, slots)

	wx, ground, err := h.conditions(ctx, stableID, today)
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
		Today: today, Profile: domain, Recent: sessions,
		Weather: wx, Ground: recommend.Ground(ground), AvailableMinutes: minutes,
		Role: recommend.RoleOwner, RiderName: a.user.Name,
		AgeYears: pc.age, Level: pc.level, Block: pc.block, RestartFrom: pc.restart, Exercises: pc.exercises,
		RiderFocuses: pc.riders[a.user.ID], Equipment: pc.equipment, EquipmentOK: slots[today.Format(dateLayout)].EquipmentOK,
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
		Phase: training.Phase(pc.level, pc.age, prof.status), RestartWeek: recommend.RestartWeek(pc.restart, today),
		Block: blockOutOf(pc.block),
	}
	if plan != nil {
		out.Reha = &plan.out
	}
	if prog, _, err := h.progress(ctx, a, prof, today); err != nil {
		h.fail(w, r, err)
		return
	} else if prog != nil {
		out.Progress = prog.brief()
	}
	// On a show day the show is the work of the day: no unit or exercise on top.
	showToday := isShowDay(in.Profile.Shows, today)
	byID := pc.lib.byID()
	for _, rec := range res.Recommendations {
		ro := recommendationOut{
			Activity: rec.Activity, Label: rec.Activity.GermanName(), Minutes: rec.Minutes,
			Intensity: intensityKey(rec.Intensity), IntensityLabel: rec.Intensity.Label(),
			Reason: rec.Reason, Note: rec.Note,
		}
		if !showToday {
			if u := recommend.PlanUnit(in, rec.Activity, rec.Minutes, recommend.Hints{}); !u.Empty() {
				ro.Focus, ro.Unit, ro.EquipmentNote = focusOfUnit(u), unitOutOfPlan(u), noteOut(u.EquipmentNote)
				if training.IsLocation(rec.Activity) {
					ro.EquipmentOK = in.EquipmentOK
				}
				if ex := u.Primary(); ex != nil {
					if row := byID[ex.ID]; row != nil {
						eo := row.exerciseOut
						ro.Exercise = &eo
					}
				}
			}
		}
		out.Recommendations = append(out.Recommendations, ro)
	}
	for _, hd := range res.Hidden {
		out.Hidden = append(out.Hidden, hiddenOut{Activity: hd.Activity, Label: hd.Activity.GermanName(), Reason: hd.Reason})
	}
	out.Week = weekDots(today, sessions, prof, slots)
	out.Context = contextLine(contextInput{
		Weather: wx, Ground: ground, Today: today, Shows: in.Profile.Shows, Reha: out.Reha,
	})
	httpx.WriteJSON(w, http.StatusOK, out)
}

// conditions returns today's weather (the latest snapshot, nil when there is none) and the
// ground condition of the stable ("" when unknown).
func (h *handler) conditions(ctx context.Context, stableID string, today time.Time) (*recommend.Weather, string, error) {
	wx, err := h.forecast(ctx, stableID, today)
	if err != nil {
		return nil, "", err
	}
	ground, err := stables.GetGroundCondition(ctx, h.deps.Pool, stableID)
	if err != nil {
		return nil, "", err
	}
	return wx, ground.Condition, nil
}

// forecast returns the weather of a stable-local day from the latest snapshot, nil when there
// is none.
func (h *handler) forecast(ctx context.Context, stableID string, day time.Time) (*recommend.Weather, error) {
	snap, err := (&weather.Store{Pool: h.deps.Pool}).Latest(ctx, stableID, day)
	switch {
	case errors.Is(err, weather.ErrNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	}
	return &recommend.Weather{Rain: snap.WillRain, TempC: snap.NightMinC}, nil
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

// isShowDay reports whether a show takes place on day.
func isShowDay(shows []training.Show, day time.Time) bool {
	for _, s := range shows {
		if training.DaysBetween(s.Date, day) == 0 {
			return true
		}
	}
	return false
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

type activeReha struct {
	phase *recommend.RehaPhase
	out   rehaOut
}

// activeRehaPhase returns the unit that the horse's active reha plan allows today (phase
// parsing and the minute ramp live in internal/reha). It returns nil without an active plan,
// before the start, after the last phase, or when the phase has no valid activity.
//
// The recommender gets the phase minimum as lower and today's ramp value as upper bound, so a
// short available time can shorten the unit but never below the phase minimum.
func (h *handler) activeRehaPhase(ctx context.Context, stableID, horseID string, today time.Time) (*activeReha, error) {
	p, err := reha.ActivePlan(ctx, h.deps.Pool, stableID, horseID)
	if err != nil || p == nil {
		return nil, err
	}
	al := p.AllowedOn(today)
	if al == nil {
		return nil, nil
	}
	done, err := reha.DoneDays(ctx, h.deps.Pool, stableID, p.ID, today, today)
	if err != nil {
		return nil, err
	}
	_, isDone := done[today.Format(dateLayout)]
	ph := al.Phase
	return &activeReha{
		phase: &recommend.RehaPhase{Name: ph.Name, Activity: training.Activity(ph.Activity), MinMinutes: ph.MinMinutes, MaxMinutes: al.Minutes, Conditions: ph.Conditions},
		out: rehaOut{PlanID: p.ID, Phase: ph.Name, PhaseIndex: al.PhaseNumber, Phases: al.Phases, Activity: ph.Activity,
			MinMinutes: ph.MinMinutes, MaxMinutes: ph.MaxMinutes, Minutes: al.Minutes, Rest: ph.Rest(), Done: isDone,
			Text: al.RuleText(), Conditions: ph.Conditions},
	}, nil
}
