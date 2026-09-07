package shift

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/francescocerri/sanitas/services/shifts/internal/testdb"
)

// One container for the whole package: much faster than one per test, at
// the cost of each test having to clean up after itself (see truncate below).
var testDB *gorm.DB

// testVolunteerID is a real registry.users row seeded by testdb.StartPostgres
// — bookings.volunteer_id is an FK, so tests need an existing user to
// reference instead of an arbitrary placeholder string.
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
		if err := testDB.Exec("TRUNCATE bookings, shift_templates").Error; err != nil {
			t.Fatalf("truncate: %v", err)
		}
	})
	return NewRepository(testDB)
}

// newTestTemplate creates a template directly through the repository —
// every booking test needs one to reference, same role testVolunteerID
// plays for the FK into registry.users.
func newTestTemplate(t *testing.T, repo *Repository) ShiftTemplate {
	t.Helper()
	tpl, err := repo.CreateTemplate(context.Background(), ShiftTemplate{
		Weekday:   int(time.Thursday),
		StartTime: "20:00",
		EndTime:   "08:00",
		Label:     "Turno notturno",
	})
	if err != nil {
		t.Fatalf("CreateTemplate (fixture): %v", err)
	}
	return tpl
}

func TestRepository_CreateAndGetTemplate(t *testing.T) {
	repo := newTestRepository(t)

	created, err := repo.CreateTemplate(context.Background(), ShiftTemplate{
		Weekday:   int(time.Saturday),
		StartTime: "08:00",
		EndTime:   "14:00",
		Label:     "Turno 1",
		Active:    false, // must still come back true: see CreateTemplate
	})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if created.ID == "" {
		t.Fatal("CreateTemplate: expected a non-empty id")
	}
	if !created.Active {
		t.Fatal("CreateTemplate: expected a new template to always be active")
	}

	got, err := repo.GetTemplate(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if got != created {
		t.Fatalf("GetTemplate: expected %+v, got %+v", created, got)
	}
}

func TestRepository_GetTemplateNotFound(t *testing.T) {
	repo := newTestRepository(t)

	_, err := repo.GetTemplate(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_ListTemplatesEmpty(t *testing.T) {
	repo := newTestRepository(t)

	got, err := repo.ListTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if got == nil {
		t.Fatal("ListTemplates: expected an empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("ListTemplates: expected no rows, got %d", len(got))
	}
}

func TestRepository_ListTemplatesOrdersByWeekdayAndStartTime(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	later, err := repo.CreateTemplate(ctx, ShiftTemplate{Weekday: int(time.Saturday), StartTime: "14:00", EndTime: "20:00", Label: "Turno 2"})
	if err != nil {
		t.Fatalf("CreateTemplate later: %v", err)
	}
	earlier, err := repo.CreateTemplate(ctx, ShiftTemplate{Weekday: int(time.Saturday), StartTime: "08:00", EndTime: "14:00", Label: "Turno 1"})
	if err != nil {
		t.Fatalf("CreateTemplate earlier: %v", err)
	}

	got, err := repo.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(got))
	}
	if got[0].ID != earlier.ID || got[1].ID != later.ID {
		t.Fatalf("expected %s before %s, got order %s, %s", earlier.ID, later.ID, got[0].ID, got[1].ID)
	}
}

func TestRepository_CreateAndGetBooking(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	created, err := repo.CreateBooking(ctx, Booking{
		TemplateID:  tpl.ID,
		VolunteerID: testVolunteerID,
		Date:        time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		StartTime:   tpl.StartTime,
		EndTime:     tpl.EndTime,
	})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	if created.ID == "" {
		t.Fatal("CreateBooking: expected a non-empty id")
	}
	if created.Status != BookingStatusPending {
		t.Fatalf("CreateBooking: expected default status %q, got %q", BookingStatusPending, created.Status)
	}
	if created.DecidedBy != nil || created.DecidedAt != nil {
		t.Fatalf("CreateBooking: expected no decider yet, got %+v", created)
	}

	got, err := repo.GetBooking(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetBooking: %v", err)
	}
	if got.ID != created.ID || got.Status != created.Status || !got.Date.Equal(created.Date) {
		t.Fatalf("GetBooking: expected %+v, got %+v", created, got)
	}
}

func TestRepository_GetBookingNotFound(t *testing.T) {
	repo := newTestRepository(t)

	_, err := repo.GetBooking(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_ListBookingsOrdersByDateAndStartTime(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	later, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Date: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), StartTime: "14:00", EndTime: "18:00"})
	if err != nil {
		t.Fatalf("CreateBooking later: %v", err)
	}
	earlier, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Date: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), StartTime: "08:00", EndTime: "12:00"})
	if err != nil {
		t.Fatalf("CreateBooking earlier: %v", err)
	}

	got, err := repo.ListBookings(ctx)
	if err != nil {
		t.Fatalf("ListBookings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 bookings, got %d", len(got))
	}
	if got[0].ID != earlier.ID || got[1].ID != later.ID {
		t.Fatalf("expected %s before %s, got order %s, %s", earlier.ID, later.ID, got[0].ID, got[1].ID)
	}
}

// A booking referencing a volunteer_id that doesn't exist in registry.users
// must be rejected — the FK is a real data-integrity constraint (see
// docs/adr/0014), not just a naming convention.
func TestRepository_CreateBookingRejectsUnknownVolunteerID(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	_, err := repo.CreateBooking(ctx, Booking{
		TemplateID:  tpl.ID,
		VolunteerID: "00000000-0000-0000-0000-000000000000",
		Date:        time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		StartTime:   "08:00",
		EndTime:     "14:00",
	})
	if err == nil {
		t.Fatal("expected an error for an unknown volunteer_id, got nil")
	}
}

// Same for an unknown template_id — new FK introduced by ADR-0025.
func TestRepository_CreateBookingRejectsUnknownTemplateID(t *testing.T) {
	repo := newTestRepository(t)

	_, err := repo.CreateBooking(context.Background(), Booking{
		TemplateID:  "00000000-0000-0000-0000-000000000000",
		VolunteerID: testVolunteerID,
		Date:        time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		StartTime:   "08:00",
		EndTime:     "14:00",
	})
	if err == nil {
		t.Fatal("expected an error for an unknown template_id, got nil")
	}
}

// The partial unique index (idx_bookings_confirmed_slot, see Migrate) only
// constrains confirmed bookings: two pending requests for the same
// template+date are both allowed (a manager picks one later), but two
// confirmed ones for the same slot are not.
func TestRepository_OnlyOneConfirmedBookingPerSlot(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	date := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

	firstPending, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking first pending: %v", err)
	}
	if _, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime}); err != nil {
		t.Fatalf("expected a second pending request for the same slot to be allowed, got: %v", err)
	}

	if err := testDB.Model(&Booking{}).Where("id = ?", firstPending.ID).Update("status", BookingStatusConfirmed).Error; err != nil {
		t.Fatalf("confirm first booking: %v", err)
	}

	second, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking second (still pending): %v", err)
	}
	if err := testDB.Model(&Booking{}).Where("id = ?", second.ID).Update("status", BookingStatusConfirmed).Error; err == nil {
		t.Fatal("expected confirming a second booking for the same template+date to fail")
	}
}
