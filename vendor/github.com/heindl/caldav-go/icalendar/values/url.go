package values

import (
	"github.com/heindl/caldav-go/icalendar/properties"
	"github.com/heindl/caldav-go/utils"
	"net/url"
)

// a representation of duration for iCalendar
type Url struct {
	u url.URL
}

// encodes the URL into iCalendar format
func (u *Url) EncodeICalValue() (string, error) {
	return u.u.String(), nil
}

// encodes the url params for the iCalendar specification
func (u *Url) EncodeICalParams() (params properties.Params, err error) {
	params = properties.Params{
		properties.ValuePropertyName: "URI",
	}
	return
}

// decodes the URL from iCalendar format
func (u *Url) DecodeICalValue(value string) error {
	if parsed, err := url.Parse(value); err != nil {
		return utils.NewError(u.ValidateICalValue, "unable to parse url", u, err)
	} else {
		u.u = *parsed
		return nil
	}
}

// validates the URL for iCalendar format
func (u *Url) ValidateICalValue() error {
	if _, err := url.Parse(u.u.String()); err != nil {
		return utils.NewError(u.ValidateICalValue, "invalid URL object", u, err)
	} else {
		return nil
	}
}

// creates a new iCalendar duration representation
func NewUrl(u url.URL) *Url {
	return &Url{u: u}
}
