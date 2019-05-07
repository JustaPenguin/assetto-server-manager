package components

import (
	"fmt"
	"github.com/heindl/caldav-go/icalendar/values"
	"github.com/heindl/caldav-go/utils"
	"time"
)

type Calendar struct {

	// specifies the identifier corresponding to the highest version number or the minimum and maximum
	// range of the iCalendar specification that is required in order to interpret the iCalendar object.
	Version string `ical:",2.0"`

	// specifies the identifier for the product that created the iCalendar object
	ProductId string `ical:"prodid,-//taviti/caldav-go//NONSGML v1.0.0//EN"`

	// specifies the text value that uniquely identifies the "VTIMEZONE" calendar component.
	TimeZoneId string `ical:"tzid,omitempty"`

	// defines the iCalendar object method associated with the calendar object.
	Method values.Method `ical:",omitempty"`

	// defines the calendar scale used for the calendar information specified in the iCalendar object.
	values.CalScale `ical:",omitempty"`

	// defines the different timezones used by the various components nested within
	TimeZones []*TimeZone `ical:",omitempty"`

	// unique events to be stored together in the icalendar file
	Events []*Event `ical:",omitempty"`
}

func (c *Calendar) UseTimeZone(location *time.Location) *TimeZone {
	tz := NewDynamicTimeZone(location)
	c.TimeZones = append(c.TimeZones, tz)
	c.TimeZoneId = tz.Id
	return tz
}

func (c *Calendar) UsingTimeZone() bool {
	return len(c.TimeZoneId) > 0
}

func (c *Calendar) UsingGlobalTimeZone() bool {
	return c.UsingTimeZone() && c.TimeZoneId[0] == '/'
}

func (c *Calendar) ValidateICalValue() error {

	for i, e := range c.Events {

		if e == nil {
			continue // skip nil events
		}

		if err := e.ValidateICalValue(); err != nil {
			msg := fmt.Sprintf("event %d failed validation", i)
			return utils.NewError(c.ValidateICalValue, msg, c, err)
		}

		if e.DateStart == nil && c.Method == "" {
			msg := fmt.Sprintf("no value for method and no start date defined on event %d", i)
			return utils.NewError(c.ValidateICalValue, msg, c, nil)
		}

	}

	if c.UsingTimeZone() && !c.UsingGlobalTimeZone() {
		for i, t := range c.TimeZones {
			if t == nil || t.Id != c.TimeZoneId {
				msg := fmt.Sprintf("timezone ID does not match timezone %d", i)
				return utils.NewError(c.ValidateICalValue, msg, c, nil)
			}
		}
	}

	return nil

}

func NewCalendar(events ...*Event) *Calendar {
	cal := new(Calendar)
	cal.Events = events
	return cal
}
