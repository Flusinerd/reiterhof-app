package presence

import (
	"context"
	"errors"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Visibility levels of users.presence_visibility.
const (
	VisibilityAll     = "all"
	VisibilityOnlyDay = "only_day"
	VisibilityHidden  = "hidden"
)

// Me is the caller's own, complete presence data.
type Me struct {
	OpenVisit  *Visit `json:"open_visit"`
	LastVisit  *Visit `json:"last_visit"`
	Visibility string `json:"visibility"`
}

// Here is a person who is at the stable right now. Since is nil for `only_day` people
// (others may see that they are there today, but not when they came).
type Here struct {
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	AvatarColor *string    `json:"avatar_color"`
	Since       *time.Time `json:"since"`
}

// Recent is the last visit of a person who is not here now ("zuletzt gesehen").
// LastSeenDate is the stable-local calendar day (YYYY-MM-DD); LastSeenAt is nil for
// `only_day` people. UsualArrivalHour (0-23) is only set for `all` people with enough
// visits in the last eight weeks.
type Recent struct {
	UserID           string     `json:"user_id"`
	Name             string     `json:"name"`
	AvatarColor      *string    `json:"avatar_color"`
	LastSeenDate     string     `json:"last_seen_date"`
	LastSeenAt       *time.Time `json:"last_seen_at"`
	Today            bool       `json:"today"`
	UsualArrivalHour *int       `json:"usual_arrival_hour"`
}

// Overview is the response of GET /api/v1/presence.
type Overview struct {
	Me     Me       `json:"me"`
	Here   []Here   `json:"here"`
	Recent []Recent `json:"recent"`
}

func (h *handler) overview(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	o, err := Load(r.Context(), h.deps.Pool, user.StableID, user.ID, h.deps.Now())
	if err != nil {
		h.internal(w, "overview", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

// Load builds the overview for one user of a stable. The visibility of the other people
// is enforced here, in SQL and in Go; there is no admin exception. The caller sees
// everything about themselves and is not part of Here/Recent.
func Load(ctx context.Context, pool *pgxpool.Pool, stableID, userID string, now time.Time) (Overview, error) {
	o := Overview{Here: []Here{}, Recent: []Recent{}}

	var tz string
	if err := pool.QueryRow(ctx, `SELECT timezone FROM stables WHERE id = $1`, stableID).Scan(&tz); err != nil {
		return o, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	// --- me
	if err := pool.QueryRow(ctx, `SELECT presence_visibility FROM users WHERE id = $1 AND stable_id = $2`,
		userID, stableID).Scan(&o.Me.Visibility); err != nil {
		return o, err
	}
	for _, open := range []bool{true, false} {
		cond := "left_at IS NULL"
		if !open {
			cond = "left_at IS NOT NULL"
		}
		var v Visit
		err := pool.QueryRow(ctx, `SELECT id, arrived_at, left_at, source FROM presence
			WHERE user_id = $1 AND stable_id = $2 AND `+cond+` ORDER BY arrived_at DESC LIMIT 1`, userID, stableID).
			Scan(&v.ID, &v.ArrivedAt, &v.LeftAt, &v.Source)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return o, err
		}
		if open {
			o.Me.OpenVisit = &v
		} else {
			o.Me.LastVisit = &v
		}
	}

	// --- here: hidden people are excluded by the query
	rows, err := pool.Query(ctx, `SELECT u.id, u.name, u.avatar_color, u.presence_visibility, p.arrived_at
		FROM presence p JOIN users u ON u.id = p.user_id AND u.stable_id = p.stable_id
		WHERE p.stable_id = $1 AND p.left_at IS NULL AND u.id <> $2 AND u.presence_visibility <> 'hidden'`,
		stableID, userID)
	if err != nil {
		return o, err
	}
	for rows.Next() {
		var e Here
		var vis string
		var arrived time.Time
		if err := rows.Scan(&e.UserID, &e.Name, &e.AvatarColor, &vis, &arrived); err != nil {
			rows.Close()
			return o, err
		}
		if vis == VisibilityAll {
			e.Since = &arrived
		}
		o.Here = append(o.Here, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return o, err
	}
	sort.Slice(o.Here, func(i, j int) bool { return strings.ToLower(o.Here[i].Name) < strings.ToLower(o.Here[j].Name) })

	// --- recent: last finished visit of people who are not here now
	rows, err = pool.Query(ctx, `SELECT DISTINCT ON (p.user_id) u.id, u.name, u.avatar_color, u.presence_visibility, p.left_at
		FROM presence p JOIN users u ON u.id = p.user_id AND u.stable_id = p.stable_id
		WHERE p.stable_id = $1 AND p.left_at IS NOT NULL AND p.left_at > $3
		  AND u.id <> $2 AND u.presence_visibility <> 'hidden'
		  AND NOT EXISTS (SELECT 1 FROM presence o WHERE o.user_id = p.user_id AND o.left_at IS NULL)
		ORDER BY p.user_id, p.left_at DESC`, stableID, userID, now.Add(-recentWindow))
	if err != nil {
		return o, err
	}
	type seen struct {
		Recent
		left time.Time
	}
	var list []seen
	for rows.Next() {
		var s seen
		var vis string
		if err := rows.Scan(&s.UserID, &s.Name, &s.AvatarColor, &vis, &s.left); err != nil {
			rows.Close()
			return o, err
		}
		local := s.left.In(loc)
		s.LastSeenDate = local.Format("2006-01-02")
		s.Today = s.LastSeenDate == now.In(loc).Format("2006-01-02")
		if vis == VisibilityAll {
			t := s.left
			s.LastSeenAt = &t
		}
		list = append(list, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return o, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].left.After(list[j].left) })
	if len(list) > recentLimit {
		list = list[:recentLimit]
	}

	hints, err := usualArrival(ctx, pool, stableID, tz, now)
	if err != nil {
		return o, err
	}
	for _, s := range list {
		if s.LastSeenAt != nil { // visibility all
			if hour, ok := hints[s.UserID]; ok {
				s.UsualArrivalHour = &hour
			}
		}
		o.Recent = append(o.Recent, s.Recent)
	}
	return o, nil
}

// usualArrival returns, per person with visibility `all` and at least hintMinVisits visits
// in the last eight weeks, the median arrival time (stable-local) rounded to an hour.
func usualArrival(ctx context.Context, pool *pgxpool.Pool, stableID, tz string, now time.Time) (map[string]int, error) {
	rows, err := pool.Query(ctx, `SELECT p.user_id,
			percentile_cont(0.5) WITHIN GROUP (ORDER BY
				(extract(hour FROM p.arrived_at AT TIME ZONE $3) * 60 + extract(minute FROM p.arrived_at AT TIME ZONE $3))::double precision)
		FROM presence p JOIN users u ON u.id = p.user_id AND u.stable_id = p.stable_id
		WHERE p.stable_id = $1 AND p.arrived_at > $2 AND u.presence_visibility = 'all'
		GROUP BY p.user_id HAVING count(*) >= $4`, stableID, now.Add(-hintWindow), tz, hintMinVisits)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var median float64
		if err := rows.Scan(&id, &median); err != nil {
			return nil, err
		}
		out[id] = int(math.Round(median/60)) % 24
	}
	return out, rows.Err()
}
