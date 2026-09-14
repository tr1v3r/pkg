package calendar

import (
	"strings"
	"testing"
)

// =============================================================================
// Malformed component-ending regression (BUG-AUDIT-REPORT H10)
//
// parseCalendar handled a component END by dereferencing the component pointer
// unconditionally:
//
//	case CompVEVENT:
//	    cal.events = append(cal.events, *event)
//
// An END without its matching BEGIN -- an untrusted, truncated or hand-edited ICS --
// left that pointer nil, so Parse panicked with a nil dereference instead of skipping
// the stray line. The same held for VTODO/VJOURNAL/VTIMEZONE/STANDARD/DAYLIGHT, and a
// stray END:VALARM inside an event dereferenced a nil alarm.
// =============================================================================

// malformedICS builds a CRLF-joined ICS payload from lines, so the fixtures below
// read like real wire input.
func malformedICS(lines ...string) string {
	return strings.Join(lines, "\r\n") + "\r\n"
}

// TestParse_MalformedComponentEndingsDoNotPanic drives Parse with component endings
// that have no matching BEGIN. Every case must parse without panicking, return no
// error (malformed input is skipped today, not rejected -- error semantics are out of
// scope here) and avoid fabricating a component.
func TestParse_MalformedComponentEndingsDoNotPanic(t *testing.T) {
	tests := []struct {
		name       string
		ics        string
		wantEvents int
		wantTodos  int
		wantTZ     int
	}{
		{
			name: "orphan END:VEVENT",
			ics:  malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "END:VEVENT", "END:VCALENDAR"),
		},
		{
			name: "orphan END:VTODO",
			ics:  malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "END:VTODO", "END:VCALENDAR"),
		},
		{
			name: "orphan END:VJOURNAL",
			ics:  malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "END:VJOURNAL", "END:VCALENDAR"),
		},
		{
			name: "orphan END:VTIMEZONE",
			ics:  malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "END:VTIMEZONE", "END:VCALENDAR"),
		},
		{
			name: "orphan END:STANDARD",
			ics:  malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "END:STANDARD", "END:VCALENDAR"),
		},
		{
			name: "orphan END:DAYLIGHT",
			ics:  malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "END:DAYLIGHT", "END:VCALENDAR"),
		},
		{
			name: "orphan END:VALARM",
			ics:  malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "END:VALARM", "END:VCALENDAR"),
		},
		{
			name: "every orphan END in one payload",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0",
				"END:VEVENT", "END:VTODO", "END:VJOURNAL", "END:VTIMEZONE",
				"END:STANDARD", "END:DAYLIGHT", "END:VALARM", "END:VCALENDAR"),
		},
		{
			name: "stray END:VALARM inside a well-formed event",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VEVENT",
				"SUMMARY:a", "END:VALARM", "END:VEVENT", "END:VCALENDAR"),
			wantEvents: 1,
		},
		{
			name: "duplicate END:VEVENT",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VEVENT",
				"SUMMARY:a", "END:VEVENT", "END:VEVENT", "END:VCALENDAR"),
			wantEvents: 1,
		},
		{
			name: "duplicate END:VTODO",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VTODO",
				"SUMMARY:a", "END:VTODO", "END:VTODO", "END:VCALENDAR"),
			wantTodos: 1,
		},
		{
			name: "duplicate END:VTIMEZONE",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VTIMEZONE",
				"TZID:X", "END:VTIMEZONE", "END:VTIMEZONE", "END:VCALENDAR"),
			wantTZ: 1,
		},
		{
			name: "END:STANDARD after END:VTIMEZONE",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VTIMEZONE",
				"TZID:X", "END:VTIMEZONE", "END:STANDARD", "END:VCALENDAR"),
			wantTZ: 1,
		},
		{
			name: "END:DAYLIGHT after END:VTIMEZONE",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VTIMEZONE",
				"TZID:X", "END:VTIMEZONE", "END:DAYLIGHT", "END:VCALENDAR"),
			wantTZ: 1,
		},
		{
			// Here the STANDARD property is genuinely open when its enclosing
			// VTIMEZONE closes, so the guard must also check the timezone pointer.
			name: "END:STANDARD after its enclosing VTIMEZONE closed early",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VTIMEZONE", "TZID:X",
				"BEGIN:STANDARD", "TZOFFSETTO:+0100", "END:VTIMEZONE",
				"END:STANDARD", "END:VCALENDAR"),
			wantTZ: 1,
		},
		{
			name: "END:DAYLIGHT after its enclosing VTIMEZONE closed early",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VTIMEZONE", "TZID:X",
				"BEGIN:DAYLIGHT", "TZOFFSETTO:+0200", "END:VTIMEZONE",
				"END:DAYLIGHT", "END:VCALENDAR"),
			wantTZ: 1,
		},
		{
			name: "STANDARD outside any VTIMEZONE",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:STANDARD",
				"TZOFFSETTO:+0100", "END:STANDARD", "END:VCALENDAR"),
		},
		{
			name: "DAYLIGHT outside any VTIMEZONE",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:DAYLIGHT",
				"TZOFFSETTO:+0200", "END:DAYLIGHT", "END:VCALENDAR"),
		},
		{
			name: "truncated event is never closed",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VEVENT",
				"SUMMARY:a", "END:VCALENDAR"),
		},
		{
			name: "nested BEGIN:VEVENT closed once",
			ics: malformedICS("BEGIN:VCALENDAR", "VERSION:2.0", "BEGIN:VEVENT",
				"SUMMARY:outer", "BEGIN:VEVENT", "SUMMARY:inner", "END:VEVENT", "END:VCALENDAR"),
			wantEvents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cal, err := Parse([]byte(tt.ics))
			if err != nil {
				t.Fatalf("Parse() error = %v, want malformed input to be skipped", err)
			}
			if cal == nil {
				t.Fatal("Parse() returned a nil Calendar")
			}
			assertEqualInt(t, "events", len(cal.Events()), tt.wantEvents)
			assertEqualInt(t, "todos", len(cal.Todos()), tt.wantTodos)
			assertEqualInt(t, "timezones", len(cal.Timezones()), tt.wantTZ)

			// The stray END must not conjure an alarm onto a real event either.
			for i, e := range cal.Events() {
				if len(e.alarms) != 0 {
					t.Errorf("event %d has %d alarms, want 0", i, len(e.alarms))
				}
			}

			// The calendar must stay usable after parsing hostile input.
			_ = cal.Output()
		})
	}
}

