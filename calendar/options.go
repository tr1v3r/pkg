package calendar

import "time"

// CalendarOption calendar option
type CalendarOption func(*Calendar) *Calendar

var (
	// WithProdID set prod id
	WithProdID = func(prodID string) CalendarOption {
		return func(c *Calendar) *Calendar {
			c.prodID = prodID
			return c
		}
	}
	// WithVersion set version
	WithVersion = func(version string) CalendarOption {
		return func(c *Calendar) *Calendar {
			c.version = version
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
	// WithStamp set stamp
	WithStamp = func(stamp time.Time) EventOption {
		return func(e *Event) *Event {
			e.Stamp = stamp
			return e
		}
	}
	// WithUID set uid
	WithUID = func(uid string) EventOption {
		return func(e *Event) *Event {
			e.UID = uid
			return e
		}
	}
	// WithClass set class
	WithClass = func(class Class) EventOption {
		return func(e *Event) *Event {
			e.Class = class
			return e
		}
	}
	// WithCreated set created
	WithCreatedAt = func(createdAt time.Time) EventOption {
		return func(e *Event) *Event {
			e.CreatedAt = createdAt
			return e
		}
	}
	// WithCreated set created
	WithModifiedAt = func(modifiedAt time.Time) EventOption {
		return func(e *Event) *Event {
			e.ModifiedAt = modifiedAt
			return e
		}
	}
	// WithLocation set location
	WithLocation = func(location string) EventOption {
		return func(e *Event) *Event {
			e.Location = location
			return e
		}
	}
	// WithSequence set location
	WithSequence = func(sequence int) EventOption {
		return func(e *Event) *Event {
			e.Sequence = sequence
			return e
		}
	}
	// WithStatus set status
	WithStatus = func(status Status) EventOption {
		return func(e *Event) *Event {
			e.Status = status
			return e
		}
	}
	// WithSummary set summary
	WithSummary = func(summary string) EventOption {
		return func(e *Event) *Event {
			e.Summary = summary
			return e
		}
	}
	// WithDesc set description
	WithDesc = func(desc string) EventOption {
		return func(e *Event) *Event {
			e.Desc = desc
			return e
		}
	}
	// WithTransparent set transparent
	WithTransparent = func(transp Transparent) EventOption {
		return func(e *Event) *Event {
			e.Transparent = transp
			return e
		}
	}
)
