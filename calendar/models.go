package calendar

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

const (
	// LayoutTimeUTC is an iCalendar time layout constant.
	LayoutTimeUTC = "20060102T150405Z" // UTC time (Z suffix)
	// LayoutTime is an iCalendar time layout constant.
	LayoutTime = "20060102T150405" // local (floating) time
	// LayoutDate is an iCalendar time layout constant.
	LayoutDate = "20060102"

	// DateFormat is the VALUE=DATE parameter marking date-only values.
	DateFormat = "VALUE=DATE"
)

// ICS component names.
const (
	CompVCALENDAR = "VCALENDAR"
	CompVEVENT    = "VEVENT"
	CompVTODO     = "VTODO"
	CompVJOURNAL  = "VJOURNAL"
	CompVTIMEZONE = "VTIMEZONE"
	CompVALARM    = "VALARM"
	CompSTANDARD  = "STANDARD"
	CompDAYLIGHT  = "DAYLIGHT"
)

type (
	// Header is a component BEGIN line value.
	Header string // 开始标记
	// Tailer is a component END line value.
	Tailer string // 结束标记

	// ============== VCALENDAR ==============
	ProdID string // 软件信息
	// Version is the VERSION property of a calendar.
	Version string // 遵循的 iCalendar 版本号
	// Scale is an iCalendar enumeration value.
	Scale string // 历法：公历
	// Method is an iCalendar enumeration value.
	Method string // 方法PUBLISH/REQUEST等日历间的信息沟通方法
	// TimeZone is the X-WR-TIMEZONE calendar property.
	TimeZone string // 通用扩展属性 表示时区
	// CalName is the X-WR-CALNAME calendar property.
	CalName string // 通用扩展属性 表示本日历的名称
	// CalDesc is the X-WR-CALDESC calendar property.
	CalDesc string // 日历描述

	// ============== VEVENT ==============
	Status string // 状态 TENTATIVE 试探 CONFIRMED 确认 CANCELLED 取消
	// Summary is the SUMMARY property text.
	Summary string // 简介 一般是标题
	// UID is the unique identifier property.
	UID string // UID
	// Class is an iCalendar enumeration value.
	Class string // 事件类型
	// Transparent is an iCalendar enumeration value.
	Transparent string // 对于忙闲查询是否透明 OPAQUE 不透明 TRANSPARENT 透明
	// Location is the LOCATION property text.
	Location string // location
	// Sequence is the SEQUENCE revision counter.
	Sequence int // 排列序号 0 最高
	// Desc is the DESCRIPTION property text.
	Desc string // 描述
	// RRULE is a recurrence rule string.
	RRULE string // 重复规则 e.g. FREQ=YEARLY

	// ============== RFC 5545 Additional ==============
	Duration string // e.g. "PT30M", "P1D"
	// Priority is the PRIORITY property value.
	Priority int // 0-9
	// URL is the URL property value.
	URL string
	// Comment is the COMMENT property text.
	Comment string
	// Contact is the CONTACT property text.
	Contact string
	// RelatedTo is the RELATED-TO property value.
	RelatedTo string
	// Resources is the RESOURCES property text.
	Resources string
	// TodoStatus is an iCalendar enumeration value.
	TodoStatus string
	// JournalStatus is an iCalendar enumeration value.
	JournalStatus string
)

