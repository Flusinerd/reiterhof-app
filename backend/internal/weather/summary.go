package weather

import (
	"errors"
	"math"
	"time"
)

// Rain decision: it counts as raining tonight when the maximum precipitation
// probability in the night window reaches RainProbabilityThreshold percent or
// the summed precipitation reaches RainSumThresholdMM.
const (
	RainProbabilityThreshold = 50.0
	RainSumThresholdMM       = 0.5
)

// Night window in stable-local wall-clock time: NightStartHour on the given day
// to NightEndHour on the next day.
const (
	NightStartHour = 18
	NightEndHour   = 8
)

// ErrNoData is returned when the forecast has no temperature in the window.
var ErrNoData = errors.New("weather: no forecast data for the night window")

// Summary is the night summary for one stable-local day (the night that starts
// on Day at 18:00).
type Summary struct {
	Day             string    `json:"day"` // stable-local date, YYYY-MM-DD
	WindowStart     time.Time `json:"window_start"`
	WindowEnd       time.Time `json:"window_end"`
	NightMinC       float64   `json:"night_min_c"`
	RainProbability int       `json:"rain_probability"` // max over the window, percent
	RainMM          float64   `json:"rain_mm"`          // sum over the window
	WindKmh         float64   `json:"wind_kmh"`         // max over the window
	WillRain        bool      `json:"will_rain"`
	Hours           int       `json:"hours"` // temperature steps that went into the summary
}

// Summarize reduces the hourly forecast to the night that starts on day
// (only the calendar date of day in loc's view is used, via day.In(loc)).
//
// The window is 18:00 local on that day until 08:00 local on the next day,
// computed as wall-clock times so the DST changeover nights are 15 hours (last
// Sunday of October) or 13 hours (last Sunday of March) long. Temperature and
// wind use steps in [start, end]; precipitation values describe the hour that
// ends at the step, so they use steps in (start, end].
func Summarize(hours []Hour, loc *time.Location, day time.Time) (Summary, error) {
	y, m, d := day.In(loc).Date()
	start := time.Date(y, m, d, NightStartHour, 0, 0, 0, loc)
	end := time.Date(y, m, d+1, NightEndHour, 0, 0, 0, loc)

	s := Summary{Day: start.Format("2006-01-02"), WindowStart: start, WindowEnd: end}
	minC := math.Inf(1)
	var maxProb, maxWind, sumMM float64
	for _, h := range hours {
		if h.Time.Before(start) || h.Time.After(end) {
			continue
		}
		if h.TempC != nil {
			minC = math.Min(minC, *h.TempC)
			s.Hours++
		}
		if h.WindKmh != nil {
			maxWind = math.Max(maxWind, *h.WindKmh)
		}
		if h.Time.Equal(start) {
			continue // precipitation of the hour before the window
		}
		if h.RainProb != nil {
			maxProb = math.Max(maxProb, *h.RainProb)
		}
		if h.RainMM != nil {
			sumMM += *h.RainMM
		}
	}
	if s.Hours == 0 {
		return Summary{}, ErrNoData
	}
	s.NightMinC = round1(minC)
	s.RainProbability = int(math.Round(maxProb))
	s.RainMM = round1(sumMM)
	s.WindKmh = round1(maxWind)
	s.WillRain = maxProb >= RainProbabilityThreshold || sumMM >= RainSumThresholdMM
	return s, nil
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
