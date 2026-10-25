package trainingapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dressage"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// Practice tests and freestyles (docs/domains/dressage-tests.md). The generator and the rules live
// in the pure package dressage; this file stores the tests and the requirement sets and maps roles
// to access: owner and admin may do everything with the tests of a horse, riders with log_sessions
// read and generate.

func (h *handler) registerDressage(route func(string, http.HandlerFunc)) {
	route("GET /api/v1/dressage/catalog", h.dressageCatalog)
	route("GET /api/v1/dressage/requirements", h.listRequirements)
	route("POST /api/v1/dressage/requirements", h.createRequirement)
	route("GET /api/v1/dressage/requirements/{id}", h.getRequirement)
	route("PUT /api/v1/dressage/requirements/{id}", h.putRequirement)
	route("DELETE /api/v1/dressage/requirements/{id}", h.deleteRequirement)
	route("POST /api/v1/horses/{id}/dressage-tests", h.generateTest)
	route("GET /api/v1/horses/{id}/dressage-tests", h.listTests)
	route("GET /api/v1/dressage-tests/{id}", h.getTest)
	route("PUT /api/v1/dressage-tests/{id}", h.putTest)
	route("DELETE /api/v1/dressage-tests/{id}", h.deleteTest)
}

// requirementOut is a saved requirement set.
type requirementOut struct {
	ID         string                 `json:"id,omitempty"`
	Class      string                 `json:"class"`
	Label      string                 `json:"label"`
	Movements  []dressage.ReqMovement `json:"movements"`
	MinSeconds int                    `json:"min_seconds"`
	MaxSeconds int                    `json:"max_seconds"`
	CreatedBy  *string                `json:"created_by,omitempty"`
	CreatedAt  *time.Time             `json:"created_at,omitempty"`
}

func (r requirementOut) pure() dressage.Requirement {
	return dressage.Requirement{Class: r.Class, Label: r.Label, Movements: r.Movements, MinSeconds: r.MinSeconds, MaxSeconds: r.MaxSeconds}
}

func scanRequirement(row pgx.Row) (requirementOut, error) {
	var r requirementOut
	var mv []byte
	var minS, maxS *int
	var created time.Time
	if err := row.Scan(&r.ID, &r.Class, &r.Label, &mv, &minS, &maxS, &r.CreatedBy, &created); err != nil {
		return r, err
	}
	r.CreatedAt = &created
	_ = json.Unmarshal(mv, &r.Movements)
	if r.Movements == nil {
		r.Movements = []dressage.ReqMovement{}
	}
	if minS != nil {
		r.MinSeconds = *minS
	}
	if maxS != nil {
		r.MaxSeconds = *maxS
	}
	return r, nil
}

const requirementCols = `id::text, class, label, movements, min_seconds, max_seconds, created_by::text, created_at`

