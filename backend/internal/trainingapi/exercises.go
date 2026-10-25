package trainingapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

type exerciseOut struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Discipline is the library of the exercise; Library says the same (the column kept its name).
	Discipline string   `json:"discipline"`
	Library    string   `json:"library"`
	Level      string   `json:"level"`
	GoalTags   []string `json:"goal_tags"`
	Global     bool     `json:"global"`
	StepCount  int      `json:"step_count"`
	Steps      []string `json:"steps,omitempty"`
	// NextExerciseID and Next describe the follow-up exercise of the progression.
	NextExerciseID *string         `json:"next_exercise_id"`
	Next           *progressionOut `json:"next,omitempty"`
	// Phase is the part of the unit the exercise belongs to: warmup (Lösen), main (Arbeiten),
	// cooldown (Entspannen) or any; Minutes its typical duration (null when unknown).
	Phase     string   `json:"phase"`
	Minutes   *int     `json:"minutes"`
	Intensity string   `json:"intensity"`
	Suits     []string `json:"suits"`
	Summary   string   `json:"summary"`
	Equipment []string `json:"equipment"`
	// Available tells whether the stable has the facility equipment the exercise needs; only
	// with ?horse=.
	Available *bool `json:"available,omitempty"`
}

// exerciseRef is an exercise as a link: id and title.
type exerciseRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// exerciseDetailOut is the answer of GET /exercises/{id}.
type exerciseDetailOut struct {
	exerciseOut
	Requires     []exerciseRef    `json:"requires"`
	Sources      []string         `json:"sources"`
	Alternatives []alternativeOut `json:"alternatives"`
}

// alternativeOut is an exercise that replaces one that needs set-up.
type alternativeOut struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Equipment []string `json:"equipment"`
}

// exerciseRow is a row of the exercise library with what the planner needs besides the JSON
// output.
type exerciseRow struct {
	exerciseOut
	requires []string
	sources  []string
}

// pure is the pure view of the row for the planner; Mastered and Rated are the caller's business.
func (r exerciseRow) pure() training.Exercise {
	ex := training.Exercise{
		ID: r.ID, Library: r.Library, Level: r.Level, Title: r.Title, Tags: r.GoalTags, Phase: r.Phase,
		Intensity: r.Intensity, Requires: r.requires, Equipment: r.Equipment, Suits: r.Suits, Own: !r.Global,
	}
	if r.Minutes != nil {
		ex.Minutes = *r.Minutes
	}
	if r.NextExerciseID != nil {
		ex.NextID = *r.NextExerciseID
	}
	return ex
}

const exerciseCols = `e.id::text, e.title, COALESCE(e.discipline, ''), COALESCE(e.level, ''), e.goal_tags,
	e.stable_id IS NULL, e.steps, e.next_exercise_id::text,
	e.phase, e.minutes, e.intensity, e.requires::text[], e.equipment, e.suits, e.summary, e.sources`

// scanExercise scans exerciseCols (plus the extra destinations, e.g. the flags of a horse).
func scanExercise(row pgx.Row, extra ...any) (exerciseRow, error) {
	var r exerciseRow
	var steps []byte
	dest := append([]any{&r.ID, &r.Title, &r.Discipline, &r.Level, &r.GoalTags, &r.Global, &steps, &r.NextExerciseID,
		&r.Phase, &r.Minutes, &r.Intensity, &r.requires, &r.Equipment, &r.Suits, &r.Summary, &r.sources}, extra...)
	if err := row.Scan(dest...); err != nil {
		return r, err
	}
	_ = json.Unmarshal(steps, &r.Steps)
	r.StepCount = len(r.Steps)
	r.Library = r.Discipline
	r.GoalTags = nonNil(r.GoalTags)
	r.Suits = nonNil(r.Suits)
	r.Equipment = nonNil(r.Equipment)
	r.requires = nonNil(r.requires)
	r.sources = nonNil(r.sources)
	return r, nil
}

// exerciseFilter holds the filters of GET /exercises.
type exerciseFilter struct {
	library, level, tag, phase string
	focusTags                  []string
	noMatch                    bool // an unknown focus: nothing matches
}

