package blankets

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// The cover window (Deckenzeitraum) of a horse starts in the afternoon or evening of the blanket
// day and ends the next morning or at noon. The end lies after the day rollover (RolloverHour)
// and not past UncoverUntilHour, from which "uncovered" belongs to the coming night. Together the
// limits stay inside the hourly forecast the weather job stores (weather.Span).
const (
	minCoverStartMinutes = 15 * 60
	maxCoverStartMinutes = 23 * 60
	minCoverEndMinutes   = RolloverHour * 60
	maxCoverEndMinutes   = UncoverUntilHour * 60
)

var hhmmRe = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

func clockMinutes(field, s string) (int, error) {
	m := hhmmRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("%s must be HH:MM", field)
	}
	hh, _ := strconv.Atoi(m[1])
	mm, _ := strconv.Atoi(m[2])
	return hh*60 + mm, nil
}

// validCoverWindow checks "HH:MM" start (15:00 to 23:00) and end (04:00 to 15:00).
func validCoverWindow(start, end string) error {
	s, err := clockMinutes("cover_start", start)
	if err != nil {
		return err
	}
	e, err := clockMinutes("cover_end", end)
	if err != nil {
		return err
	}
	if s < minCoverStartMinutes || s > maxCoverStartMinutes {
		return fmt.Errorf("cover_start must be between %02d:00 and %02d:00", minCoverStartMinutes/60, maxCoverStartMinutes/60)
	}
	if e < minCoverEndMinutes || e > maxCoverEndMinutes {
		return fmt.Errorf("cover_end must be between %02d:00 and %02d:00", minCoverEndMinutes/60, maxCoverEndMinutes/60)
	}
	return nil
}

type coverWindow struct {
	CoverStart string `json:"cover_start"`
	CoverEnd   string `json:"cover_end"`
}

// putCoverWindow handles PUT /horses/{id}/cover-window {"cover_start": "18:00", "cover_end": "12:30"}
// (owner or admin). The forecast of the horse is summarised over this window from the next read on.
func (h *handler) putCoverWindow(w http.ResponseWriter, r *http.Request) {
	user, horseID, ok := h.access(w, r, true)
	if !ok {
		return
	}
	var in coverWindow
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if err := validCoverWindow(in.CoverStart, in.CoverEnd); err != nil {
		invalid(w, err.Error())
		return
	}
	var out coverWindow
	err := h.deps.Pool.QueryRow(r.Context(), `UPDATE horses SET cover_start = $3::time, cover_end = $4::time
		WHERE id = $1 AND stable_id = $2
		RETURNING to_char(cover_start, 'HH24:MI'), to_char(cover_end, 'HH24:MI')`,
		horseID, user.StableID, in.CoverStart, in.CoverEnd).Scan(&out.CoverStart, &out.CoverEnd)
	if err != nil {
		h.internal(w, "save cover window", errors.Join(err))
		return
	}
	h.svc.hint(r.Context(), user.StableID, EventPlanChanged, map[string]any{"horse_id": horseID})
	httpx.WriteJSON(w, http.StatusOK, out)
}
