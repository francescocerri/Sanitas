package turno

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
		return fmt.Errorf("turno: ping: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

func (r *Repository) List(ctx context.Context) ([]Turno, error) {
	// []Turno{}, not a nil slice: encodes to `[]` in JSON, not `null`.
	result := []Turno{}
	if err := r.db.WithContext(ctx).Order("data, ora_inizio").Find(&result).Error; err != nil {
		return nil, fmt.Errorf("turno: list: %w", err)
	}
	return result, nil
}

// Create ignores any id/stato the caller passed in t (per the API contract,
// see @Param in httpapi) — cleared before Create so GORM lets their DB
// defaults apply instead of inserting a client-supplied value.
func (r *Repository) Create(ctx context.Context, t Turno) (Turno, error) {
	t.ID = ""
	t.Stato = ""
	if err := r.db.WithContext(ctx).Create(&t).Error; err != nil {
		return Turno{}, fmt.Errorf("turno: create: %w", err)
	}
	return t, nil
}

func (r *Repository) Get(ctx context.Context, id string) (Turno, error) {
	var t Turno
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Turno{}, ErrNotFound
	}
	if err != nil {
		return Turno{}, fmt.Errorf("turno: get: %w", err)
	}
	return t, nil
}
