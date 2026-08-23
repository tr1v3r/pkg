package calendar

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseDateEdgeCases(t *testing.T) {
	// VALUE=DATE param forces date layout
	d := parseDate("DTSTART", []string{DateFormat}, "20240601")
	if _, err := time.Parse(LayoutDate, "20240601"); d.layout != LayoutDate || err != nil {
		t.Errorf("VALUE=DATE layout = %q", d.layout)
	}

	// time layout but value has date-only length → auto date layout
	d = parseDate("DTSTART", nil, "20240601")
	if d.layout != LayoutDate {
		t.Errorf("date-only heuristic layout = %q, want %q", d.layout, LayoutDate)
	}
	if d.IsZero() {
		t.Error("date-only heuristic should parse successfully")
	}

	// unparseable value → zero time, key preserved
	d = parseDate("DTEND", nil, "not-a-date")
	if !d.IsZero() {
		t.Errorf("invalid value should yield zero time, got %v", d.Time)
	}
	if d.key != "DTEND" {
		t.Errorf("key = %q, want DTEND", d.key)
	}
}

func TestParseDateListEdgeCases(t *testing.T) {
	// mixed valid/invalid entries: only valid ones survive
	dl := parseDateList("RDATE", nil, "20240601T100000Z,bogus,20240601T110000Z")
	if len(dl.Dates) != 2 {
		t.Errorf("dates = %d, want 2 valid entries", len(dl.Dates))
	}

	// date-only first element switches the layout
	dl = parseDateList("EXDATE", nil, "20240601,20240602")
	if dl.layout != LayoutDate {
		t.Errorf("layout = %q, want %q", dl.layout, LayoutDate)
	}
	if len(dl.Dates) != 2 {
		t.Errorf("dates = %d, want 2", len(dl.Dates))
	}

	// explicit VALUE=DATE param
	dl = parseDateList("RDATE", []string{DateFormat}, "20240601")
	if dl.layout != LayoutDate || len(dl.Dates) != 1 {
		t.Errorf("VALUE=DATE: layout=%q dates=%d", dl.layout, len(dl.Dates))
	}
}

func TestParseGeoEdgeCases(t *testing.T) {
	if g := parseGeo("1.5"); g != (Geo{}) {
		t.Errorf("single part should yield zero Geo, got %+v", g)
	}
	if g := parseGeo("1;2;3"); g != (Geo{}) {
		t.Errorf("three parts should yield zero Geo, got %+v", g)
	}
	if g := parseGeo("lat;lon"); g != (Geo{}) {
		t.Errorf("non-numeric parts should yield zero Geo, got %+v", g)
	}
	if g := parseGeo("39.9;116.4"); g.Lat != 39.9 || g.Lon != 116.4 {
		t.Errorf("valid geo = %+v", g)
	}
}

type errScannerReader struct{}

func (errScannerReader) Read(_ []byte) (int, error) { return 0, errors.New("read failed") }

func TestParseReaderReadError(t *testing.T) {
	if _, err := ParseReader(errScannerReader{}); err == nil {
		t.Error("failing reader should return error")
	}
}

func TestParseEmptyAndGarbage(t *testing.T) {
	// empty input yields an empty calendar without error
	c, err := ParseReader(strings.NewReader(""))
	if err != nil {
		t.Fatalf("empty input err = %v", err)
	}
	if c == nil {
		t.Fatal("empty input calendar is nil")
	}

	// garbage lines are skipped without error
	c, err = ParseReader(strings.NewReader("RANDOM JUNK" + "\n" + "ANOTHER:line" + "\n"))
	if err != nil {
		t.Fatalf("garbage err = %v", err)
	}
	if len(c.Events()) != 0 {
		t.Errorf("garbage should not create events, got %d", len(c.Events()))
	}
}
