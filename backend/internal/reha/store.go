package reha

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// DB is what the queries need; *pgxpool.Pool and pgx.Tx satisfy it.
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// dismissCheckupReminders hides the stored vet checkup reminders (the pushes listed in the reminder
// center) of the given plan, or, with planID "", of every ended plan of the horse: the push names a
// date that was moved, or belongs to a plan that is over. The rows stay, dismissed, so the
// reminder job's claim keeps working.
func dismissCheckupReminders(ctx context.Context, db DB, stableID, horseID, planID string, now time.Time) error {
	_, err := db.Exec(ctx, `UPDATE reminders SET dismissed_at = $4
		WHERE stable_id = $1 AND source_table = 'reha_plans' AND dismissed_at IS NULL
		  AND source_id IN (SELECT id FROM reha_plans WHERE stable_id = $1 AND horse_id = $2::uuid
		                    AND ($3 = '' AND NOT active OR id::text = $3))`,
		stableID, horseID, planID, now)
	return err
}

// Plan is a row of reha_plans. Dates are UTC-midnight calendar dates.
type Plan struct {
	ID            string
	HorseID       string
	Diagnosis     string
	Vet           string
	Start         time.Time
	Phases        []Phase
	CheckupDate   *time.Time
	AbortCriteria string
	ObservationID *string
	Active        bool
	EndedAt       *time.Time
	CreatedAt     time.Time
}

// End is the last day of the plan (inclusive).
func (p *Plan) End() time.Time { return EndDate(p.Start, p.Phases) }

// AllowedOn is the unit the plan allows on day, nil if none.
func (p *Plan) AllowedOn(day time.Time) *Allowed { return AllowedOn(p.Start, p.Phases, day) }

const planColumns = `id::text, horse_id::text, diagnosis, COALESCE(vet, ''), start_date, phases, checkup_date,
	COALESCE(abort_criteria, ''), observation_id::text, active, ended_at, created_at`

func scanPlan(row pgx.Row) (*Plan, error) {
	var (
		p      Plan
		phases []byte
	)
	if err := row.Scan(&p.ID, &p.HorseID, &p.Diagnosis, &p.Vet, &p.Start, &phases, &p.CheckupDate,
		&p.AbortCriteria, &p.ObservationID, &p.Active, &p.EndedAt, &p.CreatedAt); err != nil {
		return nil, err
	}
	p.Start = training.Day(p.Start)
	if p.CheckupDate != nil {
		d := training.Day(*p.CheckupDate)
		p.CheckupDate = &d
	}
	list, err := ParsePhases(phases)
	if err != nil {
		// Unreadable phases (edited by hand) make the plan empty instead of failing every request.
		list = nil
	}
	p.Phases = list
	return &p, nil
}

// ActivePlan returns the horse's active plan or nil.
func ActivePlan(ctx context.Context, q DB, stableID, horseID string) (*Plan, error) {
	p, err := scanPlan(q.QueryRow(ctx, `SELECT `+planColumns+` FROM reha_plans
		WHERE stable_id = $1 AND horse_id = $2 AND active`, stableID, horseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// PlanByID returns a plan of the stable or pgx.ErrNoRows (also for a malformed id).
func PlanByID(ctx context.Context, q DB, stableID, id string) (*Plan, error) {
	p, err := scanPlan(q.QueryRow(ctx, `SELECT `+planColumns+` FROM reha_plans WHERE stable_id = $1 AND id::text = $2`, stableID, id))
	return p, err
}

// EndedPlans returns the horse's ended plans, newest first.
func EndedPlans(ctx context.Context, q DB, stableID, horseID string, limit int) ([]*Plan, error) {
	rows, err := q.Query(ctx, `SELECT `+planColumns+` FROM reha_plans
		WHERE stable_id = $1 AND horse_id = $2 AND NOT active
		ORDER BY start_date DESC, created_at DESC LIMIT $3`, stableID, horseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// DoneDay is a day marked as done.
type DoneDay struct {
	Day    time.Time
	DoneBy string // user id, empty if the user was deleted
	Name   string
}

// DoneDays returns the days of the plan marked done between from and to (inclusive), by date.
func DoneDays(ctx context.Context, q DB, stableID, planID string, from, to time.Time) (map[string]DoneDay, error) {
	rows, err := q.Query(ctx, `SELECT d.day, COALESCE(d.done_by::text, ''), COALESCE(u.name, '')
		FROM reha_days d LEFT JOIN users u ON u.id = d.done_by
		WHERE d.stable_id = $1 AND d.reha_plan_id = $2::uuid AND d.day BETWEEN $3 AND $4`, stableID, planID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]DoneDay{}
	for rows.Next() {
		var d DoneDay
		if err := rows.Scan(&d.Day, &d.DoneBy, &d.Name); err != nil {
			return nil, err
		}
		d.Day = training.Day(d.Day)
		out[d.Day.Format(DateLayout)] = d
	}
	return out, rows.Err()
}

// DateLayout is the date format of the API.
const DateLayout = "2006-01-02"

// RuleFor is the rule text for helpers on day (a stable-local calendar date): the allowed unit of
// the horse's active plan, or NoUnitText if the plan has none that day. It returns "" without an
// active plan. Requests of type exercise carry it (JAN-42).
func RuleFor(ctx context.Context, q DB, stableID, horseID string, day time.Time) (string, error) {
	p, err := ActivePlan(ctx, q, stableID, horseID)
	if err != nil || p == nil {
		return "", err
	}
	if a := p.AllowedOn(day); a != nil {
		return a.RuleText(), nil
	}
	return NoUnitText, nil
}

// PhasesJSON is the stored form of phases.
func PhasesJSON(phases []Phase) []byte {
	b, _ := json.Marshal(phases)
	return b
}
