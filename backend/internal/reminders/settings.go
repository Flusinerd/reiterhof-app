package reminders

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
)

// KindInfo describes one push kind for the settings screen.
type KindInfo struct {
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Description string `json:"description"`
	// Enabled is what the notifier will do for this user: opt-out kinds are on unless a
	// row switches them off, opt-in kinds (push.OptIn) are off unless a row switches them on.
	Enabled bool `json:"enabled"`
	// OptIn marks kinds that are off by default.
	OptIn bool `json:"opt_in"`
}

// catalog lists every push kind in display order. A test checks that it covers push.Kinds().
var catalog = []struct{ kind, label, description string }{
	{push.KindLastPerson, "Decken: letzte Person", "Erinnerung am Abend, wenn noch Pferde ohne Decken-Eintrag sind und du die Letzte oder der Letzte im Stall bist."},
	{push.KindWeatherChange, "Decken: Wetteränderung", "Hinweis, wenn sich die Decken-Empfehlung für eines deiner Pferde ändert."},
	{push.KindMedication, "Medikamente", "Erinnerung zur Uhrzeit, zu der ein Pferd von dir sein Medikament bekommt."},
	{push.KindHealthDue, "Fällige Termine", "Hufschmied, Impfung, Wurmkur und Co. für deine Pferde, eine Woche und einen Tag vorher."},
	{push.KindRehaCheckup, "Reha-Kontrolle", "Kontrolltermin beim Tierarzt, zwei Tage vorher und am Tag selbst."},
	{push.KindHelper, "Anfragen: Erinnerung und Änderungen", "Erinnerung, wenn du bei einer Anfrage hilfst, und Meldungen zu deinen Anfragen (jemand hilft mit, Änderungen, Absagen)."},
	{push.KindNewRequest, "Anfragen: neue Anfragen", "Eine Meldung, sobald jemand im Stall eine neue Anfrage stellt. Standardmäßig aus."},
	{push.KindTrainingPlan, "Trainingsplan am Vorabend", "Am Abend vorher: welche Pferde du morgen laut Wochenplan bewegst."},
	{push.KindUrgentObservation, "Auffälligkeiten: dringend", "Wenn bei einem Pferd etwas Dringendes auffällt und du im Stall bist."},
	{push.KindObservation, "Auffälligkeiten: Hinweise", "Neue Auffälligkeiten, die nicht dringend sind."},
}

// enabledFor applies the opt-in/opt-out semantics of push.Notifier to an optional row.
func enabledFor(kind string, row *bool) bool {
	if push.OptIn(kind) {
		return row != nil && *row
	}
	return row == nil || *row
}

type notificationsResponse struct {
	Items []KindInfo `json:"items"`
	// PushConsent tells whether the user currently allows push notifications (privacy
	// consent `push`). Without it nothing is sent, whatever the switches say.
	PushConsent bool `json:"push_consent"`
}

func (h *handler) loadNotifications(r *http.Request, userID string) (notificationsResponse, error) {
	rows, err := h.deps.Pool.Query(r.Context(), `SELECT kind, enabled FROM reminder_settings WHERE user_id = $1`, userID)
	if err != nil {
		return notificationsResponse{}, err
	}
	defer rows.Close()
	set := map[string]bool{}
	for rows.Next() {
		var kind string
		var on bool
		if err := rows.Scan(&kind, &on); err != nil {
			return notificationsResponse{}, err
		}
		set[kind] = on
	}
	if err := rows.Err(); err != nil {
		return notificationsResponse{}, err
	}
	resp := notificationsResponse{Items: make([]KindInfo, 0, len(catalog))}
	for _, c := range catalog {
		var row *bool
		if v, ok := set[c.kind]; ok {
			row = &v
		}
		resp.Items = append(resp.Items, KindInfo{Kind: c.kind, Label: c.label, Description: c.description,
			Enabled: enabledFor(c.kind, row), OptIn: push.OptIn(c.kind)})
	}
	err = h.deps.Pool.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM consents
		WHERE user_id = $1 AND kind = 'push' AND revoked_at IS NULL)`, userID).Scan(&resp.PushConsent)
	return resp, err
}

// getNotifications handles GET /settings/notifications.
func (h *handler) getNotifications(w http.ResponseWriter, r *http.Request) {
	resp, err := h.loadNotifications(r, user(r).ID)
	if err != nil {
		h.internal(w, "load notification settings", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// putNotification handles PUT /settings/notifications/{kind} {"enabled": bool} and returns
// the changed item. The same reminder_settings row is used by the push notifier and by
// GET/PATCH /requests/notify-settings (kind new_request).
func (h *handler) putNotification(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	kind := r.PathValue("kind")
	if !push.ValidKind(kind) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "unknown notification kind")
		return
	}
	var in struct {
		Enabled *bool `json:"enabled"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Enabled == nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "enabled (boolean) is required")
		return
	}
	_, err := h.deps.Pool.Exec(r.Context(), `
		INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, kind) DO UPDATE SET enabled = EXCLUDED.enabled`, u.StableID, u.ID, kind, *in.Enabled)
	if err != nil {
		h.internal(w, "save notification setting", err)
		return
	}
	resp, err := h.loadNotifications(r, u.ID)
	if err != nil {
		h.internal(w, "load notification settings", err)
		return
	}
	for _, it := range resp.Items {
		if it.Kind == kind {
			httpx.WriteJSON(w, http.StatusOK, it)
			return
		}
	}
	h.internal(w, "kind missing from catalog", errors.New(kind))
}

