package blankets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/blanketplan"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
)

// RolloverHour is the stable-local hour at which the blanket day changes. The "day" of a
// moment is the stable-local calendar day of the night it belongs to: a night starts in
// the evening of its day and still belongs to that day until 04:00 the next morning (late
// checks after midnight count for the running night). From 04:00 on the app shows the
// coming night: its forecast, recommendation and progress. The forecast of a day is the
// weather snapshot with valid_for = day.
const RolloverHour = 4

// UncoverUntilHour ends the morning in which "uncovered" still belongs to the past night:
// taking the blankets off in the morning finishes that night and must not mark the horse
// as done for the coming one.
const UncoverUntilHour = 12

// NightDay returns the blanket day (YYYY-MM-DD) of now in loc.
func NightDay(now time.Time, loc *time.Location) string {
	t := now.In(loc)
	if t.Hour() < RolloverHour {
		t = t.AddDate(0, 0, -1)
	}
	return t.Format("2006-01-02")
}

// StateDay returns the blanket day a state recorded now belongs to: the NightDay, except
// for "uncovered" in the morning (RolloverHour to UncoverUntilHour), which ends the past night.
func StateDay(now time.Time, loc *time.Location, action string) string {
	t := now.In(loc)
	if action == ActionUncovered && t.Hour() >= RolloverHour && t.Hour() < UncoverUntilHour {
		return t.AddDate(0, 0, -1).Format("2006-01-02")
	}
	return NightDay(now, loc)
}

// nightStart is the instant the blanket day begins (RolloverHour stable-local of the day). It is
// the due_at of reminder claims, so a claim is unique per night.
func nightStart(day string, loc *time.Location) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", day, loc)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(d.Year(), d.Month(), d.Day(), RolloverHour, 0, 0, 0, loc), nil
}

// Blanket is a blanket of a horse.
type Blanket struct {
	ID        string  `json:"id"`
	HorseID   string  `json:"horse_id"`
	Name      string  `json:"name"`
	FillG     int     `json:"fill_g"`
	Color     *string `json:"color"`
	Location  *string `json:"location"`
	PhotoPath *string `json:"photo_path"`
	PhotoURL  *string `json:"photo_url"`
}

// Rule is one blanket rule of a horse as returned by the API. Position is 1-based.
type Rule struct {
	ID        string   `json:"id"`
	Position  int      `json:"position"`
	TempMin   *float64 `json:"temp_min"`
	TempMax   *float64 `json:"temp_max"`
	Rain      *bool    `json:"rain"`
	BlanketID *string  `json:"blanket_id"`
	Note      string   `json:"note"`
}

// Weather is tonight's forecast (the newest snapshot of the day).
type Weather struct {
	NightMinC       float64   `json:"night_min_c"`
	WillRain        bool      `json:"will_rain"`
	RainProbability int       `json:"rain_probability"`
	RainMM          float64   `json:"rain_mm"`
	WindKmh         float64   `json:"wind_kmh"`
	FetchedAt       time.Time `json:"fetched_at"`
}

// Recommendation statuses.
const (
	StatusBlanket   = "blanket"    // a rule matched and names a blanket
	StatusNone      = "none"       // a rule matched and says "no blanket"
	StatusNoRule    = "no_rule"    // a forecast exists but no rule matches
	StatusNoWeather = "no_weather" // no snapshot for the night
)

// Recommendation is the answer of the rules for tonight.
type Recommendation struct {
	Status string `json:"status"`
	// RuleIndex is the 0-based index of the matched rule in the rules list, or null.
	RuleIndex *int `json:"rule_index"`
	// Blanket is set when Status is "blanket".
	Blanket *Blanket `json:"blanket"`
	// Note is the owner's wish of the matched rule ("" when there is none).
	Note string `json:"note"`
}

// Person is a user as embedded in states.
type Person struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// State is one recorded day state of a horse.
type State struct {
	ID              string    `json:"id"`
	HorseID         string    `json:"horse_id"`
	Day             string    `json:"day"`
	Action          string    `json:"action"`
	CoveredWith     *string   `json:"covered_with"`
	CoveredWithName *string   `json:"covered_with_name"`
	ChangedAt       time.Time `json:"changed_at"`
	ChangedBy       *Person   `json:"changed_by"`
}

