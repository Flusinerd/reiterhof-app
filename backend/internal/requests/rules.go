package requests

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/reha"
)

// generatedRulePrefixes start the lines of rules_note that the server writes itself
// (see reha.Allowed.RuleText and reha.NoUnitText). fillRules replaces them on every save.
var generatedRulePrefixes = []string{"Reha:", "Reha-Plan aktiv"}

// fillRules adds the horse's rules to the payload of an exercise request (JAN-42): while the
// horse has an active reha plan, the first line of rules_note is the plan's rule for the
// request's date, e.g. "Reha: Schritt führen 14 min, nur Boden fest". Text the requester typed
// stays below it. A line the server wrote earlier is replaced, so editing a request (or moving
// its date) refreshes the rule; without an active plan such a line is removed.
//
// A recurring series copies the payload of its template, so its later occurrences keep the rule
// of the template's day.
func (s *Service) fillRules(ctx context.Context, q querier, stableID string, in *Input) error {
	if in.Type != TypeExercise || in.HorseID == nil {
		return nil
	}
	day, err := time.Parse("2006-01-02", in.Date)
	if err != nil {
		return nil
	}
	rule, err := reha.RuleFor(ctx, q, stableID, *in.HorseID, day)
	if err != nil {
		return internal(err)
	}
	var p ExercisePayload
	if err := json.Unmarshal(in.Payload, &p); err != nil {
		return nil
	}
	p.RulesNote = mergeRulesNote(p.RulesNote, rule)
	b, err := json.Marshal(p)
	if err != nil {
		return internal(err)
	}
	in.Payload = b
	return nil
}

// mergeRulesNote puts rule on the first line and keeps the requester's own lines.
func mergeRulesNote(note, rule string) string {
	var own []string
	for _, line := range strings.Split(note, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || isGeneratedRule(line) {
			continue
		}
		own = append(own, line)
	}
	if rule != "" {
		own = append([]string{rule}, own...)
	}
	return strings.Join(own, "\n")
}

func isGeneratedRule(line string) bool {
	for _, p := range generatedRulePrefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}
