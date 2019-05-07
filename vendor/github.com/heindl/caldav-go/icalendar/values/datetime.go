package values

import (
	"fmt"
	"github.com/heindl/caldav-go/icalendar/properties"
	"github.com/heindl/caldav-go/utils"
	"log"
	"strings"
	"time"
)

var _ = log.Print
var tzidDict map[string]string

const DateFormatString = "20060102"
const DateTimeFormatString = "20060102T150405"
const UTCDateTimeFormatString = "20060102T150405Z"

// a representation of a date and time for iCalendar
type DateTime struct {
	t time.Time
}

type DateTimes []*DateTime

// The exception dates, if specified, are used in computing the recurrence set. The recurrence set is the complete set
// of recurrence instances for a calendar component. The recurrence set is generated by considering the initial
// "DTSTART" property along with the "RRULE", "RDATE", "EXDATE" and "EXRULE" properties contained within the iCalendar
// object. The "DTSTART" property defines the first instance in the recurrence set. Multiple instances of the "RRULE"
// and "EXRULE" properties can also be specified to define more sophisticated recurrence sets. The final recurrence set
// is generated by gathering all of the start date-times generated by any of the specified "RRULE" and "RDATE"
// properties, and then excluding any start date and times which fall within the union of start date and times
// generated by any specified "EXRULE" and "EXDATE" properties. This implies that start date and times within exclusion
// related properties (i.e., "EXDATE" and "EXRULE") take precedence over those specified by inclusion properties
// (i.e., "RDATE" and "RRULE"). Where duplicate instances are generated by the "RRULE" and "RDATE" properties, only
// one recurrence is considered. Duplicate instances are ignored.
//
// The "EXDATE" property can be used to exclude the value specified in "DTSTART". However, in such cases the original
// "DTSTART" date MUST still be maintained by the calendaring and scheduling system because the original "DTSTART"
// value has inherent usage dependencies by other properties such as the "RECURRENCE-ID".
type ExceptionDateTimes DateTimes

// The recurrence dates, if specified, are used in computing the recurrence set. The recurrence set is the complete set
// of recurrence instances for a calendar component. The recurrence set is generated by considering the initial
// "DTSTART" property along with the "RRULE", "RDATE", "EXDATE" and "EXRULE" properties contained within the iCalendar
// object. The "DTSTART" property defines the first instance in the recurrence set. Multiple instances of the "RRULE"
// and "EXRULE" properties can also be specified to define more sophisticated recurrence sets. The final recurrence set
// is generated by gathering all of the start date-times generated by any of the specified "RRULE" and "RDATE"
// properties, and then excluding any start date and times which fall within the union of start date and times
// generated by any specified "EXRULE" and "EXDATE" properties. This implies that start date and times within exclusion
// related properties (i.e., "EXDATE" and "EXRULE") take precedence over those specified by inclusion properties
// (i.e., "RDATE" and "RRULE"). Where duplicate instances are generated by the "RRULE" and "RDATE" properties, only
// one recurrence is considered. Duplicate instances are ignored.
type RecurrenceDateTimes DateTimes

// creates a new icalendar datetime representation
func NewDateTime(t time.Time) *DateTime {
	return &DateTime{t: t.Truncate(time.Second)}
}

// creates a new icalendar datetime array representation
func NewDateTimes(dates ...*DateTime) DateTimes {
	return DateTimes(dates)
}

// creates a new icalendar datetime array representation
func NewExceptionDateTimes(dates ...*DateTime) *ExceptionDateTimes {
	datetimes := NewDateTimes(dates...)
	return (*ExceptionDateTimes)(&datetimes)
}

// creates a new icalendar datetime array representation
func NewRecurrenceDateTimes(dates ...*DateTime) *RecurrenceDateTimes {
	datetimes := NewDateTimes(dates...)
	return (*RecurrenceDateTimes)(&datetimes)
}

// checks to see if two datetimes are equal
func (d *DateTime) Equals(test *DateTime) bool {
	return d.t.Equal(test.t)
}

// returns the native time for the datetime object
func (d *DateTime) NativeTime() time.Time {
	return d.t
}

// encodes the datetime value for the iCalendar specification
func (d *DateTime) EncodeICalValue() (string, error) {
	val := d.t.Format(DateTimeFormatString)
	loc := d.t.Location()
	if loc == time.UTC {
		val = fmt.Sprintf("%sZ", val)
	}
	return val, nil
}

