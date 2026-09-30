package reha

import (
	"context"
	"slices"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

const historyLimit = 20

type phaseOut struct {
	Name          string `json:"name"`
	Days          int    `json:"days"`
	Activity      string `json:"activity"`
	ActivityLabel string `json:"activity_label"`
	Rest          bool   `json:"rest"`
	MinMinutes    int    `json:"min_minutes"`
	MaxMinutes    int    `json:"max_minutes"`
	Conditions    string `json:"conditions"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	Status        string `json:"status"` // past | current | upcoming
}

type planOut struct {
	ID            string     `json:"id"`
	Diagnosis     string     `json:"diagnosis"`
	Vet           string     `json:"vet"`
	StartDate     string     `json:"start_date"`
	EndDate       string     `json:"end_date"`
	CheckupDate   *string    `json:"checkup_date"`
	CheckupInDays *int       `json:"checkup_in_days"` // negative: overdue
	AbortCriteria string     `json:"abort_criteria"`
	ObservationID *string    `json:"observation_id"`
	Active        bool       `json:"active"`
	EndedOn       *string    `json:"ended_on"`
	State         string     `json:"state"`         // upcoming | running | finished
	CurrentPhase  int        `json:"current_phase"` // 1-based, 0 = none today
	DayIndex      int        `json:"day_index"`     // 1-based day of the plan today, 0 = not running
	TotalDays     int        `json:"total_days"`
	Phases        []phaseOut `json:"phases"`
	DoneDays      []string   `json:"done_days"`
}

// todayOut is "Heute erlaubt".
type todayOut struct {
	Date          string `json:"date"`
	PlanID        string `json:"plan_id"`
	Phase         string `json:"phase"`
	PhaseIndex    int    `json:"phase_index"` // 1-based
	Phases        int    `json:"phases"`
	DayInPhase    int    `json:"day_in_phase"` // 1-based
	DaysInPhase   int    `json:"days_in_phase"`
	Activity      string `json:"activity"`
	ActivityLabel string `json:"activity_label"`
	Rest          bool   `json:"rest"`
	Minutes       int    `json:"minutes"` // today's ramp value, 0 for rest
	MinMinutes    int    `json:"min_minutes"`
	MaxMinutes    int    `json:"max_minutes"`
	Conditions    string `json:"conditions"`
	Text          string `json:"text"` // rule text for helpers
	Done          bool   `json:"done"`
	DoneBy        string `json:"done_by,omitempty"`
}

// viewOut is the body of GET /horses/{id}/reha. view is "full" for the owner, admins and riders
// and "today" for other members: they only get today (no diagnosis, vet, dates, history).
type viewOut struct {
	HorseID       string     `json:"horse_id"`
	HorseName     string     `json:"horse_name"`
	View          string     `json:"view"`
	Date          string     `json:"date"`
	CanEdit       bool       `json:"can_edit"`
	CanMarkDone   bool       `json:"can_mark_done"`
	HasActivePlan bool       `json:"has_active_plan"`
	Plan          *planOut   `json:"plan"`
	Today         *todayOut  `json:"today"`
	History       []*planOut `json:"history"`
}

func dateStr(t time.Time) string { return t.Format(DateLayout) }

func (h *handler) buildView(ctx context.Context, a access, today time.Time, loc *time.Location) (viewOut, error) {
	v := viewOut{
		HorseID: a.horseID, HorseName: a.horseName, View: "today", Date: dateStr(today),
		CanEdit: a.manage, CanMarkDone: a.canMark(), History: []*planOut{},
	}
	if a.full() {
		v.View = "full"
	}
	stableID := a.user.StableID
	p, err := ActivePlan(ctx, h.deps.Pool, stableID, a.horseID)
	if err != nil {
		return v, err
	}
	if p == nil {
		v.CanMarkDone = false
		if v.View == "full" {
			hist, err := h.history(ctx, stableID, a.horseID, today, loc)
			v.History = hist
			return v, err
		}
		return v, nil
	}
	v.HasActivePlan = true
	done, err := DoneDays(ctx, h.deps.Pool, stableID, p.ID, p.Start, p.End())
	if err != nil {
		return v, err
	}
	v.Today = todayFor(p, today, done)
	if v.View == "today" {
		v.CanMarkDone = false
		return v, nil
	}
	v.Plan = planView(p, today, loc, done)
	v.History, err = h.history(ctx, stableID, a.horseID, today, loc)
	return v, err
}

func (h *handler) history(ctx context.Context, stableID, horseID string, today time.Time, loc *time.Location) ([]*planOut, error) {
	plans, err := EndedPlans(ctx, h.deps.Pool, stableID, horseID, historyLimit)
	if err != nil {
		return nil, err
	}
	out := make([]*planOut, 0, len(plans))
	for _, p := range plans {
		done, err := DoneDays(ctx, h.deps.Pool, stableID, p.ID, p.Start, p.End())
		if err != nil {
			return nil, err
		}
		out = append(out, planView(p, today, loc, done))
	}
	return out, nil
}

func todayFor(p *Plan, today time.Time, done map[string]DoneDay) *todayOut {
	al := p.AllowedOn(today)
	if al == nil {
		return nil
	}
	t := &todayOut{
		Date: dateStr(today), PlanID: p.ID, Phase: al.Phase.Name, PhaseIndex: al.PhaseNumber, Phases: al.Phases,
		DayInPhase: al.DayInPhase, DaysInPhase: al.Phase.Days, Activity: al.Phase.Activity,
		ActivityLabel: ActivityLabel(al.Phase.Activity), Rest: al.Phase.Rest(), Minutes: al.Minutes,
		MinMinutes: al.Phase.MinMinutes, MaxMinutes: al.Phase.MaxMinutes, Conditions: al.Phase.Conditions,
		Text: al.RuleText(),
	}
	if d, ok := done[dateStr(today)]; ok {
		t.Done, t.DoneBy = true, d.Name
	}
	return t
}

func planView(p *Plan, today time.Time, loc *time.Location, done map[string]DoneDay) *planOut {
	out := &planOut{
		ID: p.ID, Diagnosis: p.Diagnosis, Vet: p.Vet, StartDate: dateStr(p.Start), EndDate: dateStr(p.End()),
		AbortCriteria: p.AbortCriteria, ObservationID: p.ObservationID, Active: p.Active,
		State: StateOn(p.Start, p.Phases, today), TotalDays: TotalDays(p.Phases),
		Phases: []phaseOut{}, DoneDays: []string{},
	}
	if p.CheckupDate != nil {
		s := dateStr(*p.CheckupDate)
		out.CheckupDate = &s
		n := training.DaysBetween(today, *p.CheckupDate)
		out.CheckupInDays = &n
	}
	if p.EndedAt != nil {
		s := dateStr(training.Day(p.EndedAt.In(loc)))
		out.EndedOn = &s
	}
	for _, s := range Timeline(p.Start, p.Phases) {
		st := "upcoming"
		switch {
		case s.End.Before(today):
			st = "past"
		case !s.Start.After(today):
			st = "current"
		}
		out.Phases = append(out.Phases, phaseOut{
			Name: s.Phase.Name, Days: s.Phase.Days, Activity: s.Phase.Activity, ActivityLabel: ActivityLabel(s.Phase.Activity),
			Rest: s.Phase.Rest(), MinMinutes: s.Phase.MinMinutes, MaxMinutes: s.Phase.MaxMinutes,
			Conditions: s.Phase.Conditions, StartDate: dateStr(s.Start), EndDate: dateStr(s.End), Status: st,
		})
	}
	if pos, ok := Locate(p.Start, p.Phases, today); ok {
		out.CurrentPhase, out.DayIndex = pos.PhaseNumber, pos.DayInPlan+1
	}
	for d := range done {
		out.DoneDays = append(out.DoneDays, d)
	}
	slices.Sort(out.DoneDays)
	return out
}
