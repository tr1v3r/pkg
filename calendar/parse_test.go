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
		header:   CompVCALENDAR,
		tailer:   CompVCALENDAR,
		prodID:   generalProdID,
		version:  generalVersion,
		scale:    ScaleGregorian,
		method:   MethodPublish,
		name:     "Round Trip",
		timeZone: TZShanghai,
		desc:     "round trip test",
		events: []Event{
			{
				header:      CompVEVENT,
				tailer:      CompVEVENT,
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

func TestParseEventWithAllFields(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART:20240101T090000Z
DTEND:20240101T100000Z
DTSTAMP:20240101T000000Z
UID:full-event-001
SUMMARY:Full Featured Event
DESCRIPTION:An event with all RFC 5545 fields
LOCATION:Conference Room A
CLASS:PUBLIC
STATUS:CONFIRMED
TRANSP:OPAQUE
SEQUENCE:2
PRIORITY:5
URL:http://example.com/event
COMMENT:This is a comment
CONTACT:John Doe
RELATED-TO:other-event-uid
RESOURCES:Projector
ORGANIZER;CN=Alice:mailto:alice@example.com
ATTENDEE;ROLE=REQ-PARTICIPANT;RSVP=TRUE;CN=Bob:mailto:bob@example.com
ATTENDEE;ROLE=OPT-PARTICIPANT;CN=Charlie:mailto:charlie@example.com
CATEGORIES:BUSINESS,MEETING
GEO:37.386013;-122.082932
DURATION:PT1H
EXDATE:20240201T090000Z,20240301T090000Z
RDATE:20240401T090000Z
RECURRENCE-ID:20240101T090000Z
RRULE:FREQ=MONTHLY
ATTACH;FMTTYPE=text/html:http://example.com/agenda.html
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
	assertEqual(t, "uid", string(e.uid), "full-event-001")
	assertEqual(t, "summary", string(e.summary), "Full Featured Event")
	assertEqual(t, "desc", string(e.desc), "An event with all RFC 5545 fields")
	assertEqual(t, "location", string(e.location), "Conference Room A")
	assertEqualInt(t, "priority", int(e.priority), 5)
	assertEqual(t, "url", string(e.url), "http://example.com/event")
	assertEqual(t, "comment", string(e.comment), "This is a comment")
	assertEqual(t, "contact", string(e.contact), "John Doe")
	assertEqual(t, "relatedTo", string(e.relatedTo), "other-event-uid")
	assertEqual(t, "resources", string(e.resources), "Projector")
	assertEqual(t, "duration", string(e.duration), "PT1H")
	assertEqual(t, "rrule", string(e.rrule), "FREQ=MONTHLY")

	// Organizer
	assertEqual(t, "organizer URI", e.organizer.URI, "mailto:alice@example.com")
	if len(e.organizer.params) != 1 || e.organizer.params[0] != "CN=Alice" {
		t.Errorf("organizer params: got %v, want [CN=Alice]", e.organizer.params)
	}

	// Attendees
	if len(e.attendees) != 2 {
		t.Fatalf("expected 2 attendees, got %d", len(e.attendees))
	}
	assertEqual(t, "attendee1 URI", e.attendees[0].URI, "mailto:bob@example.com")
	assertEqual(t, "attendee2 URI", e.attendees[1].URI, "mailto:charlie@example.com")

	// Categories
	if len(e.categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(e.categories))
	}
	assertEqual(t, "cat1", e.categories[0], "BUSINESS")
	assertEqual(t, "cat2", e.categories[1], "MEETING")

	// GEO
	if e.geo.Lat == 0 && e.geo.Lon == 0 {
		t.Error("geo should not be zero")
	}

	// EXDATE
	if len(e.exdates) != 1 {
		t.Fatalf("expected 1 exdate list, got %d", len(e.exdates))
	}
	if len(e.exdates[0].Dates) != 2 {
		t.Errorf("expected 2 exdate dates, got %d", len(e.exdates[0].Dates))
	}

	// RDATE
	if len(e.rdates) != 1 {
		t.Fatalf("expected 1 rdate list, got %d", len(e.rdates))
	}
	if len(e.rdates[0].Dates) != 1 {
		t.Errorf("expected 1 rdate date, got %d", len(e.rdates[0].Dates))
	}

	// Attachments
	if len(e.attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(e.attachments))
	}
	assertEqual(t, "attach URI", e.attachments[0].URI, "http://example.com/agenda.html")
}