// listExercises: GET /api/v1/exercises?library=&level=&tag=&phase=&focus=&horse= (global and
// own-stable rows). discipline is the older name of library. phase warmup, main or cooldown also
// lists the exercises of phase any. focus is a catalog key and matches the focus's tags. With
// horse the items say whether the stable has their equipment.
func (h *handler) listExercises(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	q := r.URL.Query()
	f := exerciseFilter{library: q.Get("library"), level: q.Get("level"), tag: q.Get("tag"), phase: q.Get("phase")}
	if f.library == "" {
		f.library = q.Get("discipline")
	}
	if key := q.Get("focus"); key != "" {
		if focus, ok := training.FocusByKey(key); ok {
			f.focusTags = focus.Tags
		} else {
			f.noMatch = true
		}
	}
	out := []exerciseOut{}
	if !f.noMatch {
		rows, err := h.queryExercises(r.Context(), user.StableID, f)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		for _, row := range rows {
			row.Steps = nil // the list only carries the step count
			out = append(out, row.exerciseOut)
		}
	}
	if horseID := q.Get("horse"); horseID != "" {
		a, err := h.resolve(r.Context(), user, horseID)
		switch {
		case errors.Is(err, errNotFound):
			httpx.WriteError(w, http.StatusNotFound, "not_found", "horse not found")
			return
		case err != nil:
			h.fail(w, r, err)
			return
		case !a.canRead():
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner, admins and riders can see the training of this horse")
			return
		}
		prof, err := h.loadProfile(r.Context(), user.StableID, horseID)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		cfg, _, err := h.loadEquipment(r.Context(), user.StableID)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		for i := range out {
			ok := equipmentAvailable(exerciseRow{exerciseOut: out[i]}.pure(), prof.discipline, cfg)
			out[i].Available = &ok
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"exercises": out})
}

