package weather

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/stables"
)

// Service is the snapshot job: for every stable with coordinates it fetches the
// forecast of the nearest station and stores the night summaries of today and
// tomorrow (stable-local days).
type Service struct {
	Pool    *pgxpool.Pool
	Fetcher Fetcher
	Log     *slog.Logger
	Now     func() time.Time
	// StationID forces one station for all stables (REITERHOF_WEATHER_STATION).
	// Empty means: nearest station from Stations.
	StationID string
}

// Refresh runs the job once. A failing stable does not stop the others; the
// errors are joined. Each station is downloaded at most once per run.
func (s *Service) Refresh(ctx context.Context) error {
	located, err := stables.ListLocated(ctx, s.Pool)
	if err != nil {
		return err
	}
	store := &Store{Pool: s.Pool}
	cache := map[string][]Hour{}
	var errs []error
	for _, st := range located {
		if err := s.refreshStable(ctx, store, cache, st); err != nil {
			errs = append(errs, fmt.Errorf("stable %s: %w", st.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (s *Service) refreshStable(ctx context.Context, store *Store, cache map[string][]Hour, st stables.Located) error {
	loc, err := time.LoadLocation(st.Timezone)
	if err != nil {
		return fmt.Errorf("timezone %q: %w", st.Timezone, err)
	}
	station := NearestStation(st.Lat, st.Lng)
	if s.StationID != "" {
		station = StationByID(s.StationID)
	}
	hours, ok := cache[station.ID]
	if !ok {
		hours, err = s.Fetcher.Fetch(ctx, station)
		if err != nil {
			return err
		}
		cache[station.ID] = hours
	}
	now := s.Now()
	var errs []error
	for i := 0; i < 2; i++ {
		day := now.In(loc).AddDate(0, 0, i)
		sum, err := Summarize(hours, loc, day, DefaultWindow)
		if errors.Is(err, ErrNoData) {
			// Tomorrow can lie beyond a short forecast; not an error worth failing for.
			s.log().Debug("no forecast for day", "stable", st.ID, "day", day.Format("2006-01-02"))
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		sum.Forecast = Span(hours, loc, day)
		if err := store.Save(ctx, st.ID, station.ID, now, sum); err != nil {
			errs = append(errs, err)
			continue
		}
		s.log().Info("weather snapshot stored", "stable", st.ID, "station", station.ID, "day", sum.Day,
			"night_min_c", sum.NightMinC, "will_rain", sum.WillRain)
	}
	return errors.Join(errs...)
}

func (s *Service) log() *slog.Logger {
	if s.Log == nil {
		return slog.Default()
	}
	return s.Log
}