// TestParse_WellFormedComponentsStillParse guards the opposite direction: the
// state guards added for the malformed cases must not make valid components get
// skipped. This is the main risk of the fix, so it is pinned explicitly.
func TestParse_WellFormedComponentsStillParse(t *testing.T) {
	ics := malformedICS(
		"BEGIN:VCALENDAR", "VERSION:2.0",
		"BEGIN:VTIMEZONE", "TZID:America/New_York",
		"BEGIN:STANDARD", "DTSTART:19701101T020000", "TZOFFSETFROM:-0400",
		"TZOFFSETTO:-0500", "TZNAME:EST", "END:STANDARD",
		"BEGIN:DAYLIGHT", "DTSTART:19700308T020000", "TZOFFSETFROM:-0500",
		"TZOFFSETTO:-0400", "TZNAME:EDT", "END:DAYLIGHT",
		"END:VTIMEZONE",
		"BEGIN:VEVENT", "DTSTART:20240101T090000Z", "SUMMARY:event", "END:VEVENT",
		"BEGIN:VTODO", "SUMMARY:todo", "END:VTODO",
		"BEGIN:VJOURNAL", "SUMMARY:journal", "END:VJOURNAL",
		"BEGIN:VEVENT", "SUMMARY:with alarm",
		"BEGIN:VALARM", "ACTION:DISPLAY", "TRIGGER:-PT15M", "END:VALARM",
		"END:VEVENT",
		"END:VCALENDAR",
	)

	cal, err := Parse([]byte(ics))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assertEqualInt(t, "events", len(cal.Events()), 2)
	assertEqualInt(t, "todos", len(cal.Todos()), 1)
	assertEqualInt(t, "journals", len(cal.Journals()), 1)
	assertEqualInt(t, "timezones", len(cal.Timezones()), 1)

	tz := cal.Timezones()[0]
	assertEqual(t, "tzid", tz.TZID, "America/New_York")
	assertEqualInt(t, "standard props", len(tz.Standard), 1)
	assertEqualInt(t, "daylight props", len(tz.Daylight), 1)
	assertEqual(t, "standard TZNAME", tz.Standard[0].TZName, "EST")
	assertEqual(t, "daylight TZNAME", tz.Daylight[0].TZName, "EDT")

	// A nested VALARM still attaches to the event that encloses it.
	assertEqualInt(t, "alarms on second event", len(cal.Events()[1].alarms), 1)
	assertEqual(t, "alarm action", cal.Events()[1].alarms[0].Action, "DISPLAY")
}
