package calendar

import (
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Non-UTC wall clock rendered with a UTC designator (BUG-AUDIT-REPORT H8)
//
// NewDate -- and therefore NewEvent, WithEnd and WithStamp -- defaults to
// LayoutTimeUTC, whose layout string ends in a literal "Z". Output formatted the
// instant in the value's own zone, so a +08:00 event was written as "090000Z": a
// timestamp 8 hours off, while claiming to be UTC. Tests that constructed every input
// with time.UTC hid this.
// =============================================================================

// offsetZone returns a deterministic location for an offset in seconds, independent
// of the machine's zone and of tzdata availability.
func offsetZone(name string, seconds int) *time.Location {
	return time.FixedZone(name, seconds)
}

// outputPropValue returns the value of property name in rendered ICS output.
func outputPropValue(out, name string) (string, bool) {
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSuffix(line, "\r"), name+":"); ok {
			return v, true
		}
	}
	return "", false
}

// TestDate_OutputConvertsToUTCWhenLayoutCarriesTheDesignator is the H8 regression: any
// non-zero offset must be folded into UTC before the value is written with "Z".
func TestDate_OutputConvertsToUTCWhenLayoutCarriesTheDesignator(t *testing.T) {
	tests := []struct {
		name string
		key  string
		at   time.Time
		want string
	}{
		{
			name: "positive whole-hour offset",
			key:  "DTSTART",
			at:   time.Date(2024, 1, 2, 9, 0, 0, 0, offsetZone("UTC+8", 8*3600)),
			want: "DTSTART:20240102T010000Z",
		},
		{
			name: "negative whole-hour offset",
			key:  "DTEND",
			at:   time.Date(2024, 1, 2, 9, 0, 0, 0, offsetZone("UTC-5", -5*3600)),
			want: "DTEND:20240102T140000Z",
		},
		{
			name: "half-hour offset",
			key:  "DTSTAMP",
			at:   time.Date(2024, 1, 2, 9, 0, 0, 0, offsetZone("UTC+5:30", 5*3600+30*60)),
			want: "DTSTAMP:20240102T033000Z",
		},
		{
			name: "offset crossing into the previous day",
			key:  "DTSTART",
			at:   time.Date(2024, 1, 2, 3, 0, 0, 0, offsetZone("UTC+8", 8*3600)),
			want: "DTSTART:20240101T190000Z",
		},
		{
			name: "UTC input is unchanged",
			key:  "DTSTART",
			at:   time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC),
			want: "DTSTART:20240102T010000Z",
		},
		{
			name: "zero-offset non-UTC location is unchanged",
			key:  "DTSTART",
			at:   time.Date(2024, 1, 2, 1, 0, 0, 0, offsetZone("UTC+0", 0)),
			want: "DTSTART:20240102T010000Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(NewDate(tt.key, tt.at).Output()); got != tt.want {
				t.Errorf("Output() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDate_OutputPreservesNonUTCLayouts pins the layouts that must not be converted:
// date-only values, floating times and TZID-qualified local times keep their wall
// clock, which is what makes the UTC conversion above layout-specific rather than a
// blanket UTC() call.
func TestDate_OutputPreservesNonUTCLayouts(t *testing.T) {
	tests := []struct {
		name string
		date Date
		want string
	}{
		{
			name: "VALUE=DATE keeps the calendar date",
			date: Date{
				key: "DTSTART", layout: LayoutDate, configs: []string{DateFormat},
				Time: time.Date(2024, 1, 2, 9, 0, 0, 0, offsetZone("UTC+8", 8*3600)),
			},
			want: "DTSTART;VALUE=DATE:20240102",
		},
		{
			name: "floating time keeps its wall clock",
			date: Date{
				key: "DTSTART", layout: LayoutTime,
				Time: time.Date(2024, 1, 2, 9, 0, 0, 0, offsetZone("UTC+8", 8*3600)),
			},
			want: "DTSTART:20240102T090000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.date.Output()); got != tt.want {
				t.Errorf("Output() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDate_OutputKeepsTZIDWallClock covers the TZID path: the value is stored as an
// instant but must re-render in the named zone, not in UTC.
func TestDate_OutputKeepsTZIDWallClock(t *testing.T) {
	if _, err := time.LoadLocation("America/New_York"); err != nil {
		t.Skip("tzdata unavailable")
	}

	// 2024-06-01T13:00Z is 09:00 in New York (EDT, UTC-4).
	date := Date{
		key: "DTSTART", layout: LayoutTime, configs: []string{"TZID=America/New_York"},
		tzid: "America/New_York",
		Time: time.Date(2024, 6, 1, 13, 0, 0, 0, time.UTC),
	}
	want := "DTSTART;TZID=America/New_York:20240601T090000"
	if got := string(date.Output()); got != want {
		t.Errorf("Output() = %q, want %q", got, want)
	}
}

// TestNewEvent_RendersNonUTCInputsInUTC covers the constructors that go through
// NewDate: the event start, WithEnd and WithStamp.
func TestNewEvent_RendersNonUTCInputsInUTC(t *testing.T) {
	start := time.Date(2024, 1, 2, 9, 0, 0, 0, offsetZone("UTC+8", 8*3600))

	e := NewEvent("summary", "description", start,
		WithEnd(start.Add(time.Hour)),
		WithStamp(start),
	)
	out := string(e.Output())

	for _, want := range []string{
		"DTSTART:20240102T010000Z",
		"DTEND:20240102T020000Z",
		"DTSTAMP:20240102T010000Z",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q:\n%s", want, out)
		}
	}
}

// TestNewEvent_CreatedAtIsCorrectOnNonUTCMachine covers the built-in
// createdAt=time.Now(): time.Now() returns a local time, so on a non-UTC machine the
// CREATED line used to carry the local wall clock with a "Z". time.Local is pinned so
// this assertion is discriminating on any machine.
func TestNewEvent_CreatedAtIsCorrectOnNonUTCMachine(t *testing.T) {
	// calendar tests do not run in parallel, so overriding the global is safe here;
	// restore it either way.
	original := time.Local
	time.Local = offsetZone("TEST+8", 8*3600)
	t.Cleanup(func() { time.Local = original })

	before := time.Now().UTC()
	e := NewEvent("summary", "description", time.Now())
	after := time.Now().UTC()

	created, ok := outputPropValue(string(e.Output()), "CREATED")
	if !ok {
		t.Fatalf("CREATED line missing from output:\n%s", e.Output())
	}
	if !strings.HasSuffix(created, "Z") {
		t.Fatalf("CREATED = %q, want the UTC designator", created)
	}
	got, err := time.Parse(LayoutTimeUTC, created)
	if err != nil {
		t.Fatalf("CREATED = %q is not a UTC timestamp: %v", created, err)
	}
	if got.Before(before.Add(-time.Minute)) || got.After(after.Add(time.Minute)) {
		t.Errorf("CREATED = %s denotes %s, which is outside %s..%s: the wall clock was "+
			"written as UTC without converting the offset",
			created, got.Format(time.RFC3339), before.Format(time.RFC3339), after.Format(time.RFC3339))
	}
}
