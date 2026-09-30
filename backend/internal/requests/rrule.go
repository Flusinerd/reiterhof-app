package requests

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Rule is the supported subset of RFC 5545 RRULE:
//
//	FREQ=DAILY[;UNTIL=YYYYMMDD]
//	FREQ=WEEKLY[;BYDAY=MO,WE][;UNTIL=YYYYMMDD]
//
// FREQ is required. BYDAY (two-letter weekdays MO TU WE TH FR SA SU) is only
// allowed with WEEKLY; without it the weekday of the first date is used. UNTIL is
// inclusive and may also be given as YYYYMMDDTHHMMSSZ (the date part counts).
// INTERVAL, COUNT and all other parts are rejected. The rule is anchored at the
// date of the first request and never produces earlier dates.
type Rule struct {
	Freq  string // DAILY | WEEKLY
	Days  []time.Weekday
	Until *time.Time // date at 00:00 UTC
}

var dayCodes = []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}

func weekdayFromCode(c string) (time.Weekday, bool) {
	for i, d := range dayCodes {
		if d == c {
			return time.Weekday(i), true
		}
	}
	return 0, false
}

// ParseRule parses the RRULE subset.
func ParseRule(s string) (Rule, error) {
	var r Rule
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "RRULE:"))
	if s == "" {
		return r, errors.New("recurring_rule is empty")
	}
	seen := map[string]bool{}
	for _, part := range strings.Split(s, ";") {
		k, v, ok := strings.Cut(part, "=")
		k = strings.ToUpper(strings.TrimSpace(k))
		v = strings.ToUpper(strings.TrimSpace(v))
		if !ok || v == "" || seen[k] {
			return r, fmt.Errorf("recurring_rule: invalid part %q", part)
		}
		seen[k] = true
		switch k {
		case "FREQ":
			if v != "DAILY" && v != "WEEKLY" {
				return r, errors.New("recurring_rule: FREQ must be DAILY or WEEKLY")
			}
			r.Freq = v
		case "BYDAY":
			for _, c := range strings.Split(v, ",") {
				wd, ok := weekdayFromCode(c)
				if !ok {
					return r, fmt.Errorf("recurring_rule: unknown weekday %q", c)
				}
				r.Days = append(r.Days, wd)
			}
		case "UNTIL":
			if len(v) != 8 && !(len(v) == 16 && v[8] == 'T' && v[15] == 'Z') {
				return r, errors.New("recurring_rule: UNTIL must be YYYYMMDD")
			}
			t, err := time.Parse("20060102", v[:8])
			if err != nil {
				return r, errors.New("recurring_rule: UNTIL must be YYYYMMDD")
			}
			r.Until = &t
		default:
			return r, fmt.Errorf("recurring_rule: %s is not supported", k)
		}
	}
	if r.Freq == "" {
		return r, errors.New("recurring_rule: FREQ is required")
	}
	if r.Freq == "DAILY" && len(r.Days) > 0 {
		return r, errors.New("recurring_rule: BYDAY needs FREQ=WEEKLY")
	}
	sort.Slice(r.Days, func(i, j int) bool { return weekOrder(r.Days[i]) < weekOrder(r.Days[j]) })
	out := r.Days[:0]
	for i, d := range r.Days {
		if i == 0 || d != r.Days[i-1] {
			out = append(out, d)
		}
	}
	r.Days = out
	return r, nil
}

// weekOrder sorts Monday first.
func weekOrder(d time.Weekday) int { return (int(d) + 6) % 7 }

// String is the canonical form, parseable by ParseRule.
func (r Rule) String() string {
	parts := []string{"FREQ=" + r.Freq}
	if len(r.Days) > 0 {
		codes := make([]string, len(r.Days))
		for i, d := range r.Days {
			codes[i] = dayCodes[d]
		}
		parts = append(parts, "BYDAY="+strings.Join(codes, ","))
	}
	if r.Until != nil {
		parts = append(parts, "UNTIL="+r.Until.Format("20060102"))
	}
	return strings.Join(parts, ";")
}

// Occurrences returns the dates (00:00 UTC) of the rule that lie in [from, to],
// anchored at start (the first occurrence). Dates before start are never
// returned. A weekly rule without BYDAY repeats on the weekday of start.
func (r Rule) Occurrences(start, from, to time.Time) []time.Time {
	start = dateOnly(start)
	from, to = dateOnly(from), dateOnly(to)
	if from.Before(start) {
		from = start
	}
	if r.Until != nil && to.After(*r.Until) {
		to = *r.Until
	}
	days := r.Days
	if r.Freq == "WEEKLY" && len(days) == 0 {
		days = []time.Weekday{start.Weekday()}
	}
	var out []time.Time
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if r.Freq == "DAILY" {
			out = append(out, d)
			continue
		}
		for _, wd := range days {
			if d.Weekday() == wd {
				out = append(out, d)
				break
			}
		}
	}
	return out
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
