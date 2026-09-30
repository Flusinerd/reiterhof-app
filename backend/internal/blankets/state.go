package blankets

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/requests"
)

// horseRef is the short horse description embedded in responses.
type horseRef struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Box      *string `json:"box"`
	ColorKey *string `json:"color_key"`
}

func (h horseInfo) ref() horseRef {
	return horseRef{ID: h.ID, Name: h.Name, Box: h.Box, ColorKey: h.ColorKey}
}

// --------------------------------------------------------------------- plan

type planResponse struct {
	Horse          horseRef       `json:"horse"`
	Day            string         `json:"day"`
	Weather        *Weather       `json:"weather"`
	Recommendation Recommendation `json:"recommendation"`
	Rules          []Rule         `json:"rules"`
	Blankets       []Blanket      `json:"blankets"`
	HelperNote     *string        `json:"helper_note"`
	State          *State         `json:"state"`
	CanManage      bool           `json:"can_manage"`
}

// plan answers GET /horses/{id}/blanket-plan: tonight's forecast, the matched rule, the
// recommended blanket, all rules and blankets and the helper note.
func (h *handler) plan(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, false)
	if !ok {
		return
	}
	n, err := h.svc.loadNight(r.Context(), user.StableID, horseID)
	if err != nil || len(n.Horses) != 1 {
		h.internal(w, "load plan", errors.Join(err, errors.New("horse not loaded")))
		return
	}
	can, err := auth.CanManageHorse(r.Context(), h.deps.Pool, user, horseID)
	if err != nil {
		h.internal(w, "role lookup", err)
		return
	}
	hn := n.Horses[0]
	rules, blankets := hn.Rules, hn.Blankets
	if rules == nil {
		rules = []Rule{}
	}
	if blankets == nil {
		blankets = []Blanket{}
	}
	httpx.WriteJSON(w, http.StatusOK, planResponse{
		Horse: hn.Horse.ref(), Day: n.Day, Weather: n.Weather, Recommendation: hn.Rec,
		Rules: rules, Blankets: blankets, HelperNote: hn.Horse.HelperNote, State: hn.State, CanManage: can,
	})
}

// -------------------------------------------------------------------- today

type todayHorse struct {
	Horse          horseRef       `json:"horse"`
	Recommendation Recommendation `json:"recommendation"`
	State          *State         `json:"state"`
	Done           bool           `json:"done"`
	IsMine         bool           `json:"is_mine"`
}

type todayResponse struct {
	Day          string       `json:"day"`
	Weather      *Weather     `json:"weather"`
	ReminderTime string       `json:"reminder_time"`
	Progress     progress     `json:"progress"`
	Horses       []todayHorse `json:"horses"`
}

type progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// today answers GET /blankets/today: the overview of all horses of the stable, open
// horses first (then by name), with the progress "done/total".
func (h *handler) today(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	n, err := h.svc.loadNight(r.Context(), user.StableID, "")
	if err != nil {
		h.internal(w, "load today", err)
		return
	}
	resp := todayResponse{Day: n.Day, Weather: n.Weather, ReminderTime: n.ReminderTime, Horses: []todayHorse{}}
	for _, hn := range n.Horses {
		mine := hn.Horse.OwnerID != nil && *hn.Horse.OwnerID == user.ID
		resp.Horses = append(resp.Horses, todayHorse{Horse: hn.Horse.ref(), Recommendation: hn.Rec, State: hn.State,
			Done: hn.done(), IsMine: mine})
		if hn.done() {
			resp.Progress.Done++
		}
	}
	resp.Progress.Total = len(resp.Horses)
	// Open first; the loader already sorted by name and the sort is stable.
	sort.SliceStable(resp.Horses, func(i, j int) bool { return !resp.Horses[i].Done && resp.Horses[j].Done })
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// -------------------------------------------------------------------- state

type stateInput struct {
	Action      string  `json:"action"`
	CoveredWith *string `json:"covered_with"`
}

// setState records the day state of a horse. Any member of the stable may do it.
//
// covered_with is only allowed with "covered" and must be a blanket of the horse; when
// it is omitted and the plan recommends a blanket, that blanket is stored. Posting the
// same action and blanket as the newest state of the night is a no-op (double taps do not
// clutter the history). The night is StateDay: "uncovered" in the morning ends the past night. Open blanket requests of the horse for that night are closed
// (JAN-39). The event blanket_state.changed is sent after commit.
func (h *handler) setState(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, false)
	if !ok {
		return
	}
	var in stateInput
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	switch in.Action {
	case ActionCovered, ActionUncovered, ActionChecked:
	default:
		invalid(w, "action must be covered, uncovered or checked")
		return
	}
	if in.CoveredWith != nil && in.Action != ActionCovered {
		invalid(w, "covered_with is only allowed with action covered")
		return
	}

	n, err := h.svc.loadNight(r.Context(), user.StableID, horseID)
	if err != nil || len(n.Horses) != 1 {
		h.internal(w, "load night", errors.Join(err, errors.New("horse not loaded")))
		return
	}
	hn := n.Horses[0]
	var blanketName string
	var coveredWith *string
	if in.Action == ActionCovered {
		coveredWith = in.CoveredWith
		if coveredWith == nil && hn.Rec.Blanket != nil {
			coveredWith = &hn.Rec.Blanket.ID
		}
		if coveredWith != nil {
			found := false
			for _, b := range hn.Blankets {
				if b.ID == *coveredWith {
					found, blanketName = true, b.Name
				}
			}
			if !found {
				invalid(w, "covered_with is not a blanket of this horse")
				return
			}
		}
	}

	day := StateDay(h.svc.now(), n.Loc, in.Action)

	var state State
	var closed []string
	err = h.svc.tx(r.Context(), func(tx pgx.Tx) error {
		// Serialise concurrent taps on the same horse and night.
		if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, horseID+day); err != nil {
			return err
		}
		latest, err := scanState(tx.QueryRow(r.Context(), stateSQL+`
			WHERE s.stable_id = $1 AND s.horse_id = $2 AND s.day = $3::date
			ORDER BY s.changed_at DESC, s.created_at DESC LIMIT 1`, user.StableID, horseID, day))
		switch {
		case err == nil && latest.Action == in.Action && equalPtr(latest.CoveredWith, coveredWith):
			state = latest
		case err == nil || errors.Is(err, pgx.ErrNoRows):
			state, err = insertState(r.Context(), tx, user.StableID, horseID, day, in.Action, coveredWith, h.svc.now(), &user.ID, false)
			if err != nil {
				return err
			}
		default:
			return err
		}
		closed, err = requests.CompleteBlanketRequests(r.Context(), tx, user.StableID, horseID, day,
			feedbackText(in.Action, blanketName, user.Name))
		if err != nil {
			return err
		}
		for _, id := range closed {
			if err := realtime.Publish(r.Context(), tx, user.StableID, "request.changed", map[string]any{"id": id, "kind": "done"}); err != nil {
				return err
			}
		}
		return realtime.Publish(r.Context(), tx, user.StableID, EventStateChanged, map[string]any{"horse_id": horseID, "day": day})
	})
	if err != nil {
		h.internal(w, "set state", err)
		return
	}
	if closed == nil {
		closed = []string{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"state": state, "closed_requests": closed})
}

