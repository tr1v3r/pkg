package calendar

import (
	"bytes"
	"fmt"
	"time"
)

const (
	generalProdID  = "-//True R1v3r//R1v3r Calendar 1.0//CN"
	generalVersion = "2.0"
)

type Calendar struct {
	header    Header
	prodID    ProdID
	version   Version
	scale     Scale
	method    Method
	name      CalName
	timeZone  TimeZone
	desc      CalDesc
	timezones []Timezone
	events    []Event
	todos     []Todo
	journals  []Journal
	tailer    Tailer
}

// NewCalendar build new Calendar
func NewCalendar(name, desc string, opts ...CalendarOption) *Calendar {
	c := &Calendar{
		header: CompVCALENDAR,
		tailer: CompVCALENDAR,

		prodID:  generalProdID,
		version: generalVersion,
		scale:   ScaleGregorian,
		method:  MethodPublish,

		name: CalName(name),
		desc: CalDesc(desc),
	}

	for _, opt := range opts {
		c = opt(c)
	}

	return c
}

func (c *Calendar) AddEvents(events ...Event)    { c.events = append(c.events, events...) }
func (c *Calendar) AddTimezones(tzs ...Timezone) { c.timezones = append(c.timezones, tzs...) }
func (c *Calendar) AddTodos(todos ...Todo)       { c.todos = append(c.todos, todos...) }
func (c *Calendar) AddJournals(js ...Journal)    { c.journals = append(c.journals, js...) }

func (c *Calendar) Events() []Event       { return c.events }
func (c *Calendar) Timezones() []Timezone { return c.timezones }
func (c *Calendar) Todos() []Todo         { return c.todos }
func (c *Calendar) Journals() []Journal   { return c.journals }

func (c *Calendar) Output() []byte {
	var buf bytes.Buffer

	buf.Write(c.header.Output())
	buf.WriteByte('\n')

	if c.prodID != "" {
		buf.Write(c.prodID.Output())
		buf.WriteByte('\n')
	}

	buf.Write(c.version.Output())
	buf.WriteByte('\n')

	if c.scale != "" {
		buf.Write(c.scale.Output())
		buf.WriteByte('\n')
	}
	if c.method != "" {
		buf.Write(c.method.Output())
		buf.WriteByte('\n')
	}
	if c.name != "" {
		buf.Write(c.name.Output())
		buf.WriteByte('\n')
	}
	if c.timeZone != "" {
		buf.Write(c.timeZone.Output())
		buf.WriteByte('\n')
	}
	if c.desc != "" {
		buf.Write(c.desc.Output())
		buf.WriteByte('\n')
	}

	for _, tz := range c.timezones {
		buf.Write(tz.Output())
		buf.WriteByte('\n')
	}
	for _, event := range c.events {
		buf.Write(event.Output())
	}
	for _, todo := range c.todos {
		buf.Write(todo.Output())
	}
	for _, j := range c.journals {
		buf.Write(j.Output())
	}

	buf.Write(c.tailer.Output())

	return buf.Bytes()
}

// NewEvent build new calendar event
func NewEvent(sum, description string, start time.Time, opts ...EventOption) *Event {
	e := &Event{
		header: CompVEVENT,
		tailer: CompVEVENT,

		start:   NewDate("DTSTART", start),
		summary: Summary(sum),
		desc:    Desc(description),

		createdAt:   NewDate("CREATED", time.Now()),
		transparent: TranspTransparent,
	}

	for _, opt := range opts {
		e = opt(e)
	}

	return e
}

type Event struct {
	header       Header
	start        Date
	end          Date
	stamp        Date
	uid          UID
	class        Class
	createdAt    Date
	modifiedAt   Date
	location     Location
	sequence     Sequence
	status       Status
	summary      Summary
	desc         Desc
	transparent  Transparent
	rrule        RRULE
	organizer    Organizer
	attendees    []Attendee
	categories   Categories
	priority     Priority
	url          URL
	duration     Duration
	exdates      []DateList
	rdates       []DateList
	recurrenceID Date
	geo          Geo
	comment      Comment
	contact      Contact
	relatedTo    RelatedTo
	resources    Resources
	attachments  []Attachment
	alarms       []Alarm
	tailer       Tailer
}

