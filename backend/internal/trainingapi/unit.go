package trainingapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// The unit plan ("Stundenbild", docs/domains/training-planning.md): a unit of a planned day has
// three parts, Lösen (warmup), Arbeiten (main) and Entspannen (cooldown), each with a duration in
// minutes and at most one exercise. week_slots.unit_plan stores {"warmup": {"exercise_id": ...,
// "minutes": 15}, "main": {...}, "cooldown": {...}}; a part with neither minutes nor exercise is
// left out. week_slots.exercise_id stays the main exercise for older clients.

// unitPartStored is a part as stored in week_slots.unit_plan.
type unitPartStored struct {
	ExerciseID *string `json:"exercise_id"`
	Minutes    int     `json:"minutes"`
}

type unitStored struct {
	Warmup   *unitPartStored `json:"warmup,omitempty"`
	Main     *unitPartStored `json:"main,omitempty"`
	Cooldown *unitPartStored `json:"cooldown,omitempty"`
}

// unitPartOut is a part of a unit in the answers: the exercise (null for free work) and the
// minutes of the part. ExerciseMinutes is the part of those minutes the exercise takes when it
// has a typical duration shorter than the part, the rest is free work; InsteadOf and Why say that
// the planner chose another exercise than the one that fitted best, and why (equipment, level goal).
type unitPartOut struct {
	Exercise        *planExerOut `json:"exercise"`
	Minutes         int          `json:"minutes"`
	ExerciseMinutes int          `json:"exercise_minutes,omitempty"`
	Why             string       `json:"why,omitempty"`
	InsteadOf       *planExerOut `json:"instead_of,omitempty"`
}

// unitOut is the unit plan in the answers; a part without minutes and exercise is absent.
type unitOut struct {
	Warmup   *unitPartOut `json:"warmup,omitempty"`
	Main     *unitPartOut `json:"main,omitempty"`
	Cooldown *unitPartOut `json:"cooldown,omitempty"`
}

// empty reports whether no part is set.
func (u *unitOut) empty() bool {
	return u == nil || (u.Warmup == nil && u.Main == nil && u.Cooldown == nil)
}

// exerciseInfo is what the answers need to know about an exercise of a unit plan.
type exerciseInfo struct {
	Title   string
	Minutes *int
}

