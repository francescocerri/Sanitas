package shift

import "time"

// OccurrenceStatus is computed, not stored: it reflects whether a booking
// exists for that occurrence, not a column on any table.
type OccurrenceStatus string

const (
	OccurrenceStatusFree      OccurrenceStatus = "free"
	OccurrenceStatusPending   OccurrenceStatus = "pending"
	OccurrenceStatusConfirmed OccurrenceStatus = "confirmed"
)

// Occurrence is a concrete calendar day on which a ShiftTemplate recurs,
// combined with the coverage status of that specific day+slot — the shared
// data behind every calendar view (volunteer or shift manager). Never
// persisted: generated on the fly by Repository.ListOccurrences from active
// templates and the bookings in range.
type Occurrence struct {
	TemplateID string           `json:"template_id"`
	Date       time.Time        `json:"date"`
	Weekday    int              `json:"weekday"`
	StartTime  string           `json:"start_time"`
	EndTime    string           `json:"end_time"`
	Label      string           `json:"label"`
	Status     OccurrenceStatus `json:"status"`
}
