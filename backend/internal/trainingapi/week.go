package trainingapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/reha"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/load"
)

// slotRow is a week_slots row joined with the planned user.
type slotRow struct {
	Day      string
	UserID   *string
	UserName *string
	Color    *string
	Activity *string
	Status   string // planned, done, rest
	Note     *string
	Focus    *string
	ExerID   *string
	ExerName *string
}

// slotsBetween returns the slots of the days from..to (inclusive, UTC-normalised dates)
// keyed by YYYY-MM-DD.
func (h *handler) slotsBetween(ctx context.Context, stableID, horseID string, from, to time.Time) (map[string]slotRow, error) {
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT ws.day, ws.user_id::text, u.name, u.avatar_color, ws.activity, ws.status, ws.note,
		       ws.focus, ws.exercise_id::text, e.title
		FROM week_slots ws LEFT JOIN users u ON u.id = ws.user_id
		LEFT JOIN exercises e ON e.id = ws.exercise_id
		WHERE ws.stable_id = $1 AND ws.horse_id = $2 AND ws.day >= $3 AND ws.day <= $4`,
		stableID, horseID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]slotRow{}
	for rows.Next() {
		var s slotRow
		var day time.Time
		if err := rows.Scan(&day, &s.UserID, &s.UserName, &s.Color, &s.Activity, &s.Status, &s.Note,
			&s.Focus, &s.ExerID, &s.ExerName); err != nil {
			return nil, err
		}
		s.Day = day.Format(dateLayout)
		out[s.Day] = s
	}
	return out, rows.Err()
}

// Day statuses of the week view.
const (
	dayDone    = "done"    // a session was logged (or the slot is done)
	dayToday   = "today"   // today, nothing logged yet
	dayPlanned = "planned" // someone planned or claimed the day
	dayOpen    = "open"    // future day, nobody entered ("Niemand eingetragen")
	dayEmpty   = "empty"   // past day without a session
	dayRest    = "rest"    // rest day, fixed (after a show) or planned
)

type showOut struct {
	Name    string `json:"name"`
	Classes string `json:"classes,omitempty"`
	Helper  string `json:"helper,omitempty"`
}

type weekDayOut struct {
	Date     string             `json:"date"`
	Weekday  int                `json:"weekday"` // 0 = Monday
	Status   string             `json:"status"`
	IsToday  bool               `json:"is_today"`
	User     *personOut         `json:"user"`
	IsMe     bool               `json:"is_me"`
	Activity *training.Activity `json:"activity"`
	Label    string             `json:"label,omitempty"`
	Minutes  int                `json:"minutes"`
	Note     *string            `json:"note"`
	// RestReason is after_show (fixed) or planned for status rest.
	RestReason string   `json:"rest_reason,omitempty"`
	Show       *showOut `json:"show"`
	// CanTake tells whether the caller may claim the day with "Ich".
	CanTake bool `json:"can_take"`
	// Reha is the unit the active reha plan allows that day (null without plan or outside it).
	Reha *weekRehaOut `json:"reha"`
	// Focus and Exercise were planned for the day (week plan, JAN-92).
	Focus    *string      `json:"focus"`
	Exercise *planExerOut `json:"exercise"`
}

// weekRehaOut is a reha entry of the week: planned minutes and whether "Heute erledigt" was tapped.
type weekRehaOut struct {
	PlanID        string `json:"plan_id"`
	Phase         string `json:"phase"`
	Activity      string `json:"activity"`
	ActivityLabel string `json:"activity_label"`
	Rest          bool   `json:"rest"`
	Minutes       int    `json:"minutes"`
	Done          bool   `json:"done"`
}

type segmentOut struct {
	Date     string  `json:"date"`
	Load     float64 `json:"load"`
	Level    string  `json:"level"`
	Sessions int     `json:"sessions"`
}

type weekOut struct {
	HorseID    string       `json:"horse_id"`
	Start      string       `json:"start"`
	End        string       `json:"end"`
	Days       []weekDayOut `json:"days"`
	Segments   []segmentOut `json:"segments"`
	Assessment string       `json:"assessment"`
	Sessions   int          `json:"sessions"`
	CanEdit    bool         `json:"can_edit"`
}

// getWeek: GET /api/v1/horses/{id}/week?start=YYYY-MM-DD (any day; the week is Mon-Sun).
func (h *handler) getWeek(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorize(w, r)
	if !ok {
		return
	}
	loc := h.location(r.Context(), a.user.StableID)
	start := mondayOf(training.Day(h.deps.Now().In(loc)))
	if s := r.URL.Query().Get("start"); s != "" {
		d, err := time.Parse(dateLayout, s)
		if err != nil {
			invalid(w, "start must be YYYY-MM-DD")
			return
		}
		start = mondayOf(d)
	}
	out, err := h.buildWeek(r.Context(), a, start, loc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// buildWeek aggregates sessions, slots, shows and rest rules of the week starting on
// the Monday start. Weeks are computed on calendar days in the stable's timezone, so
// the DST weeks (23 or 25 hour Sundays) still have seven days.
func (h *handler) buildWeek(ctx context.Context, a access, start time.Time, loc *time.Location) (weekOut, error) {
	stableID := a.user.StableID
	end := start.AddDate(0, 0, 6)
	today := training.Day(h.deps.Now().In(loc))

	prof, err := h.loadProfile(ctx, stableID, a.horseID)
	if err != nil {
		return weekOut{}, err
	}
	all, err := h.querySessions(ctx, stableID, a.horseID, localMidnight(start, loc), localMidnight(end.AddDate(0, 0, 1), loc), "", 0)
	if err != nil {
		return weekOut{}, err
	}
	// Loads and done-status use every session; who did what obeys the visibility rules.
	slots, err := h.slotsBetween(ctx, stableID, a.horseID, start, end)
	if err != nil {
		return weekOut{}, err
	}
	rehaPlan, err := reha.ActivePlan(ctx, h.deps.Pool, stableID, a.horseID)
	if err != nil {
		return weekOut{}, err
	}
	var rehaDone map[string]reha.DoneDay
	if rehaPlan != nil {
		if rehaDone, err = reha.DoneDays(ctx, h.deps.Pool, stableID, rehaPlan.ID, start, end); err != nil {
			return weekOut{}, err
		}
	}
	sessions := domainSessions(all, loc)
	segs := load.WeekLoad(sessions, start)
	shows := map[string]showDTO{}
	for _, s := range prof.shows {
		if _, dup := shows[s.Date]; !dup {
			shows[s.Date] = s
		}
	}
	domainShows := prof.domain("").Shows

	// Users' names and colors for logged sessions.
	byDay := map[string][]sessionRow{}
	for _, s := range all {
		d := s.StartedAt.In(loc).Format(dateLayout)
		byDay[d] = append(byDay[d], s)
	}

	out := weekOut{
		HorseID: a.horseID, Start: start.Format(dateLayout), End: end.Format(dateLayout),
		Days: make([]weekDayOut, 0, 7), Segments: make([]segmentOut, 0, 7), CanEdit: a.manage,
	}
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i)
		key := day.Format(dateLayout)
		d := weekDayOut{Date: key, Weekday: i, IsToday: day.Equal(today)}
		if sh, ok := shows[key]; ok {
			d.Show = &showOut{Name: sh.Name, Classes: sh.Classes, Helper: sh.Helper}
		}
		if rehaPlan != nil {
			if al := rehaPlan.AllowedOn(day); al != nil {
				_, isDone := rehaDone[key]
				d.Reha = &weekRehaOut{
					PlanID: rehaPlan.ID, Phase: al.Phase.Name, Activity: al.Phase.Activity,
					ActivityLabel: reha.ActivityLabel(al.Phase.Activity), Rest: al.Phase.Rest(),
					Minutes: al.Minutes, Done: isDone,
				}
			}
		}
		slot, hasSlot := slots[key]
		if hasSlot {
			d.Note, d.Focus = slot.Note, slot.Focus
			if slot.ExerID != nil && slot.ExerName != nil {
				d.Exercise = &planExerOut{ID: *slot.ExerID, Title: *slot.ExerName}
			}
			if slot.Activity != nil {
				act := training.Activity(*slot.Activity)
				d.Activity, d.Label = &act, act.GermanName()
			}
			if slot.UserID != nil && slot.UserName != nil {
				d.User = &personOut{ID: *slot.UserID, Name: *slot.UserName}
				if slot.Color != nil {
					d.User.ColorKey = *slot.Color
				}
			}
		}
		done, detailed := false, false
		for _, s := range byDay[key] {
			done = true
			if !a.manage && !s.Visible && s.UserID != a.user.ID {
				continue // hidden from this rider: counts as done, without details
			}
			// Several sessions on one day: the first one names the row, minutes add up.
			if !detailed {
				act := s.Activity
				d.User = &personOut{ID: s.UserID, Name: s.UserName}
				d.Activity, d.Label = &act, act.GermanName()
			}
			detailed = true
			d.Minutes += s.Minutes
		}
		if done && !detailed {
			// Every session of the day is hidden from this rider: the slot must not leak who or what.
			d.User, d.Activity, d.Label, d.Note, d.Focus, d.Exercise = nil, nil, "", nil, nil, nil
		}
		switch {
		case done || slot.Status == "done":
			d.Status = dayDone
		case restAfterShow(prof.rhythm, domainShows, day):
			d.Status, d.RestReason = dayRest, "after_show"
			d.User, d.Activity, d.Label, d.Focus, d.Exercise = nil, nil, "", nil, nil
		case hasSlot && slot.Status == "rest":
			d.Status, d.RestReason = dayRest, "planned"
			d.User, d.Activity, d.Label, d.Focus, d.Exercise = nil, nil, "", nil, nil
		case d.IsToday:
			d.Status = dayToday
		case hasSlot && (slot.UserID != nil || slot.Activity != nil) && day.After(today):
			d.Status = dayPlanned
		case day.After(today):
			d.Status = dayOpen
		default:
			d.Status = dayEmpty
		}
		d.IsMe = d.User != nil && d.User.ID == a.user.ID
		d.CanTake = a.canTakeSlots() && !day.Before(today) && d.Status != dayDone && d.Status != dayRest &&
			(d.User == nil || d.IsMe)
		out.Days = append(out.Days, d)
	}
	for _, s := range segs {
		out.Segments = append(out.Segments, segmentOut{
			Date: s.Day.Format(dateLayout), Load: round1(s.Load), Level: intensityKey(s.Level), Sessions: s.Sessions,
		})
		out.Sessions += s.Sessions
	}
	out.Assessment = load.Assess(segs, prof.rhythm)
	return out, nil
}

type slotIn struct {
	Status   string  `json:"status"`  // planned, rest, open (removes the entry)
	UserID   *string `json:"user_id"` // omitted: the caller; "": nobody (owner/admin only)
	Activity string  `json:"activity"`
	Note     string  `json:"note"`
	// Focus and ExerciseID come from the week plan (JAN-92). Omitted (null): the stored value
	// is kept; "": it is cleared; a value: it is set (JAN-95).
	Focus      *string `json:"focus"`
	ExerciseID *string `json:"exercise_id"`
}

// putWeekDay: PUT /api/v1/horses/{id}/week/{day}. Owners and admins edit every day; riders
// with take_week_slots may claim a free day for themselves or release their own claim.
func (h *handler) putWeekDay(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorize(w, r)
	if !ok {
		return
	}
	if !a.canTakeSlots() {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "you may not take days for this horse")
		return
	}
	day, err := time.Parse(dateLayout, r.PathValue("day"))
	if err != nil {
		invalid(w, "day must be YYYY-MM-DD")
		return
	}
	var in slotIn
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Status == "" {
		in.Status = "planned"
	}
	if in.Status != "planned" && in.Status != "rest" && in.Status != "open" {
		invalid(w, "status must be planned, rest or open")
		return
	}
	if in.Activity != "" && !training.Activity(in.Activity).Valid() {
		invalid(w, "unknown activity")
		return
	}
	if len([]rune(in.Note)) > 500 {
		invalid(w, "note is too long (max 500 characters)")
		return
	}
	var focusIn, exerciseIn string
	if in.Focus != nil {
		focusIn = strings.TrimSpace(*in.Focus)
	}
	if in.ExerciseID != nil {
		exerciseIn = *in.ExerciseID
	}
	if len([]rune(focusIn)) > 80 {
		invalid(w, "focus is too long (max 80 characters)")
		return
	}
	ctx := r.Context()
	stableID := a.user.StableID
	if exerciseIn != "" {
		var ok bool
		if err := h.deps.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM exercises WHERE id = $1 AND (stable_id IS NULL OR stable_id = $2))`,
			exerciseIn, stableID).Scan(&ok); err != nil && !isInvalidUUID(err) {
			h.fail(w, r, err)
			return
		} else if !ok {
			invalid(w, "unknown exercise_id")
			return
		}
	}
	loc := h.location(ctx, stableID)
	today := training.Day(h.deps.Now().In(loc))

	userID := a.user.ID
	if in.UserID != nil {
		userID = *in.UserID
	}
	if !a.manage {
		// Riders: only their own claim on a day that is not over.
		if userID != a.user.ID || in.Status == "rest" {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "riders may only take free days for themselves")
			return
		}
		if day.Before(today) {
			invalid(w, "the day is already over")
			return
		}
	} else if userID != "" {
		var member bool
		if err := h.deps.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND stable_id = $2)`,
			userID, stableID).Scan(&member); err != nil && !isInvalidUUID(err) {
			h.fail(w, r, err)
			return
		} else if !member {
			invalid(w, "user_id is not a member of the stable")
			return
		}
	}

	cur, err := h.slotsBetween(ctx, stableID, a.horseID, day, day)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	existing, has := cur[day.Format(dateLayout)]
	if !a.manage && has {
		mine := existing.UserID != nil && *existing.UserID == a.user.ID
		free := existing.UserID == nil && existing.Status == "planned"
		if existing.Status == "done" || existing.Status == "rest" || !(mine || free) {
			httpx.WriteError(w, http.StatusConflict, "conflict", "this day is already taken")
			return
		}
	}

	switch in.Status {
	case "open":
		if !a.manage && !(has && existing.UserID != nil) {
			httpx.WriteError(w, http.StatusConflict, "conflict", "nothing to release")
			return
		}
		_, err = h.deps.Pool.Exec(ctx, `DELETE FROM week_slots WHERE horse_id = $1 AND stable_id = $2 AND day = $3`,
			a.horseID, stableID, day)
	default:
		var uid, act, exer, focus any
		if in.Status == "planned" && focusIn != "" {
			focus = focusIn
		}
		if in.Status == "planned" && userID != "" {
			uid = userID
		}
		if in.Status == "planned" && in.Activity != "" {
			act = in.Activity
		}
		if in.Status == "planned" && exerciseIn != "" {
			exer = exerciseIn
		}
		// Fields that are not sent keep their value: a rider taking a planned day with "Ich"
		// keeps its activity, note, focus and exercise. Focus and exercise sent as "" are
		// cleared. A rest day has no activity, focus or exercise.
		_, err = h.deps.Pool.Exec(ctx, `
			INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status, note, focus, exercise_id)
			VALUES ($1, $2, $3, $4::uuid, $5, $6, NULLIF($7, ''), $8, $9::uuid)
			ON CONFLICT (horse_id, day) DO UPDATE SET
				user_id = EXCLUDED.user_id,
				activity = CASE WHEN EXCLUDED.status = 'planned' THEN COALESCE(EXCLUDED.activity, week_slots.activity) END,
				status = EXCLUDED.status,
				note = COALESCE(EXCLUDED.note, week_slots.note),
				focus = CASE WHEN EXCLUDED.status = 'planned' THEN
					CASE WHEN $10::boolean THEN EXCLUDED.focus ELSE week_slots.focus END END,
				exercise_id = CASE WHEN EXCLUDED.status = 'planned' THEN
					CASE WHEN $11::boolean THEN EXCLUDED.exercise_id ELSE week_slots.exercise_id END END`,
			stableID, a.horseID, day, uid, act, in.Status, strings.TrimSpace(in.Note), focus, exer,
			in.Focus != nil, in.ExerciseID != nil)
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out, err := h.buildWeek(ctx, a, mondayOf(day), loc)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
