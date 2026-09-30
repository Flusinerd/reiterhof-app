package blankets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/blanketplan"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/scheduler"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
)

const (
	// LastPersonUntil is the stable-local time after which the reminder stops re-checking.
	LastPersonUntil = 22 * time.Hour
	// LastPersonInterval is the re-check interval while several people are present.
	LastPersonInterval = 15 * time.Minute
	// slotTolerance: the job polls every minute; a run counts as a check slot when it
	// falls into the first two minutes of a slot. Claims make double hits harmless.
	slotTolerance = 2 * time.Minute
	// CheckoutFrom is the stable-local time from which leaving the stable as the last
	// person triggers the immediate reminder (earlier check-outs are ordinary days).
	CheckoutFrom = 17 * time.Hour
)

// Claim sources in reminders.source_table (unique index in migration 0070).
const (
	sourceNight    = "blanket_night"
	sourceCheckout = "blanket_checkout"
	sourceWeather  = "blanket_weather"
)

// Jobs returns the background jobs of this package for cmd/api: the last-person
// reminder and the automatic uncovering (JAN-78), both polled every minute (each decides
// itself whether it is due).
func (s *Service) Jobs() []scheduler.Job {
	return []scheduler.Job{
		{Name: "blanket-last-person", Schedule: scheduler.Every(time.Minute), Run: s.RunLastPerson},
		{Name: "blanket-auto-uncover", Schedule: scheduler.Every(time.Minute), Run: s.RunAutoUncover},
	}
}

type stableRow struct {
	id       string
	loc      *time.Location
	reminder string
}

