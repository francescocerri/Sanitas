// Package testdb starts a disposable Postgres container for integration
// tests (testcontainers-go), shared by every internal package's test suite
// so each one doesn't reimplement container/schema setup. Test-only: no
// production package imports this (see docs/adr/0011 in services/turni,
// same pattern replicated here).
package testdb

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// migrationPath resolves relative to this source file, not to the caller's
// working directory — `go test` runs with the package under test as cwd,
// which differs between internal/user and internal/httpapi.
func migrationPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations", "0001_init.sql")
}

// StartPostgres starts a disposable Postgres container with the service's
// real schema already applied, and returns a ready-to-use pool plus a
// cleanup function. Meant to be called once from a package's TestMain.
func StartPostgres(ctx context.Context) (*pgxpool.Pool, func(), error) {
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("anagrafica"),
		postgres.WithUsername("anagrafica"),
		postgres.WithPassword("test"),
		postgres.WithInitScripts(migrationPath()),
		// Init scripts make Postgres restart once after running them, so
		// "ready to accept connections" appears twice in the logs — waiting
		// for only the first occurrence would race with that restart.
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("testdb: start postgres container: %w", err)
	}

	// search_path=anagrafica: the migration now creates its tables inside
	// an "anagrafica" schema, not the database's default public one (the
	// two services share one database — see docs/adr/0014) — this keeps
	// existing unqualified queries (FROM users, no schema prefix) working.
	connString, err := container.ConnectionString(ctx, "sslmode=disable", "search_path=anagrafica")
	if err != nil {
		return nil, nil, fmt.Errorf("testdb: connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, nil, fmt.Errorf("testdb: connect: %w", err)
	}

	cleanup := func() {
		pool.Close()
		_ = container.Terminate(context.Background())
	}
	return pool, cleanup, nil
}