func (e *Event) Output() []byte {
	var buf bytes.Buffer

	buf.Write(e.header.Output())
	buf.WriteByte('\n')

	if !e.start.IsZero() {
		buf.Write(e.start.Output())
		buf.WriteByte('\n')
	}
	if e.rrule != "" {
		buf.Write(e.rrule.Output())
		buf.WriteByte('\n')
	}
	if !e.end.IsZero() {
		buf.Write(e.end.Output())
		buf.WriteByte('\n')
	}
	if !e.stamp.IsZero() {
		buf.Write(e.stamp.Output())
		buf.WriteByte('\n')
	}
	if e.uid != "" {
		buf.Write(e.uid.Output())
		buf.WriteByte('\n')
	}
	if e.class != "" {
		buf.Write(e.class.Output())
		buf.WriteByte('\n')
	}
	if !e.createdAt.IsZero() {
		buf.Write(e.createdAt.Output())
		buf.WriteByte('\n')
	}
	if e.desc != "" {
		buf.Write(e.desc.Output())
		buf.WriteByte('\n')
	}
	if !e.modifiedAt.IsZero() {
		buf.Write(e.modifiedAt.Output())
		buf.WriteByte('\n')
	}
	if e.location != "" {
		buf.Write(e.location.Output())
		buf.WriteByte('\n')
	}

	buf.Write(e.sequence.Output())
	buf.WriteByte('\n')

	if e.status != "" {
		buf.Write(e.status.Output())
		buf.WriteByte('\n')
	}
	if e.summary != "" {
		buf.Write(e.summary.Output())
		buf.WriteByte('\n')
	}
	if e.transparent != "" {
		buf.Write(e.transparent.Output())
		buf.WriteByte('\n')
	}
	e.outputRFC5545(&buf)

	buf.Write(e.tailer.Output())
	buf.WriteByte('\n')

	return buf.Bytes()
}

// outputRFC5545 writes RFC 5545 extended fields to buf.
func (e *Event) outputRFC5545(buf *bytes.Buffer) {
	if e.organizer.URI != "" {
		buf.Write(e.organizer.Output())
		buf.WriteByte('\n')
	}
	for _, att := range e.attendees {
		buf.Write(att.Output())
		buf.WriteByte('\n')
	}
	if len(e.categories) > 0 {
		buf.Write(e.categories.Output())
		buf.WriteByte('\n')
	}
	if e.priority != 0 {
		buf.Write(e.priority.Output())
		buf.WriteByte('\n')
	}
	if e.url != "" {
		buf.Write(e.url.Output())
		buf.WriteByte('\n')
	}
	if e.duration != "" {
		buf.Write(e.duration.Output())
		buf.WriteByte('\n')
	}
	for _, dl := range e.exdates {
		buf.Write(dl.Output())
		buf.WriteByte('\n')
	}
	for _, dl := range e.rdates {
		buf.Write(dl.Output())
		buf.WriteByte('\n')
	}
	if !e.recurrenceID.IsZero() {
		buf.Write(e.recurrenceID.Output())
		buf.WriteByte('\n')
	}
	if e.geo.Lat != 0 || e.geo.Lon != 0 {
		buf.Write(e.geo.Output())
		buf.WriteByte('\n')
	}
	if e.comment != "" {
		buf.Write(e.comment.Output())
		buf.WriteByte('\n')
	}
	if e.contact != "" {
		buf.Write(e.contact.Output())
		buf.WriteByte('\n')
	}
	if e.relatedTo != "" {
		buf.Write(e.relatedTo.Output())
		buf.WriteByte('\n')
	}
	if e.resources != "" {
		buf.Write(e.resources.Output())
		buf.WriteByte('\n')
	}
	for _, att := range e.attachments {
		buf.Write(att.Output())
		buf.WriteByte('\n')
	}
	for _, alarm := range e.alarms {
		buf.Write(alarm.Output())
		buf.WriteByte('\n')
	}
}

// ============== VTIMEZONE ==============

// Timezone represents a VTIMEZONE component.
type Timezone struct {
	TZID         string
	LastModified Date
	Standard     []TimezoneProp
	Daylight     []TimezoneProp
}

// TimezoneProp represents STANDARD or DAYLIGHT sub-component of VTIMEZONE.
type TimezoneProp struct {
	Kind         string // CompSTANDARD or CompDAYLIGHT
	DTStart      Date
	TZOffsetFrom string
	TZOffsetTo   string
	RRULE        string
	TZName       string
	Comment      string
}

