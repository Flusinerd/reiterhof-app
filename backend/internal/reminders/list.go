package reminders

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

// Ranges of GET /reminders.
const (
	RangeToday = "today"
	RangeWeek  = "week"
)

// maxItems bounds the response (overdue items could pile up).
const maxItems = 100

// Item is one entry of the reminder center: either a row of the reminders table (a push
// that was sent or is scheduled) or a computed upcoming item that is not a row yet.
type Item struct {
	// ID is the reminders row id for stored items; computed items have ids that start with "c:".
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Body  string `json:"body"`
	// DueAt is when the item is or was due. All-day items (AllDay) carry local midnight.
	DueAt  time.Time  `json:"due_at"`
	AllDay bool       `json:"all_day"`
	SentAt *time.Time `json:"sent_at"`
	// Screen is the app route to open ("" = none), the same convention as push data.screen.
	Screen string `json:"screen"`
	// Computed items are derived from other tables at request time.
	Computed bool `json:"computed"`
	// Dismissible items can be removed with POST /reminders/{id}/dismiss (stored items only).
	Dismissible bool `json:"dismissible"`
}

// Group is a titled list of items.
type Group struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Items []Item `json:"items"`
}

// BlanketCheck is tonight's blanket check for the hero of the screen.
type BlanketCheck struct {
	// Day is tonight's blanket day (the stable-local calendar day).
	Day string `json:"day"`
	// Time is the stable's reminder time (HH:MM) and DueAt that time today.
	Time  string    `json:"time"`
	DueAt time.Time `json:"due_at"`
	Done  int       `json:"done"`
	Total int       `json:"total"`
	// State is "empty" (no horses), "done" (every horse has a state), "due" (reminder
	// time reached, horses open) or "upcoming".
	State  string `json:"state"`
	Screen string `json:"screen"`
}

// Response is the body of GET /reminders.
type Response struct {
	Range        string       `json:"range"`
	Today        string       `json:"today"`
	Timezone     string       `json:"timezone"`
	BlanketCheck BlanketCheck `json:"blanket_check"`
	Groups       []Group      `json:"groups"`
}

const (
	screenBlankets = "/blankets"
	screenWeek     = "/training/week"
)

func horseScreen(horseID, sub string) string {
	if horseID == "" {
		return ""
	}
	return "/horses/" + horseID + "/" + sub
}

// list handles GET /reminders?range=today|week (default week).
func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	rng := r.URL.Query().Get("range")
	if rng == "" {
		rng = RangeWeek
	}
	if rng != RangeToday && rng != RangeWeek {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "range must be today or week")
		return
	}
	resp, err := build(r.Context(), h.deps.Pool, user(r), h.now(), rng)
	if err != nil {
		h.internal(w, "list", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// dismiss handles POST /reminders/{id}/dismiss: the reminder disappears from the list. Only
// the user's own stored reminders can be dismissed (404 otherwise); repeating it is fine.
func (h *handler) dismiss(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	id := r.PathValue("id")
	if !uuidRe.MatchString(id) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "reminder not found")
		return
	}
	tag, err := h.deps.Pool.Exec(r.Context(), `
		UPDATE reminders SET dismissed_at = COALESCE(dismissed_at, $4)
		WHERE id = $1 AND user_id = $2 AND stable_id = $3`, id, u.ID, u.StableID, h.now())
	if err != nil {
		h.internal(w, "dismiss", err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "reminder not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// window is the time frame of a request in the stable's time zone.
type window struct {
	loc      *time.Location
	now      time.Time
	today    time.Time // local midnight of today
	tomorrow time.Time // local midnight of tomorrow
	end      time.Time // exclusive end of the window (local midnight)
}

func newWindow(now time.Time, loc *time.Location, rng string) window {
	n := now.In(loc)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
	w := window{loc: loc, now: n, today: today, tomorrow: today.AddDate(0, 0, 1)}
	if rng == RangeToday {
		w.end = w.tomorrow
	} else {
		w.end = today.AddDate(0, 0, 7)
	}
	return w
}

// day returns the local midnight of t.
func (w window) day(t time.Time) time.Time {
	l := t.In(w.loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, w.loc)
}

// at returns the local wall-clock time hh:mm on the local day of d.
func (w window) at(d time.Time, hh, mm int) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), hh, mm, 0, 0, w.loc)
}

