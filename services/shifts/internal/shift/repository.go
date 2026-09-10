package shift

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrNotFound is the domain error for "no matching row" — gorm.ErrRecordNotFound
// is translated here, so callers above this layer (HTTP) don't need to know
// about GORM to tell a 404 apart from a real error. Shared by both entities
// below: the caller already knows which one it asked for, no need for a
// separate error per type.
var ErrNotFound = errors.New("not found")

// ErrBookingNotPending is returned by DecideBooking when the booking exists
// but is no longer pending — already decided by someone else, or cancelled.
// Distinct from ErrNotFound so the caller can tell "doesn't exist" (404)
// apart from "exists, but this decision no longer applies" (409).
var ErrBookingNotPending = errors.New("booking is not pending")

// ErrUnknownVolunteer is returned by CreateConfirmedBooking when
// volunteer_id doesn't reference a real registry.users row — translated
// from gorm.ErrForeignKeyViolated (available because TranslateError is set
// in cmd/server/main.go) so the caller gets a clean 400, not a raw DB error.
var ErrUnknownVolunteer = errors.New("unknown volunteer")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("shift: ping: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

func (r *Repository) CountTemplates(ctx context.Context) (int, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&ShiftTemplate{}).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("shift: count templates: %w", err)
	}
	return int(n), nil
}

// CreateTemplate ignores any id the caller passed in t, and always creates
// an active template (see the comment on ShiftTemplate.Active — no DB
// default, forced here instead, to sidestep the GORM zero-value-omits-insert
// gotcha for a bool "default"). Deactivating an existing one is a separate
// concern for whichever endpoint builds on this method.
func (r *Repository) CreateTemplate(ctx context.Context, t ShiftTemplate) (ShiftTemplate, error) {
	t.ID = ""
	t.Active = true
	if err := r.db.WithContext(ctx).Create(&t).Error; err != nil {
		return ShiftTemplate{}, fmt.Errorf("shift: create template: %w", err)
	}
	return t, nil
}

func (r *Repository) GetTemplate(ctx context.Context, id string) (ShiftTemplate, error) {
	var t ShiftTemplate
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ShiftTemplate{}, ErrNotFound
	}
	if err != nil {
		return ShiftTemplate{}, fmt.Errorf("shift: get template: %w", err)
	}
	return t, nil
}

func (r *Repository) ListTemplates(ctx context.Context) ([]ShiftTemplate, error) {
	// []ShiftTemplate{}, not a nil slice: encodes to `[]` in JSON, not `null`.
	result := []ShiftTemplate{}
	if err := r.db.WithContext(ctx).Order("weekday, start_time").Find(&result).Error; err != nil {
		return nil, fmt.Errorf("shift: list templates: %w", err)
	}
	return result, nil
}

// ListActiveTemplates is ListTemplates filtered to active ones — the only
// ones relevant when generating calendar occurrences (see ListOccurrences).
func (r *Repository) ListActiveTemplates(ctx context.Context) ([]ShiftTemplate, error) {
	result := []ShiftTemplate{}
	if err := r.db.WithContext(ctx).Where("active").Order("weekday, start_time").Find(&result).Error; err != nil {
		return nil, fmt.Errorf("shift: list active templates: %w", err)
	}
	return result, nil
}

