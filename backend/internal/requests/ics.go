package requests

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// vtimezoneBerlin is the VTIMEZONE component for Europe/Berlin (EU DST rules).
const vtimezoneBerlin = "BEGIN:VTIMEZONE\r\n" +
	"TZID:Europe/Berlin\r\n" +
	"BEGIN:STANDARD\r\n" +
	"DTSTART:19701025T030000\r\n" +
	"TZOFFSETFROM:+0200\r\n" +
	"TZOFFSETTO:+0100\r\n" +
	"RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU\r\n" +
	"TZNAME:CET\r\n" +
	"END:STANDARD\r\n" +
	"BEGIN:DAYLIGHT\r\n" +
	"DTSTART:19700329T020000\r\n" +
	"TZOFFSETFROM:+0100\r\n" +
	"TZOFFSETTO:+0200\r\n" +
	"RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU\r\n" +
	"TZNAME:CEST\r\n" +
	"END:DAYLIGHT\r\n" +
	"END:VTIMEZONE\r\n"

// escapeICS escapes a TEXT value (RFC 5545 section 3.3.11): backslash, semicolon,
// comma and line breaks.
func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	r := strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\n", `\n`)
	return r.Replace(s)
}

// foldICS folds a content line at 75 octets (RFC 5545 section 3.1): continuation
// lines start with one space. It never splits a UTF-8 sequence. The result ends
// with CRLF.
func foldICS(line string) string {
	const limit = 75
	var b strings.Builder
	n := 0 // octets on the current physical line
	for _, r := range line {
		w := utf8.RuneLen(r)
		if n+w > limit {
			b.WriteString("\r\n ")
			n = 1 // the leading space counts
		}
		b.WriteRune(r)
		n += w
	}
	b.WriteString("\r\n")
	return b.String()
}

// icsEncoder assembles a VCALENDAR.
type icsEncoder struct {
	b strings.Builder
}

func (e *icsEncoder) line(format string, a ...any) {
	e.b.WriteString(foldICS(fmt.Sprintf(format, a...)))
}

// BuildICS renders the requests as an iCalendar document. Times are
// wall-clock times in tz (TZID=Europe/Berlin with an embedded VTIMEZONE);
// requests without a start time become all-day events.
func BuildICS(reqs []Request, tz *time.Location, now time.Time) string {
	e := &icsEncoder{}
	e.line("BEGIN:VCALENDAR")
	e.line("VERSION:2.0")
	e.line("PRODID:-//Stallfunk//Anfragen//DE")
	e.line("CALSCALE:GREGORIAN")
	e.line("METHOD:PUBLISH")
	e.line("X-WR-CALNAME:Stallfunk Anfragen")
	tzid := tz.String()
	if tzid == "Europe/Berlin" {
		e.b.WriteString(vtimezoneBerlin)
	}
	for _, r := range reqs {
		writeEvent(e, r, tz, tzid, now)
	}
	e.line("END:VCALENDAR")
	return e.b.String()
}

func writeEvent(e *icsEncoder, r Request, tz *time.Location, tzid string, now time.Time) {
	e.line("BEGIN:VEVENT")
	e.line("UID:%s@reiterhof.app", r.ID)
	e.line("DTSTAMP:%s", now.UTC().Format("20060102T150405Z"))

	day, _ := time.ParseInLocation("2006-01-02", r.Date, tz)
	endDay, _ := time.ParseInLocation("2006-01-02", r.EndDate(), tz)
	if r.TimeFrom == nil {
		e.line("DTSTART;VALUE=DATE:%s", day.Format("20060102"))
		e.line("DTEND;VALUE=DATE:%s", endDay.AddDate(0, 0, 1).Format("20060102"))
	} else {
		start := clockOn(day, *r.TimeFrom, tz)
		end := start.Add(time.Hour)
		if r.TimeTo != nil {
			end = clockOn(endDay, *r.TimeTo, tz)
		}
		e.line("DTSTART;TZID=%s:%s", tzid, start.Format("20060102T150405"))
		e.line("DTEND;TZID=%s:%s", tzid, end.Format("20060102T150405"))
	}
	e.line("SUMMARY:%s", escapeICS(r.title()))
	if r.Location != "" {
		e.line("LOCATION:%s", escapeICS(r.Location))
	}
	if d := icsDescription(r); d != "" {
		e.line("DESCRIPTION:%s", escapeICS(d))
	}
	if r.Status == StatusCancelled {
		e.line("STATUS:CANCELLED")
	} else {
		e.line("STATUS:CONFIRMED")
	}
	e.line("END:VEVENT")
}

// clockOn returns HH:MM on the given day in tz.
func clockOn(day time.Time, hhmm string, tz *time.Location) time.Time {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return day
	}
	y, m, d := day.Date()
	return time.Date(y, m, d, t.Hour(), t.Minute(), 0, 0, tz)
}

func icsDescription(r Request) string {
	var lines []string
	if r.Description != "" {
		lines = append(lines, r.Description)
	}
	if list := r.checklist(); len(list) > 0 {
		lines = append(lines, "Checkliste: "+strings.Join(list, ", "))
	}
	if len(r.Helpers) > 0 {
		names := make([]string, len(r.Helpers))
		for i, h := range r.Helpers {
			names[i] = h.Name
		}
		lines = append(lines, "Helfer: "+strings.Join(names, ", "))
	}
	lines = append(lines, "Angefragt von "+r.CreatorName)
	return strings.Join(lines, "\n")
}