// decodes the datetime value from the iCalendar specification
func (d *DateTime) DecodeICalValue(value string) error {
	layout := DateTimeFormatString
	if strings.HasSuffix(value, "Z") {
		layout = UTCDateTimeFormatString
	} else if len(value) == 8 {
		layout = DateFormatString
	}
	var err error
	d.t, err = time.ParseInLocation(layout, value, time.UTC)
	if err != nil {
		return utils.NewError(d.DecodeICalValue, "unable to parse datetime value", d, err)
	} else {
		return nil
	}
}

// encodes the datetime params for the iCalendar specification
func (d *DateTime) EncodeICalParams() (params properties.Params, err error) {
	loc := d.t.Location()
	if loc != time.UTC {
		params = properties.Params{properties.TimeZoneIdPropertyName: loc.String()}
	}
	return
}

// decodes the datetime params from the iCalendar specification
func (d *DateTime) DecodeICalParams(params properties.Params) error {
	layout := DateTimeFormatString
	value := d.t.Format(layout)
	name, found := params[properties.TimeZoneIdPropertyName]
	if !found {
		return nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Attempt to get Olson's time
		olson := tzidDict[name]
		if olson != "" {
			loc, err = time.LoadLocation(olson)
			if err != nil {
				return utils.NewError(d.DecodeICalValue, "unable to parse timezone after converting to Olson's time", d, err)
			}
		} else {
			return utils.NewError(d.DecodeICalValue, "unable to parse timezone", d, err)
		}
	}
	if t, err := time.ParseInLocation(layout, value, loc); err != nil {
		return utils.NewError(d.DecodeICalValue, "unable to parse datetime value", d, err)
	} else {
		d.t = t
		return nil
	}
}

// validates the datetime value against the iCalendar specification
func (d *DateTime) ValidateICalValue() error {

	loc := d.t.Location()

	if loc == time.Local {
		msg := "DateTime location may not Local, please use UTC or explicit Location"
		return utils.NewError(d.ValidateICalValue, msg, d, nil)
	}

	if loc.String() == "" {
		msg := "DateTime location must have a valid name"
		return utils.NewError(d.ValidateICalValue, msg, d, nil)
	}

	return nil
}

// encodes the datetime value for the iCalendar specification
func (d *DateTime) String() string {
	if s, err := d.EncodeICalValue(); err != nil {
		panic(err)
	} else {
		return s
	}
}

// encodes a list of datetime values for the iCalendar specification
func (ds *DateTimes) EncodeICalValue() (string, error) {
	var csv CSV
	for i, d := range *ds {
		if s, err := d.EncodeICalValue(); err != nil {
			msg := fmt.Sprintf("unable to encode datetime at index %d", i)
			return "", utils.NewError(ds.EncodeICalValue, msg, ds, err)
		} else {
			csv = append(csv, s)
		}
	}
	return csv.EncodeICalValue()
}

// encodes a list of datetime params for the iCalendar specification
func (ds *DateTimes) EncodeICalParams() (params properties.Params, err error) {
	if len(*ds) > 0 {
		params, err = (*ds)[0].EncodeICalParams()
	}
	return
}

// decodes a list of datetime params from the iCalendar specification
func (ds *DateTimes) DecodeICalParams(params properties.Params) error {
	for i, d := range *ds {
		if err := d.DecodeICalParams(params); err != nil {
			msg := fmt.Sprintf("unable to decode datetime params for index %d", i)
			return utils.NewError(ds.DecodeICalValue, msg, ds, err)
		}
	}
	return nil
}

// encodes a list of datetime values for the iCalendar specification
func (ds *DateTimes) DecodeICalValue(value string) error {
	csv := new(CSV)
	if err := csv.DecodeICalValue(value); err != nil {
		return utils.NewError(ds.DecodeICalValue, "unable to decode datetime list as CSV", ds, err)
	}
	for i, value := range *csv {
		d := new(DateTime)
		if err := d.DecodeICalValue(value); err != nil {
			msg := fmt.Sprintf("unable to decode datetime at index %d", i)
			return utils.NewError(ds.DecodeICalValue, msg, ds, err)
		} else {
			*ds = append(*ds, d)
		}
	}
	return nil
}

