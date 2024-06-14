package calendar

// gist wiki
// https://gist.github.com/yulanggong/be953ffee1d42df53a1a

// http://tools.ietf.org/html/rfc5545
// http://en.wikipedia.org/wiki/Icalendar

// 与 Google 日历同步

// OAuth 认证
// http://en.wikipedia.org/wiki/Icalendar

// Google Calendar API 开发示例
// https://developers.google.com/google-apps/calendar/firstapp

// Google Calendar API
// https://developers.google.com/google-apps/calendar/v3/reference/

// 格式为iCalendar, 扩展名是.ics
// BEGIN:VCALENDAR #日历开始
// PRODID:-//Google Inc//Google Calendar 70.9054//EN #软件信息
// VERSION:2.0 #遵循的 iCalendar 版本号
// CALSCALE:GREGORIAN #历法：公历
// METHOD:PUBLISH #方法：公开 也可以是 REQUEST 等用于日历间的信息沟通方法
// X-WR-CALNAME:yulanggong@gmail.com #这是一个通用扩展属性 表示本日历的名称
// X-WR-TIMEZONE:Asia/Shanghai #通用扩展属性，表示时区
// BEGIN:VEVENT #事件开始
// DTSTART:20090305T112200Z #开始的日期时间
// DTEND:20090305T122200Z #结束的日期时间
// DTSTAMP:20140613T033914Z #有Method 属性时表示 实例创建时间，没有时表示最后修订的日期时间
// UID:9r5p7q78uohmk1bbt0iedof9s4@google.com #UID
// CLASS:PRIVATE #保密类型
// CREATED:20090305T092105Z #创建的日期时间
// DESCRIPTION:test #描述
// LAST-MODIFIED:20090305T092130Z #最后修改日期时间
// LOCATION:test #地址
// SEQUENCE:1 #排列序号
// STATUS:CONFIRMED #状态 TENTATIVE 试探 CONFIRMED 确认 CANCELLED 取消
// SUMMARY: test #简介 一般是标题
// TRANSP:OPAQUE #对于忙闲查询是否透明 OPAQUE 不透明 TRANSPARENT 透明
// END:VEVENT #事件结束
// END:VCALENDAR #日历结束

// BEGIN:VTIMEZONE
// TZID:Asia/Shanghai
// X-LIC-LOCATION:Asia/Shanghai
// BEGIN:STANDARD
// TZOFFSETFROM:+0800
// TZOFFSETTO:+0800
// TZNAME:CST
// DTSTART:19700101T000000
// END:STANDARD
// END:VTIMEZONE

// demo https://www.shuyz.com/githubfiles/china-holiday-calender/master/holidayCal.ics

// convert https://wallstreetcn.com/calendar
