package requests

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// apiError is an error with an HTTP status; handlers turn it into the standard envelope.
type apiError struct {
	Status int
	Code   string
	Msg    string
}

func (e *apiError) Error() string { return e.Code + ": " + e.Msg }

func errf(status int, code, format string, a ...any) *apiError {
	return &apiError{Status: status, Code: code, Msg: fmt.Sprintf(format, a...)}
}

func invalid(format string, a ...any) *apiError {
	return errf(http.StatusBadRequest, "validation_failed", format, a...)
}

var (
	errForbidden = errf(http.StatusForbidden, "forbidden", "you may not change this request")
	errMissing   = errf(http.StatusNotFound, "not_found", "request not found")
)

// Input is the body of POST /api/v1/requests and, after merging a Patch, the
// validated state that is stored.
type Input struct {
	Type           string          `json:"type"`
	HorseID        *string         `json:"horse_id"`
	Date           string          `json:"date"`     // YYYY-MM-DD
	DateEnd        *string         `json:"date_end"` // inclusive
	TimeFrom       *string         `json:"time_from"`
	TimeTo         *string         `json:"time_to"`
	Location       string          `json:"location"`
	Description    string          `json:"description"`
	Tasks          []string        `json:"tasks"`
	HelpersNeeded  int             `json:"helpers_needed"`
	RecurringRule  string          `json:"recurring_rule"`
	RemindHelperAt *time.Time      `json:"remind_helper_at"`
	Payload        json.RawMessage `json:"payload"`
}

// Patch is the body of PATCH /api/v1/requests/{id}. Absent fields stay as they
// are; "" clears horse_id, date_end, time_from, time_to and remind_helper_at.
// Scope "series" applies time_from, time_to, location, description, tasks,
// helpers_needed, payload and recurring_rule to the template and all upcoming
// open occurrences; other fields are rejected in that scope.
type Patch struct {
	Scope          string          `json:"scope"` // "" or "one" | "series"
	HorseID        *string         `json:"horse_id"`
	Date           *string         `json:"date"`
	DateEnd        *string         `json:"date_end"`
	TimeFrom       *string         `json:"time_from"`
	TimeTo         *string         `json:"time_to"`
	Location       *string         `json:"location"`
	Description    *string         `json:"description"`
	Tasks          *[]string       `json:"tasks"`
	HelpersNeeded  *int            `json:"helpers_needed"`
	RecurringRule  *string         `json:"recurring_rule"`
	RemindHelperAt *string         `json:"remind_helper_at"` // RFC 3339
	Payload        json.RawMessage `json:"payload"`
}

func inputFromRequest(r Request) Input {
	in := Input{
		Type: r.Type, HorseID: r.HorseID, Date: r.Date, DateEnd: r.DateEnd, TimeFrom: r.TimeFrom, TimeTo: r.TimeTo,
		Location: r.Location, Description: r.Description, Tasks: r.Tasks, HelpersNeeded: r.HelpersNeeded,
		RemindHelperAt: r.RemindHelperAt, Payload: r.Payload,
	}
	if r.RecurringRule != nil {
		in.RecurringRule = *r.RecurringRule
	}
	return in
}

func blankToNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}

// apply merges the patch into in.
func (p Patch) apply(in *Input) error {
	if p.HorseID != nil {
		in.HorseID = blankToNil(p.HorseID)
	}
	if p.Date != nil {
		in.Date = strings.TrimSpace(*p.Date)
	}
	if p.DateEnd != nil {
		in.DateEnd = blankToNil(p.DateEnd)
	}
	if p.TimeFrom != nil {
		in.TimeFrom = blankToNil(p.TimeFrom)
	}
	if p.TimeTo != nil {
		in.TimeTo = blankToNil(p.TimeTo)
	}
	if p.Location != nil {
		in.Location = *p.Location
	}
	if p.Description != nil {
		in.Description = *p.Description
	}
	if p.Tasks != nil {
		in.Tasks = *p.Tasks
	}
	if p.HelpersNeeded != nil {
		if *p.HelpersNeeded < 1 {
			return errors.New("helpers_needed must be at least 1")
		}
		in.HelpersNeeded = *p.HelpersNeeded
	}
	if p.RecurringRule != nil {
		in.RecurringRule = strings.TrimSpace(*p.RecurringRule)
	}
	if p.RemindHelperAt != nil {
		if strings.TrimSpace(*p.RemindHelperAt) == "" {
			in.RemindHelperAt = nil
		} else {
			t, err := time.Parse(time.RFC3339, *p.RemindHelperAt)
			if err != nil {
				return errors.New("remind_helper_at must be an RFC 3339 timestamp")
			}
			in.RemindHelperAt = &t
		}
	}
	if len(p.Payload) > 0 {
		in.Payload = p.Payload
	}
	return nil
}

