package admincli

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/blankets"
)

// Defaults of the stables table (migration 0001), used to echo the created stable.
const (
	defaultTimezone     = "Europe/Berlin"
	defaultReminderTime = "20:30"
	defaultRadiusM      = 150
)

var hhmmRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

type stableRow struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Farm            *string  `json:"farm"`
	City            *string  `json:"city"`
	Lat             *float64 `json:"lat"`
	Lng             *float64 `json:"lng"`
	Timezone        string   `json:"timezone"`
	ReminderTime    string   `json:"reminder_time"`
	GeofenceRadiusM int      `json:"geofence_radius_m"`
	// AutoUncoverTime is "HH:MM" or null (automatic uncovering off); AutoUncoverDays are ISO
	// weekdays, 1 = Monday .. 7 = Sunday.
	AutoUncoverTime *string   `json:"auto_uncover_time"`
	AutoUncoverDays []int16   `json:"auto_uncover_days"`
	Users           int       `json:"users"`
	Horses          int       `json:"horses"`
	CreatedAt       time.Time `json:"created_at"`
}

func runStableList(ctx context.Context, e *Env, args []string) error {
	if _, err := parse(e.flags("stable list"), args, 0, 0); err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	rows, err := pool.Query(ctx, `SELECT s.id::text, s.name, s.farm_name, s.city, s.lat, s.lng, s.timezone,
			to_char(s.reminder_time, 'HH24:MI'), s.geofence_radius_m,
			to_char(s.auto_uncover_time, 'HH24:MI'), s.auto_uncover_days,
			(SELECT count(*) FROM users u WHERE u.stable_id = s.id AND u.deleted_at IS NULL),
			(SELECT count(*) FROM horses h WHERE h.stable_id = s.id),
			s.created_at
		FROM stables s ORDER BY s.created_at, s.id`)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[stableRow])
	if err != nil {
		return err
	}
	if e.JSON {
		if list == nil {
			list = []stableRow{}
		}
		return e.writeJSON(list)
	}
	table := make([][]string, 0, len(list))
	for _, s := range list {
		loc := "-"
		if s.Lat != nil && s.Lng != nil {
			loc = fmt.Sprintf("%.4f,%.4f", *s.Lat, *s.Lng)
		}
		table = append(table, []string{s.ID, s.Name, orDash(deref(s.Farm)), orDash(deref(s.City)), loc,
			s.Timezone, s.ReminderTime, strconv.Itoa(s.GeofenceRadiusM), autoUncoverText(s.AutoUncoverTime, s.AutoUncoverDays), strconv.Itoa(s.Users), strconv.Itoa(s.Horses), fmtTime(s.CreatedAt)})
	}
	writeTable(e.Out, []string{"ID", "NAME", "FARM", "CITY", "LOCATION", "TIMEZONE", "REMINDER", "RADIUS_M", "AUTO_UNCOVER", "USERS", "HORSES", "CREATED (UTC)"}, table)
	if len(list) == 0 {
		fmt.Fprintln(e.Err, "no stables yet; create one with: stallfunk-admin stable create --name ...")
	}
	return nil
}

// dayNames are the ISO weekdays 1..7 as accepted and printed by --auto-uncover-days.
var dayNames = [...]string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

// dayAliases maps every accepted spelling to its ISO weekday.
var dayAliases = map[string]int16{
	"mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6, "sun": 7,
	"mo": 1, "di": 2, "mi": 3, "do": 4, "fr": 5, "sa": 6, "so": 7,
}

// parseAutoUncoverTime validates --auto-uncover: "off" clears the setting (nil), otherwise
// HH:MM within [RolloverHour:00, UncoverUntilHour:00): the state would fall outside the
// past night otherwise (see blankets.StateDay).
func parseAutoUncoverTime(v string) (*string, error) {
	if v == "off" {
		return nil, nil
	}
	if !hhmmRe.MatchString(v) {
		return nil, usagef("--auto-uncover must be HH:MM (24 hours) or off, got %q", v)
	}
	if h := int(v[0]-'0')*10 + int(v[1]-'0'); h < blankets.RolloverHour || h >= blankets.UncoverUntilHour {
		return nil, usagef("--auto-uncover must be at least %02d:00 and before %02d:00, got %s",
			blankets.RolloverHour, blankets.UncoverUntilHour, v)
	}
	return &v, nil
}

