package trainingapi

import (
	"context"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training/recommend"
)

// What the planner (training/recommend, training/weekplan) needs besides the profile and the
// sessions comes from the database here: the horse's age and level class, the block of the week,
// the return to work, the stable's equipment, the exercise library and the own focuses of the
// people who ride. The planner itself stays pure.

// historyDays is how far before the Monday of a week the sessions are read for planning: the
// Entlastungswoche looks at the three weeks before it, the load limit at two, the rotation and the
// exercise pick at 14 days.
const historyDays = 21

// planCtx is the planner's view of one horse.
type planCtx struct {
	age       int
	level     string // training.LevelClass of the profile's level
	block     recommend.Block
	restart   *time.Time
	equipment training.StableEquipment
	lib       library
	exercises []training.Exercise
	riders    map[string][]string // the own focuses of a person, by user id
}

// loadPlanCtx reads the planner's data for a horse; weekStart is the Monday of the week the block
// is asked for.
func (h *handler) loadPlanCtx(ctx context.Context, a access, prof storedProfile, domain training.Profile, weekStart, today time.Time, loc *time.Location) (planCtx, error) {
	stableID := a.user.StableID
	pc := planCtx{level: training.LevelClass(prof.level)}
	var err error
	if pc.age, err = h.horseAge(ctx, a.horseID, today); err != nil {
		return pc, err
	}
	if pc.block, err = h.blockOf(ctx, stableID, a.horseID, prof, weekStart, today, loc); err != nil {
		return pc, err
	}
	if pc.restart, err = h.restartAnchor(ctx, stableID, a.horseID, today, loc); err != nil {
		return pc, err
	}
	if pc.equipment, _, err = h.loadEquipment(ctx, stableID); err != nil {
		return pc, err
	}
	if pc.lib, err = h.planLibrary(ctx, stableID, a.horseID, domain); err != nil {
		return pc, err
	}
	pc.exercises = pc.lib.pure()
	if pc.riders, err = h.riderFocuses(ctx, a.horseID); err != nil {
		return pc, err
	}
	return pc, nil
}

// planLibrary reads the exercises of every library that fits an activity the profile allows, the
// stable's own rows included.
func (h *handler) planLibrary(ctx context.Context, stableID, horseID string, prof training.Profile) (library, error) {
	var libs []string
	seen := map[string]bool{}
	for _, al := range prof.Allowed {
		if al.Mode != training.ModeOn && al.Mode != training.ModeConditional {
			continue
		}
		for _, lib := range training.ExerciseLibraries(al.Activity, prof.Discipline) {
			if !seen[lib] {
				seen[lib] = true
				libs = append(libs, lib)
			}
		}
	}
	if len(libs) == 0 {
		return library{mastered: map[string]bool{}, rated: map[string]bool{}}, nil
	}
	return h.loadLibrary(ctx, stableID, horseID, libs)
}

// byID indexes the rows of the library.
func (l library) byID() map[string]*exerciseRow {
	out := make(map[string]*exerciseRow, len(l.rows))
	for i := range l.rows {
		out[l.rows[i].ID] = &l.rows[i]
	}
	return out
}

// attachSlots adds what the week's slots know to the sessions of the same day and activity: the
// focus and the exercises of the unit plan the session was done from, so that the rotation and the
// exercise pick see them. Nothing the session already has is replaced.
func attachSlots(sessions []training.Session, slots map[string]slotRow) {
	for i := range sessions {
		s := &sessions[i]
		slot, ok := slots[s.Day.Format(dateLayout)]
		if !ok || slot.Activity == nil || training.Activity(*slot.Activity) != s.Activity {
			continue
		}
		if s.FocusKey == "" && slot.FocusKey != nil && training.ValidFocusKey(*slot.FocusKey) {
			s.FocusKey = *slot.FocusKey
		}
		if len(s.Exercises) == 0 {
			s.Exercises = storedExerciseIDs(slot.UnitPlan)
		}
	}
}

// firstName is the part of a name before the first space, for "Mias Schwerpunkt Takt".
func firstName(name string) string {
	if f := strings.Fields(name); len(f) > 0 {
		return f[0]
	}
	return ""
}

// unitFocusOut is the focus of a planned unit with the reason for it.
type unitFocusOut struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Why   string `json:"why"`
}

// focusOfUnit is the focus of a planned unit, nil for a unit without one.
func focusOfUnit(u recommend.Unit) *unitFocusOut {
	if u.Empty() {
		return nil
	}
	return &unitFocusOut{Key: u.FocusKey, Label: training.FocusLabel(u.FocusKey), Why: u.FocusWhy}
}

// unitOutOfPlan turns a unit of the planner into its answer, nil when the unit has no part.
func unitOutOfPlan(u recommend.Unit) *unitOut {
	if u.Empty() {
		return nil
	}
	ref := func(ex *training.Exercise) *planExerOut {
		if ex == nil {
			return nil
		}
		return &planExerOut{ID: ex.ID, Title: ex.Title}
	}
	conv := func(p recommend.Part) *unitPartOut {
		if p.Minutes == 0 && p.Exercise == nil {
			return nil
		}
		out := &unitPartOut{Exercise: ref(p.Exercise), Minutes: p.Minutes, Why: p.Why, InsteadOf: ref(p.InsteadOf)}
		if p.Exercise != nil && p.Exercise.Minutes > 0 {
			typical := p.Exercise.Minutes
			out.ExerciseMinutes = exerciseMinutes(p.Minutes, &typical)
		}
		return out
	}
	out := &unitOut{Warmup: conv(u.Warmup), Main: conv(u.Main), Cooldown: conv(u.Cooldown)}
	if out.empty() {
		return nil
	}
	return out
}

// noteOut is a pointer to a German note, nil when there is none.
func noteOut(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
