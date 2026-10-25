package trainingapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/mistral"
	"github.com/Flusinerd/reiterhof-app/backend/internal/privacy"
	"github.com/Flusinerd/reiterhof-app/backend/internal/reha"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/weekplan"
)

// AI status of a week plan (planOut.AIStatus).
const (
	aiUsed          = "used"            // the model answered; accepted days have source "ai"
	aiNotConfigured = "not_configured"  // no API key on the server
	aiNoConsent     = "no_consent"      // the owner has not granted ai_training
	aiOwnerUnder16  = "owner_under_16"  // the owner has not stated to be 16 or older
	aiLimit         = "limit"           // free credits or rate limit used up
	aiFailed        = "failed"          // error or unusable answer
	aiNothingToPlan = "nothing_to_plan" // no open day in the week
)

// planTimeout bounds the call to the model including one retry after HTTP 429; the rules plan
// needs no time.
const planTimeout = 90 * time.Second

type planDayOut struct {
	Date           string            `json:"date"`
	Weekday        int               `json:"weekday"` // 0 = Monday
	Activity       training.Activity `json:"activity"`
	Label          string            `json:"label"`
	Minutes        int               `json:"minutes"`
	Intensity      string            `json:"intensity"`
	IntensityLabel string            `json:"intensity_label"`
	Reason         string            `json:"reason"`
	Note           string            `json:"note,omitempty"`
	// Source is "ai" for an accepted proposal of the model, "rules" otherwise.
	Source string `json:"source"`
	// Replaced says why the model's proposal for this day was replaced by the rules.
	Replaced string `json:"replaced,omitempty"`
	// Focus is the content of the unit in a few words (model only); Exercise the main library
	// exercise of the unit (null when it has none, for walker and rest).
	Focus    string       `json:"focus,omitempty"`
	Exercise *planExerOut `json:"exercise"`
	// FocusKey, FocusLabel and FocusWhy are the catalog focus of the unit and the reason for it,
	// Unit its three parts (Lösen, Arbeiten, Entspannen); all null for rest and walker days.
	FocusKey   *string  `json:"focus_key"`
	FocusLabel *string  `json:"focus_label"`
	FocusWhy   *string  `json:"focus_why"`
	Unit       *unitOut `json:"unit"`
	// EquipmentOK is the day's override of the set-up (null: the location's default),
	// EquipmentNote says in German that the unit is planned without set-up.
	EquipmentOK   *bool   `json:"equipment_ok"`
	EquipmentNote *string `json:"equipment_note"`
	// Level of the unit by its load: recovery, light, normal, demanding ("" for rest; JAN-93).
	Level      string `json:"level,omitempty"`
	LevelLabel string `json:"level_label,omitempty"`
	// User already claimed the day without an activity; applying the plan keeps them.
	User *personOut `json:"user"`
}

type planExerOut struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type planOut struct {
	HorseID string `json:"horse_id"`
	Start   string `json:"start"`
	End     string `json:"end"`
	// Source is "ai" when at least one day comes from the model, else "rules".
	Source   string `json:"source"`
	AIStatus string `json:"ai_status"`
	// OwnerIsMe tells whether the caller owns the horse (only the owner can grant ai_training).
	OwnerIsMe bool         `json:"owner_is_me"`
	Days      []planDayOut `json:"days"`
	// Block is the week's place in the four-week blocks (null without cycle_start).
	Block *blockOut `json:"block"`
}

// Limits of a single-day request (JAN-95).
const (
	maxDraftDays    = 7
	maxDraftMinutes = 600
)

// planRequest is the optional body of planWeek. Both fields apply only together with the
// day query parameter.
type planRequest struct {
	// Draft is the plan the owner currently sees for the other days of the week; it is the
	// context for the re-planned day.
	Draft []planDraftDay `json:"draft"`
	// Exclude lists activities the owner rejected for the re-planned day.
	Exclude []string `json:"exclude"`
}

// planDraftDay is one day of the draft: an activity with its minutes, or "rest".
type planDraftDay struct {
	Date     string `json:"date"`
	Activity string `json:"activity"`
	Minutes  int    `json:"minutes"`
}

