package trainingapi

import (
	"errors"
	"fmt"
	"math"
)

const (
	// maxTrackPoints bounds sessions.track; the app simplifies the polyline before sending.
	maxTrackPoints = 5000
	// maxGaitWindows bounds the raw windows kept for model improvement (about 100 minutes at a
	// 2 s hop). Together with a full track the body stays below the 1 MiB request limit.
	maxGaitWindows = 3000
	// gaitFeatureCount is the length of gaitWindow.F (see mobile/lib/gait/export.ts FEATURE_ORDER).
	gaitFeatureCount = 6
)

// trackPoint is one point of the simplified GPS track. T is seconds since the session start,
// Gait the gait of the segment that starts at this point.
type trackPoint struct {
	Lat  float64  `json:"lat"`
	Lon  float64  `json:"lon"`
	T    float64  `json:"t"`
	Alt  *float64 `json:"alt,omitempty"`
	Gait string   `json:"g,omitempty"` // halt, walk, trot, canter
}

// gaitWindow mirrors the app's WindowRecord: T window start (ms since the session start),
// F the features, P the predicted gait, A the accelerometer-only gait, V the GPS speed in
// km/h and C the gait the user corrected (the training label).
type gaitWindow struct {
	T float64   `json:"t"`
	F []float64 `json:"f"`
	P string    `json:"p"`
	A string    `json:"a"`
	V *float64  `json:"v,omitempty"`
	C string    `json:"c,omitempty"`
}

func validGait(g string) bool {
	switch g {
	case "halt", "walk", "trot", "canter":
		return true
	}
	return false
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func validateTrack(track []trackPoint) error {
	if len(track) > maxTrackPoints {
		return fmt.Errorf("track has too many points (max %d)", maxTrackPoints)
	}
	lastT := 0.0
	for i, p := range track {
		if !finite(p.Lat) || !finite(p.Lon) || p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 {
			return errors.New("track point coordinates are out of range")
		}
		if !finite(p.T) || p.T < 0 || p.T > maxSessionMinutes*60 || (i > 0 && p.T < lastT) {
			return errors.New("track point times must be ascending seconds within the session")
		}
		lastT = p.T
		if p.Alt != nil && (!finite(*p.Alt) || *p.Alt < -500 || *p.Alt > 9000) {
			return errors.New("track point altitude is out of range")
		}
		if p.Gait != "" && !validGait(p.Gait) {
			return errors.New("track point gait must be halt, walk, trot or canter")
		}
	}
	return nil
}

func validateGaitWindows(ws []gaitWindow) error {
	if len(ws) > maxGaitWindows {
		return fmt.Errorf("too many gait windows (max %d)", maxGaitWindows)
	}
	for _, w := range ws {
		if !finite(w.T) || w.T < 0 {
			return errors.New("gait window start is out of range")
		}
		if len(w.F) != gaitFeatureCount {
			return fmt.Errorf("a gait window needs %d features", gaitFeatureCount)
		}
		for _, f := range w.F {
			if !finite(f) || math.Abs(f) > 1e6 {
				return errors.New("gait window feature is out of range")
			}
		}
		if !validGait(w.P) || !validGait(w.A) || (w.C != "" && !validGait(w.C)) {
			return errors.New("gait window gaits must be halt, walk, trot or canter")
		}
		if w.V != nil && (!finite(*w.V) || *w.V < 0 || *w.V > 100) {
			return errors.New("gait window speed is out of range")
		}
	}
	return nil
}
