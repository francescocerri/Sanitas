package turno

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/francescocerri/sanitas/services/turni/internal/testdb"
)

// One container for the whole package: much faster than one per test, at
// the cost of each test having to clean up after itself (see truncate below).
var testPool *pgxpool.Pool

// testVolontarioID is a real anagrafica.users row seeded by testdb.StartPostgres
// — turni.turni.volontario_id is now an FK, so tests need an existing user
// to reference instead of an arbitrary placeholder string.
var testVolontarioID string

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, volontarioID, cleanup, err := testdb.StartPostgres(ctx)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	testPool = pool
	testVolontarioID = volontarioID

	os.Exit(m.Run())
}

func newTestRepository(t *testing.T) *Repository {
	t.Helper()
	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), "TRUNCATE turni"); err != nil {
			t.Fatalf("truncate turni: %v", err)
		}
	})
	return NewRepository(testPool)
}

func TestRepository_CreateAndGet(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, Turno{
		VolontarioID: testVolontarioID,
		Data:         "2026-09-10",
		OraInizio:    "08:00",
		OraFine:      "14:00",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create: expected a non-empty id")
	}
	if created.Stato != "pianificato" {
		t.Fatalf("Create: expected default stato %q, got %q", "pianificato", created.Stato)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != created {
		t.Fatalf("Get: expected %+v, got %+v", created, got)
	}
}

func TestRepository_GetNotFound(t *testing.T) {
	repo := newTestRepository(t)

	_, err := repo.Get(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_ListEmpty(t *testing.T) {
	repo := newTestRepository(t)

	got, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Fatal("List: expected an empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("List: expected no rows, got %d", len(got))
	}
}

func TestRepository_ListOrdersByDataAndOraInizio(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	later, err := repo.Create(ctx, Turno{VolontarioID: testVolontarioID, Data: "2026-09-10", OraInizio: "14:00", OraFine: "18:00"})
	if err != nil {
		t.Fatalf("Create later: %v", err)
	}
	earlier, err := repo.Create(ctx, Turno{VolontarioID: testVolontarioID, Data: "2026-09-10", OraInizio: "08:00", OraFine: "12:00"})
	if err != nil {
		t.Fatalf("Create earlier: %v", err)
	}

	got, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 turni, got %d", len(got))
	}
	if got[0].ID != earlier.ID || got[1].ID != later.ID {
		t.Fatalf("expected %s before %s, got order %s, %s", earlier.ID, later.ID, got[0].ID, got[1].ID)
	}
}

// A turno referencing a volontario_id that doesn't exist in anagrafica.users
// must be rejected — the FK is a real data-integrity constraint (see
// docs/adr/0014), not just a naming convention.
func TestRepository_CreateRejectsUnknownVolontarioID(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, Turno{
		VolontarioID: "00000000-0000-0000-0000-000000000000",
		Data:         "2026-09-10",
		OraInizio:    "08:00",
		OraFine:      "14:00",
	})
	if err == nil {
		t.Fatal("expected an error for an unknown volontario_id, got nil")
	}
}