// normalize validates in and rewrites it into its canonical stored form.
// today is the stable-local day; checkDate rejects dates in the past.
func (in *Input) normalize(today string, checkDate bool) error {
	if !ValidType(in.Type) {
		return invalid("type must be one of %s", strings.Join(Types(), ", "))
	}
	d, err := time.Parse("2006-01-02", in.Date)
	if err != nil {
		return invalid("date must be YYYY-MM-DD")
	}
	if checkDate && in.Date < today {
		return invalid("date must not be in the past")
	}
	in.DateEnd = blankToNil(in.DateEnd)
	if in.DateEnd != nil {
		e, err := time.Parse("2006-01-02", *in.DateEnd)
		if err != nil {
			return invalid("date_end must be YYYY-MM-DD")
		}
		if e.Before(d) {
			return invalid("date_end must not be before date")
		}
		if e.Equal(d) {
			in.DateEnd = nil
		}
	}
	in.TimeFrom, in.TimeTo = blankToNil(in.TimeFrom), blankToNil(in.TimeTo)
	var tf, tt time.Time
	if in.TimeFrom != nil {
		if tf, err = time.Parse("15:04", *in.TimeFrom); err != nil {
			return invalid("time_from must be HH:MM")
		}
	}
	if in.TimeTo != nil {
		if tt, err = time.Parse("15:04", *in.TimeTo); err != nil {
			return invalid("time_to must be HH:MM")
		}
	}
	if in.DateEnd == nil && in.TimeFrom != nil && in.TimeTo != nil && !tt.After(tf) {
		return invalid("time_to must be after time_from")
	}
	in.Location = strings.TrimSpace(in.Location)
	in.Description = strings.TrimSpace(in.Description)
	if len([]rune(in.Location)) > 200 {
		return invalid("location: max 200 characters")
	}
	if len([]rune(in.Description)) > 2000 {
		return invalid("description: max 2000 characters")
	}
	in.HorseID = blankToNil(in.HorseID)
	if in.HorseID != nil && !validUUID(*in.HorseID) {
		return invalid("horse_id is not a valid id")
	}
	if needsHorse(in.Type) && in.HorseID == nil {
		return invalid("horse_id is required for type %s", in.Type)
	}

	payload, showTasks, err := validatePayload(in.Type, in.Payload)
	if err != nil {
		return invalid("%s", err.Error())
	}
	in.Payload = payload
	if in.Type == TypeShowHelper {
		in.Tasks = showTasks
	} else if in.Tasks, err = cleanTasks(in.Tasks); err != nil {
		return invalid("%s", err.Error())
	}

	if in.HelpersNeeded == 0 {
		in.HelpersNeeded = 1
	}
	if in.Type == TypeRideShare {
		// A trailer seat is one helper slot.
		in.HelpersNeeded = seatsFree(in.Payload)
		if in.TimeFrom == nil {
			var p RideSharePayload
			_ = json.Unmarshal(in.Payload, &p)
			in.TimeFrom = &p.DepartureTime
		}
	}
	if in.HelpersNeeded < 1 || in.HelpersNeeded > 20 {
		return invalid("helpers_needed must be between 1 and 20")
	}

	if in.RecurringRule != "" {
		rule, err := ParseRule(in.RecurringRule)
		if err != nil {
			return invalid("%s", err.Error())
		}
		if in.DateEnd != nil {
			return invalid("a recurring request must be a single day (no date_end)")
		}
		in.RecurringRule = rule.String()
	}
	return nil
}

// German labels used in push notifications and calendar entries.
var typeLabels = map[string]string{
	TypeBlanket:              "Decken",
	TypeShowHelper:           "Turniertrottel",
	TypeRideShare:            "Mitfahrgelegenheit",
	TypeExercise:             "Bewegen",
	TypeFeedOrTurnout:        "Füttern und Rausstellen",
	TypeAppointmentCompanion: "Terminbegleitung",
	TypeOther:                "Hilfe gesucht",
}

var taskLabels = map[string]string{
	"hold_horse": "Pferd halten", "warm_up": "Abreiten begleiten", "film": "Filmen",
	"fetch_number": "Startnummer holen", "load_trailer": "Hänger beladen",
}

var (
	weekdayShort = []string{"So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"}
	monthShort   = []string{"", "Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"}
)

// whenText formats a date and optional time like "Mi, 2. Okt · 18:00".
func whenText(date string, timeFrom *string) string {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	s := fmt.Sprintf("%s, %d. %s", weekdayShort[d.Weekday()], d.Day(), monthShort[d.Month()])
	if timeFrom != nil {
		s += " · " + *timeFrom
	}
	return s
}

// checklist returns the human readable task list (German labels for show tasks).
func (r Request) checklist() []string {
	out := make([]string, 0, len(r.Tasks))
	for _, t := range r.Tasks {
		if r.Type == TypeShowHelper {
			if l, ok := taskLabels[t]; ok {
				t = l
			}
		}
		out = append(out, t)
	}
	return out
}

// title is a short German headline such as "Turniertrottel: Luna".
func (r Request) title() string {
	t := typeLabels[r.Type]
	if t == "" {
		t = "Anfrage"
	}
	if r.Type == TypeShowHelper {
		var p ShowHelperPayload
		if json.Unmarshal(r.Payload, &p) == nil && p.ShowName != "" {
			t += " · " + p.ShowName
		}
	}
	if r.HorseName != nil {
		t += ": " + *r.HorseName
	}
	return t
}
