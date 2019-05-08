package components

import (
	"github.com/heindl/caldav-go/icalendar/values"
	"github.com/heindl/caldav-go/utils"
	"time"
)

type Event struct {

	// defines the persistent, globally unique identifier for the calendar component.
	UID string `ical:",required"`

	// indicates the date/time that the instance of the iCalendar object was created.
	DateStamp *values.DateTime `ical:"dtstamp,required"`

	// specifies when the calendar component begins.
	DateStart *values.DateTime `ical:"dtstart,required"`

	// specifies the date and time that a calendar component ends.
	DateEnd *values.DateTime `ical:"dtend,omitempty"`

	// specifies a positive duration of time.
	Duration *values.Duration `ical:",omitempty"`

	// defines the access classification for a calendar component.
	AccessClassification values.EventAccessClassification `ical:"class,omitempty"`

	// specifies the date and time that the calendar information was created by the calendar user agent in the
	// calendar store.
	// Note: This is analogous to the creation date and time for a file in the file system.
	Created *values.DateTime `ical:",omitempty"`

	// provides a more complete description of the calendar component, than that provided by the Summary property.
	Description string `ical:",omitempty"`

	// specifies information related to the global position for the activity specified by a calendar component.
	Geo *values.Geo `ical:",omitempty"`

	// specifies the date and time that the information associated with the calendar component was last revised in the
	// calendar store.
	// Note: This is analogous to the modification date and time for a file in the file system.
	LastModified *values.DateTime `ical:"last-modified,omitempty"`

	// defines the intended venue for the activity defined by a calendar component.
	Location *values.Location `ical:",omitempty"`

	// defines the organizer for a calendar component.
	Organizer *values.OrganizerContact `ical:",omitempty"`

	// defines the relative priority for a calendar component.
	Priority int `ical:",omitempty"`

	// defines the revision sequence number of the calendar component within a sequence of revisions.
	Sequence int `ical:",omitempty"`

	// efines the overall status or confirmation for the calendar component.
	Status values.EventStatus `ical:",omitempty"`

	// defines a short summary or subject for the calendar component.
	Summary string `ical:",omitempty"`

	// defines whether an event is transparent or not to busy time searches.
	values.TimeTransparency `ical:"transp,omitempty"`

	// defines a Uniform Resource Locator (URL) associated with the iCalendar object.
	Url *values.Url `ical:",omitempty"`

	// used in conjunction with the "UID" and "SEQUENCE" property to identify a specific instance of a recurring
	// event calendar component. The property value is the effective value of the DateStart property of the
	// recurrence instance.
	RecurrenceId *values.DateTime `ical:"recurrence_id,omitempty"`

	// defines a rule or repeating pattern for recurring events, to-dos, or time zone definitions.
	RecurrenceRules []*values.RecurrenceRule `ical:"rrule,omitempty"`

	// property provides the capability to associate a document object with a calendar component.
	Attachment *values.Url `ical:"attach,omitempty"`

	// defines an "Attendee" within a calendar component.
	Attendees []*values.AttendeeContact `ical:"attendee,omitempty"`

	// defines the categories for a calendar component.
	Categories *values.CSV `ical:",omitempty"`

	// specifies non-processing information intended to provide a comment to the calendar user.
	Comments []values.Comment `ical:",omitempty"`

	// used to represent contact information or alternately a reference to contact information associated with the calendar component.
	ContactInfo *values.CSV `ical:"contact,omitempty"`

	// defines the list of date/time exceptions for a recurring calendar component.
	*values.ExceptionDateTimes `ical:",omitempty"`

	// defines the list of date/times for a recurrence set.
	*values.RecurrenceDateTimes `ical:",omitempty"`

	// used to represent a relationship or reference between one calendar component and another.
	RelatedTo *values.Url `ical:"related-to,omitempty"`

	// defines the equipment or resources anticipated for an activity specified by a calendar entity.
	Resources *values.CSV `ical:",omitempty"`
}

// validates the event internals
func (e *Event) ValidateICalValue() error {

	if e.UID == "" {
		return utils.NewError(e.ValidateICalValue, "the UID value must be set", e, nil)
	}

	if e.DateStart == nil {
		return utils.NewError(e.ValidateICalValue, "event start date must be set", e, nil)
	}

	if e.DateEnd == nil && e.Duration == nil {
		return utils.NewError(e.ValidateICalValue, "event end date or duration must be set", e, nil)
	}

	if e.DateEnd != nil && e.Duration != nil {
		return utils.NewError(e.ValidateICalValue, "event end date and duration are mutually exclusive fields", e, nil)
	}

	return nil

}

func (e *Event) AddAttendees(a ...*values.AttendeeContact) {
	e.Attendees = append(e.Attendees, a...)
}

// adds one or more recurrence rule to the event
func (e *Event) AddRecurrenceRules(r ...*values.RecurrenceRule) {
	e.RecurrenceRules = append(e.RecurrenceRules, r...)
}

// adds one or more recurrence rule exception to the event
func (e *Event) AddRecurrenceExceptions(d ...*values.DateTime) {
	if e.ExceptionDateTimes == nil {
		e.ExceptionDateTimes = new(values.ExceptionDateTimes)
	}
	*e.ExceptionDateTimes = append(*e.ExceptionDateTimes, d...)
}

// checks to see if the event is a recurrence
func (e *Event) IsRecurrence() bool {
	return e.RecurrenceId != nil
}

// checks to see if the event is a recurrence override
func (e *Event) IsOverride() bool {
	return e.IsRecurrence() && !e.RecurrenceId.Equals(e.DateStart)
}

// creates a new iCalendar event with no end time
func NewEvent(uid string, start time.Time) *Event {
	e := new(Event)
	e.UID = uid
	e.DateStamp = values.NewDateTime(time.Now().UTC())
	e.DateStart = values.NewDateTime(start)
	return e
}

// creates a new iCalendar event that lasts a certain duration
func NewEventWithDuration(uid string, start time.Time, duration time.Duration) *Event {
	e := NewEvent(uid, start)
	e.Duration = values.NewDuration(duration)
	return e
}

// creates a new iCalendar event that has an explicit start and end time
func NewEventWithEnd(uid string, start time.Time, end time.Time) *Event {
	e := NewEvent(uid, start)
	e.DateEnd = values.NewDateTime(end)
	return e
}
