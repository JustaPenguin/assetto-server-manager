package values

// An access classification is only one component of the general security system within a calendar application.
// It provides a method of capturing the scope of the access the calendar owner intends for information within an
// individual calendar entry. The access classification of an individual iCalendar component is useful when measured
// along with the other security components of a calendar system (e.g., calendar user authentication, authorization,
// access rights, access role, etc.). Hence, the semantics of the individual access classifications cannot be completely
// defined by this memo alone. Additionally, due to the "blind" nature of most exchange processes using this memo, these
// access classifications cannot serve as an enforcement statement for a system receiving an iCalendar object. Rather,
// they provide a method for capturing the intention of the calendar owner for the access to the calendar component.
type EventAccessClassification string

const (
	PublicEventAccessClassification       EventAccessClassification = "PUBLIC"
	PrivateEventAccessClassification                                = "PRIVATE"
	ConfidentialEventAccessClassification                           = "CONFIDENTIAL"
)