// encodes exception date times property name for icalendar
func (e *ExceptionDateTimes) EncodeICalName() (properties.PropertyName, error) {
	return properties.ExceptionDateTimesPropertyName, nil
}

// encodes recurrence date times property name for icalendar
func (r *RecurrenceDateTimes) EncodeICalName() (properties.PropertyName, error) {
	return properties.RecurrenceDateTimesPropertyName, nil
}

// encodes exception date times property value for icalendar
func (e *ExceptionDateTimes) EncodeICalValue() (string, error) {
	return (*DateTimes)(e).EncodeICalValue()
}

// encodes recurrence date times property value for icalendar
func (r *RecurrenceDateTimes) EncodeICalValue() (string, error) {
	return (*DateTimes)(r).EncodeICalValue()
}

// decodes exception date times property value for icalendar
func (e *ExceptionDateTimes) DecodeICalValue(value string) error {
	return (*DateTimes)(e).DecodeICalValue(value)
}

// decodes recurrence date times property value for icalendar
func (r *RecurrenceDateTimes) DecodeICalValue(value string) error {
	return (*DateTimes)(r).DecodeICalValue(value)
}

// encodes exception date times property params for icalendar
func (e *ExceptionDateTimes) EncodeICalParams() (params properties.Params, err error) {
	return (*DateTimes)(e).EncodeICalParams()
}

// encodes recurrence date times property params for icalendar
func (r *RecurrenceDateTimes) EncodeICalParams() (params properties.Params, err error) {
	return (*DateTimes)(r).EncodeICalParams()
}

// encodes exception date times property params for icalendar
func (e *ExceptionDateTimes) DecodeICalParams(params properties.Params) error {
	return (*DateTimes)(e).DecodeICalParams(params)
}

// encodes recurrence date times property params for icalendar
func (r *RecurrenceDateTimes) DecodeICalParams(params properties.Params) error {
	return (*DateTimes)(r).DecodeICalParams(params)
}