const (
	// eventDateStart  Item = "DTSTART;"       // 开始的日期时间: 20090305T112200Z
	// eventDateEnd    Item = "DTEND;"         // 结束的日期时间: 20090305T122200Z
	// eventDateStamp  Item = "DTSTAMP:"       // 有Method 属性时表示 实例创建时间，没有时表示最后修订的日期时间
	// eventCreatedAt  Item = "CREATED:"       // 创建的日期时间: 20090305T092105Z
	// eventModifiedAt Item = "LAST-MODIFIED:" // 最后修改日期时间: 20090305T092130Z

	// ScaleGregorian is an iCalendar enumeration value.
	ScaleGregorian Scale = "GREGORIAN"

	// MethodPublish is an iCalendar enumeration value.
	MethodPublish Method = "PUBLISH"
	// MetohdRequest is the REQUEST method value (historical spelling kept as exported API).
	MetohdRequest Method = "REQUEST"

	// https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
	TZShanghai TimeZone = "Asia/Shanghai"
	// TZSingapore is a timezone property value.
	TZSingapore TimeZone = "Asia/Singapore"

	// StatusTentative is an iCalendar enumeration value.
	StatusTentative Status = "TENTATIVE"
	// StatusConfirmed is an iCalendar enumeration value.
	StatusConfirmed Status = "CONFIRMED"
	// StatusCancelled is an iCalendar enumeration value.
	StatusCancelled Status = "CANCELLED"

	// ClassPublic is an iCalendar enumeration value.
	ClassPublic Class = "PUBLIC"
	// ClassPrivate is an iCalendar enumeration value.
	ClassPrivate Class = "PRIVATE"
	// ClassConfidential is an iCalendar enumeration value.
	ClassConfidential Class = "CONFIDENTIAL"

	// TranspTransparent is an iCalendar enumeration value.
	TranspTransparent Transparent = "TRANSPARENT"
	// TranspOpaque is an iCalendar enumeration value.
	TranspOpaque Transparent = "OPAQUE"

	// VTODO status
	TodoStatusNeedsAction TodoStatus = "NEEDS-ACTION"
	// TodoStatusInProgress is an iCalendar enumeration value.
	TodoStatusInProgress TodoStatus = "IN-PROCESS"
	// TodoStatusCompleted is an iCalendar enumeration value.
	TodoStatusCompleted TodoStatus = "COMPLETED"
	// TodoStatusCancelled is an iCalendar enumeration value.
	TodoStatusCancelled TodoStatus = "CANCELLED"

	// VJOURNAL status
	JournalStatusDraft JournalStatus = "DRAFT"
	// JournalStatusFinal is an iCalendar enumeration value.
	JournalStatusFinal JournalStatus = "FINAL"
	// JournalStatusCancelled is an iCalendar enumeration value.
	JournalStatusCancelled JournalStatus = "CANCELLED"
)

// Output renders the value as an ICS content line.
func (h Header) Output() []byte { return append([]byte("BEGIN:"), []byte(h)...) }

// Output renders the value as an ICS content line.
func (t Tailer) Output() []byte { return append([]byte("END:"), []byte(t)...) }

// Output renders the value as an ICS content line.
func (id ProdID) Output() []byte { return append([]byte("PRODID:"), []byte(id)...) }

// Output renders the value as an ICS content line.
func (v Version) Output() []byte { return append([]byte("VERSION:"), []byte(v)...) }

// Output renders the value as an ICS content line.
func (n CalName) Output() []byte { return append([]byte("X-WR-CALNAME:"), EscapeText(string(n))...) }

// Output renders the value as an ICS content line.
func (d CalDesc) Output() []byte { return append([]byte("X-WR-CALDESC:"), EscapeText(string(d))...) }

// Output renders the value as an ICS content line.
func (s Scale) Output() []byte { return append([]byte("CALSCALE:"), []byte(s)...) }

// Output renders the value as an ICS content line.
func (m Method) Output() []byte { return append([]byte("METHOD:"), []byte(m)...) }

// Output renders the value as an ICS content line.
func (tz TimeZone) Output() []byte { return append([]byte("X-WR-TIMEZONE:"), []byte(tz)...) }

// Output renders the value as an ICS content line.
func (s Status) Output() []byte { return append([]byte("STATUS:"), []byte(s)...) }

// Output renders the value as an ICS content line.
func (s Summary) Output() []byte { return append([]byte("SUMMARY:"), EscapeText(string(s))...) }

// Output renders the value as an ICS content line.
func (u UID) Output() []byte { return append([]byte("UID:"), []byte(u)...) }

// Output renders the value as an ICS content line.
func (c Class) Output() []byte { return append([]byte("CLASS:"), []byte(c)...) }

// Output renders the value as an ICS content line.
func (t Transparent) Output() []byte { return append([]byte("TRANSP:"), []byte(t)...) }

