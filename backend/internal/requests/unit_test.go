package requests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func date(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func formatDates(ds []time.Time) string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.Format("2006-01-02")
	}
	return strings.Join(out, " ")
}

func TestParseRule(t *testing.T) {
	good := map[string]string{
		"FREQ=DAILY":                          "FREQ=DAILY",
		"freq=weekly;byday=we":                "FREQ=WEEKLY;BYDAY=WE",
		"RRULE:FREQ=WEEKLY;BYDAY=FR,MO,FR":    "FREQ=WEEKLY;BYDAY=MO,FR",
		"FREQ=WEEKLY":                         "FREQ=WEEKLY",
		"FREQ=DAILY;UNTIL=20261231":           "FREQ=DAILY;UNTIL=20261231",
		"FREQ=WEEKLY;UNTIL=20261231T000000Z":  "FREQ=WEEKLY;UNTIL=20261231",
		"FREQ=WEEKLY;UNTIL=20261231;BYDAY=SU": "FREQ=WEEKLY;BYDAY=SU;UNTIL=20261231",
	}
	for in, want := range good {
		r, err := ParseRule(in)
		if err != nil {
			t.Errorf("ParseRule(%q): %v", in, err)
			continue
		}
		if got := r.String(); got != want {
			t.Errorf("ParseRule(%q).String() = %q, want %q", in, got, want)
		}
		if _, err := ParseRule(r.String()); err != nil {
			t.Errorf("canonical form %q does not parse: %v", r.String(), err)
		}
	}
	bad := []string{
		"", "BYDAY=MO", "FREQ=MONTHLY", "FREQ=DAILY;INTERVAL=2", "FREQ=DAILY;COUNT=3", "FREQ=DAILY;BYDAY=MO",
		"FREQ=WEEKLY;BYDAY=XX", "FREQ=WEEKLY;BYDAY=", "FREQ=DAILY;FREQ=DAILY", "FREQ=DAILY;UNTIL=2026", "FREQ=DAILY;UNTIL=20261301",
		"FREQ", "FREQ=YEARLY",
	}
	for _, in := range bad {
		if _, err := ParseRule(in); err == nil {
			t.Errorf("ParseRule(%q) accepted", in)
		}
	}
}

func TestRuleOccurrences(t *testing.T) {
	weekly, _ := ParseRule("FREQ=WEEKLY;BYDAY=WE")
	// 2026-09-30 is a Wednesday.
	if got, want := formatDates(weekly.Occurrences(date("2026-09-30"), date("2026-09-30"), date("2026-10-14"))),
		"2026-09-30 2026-10-07 2026-10-14"; got != want {
		t.Errorf("weekly = %s, want %s", got, want)
	}
	// Never before the anchor.
	if got := formatDates(weekly.Occurrences(date("2026-10-07"), date("2026-09-01"), date("2026-10-14"))); got != "2026-10-07 2026-10-14" {
		t.Errorf("anchor = %s", got)
	}
	// Weekly without BYDAY repeats on the weekday of the start date.
	plain, _ := ParseRule("FREQ=WEEKLY")
	if got := formatDates(plain.Occurrences(date("2026-10-02"), date("2026-10-01"), date("2026-10-20"))); got != "2026-10-02 2026-10-09 2026-10-16" {
		t.Errorf("plain weekly = %s", got)
	}
	multi, _ := ParseRule("FREQ=WEEKLY;BYDAY=MO,FR")
	if got := formatDates(multi.Occurrences(date("2026-10-02"), date("2026-10-02"), date("2026-10-13"))); got != "2026-10-02 2026-10-05 2026-10-09 2026-10-12" {
		t.Errorf("multi = %s", got)
	}
	daily, _ := ParseRule("FREQ=DAILY;UNTIL=20261003")
	if got := formatDates(daily.Occurrences(date("2026-10-01"), date("2026-09-25"), date("2026-10-10"))); got != "2026-10-01 2026-10-02 2026-10-03" {
		t.Errorf("daily until = %s", got)
	}
	if got := daily.Occurrences(date("2026-10-01"), date("2026-10-04"), date("2026-10-10")); len(got) != 0 {
		t.Errorf("after UNTIL = %v", got)
	}
}

