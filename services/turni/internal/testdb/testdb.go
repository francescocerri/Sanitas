// Package testdb starts a disposable Postgres container for integration
// tests (testcontainers-go), shared by every internal package's test suite
// so each one doesn't reimplement container/schema setup. Test-only: no
// production package imports this (see docs/adr/0011).
package testdb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/francescocerri/sanitas/services/turni/internal/schema"
)

// anagraficaFixtureSQL stands in for anagrafica's real schema — a separate
// Go module, migrated by its own GORM AutoMigrate at its own startup, not
// something this test setup can call into (see docs/adr/0019). Just enough
// for turni's FK (turni.turni.volontario_id -> anagrafica.users.id) to
// attach to — not a copy of the real table's other columns.
const anagraficaFixtureSQL = `
CREATE SCHEMA IF NOT EXISTS anagrafica;
CREATE TABLE IF NOT EXISTS anagrafica.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid()
);
`

// StartPostgres starts a disposable Postgres container, applies the
// anagrafica fixture and then turni's real schema (schema.SQL — the same
// embedded text production runs, see cmd/server/main.go) in that order, and
// seeds one anagrafica.users row so tests have a real id to satisfy the FK.
// Returns a ready-to-use pool connected with search_path=turni, the seeded
// row's id, and a cleanup function. Meant to be called once from a
// package's TestMain.
func StartPostgres(ctx context.Context) (*pgxpool.Pool, string, func(), error) {
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("sanitas"),
		postgres.WithUsername("sanitas"),
		postgres.WithPassword("test"),
		// Postgres always starts in two phases — a temporary server for
		// initdb-time setup, then the real one — so this log line appears
		// twice regardless of init scripts (there are none here anymore).
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("testdb: start postgres container: %w", err)
	}

	connString, err := container.ConnectionString(ctx, "sslmode=disable", "search_path=turni")
	if err != nil {
		return nil, "", nil, fmt.Errorf("testdb: connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, "", nil, fmt.Errorf("testdb: connect: %w", err)
	}

	if _, err := pool.Exec(ctx, anagraficaFixtureSQL); err != nil {
		pool.Close()
		return nil, "", nil, fmt.Errorf("testdb: apply anagrafica fixture: %w", err)
	}
	if _, err := pool.Exec(ctx, schema.SQL); err != nil {
		pool.Close()
		return nil, "", nil, fmt.Errorf("testdb: apply turni schema: %w", err)
	}

	var volontarioID string
	err = pool.QueryRow(ctx, `INSERT INTO anagrafica.users DEFAULT VALUES RETURNING id`).Scan(&volontarioID)
	if err != nil {
		pool.Close()
		return nil, "", nil, fmt.Errorf("testdb: seed test volontario: %w", err)
	}

	cleanup := func() {
		pool.Close()
		_ = container.Terminate(context.Background())
	}
	return pool, volontarioID, cleanup, nil
}
