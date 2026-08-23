package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestNewCalendar(t *testing.T) {

}

func TestOutput(t *testing.T) {
	c := &Calendar{
		prodID:   generalProdID,
		version:  generalVersion,
		scale:    ScaleGregorian,
		method:   MethodPublish,
		name:     "test calendar",
		timeZone: TZShanghai,
		desc:     "this is a test calendar",
		events: []Event{
			{
				start:       Date{Time: time.Now().UTC().Add(-24 * time.Hour)},
				end:         Date{Time: time.Now().UTC()},
				uid:         "abc",
				class:       ClassPublic,
				createdAt:   Date{Time: time.Now()},
				desc:        "test event A",
				location:    "SG",
				sequence:    0,
				status:      StatusConfirmed,
				summary:     "test event Title A",
				transparent: TranspTransparent,
			},
			{
				start:       Date{Time: time.Now().UTC()},
				end:         Date{Time: time.Now().UTC().Add(24 * time.Hour)},
				uid:         "def",
				class:       ClassPublic,
				createdAt:   Date{Time: time.Now()},
				desc:        "test event B",
				location:    "CN",
				sequence:    1,
				status:      StatusConfirmed,
				summary:     "test event Title B",
				transparent: TranspTransparent,
			},
		},
	}

	t.Logf("out:\n%s", c.Output())
}

func TestNewCalendarWithOptions(t *testing.T) {
	c := NewCalendar("mycal", "description",
		WithProdID("//Example//Cal 1.0//EN"),
		WithVersion("3.0"),
		WithScale(ScaleGregorian),
		WithMethod(MethodPublish),
		WithTimeZone("Asia/Shanghai"),
	)

	out := string(c.Output())
	for _, want := range []string{
		"BEGIN:VCALENDAR",
		"PRODID://Example//Cal 1.0//EN",
		"VERSION:3.0",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"X-WR-CALNAME:mycal",
		"X-WR-TIMEZONE:Asia/Shanghai",
		"X-WR-CALDESC:description",
		"END:VCALENDAR",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("calendar output missing %q", want)
		}
	}
}

func TestCalendar_AddAndAccessors(t *testing.T) {
	c := NewCalendar("cal", "")

	tz := Timezone{TZID: "Asia/Shanghai"}
	ev := Event{header: CompVEVENT, tailer: CompVEVENT, summary: "e"}
	todo := Todo{header: CompVTODO, tailer: CompVTODO, summary: "todo"}
	j := Journal{header: CompVJOURNAL, tailer: CompVJOURNAL, summary: "journal"}

	c.AddTimezones(tz)
	c.AddEvents(ev)
	c.AddTodos(todo)
	c.AddJournals(j)

	if len(c.Timezones()) != 1 || c.Timezones()[0].TZID != "Asia/Shanghai" {
		t.Error("Timezones() wrong")
	}
	if len(c.Events()) != 1 {
		t.Error("Events() wrong")
	}
	if len(c.Todos()) != 1 {
		t.Error("Todos() wrong")
	}
	if len(c.Journals()) != 1 {
		t.Error("Journals() wrong")
	}

	out := string(c.Output())
	for _, want := range []string{"BEGIN:VTIMEZONE", "TZID:Asia/Shanghai", "BEGIN:VEVENT", "BEGIN:VTODO", "BEGIN:VJOURNAL"} {
		if !strings.Contains(out, want) {
			t.Errorf("calendar output missing %q", want)
		}
	}
}