// queryExercises returns the exercises of the stable (global and own) that match the filter, in
// progression order.
func (h *handler) queryExercises(ctx context.Context, stableID string, f exerciseFilter) ([]exerciseRow, error) {
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT `+exerciseCols+`
		FROM exercises e
		WHERE (e.stable_id IS NULL OR e.stable_id = $1)
		  AND ($2 = '' OR e.discipline = $2)
		  AND ($3 = '' OR e.level = $3)
		  AND ($4 = '' OR $4 = ANY (e.goal_tags))
		  AND ($5 = '' OR e.phase = $5 OR (e.phase = 'any' AND $5 IN ('warmup', 'main', 'cooldown')))
		  AND (cardinality($6::text[]) = 0 OR e.goal_tags && $6::text[])
		ORDER BY e.title`,
		stableID, f.library, f.level, f.tag, f.phase, nonNil(f.focusTags))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []exerciseRow
	for rows.Next() {
		e, err := scanExercise(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortByProgression(out)
	return out, nil
}

// getExercise: GET /api/v1/exercises/{id} with steps, the follow-up exercise, the prerequisites,
// the sources and the alternatives that need no set-up.
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
	out := exerciseDetailOut{exerciseOut: e.exerciseOut, Requires: []exerciseRef{}, Sources: e.sources, Alternatives: []alternativeOut{}}
	ctx := r.Context()
	if e.NextExerciseID != nil {
		if out.Next, err = h.progression(ctx, user.StableID, *e.NextExerciseID); err != nil {
			h.fail(w, r, err)
			return
		}
	}
	if len(e.requires) > 0 {
		if out.Requires, err = h.exerciseRefs(ctx, user.StableID, e.requires); err != nil {
			h.fail(w, r, err)
			return
		}
	}
	if len(e.pure().FacilityEquipment()) > 0 {
		pool, err := h.queryExercises(ctx, user.StableID, exerciseFilter{library: e.Library})
		if err != nil {
			h.fail(w, r, err)
			return
		}
		var list []training.Exercise
		for _, p := range pool {
			list = append(list, p.pure())
		}
		for _, alt := range training.Alternatives(e.pure(), list, maxAlternatives) {
			out.Alternatives = append(out.Alternatives, alternativeOut{ID: alt.ID, Title: alt.Title, Equipment: nonNil(alt.Equipment)})
		}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// maxAlternatives is the number of alternatives GET /exercises/{id} lists.
const maxAlternatives = 5

// exerciseRefs returns id and title of the given exercises that the stable can see, in the order
// of ids.
func (h *handler) exerciseRefs(ctx context.Context, stableID string, ids []string) ([]exerciseRef, error) {
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT id::text, title FROM exercises WHERE id = ANY ($1::uuid[]) AND (stable_id IS NULL OR stable_id = $2)`, ids, stableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[string]exerciseRef{}
	for rows.Next() {
		var ref exerciseRef
		if err := rows.Scan(&ref.ID, &ref.Title); err != nil {
			return nil, err
		}
		byID[ref.ID] = ref
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := []exerciseRef{}
	for _, id := range ids {
		if ref, ok := byID[id]; ok {
			out = append(out, ref)
		}
	}
	return out, nil
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

// equipmentAvailable reports whether the facility equipment an exercise needs is available in at
// least one location where its library is used, with the stable's defaults and no override of
// the day. An exercise that needs no facility equipment is always available.
func equipmentAvailable(ex training.Exercise, horseDiscipline string, cfg training.StableEquipment) bool {
	if len(ex.FacilityEquipment()) == 0 {
		return true
	}
	for _, loc := range training.Locations {
		if !slices.Contains(training.ExerciseLibraries(loc, horseDiscipline), ex.Library) {
			continue
		}
		if training.FitsEquipment(ex, training.Available(cfg.Of(loc), nil)) {
			return true
		}
	}
	return false
}

// library is the exercise library as the planner sees it for one horse: the exercises of some
// libraries in progression order and what the horse did with them.
type library struct {
	rows     []exerciseRow
	mastered map[string]bool // a session rated 3 ("Sitzt")
	rated    map[string]bool // a session with any rating
}

// pure returns the exercises for the planner with the horse's flags set.
func (l library) pure() []training.Exercise {
	out := make([]training.Exercise, len(l.rows))
	for i, r := range l.rows {
		ex := r.pure()
		ex.Mastered, ex.Rated = l.mastered[ex.ID], l.rated[ex.ID]
		out[i] = ex
	}
	return out
}

// masteredSessions are one session per mastered exercise, the form training.Progress reads.
func (l library) masteredSessions() []training.Session {
	var out []training.Session
	for id, ok := range l.mastered {
		if ok {
			out = append(out, training.Session{ExerciseID: id, FocusRating: 3})
		}
	}
	return out
}

// loadLibrary reads the exercises of the libraries (every library when libs is empty) that the
// stable can see, in progression order, with the horse's mastery. Own-stable rows are included.
func (h *handler) loadLibrary(ctx context.Context, stableID, horseID string, libs []string) (library, error) {
	rows, err := h.deps.Pool.Query(ctx, `
		SELECT `+exerciseCols+`,
		       EXISTS (SELECT 1 FROM sessions s WHERE s.horse_id = $2 AND s.exercise_id = e.id AND s.focus_rating = 3),
		       EXISTS (SELECT 1 FROM sessions s WHERE s.horse_id = $2 AND s.exercise_id = e.id AND s.focus_rating IS NOT NULL)
		FROM exercises e
		WHERE (e.stable_id IS NULL OR e.stable_id = $1) AND (cardinality($3::text[]) = 0 OR e.discipline = ANY ($3::text[]))
		ORDER BY e.title`, stableID, horseID, nonNil(libs))
	if err != nil {
		return library{}, err
	}
	defer rows.Close()
	lib := library{mastered: map[string]bool{}, rated: map[string]bool{}}
	for rows.Next() {
		var mastered, rated bool
		e, err := scanExercise(rows, &mastered, &rated)
		if err != nil {
			return library{}, err
		}
		lib.mastered[e.ID], lib.rated[e.ID] = mastered, rated
		lib.rows = append(lib.rows, e)
	}
	if err := rows.Err(); err != nil {
		return library{}, err
	}
	sortByProgression(lib.rows)
	return lib, nil
}

// sortByProgression orders exercises by library, level (easiest first) and then along the
// next_exercise_id chain (predecessors first); the SQL order (title) breaks the rest.
func sortByProgression(list []exerciseRow) {
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
		if r := training.LevelRank(l); r >= 0 {
			return r
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
	sorted := make([]exerciseRow, len(list))
	for i, j := range idx {
		sorted[i] = list[j]
	}
	copy(list, sorted)
}

// exerciseDiscipline is the home library of an activity on this horse, or "" when the activity
// has no exercises (walker, rest).
func exerciseDiscipline(a training.Activity, horseDiscipline string) string {
	if libs := training.ExerciseLibraries(a, horseDiscipline); len(libs) > 0 {
		return libs[0]
	}
	return ""
}

// suggestExercise picks the first exercise of the library (easiest first, along the
// progression) that the horse has not mastered yet (no session with focus rating "Sitzt"); if
// all are mastered, the last one. It returns nil when the library has nothing.
func (h *handler) suggestExercise(ctx context.Context, stableID, horseID, discipline string) (*exerciseOut, error) {
	if discipline == "" {
		return nil, nil
	}
	lib, err := h.loadLibrary(ctx, stableID, horseID, []string{discipline})
	if err != nil || len(lib.rows) == 0 {
		return nil, err
	}
	for i := range lib.rows {
		if !lib.mastered[lib.rows[i].ID] {
			return &lib.rows[i].exerciseOut, nil
		}
	}
	return &lib.rows[len(lib.rows)-1].exerciseOut, nil
}
