// Package testdb starts a disposable Postgres container for integration
// tests (testcontainers-go), shared by every internal package's test suite
// so each one doesn't reimplement container/schema setup. Test-only: no
// production package imports this (see docs/adr/0011).
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
// which differs between internal/turno and internal/httpapi.
func migrationPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations", "0001_init.sql")
}

// anagraficaMigrationPath: turni.turni has a real FK into anagrafica.users
// (see docs/adr/0014), so the test database needs anagrafica's schema
// applied too, not just turni's own. This reaches into the sibling module's
// directory on purpose — the two services share a database in production
// (same ADR), so their tests share this one bit of coupling too.
func anagraficaMigrationPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "anagrafica", "migrations", "0001_init.sql")
}

// StartPostgres starts a disposable Postgres container with both services'
// real schemas already applied (anagrafica first, since turni's FK needs
// it), and returns a pool connected with search_path=turni — so existing
// unqualified queries (FROM turni, no schema prefix) keep working — plus the
// id of a seeded anagrafica.users row (turni.turni.volontario_id needs a
// real user to reference, since it's now a FK, not a free-text placeholder)
// and a cleanup function. Meant to be called once from a package's TestMain.
func StartPostgres(ctx context.Context) (*pgxpool.Pool, string, func(), error) {
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("sanitas"),
		postgres.WithUsername("sanitas"),
		postgres.WithPassword("test"),
		// WithOrderedInitScripts (not WithInitScripts): both services name
		// their migration file "0001_init.sql", so using just the basename
		// would collide — this prefixes each with an index and preserves
		// the anagrafica-before-turni order the FK requires.
		postgres.WithOrderedInitScripts(anagraficaMigrationPath(), migrationPath()),
		// Init scripts make Postgres restart once after running them, so
		// "ready to accept connections" appears twice in the logs — waiting
		// for only the first occurrence would race with that restart.
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

	// Schema-qualified: search_path is set to turni, not anagrafica.
	var volontarioID string
	err = pool.QueryRow(ctx,
		`INSERT INTO anagrafica.users (email, username) VALUES ($1, $2) RETURNING id`,
		"test-volontario@example.com", "test-volontario",
	).Scan(&volontarioID)
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
