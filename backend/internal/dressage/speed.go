package dressage

// Gaits and tempi.
const (
	GaitHalt   = "halt"
	GaitWalk   = "walk"
	GaitTrot   = "trot"
	GaitCanter = "canter"

	TempoWorking   = "working"
	TempoMedium    = "medium"
	TempoExtended  = "extended"
	TempoCollected = "collected"
	TempoFree      = "free"
)

// SpeedSource names where the speeds come from (docs/domains/dressage-tests.md, "Speeds and duration").
const SpeedSource = "Back & Clayton, Equine Locomotion, 2. Aufl. 2013, zitiert nach Equisense " +
	"(blog.equisense.com/de/takt-tempo-schrittlaenge/): Tempospannen von Hochleistungs-Dressurpferden"

// SpeedRange is the measured range in metres per minute of a gait for dressage horses of the
// source: the lowest value belongs to the collected, the highest to the extended tempo.
type SpeedRange struct {
	Min, Max float64
}

// SpeedRanges are the measured ranges per gait (m/min): walk 84–108, trot 192–296, canter 196–358.
var SpeedRanges = map[string]SpeedRange{
	GaitWalk:   {84, 108},
	GaitTrot:   {192, 296},
	GaitCanter: {196, 358},
}

// tempoFraction places a tempo inside the measured range of a gait (0 = collected end, 1 =
// extended end). The source gives the range only; the fractions are our own interpolation, they
// are marked as such in the docs.
var tempoFraction = map[string]float64{
	TempoCollected: 0,
	TempoWorking:   0.25,
	TempoFree:      0.4,
	TempoMedium:    0.6,
	TempoExtended:  1,
}

// Cadence is the stride frequency (strides per minute) of dressage horses in the same source,
// the basis for the choice of music: walk 52–56, trot 77–83, canter 99–105.
var Cadence = map[string][2]int{
	GaitWalk:   {52, 56},
	GaitTrot:   {77, 83},
	GaitCanter: {99, 105},
}

// Fixed times in seconds (the spec's own constants).
const (
	SaluteSeconds     = 6.0 // halt with salute
	HaltSeconds       = 4.0 // halt without salute
	TransitionSeconds = 1.0 // each change of gait or tempo
	ReinBackStepSec   = 1.0 // per step of rein back
	// SimpleChangeSeconds is the simple change of lead: two transitions and 3 to 5 walk steps
	// (about 4 at the cadence of the walk).
	SimpleChangeSeconds = 6.0
)

// Speed returns the speed in metres per second of a gait and tempo; a tempo that does not exist
// in the gait is placed in the same range. Halt has no speed.
func Speed(gait, tempo string) float64 {
	r, ok := SpeedRanges[gait]
	if !ok {
		return 0
	}
	f, ok := tempoFraction[tempo]
	if !ok {
		f = tempoFraction[TempoWorking]
	}
	if gait == GaitWalk && tempo == TempoWorking {
		f = tempoFraction[TempoMedium]
	}
	return (r.Min + (r.Max-r.Min)*f) / 60
}

// GaitPhrase is the German noun phrase after "im": "Arbeitstrab", "Mitteltrab", "freien Schritt am langen Zügel".
func GaitPhrase(gait, tempo string) string {
	switch gait {
	case GaitWalk:
		switch tempo {
		case TempoFree:
			return "freien Schritt am langen Zügel"
		case TempoCollected:
			return "versammelten Schritt"
		case TempoExtended:
			return "starken Schritt"
		}
		return "Mittelschritt"
	case GaitTrot:
		switch tempo {
		case TempoMedium:
			return "Mitteltrab"
		case TempoExtended:
			return "starken Trab"
		case TempoCollected:
			return "versammelten Trab"
		}
		return "Arbeitstrab"
	case GaitCanter:
		switch tempo {
		case TempoMedium:
			return "Mittelgalopp"
		case TempoExtended:
			return "starken Galopp"
		case TempoCollected:
			return "versammelten Galopp"
		}
		return "Arbeitsgalopp"
	}
	return ""
}

// GaitLabel is the German name of a gait.
func GaitLabel(gait string) string {
	switch gait {
	case GaitWalk:
		return "Schritt"
	case GaitTrot:
		return "Trab"
	case GaitCanter:
		return "Galopp"
	case GaitHalt:
		return "Halten"
	}
	return gait
}
