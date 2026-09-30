package trainingapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

type exerciseOut struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Discipline string   `json:"discipline"`
	Level      string   `json:"level"`
	GoalTags   []string `json:"goal_tags"`
	Global     bool     `json:"global"`
	StepCount  int      `json:"step_count"`
	Steps      []string `json:"steps,omitempty"`
	// NextExerciseID and Next describe the follow-up exercise of the progression.
	NextExerciseID *string         `json:"next_exercise_id"`
	Next           *progressionOut `json:"next,omitempty"`
}

const exerciseCols = `e.id::text, e.title, COALESCE(e.discipline, ''), COALESCE(e.level, ''), e.goal_tags,
	e.stable_id IS NULL, e.steps, e.next_exercise_id::text`

func scanExercise(row pgx.Row) (exerciseOut, error) {
	var e exerciseOut
	var steps []byte
	if err := row.Scan(&e.ID, &e.Title, &e.Discipline, &e.Level, &e.GoalTags, &e.Global, &steps, &e.NextExerciseID); err != nil {
		return e, err
	}
	_ = json.Unmarshal(steps, &e.Steps)
	e.StepCount = len(e.Steps)
	e.GoalTags = nonNil(e.GoalTags)
	return e, nil
}

// listExercises: GET /api/v1/exercises?discipline=&level=&tag= (global and own-stable rows).
func (h *handler) listExercises(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	q := r.URL.Query()
	rows, err := h.deps.Pool.Query(r.Context(), `
		SELECT `+exerciseCols+`
		FROM exercises e
		WHERE (e.stable_id IS NULL OR e.stable_id = $1)
		  AND ($2 = '' OR e.discipline = $2)
		  AND ($3 = '' OR e.level = $3)
		  AND ($4 = '' OR $4 = ANY (e.goal_tags))
		ORDER BY e.title`,
		user.StableID, q.Get("discipline"), q.Get("level"), q.Get("tag"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer rows.Close()
	out := []exerciseOut{}
	for rows.Next() {
		e, err := scanExercise(rows)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		e.Steps = nil // the list only carries the step count
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		h.fail(w, r, err)
		return
	}
	sortByProgression(out)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"exercises": out})
}

// getExercise: GET /api/v1/exercises/{id} with steps and the follow-up exercise.
func (h *handler) getExercise(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	e, err := scanExercise(h.deps.Pool.QueryRow(r.Context(), `
		SELECT `+exerciseCols+` FROM exercises e
		WHERE e.id = $1 AND (e.stable_id IS NULL OR e.stable_id = $2)`, r.PathValue("id"), user.StableID))
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "exercise not found")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if e.NextExerciseID != nil {
		if e.Next, err = h.progression(r.Context(), user.StableID, *e.NextExerciseID); err != nil {
			h.fail(w, r, err)
			return
		}
	}
	httpx.WriteJSON(w, http.StatusOK, e)
}

