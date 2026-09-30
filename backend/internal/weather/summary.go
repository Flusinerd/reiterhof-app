package weather

import (
	"errors"
	"math"
	"time"
)

// Rain decision: it counts as raining while the horse is covered when the
// maximum precipitation probability in the window reaches
// RainProbabilityThreshold percent or the summed precipitation reaches
// RainSumThresholdMM.
const (
	RainProbabilityThreshold = 50.0
	RainSumThresholdMM       = 0.5
)

// RainHourThresholdMM is the precipitation in one hour from which that hour
// counts as a rainy hour (rain_hours, rain_from, rain_until).
const RainHourThresholdMM = 0.1

// Window is the covered window in stable-local wall-clock minutes since midnight: Start on
// the given day to End on the next day. The horses are covered in the evening and uncovered
// in the morning when they come in from the paddock. Every horse has its own window
// (horses.cover_start, horses.cover_end); DefaultWindow is what the stored snapshot uses.
type Window struct {
	StartMinutes int
	EndMinutes   int
}

// DefaultWindow is 18:00 to 12:30, the default of a horse.
var DefaultWindow = Window{StartMinutes: 18 * 60, EndMinutes: 12*60 + 30}

// ErrNoData is returned when the forecast has no temperature in the window.
var ErrNoData = errors.New("weather: no forecast data for the covered window")

// HourPoint is one hourly step of the forecast inside the window (the timeline
// of the app). RainMM and RainProb describe the hour that ends at Time; the
// first point (start of the window) has none.
type HourPoint struct {
	Time     time.Time `json:"time"`
	TempC    float64   `json:"temp_c"`
	RainMM   float64   `json:"rain_mm"`
	RainProb int       `json:"rain_prob"`
	WindKmh  float64   `json:"wind_kmh"`
}

// Summary is the forecast summary for one stable-local day (the window that
// starts on Day and ends the next day, see Window). Snapshots written
// before the detail fields existed lack them (zero values).
type Summary struct {
	Day             string      `json:"day"` // stable-local date, YYYY-MM-DD
	WindowStart     time.Time   `json:"window_start"`
	WindowEnd       time.Time   `json:"window_end"`
	NightMinC       float64     `json:"night_min_c"`      // lowest temperature in the window
	TempMaxC        float64     `json:"temp_max_c"`       // highest temperature in the window
	RainProbability int         `json:"rain_probability"` // max over the window, percent
	RainMM          float64     `json:"rain_mm"`          // sum over the window
	RainPeakMM      float64     `json:"rain_peak_mm"`     // most rain in one hour
	RainHours       int         `json:"rain_hours"`       // hours with at least RainHourThresholdMM
	RainFrom        time.Time   `json:"rain_from"`        // start of the first rainy hour, zero when dry
	RainUntil       time.Time   `json:"rain_until"`       // end of the last rainy hour, zero when dry
	WindKmh         float64     `json:"wind_kmh"`         // max over the window
	WillRain        bool        `json:"will_rain"`
	Hours           int         `json:"hours"` // temperature steps that went into the summary
	Timeline        []HourPoint `json:"timeline,omitempty"`
	// Forecast is the raw hourly forecast from ForecastFrom to ForecastTo, so the summary can be
	// computed again for the window of a single horse. Absent in older snapshots.
	Forecast []Hour `json:"forecast,omitempty"`
}

// Every window of a horse starts in [ForecastFrom, ...) and ends before ForecastTo (stable-local
// hours of the day and of the next day), see the limits of the cover window in the API.
const (
	ForecastFromHour = 15
	ForecastToHour   = 15
)

// Span returns the hourly steps that any allowed cover window of the day can need: from
// ForecastFromHour on the day until ForecastToHour on the next day (both inclusive).
func Span(hours []Hour, loc *time.Location, day time.Time) []Hour {
	y, m, d := day.In(loc).Date()
	from := time.Date(y, m, d, ForecastFromHour, 0, 0, 0, loc)
	to := time.Date(y, m, d+1, ForecastToHour, 0, 0, 0, loc)
	var out []Hour
	for _, h := range hours {
		if !h.Time.Before(from) && !h.Time.After(to) {
			out = append(out, h)
		}
	}
	return out
}

// Summarize reduces the hourly forecast to the window win that starts on day
// (only the calendar date of day in loc's view is used, via day.In(loc)).
//
// The window runs from win.Start local on that day until win.End local on the next day,
// computed as wall-clock times so the DST changeover nights are one hour longer
// or shorter. Temperature and wind use steps in [start, end]. Precipitation
// values describe the hour that ends at the step, so an hour counts with the
// share that lies inside the window (with the default window the hour 12:00 to 13:00 counts half).
func Summarize(hours []Hour, loc *time.Location, day time.Time, win Window) (Summary, error) {
	y, m, d := day.In(loc).Date()
	start := time.Date(y, m, d, 0, win.StartMinutes, 0, 0, loc)
	end := time.Date(y, m, d+1, 0, win.EndMinutes, 0, 0, loc)

	s := Summary{Day: start.Format("2006-01-02"), WindowStart: start, WindowEnd: end}
	minC, maxC := math.Inf(1), math.Inf(-1)
	var maxProb, maxWind, sumMM, peakMM float64
	for _, h := range hours {
		if !h.Time.Before(start) && !h.Time.After(end) {
			pt := HourPoint{Time: h.Time}
			if h.TempC != nil {
				minC = math.Min(minC, *h.TempC)
				maxC = math.Max(maxC, *h.TempC)
				pt.TempC = round1(*h.TempC)
				s.Hours++
			}
			if h.WindKmh != nil {
				maxWind = math.Max(maxWind, *h.WindKmh)
				pt.WindKmh = round1(*h.WindKmh)
			}
			if !h.Time.Equal(start) {
				if h.RainMM != nil {
					pt.RainMM = round1(*h.RainMM)
				}
				if h.RainProb != nil {
					pt.RainProb = int(math.Round(*h.RainProb))
				}
			}
			if h.TempC != nil {
				s.Timeline = append(s.Timeline, pt)
			}
		}
		share := overlap(h.Time, start, end)
		if share == 0 {
			continue
		}
		if h.RainProb != nil {
			maxProb = math.Max(maxProb, *h.RainProb)
		}
		if h.RainMM != nil {
			mm := *h.RainMM * share
			sumMM += mm
			peakMM = math.Max(peakMM, *h.RainMM)
			if mm >= RainHourThresholdMM {
				s.RainHours++
				from, until := h.Time.Add(-time.Hour), h.Time
				if from.Before(start) {
					from = start
				}
				if until.After(end) {
					until = end
				}
				if s.RainFrom.IsZero() {
					s.RainFrom = from
				}
				s.RainUntil = until
			}
		}
	}
	if s.Hours == 0 {
		return Summary{}, ErrNoData
	}
	s.NightMinC = round1(minC)
	s.TempMaxC = round1(maxC)
	s.RainProbability = int(math.Round(maxProb))
	s.RainMM = round1(sumMM)
	s.RainPeakMM = round1(peakMM)
	s.WindKmh = round1(maxWind)
	s.WillRain = maxProb >= RainProbabilityThreshold || sumMM >= RainSumThresholdMM
	return s, nil
}

// overlap is the share (0..1) of the hour that ends at t which lies in [start, end].
func overlap(t, start, end time.Time) float64 {
	lo, hi := t.Add(-time.Hour), t
	if lo.Before(start) {
		lo = start
	}
	if hi.After(end) {
		hi = end
	}
	if !hi.After(lo) {
		return 0
	}
	return hi.Sub(lo).Hours()
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
