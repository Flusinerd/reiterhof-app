package requests

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Request types. The set matches the CHECK constraint on requests.type.
const (
	TypeBlanket              = "blanket"
	TypeShowHelper           = "show_helper"
	TypeRideShare            = "ride_share"
	TypeExercise             = "exercise"
	TypeFeedOrTurnout        = "feed_or_turnout"
	TypeAppointmentCompanion = "appointment_companion"
	TypeOther                = "other"
)

// Types lists all request types.
func Types() []string {
	return []string{TypeBlanket, TypeShowHelper, TypeRideShare, TypeExercise, TypeFeedOrTurnout, TypeAppointmentCompanion, TypeOther}
}

// ValidType reports whether t is a known request type.
func ValidType(t string) bool {
	for _, k := range Types() {
		if k == t {
			return true
		}
	}
	return false
}

// needsHorse reports whether a request of this type must reference a horse.
func needsHorse(t string) bool {
	switch t {
	case TypeExercise, TypeFeedOrTurnout, TypeAppointmentCompanion:
		return true
	}
	return false
}

// Show helper tasks.
var showTasks = map[string]bool{
	"hold_horse": true, "warm_up": true, "film": true, "fetch_number": true, "load_trailer": true,
}

// ShowHelperPayload is the payload of type show_helper ("Turniertrottel").
type ShowHelperPayload struct {
	ShowName  string      `json:"show_name"`
	Classes   []ShowClass `json:"classes"`
	Tasks     []string    `json:"tasks"`
	RideAlong bool        `json:"ride_along"`
}

// ShowClass is one class (Prüfung) the rider starts in.
type ShowClass struct {
	Name string `json:"name"`
	Time string `json:"time,omitempty"` // HH:MM
}

// RideSharePayload is the payload of type ride_share (trailer seat).
type RideSharePayload struct {
	Destination   string `json:"destination"`
	DepartureTime string `json:"departure_time"` // HH:MM
	SeatsFree     int    `json:"seats_free"`
}

// ExercisePayload is the payload of type exercise ("Bewegen").
type ExercisePayload struct {
	Mode      string `json:"mode"` // lunge | ride
	RulesNote string `json:"rules_note,omitempty"`
}

// FeedOrTurnoutPayload is the payload of type feed_or_turnout.
type FeedOrTurnoutPayload struct {
	What string `json:"what"` // feed | turnout | bring_in
}

// AppointmentPayload is the payload of type appointment_companion.
type AppointmentPayload struct {
	With string `json:"with"` // farrier | vet | other
	Note string `json:"note,omitempty"`
}

