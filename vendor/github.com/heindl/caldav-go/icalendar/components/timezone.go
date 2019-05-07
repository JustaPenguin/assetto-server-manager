package components

import (
	"fmt"
	"github.com/heindl/caldav-go/icalendar/values"
	"net/url"
	"time"
)

type TimeZone struct {

	// defines the persistent, globally unique identifier for the calendar component.
	Id string `ical:"tzid,required"`

	// the location name, as defined by the standards body
	ExtLocationName string `ical:"x-lic-location,omitempty"`

	// defines a Uniform Resource Locator (URL) associated with the iCalendar object.
	Url *values.Url `ical:"tzurl,omitempty"`

	// specifies the date and time that the information associated with the calendar component was last revised in the
	// calendar store.
	// Note: This is analogous to the modification date and time for a file in the file system.
	LastModified *values.DateTime `ical:"last-modified,omitempty"`

	// TODO need to figure out how to handle standard and daylight savings time

}

func NewDynamicTimeZone(location *time.Location) *TimeZone {
	t := new(TimeZone)
	t.Id = location.String()
	t.ExtLocationName = location.String()
	t.Url = values.NewUrl(url.URL{
		Scheme: "http",
		Host:   "tzurl.org",
		Path:   fmt.Sprintf("/zoneinfo/%s", t.Id),
	})
	return t
}