// parseDays parses --auto-uncover-days: "all", "mon-fri", or a comma list of weekdays and
// ranges ("mon,wed,sat", "mon-wed,sat"; English or German abbreviations). The result is
// sorted and free of duplicates.
func parseDays(v string) ([]int16, error) {
	have := map[int16]bool{}
	if strings.ToLower(strings.TrimSpace(v)) == "all" {
		return []int16{1, 2, 3, 4, 5, 6, 7}, nil
	}
	for _, part := range strings.Split(v, ",") {
		from, to, isRange := strings.Cut(strings.ToLower(strings.TrimSpace(part)), "-")
		a, ok := dayAliases[strings.TrimSpace(from)]
		b := a
		if isRange {
			b, ok = dayAliases[strings.TrimSpace(to)], ok && dayAliases[strings.TrimSpace(to)] != 0
		}
		if !ok || b < a {
			return nil, usagef("--auto-uncover-days must be all, mon-fri or a list like mon,tue,sat, got %q", v)
		}
		for d := a; d <= b; d++ {
			have[d] = true
		}
	}
	var out []int16
	for d := int16(1); d <= 7; d++ {
		if have[d] {
			out = append(out, d)
		}
	}
	return out, nil
}

// daysText prints ISO weekdays compactly: "all", "mon-fri", or "mon,wed,sat".
func daysText(days []int16) string {
	switch {
	case len(days) == 7:
		return "all"
	case slices.Equal(days, []int16{1, 2, 3, 4, 5}):
		return "mon-fri"
	}
	names := make([]string, 0, len(days))
	for _, d := range days {
		if d >= 1 && d <= 7 {
			names = append(names, dayNames[d-1])
		}
	}
	return strings.Join(names, ",")
}