func init() {
	tzidDict = map[string]string{
		"AUS Central Standard Time":       "Australia/Darwin",
		"AUS Eastern Standard Time":       "Australia/Sydney",
		"Afghanistan Standard Time":       "Asia/Kabul",
		"Alaskan Standard Time":           "America/Anchorage",
		"Arab Standard Time":              "Asia/Riyadh",
		"Arabian Standard Time":           "Asia/Dubai",
		"Arabic Standard Time":            "Asia/Baghdad",
		"Argentina Standard Time":         "America/Buenos_Aires",
		"Atlantic Standard Time":          "America/Halifax",
		"Azerbaijan Standard Time":        "Asia/Baku",
		"Azores Standard Time":            "Atlantic/Azores",
		"Bahia Standard Time":             "America/Bahia",
		"Bangladesh Standard Time":        "Asia/Dhaka",
		"Belarus Standard Time":           "Europe/Minsk",
		"Canada Central Standard Time":    "America/Regina",
		"Cape Verde Standard Time":        "Atlantic/Cape_Verde",
		"Caucasus Standard Time":          "Asia/Yerevan",
		"Cen. Australia Standard Time":    "Australia/Adelaide",
		"Central America Standard Time":   "America/Guatemala",
		"Central Asia Standard Time":      "Asia/Almaty",
		"Central Brazilian Standard Time": "America/Cuiaba",
		"Central Europe Standard Time":    "Europe/Budapest",
		"Central European Standard Time":  "Europe/Warsaw",
		"Central Pacific Standard Time":   "Pacific/Guadalcanal",
		"Central Standard Time":           "America/Chicago",
		"Central Standard Time (Mexico)":  "America/Mexico_City",
		"China Standard Time":             "Asia/Shanghai",
		"Dateline Standard Time":          "Etc/GMT+12",
		"E. Africa Standard Time":         "Africa/Nairobi",
		"E. Australia Standard Time":      "Australia/Brisbane",
		"E. Europe Standard Time":         "Europe/Chisinau",
		"E. South America Standard Time":  "America/Sao_Paulo",
		"Eastern Standard Time":           "America/New_York",
		"Eastern Standard Time (Mexico)":  "America/Cancun",
		"Egypt Standard Time":             "Africa/Cairo",
		"Ekaterinburg Standard Time":      "Asia/Yekaterinburg",
		"FLE Standard Time":               "Europe/Kiev",
		"Fiji Standard Time":              "Pacific/Fiji",
		"GMT Standard Time":               "Europe/London",
		"GTB Standard Time":               "Europe/Bucharest",
		"Georgian Standard Time":          "Asia/Tbilisi",
		"Greenland Standard Time":         "America/Godthab",
		"Greenwich Standard Time":         "Atlantic/Reykjavik",
		"Hawaiian Standard Time":          "Pacific/Honolulu",
		"India Standard Time":             "Asia/Calcutta",
		"Iran Standard Time":              "Asia/Tehran",
		"Israel Standard Time":            "Asia/Jerusalem",
		"Jordan Standard Time":            "Asia/Amman",
		"Kaliningrad Standard Time":       "Europe/Kaliningrad",
		"Korea Standard Time":             "Asia/Seoul",
		"Libya Standard Time":             "Africa/Tripoli",
		"Line Islands Standard Time":      "Pacific/Kiritimati",
		"Magadan Standard Time":           "Asia/Magadan",
		"Mauritius Standard Time":         "Indian/Mauritius",
		"Middle East Standard Time":       "Asia/Beirut",
		"Montevideo Standard Time":        "America/Montevideo",
		"Morocco Standard Time":           "Africa/Casablanca",
		"Mountain Standard Time":          "America/Denver",
		"Mountain Standard Time (Mexico)": "America/Chihuahua",
		"Myanmar Standard Time":           "Asia/Rangoon",
		"N. Central Asia Standard Time":   "Asia/Novosibirsk",
		"Namibia Standard Time":           "Africa/Windhoek",
		"Nepal Standard Time":             "Asia/Katmandu",
		"New Zealand Standard Time":       "Pacific/Auckland",
		"Newfoundland Standard Time":      "America/St_Johns",
		"North Asia East Standard Time":   "Asia/Irkutsk",
		"North Asia Standard Time":        "Asia/Krasnoyarsk",
		"North Korea Standard Time":       "Asia/Pyongyang",
		"Pacific SA Standard Time":        "America/Santiago",
		"Pacific Standard Time":           "America/Los_Angeles",
		"Pakistan Standard Time":          "Asia/Karachi",
		"Paraguay Standard Time":          "America/Asuncion",
		"Romance Standard Time":           "Europe/Paris",
		"Russia Time Zone 10":             "Asia/Srednekolymsk",
		"Russia Time Zone 11":             "Asia/Kamchatka",
		"Russia Time Zone 3":              "Europe/Samara",
		"Russian Standard Time":           "Europe/Moscow",
		"SA Eastern Standard Time":        "America/Cayenne",
		"SA Pacific Standard Time":        "America/Bogota",
		"SA Western Standard Time":        "America/La_Paz",
		"SE Asia Standard Time":           "Asia/Bangkok",
		"Samoa Standard Time":             "Pacific/Apia",
		"Singapore Standard Time":         "Asia/Singapore",
		"South Africa Standard Time":      "Africa/Johannesburg",
		"Sri Lanka Standard Time":         "Asia/Colombo",
		"Syria Standard Time":             "Asia/Damascus",
		"Taipei Standard Time":            "Asia/Taipei",
		"Tasmania Standard Time":          "Australia/Hobart",
		"Tokyo Standard Time":             "Asia/Tokyo",
		"Tonga Standard Time":             "Pacific/Tongatapu",
		"Turkey Standard Time":            "Europe/Istanbul",
		"US Eastern Standard Time":        "America/Indianapolis",
		"US Mountain Standard Time":       "America/Phoenix",
		"UTC":                             "Etc/GMT",
		"UTC+12":                          "Etc/GMT-12",
		"UTC-02":                          "Etc/GMT+2",
		"UTC-11":                          "Etc/GMT+11",
		"Ulaanbaatar Standard Time":       "Asia/Ulaanbaatar",
		"Venezuela Standard Time":         "America/Caracas",
		"Vladivostok Standard Time":       "Asia/Vladivostok",
		"W. Australia Standard Time":      "Australia/Perth",
		"W. Central Africa Standard Time": "Africa/Lagos",
		"W. Europe Standard Time":         "Europe/Berlin",
		"West Asia Standard Time":         "Asia/Tashkent",
		"West Pacific Standard Time":      "Pacific/Port_Moresby",
		"Yakutsk Standard Time":           "Asia/Yakutsk",
	}
}
