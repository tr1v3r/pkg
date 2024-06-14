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
		version:  version,
		scale:    ScaleGregorian,
		method:   MethodPublish,
		name:     "test calendar",
		timeZone: TZShanghai,
		desc:     "this is a test calendar",
		events: []Event{
			{
				Start:       time.Now().UTC().Add(-24 * time.Hour),
				End:         time.Now().UTC(),
				UID:         uuid.NewString(),
				Class:       ClassPublic,
				CreatedAt:   time.Now(),
				Desc:        "test event A",
				Location:    "SG",
				Sequence:    0,
				Status:      StatusConfirmed,
				Summary:     "test event Title A",
				Transparent: TranspTransparent,
			},
			{
				Start:       time.Now().UTC(),
				End:         time.Now().UTC().Add(24 * time.Hour),
				UID:         uuid.NewString(),
				Class:       ClassPublic,
				CreatedAt:   time.Now(),
				Desc:        "test event B",
				Location:    "CN",
				Sequence:    1,
				Status:      StatusConfirmed,
				Summary:     "test event Title B",
				Transparent: TranspTransparent,
			},
		},
	}

	t.Logf("out:\n%s", c.Output())
}
