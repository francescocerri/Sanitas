package shift

import "time"

// OccurrenceStatus is computed, not stored: it reflects whether a booking
// exists for that occurrence+role, not a column on any table.
type OccurrenceStatus string

const (
	OccurrenceStatusFree      OccurrenceStatus = "free"
	OccurrenceStatusPending   OccurrenceStatus = "pending"
	OccurrenceStatusConfirmed OccurrenceStatus = "confirmed"
)

// RoleCoverage is the coverage status of one of the 4 BookingRole slots on
// a given Occurrence. MyBookingStatus is nil unless the caller has a
// pending/confirmed booking of their own for this specific role — distinct
// from Status (the aggregate coverage across every volunteer for that
// role) so a client can render "this one is yours" regardless of who else
// has requested it.
type RoleCoverage struct {
	Role            BookingRole      `json:"role"`
	Status          OccurrenceStatus `json:"status"`
	MyBookingStatus *BookingStatus   `json:"my_booking_status,omitempty"`
}

// Occurrence is a concrete calendar day on which a ShiftTemplate recurs,
// combined with the coverage of each of its 4 roles — the shared data
// behind every calendar view (volunteer or shift manager). Never
// persisted: generated on the fly by Repository.ListOccurrences from active
// templates and the bookings in range. Roles always has exactly 4 entries,
// one per BookingRole in AllBookingRoles order, even when none of them
// have ever been booked.
type Occurrence struct {
	TemplateID string         `json:"template_id"`
	Date       time.Time      `json:"date"`
	Weekday    int            `json:"weekday"`
	StartTime  string         `json:"start_time"`
	EndTime    string         `json:"end_time"`
	Label      string         `json:"label"`
	Roles      []RoleCoverage `json:"roles"`
}