func TestParseTimezone(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTIMEZONE
TZID:America/New_York
LAST-MODIFIED:20050101T000000Z
BEGIN:STANDARD
DTSTART:19701101T020000
TZOFFSETFROM:-0400
TZOFFSETTO:-0500
TZNAME:EST
RRULE:FREQ=YEARLY;BYDAY=1SU;BYMONTH=11
END:STANDARD
BEGIN:DAYLIGHT
DTSTART:19700308T020000
TZOFFSETFROM:-0500
TZOFFSETTO:-0400
TZNAME:EDT
RRULE:FREQ=YEARLY;BYDAY=2SU;BYMONTH=3
END:DAYLIGHT
END:VTIMEZONE
BEGIN:VEVENT
DTSTART:20240101T090000Z
SUMMARY:Event in timezone
END:VEVENT
END:VCALENDAR`

	cal, err := Parse([]byte(ics))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(cal.timezones) != 1 {
		t.Fatalf("expected 1 timezone, got %d", len(cal.timezones))
	}
	tz := cal.timezones[0]
	assertEqual(t, "tzid", tz.TZID, "America/New_York")

	if len(tz.Standard) != 1 {
		t.Fatalf("expected 1 standard, got %d", len(tz.Standard))
	}
	s := tz.Standard[0]
	assertEqual(t, "std TZOFFSETFROM", s.TZOffsetFrom, "-0400")
	assertEqual(t, "std TZOFFSETTO", s.TZOffsetTo, "-0500")
	assertEqual(t, "std TZNAME", s.TZName, "EST")
	assertEqual(t, "std RRULE", s.RRULE, "FREQ=YEARLY;BYDAY=1SU;BYMONTH=11")

	if len(tz.Daylight) != 1 {
		t.Fatalf("expected 1 daylight, got %d", len(tz.Daylight))
	}
	d := tz.Daylight[0]
	assertEqual(t, "dst TZOFFSETFROM", d.TZOffsetFrom, "-0500")
	assertEqual(t, "dst TZOFFSETTO", d.TZOffsetTo, "-0400")
	assertEqual(t, "dst TZNAME", d.TZName, "EDT")

	// Event should still parse correctly
	if len(cal.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(cal.events))
	}
	assertEqual(t, "event summary", string(cal.events[0].summary), "Event in timezone")
}

func TestParseAlarm(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART:20240101T090000Z
SUMMARY:Event with alarm
UID:alarm-event-001
BEGIN:VALARM
ACTION:DISPLAY
TRIGGER:-PT30M
DESCRIPTION:Reminder: Meeting in 30 minutes
END:VALARM
BEGIN:VALARM
ACTION:AUDIO
TRIGGER:-PT15M
DURATION:PT5M
REPEAT:3
ATTACH;FMTTYPE=audio/basic:http://example.com/sound.wav
END:VALARM
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
	if len(e.alarms) != 2 {
		t.Fatalf("expected 2 alarms, got %d", len(e.alarms))
	}

	a1 := e.alarms[0]
	assertEqual(t, "alarm1 action", a1.Action, "DISPLAY")
	assertEqual(t, "alarm1 trigger", a1.Trigger, "-PT30M")
	assertEqual(t, "alarm1 desc", a1.Desc, "Reminder: Meeting in 30 minutes")

	a2 := e.alarms[1]
	assertEqual(t, "alarm2 action", a2.Action, "AUDIO")
	assertEqual(t, "alarm2 trigger", a2.Trigger, "-PT15M")
	assertEqual(t, "alarm2 duration", a2.Duration, "PT5M")
	assertEqualInt(t, "alarm2 repeat", a2.Repeat, 3)
	if len(a2.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(a2.Attachments))
	}
}

func TestParseTodo(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTODO
DTSTAMP:20240101T000000Z
UID:todo-001
DTSTART:20240101T090000Z
DUE:20240101T170000Z
SUMMARY:Finish report
DESCRIPTION:Complete the quarterly report
PRIORITY:3
STATUS:IN-PROCESS
SEQUENCE:1
CLASS:PUBLIC
CATEGORIES:WORK,REPORT
COMPLETED:20240101T160000Z
PERCENT-COMPLETE:75
LOCATION:Office
URL:http://example.com/todo/001
RRULE:FREQ=WEEKLY
EXDATE:20240108T090000Z
END:VTODO
END:VCALENDAR`

	cal, err := Parse([]byte(ics))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(cal.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(cal.todos))
	}

	todo := cal.todos[0]
	assertEqual(t, "todo uid", string(todo.uid), "todo-001")
	assertEqual(t, "todo summary", string(todo.summary), "Finish report")
	assertEqual(t, "todo desc", string(todo.desc), "Complete the quarterly report")
	assertEqualInt(t, "todo priority", int(todo.priority), 3)
	assertEqual(t, "todo status", string(todo.status), "IN-PROCESS")
	assertEqualInt(t, "todo percent", todo.percent, 75)
	assertEqual(t, "todo location", string(todo.location), "Office")
	assertEqual(t, "todo url", string(todo.url), "http://example.com/todo/001")
	assertEqual(t, "todo rrule", string(todo.rrule), "FREQ=WEEKLY")

	if len(todo.categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(todo.categories))
	}
	if len(todo.exdates) != 1 || len(todo.exdates[0].Dates) != 1 {
		t.Errorf("expected 1 exdate with 1 date")
	}
}

