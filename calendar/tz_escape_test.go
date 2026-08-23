package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestParseDateTZID(t *testing.T) {
	// Regression: DTSTART;TZID=America/New_York:20240601T090000 used to fail
	// parsing entirely (LayoutTimeUTC has a Z suffix) and zeroed the time.
	d := parseDate("DTSTART", []string{"TZID=America/New_York"}, "20240601T090000")
	if d.IsZero() {
		t.Fatal("TZID date failed to parse")
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	want := time.Date(2024, 6, 1, 9, 0, 0, 0, loc)
	if !d.Equal(want) {
		t.Errorf("parsed = %v, want %v", d.Time, want)
	}

	// Output must re-render in the TZID zone, wall-clock preserved.
	out := string(d.Output())
	if !strings.Contains(out, "DTSTART;TZID=America/New_York:20240601T090000") {
		t.Errorf("output = %q, TZID wall-clock not preserved", out)
	}
}

func TestParseDateFloatingStaysWallClock(t *testing.T) {
	// Floating time (no Z, no TZID) must not be shifted by UTC() on output.
	d := parseDate("DTSTART", nil, "20240601T090000")
	if d.IsZero() {
		t.Fatal("floating date failed to parse")
	}
	if got := string(d.Output()); got != "DTSTART:20240601T090000" {
		t.Errorf("floating output = %q, want wall-clock preserved", got)
	}
}

func TestParseDateUTCLayout(t *testing.T) {
	d := parseDate("DTSTART", nil, "20240601T090000Z")
	if d.IsZero() {
		t.Fatal("UTC date failed to parse")
	}
	if got := string(d.Output()); got != "DTSTART:20240601T090000Z" {
		t.Errorf("UTC output = %q", got)
	}
}

func TestParseDateDateValue(t *testing.T) {
	d := parseDate("DTSTART", []string{DateFormat}, "20240601")
	if d.IsZero() {
		t.Fatal("VALUE=DATE failed to parse")
	}
	if got := string(d.Output()); got != "DTSTART;VALUE=DATE:20240601" {
		t.Errorf("date output = %q", got)
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	cases := []string{
		"plain",
		"comma, inside",
		"semi; inside",
		"back\\slash",
		"line1\nline2",
		"mixed, ; \\ and \n together",
		"",
	}
	for _, c := range cases {
		if got := UnescapeText(EscapeText(c)); got != c {
			t.Errorf("round trip %q -> %q -> %q", c, EscapeText(c), got)
		}
	}
}

func TestParseUnescapesTextValues(t *testing.T) {
	// On the wire, RFC 5545 escapes are literal backslash sequences:
	// comma -> \,  semicolon -> \;  newline -> \n
	ics := "BEGIN:VCALENDAR\n" +
		"BEGIN:VEVENT\n" +
		"SUMMARY:Title with comma\\, and break\\nline\n" +
		"DESCRIPTION:semi\\;colon\n" +
		"END:VEVENT\n" +
		"END:VCALENDAR\n"

	cal, err := Parse([]byte(ics))
	if err != nil {
		t.Fatalf("parse fail: %s", err)
	}
	if len(cal.Events()) != 1 {
		t.Fatalf("events = %d", len(cal.Events()))
	}

	sum := string(cal.Events()[0].summary)
	if sum != "Title with comma, and break\nline" {
		t.Errorf("summary = %q, escapes not unescaped", sum)
	}
	if desc := string(cal.Events()[0].desc); desc != "semi;colon" {
		t.Errorf("description = %q", desc)
	}
}

func TestParsePropertyQuotedParams(t *testing.T) {
	// Semicolons inside quoted parameter values must not split params.
	name, params, value := parseProperty("ATTENDEE;CN=\"Smith;John\";ROLE=REQ-PARTICIPANT:mailto:x@y.z")
	if name != "ATTENDEE" {
		t.Errorf("name = %q", name)
	}
	if len(params) != 2 {
		t.Fatalf("params = %v, want 2 (quoted ; must not split)", params)
	}
	if params[0] != "CN=\"Smith;John\"" {
		t.Errorf("params[0] = %q", params[0])
	}
	if value != "mailto:x@y.z" {
		t.Errorf("value = %q", value)
	}
}