// validatePayload checks the type-specific payload and returns its canonical
// JSON form (defaults filled in, nil slices replaced by empty ones). For
// show_helper the canonical tasks are also returned; nil for other types.
func validatePayload(typ string, raw json.RawMessage) (canonical json.RawMessage, tasks []string, err error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage(`{}`)
	}
	strict := func(v any) error {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(v); err != nil {
			return fmt.Errorf("payload: %w", err)
		}
		if dec.More() {
			return errors.New("payload: unexpected data")
		}
		return nil
	}
	var out any
	switch typ {
	case TypeBlanket:
		// Owned by the blanket feature: any JSON object up to the body limit.
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
			return nil, nil, errors.New("payload: must be a JSON object")
		}
		return raw, nil, nil
	case TypeOther:
		var obj map[string]any
		if err := strict(&obj); err != nil {
			return nil, nil, err
		}
		if len(obj) > 0 {
			return nil, nil, errors.New("payload: must be empty for type other")
		}
		return json.RawMessage(`{}`), nil, nil
	case TypeShowHelper:
		var p ShowHelperPayload
		if err := strict(&p); err != nil {
			return nil, nil, err
		}
		p.ShowName = strings.TrimSpace(p.ShowName)
		if p.ShowName == "" || len([]rune(p.ShowName)) > 120 {
			return nil, nil, errors.New("payload.show_name is required (max 120 characters)")
		}
		if len(p.Classes) > 20 {
			return nil, nil, errors.New("payload.classes: at most 20 classes")
		}
		if p.Classes == nil {
			p.Classes = []ShowClass{}
		}
		for i := range p.Classes {
			c := &p.Classes[i]
			c.Name = strings.TrimSpace(c.Name)
			if c.Name == "" || len([]rune(c.Name)) > 80 {
				return nil, nil, fmt.Errorf("payload.classes[%d].name is required (max 80 characters)", i)
			}
			if c.Time != "" {
				if _, err := time.Parse("15:04", c.Time); err != nil {
					return nil, nil, fmt.Errorf("payload.classes[%d].time must be HH:MM", i)
				}
			}
		}
		seen := map[string]bool{}
		clean := []string{}
		for _, t := range p.Tasks {
			if !showTasks[t] {
				return nil, nil, fmt.Errorf("payload.tasks: unknown task %q (allowed: hold_horse, warm_up, film, fetch_number, load_trailer)", t)
			}
			if !seen[t] {
				seen[t] = true
				clean = append(clean, t)
			}
		}
		p.Tasks = clean
		tasks = clean
		out = p
	case TypeRideShare:
		var p RideSharePayload
		if err := strict(&p); err != nil {
			return nil, nil, err
		}
		p.Destination = strings.TrimSpace(p.Destination)
		if p.Destination == "" || len([]rune(p.Destination)) > 200 {
			return nil, nil, errors.New("payload.destination is required (max 200 characters)")
		}
		if _, err := time.Parse("15:04", p.DepartureTime); err != nil {
			return nil, nil, errors.New("payload.departure_time must be HH:MM")
		}
		if p.SeatsFree < 1 || p.SeatsFree > 8 {
			return nil, nil, errors.New("payload.seats_free must be between 1 and 8")
		}
		out = p
	case TypeExercise:
		var p ExercisePayload
		if err := strict(&p); err != nil {
			return nil, nil, err
		}
		if p.Mode != "lunge" && p.Mode != "ride" {
			return nil, nil, errors.New("payload.mode must be lunge or ride")
		}
		p.RulesNote = strings.TrimSpace(p.RulesNote)
		if len([]rune(p.RulesNote)) > 500 {
			return nil, nil, errors.New("payload.rules_note: max 500 characters")
		}
		out = p
	case TypeFeedOrTurnout:
		var p FeedOrTurnoutPayload
		if err := strict(&p); err != nil {
			return nil, nil, err
		}
		if p.What != "feed" && p.What != "turnout" && p.What != "bring_in" {
			return nil, nil, errors.New("payload.what must be feed, turnout or bring_in")
		}
		out = p
	case TypeAppointmentCompanion:
		var p AppointmentPayload
		if err := strict(&p); err != nil {
			return nil, nil, err
		}
		if p.With != "farrier" && p.With != "vet" && p.With != "other" {
			return nil, nil, errors.New("payload.with must be farrier, vet or other")
		}
		p.Note = strings.TrimSpace(p.Note)
		if len([]rune(p.Note)) > 500 {
			return nil, nil, errors.New("payload.note: max 500 characters")
		}
		out = p
	default:
		return nil, nil, fmt.Errorf("unknown type %q", typ)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, nil, err
	}
	return b, tasks, nil
}

// seatsFree returns payload.seats_free of a ride_share payload (0 if absent).
func seatsFree(raw json.RawMessage) int {
	var p RideSharePayload
	_ = json.Unmarshal(raw, &p)
	return p.SeatsFree
}

// cleanTasks trims and validates a free-text checklist.
func cleanTasks(in []string) ([]string, error) {
	if len(in) > 20 {
		return nil, errors.New("tasks: at most 20 entries")
	}
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len([]rune(t)) > 80 {
			return nil, errors.New("tasks: entries may have at most 80 characters")
		}
		out = append(out, t)
	}
	return out, nil
}