func TestParseJournal(t *testing.T) {
	ics := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VJOURNAL
DTSTAMP:20240101T000000Z
UID:journal-001
DTSTART;VALUE=DATE:20240101
SUMMARY:Daily Notes
DESCRIPTION:Today I worked on the calendar package
CLASS:PUBLIC
CATEGORIES:DEVELOPMENT
STATUS:DRAFT
URL:http://example.com/journal/001
ORGANIZER;CN=Alice:mailto:alice@example.com
ATTENDEE;CN=Bob:mailto:bob@example.com
END:VJOURNAL
END:VCALENDAR`

	cal, err := Parse([]byte(ics))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(cal.journals) != 1 {
		t.Fatalf("expected 1 journal, got %d", len(cal.journals))
	}

	j := cal.journals[0]
	assertEqual(t, "journal uid", string(j.uid), "journal-001")
	assertEqual(t, "journal summary", string(j.summary), "Daily Notes")
	assertEqual(t, "journal desc", string(j.desc), "Today I worked on the calendar package")
	assertEqual(t, "journal status", string(j.status), "DRAFT")
	assertEqual(t, "journal url", string(j.url), "http://example.com/journal/001")
	assertEqual(t, "journal organizer", j.organizer.URI, "mailto:alice@example.com")

	if len(j.attendees) != 1 {
		t.Errorf("expected 1 attendee, got %d", len(j.attendees))
	}
	if len(j.categories) != 1 || j.categories[0] != "DEVELOPMENT" {
		t.Errorf("categories: got %v", j.categories)
	}
}

func TestRoundTripAllComponents(t *testing.T) {
	original := &Calendar{
		header:   CompVCALENDAR,
		tailer:   CompVCALENDAR,
		prodID:   generalProdID,
		version:  generalVersion,
		scale:    ScaleGregorian,
		method:   MethodPublish,
		name:     "Round Trip All",
		timeZone: TZShanghai,
		timezones: []Timezone{
			{
				TZID: "Asia/Shanghai",
				Standard: []TimezoneProp{
					{
						Kind:         CompSTANDARD,
						DTStart:      Date{key: "DTSTART", layout: LayoutTime, Time: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)},
						TZOffsetFrom: "+0900",
						TZOffsetTo:   "+0800",
						TZName:       "CST",
					},
				},
			},
		},
		events: []Event{
			{
				header:     CompVEVENT,
				tailer:     CompVEVENT,
				start:      Date{key: "DTSTART", layout: LayoutTime, Time: time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC)},
				end:        Date{key: "DTEND", layout: LayoutTime, Time: time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)},
				uid:        "rt-001",
				summary:    "Round Trip Event",
				desc:       "Test",
				location:   "Shanghai",
				sequence:   0,
				status:     StatusConfirmed,
				organizer:  Organizer{params: []string{"CN=Alice"}, URI: "mailto:alice@example.com"},
				attendees:  []Attendee{{params: []string{"ROLE=REQ-PARTICIPANT", "RSVP=TRUE"}, URI: "mailto:bob@example.com"}},
				categories: Categories{"BUSINESS", "MEETING"},
				priority:   5,
				url:        "http://example.com",
				geo:        Geo{Lat: 31.2304, Lon: 121.4737},
				alarms:     []Alarm{{Action: "DISPLAY", Trigger: "-PT30M", Desc: "Reminder"}},
			},
		},
		todos: []Todo{
			{
				header:   CompVTODO,
				tailer:   CompVTODO,
				stamp:    Date{key: "DTSTAMP", layout: LayoutTime, Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				uid:      "todo-rt-001",
				summary:  "Round Trip Todo",
				priority: 3,
				status:   TodoStatusNeedsAction,
				seq:      1,
			},
		},
		journals: []Journal{
			{
				header:  CompVJOURNAL,
				tailer:  CompVJOURNAL,
				stamp:   Date{key: "DTSTAMP", layout: LayoutTime, Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				uid:     "journal-rt-001",
				summary: "Round Trip Journal",
				status:  JournalStatusDraft,
			},
		},
	}

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
