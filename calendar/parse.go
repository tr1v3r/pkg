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

// ICS property name constants.
const (
	propDTSTART      = "DTSTART"
	propDTEND        = "DTEND"
	propDTSTAMP      = "DTSTAMP"
	propCREATED      = "CREATED"
	propLASTMODIFIED = "LAST-MODIFIED"
	propUID          = "UID"
	propCLASS        = "CLASS"
	propDESC         = "DESCRIPTION"
	propLOCATION     = "LOCATION"
	propSEQUENCE     = "SEQUENCE"
	propSTATUS       = "STATUS"
	propSUMMARY      = "SUMMARY"
	propTRANSP       = "TRANSP"
	propRRULE        = "RRULE"
	propORGANIZER    = "ORGANIZER"
	propATTENDEE     = "ATTENDEE"
	propCATEGORIES   = "CATEGORIES"
	propPRIORITY     = "PRIORITY"
	propURL          = "URL"
	propDURATION     = "DURATION"
	propEXDATE       = "EXDATE"
	propRDATE        = "RDATE"
	propRECURRENCEID = "RECURRENCE-ID"
	propGEO          = "GEO"
	propCOMMENT      = "COMMENT"
	propCONTACT      = "CONTACT"
	propRELATEDTO    = "RELATED-TO"
	propRESOURCES    = "RESOURCES"
	propATTACH       = "ATTACH"
	propACTION       = "ACTION"
	propTRIGGER      = "TRIGGER"
	propREPEAT       = "REPEAT"
	propTZID         = "TZID"
	propTZOFFSETFROM = "TZOFFSETFROM"
	propTZOFFSETTO   = "TZOFFSETTO"
	propTZNAME       = "TZNAME"
	propDUE          = "DUE"
	propCOMPLETED    = "COMPLETED"
	propPERCENTCOMP  = "PERCENT-COMPLETE"
	propPRODID       = "PRODID"
	propVERSION      = "VERSION"
	propCALSCALE     = "CALSCALE"
	propMETHOD       = "METHOD"
	propXWRCALNAME   = "X-WR-CALNAME"
	propXWRCALDESC   = "X-WR-CALDESC"
	propXWRTIMEZONE  = "X-WR-TIMEZONE"

	// ICS control tokens.
	tokenBEGIN = "BEGIN"
	tokenEND   = "END"
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

	var (
		event     *Event
		inEvent   bool
		alarm     *Alarm
		inAlarm   bool
		tz        *Timezone
		inTZ      bool
		tzProp    *TimezoneProp
		inTZProp  bool
		todo      *Todo
		inTodo    bool
		journal   *Journal
		inJournal bool
		skipDepth int
	)

	for _, line := range lines {
		if line == "" {
			continue
		}

		propName, params, value := parseProperty(line)

		switch propName {
		case tokenBEGIN:
			switch value {
			case CompVCALENDAR:
				cal.header = Header(value)
			case CompVEVENT:
				event = &Event{header: Header(value), tailer: CompVEVENT}
				inEvent = true
			case CompVTODO:
				todo = &Todo{header: Header(value), tailer: CompVTODO}
				inTodo = true
			case CompVJOURNAL:
				journal = &Journal{header: Header(value), tailer: CompVJOURNAL}
				inJournal = true
			case CompVTIMEZONE:
				tz = &Timezone{}
				inTZ = true
			case CompVALARM:
				alarm = &Alarm{}
				inAlarm = true
			case CompSTANDARD, CompDAYLIGHT:
				tzProp = &TimezoneProp{Kind: value}
				inTZProp = true
			default:
				skipDepth++
			}
			continue

		case tokenEND:
			switch value {
			case CompVCALENDAR:
				cal.tailer = Tailer(value)
			case CompVEVENT:
				cal.events = append(cal.events, *event)
				inEvent = false
				event = nil
			case CompVTODO:
				cal.todos = append(cal.todos, *todo)
				inTodo = false
				todo = nil
			case CompVJOURNAL:
				cal.journals = append(cal.journals, *journal)
				inJournal = false
				journal = nil
			case CompVTIMEZONE:
				cal.timezones = append(cal.timezones, *tz)
				inTZ = false
				tz = nil
			case CompVALARM:
				if inEvent {
					event.alarms = append(event.alarms, *alarm)
				} else if inTodo {
					todo.alarms = append(todo.alarms, *alarm)
				}
				inAlarm = false
				alarm = nil
			case CompSTANDARD:
				tz.Standard = append(tz.Standard, *tzProp)
				inTZProp = false
				tzProp = nil
			case CompDAYLIGHT:
				tz.Daylight = append(tz.Daylight, *tzProp)
				inTZProp = false
				tzProp = nil
			default:
				if skipDepth > 0 {
					skipDepth--
				}
			}
			continue
		}

		if skipDepth > 0 {
			continue
		}

		// Dispatch to the current context
		switch {
		case inAlarm:
			setAlarmField(alarm, propName, params, value)
		case inTZProp:
			setTZPropField(tzProp, propName, params, value)
		case inEvent:
			setEventField(event, propName, params, value)
		case inTZ:
			setTimezoneField(tz, propName, params, value)
		case inTodo:
			setTodoField(todo, propName, params, value)
		case inJournal:
			setJournalField(journal, propName, params, value)
		default:
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
	case propPRODID:
		cal.prodID = ProdID(value)
	case propVERSION:
		cal.version = Version(value)
	case propCALSCALE:
		cal.scale = Scale(value)
	case propMETHOD:
		cal.method = Method(value)
	case propXWRCALNAME:
		cal.name = CalName(value)
	case propXWRCALDESC:
		cal.desc = CalDesc(value)
	case propXWRTIMEZONE:
		cal.timeZone = TimeZone(value)
	}
}

func setEventField(event *Event, propName string, params []string, value string) {
	switch propName {
	case propDTSTART:
		event.start = parseDate(propDTSTART, params, value)
	case propDTEND:
		event.end = parseDate(propDTEND, params, value)
	case propDTSTAMP:
		event.stamp = parseDate(propDTSTAMP, params, value)
	case propCREATED:
		event.createdAt = parseDate(propCREATED, params, value)
	case propLASTMODIFIED:
		event.modifiedAt = parseDate(propLASTMODIFIED, params, value)
	case propUID:
		event.uid = UID(value)
	case propCLASS:
		event.class = Class(value)
	case propDESC:
		event.desc = Desc(value)
	case propLOCATION:
		event.location = Location(value)
	case propSEQUENCE:
		if n, err := strconv.Atoi(value); err == nil {
			event.sequence = Sequence(n)
		}
	case propSTATUS:
		event.status = Status(value)
	case propSUMMARY:
		event.summary = Summary(value)
	case propTRANSP:
		event.transparent = Transparent(value)
	case propRRULE:
		event.rrule = RRULE(value)
	default:
		setEventFieldExt(event, propName, params, value)
	}
}

func setEventFieldExt(event *Event, propName string, params []string, value string) {
	switch propName {
	case propORGANIZER:
		event.organizer = Organizer{params: params, URI: value}
	case propATTENDEE:
		event.attendees = append(event.attendees, Attendee{params: params, URI: value})
	case propCATEGORIES:
		event.categories = strings.Split(value, ",")
	case propPRIORITY:
		if n, err := strconv.Atoi(value); err == nil {
			event.priority = Priority(n)
		}
	case propURL:
		event.url = URL(value)
	case propDURATION:
		event.duration = Duration(value)
	case propEXDATE:
		event.exdates = append(event.exdates, parseDateList(propEXDATE, params, value))
	case propRDATE:
		event.rdates = append(event.rdates, parseDateList(propRDATE, params, value))
	case propRECURRENCEID:
		event.recurrenceID = parseDate(propRECURRENCEID, params, value)
	case propGEO:
		event.geo = parseGeo(value)
	case propCOMMENT:
		event.comment = Comment(value)
	case propCONTACT:
		event.contact = Contact(value)
	case propRELATEDTO:
		event.relatedTo = RelatedTo(value)
	case propRESOURCES:
		event.resources = Resources(value)
	case propATTACH:
		event.attachments = append(event.attachments, Attachment{params: params, URI: value})
	}
}

func setAlarmField(alarm *Alarm, propName string, params []string, value string) {
	switch propName {
	case propACTION:
		alarm.Action = value
	case propTRIGGER:
		alarm.Trigger = value
	case propDESC:
		alarm.Desc = value
	case propSUMMARY:
		alarm.Summary = value
	case propDURATION:
		alarm.Duration = value
	case propREPEAT:
		if n, err := strconv.Atoi(value); err == nil {
			alarm.Repeat = n
		}
	case propATTENDEE:
		alarm.Attendees = append(alarm.Attendees, Attendee{params: params, URI: value})
	case propATTACH:
		alarm.Attachments = append(alarm.Attachments, Attachment{params: params, URI: value})
	}
}

func setTimezoneField(tz *Timezone, propName string, params []string, value string) {
	switch propName {
	case propTZID:
		tz.TZID = value
	case propLASTMODIFIED:
		tz.LastModified = parseDate(propLASTMODIFIED, params, value)
	}
}

func setTZPropField(tp *TimezoneProp, propName string, params []string, value string) {
	switch propName {
	case propDTSTART:
		tp.DTStart = parseDate(propDTSTART, params, value)
	case propTZOFFSETFROM:
		tp.TZOffsetFrom = value
	case propTZOFFSETTO:
		tp.TZOffsetTo = value
	case propRRULE:
		tp.RRULE = value
	case propTZNAME:
		tp.TZName = value
	case propCOMMENT:
		tp.Comment = value
	}
}

func setTodoField(todo *Todo, propName string, params []string, value string) {
	switch propName {
	case propDTSTAMP:
		todo.stamp = parseDate(propDTSTAMP, params, value)
	case propUID:
		todo.uid = UID(value)
	case propDTSTART:
		todo.start = parseDate(propDTSTART, params, value)
	case propDUE:
		todo.due = parseDate(propDUE, params, value)
	case propDURATION:
		todo.duration = Duration(value)
	case propSUMMARY:
		todo.summary = Summary(value)
	case propDESC:
		todo.desc = Desc(value)
	case propPRIORITY:
		if n, err := strconv.Atoi(value); err == nil {
			todo.priority = Priority(n)
		}
	case propSTATUS:
		todo.status = TodoStatus(value)
	case propSEQUENCE:
		if n, err := strconv.Atoi(value); err == nil {
			todo.seq = Sequence(n)
		}
	case propCLASS:
		todo.class = Class(value)
	case propCATEGORIES:
		todo.categories = strings.Split(value, ",")
	case propCOMPLETED:
		todo.completed = parseDate(propCOMPLETED, params, value)
	case propPERCENTCOMP:
		if n, err := strconv.Atoi(value); err == nil {
			todo.percent = n
		}
	case propLOCATION:
		todo.location = Location(value)
	case propORGANIZER:
		todo.organizer = Organizer{params: params, URI: value}
	case propATTENDEE:
		todo.attendees = append(todo.attendees, Attendee{params: params, URI: value})
	case propURL:
		todo.url = URL(value)
	case propRRULE:
		todo.rrule = RRULE(value)
	case propEXDATE:
		todo.exdates = append(todo.exdates, parseDateList(propEXDATE, params, value))
	case propRDATE:
		todo.rdates = append(todo.rdates, parseDateList(propRDATE, params, value))
	}
}

func setJournalField(journal *Journal, propName string, params []string, value string) {
	switch propName {
	case propDTSTAMP:
		journal.stamp = parseDate(propDTSTAMP, params, value)
	case propUID:
		journal.uid = UID(value)
	case propDTSTART:
		journal.start = parseDate(propDTSTART, params, value)
	case propSUMMARY:
		journal.summary = Summary(value)
	case propDESC:
		journal.desc = Desc(value)
	case propCLASS:
		journal.class = Class(value)
	case propCATEGORIES:
		journal.categories = strings.Split(value, ",")
	case propSTATUS:
		journal.status = JournalStatus(value)
	case propURL:
		journal.url = URL(value)
	case propORGANIZER:
		journal.organizer = Organizer{params: params, URI: value}
	case propATTENDEE:
		journal.attendees = append(journal.attendees, Attendee{params: params, URI: value})
	}
}

// parseDate parses a Date from ICS value with optional params.
func parseDate(key string, params []string, value string) Date {
	layout := LayoutTime
	configs := params

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

// parseDateList parses a list of dates for EXDATE or RDATE.
func parseDateList(key string, params []string, value string) DateList {
	layout := LayoutTime
	if slices.Contains(params, DateFormat) {
		layout = LayoutDate
	}

	dateStrs := strings.Split(value, ",")
	if layout == LayoutTime && len(dateStrs) > 0 && len(dateStrs[0]) == len(LayoutDate) {
		layout = LayoutDate
	}

	var times []time.Time
	for _, d := range dateStrs {
		t, err := time.Parse(layout, d)
		if err == nil {
			times = append(times, t)
		}
	}

	return DateList{key: key, layout: layout, configs: params, Dates: times}
}

// parseGeo parses GEO value "lat;lon".
func parseGeo(value string) Geo {
	parts := strings.Split(value, ";")
	if len(parts) != 2 {
		return Geo{}
	}
	lat, _ := strconv.ParseFloat(parts[0], 64)
	lon, _ := strconv.ParseFloat(parts[1], 64)
	return Geo{Lat: lat, Lon: lon}
}
