package calendar

import "time"

// CalendarOption calendar option
type CalendarOption func(*Calendar) *Calendar

var (
	// WithProdID set prod id
	WithProdID = func(prodID string) CalendarOption {
		return func(c *Calendar) *Calendar {
			c.prodID = ProdID(prodID)
			return c
		}
	}
	// WithVersion set version
	WithVersion = func(version string) CalendarOption {
		return func(c *Calendar) *Calendar {
			c.version = Version(version)
			return c
		}
	}
	// WithScale set scale
	WithScale = func(scale Scale) CalendarOption {
		return func(c *Calendar) *Calendar {
			c.scale = scale
			return c
		}
	}
	// WithMethod set method
	WithMethod = func(method Method) CalendarOption {
		return func(c *Calendar) *Calendar {
			c.method = method
			return c
		}
	}
	// WithTimeZone set timezone
	WithTimeZone = func(timezone TimeZone) CalendarOption {
		return func(c *Calendar) *Calendar {
			c.timeZone = timezone
			return c
		}
	}
)

// EventOption calendar event option
type EventOption func(*Event) *Event

var (
	// SetStartFormat set start format
	SetStartFormat = func(layout string, configs ...string) EventOption {
		return func(e *Event) *Event {
			setTimeFormat(&e.start, layout, configs...)
			return e
		}
	}
	// WithEnd set end time
	WithEnd = func(end time.Time) EventOption {
		return func(e *Event) *Event {
			e.end = NewDate("DTEND", end)
			return e
		}
	}
	// SetEndFormat set date format
	SetEndFormat = func(layout string, configs ...string) EventOption {
		return func(e *Event) *Event {
			setTimeFormat(&e.end, layout, configs...)
			return e
		}
	}
	// WithStamp set stamp
	WithStamp = func(stamp time.Time) EventOption {
		return func(e *Event) *Event {
			e.stamp = NewDate("DTSTAMP", stamp)
			return e
		}
	}
	// SetStampFormat set date format
	SetStampFormat = func(layout string, configs ...string) EventOption {
		return func(e *Event) *Event {
			setTimeFormat(&e.stamp, layout, configs...)
			return e
		}
	}
	// WithUID set uid
	WithUID = func(uid string) EventOption {
		return func(e *Event) *Event {
			e.uid = UID(uid)
			return e
		}
	}
	// WithClass set class
	WithClass = func(class Class) EventOption {
		return func(e *Event) *Event {
			e.class = class
			return e
		}
	}
	// WithCreatedAt set created
	WithCreatedAt = func(createdAt time.Time) EventOption {
		return func(e *Event) *Event {
			e.createdAt = NewDate("CREATED", createdAt)
			return e
		}
	}
	// SetCreatedAtFormat set date format
	SetCreatedAtFormat = func(layout string, configs ...string) EventOption {
		return func(e *Event) *Event {
			setTimeFormat(&e.createdAt, layout, configs...)
			return e
		}
	}
	// WithModifiedAt set modified
	WithModifiedAt = func(modifiedAt time.Time) EventOption {
		return func(e *Event) *Event {
			e.modifiedAt = NewDate("LAST-MODIFIED", modifiedAt)
			return e
		}
	}
	// SetModifiedAtFormat set date format
	SetModifiedAtFormat = func(layout string, configs ...string) EventOption {
		return func(e *Event) *Event {
			setTimeFormat(&e.modifiedAt, layout, configs...)
			return e
		}
	}
	// WithLocation set location
	WithLocation = func(location string) EventOption {
		return func(e *Event) *Event {
			e.location = Location(location)
			return e
		}
	}
	// WithSequence set sequence
	WithSequence = func(sequence int) EventOption {
		return func(e *Event) *Event {
			e.sequence = Sequence(sequence)
			return e
		}
	}
	// WithStatus set status
	WithStatus = func(status Status) EventOption {
		return func(e *Event) *Event {
			e.status = status
			return e
		}
	}
	// WithSummary set summary
	WithSummary = func(summary string) EventOption {
		return func(e *Event) *Event {
			e.summary = Summary(summary)
			return e
		}
	}
	// WithDesc set description
	WithDesc = func(s string) EventOption {
		return func(e *Event) *Event {
			e.desc = Desc(s)
			return e
		}
	}
	// WithTransparent set transparent
	WithTransparent = func(transp Transparent) EventOption {
		return func(e *Event) *Event {
			e.transparent = transp
			return e
		}
	}
	// WithRRULE set recurrence rule
	WithRRULE = func(rrule string) EventOption {
		return func(e *Event) *Event {
			e.rrule = RRULE(rrule)
			return e
		}
	}
	// WithOrganizer set organizer
	WithOrganizer = func(uri string, params ...string) EventOption {
		return func(e *Event) *Event {
			e.organizer = NewOrganizer(uri, params...)
			return e
		}
	}
	// WithAttendee add attendee
	WithAttendee = func(uri string, params ...string) EventOption {
		return func(e *Event) *Event {
			e.attendees = append(e.attendees, NewAttendee(uri, params...))
			return e
		}
	}
	// WithCategories set categories
	WithCategories = func(cats ...string) EventOption {
		return func(e *Event) *Event {
			e.categories = Categories(cats)
			return e
		}
	}
	// WithPriority set priority (0-9)
	WithPriority = func(p int) EventOption {
		return func(e *Event) *Event {
			e.priority = Priority(p)
			return e
		}
	}
	// WithEventURL set URL
	WithEventURL = func(u string) EventOption {
		return func(e *Event) *Event {
			e.url = URL(u)
			return e
		}
	}
	// WithDuration set duration
	WithDuration = func(d string) EventOption {
		return func(e *Event) *Event {
			e.duration = Duration(d)
			return e
		}
	}
	// WithExDate add exception dates
	WithExDate = func(dates ...time.Time) EventOption {
		return func(e *Event) *Event {
			e.exdates = append(e.exdates, NewDateList("EXDATE", dates))
			return e
		}
	}
	// WithRDate add recurrence dates
	WithRDate = func(dates ...time.Time) EventOption {
		return func(e *Event) *Event {
			e.rdates = append(e.rdates, NewDateList("RDATE", dates))
			return e
		}
	}
	// WithRecurrenceID set recurrence ID
	WithRecurrenceID = func(t time.Time) EventOption {
		return func(e *Event) *Event {
			e.recurrenceID = NewDate("RECURRENCE-ID", t)
			return e
		}
	}
	// WithGeo set geographic coordinates
	WithGeo = func(lat, lon float64) EventOption {
		return func(e *Event) *Event {
			e.geo = Geo{Lat: lat, Lon: lon}
			return e
		}
	}
	// WithComment set comment
	WithComment = func(c string) EventOption {
		return func(e *Event) *Event {
			e.comment = Comment(c)
			return e
		}
	}
	// WithContact set contact
	WithContact = func(c string) EventOption {
		return func(e *Event) *Event {
			e.contact = Contact(c)
			return e
		}
	}
	// WithRelatedTo set related-to
	WithRelatedTo = func(uid string) EventOption {
		return func(e *Event) *Event {
			e.relatedTo = RelatedTo(uid)
			return e
		}
	}
	// WithResources set resources
	WithResources = func(r string) EventOption {
		return func(e *Event) *Event {
			e.resources = Resources(r)
			return e
		}
	}
	// WithAttachment add attachment
	WithAttachment = func(uri string, params ...string) EventOption {
		return func(e *Event) *Event {
			e.attachments = append(e.attachments, NewAttachment(uri, params...))
			return e
		}
	}
	// WithAlarm add alarm
	WithAlarm = func(a Alarm) EventOption {
		return func(e *Event) *Event {
			e.alarms = append(e.alarms, a)
			return e
		}
	}
)

func setTimeFormat(d *Date, layout string, configs ...string) {
	d.configs = append(d.configs, configs...)
	d.layout = layout
}