type horseInfo struct {
	ID         string
	Name       string
	Box        *string
	ColorKey   *string
	HelperNote *string
	OwnerID    *string
}

// horseNight is everything about one horse for the current night.
type horseNight struct {
	Horse    horseInfo
	Rules    []Rule
	Blankets []Blanket
	State    *State
	Rec      Recommendation
}

func (h horseNight) done() bool { return h.State != nil }

// night is the loaded state of a stable for one blanket day.
type night struct {
	StableID     string
	Loc          *time.Location
	ReminderTime string // HH:MM
	Day          string
	Weather      *Weather
	Horses       []horseNight
}

func newBlanket(id, horseID, name string, fillG int, color, location, photo *string) Blanket {
	return Blanket{ID: id, HorseID: horseID, Name: name, FillG: fillG, Color: color, Location: location,
		PhotoPath: photo, PhotoURL: photoURL(photo)}
}

// recommend applies the rules to the forecast. w may be nil (no forecast yet).
func recommend(rules []Rule, blankets []Blanket, w *Weather) Recommendation {
	if w == nil {
		return Recommendation{Status: StatusNoWeather}
	}
	plan := make([]blanketplan.Rule, len(rules))
	for i, r := range rules {
		plan[i] = blanketplan.Rule{Position: r.Position, TempMin: r.TempMin, TempMax: r.TempMax, Rain: r.Rain,
			BlanketID: r.BlanketID, Note: r.Note}
	}
	rule, ok := blanketplan.Recommend(plan, blanketplan.Forecast{NightMinC: w.NightMinC, WillRain: w.WillRain})
	if !ok {
		return Recommendation{Status: StatusNoRule}
	}
	idx := 0
	for i, r := range rules {
		if r.Position == rule.Position {
			idx = i
		}
	}
	rec := Recommendation{Status: StatusNone, RuleIndex: &idx, Note: rule.Note}
	if rule.BlanketID != nil {
		for i := range blankets {
			if blankets[i].ID == *rule.BlanketID {
				b := blankets[i]
				rec.Status, rec.Blanket = StatusBlanket, &b
			}
		}
	}
	return rec
}

