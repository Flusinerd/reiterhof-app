package trainingapi

import (
	"context"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// progressOut says how far a horse is on the way to its goal level (training.Progress): the class
// it is in, the goal, the lessons of the goal level it has mastered, the next ones and a German
// sentence. GoalUntil is the stored date of the goal (null for none).
type progressOut struct {
	Level     *string       `json:"level"`
	GoalLevel string        `json:"goal_level"`
	GoalUntil *string       `json:"goal_until"`
	Mastered  int           `json:"mastered"`
	Total     int           `json:"total"`
	Next      []exerciseRef `json:"next"`
	Ready     bool          `json:"ready"`
	Text      string        `json:"text"`
}

// progressBrief is the part of the progress "Was heute?" shows: the sentence and the next lessons.
type progressBrief struct {
	Text string        `json:"text"`
	Next []exerciseRef `json:"next"`
}

func (p progressOut) brief() *progressBrief { return &progressBrief{Text: p.Text, Next: p.Next} }

// progress computes the horse's progress towards its goal level, nil when it has no goal (an
// advanced horse, or an unknown level without a stored goal) or its goal level has no lessons in
// the horse's home libraries. levelUp is the warning text when the horse is ready, else "".
func (h *handler) progress(ctx context.Context, a access, p storedProfile, today time.Time) (out *progressOut, levelUp string, err error) {
	levelClass := training.LevelClass(p.level)
	goal := training.GoalLevel(levelClass, p.goalLevel)
	if goal == "" {
		return nil, "", nil
	}
	home := training.HomeLibraries(p.discipline)
	lib, err := h.loadLibrary(ctx, a.user.StableID, a.horseID, home)
	if err != nil {
		return nil, "", err
	}
	res := training.Progress(goal, home, lib.pure(), lib.masteredSessions())
	if res.Total == 0 {
		return nil, "", nil
	}
	var until *time.Time // the date only counts while it is not over
	if p.goalUntil != nil && training.DaysBetween(today, *p.goalUntil) >= 0 {
		until = p.goalUntil
	}
	out = &progressOut{
		GoalLevel: goal, Mastered: res.Mastered, Total: res.Total, Ready: res.Ready, Next: []exerciseRef{},
		Text: res.Text(a.horseName, until),
	}
	if levelClass != "" {
		out.Level = &levelClass
	}
	if p.goalUntil != nil {
		s := p.goalUntil.Format(dateLayout)
		out.GoalUntil = &s
	}
	for _, ex := range res.Next {
		out.Next = append(out.Next, exerciseRef{ID: ex.ID, Title: ex.Title})
	}
	if res.Ready {
		levelUp = res.LevelUpText(a.horseName)
	}
	return out, levelUp, nil
}

// fillPlanning adds the focus and level data to the profile response: the focuses, the block
// start, the horse's phase with the hints and warnings, the level goal and the progress.
func (h *handler) fillPlanning(ctx context.Context, a access, p storedProfile, out *profileOut) error {
	today := training.Day(h.deps.Now().In(h.location(ctx, a.user.StableID)))
	age, err := h.horseAge(ctx, a.horseID, today)
	if err != nil {
		return err
	}
	out.Phase = training.Phase(training.LevelClass(p.level), age, p.status)
	out.Focuses = focusChoicesOut(p.focuses)
	if p.cycleStart != nil {
		s := p.cycleStart.Format(dateLayout)
		out.CycleStart = &s
	}
	if p.goalLevel != "" {
		g := p.goalLevel
		out.GoalLevel = &g
	}
	if p.goalUntil != nil {
		s := p.goalUntil.Format(dateLayout)
		out.GoalUntil = &s
	}
	keys := make([]string, 0, len(p.focuses))
	for _, f := range p.focuses {
		keys = append(keys, f.Key)
	}
	out.FocusHints = training.HintKeys(out.Phase, keys)
	out.Warnings = []warningOut{}
	for _, k := range keys {
		if f, _ := training.FocusByKey(k); !f.AllowedIn(out.Phase) {
			out.Warnings = append(out.Warnings, warningOut{Key: k, Text: f.Needs})
		}
	}
	prog, levelUp, err := h.progress(ctx, a, p, today)
	if err != nil {
		return err
	}
	out.Progress = prog
	if levelUp != "" {
		out.Warnings = append(out.Warnings, warningOut{Key: "level_up", Text: levelUp})
	}
	return nil
}
