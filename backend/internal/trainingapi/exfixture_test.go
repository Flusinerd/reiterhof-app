package trainingapi_test

import (
	"context"
	"fmt"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// exSpec describes an exercise row for the tests. The library of the catalog changes with the
// catalog migrations, so tests that depend on exact exercises delete the global rows and insert
// their own.
type exSpec struct {
	id        string
	library   string
	level     string
	title     string
	phase     string // default main
	intensity string // default normal
	tags      []string
	equipment []string
	requires  []string
	suits     []string
	minutes   int // 0 = none
	next      string
	own       bool // a row of the demo stable instead of a global one
}

// count runs a query that returns one number.
func (e *env) count(sql string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	return n
}

// exID returns the id of the n-th test exercise.
func exID(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-0000000e%04x", n) }

// clearLibrary removes every global exercise (sessions and slots keep their rows).
func (e *env) clearLibrary() {
	e.t.Helper()
	e.exec(`DELETE FROM exercises WHERE stable_id IS NULL`)
}

// addExercise inserts an exercise.
func (e *env) addExercise(x exSpec) {
	e.t.Helper()
	if x.phase == "" {
		x.phase = "main"
	}
	if x.intensity == "" {
		x.intensity = "normal"
	}
	var stable, next any
	if x.own {
		stable = seed.StableB
	}
	if x.next != "" {
		next = x.next
	}
	var minutes any
	if x.minutes > 0 {
		minutes = x.minutes
	}
	nn := func(l []string) []string {
		if l == nil {
			return []string{}
		}
		return l
	}
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO exercises (id, stable_id, discipline, level, goal_tags, title, steps, phase, minutes, intensity,
		                       requires, equipment, suits, summary, sources)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, '["Schritt eins", "Schritt zwei"]', $7, $8, $9,
		        $10::uuid[], $11, $12, 'Wozu', '{}')`,
		x.id, stable, x.library, x.level, nn(x.tags), x.title, x.phase, minutes, x.intensity,
		nn(x.requires), nn(x.equipment), nn(x.suits))
	if err != nil {
		e.t.Fatalf("add exercise %q: %v", x.title, err)
	}
	if next != nil {
		e.exec(`UPDATE exercises SET next_exercise_id = $2 WHERE id = $1`, x.id, next)
	}
}

// rate logs a session of the horse that practised the exercise with the focus rating (1 Schwer,
// 2 Besser, 3 Sitzt) on the day, 17:00 local time.
func (e *env) rate(horse, exercise string, rating int, day string, activity string) {
	e.t.Helper()
	e.exec(`INSERT INTO sessions (stable_id, horse_id, user_id, activity, started_at, duration_min, load_score, exercise_id, focus_rating)
	        VALUES ($1, $2, $3, $4, ($5::date + time '17:00') AT TIME ZONE 'Europe/Berlin', 40, 32, $6, $7)`,
		seed.StableB, horse, seed.UserJan, activity, day, exercise, rating)
}