// dateOnly is a date column value as a time at UTC midnight; this turns it into the local midnight.
func (w window) localDate(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, w.loc)
}

// pgDate formats a local midnight as a date argument.
func pgDate(t time.Time) string { return t.Format("2006-01-02") }

type builder struct {
	pool *pgxpool.Pool
	u    auth.User
	w    window
	// keys of stored items in the window; computed items with the same key are dropped so
	// that a push that was already sent is not listed twice.
	stored map[string]bool
	items  []Item
}

func build(ctx context.Context, pool *pgxpool.Pool, u auth.User, now time.Time, rng string) (Response, error) {
	var tz, reminder string
	if err := pool.QueryRow(ctx, `SELECT timezone, to_char(reminder_time, 'HH24:MI') FROM stables WHERE id = $1`,
		u.StableID).Scan(&tz, &reminder); err != nil {
		return Response{}, fmt.Errorf("stable: %w", err)
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	b := &builder{pool: pool, u: u, w: newWindow(now, loc, rng), stored: map[string]bool{}}

	if err := b.storedItems(ctx); err != nil {
		return Response{}, fmt.Errorf("stored reminders: %w", err)
	}
	for _, part := range []struct {
		name string
		fn   func(context.Context) error
	}{
		{"health", b.healthItems}, {"reha", b.rehaItems}, {"requests", b.requestItems}, {"training", b.trainingItems},
	} {
		if err := part.fn(ctx); err != nil {
			return Response{}, fmt.Errorf("%s items: %w", part.name, err)
		}
	}
	check, err := b.blanketCheck(ctx, reminder)
	if err != nil {
		return Response{}, fmt.Errorf("blanket check: %w", err)
	}

	resp := Response{Range: rng, Today: pgDate(b.w.today), Timezone: tz, BlanketCheck: check}
	resp.Groups = b.groups(rng)
	return resp, nil
}

// sortKey is the time an item is placed at: when it was sent, else when it is due.
func sortKey(it Item) time.Time {
	if it.SentAt != nil {
		return *it.SentAt
	}
	return it.DueAt
}

func (b *builder) groups(rng string) []Group {
	sort.SliceStable(b.items, func(i, j int) bool {
		ti, tj := sortKey(b.items[i]), sortKey(b.items[j])
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return b.items[i].ID < b.items[j].ID
	})
	if len(b.items) > maxItems {
		b.items = b.items[:maxItems]
	}
	today := Group{Key: "today", Label: "Heute", Items: []Item{}}
	week := Group{Key: "week", Label: "Diese Woche", Items: []Item{}}
	for _, it := range b.items {
		// Overdue items (before today) belong to "Heute".
		if sortKey(it).Before(b.w.tomorrow) {
			today.Items = append(today.Items, it)
		} else {
			week.Items = append(week.Items, it)
		}
	}
	if rng == RangeToday {
		return []Group{today}
	}
	return []Group{today, week}
}

// ------------------------------------------------------------ stored rows

func (b *builder) storedItems(ctx context.Context) error {
	rows, err := b.pool.Query(ctx, `
		SELECT r.id::text, r.kind, r.title, COALESCE(r.body, ''), r.due_at, r.sent_at,
		       COALESCE(r.source_table, ''), COALESCE(r.source_id::text, ''),
		       COALESCE(hi.horse_id, rp.horse_id)::text
		FROM reminders r
		LEFT JOIN health_items hi ON r.source_table = 'health_items' AND hi.id = r.source_id
		LEFT JOIN reha_plans rp ON r.source_table = 'reha_plans' AND rp.id = r.source_id
		WHERE r.stable_id = $1 AND r.user_id = $2 AND r.dismissed_at IS NULL
		  AND COALESCE(r.sent_at, r.due_at) >= $3 AND COALESCE(r.sent_at, r.due_at) < $4
		ORDER BY COALESCE(r.sent_at, r.due_at), r.id`,
		b.u.StableID, b.u.ID, b.w.today, b.w.end)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var it Item
		var table, source string
		var horseID *string
		if err := rows.Scan(&it.ID, &it.Kind, &it.Title, &it.Body, &it.DueAt, &it.SentAt, &table, &source, &horseID); err != nil {
			return err
		}
		it.Dismissible = true
		hid := ""
		if horseID != nil {
			hid = *horseID
		}
		switch table {
		case "blanket_night", "blanket_checkout":
			it.Screen = screenBlankets
		case "blanket_weather":
			it.Screen = horseScreen(source, "blanket-plan")
		case "health_items":
			it.Screen = horseScreen(hid, "health")
		case "reha_plans":
			it.Screen = horseScreen(hid, "reha")
		case "requests":
			if source != "" {
				it.Screen = "/requests/" + source
			}
		case "training_plan":
			it.Screen = screenWeek
		}
		b.stored[storedKey(it.Kind, table, source, it.DueAt, b.w)] = true
		b.items = append(b.items, it)
	}
	return rows.Err()
}

