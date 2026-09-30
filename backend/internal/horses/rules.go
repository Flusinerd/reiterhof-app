package horses

import "fmt"

// Rider rules (positive list, JAN-10). A rider (RB) may only do what is listed in
// horse_riders.rules for that horse. The owner (and admins) always may everything.
// Other packages check them with auth.RiderRules and compare against these constants.
const (
	// RuleRide is the general permission to ride the horse (legacy value from the seed data).
	RuleRide = "ride"
	// RuleGroom is the permission to groom and care for the horse (legacy value from the seed data).
	RuleGroom = "groom"
	// RuleLogSessions lets the rider log training sessions for the horse.
	RuleLogSessions = "log_sessions"
	// RuleReportObservations lets the rider report observations (Auffälligkeiten).
	RuleReportObservations = "report_observations"
	// RuleTakeWeekSlots lets the rider take slots in the week plan.
	RuleTakeWeekSlots = "take_week_slots"
	// RuleHackAlone allows hacking out alone.
	RuleHackAlone = "hack_alone"
	// RuleShows allows riding the horse at shows.
	RuleShows = "shows"
)

// Rules returns all valid rider rules in their canonical order.
func Rules() []string {
	return []string{
		RuleRide, RuleGroom, RuleLogSessions, RuleReportObservations,
		RuleTakeWeekSlots, RuleHackAlone, RuleShows,
	}
}

// DefaultRiderRules are granted when a rider is added without an explicit list.
func DefaultRiderRules() []string {
	return []string{RuleLogSessions, RuleReportObservations, RuleTakeWeekSlots}
}

// ValidRule reports whether rule is in the positive list.
func ValidRule(rule string) bool {
	for _, r := range Rules() {
		if r == rule {
			return true
		}
	}
	return false
}

// NormalizeRules validates the rules, removes duplicates and returns them in
// canonical order. An empty list is valid (the rider may do nothing special).
func NormalizeRules(in []string) ([]string, error) {
	seen := make(map[string]bool, len(in))
	for _, r := range in {
		if !ValidRule(r) {
			return nil, fmt.Errorf("unknown rule %q", r)
		}
		seen[r] = true
	}
	out := make([]string, 0, len(seen))
	for _, r := range Rules() {
		if seen[r] {
			out = append(out, r)
		}
	}
	return out, nil
}
