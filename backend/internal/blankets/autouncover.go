package blankets

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/requests"
)

// autoUncoverFeedback is stored as payload.feedback of blanket requests that the job closes.
const autoUncoverFeedback = "Automatisch abgedeckt (Hof)"

type autoUncoverStable struct {
	id    string
	loc   *time.Location
	clock string // HH:MM
	days  []int16
}

// autoUncoverDay reports the blanket day to uncover at now (stable-local via loc): the
// previous calendar date, while the weekday is one of days (ISO, 1 = Monday) and the local
// time is in [clock, UncoverUntilHour). ok is false when nothing is due. The day equals
// StateDay(now, loc, ActionUncovered).
func autoUncoverDay(now time.Time, loc *time.Location, clock string, days []int16) (day string, ok bool) {
	t := now.In(loc)
	wd := int16(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	if !slices.Contains(days, wd) || t.Hour() >= UncoverUntilHour {
		return "", false
	}
	// Wall clock, not elapsed time since midnight: stays right on DST change days.
	wall := time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute
	if wall < clockOf(clock) {
		return "", false
	}
	return t.AddDate(0, 0, -1).Format("2006-01-02"), true
}

// RunAutoUncover is the job blanket-auto-uncover (JAN-78), polled every minute. The farm
// staff take the blankets off on the configured weekdays; the app is not used for that.
// For each stable with auto_uncover_time set, from that time until UncoverUntilHour on one
// of auto_uncover_days, every horse whose newest state of the past night is "covered" gets
// an automatic "uncovered" state (changed_by NULL, automatic = true). It is idempotent:
// the newest state is re-checked under the advisory lock of setState, so a later run, a
// concurrent tap or a manual "uncovered" leaves nothing to do.
func (s *Service) RunAutoUncover(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, timezone, to_char(auto_uncover_time, 'HH24:MI'), auto_uncover_days
		FROM stables WHERE auto_uncover_time IS NOT NULL`)
	if err != nil {
		return err
	}
	var stables []autoUncoverStable
	for rows.Next() {
		var st autoUncoverStable
		var tz string
		if err := rows.Scan(&st.id, &tz, &st.clock, &st.days); err != nil {
			rows.Close()
			return err
		}
		if st.loc, err = time.LoadLocation(tz); err != nil {
			st.loc = time.UTC
		}
		stables = append(stables, st)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	now := s.now()
	var errs []error
	for _, st := range stables {
		day, ok := autoUncoverDay(now, st.loc, st.clock, st.days)
		if !ok {
			continue
		}
		n, err := s.autoUncoverStable(ctx, st.id, day, now)
		if err != nil {
			s.log().Error("blankets: auto uncover", "stable", st.id, "err", err)
			errs = append(errs, err)
		}
		if n > 0 {
			s.log().Info("blankets: auto uncovered", "stable", st.id, "day", day, "horses", n)
		}
	}
	return errors.Join(errs...)
}

// autoUncoverStable uncovers the covered horses of the stable for day and returns how many
// it did. A failing horse does not stop the others.
func (s *Service) autoUncoverStable(ctx context.Context, stableID, day string, now time.Time) (int, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT h.id::text FROM horses h
		WHERE h.stable_id = $1 AND (
			SELECT s.action FROM blanket_states s WHERE s.horse_id = h.id AND s.day = $2::date
			ORDER BY s.changed_at DESC, s.created_at DESC LIMIT 1) = 'covered'
		ORDER BY h.name, h.id`, stableID, day)
	if err != nil {
		return 0, err
	}
	var horseIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		horseIDs = append(horseIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	done := 0
	var errs []error
	for _, horseID := range horseIDs {
		did, err := s.autoUncoverHorse(ctx, stableID, horseID, day, now)
		if err != nil {
			s.log().Error("blankets: auto uncover horse", "stable", stableID, "horse", horseID, "err", err)
			errs = append(errs, err)
			continue
		}
		if did {
			done++
		}
	}
	return done, errors.Join(errs...)
}

// autoUncoverHorse records the automatic "uncovered" of one horse if its newest state of
// day is still "covered", and closes the open blanket requests like setState does.
func (s *Service) autoUncoverHorse(ctx context.Context, stableID, horseID, day string, now time.Time) (bool, error) {
	did := false
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, horseID+day); err != nil {
			return err
		}
		latest, err := scanState(tx.QueryRow(ctx, stateSQL+`
			WHERE s.stable_id = $1 AND s.horse_id = $2 AND s.day = $3::date
			ORDER BY s.changed_at DESC, s.created_at DESC LIMIT 1`, stableID, horseID, day))
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && latest.Action != ActionCovered) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := insertState(ctx, tx, stableID, horseID, day, ActionUncovered, nil, now, nil, true); err != nil {
			return err
		}
		closed, err := requests.CompleteBlanketRequests(ctx, tx, stableID, horseID, day, autoUncoverFeedback)
		if err != nil {
			return err
		}
		for _, id := range closed {
			if err := realtime.Publish(ctx, tx, stableID, "request.changed", map[string]any{"id": id, "kind": "done"}); err != nil {
				return err
			}
		}
		if err := realtime.Publish(ctx, tx, stableID, EventStateChanged, map[string]any{"horse_id": horseID, "day": day}); err != nil {
			return err
		}
		did = true
		return nil
	})
	return did && err == nil, err
}
