package shift

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNotFound is the domain error for "no matching row" — gorm.ErrRecordNotFound
// is translated here, so callers above this layer (HTTP) don't need to know
// about GORM to tell a 404 apart from a real error.
var ErrNotFound = errors.New("shift not found")

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

func (r *Repository) List(ctx context.Context) ([]Shift, error) {
	// []Shift{}, not a nil slice: encodes to `[]` in JSON, not `null`.
	result := []Shift{}
	if err := r.db.WithContext(ctx).Order("date, start_time").Find(&result).Error; err != nil {
		return nil, fmt.Errorf("shift: list: %w", err)
	}
	return result, nil
}

// Create ignores any id/status the caller passed in s (per the API contract,
// see @Param in httpapi) — cleared before Create so GORM lets their DB
// defaults apply instead of inserting a client-supplied value.
func (r *Repository) Create(ctx context.Context, s Shift) (Shift, error) {
	s.ID = ""
	s.Status = ""
	if err := r.db.WithContext(ctx).Create(&s).Error; err != nil {
		return Shift{}, fmt.Errorf("shift: create: %w", err)
	}
	return s, nil
}

func (r *Repository) Get(ctx context.Context, id string) (Shift, error) {
	var s Shift
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Shift{}, ErrNotFound
	}
	if err != nil {
		return Shift{}, fmt.Errorf("shift: get: %w", err)
	}
	return s, nil
}
