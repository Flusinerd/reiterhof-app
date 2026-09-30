// Package stables holds stable-level data access that is not tied to an HTTP
// endpoint yet: the ground condition and the list of stables with a location.
package stables

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Ground condition values (stables.ground_condition CHECK constraint).
const (
	GroundDry    = "dry"
	GroundWet    = "wet"
	GroundFrozen = "frozen"
	GroundMuddy  = "muddy"
)

var (
	// ErrNotFound is returned when the stable does not exist.
	ErrNotFound = errors.New("stables: not found")
	// ErrInvalidCondition is returned for an unknown ground condition.
	ErrInvalidCondition = errors.New("stables: invalid ground condition")
)

// Ground is the current ground condition. Condition is empty and UpdatedAt nil
// when it was never set.
type Ground struct {
	Condition string
	UpdatedAt *time.Time
}

// ValidCondition reports whether c is a known ground condition.
func ValidCondition(c string) bool {
	switch c {
	case GroundDry, GroundWet, GroundFrozen, GroundMuddy:
		return true
	}
	return false
}

// SetGroundCondition stores the condition and the update time (now).
func SetGroundCondition(ctx context.Context, pool *pgxpool.Pool, stableID, condition string, now time.Time) error {
	if !ValidCondition(condition) {
		return ErrInvalidCondition
	}
	tag, err := pool.Exec(ctx,
		`UPDATE stables SET ground_condition = $2, ground_condition_updated_at = $3 WHERE id = $1`,
		stableID, condition, now)
	if err != nil {
		return fmt.Errorf("stables: set ground condition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetGroundCondition returns the current ground condition of the stable.
func GetGroundCondition(ctx context.Context, pool *pgxpool.Pool, stableID string) (Ground, error) {
	var c *string
	var g Ground
	err := pool.QueryRow(ctx,
		`SELECT ground_condition, ground_condition_updated_at FROM stables WHERE id = $1`, stableID).
		Scan(&c, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ground{}, ErrNotFound
	}
	if err != nil {
		return Ground{}, fmt.Errorf("stables: get ground condition: %w", err)
	}
	if c != nil {
		g.Condition = *c
	}
	return g, nil
}

// Located is a stable with coordinates, as needed by the weather job.
type Located struct {
	ID       string
	Lat, Lng float64
	Timezone string
}

// ListLocated returns all stables that have both lat and lng.
func ListLocated(ctx context.Context, pool *pgxpool.Pool) ([]Located, error) {
	rows, err := pool.Query(ctx,
		`SELECT id::text, lat, lng, timezone FROM stables WHERE lat IS NOT NULL AND lng IS NOT NULL ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("stables: list located: %w", err)
	}
	defer rows.Close()
	var out []Located
	for rows.Next() {
		var l Located
		if err := rows.Scan(&l.ID, &l.Lat, &l.Lng, &l.Timezone); err != nil {
			return nil, fmt.Errorf("stables: scan: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
