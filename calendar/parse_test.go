package calendar

import (
	"strings"
	"testing"
	"time"
)

const testICS = `BEGIN:VCALENDAR
PRODID:-//Test//Test Calendar//EN
VERSION:2.0
CALSCALE:GREGORIAN
METHOD:PUBLISH
X-WR-CALNAME:Test Calendar
X-WR-TIMEZONE:Asia/Shanghai
X-WR-CALDESC:A test calendar
BEGIN:VEVENT
DTSTART:20240101T090000Z
DTEND:20240101T100000Z
DTSTAMP:20240101T000000Z
UID:test-uid-001
CLASS:PUBLIC
CREATED:20231230T120000Z
DESCRIPTION:Test event description
LAST-MODIFIED:20231231T120000Z
LOCATION:Beijing
SEQUENCE:0
STATUS:CONFIRMED
SUMMARY:New Year Event
TRANSP:OPAQUE
RRULE:FREQ=YEARLY
END:VEVENT
BEGIN:VEVENT
DTSTART:20240115T140000Z
DTEND:20240115T150000Z
DTSTAMP:20240115T000000Z
UID:test-uid-002
SUMMARY:Team Meeting
SEQUENCE:1
END:VEVENT
END:VCALENDAR`

func TestParse(t *testing.T) {
	cal, err := Parse([]byte(testICS))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Calendar-level fields
	assertEqual(t, "prodID", string(cal.prodID), "-//Test//Test Calendar//EN")
	assertEqual(t, "version", string(cal.version), "2.0")
	assertEqual(t, "scale", string(cal.scale), "GREGORIAN")
	assertEqual(t, "method", string(cal.method), "PUBLISH")
	assertEqual(t, "name", string(cal.name), "Test Calendar")
	assertEqual(t, "timezone", string(cal.timeZone), "Asia/Shanghai")
	assertEqual(t, "desc", string(cal.desc), "A test calendar")

	// Events
	if len(cal.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(cal.events))
	}

	// Event 1
	e1 := cal.events[0]
	assertEqual(t, "event1 uid", string(e1.uid), "test-uid-001")
	assertEqual(t, "event1 summary", string(e1.summary), "New Year Event")
	assertEqual(t, "event1 desc", string(e1.desc), "Test event description")
	assertEqual(t, "event1 location", string(e1.location), "Beijing")
	assertEqual(t, "event1 class", string(e1.class), "PUBLIC")
	assertEqual(t, "event1 status", string(e1.status), "CONFIRMED")
	assertEqual(t, "event1 transp", string(e1.transparent), "OPAQUE")
	assertEqual(t, "event1 rrule", string(e1.rrule), "FREQ=YEARLY")
	assertEqualInt(t, "event1 sequence", int(e1.sequence), 0)
	assertTime(t, "event1 start", e1.start, "2024-01-01T09:00:00Z")
	assertTime(t, "event1 end", e1.end, "2024-01-01T10:00:00Z")
	assertTime(t, "event1 stamp", e1.stamp, "2024-01-01T00:00:00Z")
	assertTime(t, "event1 created", e1.createdAt, "2023-12-30T12:00:00Z")
	assertTime(t, "event1 modified", e1.modifiedAt, "2023-12-31T12:00:00Z")

	// Event 2
	e2 := cal.events[1]
	assertEqual(t, "event2 uid", string(e2.uid), "test-uid-002")
	assertEqual(t, "event2 summary", string(e2.summary), "Team Meeting")
	assertEqualInt(t, "event2 sequence", int(e2.sequence), 1)
}

func TestParseDateOnly(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART;VALUE=DATE:20240101
SUMMARY:All Day Event
END:VEVENT
END:VCALENDAR`

	cal, err := Parse([]byte(ics))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(cal.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(cal.events))
	}
	e := cal.events[0]
	assertEqual(t, "summary", string(e.summary), "All Day Event")
	assertTime(t, "start", e.start, "2024-01-01T00:00:00Z")
	// Date-only should use LayoutDate
	if e.start.layout != LayoutDate {
		t.Errorf("expected layout %q, got %q", LayoutDate, e.start.layout)
	}
}

func TestParseLineUnfolding(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nSUMMARY:This is a very\r\n long description that\r\n spans multiple lines\r\nEND:VEVENT\r\nEND:VCALENDAR"

	cal, err := Parse([]byte(ics))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(cal.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(cal.events))
	}
	assertEqual(t, "summary", string(cal.events[0].summary), "This is a verylong description thatspans multiple lines")
}

func TestParseSkipsUnknownComponents(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTIMEZONE
TZID:Asia/Shanghai
BEGIN:STANDARD
DTSTART:19700101T000000
TZOFFSETFROM:+0800
TZOFFSETTO:+0800
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
DTSTART:20240101T090000Z
SUMMARY:Event After Timezone
END:VEVENT
END:VCALENDAR`

	cal, err := Parse([]byte(ics))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(cal.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(cal.events))
	}
	assertEqual(t, "summary", string(cal.events[0].summary), "Event After Timezone")
}

func TestRoundTrip(t *testing.T) {
	// Build a calendar, output it, parse it back, output again — should match.
	original := &Calendar{
		header:      "VCALENDAR",
		tailer:      "VCALENDAR",
		prodID:      generalProdID,
		version:     generalVersion,
		scale:       ScaleGregorian,
		method:      MethodPublish,
		name:        "Round Trip",
		timeZone:    TZShanghai,
		desc:        "round trip test",
		events: []Event{
			{
				header:      "VEVENT",
				tailer:      "VEVENT",
				start:       Date{key: "DTSTART", layout: LayoutTime, Time: time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC)},
				end:         Date{key: "DTEND", layout: LayoutTime, Time: time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)},
				uid:         "roundtrip-001",
				class:       ClassPublic,
				createdAt:   Date{key: "CREATED", layout: LayoutTime, Time: time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)},
				desc:        "Round trip test event",
				location:    "Shanghai",
				sequence:    2,
				status:      StatusConfirmed,
				summary:     "Round Trip Event",
				transparent: TranspOpaque,
			},
		},
	}

	// Output → Parse → Output
	output1 := original.Output()
	cal, err := Parse(output1)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	output2 := cal.Output()

	if string(output1) != string(output2) {
		t.Errorf("round trip mismatch:\n--- output1\n%s\n--- output2\n%s", output1, output2)
	}
}

func TestParseReader(t *testing.T) {
	r := strings.NewReader(testICS)
	cal, err := ParseReader(r)
	if err != nil {
		t.Fatalf("ParseReader() error: %v", err)
	}
	if len(cal.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(cal.events))
	}
}

// helpers

func assertEqual(t *testing.T, name string, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", name, got, want)
	}
}

func assertEqualInt(t *testing.T, name string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %d, want %d", name, got, want)
	}
}

func assertTime(t *testing.T, name string, d Date, wantRFC3339 string) {
	t.Helper()
	want, err := time.Parse(time.RFC3339, wantRFC3339)
	if err != nil {
		t.Fatalf("bad test time %q: %v", wantRFC3339, err)
	}
	if !d.Equal(want) {
		t.Errorf("%s: got %v, want %v", name, d.Time, want)
	}
}