func autoUncoverText(clock *string, days []int16) string {
	if clock == nil {
		return "off"
	}
	return *clock + " " + daysText(days)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// stableFlags are the flags shared by stable create and update.
type stableFlags struct {
	name, farm, city, timezone, reminderTime string
	lat, lng                                 float64
	radius                                   int
	autoUncover, autoUncoverDays             string
}

func (f *stableFlags) register(e *Env, name string) func(args []string) ([]string, map[string]bool, error) {
	fs := e.flags(name)
	fs.StringVar(&f.name, "name", "", "stable name")
	fs.StringVar(&f.farm, "farm", "", "farm name")
	fs.StringVar(&f.city, "city", "", "city")
	fs.Float64Var(&f.lat, "lat", 0, "latitude (needed for the weather)")
	fs.Float64Var(&f.lng, "lng", 0, "longitude (needed for the weather)")
	fs.StringVar(&f.timezone, "timezone", "", "IANA time zone (default "+defaultTimezone+")")
	fs.StringVar(&f.reminderTime, "reminder-time", "", "time of the last-person blanket reminder, HH:MM (default "+defaultReminderTime+")")
	fs.StringVar(&f.autoUncover, "auto-uncover", "", "time the blankets are taken off automatically, HH:MM from "+
		fmt.Sprintf("%02d:00 to before %02d:00", blankets.RolloverHour, blankets.UncoverUntilHour)+", or off (default off)")
	fs.StringVar(&f.autoUncoverDays, "auto-uncover-days", "", "days of the automatic uncovering: mon-fri (default), all, or a list like mon,wed,sat")
	fs.IntVar(&f.radius, "geofence-radius", 0, "geofence radius in metres (default "+strconv.Itoa(defaultRadiusM)+")")
	return func(args []string) ([]string, map[string]bool, error) {
		pos, err := parse(fs, args, 0, 1)
		return pos, setFlags(fs), err
	}
}

// validate checks the values of the flags that were given.
func (f *stableFlags) validate(set map[string]bool) error {
	if set["name"] && strings.TrimSpace(f.name) == "" {
		return usagef("--name must not be empty")
	}
	if set["lat"] && (f.lat < -90 || f.lat > 90) {
		return usagef("--lat must be between -90 and 90")
	}
	if set["lng"] && (f.lng < -180 || f.lng > 180) {
		return usagef("--lng must be between -180 and 180")
	}
	if set["timezone"] {
		if _, err := time.LoadLocation(f.timezone); err != nil || f.timezone == "" || f.timezone == "Local" {
			return usagef("--timezone %q is not an IANA time zone (e.g. Europe/Berlin)", f.timezone)
		}
	}
	if set["reminder-time"] && !hhmmRe.MatchString(f.reminderTime) {
		return usagef("--reminder-time must be HH:MM (24 hours), got %q", f.reminderTime)
	}
	if set["geofence-radius"] && f.radius <= 0 {
		return usagef("--geofence-radius must be a positive number of metres")
	}
	if set["auto-uncover"] {
		if _, err := parseAutoUncoverTime(f.autoUncover); err != nil {
			return err
		}
	}
	if set["auto-uncover-days"] {
		if _, err := parseDays(f.autoUncoverDays); err != nil {
			return err
		}
	}
	return nil
}

func runStableCreate(ctx context.Context, e *Env, args []string) error {
	var f stableFlags
	parseFn := f.register(e, "stable create")
	pos, set, err := parseFn(args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return usagef("stable create takes flags only (got %q)", pos)
	}
	if !set["name"] {
		return usagef("--name is required")
	}
	if set["lat"] != set["lng"] {
		return usagef("--lat and --lng must be given together")
	}
	if err := f.validate(set); err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	var lat, lng *float64
	if set["lat"] {
		lat, lng = &f.lat, &f.lng
	}
	tz, reminder, radius := defaultTimezone, defaultReminderTime, defaultRadiusM
	if set["timezone"] {
		tz = f.timezone
	}
	if set["reminder-time"] {
		reminder = f.reminderTime
	}
	if set["geofence-radius"] {
		radius = f.radius
	}
	var autoTime *string
	if set["auto-uncover"] {
		autoTime, _ = parseAutoUncoverTime(f.autoUncover) // validated above
	}
	autoDays := []int16{1, 2, 3, 4, 5}
	if set["auto-uncover-days"] {
		autoDays, _ = parseDays(f.autoUncoverDays)
	}
	var id string
	err = pool.QueryRow(ctx, `INSERT INTO stables (name, farm_name, city, lat, lng, timezone, reminder_time, geofence_radius_m,
			auto_uncover_time, auto_uncover_days)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, $5, $6, $7::time, $8, $9::time, $10::smallint[]) RETURNING id::text`,
		strings.TrimSpace(f.name), strings.TrimSpace(f.farm), strings.TrimSpace(f.city), lat, lng, tz, reminder, radius,
		autoTime, autoDays).Scan(&id)
	if err != nil {
		return fmt.Errorf("create stable: %w", err)
	}
	if e.JSON {
		return e.writeJSON(map[string]any{"id": id, "name": strings.TrimSpace(f.name)})
	}
	fmt.Fprintln(e.Out, id)
	fmt.Fprintf(e.Err, "created stable %q\n", strings.TrimSpace(f.name))
	if lat == nil {
		fmt.Fprintln(e.Err, "note: no location set, so the weather is not fetched for this stable (stable update ID --lat N --lng N)")
	}
	return nil
}

func runStableUpdate(ctx context.Context, e *Env, args []string) error {
	var f stableFlags
	parseFn := f.register(e, "stable update")
	pos, set, err := parseFn(args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usagef("stable update needs the stable id (see: stallfunk-admin stable list)")
	}
	if err := f.validate(set); err != nil {
		return err
	}
	var sets []string
	var params []any
	add := func(expr string, v any) {
		params = append(params, v)
		sets = append(sets, fmt.Sprintf(expr, len(params)+1)) // $1 is the id
	}
	if set["name"] {
		add("name = $%d", strings.TrimSpace(f.name))
	}
	if set["farm"] {
		add("farm_name = NULLIF($%d, '')", strings.TrimSpace(f.farm))
	}
	if set["city"] {
		add("city = NULLIF($%d, '')", strings.TrimSpace(f.city))
	}
	if set["lat"] {
		add("lat = $%d", f.lat)
	}
	if set["lng"] {
		add("lng = $%d", f.lng)
	}
	if set["timezone"] {
		add("timezone = $%d", f.timezone)
	}
	if set["reminder-time"] {
		add("reminder_time = $%d::time", f.reminderTime)
	}
	if set["geofence-radius"] {
		add("geofence_radius_m = $%d", f.radius)
	}
	if set["auto-uncover"] {
		t, _ := parseAutoUncoverTime(f.autoUncover) // validated above
		add("auto_uncover_time = $%d::time", t)
	}
	if set["auto-uncover-days"] {
		d, _ := parseDays(f.autoUncoverDays)
		add("auto_uncover_days = $%d::smallint[]", d)
	}
	if len(sets) == 0 {
		return usagef("nothing to update: give at least one of --name, --farm, --city, --lat, --lng, --timezone, --reminder-time, --geofence-radius, --auto-uncover, --auto-uncover-days")
	}
	st, err := e.resolveStable(ctx, pos[0])
	if err != nil {
		return err
	}
	pool, err := e.pool(ctx)
	if err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, "UPDATE stables SET "+strings.Join(sets, ", ")+" WHERE id = $1", append([]any{st.ID}, params...)...); err != nil {
		return fmt.Errorf("update stable: %w", err)
	}
	fmt.Fprintf(e.Out, "updated stable %s (%s)\n", st.ID, st.Name)
	return nil
}