// exerciseVisible reports whether the exercise is global or belongs to the stable.
func (h *handler) exerciseVisible(ctx context.Context, stableID, id string) (bool, error) {
	var ok bool
	err := h.deps.Pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM exercises WHERE id = $1 AND (stable_id IS NULL OR stable_id = $2))`, id, stableID).Scan(&ok)
	if isInvalidUUID(err) {
		return false, nil
	}
	return ok, err
}

func (h *handler) progression(ctx context.Context, stableID, id string) (*progressionOut, error) {
	var p progressionOut
	err := h.deps.Pool.QueryRow(ctx, `
		SELECT id::text, title, COALESCE(discipline, ''), COALESCE(level, '') FROM exercises
		WHERE id = $1 AND (stable_id IS NULL OR stable_id = $2)`, id, stableID).
		Scan(&p.ID, &p.Title, &p.Discipline, &p.Level)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// nextProgression returns the follow-up exercise of exerciseID, or nil.
func (h *handler) nextProgression(ctx context.Context, stableID, exerciseID string) (*progressionOut, error) {
	var next *string
	err := h.deps.Pool.QueryRow(ctx, `SELECT next_exercise_id::text FROM exercises WHERE id = $1`, exerciseID).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) || next == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return h.progression(ctx, stableID, *next)
}

// exerciseDiscipline is the library discipline that belongs to an activity on this horse,
// or "" when the activity has no exercises (hack, walker, rest).
func exerciseDiscipline(a training.Activity, horseDiscipline string) string {
	switch a {
	case training.ActivityJumping:
		return "jumping"
	case training.ActivityGroundwork:
		return "groundwork"
	case training.ActivityHall, training.ActivityArena, training.ActivityLunge:
		if horseDiscipline == "jumping" {
			return "jumping"
		}
		return "dressage"
	}
	return ""
}

// sortByProgression orders exercises by discipline, level (easiest first) and then along
// the next_exercise_id chain (predecessors first); the SQL order (title) breaks the rest.
func sortByProgression(list []exerciseOut) {
	byID := make(map[string]int, len(list))
	for i, e := range list {
		byID[e.ID] = i
	}
	depth := make([]int, len(list)) // number of predecessors in the chain
	for i := range list {
		seen := map[string]bool{}
		for cur := list[i].NextExerciseID; cur != nil && !seen[*cur]; {
			seen[*cur] = true
			j, ok := byID[*cur]
			if !ok {
				break
			}
			depth[j]++
			cur = list[j].NextExerciseID
		}
	}
	idx := make([]int, len(list))
	for i := range idx {
		idx[i] = i
	}
	rank := func(l string) int {
		switch l {
		case "beginner":
			return 0
		case "intermediate":
			return 1
		case "advanced":
			return 2
		}
		return 3
	}
	sort.SliceStable(idx, func(a, b int) bool {
		x, y := list[idx[a]], list[idx[b]]
		if x.Discipline != y.Discipline {
			return x.Discipline < y.Discipline
		}
		if rx, ry := rank(x.Level), rank(y.Level); rx != ry {
			return rx < ry
		}
		return depth[idx[a]] < depth[idx[b]]
	})
	sorted := make([]exerciseOut, len(list))
	for i, j := range idx {
		sorted[i] = list[j]
	}
	copy(list, sorted)
}

// suggestExercise picks the first exercise of the discipline (easiest first, along the
// progression) that the horse has not mastered yet (no session with focus rating "Sitzt");
// if all are mastered, the last one. It returns nil when the library has nothing for the
// discipline.
func (h *handler) suggestExercise(ctx context.Context, stableID, horseID, discipline string) (*exerciseOut, error) {
	if discipline == "" {
		return nil, nil
	}
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT `+exerciseCols+`,
		       EXISTS (SELECT 1 FROM sessions s WHERE s.horse_id = $3 AND s.exercise_id = e.id AND s.focus_rating = 3)
		FROM exercises e
		WHERE (e.stable_id IS NULL OR e.stable_id = $1) AND e.discipline = $2
		ORDER BY e.title`, stableID, discipline, horseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []exerciseOut
	mastered := map[string]bool{}
	for rows.Next() {
		var e exerciseOut
		var steps []byte
		var done bool
		if err := rows.Scan(&e.ID, &e.Title, &e.Discipline, &e.Level, &e.GoalTags, &e.Global, &steps, &e.NextExerciseID, &done); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(steps, &e.Steps)
		e.StepCount = len(e.Steps)
		e.GoalTags = nonNil(e.GoalTags)
		mastered[e.ID] = done
		all = append(all, e)
	}
	if err := rows.Err(); err != nil || len(all) == 0 {
		return nil, err
	}
	sortByProgression(all)
	for i := range all {
		if !mastered[all[i].ID] {
			return &all[i], nil
		}
	}
	return &all[len(all)-1], nil
}