// stableInfo reads the time zone and reminder time of a stable.
func (s *Service) stableInfo(ctx context.Context, q querier, stableID string) (*time.Location, string, error) {
	var tz, reminder string
	err := q.QueryRow(ctx, `SELECT timezone, to_char(reminder_time, 'HH24:MI') FROM stables WHERE id = $1`, stableID).Scan(&tz, &reminder)
	if err != nil {
		return nil, "", err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return loc, reminder, nil
}

// loadWeather returns the newest snapshot for the day, or nil when there is none.
func (s *Service) loadWeather(ctx context.Context, stableID, day string) (*Weather, error) {
	d, err := time.Parse("2006-01-02", day)
	if err != nil {
		return nil, err
	}
	snap, err := (&weather.Store{Pool: s.Pool}).Latest(ctx, stableID, d)
	if errors.Is(err, weather.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Weather{NightMinC: snap.NightMinC, WillRain: snap.WillRain, RainProbability: snap.RainProb,
		RainMM: snap.RainMM, WindKmh: snap.WindKmh, FetchedAt: snap.FetchedAt}, nil
}

// loadNight loads the horses of a stable (or only horseID when not empty) with rules,
// blankets, the current state and the recommendation for the blanket day of now.
func (s *Service) loadNight(ctx context.Context, stableID, horseID string) (*night, error) {
	loc, reminder, err := s.stableInfo(ctx, s.Pool, stableID)
	if err != nil {
		return nil, fmt.Errorf("stable: %w", err)
	}
	n := &night{StableID: stableID, Loc: loc, ReminderTime: reminder, Day: NightDay(s.now(), loc)}
	if n.Weather, err = s.loadWeather(ctx, stableID, n.Day); err != nil {
		return nil, fmt.Errorf("weather: %w", err)
	}
	var only *string
	if horseID != "" {
		only = &horseID
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, name, box, color_key, helper_note, owner_id::text
		FROM horses WHERE stable_id = $1 AND ($2::uuid IS NULL OR id = $2::uuid)
		ORDER BY name, id`, stableID, only)
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for rows.Next() {
		var h horseInfo
		if err := rows.Scan(&h.ID, &h.Name, &h.Box, &h.ColorKey, &h.HelperNote, &h.OwnerID); err != nil {
			rows.Close()
			return nil, err
		}
		idx[h.ID] = len(n.Horses)
		n.Horses = append(n.Horses, horseNight{Horse: h})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	blanketRows, err := s.Pool.Query(ctx, `
		SELECT id::text, horse_id::text, name, fill_g, color, location, photo_path
		FROM blankets WHERE stable_id = $1 AND ($2::uuid IS NULL OR horse_id = $2::uuid)
		ORDER BY fill_g, name, id`, stableID, only)
	if err != nil {
		return nil, err
	}
	for blanketRows.Next() {
		var b Blanket
		if err := blanketRows.Scan(&b.ID, &b.HorseID, &b.Name, &b.FillG, &b.Color, &b.Location, &b.PhotoPath); err != nil {
			blanketRows.Close()
			return nil, err
		}
		b.PhotoURL = photoURL(b.PhotoPath)
		if i, ok := idx[b.HorseID]; ok {
			n.Horses[i].Blankets = append(n.Horses[i].Blankets, b)
		}
	}
	blanketRows.Close()
	if err := blanketRows.Err(); err != nil {
		return nil, err
	}

	ruleRows, err := s.Pool.Query(ctx, `
		SELECT id::text, horse_id::text, position, temp_min::float8, temp_max::float8, rain, blanket_id::text, COALESCE(note, '')
		FROM blanket_rules WHERE stable_id = $1 AND ($2::uuid IS NULL OR horse_id = $2::uuid)
		ORDER BY position`, stableID, only)
	if err != nil {
		return nil, err
	}
	for ruleRows.Next() {
		var r Rule
		var hid string
		if err := ruleRows.Scan(&r.ID, &hid, &r.Position, &r.TempMin, &r.TempMax, &r.Rain, &r.BlanketID, &r.Note); err != nil {
			ruleRows.Close()
			return nil, err
		}
		if i, ok := idx[hid]; ok {
			n.Horses[i].Rules = append(n.Horses[i].Rules, r)
		}
	}
	ruleRows.Close()
	if err := ruleRows.Err(); err != nil {
		return nil, err
	}

	stateRows, err := s.Pool.Query(ctx, stateSQL+`
		WHERE s.stable_id = $1 AND s.day = $3::date AND ($2::uuid IS NULL OR s.horse_id = $2::uuid)
		ORDER BY s.horse_id, s.changed_at DESC, s.created_at DESC`, stableID, only, n.Day)
	if err != nil {
		return nil, err
	}
	// The newest row per horse wins (DISTINCT ON is not needed: the first row per horse is kept).
	for stateRows.Next() {
		st, err := scanState(stateRows)
		if err != nil {
			stateRows.Close()
			return nil, err
		}
		if i, ok := idx[st.HorseID]; ok && n.Horses[i].State == nil {
			n.Horses[i].State = &st
		}
	}
	stateRows.Close()
	if err := stateRows.Err(); err != nil {
		return nil, err
	}

	for i := range n.Horses {
		h := &n.Horses[i]
		h.Rec = recommend(h.Rules, h.Blankets, n.Weather)
	}
	return n, nil
}

// stateSQL selects state columns; callers append WHERE and ORDER BY.
const stateSQL = `
	SELECT s.id::text, s.horse_id::text, to_char(s.day, 'YYYY-MM-DD'), s.action, s.covered_with::text, b.name,
	       s.changed_at, s.changed_by::text, u.name
	FROM blanket_states s
	LEFT JOIN blankets b ON b.id = s.covered_with
	LEFT JOIN users u ON u.id = s.changed_by`

func scanState(row pgx.Row) (State, error) {
	var st State
	var by, byName *string
	if err := row.Scan(&st.ID, &st.HorseID, &st.Day, &st.Action, &st.CoveredWith, &st.CoveredWithName,
		&st.ChangedAt, &by, &byName); err != nil {
		return State{}, err
	}
	if by != nil {
		name := ""
		if byName != nil {
			name = *byName
		}
		st.ChangedBy = &Person{ID: *by, Name: name}
	}
	return st, nil
}