func (s *Service) stables(ctx context.Context) ([]stableRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text, timezone, to_char(reminder_time, 'HH24:MI') FROM stables`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []stableRow
	for rows.Next() {
		var st stableRow
		var tz string
		if err := rows.Scan(&st.id, &tz, &st.reminder); err != nil {
			return nil, err
		}
		if st.loc, err = time.LoadLocation(tz); err != nil {
			st.loc = time.UTC
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// clockOf parses "HH:MM" into a duration since local midnight.
func clockOf(hhmm string) time.Duration {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return 20*time.Hour + 30*time.Minute
	}
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute
}

// lastPersonSlot reports whether now (stable-local) is a check slot of the last-person
// reminder: from the reminder time R every 15 minutes until 22:00 (R itself is always a
// slot, even after 22:00).
func lastPersonSlot(now time.Time, reminder string) bool {
	y, m, d := now.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	start := midnight.Add(clockOf(reminder))
	end := midnight.Add(LastPersonUntil)
	if !end.After(start) {
		end = start.Add(time.Minute)
	}
	if now.Before(start) || !now.Before(end) {
		return false
	}
	return now.Sub(start)%LastPersonInterval < slotTolerance
}

// RunLastPerson is the "Letzte Person" reminder (JAN-33). At the stable's reminder time
// (default 20:30) and then every 15 minutes until 22:00 it looks at the horses without a
// day state tonight:
//
//   - exactly one person is present (open presence visit): push to that person listing
//     the open horses;
//   - nobody is present: push to the owners of the open horses (each their own horses);
//   - several people are present: nothing yet, the next slot checks again.
//
// Nothing is sent when every horse is done. Each user is reminded at most once per
// night (claim in reminders). Presence visibility does not matter here: the count
// includes people who hide from others, and nobody but the recipient learns about it.
// The kind is last_person, so users can opt out.
func (s *Service) RunLastPerson(ctx context.Context) error {
	stables, err := s.stables(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	var errs []error
	for _, st := range stables {
		if !lastPersonSlot(now.In(st.loc), st.reminder) {
			continue
		}
		if err := s.lastPersonStable(ctx, st); err != nil {
			s.log().Error("blankets: last person", "stable", st.id, "err", err)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type openHorse struct {
	id, name string
	owner    *string
}

// openHorses lists the horses without any state for the day, by name.
func (s *Service) openHorses(ctx context.Context, stableID, day string) ([]openHorse, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT h.id::text, h.name, h.owner_id::text FROM horses h
		WHERE h.stable_id = $1
		  AND NOT EXISTS (SELECT 1 FROM blanket_states s WHERE s.horse_id = h.id AND s.day = $2::date)
		ORDER BY h.name, h.id`, stableID, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []openHorse
	for rows.Next() {
		var h openHorse
		if err := rows.Scan(&h.id, &h.name, &h.owner); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Service) present(ctx context.Context, stableID string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT user_id::text FROM presence WHERE stable_id = $1 AND left_at IS NULL`, stableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Service) lastPersonStable(ctx context.Context, st stableRow) error {
	day := NightDay(s.now(), st.loc)
	open, err := s.openHorses(ctx, st.id, day)
	if err != nil || len(open) == 0 {
		return err
	}
	due, err := nightStart(day, st.loc)
	if err != nil {
		return err
	}
	present, err := s.present(ctx, st.id)
	if err != nil {
		return err
	}
	switch len(present) {
	case 1:
		return s.remind(ctx, st.id, sourceNight, st.id, due, push.KindLastPerson,
			"Letzte Person im Stall",
			"Noch ohne Deckenstatus: "+horseNames(open)+".",
			present[0], map[string]any{"screen": "/blankets"})
	case 0:
		byOwner := map[string][]openHorse{}
		var owners []string
		for _, h := range open {
			if h.owner == nil {
				continue
			}
			if _, seen := byOwner[*h.owner]; !seen {
				owners = append(owners, *h.owner)
			}
			byOwner[*h.owner] = append(byOwner[*h.owner], h)
		}
		var errs []error
		for _, owner := range owners {
			hs := byOwner[owner]
			errs = append(errs, s.remind(ctx, st.id, sourceNight, st.id, due, push.KindLastPerson,
				"Stall leer",
				horseNames(hs)+": noch ohne Deckenstatus.",
				owner, map[string]any{"screen": "/blankets"}))
		}
		return errors.Join(errs...)
	}
	return nil
}

// OnCheckOut is the immediate variant of the last-person reminder: called (through
// presence.AfterCheckOut) after a user checked out. If nobody else is present, it is
// evening (from CheckoutFrom, 17:00 stable-local) and horses are still open tonight, the
// user gets a push. One per user and night.
func (s *Service) OnCheckOut(ctx context.Context, stableID, userID string) {
	if err := s.onCheckOut(ctx, stableID, userID); err != nil {
		s.log().Error("blankets: check-out reminder", "stable", stableID, "err", err)
	}
}

func (s *Service) onCheckOut(ctx context.Context, stableID, userID string) error {
	loc, _, err := s.stableInfo(ctx, s.Pool, stableID)
	if err != nil {
		return err
	}
	now := s.now().In(loc)
	y, m, d := now.Date()
	if now.Sub(time.Date(y, m, d, 0, 0, 0, 0, loc)) < CheckoutFrom {
		return nil
	}
	present, err := s.present(ctx, stableID)
	if err != nil || len(present) > 0 {
		return err
	}
	day := NightDay(now, loc)
	open, err := s.openHorses(ctx, stableID, day)
	if err != nil || len(open) == 0 {
		return err
	}
	due, err := nightStart(day, loc)
	if err != nil {
		return err
	}
	return s.remind(ctx, stableID, sourceCheckout, stableID, due, push.KindLastPerson,
		"Letzte Person im Stall",
		"Noch ohne Deckenstatus: "+horseNames(open)+".",
		userID, map[string]any{"screen": "/blankets"})
}

// horseNames joins up to five names ("Luna, Balu und 2 weitere").
func horseNames(hs []openHorse) string {
	const shown = 5
	names := make([]string, 0, shown)
	for i, h := range hs {
		if i == shown {
			break
		}
		names = append(names, h.name)
	}
	out := strings.Join(names, ", ")
	if len(hs) > shown {
		out += fmt.Sprintf(" und %d weitere", len(hs)-shown)
	}
	return out
}

// remind claims (user, kind, source, night) and pushes if the claim is new. A failed
// push releases the claim so the next run retries.
func (s *Service) remind(ctx context.Context, stableID, source, sourceID string, due time.Time, kind, title, body, userID string, data map[string]any) error {
	tag, err := s.Pool.Exec(ctx, `
		INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, kind, source_table, source_id, due_at)
		WHERE source_table IN ('blanket_night', 'blanket_checkout', 'blanket_weather') DO NOTHING`,
		stableID, userID, kind, title, body, due, source, sourceID, s.now())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	if err := s.Notify.NotifyUsers(ctx, stableID, []string{userID}, kind, title, body, data); err != nil {
		if _, derr := s.Pool.Exec(context.WithoutCancel(ctx), `
			DELETE FROM reminders WHERE user_id = $1 AND kind = $2 AND source_table = $3 AND source_id = $4 AND due_at = $5`,
			userID, kind, source, sourceID, due); derr != nil {
			s.log().Error("blankets: release reminder claim", "err", derr)
		}
		return err
	}
	return nil
}

// --------------------------------------------------------- weather change

// CheckWeatherChange is the weather change push (JAN-34). Call it after every weather
// refresh. For each horse that already has a day state tonight it compares the
// recommendation of the newest snapshot of the night with the recommendation of the
// snapshot that was current when the state was set (the forecast the decision was based
// on). If the blanket differs (blanketplan.Changed) the owner and the riders get a
// weather_change push, at most once per horse and night (claim in reminders). Horses
// without a state, or states set before any forecast existed, are ignored.
func (s *Service) CheckWeatherChange(ctx context.Context) error {
	stables, err := s.stables(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, st := range stables {
		if err := s.weatherChangeStable(ctx, st); err != nil {
			s.log().Error("blankets: weather change", "stable", st.id, "err", err)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Service) weatherChangeStable(ctx context.Context, st stableRow) error {
	day := NightDay(s.now(), st.loc)
	d, err := time.ParseInLocation("2006-01-02", day, st.loc)
	if err != nil {
		return err
	}
	snaps, err := (&weather.Store{Pool: s.Pool}).ForDay(ctx, st.id, d)
	if err != nil {
		return err
	}
	if len(snaps) < 2 {
		return nil
	}
	latest := snaps[len(snaps)-1]

	n, err := s.loadNight(ctx, st.id, "")
	if err != nil {
		return err
	}
	due, err := nightStart(day, st.loc)
	if err != nil {
		return err
	}
	var errs []error
	for _, hn := range n.Horses {
		if hn.State == nil {
			continue
		}
		// The forecast the state was based on: the newest snapshot fetched before it. Both
		// forecasts are summarised over the cover window of the horse.
		var base *weather.Snapshot
		for i := range snaps {
			if !snaps[i].FetchedAt.After(hn.State.ChangedAt) {
				base = &snaps[i]
			}
		}
		if base == nil || base.FetchedAt.Equal(latest.FetchedAt) {
			continue
		}
		win := windowOf(hn.Horse.CoverStart, hn.Horse.CoverEnd)
		prev := ruleFor(hn.Rules, *weatherOf(*base, st.loc, day, win))
		next := ruleFor(hn.Rules, *weatherOf(latest, st.loc, day, win))
		if !blanketplan.Changed(prev, next) {
			continue
		}
		recipients, err := s.horseRecipients(ctx, st.id, hn.Horse.ID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		title := "Wetter geändert"
		body := fmt.Sprintf("%s: jetzt %s statt %s.", hn.Horse.Name, describe(next, hn.Blankets), describe(prev, hn.Blankets))
		data := map[string]any{"horse_id": hn.Horse.ID, "screen": "/horses/" + hn.Horse.ID + "/blanket-plan"}
		for _, uid := range recipients {
			if err := s.remind(ctx, st.id, sourceWeather, hn.Horse.ID, due, push.KindWeatherChange, title, body, uid, data); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// ruleFor evaluates the rules; no match counts as "no blanket".
func ruleFor(rules []Rule, w Weather) blanketplan.Rule {
	plan := make([]blanketplan.Rule, len(rules))
	for i, r := range rules {
		plan[i] = blanketplan.Rule{Position: r.Position, TempMin: r.TempMin, TempMax: r.TempMax, Rain: r.Rain,
			BlanketID: r.BlanketID, Note: r.Note}
	}
	rule, _ := blanketplan.Recommend(plan, blanketplan.Forecast{NightMinC: w.NightMinC, WillRain: w.WillRain})
	return rule
}

// describe names the blanket of a rule ("jetzt Decke 100 g" / "keine Decke").
func describe(rule blanketplan.Rule, blankets []Blanket) string {
	if rule.BlanketID != nil {
		for _, b := range blankets {
			if b.ID == *rule.BlanketID {
				return b.Name
			}
		}
	}
	return "keine Decke"
}

// horseRecipients are the owner and the riders of the horse.
func (s *Service) horseRecipients(ctx context.Context, stableID, horseID string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT owner_id::text FROM horses WHERE id = $1 AND stable_id = $2 AND owner_id IS NOT NULL
		UNION
		SELECT user_id::text FROM horse_riders WHERE horse_id = $1 AND stable_id = $2
		ORDER BY 1`, horseID, stableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