func (h *handler) requirementsOf(ctx context.Context, stableID string) ([]requirementOut, error) {
	rows, err := h.deps.Pool.Query(ctx, `SELECT `+requirementCols+` FROM freestyle_requirements WHERE stable_id = $1 ORDER BY class, label, created_at`, stableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []requirementOut{}
	for rows.Next() {
		r, err := scanRequirement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// dressageCatalog is the static catalog (arenas with letters, classes, movements with their lowest
// class and source, speeds) plus the saved requirement sets of the stable.
func (h *handler) dressageCatalog(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	sets, err := h.requirementsOf(r.Context(), user.StableID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		dressage.CatalogInfo
		RequirementSets []requirementOut `json:"requirement_sets"`
	}{dressage.Catalog(), sets})
}

// canWriteRequirements: admins, owners of a horse of the stable and riders with log_sessions.
func (h *handler) canWriteRequirements(ctx context.Context, user auth.User) (bool, error) {
	if user.IsAdmin {
		return true, nil
	}
	var ok bool
	err := h.deps.Pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM horses WHERE stable_id = $1 AND owner_id = $2)
		    OR EXISTS (SELECT 1 FROM horse_riders WHERE stable_id = $1 AND user_id = $2 AND rules ?| ARRAY['log_sessions', 'ride'])`,
		user.StableID, user.ID).Scan(&ok)
	return ok, err
}

func (h *handler) requireRequirementWriter(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, _ := auth.UserFrom(r.Context())
	ok, err := h.canWriteRequirements(r.Context(), user)
	if err != nil {
		h.fail(w, r, err)
		return user, false
	}
	if !ok {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "only owners, admins and riders who may log sessions can change requirement sets")
		return user, false
	}
	return user, true
}

func (h *handler) listRequirements(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	sets, err := h.requirementsOf(r.Context(), user.StableID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"requirements": sets})
}

// requirementBody is the body of POST and PUT of a requirement set.
type requirementBody struct {
	Class      string                 `json:"class"`
	Label      string                 `json:"label"`
	Movements  []dressage.ReqMovement `json:"movements"`
	MinSeconds int                    `json:"min_seconds"`
	MaxSeconds int                    `json:"max_seconds"`
}

func (b requirementBody) validate(w http.ResponseWriter) bool {
	b.Label = strings.TrimSpace(b.Label)
	if b.Label == "" {
		invalid(w, "label is required")
		return false
	}
	if errs := dressage.ValidateRequirement(dressage.Requirement{Class: b.Class, Label: b.Label, Movements: b.Movements, MinSeconds: b.MinSeconds, MaxSeconds: b.MaxSeconds}); len(errs) > 0 {
		reasons(w, http.StatusBadRequest, "validation_failed", errs)
		return false
	}
	return true
}

// reasons answers with the error envelope plus the list of German reasons.
func reasons(w http.ResponseWriter, status int, code string, why []string) {
	httpx.WriteJSON(w, status, struct {
		httpx.ErrorBody
		Reasons []string `json:"reasons"`
	}{httpx.ErrorBody{Error: httpx.ErrorDetail{Code: code, Message: strings.Join(why, " ")}}, why})
}

func (h *handler) createRequirement(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireRequirementWriter(w, r)
	if !ok {
		return
	}
	var b requirementBody
	if !httpx.ReadJSON(w, r, &b) || !b.validate(w) {
		return
	}
	mv, _ := json.Marshal(nonNilMovements(b.Movements))
	row := h.deps.Pool.QueryRow(r.Context(), `
		INSERT INTO freestyle_requirements (stable_id, class, label, movements, min_seconds, max_seconds, created_by)
		VALUES ($1, $2, $3, $4, NULLIF($5, 0), NULLIF($6, 0), $7) RETURNING `+requirementCols,
		user.StableID, b.Class, strings.TrimSpace(b.Label), mv, b.MinSeconds, b.MaxSeconds, user.ID)
	out, err := scanRequirement(row)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, out)
}

func nonNilMovements(m []dressage.ReqMovement) []dressage.ReqMovement {
	if m == nil {
		return []dressage.ReqMovement{}
	}
	return m
}

func (h *handler) loadRequirement(ctx context.Context, stableID, id string) (requirementOut, error) {
	out, err := scanRequirement(h.deps.Pool.QueryRow(ctx, `SELECT `+requirementCols+` FROM freestyle_requirements WHERE id = $1 AND stable_id = $2`, id, stableID))
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return out, errNotFound
	}
	return out, err
}

func (h *handler) getRequirement(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	out, err := h.loadRequirement(r.Context(), user.StableID, r.PathValue("id"))
	if errors.Is(err, errNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "requirement set not found")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *handler) putRequirement(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireRequirementWriter(w, r)
	if !ok {
		return
	}
	var b requirementBody
	if !httpx.ReadJSON(w, r, &b) || !b.validate(w) {
		return
	}
	mv, _ := json.Marshal(nonNilMovements(b.Movements))
	out, err := scanRequirement(h.deps.Pool.QueryRow(r.Context(), `
		UPDATE freestyle_requirements SET class = $3, label = $4, movements = $5, min_seconds = NULLIF($6, 0),
		       max_seconds = NULLIF($7, 0), updated_at = now()
		WHERE id = $1 AND stable_id = $2 RETURNING `+requirementCols,
		r.PathValue("id"), user.StableID, b.Class, strings.TrimSpace(b.Label), mv, b.MinSeconds, b.MaxSeconds))
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "requirement set not found")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *handler) deleteRequirement(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireRequirementWriter(w, r)
	if !ok {
		return
	}
	tag, err := h.deps.Pool.Exec(r.Context(), `DELETE FROM freestyle_requirements WHERE id = $1 AND stable_id = $2`, r.PathValue("id"), user.StableID)
	if isInvalidUUID(err) || (err == nil && tag.RowsAffected() == 0) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "requirement set not found")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// testOut is a test in the answers; ID is nil for a preview that is not stored.
type testOut struct {
	ID           *string         `json:"id"`
	HorseID      string          `json:"horse_id"`
	Kind         string          `json:"kind"`
	Class        string          `json:"class"`
	Arena        string          `json:"arena"`
	Title        string          `json:"title"`
	FocusKey     *string         `json:"focus_key"`
	Seed         int64           `json:"seed"`
	Requirements *requirementOut `json:"requirements"`
	Plan         dressage.Plan   `json:"plan"`
	Saved        bool            `json:"saved"`
	CreatedBy    *string         `json:"created_by,omitempty"`
	CreatedAt    *time.Time      `json:"created_at,omitempty"`
	UpdatedAt    *time.Time      `json:"updated_at,omitempty"`
}

// testSummary is a row of the list: without the plan.
type testSummary struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Class     string    `json:"class"`
	Arena     string    `json:"arena"`
	Title     string    `json:"title"`
	FocusKey  *string   `json:"focus_key"`
	Seconds   float64   `json:"seconds"`
	Lines     int       `json:"lines"`
	CreatedBy *string   `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// canDressage: owner, admin, or a rider with log_sessions.
func (a access) canDressage() bool { return a.canLog() }

// authorizeDressage resolves the horse of the path and checks the access to its tests.
func (h *handler) authorizeDressage(w http.ResponseWriter, r *http.Request) (access, bool) {
	a, ok := h.authorize(w, r)
	if !ok {
		return a, false
	}
	if !a.canDressage() {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner, admins and riders who may log sessions can see the tests of this horse")
		return a, false
	}
	return a, true
}

type generateBody struct {
	Kind           string                  `json:"kind"`
	Class          string                  `json:"class"`
	Arena          string                  `json:"arena"`
	FocusKey       string                  `json:"focus_key"`
	Length         string                  `json:"length"`
	Seed           *int64                  `json:"seed"`
	Save           bool                    `json:"save"`
	Title          string                  `json:"title"`
	RequirementsID string                  `json:"requirements_id"`
	Requirements   *requirementInlineInput `json:"requirements"`
}

// requirementInlineInput is a requirement set sent with the request instead of a saved one.
type requirementInlineInput struct {
	Label      string                 `json:"label"`
	Movements  []dressage.ReqMovement `json:"movements"`
	MinSeconds int                    `json:"min_seconds"`
	MaxSeconds int                    `json:"max_seconds"`
}

func (h *handler) generateTest(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorizeDressage(w, r)
	if !ok {
		return
	}
	var b generateBody
	if !httpx.ReadJSON(w, r, &b) {
		return
	}
	if b.Kind != dressage.KindPractice && b.Kind != dressage.KindFreestyle {
		invalid(w, "kind must be practice or freestyle")
		return
	}
	if dressage.ClassRank(b.Class) < 0 {
		invalid(w, "class is unknown")
		return
	}
	if b.Kind == dressage.KindPractice && !slices.Contains(dressage.PracticeClasses, b.Class) {
		invalid(w, "practice tests exist for the classes "+strings.Join(dressage.PracticeClasses, ", "))
		return
	}
	if !slices.Contains(dressage.Sizes, b.Arena) {
		invalid(w, "arena must be 20x40 or 20x60")
		return
	}
	if b.Length != "" && b.Length != dressage.LengthShort && b.Length != dressage.LengthNormal {
		invalid(w, "length must be short or normal")
		return
	}
	if b.FocusKey != "" && !training.ValidFocusKey(b.FocusKey) {
		invalid(w, "unknown focus")
		return
	}
	if len([]rune(b.Title)) > 120 {
		invalid(w, "title is too long")
		return
	}
	seed := h.deps.Now().UnixNano() % (1 << 31)
	if b.Seed != nil {
		seed = *b.Seed
	}

	var plan dressage.Plan
	var req *requirementOut
	var err error
	switch b.Kind {
	case dressage.KindPractice:
		hints, herr := h.dressageHints(r.Context(), a, b.FocusKey)
		if herr != nil {
			h.fail(w, r, herr)
			return
		}
		plan, err = dressage.GeneratePractice(dressage.PracticeParams{Class: b.Class, Arena: b.Arena, Length: b.Length, Seed: seed, Hints: hints})
	default:
		var done bool
		req, done = h.freestyleRequirement(w, r, a, b)
		if done {
			return
		}
		plan, err = dressage.GenerateFreestyle(dressage.FreestyleParams{Class: b.Class, Arena: b.Arena, Seed: seed, Requirement: req.pure()})
	}
	if err != nil {
		h.deps.Log.WarnContext(r.Context(), "dressage generator failed", "err", err, "kind", b.Kind, "class", b.Class, "arena", b.Arena)
		httpx.WriteError(w, http.StatusUnprocessableEntity, "no_plan", "Für diese Eingaben ließ sich keine Aufgabe erstellen.")
		return
	}
	title := strings.TrimSpace(b.Title)
	if title == "" {
		title = dressage.Title(b.Kind, b.Class, b.Arena)
	}
	out := testOut{HorseID: a.horseID, Kind: b.Kind, Class: b.Class, Arena: b.Arena, Title: title, Seed: seed, Requirements: req, Plan: plan}
	if b.FocusKey != "" {
		out.FocusKey = &b.FocusKey
	}
	if !b.Save {
		httpx.WriteJSON(w, http.StatusOK, out)
		return
	}
	planJSON, _ := json.Marshal(plan)
	var reqJSON []byte
	if req != nil {
		reqJSON, _ = json.Marshal(req.pure())
	}
	var id string
	var created, updated time.Time
	err = h.deps.Pool.QueryRow(r.Context(), `
		INSERT INTO dressage_tests (stable_id, horse_id, created_by, kind, class, arena, title, focus_key, seed, requirements, plan)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9, $10, $11)
		RETURNING id::text, created_at, updated_at`,
		a.user.StableID, a.horseID, a.user.ID, b.Kind, b.Class, b.Arena, title, b.FocusKey, seed, reqJSON, planJSON).Scan(&id, &created, &updated)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	uid := a.user.ID
	out.ID, out.Saved, out.CreatedBy, out.CreatedAt, out.UpdatedAt = &id, true, &uid, &created, &updated
	httpx.WriteJSON(w, http.StatusCreated, out)
}

// freestyleRequirement resolves the requirement set of a freestyle request: a saved one, an inline
// one, or none (a freestyle without required movements). done is true when it wrote an error.
func (h *handler) freestyleRequirement(w http.ResponseWriter, r *http.Request, a access, b generateBody) (*requirementOut, bool) {
	switch {
	case b.RequirementsID != "" && b.Requirements != nil:
		invalid(w, "send requirements_id or requirements, not both")
		return nil, true
	case b.RequirementsID != "":
		set, err := h.loadRequirement(r.Context(), a.user.StableID, b.RequirementsID)
		if errors.Is(err, errNotFound) {
			invalid(w, "requirement set not found")
			return nil, true
		}
		if err != nil {
			h.fail(w, r, err)
			return nil, true
		}
		if set.Class != b.Class {
			invalid(w, "the requirement set belongs to class "+set.Class)
			return nil, true
		}
		return &set, false
	case b.Requirements != nil:
		set := requirementOut{Class: b.Class, Label: b.Requirements.Label, Movements: nonNilMovements(b.Requirements.Movements),
			MinSeconds: b.Requirements.MinSeconds, MaxSeconds: b.Requirements.MaxSeconds}
		if errs := dressage.ValidateRequirement(set.pure()); len(errs) > 0 {
			reasons(w, http.StatusBadRequest, "validation_failed", errs)
			return nil, true
		}
		return &set, false
	}
	return &requirementOut{Class: b.Class, Movements: []dressage.ReqMovement{}}, false
}

// dressageHints reads what the generator needs to know about the horse: the lessons it mastered
// (a session rated 3 for an exercise), the next lessons of its level goal and, for a focus, the
// exercises that carry one of the focus's tags.
func (h *handler) dressageHints(ctx context.Context, a access, focusKey string) (dressage.Hints, error) {
	lib, err := h.loadLibrary(ctx, a.user.StableID, a.horseID, nil)
	if err != nil {
		return dressage.Hints{}, err
	}
	hints := dressage.Hints{HasSessions: len(lib.rated) > 0, Mastered: lib.mastered, Next: map[string]bool{}, FocusExercises: map[string]bool{}}
	p, err := h.loadProfile(ctx, a.user.StableID, a.horseID)
	if err != nil {
		return hints, err
	}
	today := training.Day(h.deps.Now().In(h.location(ctx, a.user.StableID)))
	prog, _, err := h.progress(ctx, a, p, today)
	if err != nil {
		return hints, err
	}
	if prog != nil {
		for _, n := range prog.Next {
			hints.Next[n.ID] = true
		}
	}
	if f, ok := training.FocusByKey(focusKey); ok {
		for _, row := range lib.rows {
			if row.pure().SharedTags(f.Tags) > 0 {
				hints.FocusExercises[row.ID] = true
			}
		}
	}
	return hints, nil
}

func (h *handler) listTests(w http.ResponseWriter, r *http.Request) {
	a, ok := h.authorizeDressage(w, r)
	if !ok {
		return
	}
	rows, err := h.deps.Pool.Query(r.Context(), `
		SELECT id::text, kind, class, arena, title, focus_key, COALESCE((plan->>'seconds')::float8, 0),
		       COALESCE(jsonb_array_length(plan->'lines'), 0), created_by::text, created_at
		FROM dressage_tests WHERE stable_id = $1 AND horse_id = $2 ORDER BY created_at DESC, id LIMIT 200`, a.user.StableID, a.horseID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer rows.Close()
	out := []testSummary{}
	for rows.Next() {
		var s testSummary
		if err := rows.Scan(&s.ID, &s.Kind, &s.Class, &s.Arena, &s.Title, &s.FocusKey, &s.Seconds, &s.Lines, &s.CreatedBy, &s.CreatedAt); err != nil {
			h.fail(w, r, err)
			return
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tests": out})
}

// storedTest is a dressage_tests row.
type storedTest struct {
	testOut
	stableID string
	reqJSON  []byte
	planJSON []byte
}

func (h *handler) loadTest(ctx context.Context, user auth.User, id string) (storedTest, error) {
	var t storedTest
	var tid string
	var created, updated time.Time
	err := h.deps.Pool.QueryRow(ctx, `
		SELECT id::text, stable_id::text, horse_id::text, created_by::text, kind, class, arena, title, focus_key, seed, requirements, plan, created_at, updated_at
		FROM dressage_tests WHERE id = $1 AND stable_id = $2`, id, user.StableID).
		Scan(&tid, &t.stableID, &t.HorseID, &t.CreatedBy, &t.Kind, &t.Class, &t.Arena, &t.Title, &t.FocusKey, &t.Seed, &t.reqJSON, &t.planJSON, &created, &updated)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return t, errNotFound
	}
	if err != nil {
		return t, err
	}
	t.ID, t.Saved, t.CreatedAt, t.UpdatedAt = &tid, true, &created, &updated
	if err := json.Unmarshal(t.planJSON, &t.Plan); err != nil {
		return t, err
	}
	if len(t.reqJSON) > 0 && string(t.reqJSON) != "null" {
		var pr dressage.Requirement
		if err := json.Unmarshal(t.reqJSON, &pr); err == nil {
			t.Requirements = &requirementOut{Class: pr.Class, Label: pr.Label, Movements: nonNilMovements(pr.Movements), MinSeconds: pr.MinSeconds, MaxSeconds: pr.MaxSeconds}
		}
	}
	return t, nil
}

// testAccess loads the test of the path and the caller's access to its horse (404 for a test of
// another stable, 403 without the rights of a rider who may log sessions).
func (h *handler) testAccess(w http.ResponseWriter, r *http.Request) (storedTest, access, bool) {
	user, _ := auth.UserFrom(r.Context())
	t, err := h.loadTest(r.Context(), user, r.PathValue("id"))
	if errors.Is(err, errNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "test not found")
		return t, access{}, false
	}
	if err != nil {
		h.fail(w, r, err)
		return t, access{}, false
	}
	a, err := h.resolve(r.Context(), user, t.HorseID)
	if err != nil {
		h.fail(w, r, err)
		return t, a, false
	}
	if !a.canDressage() {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner, admins and riders who may log sessions can see the tests of this horse")
		return t, a, false
	}
	return t, a, true
}

func (h *handler) getTest(w http.ResponseWriter, r *http.Request) {
	t, _, ok := h.testAccess(w, r)
	if !ok {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t.testOut)
}

type putTestBody struct {
	Title *string         `json:"title"`
	Plan  json.RawMessage `json:"plan"`
}

// putTest stores the user's edits: the lines of the plan in their new order, with lines removed.
// The plan is rebuilt from the figure library (only the option, gait and tempo of a line count),
// checked with the rules of its kind, and rejected with the German reasons when a rule breaks.
func (h *handler) putTest(w http.ResponseWriter, r *http.Request) {
	t, _, ok := h.testAccess(w, r)
	if !ok {
		return
	}
	var b putTestBody
	if !httpx.ReadJSON(w, r, &b) {
		return
	}
	title := t.Title
	if b.Title != nil {
		title = strings.TrimSpace(*b.Title)
		if title == "" || len([]rune(title)) > 120 {
			invalid(w, "title must have 1 to 120 characters")
			return
		}
	}
	plan := t.Plan
	if len(b.Plan) > 0 && string(b.Plan) != "null" {
		var in struct {
			Lines []dressage.Spec `json:"lines"`
		}
		if err := json.Unmarshal(b.Plan, &in); err != nil {
			invalid(w, "plan is malformed")
			return
		}
		var req dressage.Requirement
		if t.Requirements != nil {
			req = t.Requirements.pure()
		}
		built, err := dressage.Evaluate(t.Kind, t.Class, t.Arena, t.Seed, in.Lines, req)
		if err != nil {
			reasons(w, http.StatusBadRequest, "rules_broken", []string{err.Error()})
			return
		}
		if why := dressage.RuleErrors(built.Checks); len(why) > 0 {
			reasons(w, http.StatusBadRequest, "rules_broken", why)
			return
		}
		plan = built
	}
	planJSON, _ := json.Marshal(plan)
	var updated time.Time
	if err := h.deps.Pool.QueryRow(r.Context(), `UPDATE dressage_tests SET title = $2, plan = $3, updated_at = now() WHERE id = $1 RETURNING updated_at`,
		*t.ID, title, planJSON).Scan(&updated); err != nil {
		h.fail(w, r, err)
		return
	}
	t.Title, t.Plan, t.UpdatedAt = title, plan, &updated
	httpx.WriteJSON(w, http.StatusOK, t.testOut)
}

func (h *handler) deleteTest(w http.ResponseWriter, r *http.Request) {
	t, a, ok := h.testAccess(w, r)
	if !ok {
		return
	}
	if !a.manage && (t.CreatedBy == nil || *t.CreatedBy != a.user.ID) {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "only the owner, admins and the author can delete a test")
		return
	}
	if _, err := h.deps.Pool.Exec(r.Context(), `DELETE FROM dressage_tests WHERE id = $1`, *t.ID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
