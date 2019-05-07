package values

import (
	"github.com/heindl/caldav-go/icalendar/properties"
	"github.com/heindl/caldav-go/utils"
	"log"
	"net/url"
)

var _ = log.Print

// Specific venues such as conference or meeting rooms may be explicitly specified using this property. An alternate
// representation may be specified that is a URI that points to directory information with more structured specification
// of the location. For example, the alternate representation may specify either an LDAP URI pointing to an LDAP server
// entry or a CID URI pointing to a MIME body part containing a vCard [RFC 2426] for the location.
type Location struct {
	value  string
	altrep *url.URL
}

// creates a new icalendar location representation
func NewLocation(value string, altrep ...*url.URL) *Location {
	loc := &Location{value: value}
	if len(altrep) > 0 {
		loc.altrep = altrep[0]
	}
	return loc
}

// returns an alternate representation for the location
// if one exists
func (l *Location) AltRep() *url.URL {
	return l.altrep
}

// encodes the location for the iCalendar specification
func (l *Location) EncodeICalValue() (string, error) {
	return l.value, nil
}

// decodes the location from the iCalendar specification
func (l *Location) DecodeICalValue(value string) error {
	l.value = value
	return nil
}

// encodes the location params for the iCalendar specification
func (l *Location) EncodeICalParams() (params properties.Params, err error) {
	if l.altrep != nil {
		params = properties.Params{properties.AlternateRepresentationName: l.altrep.String()}
	}
	return
}

// decodes the location params from the iCalendar specification
func (l *Location) DecodeICalParams(params properties.Params) error {
	if rep, found := params[properties.AlternateRepresentationName]; !found {
		return nil
	} else if altrep, err := url.Parse(rep); err != nil {
		return utils.NewError(l.DecodeICalValue, "unable to parse alternate representation", l, err)
	} else {
		l.altrep = altrep
		return nil
	}
}

// validates the location against the iCalendar specification
func (l *Location) ValidateICalValue() error {

	if l.altrep != nil {
		if _, err := url.Parse(l.altrep.String()); err != nil {
			msg := "location alternate representation must be a valid url"
			return utils.NewError(l.ValidateICalValue, msg, l, err)
		}
	}

	return nil
}
