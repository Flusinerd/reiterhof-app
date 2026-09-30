package trainingapi

import (
	"context"
	"errors"
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
	aiLimit         = "limit"           // free credits or rate limit used up
	aiFailed        = "failed"          // error or unusable answer
	aiNothingToPlan = "nothing_to_plan" // no open day in the week
)

// planTimeout bounds the call to the model; the rules plan needs no time.
const planTimeout = 40 * time.Second

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
	// User already claimed the day without an activity; applying the plan keeps them.
	User *personOut `json:"user"`
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
}

// planWeek: POST /api/v1/horses/{id}/week/plan?start=YYYY-MM-DD. Owners and admins get a plan
// for the open days of the week (JAN-89). Nothing is stored; the app applies the days it
// keeps with PUT /week/{day}.
//
// The language model is asked only when it is configured and the owner of the horse has
// granted the ai_training consent; every proposal is checked by the rules
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
	in, week, err := h.planInput(ctx, a, start, today, loc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var ownerID *string
	if err := h.deps.Pool.QueryRow(ctx, `SELECT owner_id::text FROM horses WHERE id = $1`, a.horseID).Scan(&ownerID); err != nil {
		h.fail(w, r, err)
		return
	}
	out := planOut{
		HorseID: a.horseID, Start: week.Start, End: week.End, Source: weekplan.SourceRules,
		OwnerIsMe: ownerID != nil && *ownerID == a.user.ID, Days: []planDayOut{},
	}

	var entries []weekplan.Entry
	switch {
	case !in.OpenDays():
		out.AIStatus = aiNothingToPlan
	case h.deps.Chat == nil:
		out.AIStatus = aiNotConfigured
	default:
		consented := false
		if ownerID != nil {
			if consented, err = privacy.Has(ctx, h.deps.Pool, *ownerID, privacy.KindAITraining); err != nil {
				h.fail(w, r, err)
				return
			}
		}
		if !consented {
			out.AIStatus = aiNoConsent
			break
		}
		entries, out.AIStatus = h.askModel(ctx, in)
	}
	if entries == nil {
		entries = weekplan.Rules(in)
	}

	users := map[string]*personOut{}
	for _, d := range week.Days {
		users[d.Date] = d.User
	}
	for _, e := range entries {
		rec := e.Recommendation
		key := e.Date.Format(dateLayout)
		out.Days = append(out.Days, planDayOut{
			Date: key, Weekday: training.DaysBetween(start, e.Date), Activity: rec.Activity, Label: rec.Activity.GermanName(),
			Minutes: rec.Minutes, Intensity: intensityKey(rec.Intensity), IntensityLabel: rec.Intensity.Label(),
			Reason: rec.Reason, Note: rec.Note, Source: e.Source, Replaced: e.Replaced, User: users[key],
		})
		if e.Source == weekplan.SourceAI {
			out.Source = weekplan.SourceAI
		}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
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

// planInput collects the week (as the week view shows it to an owner), the sessions of the
// 14 days before it, the reha units, today's weather and ground.
//
// A day is open when it is today or later and nobody planned an activity for it: status
// open, today, or planned by someone without an activity. Done days, rest days and days
// with a planned activity are context; a planned activity counts as a session.
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
	rows, err := h.querySessions(ctx, stableID, a.horseID,
		localMidnight(start.AddDate(0, 0, -recommend.RecentDays), loc), localMidnight(end.AddDate(0, 0, 1), loc), "", 0)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	plan, err := reha.ActivePlan(ctx, h.deps.Pool, stableID, a.horseID)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	wx, ground, err := h.conditions(ctx, stableID, today)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	in := weekplan.Input{
		Today: today, Profile: prof.domain(a.horseName), Recent: domainSessions(rows, loc),
		Weather: wx, Ground: recommend.Ground(ground),
	}
	for _, d := range week.Days {
		day, _ := time.Parse(dateLayout, d.Date)
		wd := weekplan.Day{Date: day, Rest: d.Status == dayRest}
		switch {
		case day.Before(today) || d.Status == dayDone || d.Status == dayRest || d.Status == dayEmpty:
		case d.Activity != nil:
			wd.Planned = *d.Activity
		default:
			wd.Open = true
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