// storedKey identifies what a stored reminder is about. Computed items build the same key.
func storedKey(kind, table, source string, due time.Time, w window) string {
	if table == "training_plan" {
		// The push is claimed for the evening before; the plan is for the next day.
		return "training_plan:" + pgDate(w.day(due).AddDate(0, 0, 1))
	}
	return kind + ":" + source
}

// ------------------------------------------------------------ computed items

// ownHorses is the SQL condition "the user owns or rides horse h".
const ownHorses = `(h.owner_id = $2 OR EXISTS (SELECT 1 FROM horse_riders r WHERE r.horse_id = h.id AND r.user_id = $2))`

// healthItems lists, for the user's horses, permanent medication of today and health items
// that are due (or overdue) within the window.
func (b *builder) healthItems(ctx context.Context) error {
	rows, err := b.pool.Query(ctx, `
		SELECT hi.id::text, h.id::text, h.name, hi.kind, hi.label, hi.due_date, COALESCE(to_char(hi.daily_time, 'HH24:MI'), '')
		FROM health_items hi JOIN horses h ON h.id = hi.horse_id
		WHERE hi.stable_id = $1 AND `+ownHorses+`
		  AND ((hi.daily_time IS NOT NULL AND (hi.due_date IS NULL OR hi.due_date >= $3::date))
		    OR (hi.daily_time IS NULL AND hi.due_date IS NOT NULL AND hi.due_date < $4::date))
		ORDER BY hi.due_date NULLS LAST, h.name, hi.id`,
		b.u.StableID, b.u.ID, pgDate(b.w.today), pgDate(b.w.end))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, horseID, horse, kind, label, daily string
		var due *time.Time
		if err := rows.Scan(&id, &horseID, &horse, &kind, &label, &due, &daily); err != nil {
			return err
		}
		it := Item{Computed: true, Screen: horseScreen(horseID, "health")}
		if daily != "" {
			var hh, mm int
			if _, err := fmt.Sscanf(daily, "%d:%d", &hh, &mm); err != nil {
				continue
			}
			it.Kind = push.KindMedication
			it.ID = "c:medication:" + id + ":" + pgDate(b.w.today)
			it.Title = "Medikament: " + horse
			it.Body = fmt.Sprintf("%s, %s Uhr.", label, daily)
			it.DueAt = b.w.at(b.w.today, hh, mm)
			if b.stored[it.Kind+":"+id] {
				continue
			}
		} else {
			d := b.w.localDate(*due)
			it.Kind = push.KindHealthDue
			it.ID = "c:health_due:" + id + ":" + pgDate(d)
			it.Title = "Fällig: " + horse
			it.Body = fmt.Sprintf("%s: %s (%s).", label, dueText(daysBetween(b.w.today, d)), d.Format("02.01.2006"))
			it.DueAt, it.AllDay = d, true
			if b.stored[it.Kind+":"+id] {
				continue
			}
		}
		b.items = append(b.items, it)
	}
	return rows.Err()
}

// daysBetween counts calendar days from a to b (both local midnights; DST safe).
func daysBetween(a, b time.Time) int {
	ua := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	ub := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(ub.Sub(ua).Hours() / 24)
}