// exerciseInfos returns title and typical duration of the exercises the stable can see.
func (h *handler) exerciseInfos(ctx context.Context, stableID string, ids []string) (map[string]exerciseInfo, error) {
	out := map[string]exerciseInfo{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT id::text, title, minutes FROM exercises WHERE id = ANY ($1::uuid[]) AND (stable_id IS NULL OR stable_id = $2)`, ids, stableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var info exerciseInfo
		if err := rows.Scan(&id, &info.Title, &info.Minutes); err != nil {
			return nil, err
		}
		out[id] = info
	}
	return out, rows.Err()
}

// storedExerciseIDs lists the exercise ids of a stored unit plan.
func storedExerciseIDs(raw []byte) []string {
	var u unitStored
	if len(raw) == 0 || json.Unmarshal(raw, &u) != nil {
		return nil
	}
	var ids []string
	for _, p := range []*unitPartStored{u.Warmup, u.Main, u.Cooldown} {
		if p != nil && p.ExerciseID != nil && *p.ExerciseID != "" {
			ids = append(ids, *p.ExerciseID)
		}
	}
	return ids
}

// unitOutOf turns a stored unit plan into the answer, nil when the slot has none.
func unitOutOf(raw []byte, infos map[string]exerciseInfo) *unitOut {
	var u unitStored
	if len(raw) == 0 || json.Unmarshal(raw, &u) != nil {
		return nil
	}
	conv := func(p *unitPartStored) *unitPartOut {
		if p == nil {
			return nil
		}
		out := &unitPartOut{Minutes: p.Minutes}
		if p.ExerciseID != nil {
			if info, ok := infos[*p.ExerciseID]; ok {
				out.Exercise = &planExerOut{ID: *p.ExerciseID, Title: info.Title}
				out.ExerciseMinutes = exerciseMinutes(p.Minutes, info.Minutes)
			}
		}
		if out.Exercise == nil && out.Minutes == 0 {
			return nil
		}
		return out
	}
	out := &unitOut{Warmup: conv(u.Warmup), Main: conv(u.Main), Cooldown: conv(u.Cooldown)}
	if out.empty() {
		return nil
	}
	return out
}

// exerciseMinutes is the time an exercise takes within a part: its typical duration, capped by the
// part; 0 when it has none (the whole part belongs to it) or the cap is the whole part.
func exerciseMinutes(part int, typical *int) int {
	if typical == nil || *typical <= 0 || *typical >= part {
		return 0
	}
	return *typical
}

// unitPartIn is a part of a unit in a PUT. The answer's own shape is accepted too (exercise
// {id, title} instead of exercise_id), so an app can send back what it got.
type unitPartIn struct {
	ExerciseID      *string      `json:"exercise_id"`
	Exercise        *exerciseRef `json:"exercise"`
	Minutes         int          `json:"minutes"`
	ExerciseMinutes int          `json:"exercise_minutes"` // ignored
	Why             string       `json:"why"`              // ignored
	InsteadOf       *exerciseRef `json:"instead_of"`       // ignored
}

// unitIn is the unit plan of a PUT; the parts that are not sent are absent.
type unitIn struct {
	Warmup   *unitPartIn `json:"warmup"`
	Main     *unitPartIn `json:"main"`
	Cooldown *unitPartIn `json:"cooldown"`
}

const maxPartMinutes = 240

// exerciseID is the exercise of the part: exercise_id, else exercise.id; "" for none.
func (p *unitPartIn) exerciseID() string {
	switch {
	case p == nil:
		return ""
	case p.ExerciseID != nil:
		return *p.ExerciseID
	case p.Exercise != nil:
		return p.Exercise.ID
	}
	return ""
}

// validateUnit checks the unit of a PUT and returns the plan to store (nil for an empty unit):
// minutes between 0 and 240 per part, 600 in all, and exercises the stable can see. msg is the
// message of a validation error (the plan is then nil), err a failure of the database.
func (h *handler) validateUnit(ctx context.Context, stableID string, in *unitIn) (plan *unitStored, msg string, err error) {
	var out unitStored
	total := 0
	for _, part := range [3]struct {
		name string
		in   *unitPartIn
		dst  **unitPartStored
	}{{"warmup", in.Warmup, &out.Warmup}, {"main", in.Main, &out.Main}, {"cooldown", in.Cooldown, &out.Cooldown}} {
		if part.in == nil {
			continue
		}
		if part.in.Minutes < 0 || part.in.Minutes > maxPartMinutes {
			return nil, fmt.Sprintf("unit.%s.minutes must be between 0 and %d", part.name, maxPartMinutes), nil
		}
		total += part.in.Minutes
		id := part.in.exerciseID()
		if id != "" {
			ok, err := h.exerciseVisible(ctx, stableID, id)
			if err != nil {
				return nil, "", err
			}
			if !ok {
				return nil, fmt.Sprintf("unit.%s: unknown exercise", part.name), nil
			}
		}
		if id == "" && part.in.Minutes == 0 {
			continue
		}
		stored := &unitPartStored{Minutes: part.in.Minutes}
		if id != "" {
			stored.ExerciseID = &id
		}
		*part.dst = stored
	}
	if total > maxSessionMinutes {
		return nil, "unit is too long", nil
	}
	if out.Warmup == nil && out.Main == nil && out.Cooldown == nil {
		return nil, "", nil
	}
	return &out, "", nil
}

// mainExerciseID is the exercise of the main part of a stored unit plan, "" for none.
func (u *unitStored) mainExerciseID() string {
	if u == nil || u.Main == nil || u.Main.ExerciseID == nil {
		return ""
	}
	return *u.Main.ExerciseID
}

// withMainExercise returns the stored plan with the exercise of its main part replaced (cleared
// for ""); nil when raw holds no plan. The minutes stay.
func withMainExercise(raw []byte, exerciseID string) *unitStored {
	var u unitStored
	if len(raw) == 0 || json.Unmarshal(raw, &u) != nil || (u.Warmup == nil && u.Main == nil && u.Cooldown == nil) {
		return nil
	}
	if u.Main == nil {
		u.Main = &unitPartStored{}
	}
	if exerciseID == "" {
		u.Main.ExerciseID = nil
	} else {
		u.Main.ExerciseID = &exerciseID
	}
	if u.Main.ExerciseID == nil && u.Main.Minutes == 0 {
		u.Main = nil
	}
	if u.Warmup == nil && u.Main == nil && u.Cooldown == nil {
		return nil
	}
	return &u
}

// unitHasContent reports whether a unit of a PUT sets any part (an empty object clears the plan).
func (u *unitIn) hasContent() bool {
	return u != nil && (u.Warmup != nil || u.Main != nil || u.Cooldown != nil)
}

// planPartsExercises lists the exercises of a planner unit, for the "used" bookkeeping.
func planPartsExercises(parts ...*training.Exercise) []string {
	var out []string
	for _, p := range parts {
		if p != nil {
			out = append(out, p.ID)
		}
	}
	return out
}
