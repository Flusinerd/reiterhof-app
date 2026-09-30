package requests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Request statuses (CHECK constraint on requests.status).
const (
	StatusOpen      = "open"
	StatusAssigned  = "assigned"
	StatusDone      = "done"
	StatusCancelled = "cancelled"
)

// querier is implemented by *pgxpool.Pool and pgx.Tx.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Helper is a user who accepted a request.
type Helper struct {
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	AvatarColor *string   `json:"avatar_color"`
	Thanked     bool      `json:"thanked"`
	JoinedAt    time.Time `json:"joined_at"`
}

// Request is the API representation of a help request, including the viewer's
// perspective (is_creator, is_helper, can_accept).
type Request struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	HorseID       *string         `json:"horse_id"`
	HorseName     *string         `json:"horse_name"`
	HorseColorKey *string         `json:"horse_color_key"`
	CreatedBy     string          `json:"created_by"`
	CreatorName   string          `json:"creator_name"`
	Date          string          `json:"date"`     // YYYY-MM-DD, stable-local
	DateEnd       *string         `json:"date_end"` // inclusive
	TimeFrom      *string         `json:"time_from"`
	TimeTo        *string         `json:"time_to"`
	Location      string          `json:"location"`
	Description   string          `json:"description"`
	Tasks         []string        `json:"tasks"`
	HelpersNeeded int             `json:"helpers_needed"`
	Status        string          `json:"status"`
	RecurringRule *string         `json:"recurring_rule"`
	SeriesID      *string         `json:"series_id"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`

	Helpers      []Helper `json:"helpers"`
	HelpersCount int      `json:"helpers_count"`
	SpotsLeft    int      `json:"spots_left"`
	IsCreator    bool     `json:"is_creator"`
	IsHelper     bool     `json:"is_helper"`
	CanAccept    bool     `json:"can_accept"`
	// MyRemindAt is the viewer's own reminder as helper; other helpers' reminders are private.
	MyRemindAt *time.Time `json:"my_remind_at"`
}

// EndDate is the last day of the request (date_end or date).
func (r Request) EndDate() string {
	if r.DateEnd != nil {
		return *r.DateEnd
	}
	return r.Date
}

const selectRequest = `
SELECT r.id::text, r.type, r.horse_id::text, h.name, h.color_key, r.created_by::text, u.name,
       to_char(r.date, 'YYYY-MM-DD'), to_char(r.date_end, 'YYYY-MM-DD'),
       to_char(r.time_from, 'HH24:MI'), to_char(r.time_to, 'HH24:MI'),
       COALESCE(r.location, ''), COALESCE(r.description, ''), r.tasks, r.helpers_needed, r.status,
       r.recurring_rule, r.series_id::text, r.payload, r.created_at
FROM requests r
JOIN users u ON u.id = r.created_by
LEFT JOIN horses h ON h.id = r.horse_id AND h.stable_id = r.stable_id
`

