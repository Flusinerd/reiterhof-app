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
	// Focus is the content of the unit in a few words (model only); Exercise the library
	// exercise for it (null for hack, walker and rest).
	Focus    string       `json:"focus,omitempty"`
	Exercise *planExerOut `json:"exercise"`
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
}

// planWeek: POST /api/v1/horses/{id}/week/plan?start=YYYY-MM-DD. Owners and admins get a plan
// for the open days of the week (JAN-89). Nothing is stored; the app applies the days it
// keeps with PUT /week/{day}.
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
	for _, d := range week.Days {
		users[d.Date] = d.User
	}
	for _, e := range entries {
		rec := e.Recommendation
		key := e.Date.Format(dateLayout)
		pd := planDayOut{
			Date: key, Weekday: training.DaysBetween(start, e.Date), Activity: rec.Activity, Label: rec.Activity.GermanName(),
			Minutes: rec.Minutes, Intensity: intensityKey(rec.Intensity), IntensityLabel: rec.Intensity.Label(),
			Reason: rec.Reason, Note: rec.Note, Source: e.Source, Replaced: e.Replaced, User: users[key], Focus: e.Focus,
			Level: e.Level,
		}
		if e.Level != "" {
			pd.LevelLabel = recommend.LevelLabel(e.Level)
		}
		if ex := e.Exercise; ex != nil {
			pd.Exercise = &planExerOut{ID: ex.ID, Title: ex.Title}
		}
		out.Days = append(out.Days, pd)
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
// 14 days before it, the reha units, the weather of today and tomorrow, today's ground, the
// horse's age and level and the candidate exercises of the global library.
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
	tomorrow, err := h.forecast(ctx, stableID, today.AddDate(0, 0, 1))
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	var birthYear *int
	if err := h.deps.Pool.QueryRow(ctx, `SELECT birth_year FROM horses WHERE id = $1`, a.horseID).Scan(&birthYear); err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	domain := prof.domain(a.horseName)
	exercises, err := h.planExercises(ctx, a.horseID, domain)
	if err != nil {
		return weekplan.Input{}, weekOut{}, err
	}
	in := weekplan.Input{
		Today: today, Profile: domain, Recent: domainSessions(rows, loc),
		Weather: wx, Tomorrow: tomorrow, Ground: recommend.Ground(ground),
		Level: training.LevelClass(prof.level), Exercises: exercises,
	}
	if birthYear != nil {
		if age := today.Year() - *birthYear; age >= 0 && age <= 45 {
			in.AgeYears = age
		}
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

// planExercises returns the candidates for the model: the exercises of the global library
// (stable_id NULL; own-stable exercises are free text and stay on the server) in the libraries
// that fit the allowed activities, in progression order, with whether the horse mastered them.
func (h *handler) planExercises(ctx context.Context, horseID string, prof training.Profile) ([]weekplan.Exercise, error) {
	var libs []string
	seen := map[string]bool{}
	for _, al := range prof.Allowed {
		if al.Mode != training.ModeOn && al.Mode != training.ModeConditional {
			continue
		}
		if lib := training.ExerciseLibrary(al.Activity, prof.Discipline); lib != "" && !seen[lib] {
			seen[lib] = true
			libs = append(libs, lib)
		}
	}
	if len(libs) == 0 {
		return nil, nil
	}
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT `+exerciseCols+`,
		       EXISTS (SELECT 1 FROM sessions s WHERE s.horse_id = $2 AND s.exercise_id = e.id AND s.focus_rating = 3)
		FROM exercises e
		WHERE e.stable_id IS NULL AND e.discipline = ANY ($1)
		ORDER BY e.title`, libs, horseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []exerciseOut
	mastered := map[string]bool{}
	for rows.Next() {
		var e exerciseOut
		var steps []byte
		var done bool
		if err := rows.Scan(&e.ID, &e.Title, &e.Discipline, &e.Level, &e.GoalTags, &e.Global, &steps, &e.NextExerciseID, &done); err != nil {
			return nil, err
		}
		mastered[e.ID] = done
		list = append(list, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortByProgression(list)
	out := make([]weekplan.Exercise, 0, len(list))
	for _, e := range list {
		ex := weekplan.Exercise{ID: e.ID, Library: e.Discipline, Level: e.Level, Title: e.Title, Tags: e.GoalTags, Mastered: mastered[e.ID]}
		if e.NextExerciseID != nil {
			ex.NextID = *e.NextExerciseID
		}
		out = append(out, ex)
	}
	return out, nil
}
