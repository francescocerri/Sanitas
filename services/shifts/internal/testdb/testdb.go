// Package testdb starts a disposable Postgres container for integration
// tests (testcontainers-go), shared by every internal package's test suite
// so each one doesn't reimplement container/schema setup. Test-only: no
// production package imports this (see docs/adr/0011).
package testdb

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// registryFixtureSQL stands in for registry's real schema — a separate Go
// module, migrated by its own GORM AutoMigrate at its own startup, not
// something this test setup can call into (see docs/adr/0019). Just enough
// for shifts' FK (shifts.shifts.volunteer_id -> registry.users.id) to
// attach to — not a copy of the real table's other columns.
const registryFixtureSQL = `
CREATE SCHEMA IF NOT EXISTS registry;
CREATE TABLE IF NOT EXISTS registry.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid()
);
`

// StartPostgres starts a disposable Postgres container, applies the
// registry fixture and then migrate (the same function production startup
// calls — shift.Migrate, see docs/adr/0019/0020) in that order, and seeds
// one registry.users row so tests have a real id to satisfy the FK.
// Returns a ready-to-use GORM connection (search_path=shifts), the seeded
// row's id, and a cleanup function. Meant to be called once from a
// package's TestMain.
//
// migrate is a parameter, not an import of internal/shift, on purpose:
// internal/shift's own tests use this package too, and internal/shift
// importing testdb while testdb imported internal/shift back would be an
// import cycle.
func StartPostgres(ctx context.Context, migrate func(*gorm.DB) error) (*gorm.DB, string, func(), error) {
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("sanitas"),
		postgres.WithUsername("sanitas"),
		postgres.WithPassword("test"),
		// Postgres always starts in two phases — a temporary server for
		// initdb-time setup, then the real one — so this log line appears
		// twice regardless of init scripts (there are none here).
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("testdb: start postgres container: %w", err)
	}

	connString, err := container.ConnectionString(ctx, "sslmode=disable", "search_path=shifts")
	if err != nil {
		return nil, "", nil, fmt.Errorf("testdb: connection string: %w", err)
	}

	db, err := gorm.Open(gormpostgres.Open(connString), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, "", nil, fmt.Errorf("testdb: connect: %w", err)
	}

	if err := db.Exec(registryFixtureSQL).Error; err != nil {
		return nil, "", nil, fmt.Errorf("testdb: apply registry fixture: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, "", nil, fmt.Errorf("testdb: migrate: %w", err)
	}

	var volunteerID string
	err = db.Raw(`INSERT INTO registry.users DEFAULT VALUES RETURNING id`).Scan(&volunteerID).Error
	if err != nil {
		return nil, "", nil, fmt.Errorf("testdb: seed test volunteer: %w", err)
	}

	cleanup := func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		_ = container.Terminate(context.Background())
	}
	return db, volunteerID, cleanup, nil
}