// readOptionalJSON decodes the request body into v like httpx.ReadJSON, but accepts an empty
// body (older clients send none) as "nothing set".
func readOptionalJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, httpx.MaxBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body is too large")
		} else {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body: "+err.Error())
		}
		return false
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return true
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return httpx.ReadJSON(w, r, v)
}

// planWeek: POST /api/v1/horses/{id}/week/plan?start=YYYY-MM-DD[&day=YYYY-MM-DD]. Owners and
// admins get a plan for the open days of the week (JAN-89). Nothing is stored; the app applies
// the days it keeps with PUT /week/{day}.
//
// With day only that day is planned (JAN-95), also when somebody already planned it; the
// other days are context. The optional body carries the draft the owner sees for them and the
// activities rejected for the day (planRequest).
//
// The language model is asked only when it is configured and the owner of the horse has
// granted the ai_training consent and stated to be 16 or older (Mistral's terms forbid personal
// data of children under the age of digital consent, even with a parent's consent); every
// proposal is checked by the rules
// (recommend.Check). Without the model, or when it fails, the rules plan every day.
func (h *handler) planWeek(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorize(w, r)
	if !ok {
		return
	}
	if !a.manage {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner and admins can plan the week")
		return
	}
	ctx := r.Context()
	stableID := a.user.StableID
	loc := h.location(ctx, stableID)
	today := training.Day(h.deps.Now().In(loc))
	start := mondayOf(today)
	if s := r.URL.Query().Get("start"); s != "" {
		d, err := time.Parse(dateLayout, s)
		if err != nil {
			invalid(w, "start must be YYYY-MM-DD")
			return
		}
		start = mondayOf(d)
	}
	var single time.Time // zero: plan every open day
	if s := r.URL.Query().Get("day"); s != "" {
		d, err := time.Parse(dateLayout, s)
		if err != nil {
			invalid(w, "day must be YYYY-MM-DD")
			return
		}
		single = d
	}
	var req planRequest
	if !readOptionalJSON(w, r, &req) {
		return
	}
	in, week, err := h.planInput(ctx, a, start, today, loc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if !single.IsZero() {
		if err := applyDayOverrides(&in, week, today, single, req); err != nil {
			invalid(w, err.Error())
			return
		}
	}
	var ownerID *string
	if err := h.deps.Pool.QueryRow(ctx, `SELECT owner_id::text FROM horses WHERE id = $1`, a.horseID).Scan(&ownerID); err != nil {
		h.fail(w, r, err)
		return
	}
	out := planOut{
		HorseID: a.horseID, Start: week.Start, End: week.End, Source: weekplan.SourceRules,
		OwnerIsMe: ownerID != nil && *ownerID == a.user.ID, Days: []planDayOut{}, Block: week.Block,
	}

	var entries []weekplan.Entry
	switch {
	case !in.OpenDays():
		out.AIStatus = aiNothingToPlan
	case h.deps.Chat == nil:
		out.AIStatus = aiNotConfigured
	default:
		consented, adult := false, false
		if ownerID != nil {
			if consented, err = privacy.Has(ctx, h.deps.Pool, *ownerID, privacy.KindAITraining); err != nil {
				h.fail(w, r, err)
				return
			}
			if err = h.deps.Pool.QueryRow(ctx, `SELECT age_confirmed_at IS NOT NULL FROM users WHERE id = $1`, *ownerID).Scan(&adult); err != nil {
				h.fail(w, r, err)
				return
			}
		}
		switch {
		case !consented:
			out.AIStatus = aiNoConsent
		case !adult:
			out.AIStatus = aiOwnerUnder16
		default:
			entries, out.AIStatus = h.askModel(ctx, in)
		}
	}
	if entries == nil {
		entries = weekplan.Rules(in)
	}

	users := map[string]*personOut{}
	equipmentOK := map[string]*bool{}
	for _, d := range week.Days {
		users[d.Date], equipmentOK[d.Date] = d.User, d.EquipmentOK
	}
	for _, e := range entries {
		rec := e.Recommendation
		key := e.Date.Format(dateLayout)
		pd := planDayOut{
			Date: key, Weekday: training.DaysBetween(start, e.Date), Activity: rec.Activity, Label: rec.Activity.GermanName(),
			Minutes: rec.Minutes, Intensity: intensityKey(rec.Intensity), IntensityLabel: rec.Intensity.Label(),
			Reason: rec.Reason, Note: rec.Note, Source: e.Source, Replaced: e.Replaced, User: users[key], Focus: e.Focus,
			Level: e.Level, EquipmentOK: equipmentOK[key],
		}
		if e.Level != "" {
			pd.LevelLabel = recommend.LevelLabel(e.Level)
		}
		if ex := e.Exercise; ex != nil {
			pd.Exercise = &planExerOut{ID: ex.ID, Title: ex.Title}
		}
		if f := focusOfUnit(e.Unit); f != nil {
			pd.FocusKey, pd.FocusLabel, pd.FocusWhy = &f.Key, &f.Label, &f.Why
			pd.Unit = unitOutOfPlan(e.Unit)
			pd.EquipmentNote = noteOut(e.Unit.EquipmentNote)
		}
		out.Days = append(out.Days, pd)
		if e.Source == weekplan.SourceAI {
			out.Source = weekplan.SourceAI
		}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// applyDayOverrides turns the input of planInput into the input for re-planning one day: that
// day is the only open one (a planned or claimed day is opened, its user stays in the
// response), the draft of the other open days becomes context (a planned activity with its
// minutes, or a rest day) and the rejected activities go to the day. The day must be in the
// week, not over, not done, not a show day and not the fixed rest day after a show. Every error is a
// validation error with a message for the client.
func applyDayOverrides(in *weekplan.Input, week weekOut, today, day time.Time, req planRequest) error {
	idx := -1
	for i, d := range in.Days {
		if d.Date.Equal(day) {
			idx = i
		}
	}
	if idx < 0 {
		return errors.New("day cannot be planned")
	}
	if wd := week.Days[idx]; day.Before(today) || wd.Status == dayDone || wd.Status == dayEmpty ||
		(wd.Status == dayRest && wd.RestReason == "after_show") || wd.Show != nil {
		return errors.New("day cannot be planned")
	}

	if len(req.Draft) > maxDraftDays {
		return errors.New("draft has too many days (max 7)")
	}
	draft := map[string]planDraftDay{}
	for _, d := range req.Draft {
		date, err := time.Parse(dateLayout, d.Date)
		if err != nil {
			return errors.New("draft date must be YYYY-MM-DD")
		}
		inWeek := false
		for _, wd := range in.Days {
			inWeek = inWeek || wd.Date.Equal(date)
		}
		if !inWeek {
			return errors.New("draft date is outside the week")
		}
		if d.Activity != "rest" && !training.Activity(d.Activity).Valid() {
			return errors.New("draft activity is unknown")
		}
		if d.Minutes < 0 || d.Minutes > maxDraftMinutes {
			return errors.New("draft minutes must be between 0 and 600")
		}
		draft[date.Format(dateLayout)] = d
	}
	var exclude []training.Activity
	for _, s := range req.Exclude {
		if !training.Activity(s).Valid() {
			return errors.New("exclude activity is unknown")
		}
		exclude = append(exclude, training.Activity(s))
	}

	for i := range in.Days {
		d := &in.Days[i]
		if i == idx {
			d.Open, d.Planned, d.Minutes, d.Rest, d.Exclude = true, "", 0, false, exclude
			d.FocusKey, d.Exercises = "", nil
			continue
		}
		wasOpen := d.Open
		d.Open = false
		if e, ok := draft[d.Date.Format(dateLayout)]; ok && wasOpen {
			if e.Activity == "rest" {
				d.Rest = true
			} else {
				d.Planned, d.Minutes = training.Activity(e.Activity), e.Minutes
			}
		}
	}
	return nil
}

// askModel asks the language model and checks its answer. On failure it returns nil entries
// (the caller falls back to the rules) and the status; the log line carries no content.
func (h *handler) askModel(ctx context.Context, in weekplan.Input) ([]weekplan.Entry, string) {
	ctx, cancel := context.WithTimeout(ctx, planTimeout)
	defer cancel()
	answer, err := h.deps.Chat.CompleteJSON(ctx, weekplan.SystemPrompt, weekplan.Prompt(in))
	if err != nil {
		h.deps.Log.WarnContext(ctx, "week plan: language model failed, using the rules", "err", err)
		if errors.Is(err, mistral.ErrLimit) {
			return nil, aiLimit
		}
		return nil, aiFailed
	}
	proposals, err := weekplan.Parse(in, answer)
	if err != nil {
		h.deps.Log.WarnContext(ctx, "week plan: unusable answer, using the rules", "err", err)
		return nil, aiFailed
	}
	return weekplan.Merge(in, proposals), aiUsed
}

// planInput collects the week (as the week view shows it to an owner), the sessions of the 21
// days before it, the reha units, the weather of today and tomorrow, today's ground, the horse's
// age and level, its focuses and the focuses of the people who claimed days, the block of the week,
// the return to work, the stable's equipment and the exercise library.
//
// A day is open when it is today or later and nobody planned an activity for it: status
// open, today, or planned by someone without an activity. Done days, rest days and days
// with a planned activity are context; a planned activity counts as a session with its focus
// and exercises.
func (h *handler) planInput(ctx context.Context, a access, start, today time.Time, loc *time.Location) (weekplan.Input, weekOut, error) {
	stableID := a.user.StableID
	week, err := h.buildWeek(ctx, a, start, loc)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	prof, err := h.loadProfile(ctx, stableID, a.horseID)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	end := start.AddDate(0, 0, 6)
	from := start.AddDate(0, 0, -historyDays)
	rows, err := h.querySessions(ctx, stableID, a.horseID, localMidnight(from, loc), localMidnight(end.AddDate(0, 0, 1), loc), "", 0)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	slots, err := h.slotsBetween(ctx, stableID, a.horseID, from, end)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	sessions := domainSessions(rows, loc)
	attachSlots(sessions, slots)
	plan, err := reha.ActivePlan(ctx, h.deps.Pool, stableID, a.horseID)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	wx, ground, err := h.conditions(ctx, stableID, today)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	tomorrow, err := h.forecast(ctx, stableID, today.AddDate(0, 0, 1))
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	domain := prof.domain(a.horseName)
	pc, err := h.loadPlanCtx(ctx, a, prof, domain, start, today, loc)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	in := weekplan.Input{
		Today: today, Profile: domain, Recent: sessions,
		Weather: wx, Tomorrow: tomorrow, Ground: recommend.Ground(ground),
		Level: pc.level, AgeYears: pc.age, Exercises: pc.exercises,
		Block: pc.block, RestartFrom: pc.restart, Equipment: pc.equipment,
	}
	for _, d := range week.Days {
		day, _ := time.Parse(dateLayout, d.Date)
		wd := weekplan.Day{Date: day, Rest: d.Status == dayRest, EquipmentOK: d.EquipmentOK}
		switch {
		case day.Before(today) || d.Status == dayDone || d.Status == dayRest || d.Status == dayEmpty || d.Show != nil:
			// A show day is context: the show is the work of the day, no unit or exercise.
		case d.Activity != nil:
			wd.Planned = *d.Activity
			if d.FocusKey != nil {
				wd.FocusKey = *d.FocusKey
			}
			wd.Exercises = unitExerciseIDs(d.Unit)
		default:
			wd.Open = true
			if d.User != nil {
				if keys := pc.riders[d.User.ID]; len(keys) > 0 {
					wd.RiderFocuses, wd.RiderName = keys, firstName(d.User.Name)
				}
			}
		}
		if plan != nil {
			if al := plan.AllowedOn(day); al != nil {
				ph := al.Phase
				wd.Reha = &recommend.RehaPhase{Name: ph.Name, Activity: training.Activity(ph.Activity),
					MinMinutes: ph.MinMinutes, MaxMinutes: al.Minutes, Conditions: ph.Conditions}
			}
		}
		in.Days = append(in.Days, wd)
	}
	return in, week, nil
}

// unitExerciseIDs lists the exercises of a unit plan, the main one first.
func unitExerciseIDs(u *unitOut) []string {
	if u == nil {
		return nil
	}
	var ids []string
	for _, p := range []*unitPartOut{u.Main, u.Warmup, u.Cooldown} {
		if p != nil && p.Exercise != nil {
			ids = append(ids, p.Exercise.ID)
		}
	}
	return ids
}
