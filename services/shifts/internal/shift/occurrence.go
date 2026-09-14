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

// OperationalStatus is the past-tense verdict for an occurrence once its
// date is behind us: did the shift actually run, and with which crew.
// Nil while the occurrence hasn't happened yet (today included) — until
// then, the per-role Status/MyBookingStatus above are the only relevant
// signal (a shift can still gain/lose confirmations). Never stored:
// computed the same way as everything else in Occurrence, by
// Repository.ListOccurrences from the same confirmed bookings.
type OperationalStatus string

const (
	// OperationalStatusComplete: driver, leader and rescuer all confirmed.
	OperationalStatusComplete OperationalStatus = "complete"
	// OperationalStatusReduced: only driver and leader confirmed (rescuer
	// not) — still enough crew for the shift to have run.
	OperationalStatusReduced OperationalStatus = "reduced"
	// OperationalStatusClosed: driver or leader (or both) not confirmed —
	// not enough crew, the shift did not run.
	OperationalStatusClosed OperationalStatus = "closed"
)

// operationalStatusFor derives the operational verdict for a set of role
// coverages, once the occurrence's date has passed — see OperationalStatus.
// Driver and leader must BOTH be confirmed for the shift to have run at
// all (a domain rule, not derivable from RoleCoverage alone): the
// rescuer's presence only distinguishes complete from reduced, it never
// makes up for a missing driver or leader.
func operationalStatusFor(roles []RoleCoverage) OperationalStatus {
	var driverConfirmed, leaderConfirmed, rescuerConfirmed bool
	for _, rc := range roles {
		if rc.Status != OccurrenceStatusConfirmed {
			continue
		}
		switch rc.Role {
		case BookingRoleDriver:
			driverConfirmed = true
		case BookingRoleLeader:
			leaderConfirmed = true
		case BookingRoleRescuer:
			rescuerConfirmed = true
		}
	}
	switch {
	case driverConfirmed && leaderConfirmed && rescuerConfirmed:
		return OperationalStatusComplete
	case driverConfirmed && leaderConfirmed:
		return OperationalStatusReduced
	default:
		return OperationalStatusClosed
	}
}

// RoleCoverage is the coverage status of one of the 4 BookingRole slots on
// a given Occurrence. MyBookingStatus is nil unless the caller has a
// pending/confirmed booking of their own for this specific role — distinct
// from Status (the aggregate coverage across every volunteer for that
// role) so a client can render "this one is yours" regardless of who else
// has requested it. VolunteerID is set only when Status is confirmed: who
// is confirmed on a role is visible to anyone with shifts:read (see
// docs/adr/0025-modello-dati-turni.md "Aggiornamento"), but a merely
// pending request's identity stays hidden until it's decided.
type RoleCoverage struct {
	Role            BookingRole      `json:"role"`
	Status          OccurrenceStatus `json:"status"`
	MyBookingStatus *BookingStatus   `json:"my_booking_status,omitempty"`
	VolunteerID     *string          `json:"volunteer_id,omitempty"`
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

	// OperationalStatus is set only once Date is in the past — see
	// OperationalStatus.
	OperationalStatus *OperationalStatus `json:"operational_status,omitempty"`
}
