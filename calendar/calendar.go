package calendar

import (
	"bytes"
	"time"
)

const (
	generalProdID  = "-//True R1v3r//R1v3r Calendar 1.0//CN"
	generalVersion = "2.0"
)

type Calendar struct {
	header   Header
	prodID   ProdID
	version  Version
	scale    Scale
	method   Method
	name     CalName
	timeZone TimeZone
	desc     CalDesc
	events   []Event
	tailer   Tailer
}

// NewCalendar build new Calendar
func NewCalendar(name, desc string, opts ...CalendarOption) *Calendar {
	c := &Calendar{
		header: "VCALENDAR",
		tailer: "VCALENDAR",

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

func (c *Calendar) AddEvents(events ...Event) { c.events = append(c.events, events...) }

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

	for _, event := range c.events {
		buf.Write(event.Output())
	}

	buf.Write(c.tailer.Output())

	return buf.Bytes()
}

// NewEvent build new calendar event
func NewEvent(sum, description string, start time.Time, opts ...EventOption) *Event {
	e := &Event{
		header: "VEVENT",
		tailer: "VEVENT",

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
	header      Header
	start       Date
	end         Date
	stamp       Date
	uid         UID
	class       Class
	createdAt   Date
	modifiedAt  Date
	location    Location
	sequence    Sequence
	status      Status
	summary     Summary
	desc        Desc
	transparent Transparent
	tailer      Tailer
}

func (e *Event) Output() []byte {
	var buf bytes.Buffer

	buf.Write(e.header.Output())
	buf.WriteByte('\n')

	if !e.start.IsZero() {
		buf.Write(e.start.Output())
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

	buf.Write(e.tailer.Output())
	buf.WriteByte('\n')

	return buf.Bytes()
}
