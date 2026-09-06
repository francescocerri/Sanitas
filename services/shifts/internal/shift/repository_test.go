package shift

import (
	"context"
	"errors"
	"os"
	"testing"

	"gorm.io/gorm"

	"github.com/francescocerri/sanitas/services/shifts/internal/testdb"
)

// One container for the whole package: much faster than one per test, at
// the cost of each test having to clean up after itself (see truncate below).
var testDB *gorm.DB

// testVolunteerID is a real registry.users row seeded by testdb.StartPostgres
// — shifts.shifts.volunteer_id is now an FK, so tests need an existing user
// to reference instead of an arbitrary placeholder string.
var testVolunteerID string

func TestMain(m *testing.M) {
	ctx := context.Background()
	db, volunteerID, cleanup, err := testdb.StartPostgres(ctx, Migrate)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	testDB = db
	testVolunteerID = volunteerID

	os.Exit(m.Run())
}

func newTestRepository(t *testing.T) *Repository {
	t.Helper()
	t.Cleanup(func() {
		if err := testDB.Exec("TRUNCATE shifts").Error; err != nil {
			t.Fatalf("truncate shifts: %v", err)
		}
	})
	return NewRepository(testDB)
}

func TestRepository_CreateAndGet(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, Shift{
		VolunteerID: testVolunteerID,
		Date:        "2026-09-10",
		StartTime:   "08:00",
		EndTime:     "14:00",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create: expected a non-empty id")
	}
	if created.Status != "planned" {
		t.Fatalf("Create: expected default status %q, got %q", "planned", created.Status)
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

func TestRepository_ListOrdersByDateAndStartTime(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	later, err := repo.Create(ctx, Shift{VolunteerID: testVolunteerID, Date: "2026-09-10", StartTime: "14:00", EndTime: "18:00"})
	if err != nil {
		t.Fatalf("Create later: %v", err)
	}
	earlier, err := repo.Create(ctx, Shift{VolunteerID: testVolunteerID, Date: "2026-09-10", StartTime: "08:00", EndTime: "12:00"})
	if err != nil {
		t.Fatalf("Create earlier: %v", err)
	}

	got, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 shifts, got %d", len(got))
	}
	if got[0].ID != earlier.ID || got[1].ID != later.ID {
		t.Fatalf("expected %s before %s, got order %s, %s", earlier.ID, later.ID, got[0].ID, got[1].ID)
	}
}

// A shift referencing a volunteer_id that doesn't exist in registry.users
// must be rejected — the FK is a real data-integrity constraint (see
// docs/adr/0014), not just a naming convention.
func TestRepository_CreateRejectsUnknownVolunteerID(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, Shift{
		VolunteerID: "00000000-0000-0000-0000-000000000000",
		Date:        "2026-09-10",
		StartTime:   "08:00",
		EndTime:     "14:00",
	})
	if err == nil {
		t.Fatal("expected an error for an unknown volunteer_id, got nil")
	}
}
