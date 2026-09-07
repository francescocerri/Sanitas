package shift

import "time"

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "pending"
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusRejected  BookingStatus = "rejected"
	BookingStatusCancelled BookingStatus = "cancelled"
)

// Booking is a volunteer's booking of a concrete occurrence of a
// ShiftTemplate (one specific day + slot) — see
// docs/adr/0025-modello-dati-turni.md. StartTime/EndTime are a snapshot
// copied from the template at request time: a later edit to the template
// doesn't retroactively change bookings already made.
type Booking struct {
	ID          string        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id,omitempty"`
	TemplateID  string        `gorm:"column:template_id;type:uuid;not null" json:"template_id"`
	VolunteerID string        `gorm:"column:volunteer_id;type:uuid;not null" json:"volunteer_id"`
	Date        time.Time     `gorm:"type:date;not null" json:"date"`
	StartTime   string        `gorm:"column:start_time;not null" json:"start_time"`
	EndTime     string        `gorm:"column:end_time;not null" json:"end_time"`
	Status      BookingStatus `gorm:"not null;default:pending" json:"status,omitempty"`
	// DecidedBy/DecidedAt stay nil while the request is pending; for a
	// manager's direct booking (no approval needed) they're already set at
	// creation time.
	DecidedBy *string    `gorm:"column:decided_by;type:uuid" json:"decided_by,omitempty"`
	DecidedAt *time.Time `gorm:"column:decided_at" json:"decided_at,omitempty"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (Booking) TableName() string { return "bookings" }
