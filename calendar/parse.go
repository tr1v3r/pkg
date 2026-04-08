package calendar

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Parse parses ICS content into a Calendar.
func Parse(data []byte) (*Calendar, error) {
	return ParseReader(bytes.NewReader(data))
}

// ParseReader reads ICS content from an io.Reader and returns a Calendar.
func ParseReader(r io.Reader) (*Calendar, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read lines: %w", err)
	}

	lines = unfoldLines(lines)
	return parseCalendar(lines)
}

// unfoldLines joins continuation lines per RFC 5545 §3.1.
// A line starting with space or tab is a continuation of the previous line.
func unfoldLines(lines []string) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(result) > 0 && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			result[len(result)-1] += strings.TrimLeft(line, " \t")
			continue
		}
		result = append(result, line)
	}
	return result
}

func parseCalendar(lines []string) (*Calendar, error) {
	cal := &Calendar{}
	var event *Event
	var inEvent bool
	var skipDepth int // depth counter for skipping unknown components

	for _, line := range lines {
		if line == "" {
			continue
		}

		propName, params, value := parseProperty(line)

		// Handle BEGIN/END
		switch propName {
		case "BEGIN":
			switch value {
			case "VCALENDAR":
				cal.header = Header(value)
			case "VEVENT":
				if skipDepth == 0 {
					event = &Event{header: Header(value), tailer: "VEVENT"}
					inEvent = true
				} else {
					skipDepth++
				}
			default:
				// Skip unknown components (VTIMEZONE, VALARM, etc.)
				skipDepth++
			}
			continue

		case "END":
			switch value {
			case "VCALENDAR":
				cal.tailer = Tailer(value)
			case "VEVENT":
				if skipDepth == 0 && inEvent {
					cal.events = append(cal.events, *event)
					inEvent = false
					event = nil
				} else if skipDepth > 0 {
					skipDepth--
				}
			default:
				if skipDepth > 0 {
					skipDepth--
				}
			}
			continue
		}

		// Skip properties inside unknown components
		if skipDepth > 0 {
			continue
		}

		// Dispatch to calendar or event
		if inEvent {
			setEventField(event, propName, params, value)
		} else {
			setCalendarField(cal, propName, value)
		}
	}

	return cal, nil
}

// parseProperty splits an ICS content line into property name, parameters, and value.
// e.g. "DTSTART;VALUE=DATE:20240101" → ("DTSTART", ["VALUE=DATE"], "20240101")
func parseProperty(line string) (name string, params []string, value string) {
	keyPart, value, found := strings.Cut(line, ":")
	if !found {
		return line, nil, ""
	}

	parts := strings.Split(keyPart, ";")
	name = parts[0]
	if len(parts) > 1 {
		params = parts[1:]
	}

	return name, params, value
}

func setCalendarField(cal *Calendar, propName, value string) {
	switch propName {
	case "PRODID":
		cal.prodID = ProdID(value)
	case "VERSION":
		cal.version = Version(value)
	case "CALSCALE":
		cal.scale = Scale(value)
	case "METHOD":
		cal.method = Method(value)
	case "X-WR-CALNAME":
		cal.name = CalName(value)
	case "X-WR-CALDESC":
		cal.desc = CalDesc(value)
	case "X-WR-TIMEZONE":
		cal.timeZone = TimeZone(value)
	}
}

func setEventField(event *Event, propName string, params []string, value string) {
	switch propName {
	case "DTSTART":
		event.start = parseDate("DTSTART", params, value)
	case "DTEND":
		event.end = parseDate("DTEND", params, value)
	case "DTSTAMP":
		event.stamp = parseDate("DTSTAMP", params, value)
	case "CREATED":
		event.createdAt = parseDate("CREATED", params, value)
	case "LAST-MODIFIED":
		event.modifiedAt = parseDate("LAST-MODIFIED", params, value)
	case "UID":
		event.uid = UID(value)
	case "CLASS":
		event.class = Class(value)
	case "DESCRIPTION":
		event.desc = Desc(value)
	case "LOCATION":
		event.location = Location(value)
	case "SEQUENCE":
		if n, err := strconv.Atoi(value); err == nil {
			event.sequence = Sequence(n)
		}
	case "STATUS":
		event.status = Status(value)
	case "SUMMARY":
		event.summary = Summary(value)
	case "TRANSP":
		event.transparent = Transparent(value)
	case "RRULE":
		event.rrule = RRULE(value)
	}
}

// parseDate parses a Date from ICS value with optional params.
func parseDate(key string, params []string, value string) Date {
	layout := LayoutTime
	configs := params

	// Check for VALUE=DATE param or date-only value
	if slices.Contains(params, DateFormat) {
		layout = LayoutDate
	}
	if layout == LayoutTime && len(value) == len(LayoutDate) {
		layout = LayoutDate
	}

	t, err := time.Parse(layout, value)
	if err != nil {
		return Date{key: key, layout: layout, configs: configs}
	}

	return Date{key: key, layout: layout, configs: configs, Time: t}
}
