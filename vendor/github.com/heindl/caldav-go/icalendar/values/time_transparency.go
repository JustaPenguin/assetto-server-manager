package values

// Time Transparency is the characteristic of an event that determines whether it appears to consume time on a calendar.
// Events that consume actual time for the individual or resource associated with the calendar SHOULD be recorded as
// OPAQUE, allowing them to be detected by free-busy time searches. Other events, which do not take up the individual's
// (or resource's) time SHOULD be recorded as TRANSPARENT, making them invisible to free-busy time searches.
type TimeTransparency string

const (
	OpaqueTimeTransparency      TimeTransparency = "OPAQUE"      // Blocks or opaque on busy time searches. DEFAULT
	TransparentTimeTransparency                  = "TRANSPARENT" // Transparent on busy time searches.
)