func TestNewEventWithOptions(t *testing.T) {
	base := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	e := NewEvent("sum", "desc", base,
		WithEnd(base.Add(time.Hour)),
		WithStamp(base),
		WithUID("uid-1"),
		WithClass(ClassPrivate),
		WithCreatedAt(base),
		WithModifiedAt(base),
		WithLocation("Beijing"),
		WithSequence(3),
		WithStatus(StatusTentative),
		WithTransparent(TranspOpaque),
		WithRRULE("FREQ=DAILY;COUNT=5"),
		WithOrganizer("mailto:org@example.com", "CN=Org"),
		WithAttendee("mailto:a@example.com", "RSVP=TRUE"),
		WithAttendee("mailto:b@example.com"),
		WithCategories("work", "meeting"),
		WithPriority(5),
		WithEventURL("https://example.com/e"),
		WithDuration("PT1H"),
		WithExDate(base.Add(24*time.Hour)),
		WithRDate(base.Add(48*time.Hour)),
		WithRecurrenceID(base.Add(72*time.Hour)),
		WithGeo(39.9, 116.4),
		WithComment("comment"),
		WithContact("contact"),
		WithRelatedTo("rel-uid"),
		WithResources("room"),
		WithAttachment("https://example.com/f.txt", "FMTTYPE=text/plain"),
		WithAlarm(Alarm{Action: "DISPLAY", Trigger: "-PT30M"}),
	)

	out := string(e.Output())
	for _, want := range []string{
		"BEGIN:VEVENT",
		"DTSTART:20240601T100000Z",
		"DTEND:20240601T110000Z",
		"DTSTAMP:20240601T100000Z",
		"UID:uid-1",
		"CLASS:PRIVATE",
		"LOCATION:Beijing",
		"SEQUENCE:3",
		"STATUS:TENTATIVE",
		"TRANSP:OPAQUE",
		"RRULE:FREQ=DAILY;COUNT=5",
		"ORGANIZER;CN=Org:mailto:org@example.com",
		"ATTENDEE;RSVP=TRUE:mailto:a@example.com",
		"ATTENDEE:mailto:b@example.com",
		"CATEGORIES:work,meeting",
		"PRIORITY:5",
		"URL:https://example.com/e",
		"DURATION:PT1H",
		"EXDATE:20240602T100000Z",
		"RDATE:20240603T100000Z",
		"RECURRENCE-ID:20240604T100000Z",
		"GEO:39.9;116.4",
		"COMMENT:comment",
		"CONTACT:contact",
		"RELATED-TO:rel-uid",
		"RESOURCES:room",
		"ATTACH;FMTTYPE=text/plain:https://example.com/f.txt",
		"BEGIN:VALARM",
		"ACTION:DISPLAY",
		"TRIGGER:-PT30M",
		"END:VALARM",
		"END:VEVENT",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("event output missing %q", out)
		}
	}
}

func TestEventCustomTimeFormats(t *testing.T) {
	base := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	e := NewEvent("s", "d", base,
		WithEnd(base.Add(time.Hour)),
		SetStartFormat("20060102", "VALUE=DATE"),
		SetEndFormat("20060102", "VALUE=DATE"),
	)

	out := string(e.Output())
	if !strings.Contains(out, "DTSTART;VALUE=DATE:20240601") {
		t.Errorf("start with custom format = %q", out)
	}
	if !strings.Contains(out, "DTEND;VALUE=DATE:20240601") {
		t.Errorf("end with custom format = %q", out)
	}
}

func TestModelsOutput(t *testing.T) {
	ts := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	if got := string(NewDate("X-TIME", ts).Output()); got != "X-TIME:20240102T030405Z" {
		t.Errorf("NewDate output = %q", got)
	}

	dl := NewDateList("RDATE", []time.Time{ts, ts.Add(time.Hour)})
	if got := string(dl.Output()); got != "RDATE:20240102T030405Z,20240102T040405Z" {
		t.Errorf("NewDateList output = %q", got)
	}

	att := NewAttendee("mailto:x@example.com", "ROLE=REQ-PARTICIPANT")
	if got := string(att.Output()); !strings.Contains(got, "ATTENDEE;ROLE=REQ-PARTICIPANT:mailto:x@example.com") {
		t.Errorf("NewAttendee output = %q", got)
	}

	org := NewOrganizer("mailto:o@example.com", "CN=O")
	if got := string(org.Output()); !strings.Contains(got, "ORGANIZER;CN=O:mailto:o@example.com") {
		t.Errorf("NewOrganizer output = %q", got)
	}

	attach := NewAttachment("mailto:attach", "FMTTYPE=text/plain")
	if got := string(attach.Output()); !strings.Contains(got, "ATTACH;FMTTYPE=text/plain:mailto:attach") {
		t.Errorf("NewAttachment output = %q", got)
	}

	if got := string(Geo{Lat: 1.5, Lon: -2.25}.Output()); got != "GEO:1.5;-2.25" {
		t.Errorf("Geo output = %q", got)
	}
}

