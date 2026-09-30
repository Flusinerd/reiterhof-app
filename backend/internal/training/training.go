// Package training holds the pure (no HTTP, no DB, no clock) domain model of the
// training feature: activities, profile, rider rules, sessions. The load score
// lives in training/load, the "Was heute?" recommender in training/recommend.
//
// All functions are deterministic: "today" is always passed in, never read from
// the clock. Day arithmetic uses the calendar fields (year, month, day) of the
// time.Time values as given, so callers should pass times in the stable's
// timezone.
package training

import "time"

// Activity is a kind of training session. Values match the API/DB strings.
type Activity string

const (
	ActivityHall       Activity = "hall"
	ActivityArena      Activity = "arena"
	ActivityHack       Activity = "hack"
	ActivityLunge      Activity = "lunge"
	ActivityJumping    Activity = "jumping"
	ActivityGroundwork Activity = "groundwork"
	ActivityWalker     Activity = "walker"
	// ActivityRest is only produced by the recommender ("rest day").
	ActivityRest Activity = "rest"
)

// AllActivities is the canonical order. It is also the deterministic tie-break
// order of the recommender.
var AllActivities = []Activity{
	ActivityHall, ActivityArena, ActivityHack, ActivityLunge,
	ActivityJumping, ActivityGroundwork, ActivityWalker,
}

var germanNames = map[Activity]string{
	ActivityHall:       "Halle",
	ActivityArena:      "Platz",
	ActivityHack:       "Ausritt",
	ActivityLunge:      "Longe",
	ActivityJumping:    "Springen",
	ActivityGroundwork: "Bodenarbeit",
	ActivityWalker:     "Führanlage",
	ActivityRest:       "Ruhetag",
}

// GermanName returns the UI label of the activity.
func (a Activity) GermanName() string {
	if n, ok := germanNames[a]; ok {
		return n
	}
	return string(a)
}

// Valid reports whether a is one of the seven trainable activities.
func (a Activity) Valid() bool {
	_, ok := germanNames[a]
	return ok && a != ActivityRest
}

// Intensity is the coarse effort level of an activity or a day.
type Intensity int

const (
	IntensityNone Intensity = iota // rest / no load
	IntensityLight
	IntensityMedium
	IntensityIntense
)

// Label returns the German UI label.
func (i Intensity) Label() string {
	switch i {
	case IntensityLight:
		return "leicht"
	case IntensityMedium:
		return "mittel"
	case IntensityIntense:
		return "intensiv"
	}
	return "keine"
}

// Mode says whether an activity is allowed in the training profile.
type Mode string

const (
	ModeOn          Mode = "on"
	ModeOff         Mode = "off"
	ModeConditional Mode = "conditional"
)

// AllowedActivity is one entry of the profile's allowed_activities.
// Activities missing from the list are treated as off.
type AllowedActivity struct {
	Activity Activity `json:"activity"`
	Mode     Mode     `json:"mode"`
	Note     string   `json:"note,omitempty"` // condition, shown for ModeConditional
}

// Status is the horse's training status.
type Status string

const (
	StatusFit   Status = "fit"
	StatusReha  Status = "reha"
	StatusPause Status = "pause"
)

// Show is a competition date.
type Show struct {
	Date time.Time `json:"date"`
	Name string    `json:"name"`
}

// Rhythm is the desired weekly rhythm.
type Rhythm struct {
	SessionsMin int `json:"sessions_min"` // e.g. 4
	SessionsMax int `json:"sessions_max"` // e.g. 5
	RestDaysMin int `json:"rest_days_min"`
	RestDaysMax int `json:"rest_days_max"`
	MaxMinutes  int `json:"max_minutes"` // per session, 0 = no limit
	// RestAfterShow makes the day after a show a mandatory rest day.
	// DefaultRhythm sets it to true; the zero value disables the rule, so
	// decoders must start from DefaultRhythm.
	RestAfterShow bool `json:"rest_after_show"`
}

// DefaultRhythm is the rhythm from the spec: 4-5 sessions, 1-2 rest days,
// max 60 minutes, mandatory rest day after a show.
func DefaultRhythm() Rhythm {
	return Rhythm{SessionsMin: 4, SessionsMax: 5, RestDaysMin: 1, RestDaysMax: 2, MaxMinutes: 60, RestAfterShow: true}
}

// Profile is the owner-only training profile of a horse.
type Profile struct {
	HorseName  string            `json:"horse_name"`
	Discipline string            `json:"discipline"` // dressage, jumping, eventing, leisure, western, young_horse
	Level      string            `json:"level"`
	Allowed    []AllowedActivity `json:"allowed_activities"`
	Shows      []Show            `json:"shows"` // past and future
	SeasonEnd  *time.Time        `json:"season_end,omitempty"`
	Rhythm     Rhythm            `json:"rhythm"`
	Status     Status            `json:"status"`
}

// RiderRules are the owner's rules for one rider (rb_rules).
type RiderRules struct {
	// AllowedActivities is an explicit allow-list; nil or empty allows nothing.
	AllowedActivities []Activity `json:"allowed_activities"`
	// MaxIntensity is the highest intensity the rider may do; IntensityNone
	// (zero value) means no limit.
	MaxIntensity Intensity `json:"max_intensity"`
	MayHackAlone bool      `json:"may_hack_alone"`
	// MayRideShows is stored for completeness; the recommender does not use it.
	MayRideShows bool `json:"may_ride_shows"`
}

// Session is a finished training session as far as the logic cares.
type Session struct {
	Day         time.Time `json:"day"`
	Activity    Activity  `json:"activity"`
	Tags        []string  `json:"tags,omitempty"` // focus/goal tags
	Minutes     int       `json:"minutes"`
	CanterShare float64   `json:"canter_share"`
	Load        float64   `json:"load"` // see load.Score
	// Feel is how the horse felt (fresh, loose, tired, tense), empty if not logged.
	Feel string `json:"feel,omitempty"`
}

// Day normalises t to midnight UTC of its own calendar date.
func Day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// DaysBetween returns the number of calendar days from a to b (b - a).
func DaysBetween(a, b time.Time) int {
	return int(Day(b).Sub(Day(a)) / (24 * time.Hour))
}

// DefaultIntensity is the static intensity level of an activity, used for
// show/pause/recovery decisions and the rider's max intensity.
func DefaultIntensity(a Activity) Intensity {
	switch a {
	case ActivityWalker, ActivityGroundwork, ActivityLunge:
		return IntensityLight
	case ActivityHack, ActivityHall, ActivityArena:
		return IntensityMedium
	case ActivityJumping:
		return IntensityIntense
	}
	return IntensityNone
}