// insertState records a state row and returns it as the API shows it. The caller holds the
// advisory lock of horse and day. changedBy is nil for automatic states.
func insertState(ctx context.Context, tx pgx.Tx, stableID, horseID, day, action string, coveredWith *string,
	at time.Time, changedBy *string, automatic bool) (State, error) {
	return scanState(tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO blanket_states (stable_id, horse_id, day, action, covered_with, changed_at, changed_by, automatic)
			VALUES ($1, $2, $3::date, $4, $5::uuid, $6, $7::uuid, $8)
			RETURNING *)
		SELECT s.id::text, s.horse_id::text, to_char(s.day, 'YYYY-MM-DD'), s.action, s.covered_with::text, b.name,
		       s.changed_at, s.changed_by::text, u.name, s.automatic
		FROM ins s LEFT JOIN blankets b ON b.id = s.covered_with LEFT JOIN users u ON u.id = s.changed_by`,
		stableID, horseID, day, action, coveredWith, at, changedBy, automatic))
}

func equalPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// feedbackText is stored as payload.feedback of a blanket request that the state closed.
func feedbackText(action, blanketName, who string) string {
	var s string
	switch action {
	case ActionCovered:
		s = "Eingedeckt"
		if blanketName != "" {
			s += " mit " + blanketName
		}
	case ActionUncovered:
		s = "Abgedeckt"
	default:
		s = "Geprüft, keine Decke nötig"
	}
	if who != "" {
		s += " (" + who + ")"
	}
	return s
}

// ------------------------------------------------------------------ history

// history answers GET /horses/{id}/blanket-states?days=14: every recorded state of the
// last days blanket days including today's, newest first.
func (h *handler) history(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, false)
	if !ok {
		return
	}
	days := defaultDays
	if v := r.URL.Query().Get("days"); v != "" {
		d, err := strconv.Atoi(v)
		if err != nil || d < 1 || d > maxDays {
			invalid(w, "days must be between 1 and 90")
			return
		}
		days = d
	}
	loc, _, err := h.svc.stableInfo(r.Context(), h.deps.Pool, user.StableID)
	if err != nil {
		h.internal(w, "stable", err)
		return
	}
	today := NightDay(h.svc.now(), loc)
	rows, err := h.deps.Pool.Query(r.Context(), stateSQL+`
		WHERE s.stable_id = $1 AND s.horse_id = $2 AND s.day > $3::date - $4::int AND s.day <= $3::date
		ORDER BY s.day DESC, s.changed_at DESC, s.created_at DESC`, user.StableID, horseID, today, days)
	if err != nil {
		h.internal(w, "history", err)
		return
	}
	defer rows.Close()
	out := []State{}
	for rows.Next() {
		st, err := scanState(rows)
		if err != nil {
			h.internal(w, "scan state", err)
			return
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		h.internal(w, "history", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"today": today, "states": out})
}