func TestTimezoneAndAlarmOutput(t *testing.T) {
	tz := Timezone{
		TZID:     "Europe/Berlin",
		Standard: []TimezoneProp{{Kind: "STANDARD", TZOffsetFrom: "+02:00", TZOffsetTo: "+01:00", TZName: "CET"}},
		Daylight: []TimezoneProp{{Kind: "DAYLIGHT", TZOffsetFrom: "+01:00", TZOffsetTo: "+02:00", RRULE: "FREQ=YEARLY", Comment: "c"}},
	}
	out := string(tz.Output())
	for _, want := range []string{
		"BEGIN:VTIMEZONE",
		"TZID:Europe/Berlin",
		"BEGIN:STANDARD",
		"TZOFFSETFROM:+02:00",
		"TZOFFSETTO:+01:00",
		"TZNAME:CET",
		"END:STANDARD",
		"BEGIN:DAYLIGHT",
		"RRULE:FREQ=YEARLY",
		"COMMENT:c",
		"END:DAYLIGHT",
		"END:VTIMEZONE",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("timezone output missing %q", want)
		}
	}

	a := Alarm{
		Action:    "EMAIL",
		Trigger:   "-PT15M",
		Desc:      "d",
		Summary:   "s",
		Duration:  "PT5M",
		Repeat:    3,
		Attendees: []Attendee{NewAttendee("mailto:a@e.com")},
	}
	aout := string(a.Output())
	for _, want := range []string{
		"BEGIN:VALARM", "ACTION:EMAIL", "TRIGGER:-PT15M",
		"DESCRIPTION:d", "SUMMARY:s", "DURATION:PT5M", "REPEAT:3",
		"ATTENDEE:mailto:a@e.com", "END:VALARM",
	} {
		if !strings.Contains(aout, want) {
			t.Errorf("alarm output missing %q", want)
		}
	}
}

func TestTodoOutputFullFields(t *testing.T) {
	ts := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	todo := Todo{
		header:     CompVTODO,
		tailer:     CompVTODO,
		stamp:      NewDate("DTSTAMP", ts),
		uid:        "todo-1",
		start:      NewDate("DTSTART", ts),
		due:        NewDate("DUE", ts.Add(24*time.Hour)),
		duration:   "P1D",
		summary:    "write tests",
		desc:       "cover all todo fields",
		priority:   7,
		status:     TodoStatusNeedsAction,
		seq:        2,
		class:      ClassConfidential,
		categories: Categories{"qa"},
		completed:  NewDate("COMPLETED", ts.Add(48*time.Hour)),
		percent:    80,
		location:   "remote",
		organizer:  NewOrganizer("mailto:org@example.com"),
		attendees:  []Attendee{NewAttendee("mailto:doer@example.com")},
		url:        "https://example.com/todo",
		rrule:      "FREQ=WEEKLY",
		exdates:    []DateList{NewDateList("EXDATE", []time.Time{ts})},
		rdates:     []DateList{NewDateList("RDATE", []time.Time{ts})},
		alarms:     []Alarm{{Action: "DISPLAY", Trigger: "-PT10M"}},
	}

	out := string(todo.Output())
	for _, want := range []string{
		"BEGIN:VTODO",
		"DTSTAMP:20240601T100000Z",
		"UID:todo-1",
		"DTSTART:20240601T100000Z",
		"DUE:20240602T100000Z",
		"DURATION:P1D",
		"SUMMARY:write tests",
		"DESCRIPTION:cover all todo fields",
		"PRIORITY:7",
		"STATUS:NEEDS-ACTION",
		"SEQUENCE:2",
		"CLASS:CONFIDENTIAL",
		"CATEGORIES:qa",
		"COMPLETED:20240603T100000Z",
		"PERCENT-COMPLETE:80",
		"LOCATION:remote",
		"ORGANIZER:mailto:org@example.com",
		"ATTENDEE:mailto:doer@example.com",
		"URL:https://example.com/todo",
		"RRULE:FREQ=WEEKLY",
		"EXDATE:20240601T100000Z",
		"RDATE:20240601T100000Z",
		"BEGIN:VALARM",
		"END:VTODO",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("todo output missing %q", want)
		}
	}
}

func TestJournalOutputFullFields(t *testing.T) {
	ts := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	j := Journal{
		header:     CompVJOURNAL,
		tailer:     CompVJOURNAL,
		stamp:      NewDate("DTSTAMP", ts),
		uid:        "j-1",
		start:      NewDate("DTSTART", ts),
		summary:    "daily notes",
		desc:       "stuff",
		class:      ClassPublic,
		categories: Categories{"log"},
		status:     JournalStatusDraft,
		url:        "https://example.com/j",
		organizer:  NewOrganizer("mailto:org@example.com"),
		attendees:  []Attendee{NewAttendee("mailto:a@example.com")},
	}

	out := string(j.Output())
	for _, want := range []string{
		"BEGIN:VJOURNAL",
		"DTSTAMP:20240601T100000Z",
		"UID:j-1",
		"DTSTART:20240601T100000Z",
		"SUMMARY:daily notes",
		"DESCRIPTION:stuff",
		"CLASS:PUBLIC",
		"CATEGORIES:log",
		"STATUS:DRAFT",
		"URL:https://example.com/j",
		"ORGANIZER:mailto:org@example.com",
		"ATTENDEE:mailto:a@example.com",
		"END:VJOURNAL",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("journal output missing %q", want)
		}
	}
}
