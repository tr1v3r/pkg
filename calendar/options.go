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
	// WithMethod set method
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
	// WithCreated set created
	WithCreatedAt = func(createdAt time.Time) EventOption {
		return func(e *Event) *Event {
			e.createdAt = NewDate("CREATED", createdAt)
			return e
		}
	}
	// SetStampFormat set date format
	SetCreatedAtFormat = func(layout string, configs ...string) EventOption {
		return func(e *Event) *Event {
			setTimeFormat(&e.createdAt, layout, configs...)
			return e
		}
	}
	// WithCreated set created
	WithModifiedAt = func(modifiedAt time.Time) EventOption {
		return func(e *Event) *Event {
			e.modifiedAt = NewDate("LAST-MODIFIED", modifiedAt)
			return e
		}
	}
	// SetStampFormat set date format
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
	// WithSequence set location
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
)

func setTimeFormat(d *Date, layout string, configs ...string) {
	d.configs = append(d.configs, configs...)
	d.layout = layout
}
