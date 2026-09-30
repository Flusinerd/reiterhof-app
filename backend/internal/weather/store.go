package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned by Latest when there is no snapshot for the day.
var ErrNotFound = errors.New("weather: no snapshot")

// Snapshot is a stored weather_snapshots row.
type Snapshot struct {
	StableID   string
	FetchedAt  time.Time
	ValidFor   string // YYYY-MM-DD, the stable-local day the night starts on
	NightMinC  float64
	RainProb   int
	RainMM     float64
	WindKmh    float64
	WillRain   bool
	StationID  string
	RawSummary Summary
}

// Store reads and writes weather_snapshots.
type Store struct {
	Pool *pgxpool.Pool
}

type raw struct {
	Station string  `json:"station"`
	Summary Summary `json:"summary"`
}

// Save inserts a snapshot. Snapshots are append-only; Latest picks the newest.
func (s *Store) Save(ctx context.Context, stableID, stationID string, fetchedAt time.Time, sum Summary) error {
	rawJSON, err := json.Marshal(raw{Station: stationID, Summary: sum})
	if err != nil {
		return fmt.Errorf("weather: marshal raw: %w", err)
	}
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO weather_snapshots
			(stable_id, fetched_at, valid_for, night_min_c, rain_probability, rain_mm, wind_kmh, will_rain, raw)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, $9)`,
		stableID, fetchedAt, sum.Day, sum.NightMinC, sum.RainProbability, sum.RainMM, sum.WindKmh, sum.WillRain, rawJSON)
	if err != nil {
		return fmt.Errorf("weather: save snapshot: %w", err)
	}
	return nil
}

// Latest returns the newest snapshot for the stable-local day (only the
// calendar date of day in its own location is used) or ErrNotFound.
func (s *Store) Latest(ctx context.Context, stableID string, day time.Time) (Snapshot, error) {
	var (
		snap    Snapshot
		rawJSON []byte
		valid   time.Time
	)
	err := s.Pool.QueryRow(ctx, `
		SELECT stable_id::text, fetched_at, valid_for,
		       night_min_c::float8, rain_probability, rain_mm::float8, wind_kmh::float8, will_rain, raw
		FROM weather_snapshots
		WHERE stable_id = $1 AND valid_for = $2::date
		ORDER BY fetched_at DESC, created_at DESC
		LIMIT 1`, stableID, day.Format("2006-01-02")).
		Scan(&snap.StableID, &snap.FetchedAt, &valid, &snap.NightMinC, &snap.RainProb, &snap.RainMM, &snap.WindKmh, &snap.WillRain, &rawJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return Snapshot{}, ErrNotFound
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("weather: latest snapshot: %w", err)
	}
	snap.ValidFor = valid.Format("2006-01-02")
	var r raw
	if err := json.Unmarshal(rawJSON, &r); err == nil {
		snap.StationID, snap.RawSummary = r.Station, r.Summary
	}
	return snap, nil
}

// ForDay returns all snapshots of the stable-local day, oldest first.
func (s *Store) ForDay(ctx context.Context, stableID string, day time.Time) ([]Snapshot, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT stable_id::text, fetched_at, valid_for,
		       night_min_c::float8, rain_probability, rain_mm::float8, wind_kmh::float8, will_rain, raw
		FROM weather_snapshots
		WHERE stable_id = $1 AND valid_for = $2::date AND night_min_c IS NOT NULL
		ORDER BY fetched_at, created_at`, stableID, day.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("weather: snapshots of the day: %w", err)
	}
	defer rows.Close()
	var out []Snapshot
	for rows.Next() {
		var (
			snap    Snapshot
			rawJSON []byte
			valid   time.Time
		)
		if err := rows.Scan(&snap.StableID, &snap.FetchedAt, &valid, &snap.NightMinC, &snap.RainProb, &snap.RainMM, &snap.WindKmh, &snap.WillRain, &rawJSON); err != nil {
			return nil, fmt.Errorf("weather: scan snapshot: %w", err)
		}
		snap.ValidFor = valid.Format("2006-01-02")
		var r raw
		if err := json.Unmarshal(rawJSON, &r); err == nil {
			snap.StationID, snap.RawSummary = r.Station, r.Summary
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}