// UpdateTemplate replaces weekday/start_time/end_time/label/active for id.
// A map, not `.Updates(t)`: GORM's struct-based Updates skips Go zero
// values (same gotcha as Create, see the comment on ShiftTemplate.Active)
// — a caller setting Active to false would otherwise be silently ignored.
func (r *Repository) UpdateTemplate(ctx context.Context, id string, t ShiftTemplate) (ShiftTemplate, error) {
	result := r.db.WithContext(ctx).Model(&ShiftTemplate{}).Where("id = ?", id).Updates(map[string]any{
		"weekday":    t.Weekday,
		"start_time": t.StartTime,
		"end_time":   t.EndTime,
		"label":      t.Label,
		"active":     t.Active,
	})
	if result.Error != nil {
		return ShiftTemplate{}, fmt.Errorf("shift: update template: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ShiftTemplate{}, ErrNotFound
	}
	return r.GetTemplate(ctx, id)
}

// CreateBooking ignores any id/status/decided_by/decided_at the caller
// passed in b — cleared before Create so GORM lets the DB default
// ("pending") apply instead of inserting a client-supplied value. Every
// booking starts pending; confirming one already at creation time (a
// manager's direct booking, no approval needed) is a separate follow-up
// concern for whichever endpoint builds on this method.
func (r *Repository) CreateBooking(ctx context.Context, b Booking) (Booking, error) {
	b.ID = ""
	b.Status = ""
	b.DecidedBy = nil
	b.DecidedAt = nil
	if err := r.db.WithContext(ctx).Create(&b).Error; err != nil {
		return Booking{}, fmt.Errorf("shift: create booking: %w", err)
	}
	return b, nil
}

// CreateConfirmedBooking is CreateBooking's counterpart for a shift
// manager's direct booking: no pending step, the booking is already
// decided at creation time. Doesn't reuse CreateBooking (which always
// forces status back to pending) — different, purpose-built contract
// instead of one method branching on a flag.
func (r *Repository) CreateConfirmedBooking(ctx context.Context, b Booking, decidedBy string) (Booking, error) {
	b.ID = ""
	b.Status = BookingStatusConfirmed
	b.DecidedBy = &decidedBy
	now := time.Now()
	b.DecidedAt = &now
	if err := r.db.WithContext(ctx).Create(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			return Booking{}, ErrUnknownVolunteer
		}
		return Booking{}, fmt.Errorf("shift: create confirmed booking: %w", err)
	}
	return b, nil
}

// CreateBookingsAtomic inserts every booking in bookings inside a single DB
// transaction — the caller has already validated each one individually
// (see handleCreateBulkBooking), this only guarantees the multi-row insert
// itself is all-or-nothing: if any row fails to insert, every row already
// inserted in this call rolls back too.
func (r *Repository) CreateBookingsAtomic(ctx context.Context, bookings []Booking) ([]Booking, error) {
	created := make([]Booking, 0, len(bookings))
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, b := range bookings {
			b.ID = ""
			b.Status = ""
			b.DecidedBy = nil
			b.DecidedAt = nil
			if err := tx.Create(&b).Error; err != nil {
				return err
			}
			created = append(created, b)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("shift: create bookings atomic: %w", err)
	}
	return created, nil
}

func (r *Repository) GetBooking(ctx context.Context, id string) (Booking, error) {
	var b Booking
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Booking{}, ErrNotFound
	}
	if err != nil {
		return Booking{}, fmt.Errorf("shift: get booking: %w", err)
	}
	return b, nil
}

func (r *Repository) ListBookings(ctx context.Context) ([]Booking, error) {
	result := []Booking{}
	if err := r.db.WithContext(ctx).Order("date, start_time").Find(&result).Error; err != nil {
		return nil, fmt.Errorf("shift: list bookings: %w", err)
	}
	return result, nil
}

// ListBookingsInRange returns bookings whose date falls within [from, to],
// both inclusive — the raw material ListOccurrences uses to compute
// coverage, not meant to be returned to a client as-is.
func (r *Repository) ListBookingsInRange(ctx context.Context, from, to time.Time) ([]Booking, error) {
	result := []Booking{}
	if err := r.db.WithContext(ctx).Where("date BETWEEN ? AND ?", from, to).Find(&result).Error; err != nil {
		return nil, fmt.Errorf("shift: list bookings in range: %w", err)
	}
	return result, nil
}

// occurrenceKey identifies one calendar slot: a template on a specific day.
type occurrenceKey struct {
	templateID string
	date       string // time.Time isn't a valid map key across different locations/monotonic readings; the DATE-only value formatted as YYYY-MM-DD is.
}

