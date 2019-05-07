package values

// In a group scheduled calendar component, the property is used by the "Organizer" to provide a confirmation of the
// event to the "Attendees".
// For example in an Event calendar component, the "Organizer" can indicate that a meeting is tentative, confirmed or
// cancelled.
type EventStatus string

const (
	TentativeEventStatus EventStatus = "TENTATIVE" // Indicates event is tentative.
	ConfirmedEventStatus             = "CONFIRMED" // Indicates event is definite.
	CancelledEventStatus             = "CANCELLED" // Indicates event is cancelled.
)
