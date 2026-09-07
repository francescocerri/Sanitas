package shift

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNotFound is the domain error for "no matching row" — gorm.ErrRecordNotFound
// is translated here, so callers above this layer (HTTP) don't need to know
// about GORM to tell a 404 apart from a real error. Shared by both entities
// below: the caller already knows which one it asked for, no need for a
// separate error per type.
var ErrNotFound = errors.New("not found")

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
