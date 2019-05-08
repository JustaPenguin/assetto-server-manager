package values

import (
	"fmt"
	"github.com/heindl/caldav-go/utils"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var _ = log.Print

// a representation of duration for iCalendar
type Duration struct {
	d time.Duration
}

// breaks apart the duration into its component time parts
func (d *Duration) Decompose() (weeks, days, hours, minutes, seconds int64) {

	// chip away at this
	rem := time.Duration(math.Abs(float64(d.d)))

	div := time.Hour * 24 * 7
	weeks = int64(rem / div)
	rem = rem % div
	div = div / 7
	days = int64(rem / div)
	rem = rem % div
	div = div / 24
	hours = int64(rem / div)
	rem = rem % div
	div = div / 60
	minutes = int64(rem / div)
	rem = rem % div
	div = div / 60
	seconds = int64(rem / div)

	return

}

// returns the native golang duration
func (d *Duration) NativeDuration() time.Duration {
	return d.d
}

// returns true if the duration is negative
func (d *Duration) IsPast() bool {
	return d.d < 0
}

// encodes the duration of time into iCalendar format
func (d *Duration) EncodeICalValue() (string, error) {
	var parts []string
	weeks, days, hours, minutes, seconds := d.Decompose()
	if d.IsPast() {
		parts = append(parts, "-")
	}
	parts = append(parts, "P")
	if weeks > 0 {
		parts = append(parts, fmt.Sprintf("%dW", weeks))
	}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dD", days))
	}
	if hours > 0 || minutes > 0 || seconds > 0 {
		parts = append(parts, "T")
		if hours > 0 {
			parts = append(parts, fmt.Sprintf("%dH", hours))
		}
		if minutes > 0 {
			parts = append(parts, fmt.Sprintf("%dM", minutes))
		}
		if seconds > 0 {
			parts = append(parts, fmt.Sprintf("%dS", seconds))
		}
	}
	return strings.Join(parts, ""), nil
}

var durationRegEx = regexp.MustCompile("(\\d+)(\\w)")

// decodes the duration of time from iCalendar format
func (d *Duration) DecodeICalValue(value string) error {
	var seconds int64
	var isPast = strings.HasPrefix(value, "-P")
	var matches = durationRegEx.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		var multiplier int64
		ivalue, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return utils.NewError(d.DecodeICalValue, "unable to decode duration value "+match[1], d, nil)
		}
		switch match[2] {
		case "S":
			multiplier = 1
		case "M":
			multiplier = 60
		case "H":
			multiplier = 60 * 60
		case "D":
			multiplier = 60 * 60 * 24
		case "W":
			multiplier = 60 * 60 * 24 * 7
		default:
			return utils.NewError(d.DecodeICalValue, "unable to decode duration segment "+match[2], d, nil)
		}
		seconds = seconds + multiplier*ivalue
	}
	d.d = time.Duration(seconds) * time.Second
	if isPast {
		d.d = -d.d
	}
	return nil
}

func (d *Duration) String() string {
	if s, err := d.EncodeICalValue(); err != nil {
		panic(err)
	} else {
		return s
	}
}

// creates a new iCalendar duration representation
func NewDuration(d time.Duration) *Duration {
	return &Duration{d: d}
}
