package values

import (
	"fmt"
	"github.com/heindl/caldav-go/icalendar/properties"
	"github.com/heindl/caldav-go/utils"
	"log"
	"net/mail"
	"strings"
)

var _ = log.Print

// Specifies the organizer of a group scheduled calendar entity. The property is specified within the "VFREEBUSY"
// calendar component to specify the calendar user requesting the free or busy time. When publishing a "VFREEBUSY"
// calendar component, the property is used to specify the calendar that the published busy time came from.
//
// The property has the property parameters CN, for specifying the common or display name associated with the
// "Organizer", DIR, for specifying a pointer to the directory information associated with the "Organizer",
// SENT-BY, for specifying another calendar user that is acting on behalf of the "Organizer". The non-standard
// parameters may also be specified on this property. If the LANGUAGE property parameter is specified, the identified
// language applies to the CN parameter value.
type Contact struct {
	Entry mail.Address
}

type AttendeeContact Contact
type OrganizerContact Contact

// creates a new icalendar attendee representation
func NewAttendeeContact(name, email string) *AttendeeContact {
	return &AttendeeContact{Entry: mail.Address{Name: name, Address: email}}
}

// creates a new icalendar organizer representation
func NewOrganizerContact(name, email string) *OrganizerContact {
	return &OrganizerContact{Entry: mail.Address{Name: name, Address: email}}
}

// validates the contact value for the iCalendar specification
func (c *Contact) ValidateICalValue() error {
	email := c.Entry.String()
	if _, err := mail.ParseAddress(email); err != nil {
		msg := fmt.Sprintf("unable to validate address %s", email)
		return utils.NewError(c.ValidateICalValue, msg, c, err)
	} else {
		return nil
	}
}

// encodes the contact value for the iCalendar specification
func (c *Contact) EncodeICalValue() (string, error) {
	return fmt.Sprintf("MAILTO:%s", c.Entry.Address), nil
}

// encodes the contact params for the iCalendar specification
func (c *Contact) EncodeICalParams() (params properties.Params, err error) {
	if c.Entry.Name != "" {
		params = properties.Params{properties.CanonicalNameParameterName: c.Entry.Name}
	}
	return
}

// decodes the contact value from the iCalendar specification
func (c *Contact) DecodeICalValue(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) > 1 {
		c.Entry.Address = parts[1]
	}
	return nil
}

// decodes the contact params from the iCalendar specification
func (c *Contact) DecodeICalParams(params properties.Params) error {
	if name, found := params[properties.CanonicalNameParameterName]; found {
		c.Entry.Name = name
	}
	return nil
}

// validates the contact value for the iCalendar specification
func (c *OrganizerContact) ValidateICalValue() error {
	return (*Contact)(c).ValidateICalValue()
}

// encodes the contact value for the iCalendar specification
func (c *OrganizerContact) EncodeICalValue() (string, error) {
	return (*Contact)(c).EncodeICalValue()
}

// encodes the contact params for the iCalendar specification
func (c *OrganizerContact) EncodeICalParams() (params properties.Params, err error) {
	return (*Contact)(c).EncodeICalParams()
}

// decodes the contact value from the iCalendar specification
func (c *OrganizerContact) DecodeICalValue(value string) error {
	return (*Contact)(c).DecodeICalValue(value)
}

// decodes the contact params from the iCalendar specification
func (c *OrganizerContact) DecodeICalParams(params properties.Params) error {
	return (*Contact)(c).DecodeICalParams(params)
}

// encodes the contact property name for the iCalendar specification
func (o *OrganizerContact) EncodeICalName() (properties.PropertyName, error) {
	return properties.OrganizerPropertyName, nil
}

// validates the contact value for the iCalendar specification
func (c *AttendeeContact) ValidateICalValue() error {
	return (*Contact)(c).ValidateICalValue()
}

// encodes the contact value for the iCalendar specification
func (c *AttendeeContact) EncodeICalValue() (string, error) {
	return (*Contact)(c).EncodeICalValue()
}

// encodes the contact params for the iCalendar specification
func (c *AttendeeContact) EncodeICalParams() (params properties.Params, err error) {
	return (*Contact)(c).EncodeICalParams()
}

// decodes the contact value from the iCalendar specification
func (c *AttendeeContact) DecodeICalValue(value string) error {
	return (*Contact)(c).DecodeICalValue(value)
}

// decodes the contact params from the iCalendar specification
func (c *AttendeeContact) DecodeICalParams(params properties.Params) error {
	return (*Contact)(c).DecodeICalParams(params)
}

// encodes the contact property name for the iCalendar specification
func (o *AttendeeContact) EncodeICalName() (properties.PropertyName, error) {
	return properties.AttendeePropertyName, nil
}
