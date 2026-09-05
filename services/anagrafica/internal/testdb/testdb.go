// Package testdb starts a disposable Postgres container for integration
// tests (testcontainers-go), shared by every internal package's test suite
// so each one doesn't reimplement container/schema setup. Test-only: no
// production package imports this (see docs/adr/0011 in services/turni,
// same pattern replicated here).
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

// StartPostgres starts a disposable Postgres container and applies migrate
// (the same function production startup calls — e.g. user.Migrate, see
// docs/adr/0019) against a fresh GORM connection, returning it plus a
// cleanup function. Meant to be called once from a package's TestMain.
//
// migrate is a parameter, not an import of internal/user, on purpose:
// internal/user's own tests use this package too, and internal/user
// importing testdb while testdb imported internal/user back would be an
// import cycle.
func StartPostgres(ctx context.Context, migrate func(*gorm.DB) error) (*gorm.DB, func(), error) {
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("anagrafica"),
		postgres.WithUsername("anagrafica"),
		postgres.WithPassword("test"),
		// Postgres always starts in two phases — a temporary server for
		// initdb-time setup, then the real one — so this log line appears
		// twice regardless of init scripts; waiting for only the first
		// occurrence races with the second startup.
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("testdb: start postgres container: %w", err)
	}

	// search_path=anagrafica: Migrate creates the tables inside an
	// "anagrafica" schema, not the database's default public one (the two
	// services share one database — see docs/adr/0014) — this keeps
	// existing unqualified queries (FROM users, no schema prefix) working.
	connString, err := container.ConnectionString(ctx, "sslmode=disable", "search_path=anagrafica")
	if err != nil {
		return nil, nil, fmt.Errorf("testdb: connection string: %w", err)
	}

	db, err := gorm.Open(gormpostgres.Open(connString), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("testdb: connect: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, nil, fmt.Errorf("testdb: migrate: %w", err)
	}

	cleanup := func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		_ = container.Terminate(context.Background())
	}
	return db, cleanup, nil
}
