package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestTextOutputIsEscaped(t *testing.T) {
	e := NewEvent("Title, with comma; and semi", "Line1\nLine2", time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC))

	out := string(e.Output())
	if !strings.Contains(out, "SUMMARY:Title\\, with comma\\; and semi") {
		t.Errorf("summary not escaped: %q", out)
	}
	if !strings.Contains(out, "DESCRIPTION:Line1\\nLine2") {
		t.Errorf("description newline not escaped: %q", out)
	}

	// full round trip survives
	cal, err := Parse([]byte(out))
	if err != nil {
		t.Fatalf("re-parse fail: %s", err)
	}
	events := cal.Events()
	if len(events) != 1 {
		t.Fatalf("round-trip events = %d", len(events))
	}
	if string(events[0].summary) != "Title, with comma; and semi" {
		t.Errorf("round-trip summary = %q", events[0].summary)
	}
	if string(events[0].desc) != "Line1\nLine2" {
		t.Errorf("round-trip description = %q", events[0].desc)
	}
}

func TestFoldLinesRFC5545(t *testing.T) {
	long := strings.Repeat("x", 200)
	data := []byte("BEGIN:VCALENDAR\nSUMMARY:" + long + "\nEND:VCALENDAR\n")

	folded := foldLines(data)

	// No line (after splitting on \n) may exceed 75 octets.
	for _, line := range strings.Split(string(folded), "\n") {
		if len(line) > 75 {
			t.Errorf("line %q... is %d octets, exceeds 75", line[:40], len(line))
		}
	}

	// Unfolding (parse-side) must restore the original.
	unfolded := unfoldLines(strings.Split(string(folded), "\n"))
	joined := strings.Join(unfolded, "\n") + "\n"
	if joined != string(data) {
		t.Errorf("unfold mismatch:\n got %q\nwant %q", joined, string(data))
	}
}

func TestFoldLinesMultibyteSafe(t *testing.T) {
	// 40 CJK chars = 120 octets; folding must not split a rune.
	long := strings.Repeat("好", 40)
	data := []byte("SUMMARY:" + long + "\n")

	folded := string(foldLines(data))

	// every continuation boundary must sit on a rune start: round-trip restores
	unfolded := unfoldLines(strings.Split(folded, "\n"))
	if strings.Join(unfolded, "\n")+"\n" != string(data) {
		t.Error("multibyte fold does not round-trip")
	}
	// and the folded form contains no replacement chars
	if strings.ContainsRune(folded, 0xFFFD) {
		t.Error("fold split a rune")
	}
}

func TestCalendarOutputAllLinesFolded(t *testing.T) {
	c := NewCalendar("cal", strings.Repeat("long desc ", 30))
	c.AddEvents(*NewEvent(strings.Repeat("long summary ", 20), "d",
		time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)))

	out := string(c.Output())
	for _, line := range strings.Split(out, "\n") {
		if len(line) > 75 {
			t.Errorf("calendar output line exceeds 75 octets: %d (%q...)", len(line), line[:min(len(line), 40)])
		}
	}

	// still parses back
	if _, err := Parse([]byte(out)); err != nil {
		t.Fatalf("folded calendar re-parse fail: %s", err)
	}
}