// dueText: "heute", "in 3 Tagen", "seit 2 Tagen überfällig".
func dueText(days int) string {
	switch {
	case days < -1:
		return fmt.Sprintf("seit %d Tagen überfällig", -days)
	case days == -1:
		return "seit gestern überfällig"
	case days == 0:
		return "heute"
	case days == 1:
		return "morgen"
	}
	return fmt.Sprintf("in %d Tagen", days)
}

// rehaItems lists vet checkups of active reha plans of the user's horses within the window.
func (b *builder) rehaItems(ctx context.Context) error {
	rows, err := b.pool.Query(ctx, `
		SELECT p.id::text, h.id::text, h.name, p.checkup_date, COALESCE(p.vet, '')
		FROM reha_plans p JOIN horses h ON h.id = p.horse_id
		WHERE p.stable_id = $1 AND p.active AND `+ownHorses+`
		  AND p.checkup_date >= $3::date AND p.checkup_date < $4::date
		ORDER BY p.checkup_date, h.name, p.id`,
		b.u.StableID, b.u.ID, pgDate(b.w.today), pgDate(b.w.end))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, horseID, horse, vet string
		var date time.Time
		if err := rows.Scan(&id, &horseID, &horse, &date, &vet); err != nil {
			return err
		}
		if b.stored[push.KindRehaCheckup+":"+id] {
			continue
		}
		d := b.w.localDate(date)
		body := fmt.Sprintf("Kontrolle am %s.", d.Format("02.01.2006"))
		if vet != "" {
			body = fmt.Sprintf("Kontrolle bei %s am %s.", vet, d.Format("02.01.2006"))
		}
		b.items = append(b.items, Item{
			ID: "c:reha_checkup:" + id + ":" + pgDate(d), Kind: push.KindRehaCheckup,
			Title: "Reha-Kontrolle: " + horse, Body: body, DueAt: d, AllDay: true,
			Screen: horseScreen(horseID, "reha"), Computed: true,
		})
	}
	return rows.Err()
}

var requestTypeLabels = map[string]string{
	"blanket":               "Decken",
	"show_helper":           "Turniertrottel",
	"ride_share":            "Mitfahrgelegenheit",
	"exercise":              "Bewegen",
	"feed_or_turnout":       "Füttern und Rausstellen",
	"appointment_companion": "Terminbegleitung",
	"other":                 "Hilfe gesucht",
}

// requestItems lists open or assigned requests the user helps with within the window.
func (b *builder) requestItems(ctx context.Context) error {
	rows, err := b.pool.Query(ctx, `
		SELECT q.id::text, q.type, COALESCE(h.name, ''), q.date, COALESCE(to_char(q.time_from, 'HH24:MI'), ''),
		       COALESCE(q.location, '')
		FROM requests q
		JOIN request_assignees a ON a.request_id = q.id AND a.user_id = $2
		LEFT JOIN horses h ON h.id = q.horse_id
		WHERE q.stable_id = $1 AND q.status IN ('open', 'assigned')
		  AND COALESCE(q.date_end, q.date) >= $3::date AND q.date < $4::date
		ORDER BY q.date, q.time_from NULLS FIRST, q.id`,
		b.u.StableID, b.u.ID, pgDate(b.w.today), pgDate(b.w.end))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, typ, horse, from, location string
		var date time.Time
		if err := rows.Scan(&id, &typ, &horse, &date, &from, &location); err != nil {
			return err
		}
		if b.stored[push.KindHelper+":"+id] {
			continue
		}
		d := b.w.localDate(date)
		if d.Before(b.w.today) { // a multi-day request that already started
			d = b.w.today
		}
		title := requestTypeLabels[typ]
		if title == "" {
			title = "Anfrage"
		}
		if horse != "" {
			title += ": " + horse
		}
		it := Item{ID: "c:helper:" + id + ":" + pgDate(d), Kind: push.KindHelper, Title: title,
			Body: "Du hilfst mit.", DueAt: d, AllDay: from == "", Screen: "/requests/" + id, Computed: true}
		if from != "" {
			var hh, mm int
			if _, err := fmt.Sscanf(from, "%d:%d", &hh, &mm); err == nil {
				it.DueAt = b.w.at(d, hh, mm)
			}
		}
		if location != "" {
			it.Body = "Du hilfst mit · " + location
		}
		b.items = append(b.items, it)
	}
	return rows.Err()
}

