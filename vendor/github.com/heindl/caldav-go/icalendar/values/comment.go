package values

import (
	"github.com/heindl/caldav-go/icalendar/properties"
)

// specifies non-processing information intended to provide a comment to the calendar user.
type Comment string

// encodes the comment value for the iCalendar specification
func (c Comment) EncodeICalValue() (string, error) {
	return string(c), nil
}

// decodes the comment value from the iCalendar specification
func (c Comment) DecodeICalValue(value string) error {
	c = Comment(value)
	return nil
}

// encodes the comment value for the iCalendar specification
func (c Comment) EncodeICalName() (properties.PropertyName, error) {
	return properties.CommentPropertyName, nil
}

// creates a list of comments from strings
func NewComments(comments ...string) []Comment {
	var _comments []Comment
	for _, comment := range comments {
		_comments = append(_comments, Comment(comment))
	}
	return _comments
}
