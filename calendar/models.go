package calendar

const (
	timeLayout = "20060102T150405Z"
	dateLayout = "20060102"

	tagCalendar = "VCALENDAR"
	tagEvent    = "VEVENT"

	header Item = "BEGIN:"
	tailer Item = "END:"

	calProdID   Item = "PRODID:"        // 软件信息
	calVer      Item = "VERSION:"       // 遵循的 iCalendar 版本号
	calScale    Item = "CALSCALE:"      // 历法：公历
	calMethod   Item = "METHOD:"        // 方法PUBLISH/REQUEST等日历间的信息沟通方法
	calName     Item = "X-WR-CALNAME:"  // 通用扩展属性 表示本日历的名称
	calTimeZone Item = "X-WR-TIMEZONE:" // 通用扩展属性 表示时区
	calDesc     Item = "X-WR-CALDESC:"  // 日历描述
	calTailer   Item = "END:VCALENDAR"  // 日历结束

	eventDateStart   Item = "DTSTART;"       // 开始的日期时间: 20090305T112200Z
	eventDateEnd     Item = "DTEND;"         // 结束的日期时间: 20090305T122200Z
	eventDateStamp   Item = "DTSTAMP:"       // 有Method 属性时表示 实例创建时间，没有时表示最后修订的日期时间
	eventUID         Item = "UID:"           // UID
	eventClass       Item = "CLASS:"         // 保密类型: PRIVATE
	eventCreatedAt   Item = "CREATED:"       // 创建的日期时间: 20090305T092105Z
	eventDesc        Item = "DESCRIPTION:"   // 描述
	eventModifiedAt  Item = "LAST-MODIFIED:" // 最后修改日期时间: 20090305T092130Z
	eventLocation    Item = "LOCATION:"      // 地址
	eventSequence    Item = "SEQUENCE:"      // 排列序号: 1
	eventStatus      Item = "STATUS:"        // 状态 TENTATIVE 试探 CONFIRMED 确认 CANCELLED 取消
	eventSummary     Item = "SUMMARY:"       // 简介 一般是标题
	eventTransparent Item = "TRANSP:"        // 对于忙闲查询是否透明 OPAQUE 不透明 TRANSPARENT 透明
)

// Scale 使用的历法
type Scale string

const (
	ScaleGregorian Scale = "GREGORIAN"
)

// Method method
type Method string

const (
	MethodPublish Method = "PUBLISH"
	MetohdRequest Method = "REQUEST"
)

type TimeZone string

const (
	// https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
	TZShanghai  TimeZone = "Asia/Shanghai"
	TZSingapore TimeZone = "Asia/Singapore"
)

type Class string

const (
	ClassPublic       Class = "PUBLIC"
	ClassPrivate      Class = "PRIVATE"
	ClassConfidential Class = "CONFIDENTIAL"
)

type Transparent string

const (
	TranspTransparent Transparent = "TRANSPARENT"
	TranspOpaque      Transparent = "OPAQUE"
)

// ============== VEVENT ==============

type Status string

const (
	StatusTentative Status = "TENTATIVE"
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelled Status = "CANCELLED"
)
