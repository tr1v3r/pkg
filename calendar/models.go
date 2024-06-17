package calendar

import (
	"bytes"
	"fmt"
	"time"
)

const (
	LayoutTime = "20060102T150405Z"
	LayoutDate = "20060102"

	DateFormat = "VALUE=DATE"
)

type (
	Header string // 开始标记
	Tailer string // 结束标记

	// ============== VCALENDAR ==============
	ProdID   string // 软件信息
	Version  string // 遵循的 iCalendar 版本号
	Scale    string // 历法：公历
	Method   string // 方法PUBLISH/REQUEST等日历间的信息沟通方法
	TimeZone string // 通用扩展属性 表示时区
	CalName  string // 通用扩展属性 表示本日历的名称
	CalDesc  string // 日历描述

	// ============== VEVENT ==============
	Status      string // 状态 TENTATIVE 试探 CONFIRMED 确认 CANCELLED 取消
	Summary     string // 简介 一般是标题
	UID         string // UID
	Class       string // 事件类型
	Transparent string // 对于忙闲查询是否透明 OPAQUE 不透明 TRANSPARENT 透明
	Location    string // location
	Sequence    int    // 排列序号 0 最高
	Desc        string // 描述
)

const (
	// eventDateStart  Item = "DTSTART;"       // 开始的日期时间: 20090305T112200Z
	// eventDateEnd    Item = "DTEND;"         // 结束的日期时间: 20090305T122200Z
	// eventDateStamp  Item = "DTSTAMP:"       // 有Method 属性时表示 实例创建时间，没有时表示最后修订的日期时间
	// eventCreatedAt  Item = "CREATED:"       // 创建的日期时间: 20090305T092105Z
	// eventModifiedAt Item = "LAST-MODIFIED:" // 最后修改日期时间: 20090305T092130Z

	ScaleGregorian Scale = "GREGORIAN"

	MethodPublish Method = "PUBLISH"
	MetohdRequest Method = "REQUEST"

	// https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
	TZShanghai  TimeZone = "Asia/Shanghai"
	TZSingapore TimeZone = "Asia/Singapore"

	StatusTentative Status = "TENTATIVE"
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelled Status = "CANCELLED"

	ClassPublic       Class = "PUBLIC"
	ClassPrivate      Class = "PRIVATE"
	ClassConfidential Class = "CONFIDENTIAL"

	TranspTransparent Transparent = "TRANSPARENT"
	TranspOpaque      Transparent = "OPAQUE"
)

func (h Header) Output() []byte      { return append([]byte("BEGIN:"), []byte(h)...) }
func (t Tailer) Output() []byte      { return append([]byte("END:"), []byte(t)...) }
func (id ProdID) Output() []byte     { return append([]byte("PRODID:"), []byte(id)...) }
func (v Version) Output() []byte     { return append([]byte("VERSION:"), []byte(v)...) }
func (n CalName) Output() []byte     { return append([]byte("X-WR-CALNAME:"), []byte(n)...) }
func (d CalDesc) Output() []byte     { return append([]byte("X-WR-CALDESC:"), []byte(d)...) }
func (s Scale) Output() []byte       { return append([]byte("CALSCALE:"), []byte(s)...) }
func (m Method) Output() []byte      { return append([]byte("METHOD:"), []byte(m)...) }
func (tz TimeZone) Output() []byte   { return append([]byte("X-WR-TIMEZONE:"), []byte(tz)...) }
func (s Status) Output() []byte      { return append([]byte("STATUS:"), []byte(s)...) }
func (s Summary) Output() []byte     { return append([]byte("SUMMARY:"), []byte(s)...) }
func (u UID) Output() []byte         { return append([]byte("UID:"), []byte(u)...) }
func (c Class) Output() []byte       { return append([]byte("CLASS:"), []byte(c)...) }
func (t Transparent) Output() []byte { return append([]byte("TRANSP:"), []byte(t)...) }
func (l Location) Output() []byte    { return append([]byte("LOCATION:"), []byte(l)...) }
func (s Sequence) Output() []byte    { return append([]byte("SEQUENCE:"), []byte(fmt.Sprint(s))...) }
func (d Desc) Output() []byte        { return append([]byte("DESCRIPTION:"), []byte(d)...) }

// ============== Date ==============

func NewDate(key string, t time.Time) Date { return Date{key: key, layout: LayoutTime, Time: t} }

// Date
// DTSTART:19980313T141711Z
// DTSTART;VALUE=DATE:19970317
// DTSTART;TZID=America/New_York:19970902T090000
type Date struct {
	key     string
	configs []string
	layout  string

	time.Time
}

func (d Date) Output() []byte {
	var buf bytes.Buffer

	buf.WriteString(d.key)

	for _, config := range d.configs {
		buf.WriteByte(';')
		buf.WriteString(config)
	}

	buf.WriteByte(':')
	buf.WriteString(d.Format(d.layout))

	return buf.Bytes()
}