func scanRequest(row pgx.Row) (Request, error) {
	var r Request
	var tasks, payload []byte
	err := row.Scan(&r.ID, &r.Type, &r.HorseID, &r.HorseName, &r.HorseColorKey, &r.CreatedBy, &r.CreatorName,
		&r.Date, &r.DateEnd, &r.TimeFrom, &r.TimeTo, &r.Location, &r.Description, &tasks, &r.HelpersNeeded,
		&r.Status, &r.RecurringRule, &r.SeriesID, &payload, &r.CreatedAt)
	if err != nil {
		return r, err
	}
	r.Tasks = []string{}
	if len(tasks) > 0 {
		if err := json.Unmarshal(tasks, &r.Tasks); err != nil {
			return r, fmt.Errorf("decode tasks: %w", err)
		}
	}
	if r.Tasks == nil {
		r.Tasks = []string{}
	}
	r.Payload = json.RawMessage(payload)
	if len(r.Payload) == 0 {
		r.Payload = json.RawMessage(`{}`)
	}
	r.Helpers = []Helper{}
	return r, nil
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validUUID(s string) bool { return uuidRe.MatchString(s) }

// listFilter selects requests of one stable.
type listFilter struct {
	Statuses   []string
	Type       string
	HorseID    string
	CreatedBy  string // mine
	AssignedTo string // I help
	From, To   string // YYYY-MM-DD; From compares with the last day, To with the first
	Limit      int
	Desc       bool
	Mine       bool // created by the viewer
	Assigned   bool // the viewer helps
}

func (s *Service) list(ctx context.Context, q querier, stableID, viewerID string, f listFilter, today string) ([]Request, error) {
	where := []string{"r.stable_id = $1"}
	args := []any{stableID}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(args))))
	}
	if len(f.Statuses) > 0 {
		add("r.status = ANY(?::text[])", f.Statuses)
	}
	if f.Type != "" {
		add("r.type = ?", f.Type)
	}
	if f.HorseID != "" {
		add("r.horse_id = ?::uuid", f.HorseID)
	}
	if f.CreatedBy != "" {
		add("r.created_by = ?::uuid", f.CreatedBy)
	}
	if f.AssignedTo != "" {
		add("EXISTS (SELECT 1 FROM request_assignees a WHERE a.request_id = r.id AND a.user_id = ?::uuid)", f.AssignedTo)
	}
	if f.From != "" {
		add("COALESCE(r.date_end, r.date) >= ?::date", f.From)
	}
	if f.To != "" {
		add("r.date <= ?::date", f.To)
	}
	order := "r.date, r.time_from NULLS LAST, r.created_at"
	if f.Desc {
		order = "r.date DESC, r.time_from DESC NULLS LAST, r.created_at DESC"
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := q.Query(ctx, selectRequest+" WHERE "+strings.Join(where, " AND ")+
		" ORDER BY "+order+fmt.Sprintf(" LIMIT %d", limit), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Request
	for rows.Next() {
		r, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err := s.attach(ctx, q, stableID, viewerID, today, out); err != nil {
		return nil, err
	}
	return out, nil
}

// get loads one request of the stable; pgx.ErrNoRows if it does not exist there.
func (s *Service) get(ctx context.Context, q querier, stableID, viewerID, id, today string) (Request, error) {
	if !validUUID(id) {
		return Request{}, pgx.ErrNoRows
	}
	r, err := scanRequest(q.QueryRow(ctx, selectRequest+" WHERE r.stable_id = $1 AND r.id = $2::uuid", stableID, id))
	if err != nil {
		return Request{}, err
	}
	list := []Request{r}
	if err := s.attach(ctx, q, stableID, viewerID, today, list); err != nil {
		return Request{}, err
	}
	return list[0], nil
}

// attach loads the helpers of the requests and derives the viewer flags.
func (s *Service) attach(ctx context.Context, q querier, stableID, viewerID, today string, reqs []Request) error {
	if len(reqs) == 0 {
		return nil
	}
	ids := make([]string, len(reqs))
	idx := make(map[string]int, len(reqs))
	for i, r := range reqs {
		ids[i] = r.ID
		idx[r.ID] = i
	}
	rows, err := q.Query(ctx, `
		SELECT a.request_id::text, u.id::text, u.name, u.avatar_color, a.thanked, a.created_at, a.remind_at
		FROM request_assignees a JOIN users u ON u.id = a.user_id
		WHERE a.stable_id = $1 AND a.request_id = ANY($2::uuid[])
		ORDER BY a.created_at, u.name`, stableID, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var reqID string
		var h Helper
		var remind *time.Time
		if err := rows.Scan(&reqID, &h.UserID, &h.Name, &h.AvatarColor, &h.Thanked, &h.JoinedAt, &remind); err != nil {
			return err
		}
		i := idx[reqID]
		if h.UserID == viewerID {
			reqs[i].MyRemindAt = remind
		}
		reqs[i].Helpers = append(reqs[i].Helpers, h)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range reqs {
		r := &reqs[i]
		r.HelpersCount = len(r.Helpers)
		r.SpotsLeft = max(0, r.HelpersNeeded-r.HelpersCount)
		r.IsCreator = r.CreatedBy == viewerID
		for _, h := range r.Helpers {
			if h.UserID == viewerID {
				r.IsHelper = true
			}
		}
		r.CanAccept = r.Status == StatusOpen && !r.IsCreator && !r.IsHelper && r.SpotsLeft > 0 && r.EndDate() >= today
	}
	return nil
}

// lockRow is the part of a request the state transitions need, read FOR UPDATE.
type lockRow struct {
	ID            string
	Type          string
	CreatedBy     string
	Status        string
	HelpersNeeded int
	Date          string
	EndDate       string
	SeriesID      *string
	Helpers       int
}

var errNotFound = errors.New("request not found")

// lock reads and locks a request of the stable (serialises concurrent accepts).
func lock(ctx context.Context, tx pgx.Tx, stableID, id string) (lockRow, error) {
	var l lockRow
	if !validUUID(id) {
		return l, errNotFound
	}
	err := tx.QueryRow(ctx, `
		SELECT id::text, type, created_by::text, status, helpers_needed,
		       to_char(date, 'YYYY-MM-DD'), to_char(COALESCE(date_end, date), 'YYYY-MM-DD'),
		       series_id::text
		FROM requests WHERE stable_id = $1 AND id = $2::uuid FOR UPDATE`, stableID, id).
		Scan(&l.ID, &l.Type, &l.CreatedBy, &l.Status, &l.HelpersNeeded, &l.Date, &l.EndDate, &l.SeriesID)
	if errors.Is(err, pgx.ErrNoRows) {
		return l, errNotFound
	}
	if err != nil {
		return l, err
	}
	err = tx.QueryRow(ctx, `SELECT count(*) FROM request_assignees WHERE request_id = $1::uuid`, id).Scan(&l.Helpers)
	return l, err
}

// stableLocation returns the stable's time zone (Europe/Berlin if unknown).
func stableLocation(ctx context.Context, q querier, stableID string) *time.Location {
	var name string
	if err := q.QueryRow(ctx, `SELECT timezone FROM stables WHERE id = $1`, stableID).Scan(&name); err == nil {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.UTC
	}
	return loc
}

func today(now time.Time, loc *time.Location) string { return now.In(loc).Format("2006-01-02") }

func isUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

// defaultReminder is the day before the (first) date at 18:00 stable-local time.
func defaultReminder(date string, loc *time.Location) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return time.Time{}, err
	}
	y, m, day := d.AddDate(0, 0, -1).Date()
	return time.Date(y, m, day, 18, 0, 0, 0, loc), nil
}