// Output renders the value as an ICS content line.
func (l Location) Output() []byte { return append([]byte("LOCATION:"), EscapeText(string(l))...) }

// Output renders the value as an ICS content line.
func (s Sequence) Output() []byte { return append([]byte("SEQUENCE:"), fmt.Append(nil, s)...) }

// Output renders the value as an ICS content line.
func (d Desc) Output() []byte { return append([]byte("DESCRIPTION:"), EscapeText(string(d))...) }

// Output renders the value as an ICS content line.
func (r RRULE) Output() []byte { return append([]byte("RRULE:"), []byte(r)...) }

// Output renders the value as an ICS content line.
func (d Duration) Output() []byte { return append([]byte("DURATION:"), []byte(d)...) }

// Output renders the value as an ICS content line.
func (p Priority) Output() []byte { return append([]byte("PRIORITY:"), fmt.Append(nil, int(p))...) }

// Output renders the value as an ICS content line.
func (u URL) Output() []byte { return append([]byte("URL:"), []byte(u)...) }

// Output renders the value as an ICS content line.
func (c Comment) Output() []byte { return append([]byte("COMMENT:"), EscapeText(string(c))...) }

// Output renders the value as an ICS content line.
func (c Contact) Output() []byte { return append([]byte("CONTACT:"), EscapeText(string(c))...) }

// Output renders the value as an ICS content line.
func (r RelatedTo) Output() []byte { return append([]byte("RELATED-TO:"), []byte(r)...) }

// Output renders the value as an ICS content line.
func (r Resources) Output() []byte { return append([]byte("RESOURCES:"), EscapeText(string(r))...) }

// Output renders the value as an ICS content line.
func (s TodoStatus) Output() []byte { return append([]byte("STATUS:"), []byte(s)...) }

// Output renders the value as an ICS content line.
func (s JournalStatus) Output() []byte { return append([]byte("STATUS:"), []byte(s)...) }

// ============== Date ==============

// NewDate creates a new date.
func NewDate(key string, t time.Time) Date { return Date{key: key, layout: LayoutTimeUTC, Time: t} }

// Date
// DTSTART:19980313T141711Z
// DTSTART;VALUE=DATE:19970317
// DTSTART;TZID=America/New_York:19970902T090000
type Date struct {
	key     string
	configs []string
	layout  string
	tzid    string // TZID param value; empty means the time carries its own offset (Z) or is a DATE

	time.Time
}

// Output renders the value as an ICS content line.
func (d Date) Output() []byte {
	var buf bytes.Buffer

	buf.WriteString(d.key)

	for _, config := range d.configs {
		buf.WriteByte(';')
		buf.WriteString(config)
	}

	buf.WriteByte(':')

	// Preserve the time semantics instead of forcing UTC:
	//   - DATE values are calendar dates, offset-free by definition;
	//   - floating times (no offset, no TZID) must stay wall-clock;
	//   - offset-carrying times print in their own zone.
	t := d.Time
	if t.Location() != time.Local || d.hasExplicitZone() {
		t = d.In(d.wireLocation())
	}
	buf.WriteString(t.Format(d.layout))

	return buf.Bytes()
}

// hasExplicitZone reports whether the value carries a UTC designator or offset.
func (d Date) hasExplicitZone() bool {
	if d.tzid != "" {
		return true
	}
	_, off := d.Zone()
	if off == 0 {
		// zero offset is indistinguishable from a parsed local time at UTC;
		// trust the layout: LayoutTimeUTC means the wire form had a Z.
		return d.layout == LayoutTimeUTC
	}
	return true
}

// wireLocation returns the location the value should be printed in.
func (d Date) wireLocation() *time.Location {
	if loc, err := time.LoadLocation(d.tzid); d.tzid != "" && err == nil {
		return loc
	}
	return d.Location()
}

// ============== RFC 5545 Text Escaping ==============