func TestValidatePayload(t *testing.T) {
	ok := []struct{ typ, payload string }{
		{TypeShowHelper, `{"show_name":"Turnier Dorsten","classes":[{"name":"E-Dressur","time":"09:30"}],"tasks":["hold_horse","film"]}`},
		{TypeShowHelper, `{"show_name":"Turnier"}`},
		{TypeRideShare, `{"destination":"Haltern","departure_time":"07:15","seats_free":2}`},
		{TypeExercise, `{"mode":"lunge","rules_note":"nur Schritt und Trab"}`},
		{TypeExercise, `{"mode":"ride"}`},
		{TypeFeedOrTurnout, `{"what":"turnout"}`},
		{TypeAppointmentCompanion, `{"with":"vet","note":"Impfung"}`},
		{TypeOther, `{}`},
		{TypeOther, ``},
		{TypeBlanket, `{"anything":1}`},
	}
	for _, c := range ok {
		if _, _, err := validatePayload(c.typ, json.RawMessage(c.payload)); err != nil {
			t.Errorf("%s %s: %v", c.typ, c.payload, err)
		}
	}
	bad := []struct{ typ, payload string }{
		{TypeShowHelper, `{}`},
		{TypeShowHelper, `{"show_name":"x","tasks":["dance"]}`},
		{TypeShowHelper, `{"show_name":"x","classes":[{"name":""}]}`},
		{TypeShowHelper, `{"show_name":"x","classes":[{"name":"E","time":"9h"}]}`},
		{TypeShowHelper, `{"show_name":"x","unknown":1}`},
		{TypeRideShare, `{"destination":"Haltern","departure_time":"07:15","seats_free":0}`},
		{TypeRideShare, `{"destination":"Haltern","departure_time":"07:15","seats_free":9}`},
		{TypeRideShare, `{"destination":"","departure_time":"07:15","seats_free":1}`},
		{TypeRideShare, `{"destination":"H","departure_time":"quarter past","seats_free":1}`},
		{TypeExercise, `{"mode":"walk"}`},
		{TypeExercise, `{}`},
		{TypeFeedOrTurnout, `{"what":"sleep"}`},
		{TypeAppointmentCompanion, `{"with":"dentist"}`},
		{TypeOther, `{"x":1}`},
		{TypeBlanket, `[]`},
		{"nope", `{}`},
	}
	for _, c := range bad {
		if _, _, err := validatePayload(c.typ, json.RawMessage(c.payload)); err == nil {
			t.Errorf("%s %s accepted", c.typ, c.payload)
		}
	}
	canon, tasks, err := validatePayload(TypeShowHelper, json.RawMessage(`{"show_name":" Cup ","tasks":["film","film","warm_up"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 || tasks[0] != "film" || tasks[1] != "warm_up" {
		t.Errorf("tasks = %v", tasks)
	}
	if !strings.Contains(string(canon), `"show_name":"Cup"`) || !strings.Contains(string(canon), `"classes":[]`) {
		t.Errorf("canonical = %s", canon)
	}
}

func TestEscapeAndFoldICS(t *testing.T) {
	if got, want := escapeICS("a,b;c\\d\ne\r\nf"), `a\,b\;c\\d\ne\nf`; got != want {
		t.Errorf("escape = %q, want %q", got, want)
	}
	// Folding: physical lines are at most 75 octets, unfolding restores the input,
	// and multi-byte characters are never split.
	long := "DESCRIPTION:" + strings.Repeat("Füße und Hufe ", 20)
	folded := foldICS(long)
	if !strings.HasSuffix(folded, "\r\n") {
		t.Fatal("no CRLF")
	}
	lines := strings.Split(strings.TrimSuffix(folded, "\r\n"), "\r\n")
	if len(lines) < 3 {
		t.Fatalf("expected several lines, got %d", len(lines))
	}
	for i, l := range lines {
		if len(l) > 75 {
			t.Errorf("line %d has %d octets", i, len(l))
		}
		if !utf8.ValidString(l) {
			t.Errorf("line %d is not valid UTF-8", i)
		}
		if i > 0 && !strings.HasPrefix(l, " ") {
			t.Errorf("continuation %d lacks leading space", i)
		}
	}
	unfolded := strings.ReplaceAll(strings.TrimSuffix(folded, "\r\n"), "\r\n ", "")
	if unfolded != long {
		t.Error("unfolding does not restore the line")
	}
	if got := foldICS("SHORT:x"); got != "SHORT:x\r\n" {
		t.Errorf("short = %q", got)
	}
}

func TestWhenText(t *testing.T) {
	tm := "18:00"
	if got := whenText("2026-10-02", &tm); got != "Fr, 2. Okt · 18:00" {
		t.Errorf("whenText = %q", got)
	}
	if got := whenText("2026-03-04", nil); got != "Mi, 4. Mär" {
		t.Errorf("whenText = %q", got)
	}
}

func TestDefaultReminder(t *testing.T) {
	berlin, _ := time.LoadLocation("Europe/Berlin")
	got, err := defaultReminder("2026-10-02", berlin)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 10, 1, 16, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("reminder = %v, want %v (18:00 CEST)", got, want)
	}
	// Winter time: 18:00 CET is 17:00 UTC.
	got, _ = defaultReminder("2026-12-24", berlin)
	if want := time.Date(2026, 12, 23, 17, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("reminder = %v, want %v (18:00 CET)", got, want)
	}
}
