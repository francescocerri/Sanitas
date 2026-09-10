package shift

import "time"

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "pending"
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusRejected  BookingStatus = "rejected"
	BookingStatusCancelled BookingStatus = "cancelled"
)

// BookingRole is one of the (fixed, not per-committee — a CRI crew's
// composition, not a Pavullo-specific choice) 4 positions a shift is made
// of. A volunteer/manager always books a specific role, never "the shift"
// as a whole — up to 4 independent bookings (one per role) can exist for
// the same template+date.
type BookingRole string

const (
	BookingRoleDriver   BookingRole = "driver"
	BookingRoleLeader   BookingRole = "leader"
	BookingRoleRescuer  BookingRole = "rescuer"
	BookingRoleObserver BookingRole = "observer"
)

// AllBookingRoles is the canonical order roles are validated against and
// listed in — both the HTTP layer (valid Role values) and
// Repository.ListOccurrences (generating one RoleCoverage per role) use
// this instead of duplicating the 4 values.
var AllBookingRoles = []BookingRole{
	BookingRoleDriver,
	BookingRoleLeader,
	BookingRoleRescuer,
	BookingRoleObserver,
}

// Booking is a volunteer's booking of a concrete occurrence of a
// ShiftTemplate (one specific day + slot) — see
// docs/adr/0025-modello-dati-turni.md. StartTime/EndTime are a snapshot
// copied from the template at request time: a later edit to the template
// doesn't retroactively change bookings already made.
type Booking struct {
	ID          string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id,omitempty"`
	TemplateID  string `gorm:"column:template_id;type:uuid;not null" json:"template_id"`
	VolunteerID string `gorm:"column:volunteer_id;type:uuid;not null" json:"volunteer_id"`
	// Role has no DB default: always explicit, like VolunteerID — a
	// booking always names which of the 4 positions it's for, see
	// BookingRole.
	Role      BookingRole   `gorm:"column:role;not null" json:"role"`
	Date      time.Time     `gorm:"type:date;not null" json:"date"`
	StartTime string        `gorm:"column:start_time;not null" json:"start_time"`
	EndTime   string        `gorm:"column:end_time;not null" json:"end_time"`
	Status    BookingStatus `gorm:"not null;default:pending" json:"status,omitempty"`
	// DecidedBy/DecidedAt stay nil while the request is pending; for a
	// manager's direct booking (no approval needed) they're already set at
	// creation time.
	DecidedBy *string    `gorm:"column:decided_by;type:uuid" json:"decided_by,omitempty"`
	DecidedAt *time.Time `gorm:"column:decided_at" json:"decided_at,omitempty"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (Booking) TableName() string { return "bookings" }