// EscapeText escapes a property value per RFC 5545 §3.3.11: backslash,
// semicolon, comma and newlines must not appear raw in TEXT values.
func EscapeText(s string) string {
	if !strings.ContainsAny(s, "\\;,\r\n") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case ';':
			b.WriteString("\\;")
		case ',':
			b.WriteString("\\,")
		case '\r':
			// skip: real newlines become \n below
		case '\n':
			b.WriteString("\\n")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// UnescapeText reverses EscapeText for values read from the wire.
func UnescapeText(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n', 'N':
			b.WriteByte('\n')
		case '\\', ';', ',':
			b.WriteByte(s[i])
		default:
			// not a recognized escape: keep both bytes verbatim
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// ============== RFC 5545 Struct Types ==============

// Attendee represents ATTENDEE property with parameters.
// e.g. ATTENDEE;ROLE=REQ-PARTICIPANT;RSVP=TRUE;CN=John:mailto:john@example.com
type Attendee struct {
	params []string
	URI    string
}

// NewAttendee creates a new attendee.
func NewAttendee(uri string, params ...string) Attendee {
	return Attendee{URI: uri, params: params}
}

// Output renders the value as an ICS content line.
func (a Attendee) Output() []byte {
	var buf bytes.Buffer
	buf.WriteString("ATTENDEE")
	for _, p := range a.params {
		buf.WriteByte(';')
		buf.WriteString(p)
	}
	buf.WriteByte(':')
	buf.WriteString(a.URI)
	return buf.Bytes()
}

// Organizer represents ORGANIZER property with parameters.
// e.g. ORGANIZER;CN=John:mailto:john@example.com
type Organizer struct {
	params []string
	URI    string
}

// NewOrganizer creates a new organizer.
func NewOrganizer(uri string, params ...string) Organizer {
	return Organizer{URI: uri, params: params}
}

// Output renders the value as an ICS content line.
func (o Organizer) Output() []byte {
	var buf bytes.Buffer
	buf.WriteString("ORGANIZER")
	for _, p := range o.params {
		buf.WriteByte(';')
		buf.WriteString(p)
	}
	buf.WriteByte(':')
	buf.WriteString(o.URI)
	return buf.Bytes()
}

// Attachment represents ATTACH property with optional format type.
// e.g. ATTACH;FMTTYPE=application/msword:http://example.com/file.doc
type Attachment struct {
	params []string
	URI    string
}

// NewAttachment creates a new attachment.
func NewAttachment(uri string, params ...string) Attachment {
	return Attachment{URI: uri, params: params}
}

// Output renders the value as an ICS content line.
func (a Attachment) Output() []byte {
	var buf bytes.Buffer
	buf.WriteString("ATTACH")
	for _, p := range a.params {
		buf.WriteByte(';')
		buf.WriteString(p)
	}
	buf.WriteByte(':')
	buf.WriteString(a.URI)
	return buf.Bytes()
}

// DateList represents a list of dates for EXDATE or RDATE properties.
type DateList struct {
	key     string
	layout  string
	configs []string
	Dates   []time.Time
}

// NewDateList creates a new datelist.
func NewDateList(key string, dates []time.Time) DateList {
	return DateList{key: key, layout: LayoutTimeUTC, Dates: dates}
}

// Output renders the value as an ICS content line.
func (dl DateList) Output() []byte {
	var buf bytes.Buffer
	buf.WriteString(dl.key)
	for _, c := range dl.configs {
		buf.WriteByte(';')
		buf.WriteString(c)
	}
	buf.WriteByte(':')
	for i, t := range dl.Dates {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(t.UTC().Format(dl.layout))
	}
	return buf.Bytes()
}

// Geo represents GEO property with latitude and longitude.
type Geo struct {
	Lat float64
	Lon float64
}

// Output renders the value as an ICS content line.
func (g Geo) Output() []byte {
	return fmt.Appendf(nil, "GEO:%s;%s", formatGeoCoord(g.Lat), formatGeoCoord(g.Lon))
}

func formatGeoCoord(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", v), "0"), ".")
}

// Categories represents CATEGORIES property (comma-separated values).
type Categories []string

// Output renders the value as an ICS content line.
func (c Categories) Output() []byte {
	return append([]byte("CATEGORIES:"), []byte(strings.Join(c, ","))...)
}