// TrainingPlanHour is the stable-local hour of the evening-before training plan.
const TrainingPlanHour = 19

var weekdayNames = [...]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}

// trainingTitle is "Training morgen" or "Training am Freitag".
func trainingTitle(day, today time.Time) string {
	if day.Equal(today.AddDate(0, 0, 1)) {
		return "Training morgen"
	}
	return "Training am " + weekdayNames[day.Weekday()]
}

type slotText struct{ horse, activity string }

// trainingBody is "Luna: Ausritt, Fanta: Longe".
func trainingBody(slots []slotText) string {
	parts := make([]string, len(slots))
	for i, s := range slots {
		parts[i] = s.horse
		if s.activity != "" {
			parts[i] += ": " + training.Activity(s.activity).GermanName()
		}
	}
	return strings.Join(parts, ", ")
}

// trainingItems lists the evening-before plan for every day of the window (the item for a
// day is due at 19:00 the day before) when the user has planned week slots that day. Users who
// switched off the kind training_plan do not get it ("optional").
func (b *builder) trainingItems(ctx context.Context) error {
	var off bool
	if err := b.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM reminder_settings
		WHERE user_id = $1 AND kind = $2 AND NOT enabled)`, b.u.ID, push.KindTrainingPlan).Scan(&off); err != nil {
		return err
	}
	if off {
		return nil
	}
	// Plans for day D are due on D-1, so D runs from tomorrow up to the last day of the window plus one.
	rows, err := b.pool.Query(ctx, `
		SELECT ws.day, h.name, COALESCE(ws.activity, '')
		FROM week_slots ws JOIN horses h ON h.id = ws.horse_id
		WHERE ws.stable_id = $1 AND ws.user_id = $2 AND ws.status = 'planned'
		  AND ws.day >= $3::date AND ws.day <= $4::date
		ORDER BY ws.day, h.name`,
		b.u.StableID, b.u.ID, pgDate(b.w.tomorrow), pgDate(b.w.end))
	if err != nil {
		return err
	}
	defer rows.Close()
	byDay := map[string][]slotText{}
	var days []time.Time
	for rows.Next() {
		var day time.Time
		var s slotText
		if err := rows.Scan(&day, &s.horse, &s.activity); err != nil {
			return err
		}
		d := b.w.localDate(day)
		key := pgDate(d)
		if _, ok := byDay[key]; !ok {
			days = append(days, d)
		}
		byDay[key] = append(byDay[key], s)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, d := range days {
		key := pgDate(d)
		if b.stored["training_plan:"+key] {
			continue
		}
		b.items = append(b.items, Item{
			ID: "c:training_plan:" + key, Kind: push.KindTrainingPlan,
			Title: trainingTitle(d, b.w.today), Body: trainingBody(byDay[key]),
			DueAt: b.w.at(d.AddDate(0, 0, -1), TrainingPlanHour, 0), Screen: screenWeek, Computed: true,
		})
	}
	return nil
}

// ------------------------------------------------------------ blanket check

// blanketCheck is tonight's check: the stable's reminder time and how many horses already
// have a state for the blanket day (the local calendar day of tonight).
func (b *builder) blanketCheck(ctx context.Context, reminder string) (BlanketCheck, error) {
	var hh, mm int
	if _, err := fmt.Sscanf(reminder, "%d:%d", &hh, &mm); err != nil {
		hh, mm = 20, 30
	}
	c := BlanketCheck{Day: pgDate(b.w.today), Time: reminder, DueAt: b.w.at(b.w.today, hh, mm), Screen: screenBlankets}
	err := b.pool.QueryRow(ctx, `
		SELECT count(*)::int,
		       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM blanket_states s WHERE s.horse_id = h.id AND s.day = $2::date))::int
		FROM horses h WHERE h.stable_id = $1`, b.u.StableID, c.Day).Scan(&c.Total, &c.Done)
	if err != nil {
		return c, err
	}
	switch {
	case c.Total == 0:
		c.State = "empty"
	case c.Done >= c.Total:
		c.State = "done"
	case !b.w.now.Before(c.DueAt):
		c.State = "due"
	default:
		c.State = "upcoming"
	}
	return c, nil
}