func (tz Timezone) Output() []byte {
	var buf bytes.Buffer
	buf.WriteString("BEGIN:VTIMEZONE\n")
	buf.WriteString("TZID:")
	buf.WriteString(tz.TZID)
	buf.WriteByte('\n')
	if !tz.LastModified.IsZero() {
		buf.Write(tz.LastModified.Output())
		buf.WriteByte('\n')
	}
	for _, s := range tz.Standard {
		buf.Write(s.Output())
		buf.WriteByte('\n')
	}
	for _, d := range tz.Daylight {
		buf.Write(d.Output())
		buf.WriteByte('\n')
	}
	buf.WriteString("END:VTIMEZONE")
	return buf.Bytes()
}

func (tp TimezoneProp) Output() []byte {
	var buf bytes.Buffer
	buf.WriteString("BEGIN:")
	buf.WriteString(tp.Kind)
	buf.WriteByte('\n')
	if !tp.DTStart.IsZero() {
		buf.Write(tp.DTStart.Output())
		buf.WriteByte('\n')
	}
	if tp.TZOffsetFrom != "" {
		buf.WriteString("TZOFFSETFROM:")
		buf.WriteString(tp.TZOffsetFrom)
		buf.WriteByte('\n')
	}
	if tp.TZOffsetTo != "" {
		buf.WriteString("TZOFFSETTO:")
		buf.WriteString(tp.TZOffsetTo)
		buf.WriteByte('\n')
	}
	if tp.RRULE != "" {
		buf.WriteString("RRULE:")
		buf.WriteString(tp.RRULE)
		buf.WriteByte('\n')
	}
	if tp.TZName != "" {
		buf.WriteString("TZNAME:")
		buf.WriteString(tp.TZName)
		buf.WriteByte('\n')
	}
	if tp.Comment != "" {
		buf.WriteString("COMMENT:")
		buf.WriteString(tp.Comment)
		buf.WriteByte('\n')
	}
	buf.WriteString("END:")
	buf.WriteString(tp.Kind)
	return buf.Bytes()
}

// ============== VALARM ==============

// Alarm represents a VALARM component.
type Alarm struct {
	Action      string // AUDIO, DISPLAY, EMAIL
	Trigger     string // duration like "-PT30M" or absolute datetime
	Desc        string // for DISPLAY/EMAIL
	Summary     string // for EMAIL
	Duration    string // repeat interval
	Repeat      int    // repeat count
	Attendees   []Attendee
	Attachments []Attachment
}

func (a Alarm) Output() []byte {
	var buf bytes.Buffer
	buf.WriteString("BEGIN:VALARM\n")
	if a.Action != "" {
		buf.WriteString("ACTION:")
		buf.WriteString(a.Action)
		buf.WriteByte('\n')
	}
	if a.Trigger != "" {
		buf.WriteString("TRIGGER:")
		buf.WriteString(a.Trigger)
		buf.WriteByte('\n')
	}
	if a.Desc != "" {
		buf.WriteString("DESCRIPTION:")
		buf.WriteString(a.Desc)
		buf.WriteByte('\n')
	}
	if a.Summary != "" {
		buf.WriteString("SUMMARY:")
		buf.WriteString(a.Summary)
		buf.WriteByte('\n')
	}
	if a.Duration != "" {
		buf.WriteString("DURATION:")
		buf.WriteString(a.Duration)
		buf.WriteByte('\n')
	}
	if a.Repeat > 0 {
		buf.WriteString("REPEAT:")
		buf.WriteString(fmt.Sprint(a.Repeat))
		buf.WriteByte('\n')
	}
	for _, att := range a.Attendees {
		buf.Write(att.Output())
		buf.WriteByte('\n')
	}
	for _, att := range a.Attachments {
		buf.Write(att.Output())
		buf.WriteByte('\n')
	}
	buf.WriteString("END:VALARM")
	return buf.Bytes()
}

// ============== VTODO ==============

// Todo represents a VTODO component.
type Todo struct {
	header     Header
	tailer     Tailer
	stamp      Date
	uid        UID
	start      Date
	due        Date
	duration   Duration
	summary    Summary
	desc       Desc
	priority   Priority
	status     TodoStatus
	seq        Sequence
	class      Class
	categories Categories
	completed  Date
	percent    int
	location   Location
	organizer  Organizer
	attendees  []Attendee
	url        URL
	rrule      RRULE
	exdates    []DateList
	rdates     []DateList
	alarms     []Alarm
}

