package calendar

import (
	"bytes"
	"fmt"
	"time"
)

const (
	generalProdID = "-//True R1v3r//R1v3r Calendar 1.0//CN"
	version       = "2.0"
)

type Calendar struct {
	prodID   string
	version  string
	scale    Scale
	method   Method
	name     string
	timeZone TimeZone
	desc     string

	events []Event
}

// NewCalendar build new Calendar
func NewCalendar(name, desc string, opts ...CalendarOption) *Calendar {
	c := &Calendar{
		prodID:  generalProdID,
		version: version,
		scale:   ScaleGregorian,
		method:  MethodPublish,

		name: name,
		desc: desc,
	}

	for _, opt := range opts {
		c = opt(c)
	}

	return c
}

func (c *Calendar) AddEvents(events ...Event) { c.events = append(c.events, events...) }

func (c *Calendar) Output() []byte {
	var buf bytes.Buffer

	buf.Write(header.With(tagCalendar))
	buf.WriteByte('\n')

	if c.prodID != "" {
		buf.Write(calProdID.With(c.prodID))
		buf.WriteByte('\n')
	}

	buf.Write(calVer.With(version))
	buf.WriteByte('\n')

	if c.scale != "" {
		buf.Write(calScale.With(string(c.scale)))
		buf.WriteByte('\n')
	}
	if c.method != "" {
		buf.Write(calMethod.With(string(c.method)))
		buf.WriteByte('\n')
	}
	if c.name != "" {
		buf.Write(calName.With(c.name))
		buf.WriteByte('\n')
	}
	if c.timeZone != "" {
		buf.Write(calTimeZone.With(string(c.timeZone)))
		buf.WriteByte('\n')
	}
	if c.desc != "" {
		buf.Write(calDesc.With(c.desc))
		buf.WriteByte('\n')
	}

	for _, event := range c.events {
		buf.Write(event.Output())
	}

	buf.Write(tailer.With(tagCalendar))

	return buf.Bytes()
}

// NewEvent build new calendar event
func NewEvent(summary, desc string, start, end time.Time, opts ...EventOption) *Event {
	e := &Event{
		Start:   start,
		End:     end,
		Summary: summary,
		Desc:    desc,

		CreatedAt: time.Now(),
	}

	for _, opt := range opts {
		e = opt(e)
	}

	return e
}

type Event struct {
	Start       time.Time
	End         time.Time
	Stamp       time.Time
	UID         string
	Class       Class
	CreatedAt   time.Time
	ModifiedAt  time.Time
	Location    string
	Sequence    int
	Status      Status
	Summary     string
	Desc        string
	Transparent Transparent
}

func (e *Event) Output() []byte {
	var buf bytes.Buffer

	buf.Write(header.With(tagEvent))
	buf.WriteByte('\n')

	if !e.Start.IsZero() {
		buf.Write(eventDateStart.With(fmt.Sprintf("VALUE=DATE:%s", e.Start.Format(dateLayout))))
		buf.WriteByte('\n')
	}
	if !e.End.IsZero() {
		buf.Write(eventDateEnd.With(fmt.Sprintf("VALUE=DATE:%s", e.End.Format(dateLayout))))
		buf.WriteByte('\n')
	}
	if !e.Stamp.IsZero() {
		buf.Write(eventDateStamp.With(e.Stamp.Format(timeLayout)))
		buf.WriteByte('\n')
	}
	if e.UID != "" {
		buf.Write(eventUID.With(e.UID))
		buf.WriteByte('\n')
	}
	if e.Class != "" {
		buf.Write(eventClass.With(string(e.Class)))
		buf.WriteByte('\n')
	}
	if !e.CreatedAt.IsZero() {
		buf.Write(eventCreatedAt.With(e.CreatedAt.Format(timeLayout)))
		buf.WriteByte('\n')
	}
	if e.Desc != "" {
		buf.Write(eventDesc.With(e.Desc))
		buf.WriteByte('\n')
	}
	if !e.ModifiedAt.IsZero() {
		buf.Write(eventModifiedAt.With(e.ModifiedAt.Format(timeLayout)))
		buf.WriteByte('\n')
	}
	if e.Location != "" {
		buf.Write(eventLocation.With(e.Location))
		buf.WriteByte('\n')
	}

	buf.Write(eventSequence.With(fmt.Sprint(e.Sequence)))
	buf.WriteByte('\n')

	if e.Status != "" {
		buf.Write(eventStatus.With(string(e.Status)))
		buf.WriteByte('\n')
	}
	if e.Summary != "" {
		buf.Write(eventSummary.With(e.Summary))
		buf.WriteByte('\n')
	}
	if e.Transparent != "" {
		buf.Write(eventTransparent.With(string(e.Transparent)))
		buf.WriteByte('\n')
	}

	buf.Write(tailer.With(tagEvent))
	buf.WriteByte('\n')

	return buf.Bytes()
}

type Item string

func (i Item) With(value string) []byte { return append([]byte(i), []byte(value)...) }
