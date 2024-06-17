package calendar

import (
	"testing"
	"time"
)

func TestNewCalendar(t *testing.T) {

}

func TestOutput(t *testing.T) {
	c := &Calendar{
		prodID:   generalProdID,
		version:  generalVersion,
		scale:    ScaleGregorian,
		method:   MethodPublish,
		name:     "test calendar",
		timeZone: TZShanghai,
		desc:     "this is a test calendar",
		events: []Event{
			{
				start:       Date{Time: time.Now().UTC().Add(-24 * time.Hour)},
				end:         Date{Time: time.Now().UTC()},
				uid:         "abc",
				class:       ClassPublic,
				createdAt:   Date{Time: time.Now()},
				desc:        "test event A",
				location:    "SG",
				sequence:    0,
				status:      StatusConfirmed,
				summary:     "test event Title A",
				transparent: TranspTransparent,
			},
			{
				start:       Date{Time: time.Now().UTC()},
				end:         Date{Time: time.Now().UTC().Add(24 * time.Hour)},
				uid:         "def",
				class:       ClassPublic,
				createdAt:   Date{Time: time.Now()},
				desc:        "test event B",
				location:    "CN",
				sequence:    1,
				status:      StatusConfirmed,
				summary:     "test event Title B",
				transparent: TranspTransparent,
			},
		},
	}

	t.Logf("out:\n%s", c.Output())
}