// ListOccurrences generates every occurrence of an active template within
// [from, to] (both inclusive) and attaches its coverage status by looking
// up bookings in the same range — see docs/backlog.md "Gestione turni",
// voce 3. Occurrences are never stored: a day/slot combination only exists
// as the join of a template's weekday against the calendar.
//
// Status precedence when multiple bookings exist for the same slot+date
// (e.g. several pending requests before one is confirmed): confirmed beats
// pending beats free. rejected/cancelled bookings never affect status.
//
// callerID additionally populates MyBookingStatus on each occurrence where
// callerID itself has a pending/confirmed booking — the caller is always
// authenticated (this repository method backs an endpoint behind
// shifts:read), so this is never empty in practice.
func (r *Repository) ListOccurrences(ctx context.Context, from, to time.Time, callerID string) ([]Occurrence, error) {
	templates, err := r.ListActiveTemplates(ctx)
	if err != nil {
		return nil, err
	}
	bookings, err := r.ListBookingsInRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	statusByKey := make(map[occurrenceKey]OccurrenceStatus, len(bookings))
	mineByKey := make(map[occurrenceKey]BookingStatus, len(bookings))
	for _, b := range bookings {
		if b.Status != BookingStatusPending && b.Status != BookingStatusConfirmed {
			continue
		}
		key := occurrenceKey{templateID: b.TemplateID, date: b.Date.Format("2006-01-02")}
		status := OccurrenceStatusPending
		if b.Status == BookingStatusConfirmed {
			status = OccurrenceStatusConfirmed
		}
		if status == OccurrenceStatusConfirmed || statusByKey[key] != OccurrenceStatusConfirmed {
			statusByKey[key] = status
		}
		if b.VolunteerID == callerID {
			mineByKey[key] = b.Status
		}
	}

	// Date-outer, template-inner: templates are already ordered by
	// (weekday, start_time), so this loop produces occurrences already
	// sorted by (date, start_time) — no separate sort needed.
	occurrences := []Occurrence{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		weekday := int(d.Weekday())
		for _, t := range templates {
			if t.Weekday != weekday {
				continue
			}
			key := occurrenceKey{templateID: t.ID, date: d.Format("2006-01-02")}
			status := OccurrenceStatusFree
			if s, ok := statusByKey[key]; ok {
				status = s
			}
			var myStatus *BookingStatus
			if s, ok := mineByKey[key]; ok {
				myStatus = &s
			}
			occurrences = append(occurrences, Occurrence{
				TemplateID:      t.ID,
				Date:            d,
				Weekday:         weekday,
				StartTime:       t.StartTime,
				EndTime:         t.EndTime,
				Label:           t.Label,
				Status:          status,
				MyBookingStatus: myStatus,
			})
		}
	}
	return occurrences, nil
}

// HasConfirmedBooking reports whether a template+date slot is already
// occupied by a confirmed booking — the availability check a new request
// must pass (see handleCreateBooking).
func (r *Repository) HasConfirmedBooking(ctx context.Context, templateID string, date time.Time) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Booking{}).
		Where("template_id = ? AND date = ? AND status = ?", templateID, date, BookingStatusConfirmed).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("shift: has confirmed booking: %w", err)
	}
	return n > 0, nil
}

// HasBookingForVolunteer reports whether volunteerID already has a
// pending or confirmed booking for this template+date — rejected/cancelled
// ones don't count, a volunteer can always request again after either.
func (r *Repository) HasBookingForVolunteer(ctx context.Context, templateID string, date time.Time, volunteerID string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Booking{}).
		Where("template_id = ? AND date = ? AND volunteer_id = ? AND status IN ?", templateID, date, volunteerID,
			[]BookingStatus{BookingStatusPending, BookingStatusConfirmed}).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("shift: has booking for volunteer: %w", err)
	}
	return n > 0, nil
}

// CountPendingBookings is the shift manager's in-app badge count — global,
// not scoped to any date range (unlike ListOccurrences), so it stays
// accurate regardless of what the calendar happens to be showing.
func (r *Repository) CountPendingBookings(ctx context.Context) (int, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&Booking{}).Where("status = ?", BookingStatusPending).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("shift: count pending bookings: %w", err)
	}
	return int(n), nil
}

// DecideBooking approves or rejects a pending booking — status must be
// BookingStatusConfirmed or BookingStatusRejected (the caller validates
// that before reaching here). The WHERE clause requires status='pending'
// so the transition itself is atomic: two concurrent decisions on the same
// booking can't both succeed, whichever loses the race gets
// ErrBookingNotPending below instead of silently overwriting the other.
func (r *Repository) DecideBooking(ctx context.Context, id string, status BookingStatus, decidedBy string) (Booking, error) {
	result := r.db.WithContext(ctx).Model(&Booking{}).
		Where("id = ? AND status = ?", id, BookingStatusPending).
		Updates(map[string]any{
			"status":     status,
			"decided_by": decidedBy,
			"decided_at": time.Now(),
		})
	if result.Error != nil {
		return Booking{}, fmt.Errorf("shift: decide booking: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		// RowsAffected == 0 means either the id doesn't exist, or it does
		// but isn't pending anymore — GetBooking tells these apart.
		if _, err := r.GetBooking(ctx, id); err != nil {
			return Booking{}, err
		}
		return Booking{}, ErrBookingNotPending
	}
	return r.GetBooking(ctx, id)
}