func (t Todo) Output() []byte {
	var buf bytes.Buffer
	buf.Write(t.header.Output())
	buf.WriteByte('\n')

	if !t.stamp.IsZero() {
		buf.Write(t.stamp.Output())
		buf.WriteByte('\n')
	}
	if t.uid != "" {
		buf.Write(t.uid.Output())
		buf.WriteByte('\n')
	}
	if !t.start.IsZero() {
		buf.Write(t.start.Output())
		buf.WriteByte('\n')
	}
	if !t.due.IsZero() {
		buf.Write(t.due.Output())
		buf.WriteByte('\n')
	}
	if t.duration != "" {
		buf.Write(t.duration.Output())
		buf.WriteByte('\n')
	}
	if t.summary != "" {
		buf.Write(t.summary.Output())
		buf.WriteByte('\n')
	}
	if t.desc != "" {
		buf.Write(t.desc.Output())
		buf.WriteByte('\n')
	}
	if t.priority != 0 {
		buf.Write(t.priority.Output())
		buf.WriteByte('\n')
	}
	if t.status != "" {
		buf.Write(t.status.Output())
		buf.WriteByte('\n')
	}
	buf.Write(t.seq.Output())
	buf.WriteByte('\n')
	if t.class != "" {
		buf.Write(t.class.Output())
		buf.WriteByte('\n')
	}
	if len(t.categories) > 0 {
		buf.Write(t.categories.Output())
		buf.WriteByte('\n')
	}
	if !t.completed.IsZero() {
		buf.Write(t.completed.Output())
		buf.WriteByte('\n')
	}
	if t.percent > 0 {
		buf.WriteString("PERCENT-COMPLETE:")
		buf.WriteString(fmt.Sprint(t.percent))
		buf.WriteByte('\n')
	}
	if t.location != "" {
		buf.Write(t.location.Output())
		buf.WriteByte('\n')
	}
	if t.organizer.URI != "" {
		buf.Write(t.organizer.Output())
		buf.WriteByte('\n')
	}
	for _, att := range t.attendees {
		buf.Write(att.Output())
		buf.WriteByte('\n')
	}
	if t.url != "" {
		buf.Write(t.url.Output())
		buf.WriteByte('\n')
	}
	if t.rrule != "" {
		buf.Write(t.rrule.Output())
		buf.WriteByte('\n')
	}
	for _, dl := range t.exdates {
		buf.Write(dl.Output())
		buf.WriteByte('\n')
	}
	for _, dl := range t.rdates {
		buf.Write(dl.Output())
		buf.WriteByte('\n')
	}
	for _, alarm := range t.alarms {
		buf.Write(alarm.Output())
		buf.WriteByte('\n')
	}

	buf.Write(t.tailer.Output())
	buf.WriteByte('\n')
	return buf.Bytes()
}

// ============== VJOURNAL ==============

// Journal represents a VJOURNAL component.
type Journal struct {
	header     Header
	tailer     Tailer
	stamp      Date
	uid        UID
	start      Date
	summary    Summary
	desc       Desc
	class      Class
	categories Categories
	status     JournalStatus
	url        URL
	organizer  Organizer
	attendees  []Attendee
}

func (j Journal) Output() []byte {
	var buf bytes.Buffer
	buf.Write(j.header.Output())
	buf.WriteByte('\n')

	if !j.stamp.IsZero() {
		buf.Write(j.stamp.Output())
		buf.WriteByte('\n')
	}
	if j.uid != "" {
		buf.Write(j.uid.Output())
		buf.WriteByte('\n')
	}
	if !j.start.IsZero() {
		buf.Write(j.start.Output())
		buf.WriteByte('\n')
	}
	if j.summary != "" {
		buf.Write(j.summary.Output())
		buf.WriteByte('\n')
	}
	if j.desc != "" {
		buf.Write(j.desc.Output())
		buf.WriteByte('\n')
	}
	if j.class != "" {
		buf.Write(j.class.Output())
		buf.WriteByte('\n')
	}
	if len(j.categories) > 0 {
		buf.Write(j.categories.Output())
		buf.WriteByte('\n')
	}
	if j.status != "" {
		buf.Write(j.status.Output())
		buf.WriteByte('\n')
	}
	if j.url != "" {
		buf.Write(j.url.Output())
		buf.WriteByte('\n')
	}
	if j.organizer.URI != "" {
		buf.Write(j.organizer.Output())
		buf.WriteByte('\n')
	}
	for _, att := range j.attendees {
		buf.Write(att.Output())
		buf.WriteByte('\n')
	}

	buf.Write(j.tailer.Output())
	buf.WriteByte('\n')
	return buf.Bytes()
}