// ------------------------------------------------------------ reminder time

// The last-person job starts at the reminder time and re-checks every 15 minutes until
// 22:00, so the time has to leave room before that and must stay in the evening (the
// blanket day rolls over at 04:00).
const (
	minReminderMinutes = 16 * 60
	maxReminderMinutes = 22 * 60
)

var hhmmRe = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

// parseReminderTime validates "HH:MM" between 16:00 and 22:00.
func parseReminderTime(s string) (string, error) {
	m := hhmmRe.FindStringSubmatch(s)
	if m == nil {
		return "", errors.New("reminder_time must be HH:MM")
	}
	hh, _ := strconv.Atoi(m[1])
	mm, _ := strconv.Atoi(m[2])
	if total := hh*60 + mm; total < minReminderMinutes || total > maxReminderMinutes {
		return "", fmt.Errorf("reminder_time must be between %02d:00 and %02d:00", minReminderMinutes/60, maxReminderMinutes/60)
	}
	return s, nil
}

type reminderTimeResponse struct {
	ReminderTime string `json:"reminder_time"`
	// CanEdit is true for admins; everybody else only sees the time.
	CanEdit bool `json:"can_edit"`
	// AutoUncoverTime ("HH:MM", null = off) and AutoUncoverDays (ISO weekdays, 1 = Monday)
	// are the read-only settings of the automatic uncovering (JAN-78); the operator sets
	// them with stallfunk-admin.
	AutoUncoverTime *string `json:"auto_uncover_time"`
	AutoUncoverDays []int16 `json:"auto_uncover_days"`
}

// getReminderTime handles GET /stables/reminder-time (every member).
func (h *handler) getReminderTime(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	resp := reminderTimeResponse{CanEdit: u.IsAdmin}
	err := h.deps.Pool.QueryRow(r.Context(),
		`SELECT to_char(reminder_time, 'HH24:MI'), to_char(auto_uncover_time, 'HH24:MI'), auto_uncover_days
		FROM stables WHERE id = $1`, u.StableID).Scan(&resp.ReminderTime, &resp.AutoUncoverTime, &resp.AutoUncoverDays)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "stable not found")
		return
	}
	if err != nil {
		h.internal(w, "load reminder time", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// putReminderTime handles PUT /stables/reminder-time {"reminder_time": "20:45"} (admin only).
// The blanket reminder job reads stables.reminder_time on every run, so the change applies
// from the next evening without a restart.
func (h *handler) putReminderTime(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	var in struct {
		ReminderTime string `json:"reminder_time"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	t, err := parseReminderTime(in.ReminderTime)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}
	resp := reminderTimeResponse{CanEdit: true}
	err = h.deps.Pool.QueryRow(r.Context(), `UPDATE stables SET reminder_time = $2::time WHERE id = $1
		RETURNING to_char(reminder_time, 'HH24:MI'), to_char(auto_uncover_time, 'HH24:MI'), auto_uncover_days`,
		u.StableID, t).Scan(&resp.ReminderTime, &resp.AutoUncoverTime, &resp.AutoUncoverDays)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "stable not found")
		return
	}
	if err != nil {
		h.internal(w, "save reminder time", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
