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

func TestRepository_CountTemplates(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	n, err := repo.CountTemplates(ctx)
	if err != nil {
		t.Fatalf("CountTemplates: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 templates, got %d", n)
	}

	newTestTemplate(t, repo)

	n, err = repo.CountTemplates(ctx)
	if err != nil {
		t.Fatalf("CountTemplates: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 template, got %d", n)
	}
}

func TestRepository_UpdateTemplate(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	updated, err := repo.UpdateTemplate(ctx, tpl.ID, ShiftTemplate{
		Weekday:   int(time.Sunday),
		StartTime: "09:00",
		EndTime:   "13:00",
		Label:     "Turno modificato",
		Active:    false,
	})
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if updated.Weekday != int(time.Sunday) || updated.StartTime != "09:00" || updated.EndTime != "13:00" || updated.Label != "Turno modificato" {
		t.Fatalf("unexpected fields after update: %+v", updated)
	}
	// The whole point of the map-based update (see UpdateTemplate): Active
	// must actually become false, not silently stay true.
	if updated.Active {
		t.Fatal("expected Active to be false after the update, got true")
	}

	got, err := repo.GetTemplate(ctx, tpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if got.Active {
		t.Fatal("expected the persisted row to have Active=false")
	}
}

func TestRepository_UpdateTemplateNotFound(t *testing.T) {
	repo := newTestRepository(t)

	_, err := repo.UpdateTemplate(context.Background(), "00000000-0000-0000-0000-000000000000", ShiftTemplate{
		Weekday: int(time.Monday), StartTime: "08:00", EndTime: "14:00", Label: "x", Active: true,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_CreateAndGetBooking(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	created, err := repo.CreateBooking(ctx, Booking{
		TemplateID:  tpl.ID,
		VolunteerID: testVolunteerID,
		Role:        BookingRoleDriver,
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
	if created.Role != BookingRoleDriver {
		t.Fatalf("CreateBooking: expected role to be preserved, got %q", created.Role)
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

	later, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), StartTime: "14:00", EndTime: "18:00"})
	if err != nil {
		t.Fatalf("CreateBooking later: %v", err)
	}
	earlier, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), StartTime: "08:00", EndTime: "12:00"})
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
		Role:        BookingRoleDriver,
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
		Role:        BookingRoleDriver,
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
// template+date+role are both allowed (a manager picks one later), but two
// confirmed ones for the same slot+role are not.
func TestRepository_OnlyOneConfirmedBookingPerSlotAndRole(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	date := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

	firstPending, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking first pending: %v", err)
	}
	if _, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime}); err != nil {
		t.Fatalf("expected a second pending request for the same slot+role to be allowed, got: %v", err)
	}

	if err := testDB.Model(&Booking{}).Where("id = ?", firstPending.ID).Update("status", BookingStatusConfirmed).Error; err != nil {
		t.Fatalf("confirm first booking: %v", err)
	}

	second, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking second (still pending): %v", err)
	}
	if err := testDB.Model(&Booking{}).Where("id = ?", second.ID).Update("status", BookingStatusConfirmed).Error; err == nil {
		t.Fatal("expected confirming a second booking for the same template+date+role to fail")
	}
}

// Confirming one role must not affect a different role on the same
// template+date — the whole point of scoping the unique index by role
// (see docs/adr/0025-modello-dati-turni.md "Aggiornamento").
func TestRepository_ConfirmingOneRoleDoesNotBlockAnother(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	date := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

	driver, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking driver: %v", err)
	}
	if err := testDB.Model(&Booking{}).Where("id = ?", driver.ID).Update("status", BookingStatusConfirmed).Error; err != nil {
		t.Fatalf("confirm driver booking: %v", err)
	}

	other := newTestVolunteer(t)
	leader, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: other, Role: BookingRoleLeader, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking leader: %v", err)
	}
	if err := testDB.Model(&Booking{}).Where("id = ?", leader.ID).Update("status", BookingStatusConfirmed).Error; err != nil {
		t.Fatalf("expected confirming a different role on the same slot to succeed, got: %v", err)
	}
}

func TestRepository_HasConfirmedBooking(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	date := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

	has, err := repo.HasConfirmedBooking(ctx, tpl.ID, date, BookingRoleDriver)
	if err != nil {
		t.Fatalf("HasConfirmedBooking: %v", err)
	}
	if has {
		t.Fatal("expected no confirmed booking yet")
	}

	pending, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	has, err = repo.HasConfirmedBooking(ctx, tpl.ID, date, BookingRoleDriver)
	if err != nil {
		t.Fatalf("HasConfirmedBooking: %v", err)
	}
	if has {
		t.Fatal("a pending booking must not count as confirmed")
	}

	if err := testDB.Model(&Booking{}).Where("id = ?", pending.ID).Update("status", BookingStatusConfirmed).Error; err != nil {
		t.Fatalf("confirm booking: %v", err)
	}
	has, err = repo.HasConfirmedBooking(ctx, tpl.ID, date, BookingRoleDriver)
	if err != nil {
		t.Fatalf("HasConfirmedBooking: %v", err)
	}
	if !has {
		t.Fatal("expected a confirmed booking now")
	}

	// A different role on the same template+date must not be affected.
	has, err = repo.HasConfirmedBooking(ctx, tpl.ID, date, BookingRoleLeader)
	if err != nil {
		t.Fatalf("HasConfirmedBooking (leader): %v", err)
	}
	if has {
		t.Fatal("expected the leader role to be unaffected by the driver's confirmed booking")
	}
}

func TestRepository_HasBookingForVolunteer(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	date := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

	has, err := repo.HasBookingForVolunteer(ctx, tpl.ID, date, testVolunteerID)
	if err != nil {
		t.Fatalf("HasBookingForVolunteer: %v", err)
	}
	if has {
		t.Fatal("expected no booking yet")
	}

	booking, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	has, err = repo.HasBookingForVolunteer(ctx, tpl.ID, date, testVolunteerID)
	if err != nil {
		t.Fatalf("HasBookingForVolunteer: %v", err)
	}
	if !has {
		t.Fatal("expected a pending booking to count")
	}

	// rejected/cancelled don't block a new request.
	if err := testDB.Model(&Booking{}).Where("id = ?", booking.ID).Update("status", BookingStatusRejected).Error; err != nil {
		t.Fatalf("reject booking: %v", err)
	}
	has, err = repo.HasBookingForVolunteer(ctx, tpl.ID, date, testVolunteerID)
	if err != nil {
		t.Fatalf("HasBookingForVolunteer: %v", err)
	}
	if has {
		t.Fatal("a rejected booking must not block a new request")
	}
}

// HasBookingForVolunteer is deliberately NOT scoped by role: a volunteer
// who already holds one role on a template+date must show up as "has a
// booking" regardless of which OTHER role is being checked — one person
// holds at most one role per occurrence (see docs/adr/0025-modello-dati-turni.md
// "Aggiornamento").
func TestRepository_HasBookingForVolunteer_IgnoresRole(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	date := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

	if _, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime}); err != nil {
		t.Fatalf("CreateBooking (driver): %v", err)
	}

	has, err := repo.HasBookingForVolunteer(ctx, tpl.ID, date, testVolunteerID)
	if err != nil {
		t.Fatalf("HasBookingForVolunteer: %v", err)
	}
	if !has {
		t.Fatal("expected the volunteer's driver booking to block them regardless of which role is being requested next")
	}
}

func TestRepository_CountPendingBookings(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	date := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

	n, err := repo.CountPendingBookings(ctx)
	if err != nil {
		t.Fatalf("CountPendingBookings: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}

	confirmed, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	if err := testDB.Model(&Booking{}).Where("id = ?", confirmed.ID).Update("status", BookingStatusConfirmed).Error; err != nil {
		t.Fatalf("confirm booking: %v", err)
	}
	if _, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date.AddDate(0, 0, 7), StartTime: tpl.StartTime, EndTime: tpl.EndTime}); err != nil {
		t.Fatalf("CreateBooking pending: %v", err)
	}

	n, err = repo.CountPendingBookings(ctx)
	if err != nil {
		t.Fatalf("CountPendingBookings: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 (confirmed one excluded), got %d", n)
	}
}

func TestRepository_ListPendingBookings_OnlyPendingStatus(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	date := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

	pending, err := repo.ListPendingBookings(ctx)
	if err != nil {
		t.Fatalf("ListPendingBookings: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 before any booking exists, got %d", len(pending))
	}

	confirmed, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: date, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking (confirmed): %v", err)
	}
	if err := testDB.Model(&Booking{}).Where("id = ?", confirmed.ID).Update("status", BookingStatusConfirmed).Error; err != nil {
		t.Fatalf("confirm booking: %v", err)
	}
	stillPending, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleLeader, Date: date.AddDate(0, 0, 7), StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking (pending): %v", err)
	}
	rejected, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleRescuer, Date: date.AddDate(0, 0, 14), StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking (rejected): %v", err)
	}
	if err := testDB.Model(&Booking{}).Where("id = ?", rejected.ID).Update("status", BookingStatusRejected).Error; err != nil {
		t.Fatalf("reject booking: %v", err)
	}

	pending, err = repo.ListPendingBookings(ctx)
	if err != nil {
		t.Fatalf("ListPendingBookings: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 (only the pending one), got %d", len(pending))
	}
	if pending[0].ID != stillPending.ID {
		t.Fatalf("expected the pending booking %s, got %s", stillPending.ID, pending[0].ID)
	}
}

func TestRepository_DecideBooking_Confirm(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	booking, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}

	decided, err := repo.DecideBooking(ctx, booking.ID, BookingStatusConfirmed, testVolunteerID)
	if err != nil {
		t.Fatalf("DecideBooking: %v", err)
	}
	if decided.Status != BookingStatusConfirmed {
		t.Fatalf("expected status confirmed, got %s", decided.Status)
	}
	if decided.DecidedBy == nil || *decided.DecidedBy != testVolunteerID {
		t.Fatalf("expected decided_by to be set, got %+v", decided.DecidedBy)
	}
	if decided.DecidedAt == nil {
		t.Fatal("expected decided_at to be set")
	}
}

func TestRepository_DecideBooking_Reject(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	booking, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}

	decided, err := repo.DecideBooking(ctx, booking.ID, BookingStatusRejected, testVolunteerID)
	if err != nil {
		t.Fatalf("DecideBooking: %v", err)
	}
	if decided.Status != BookingStatusRejected {
		t.Fatalf("expected status rejected, got %s", decided.Status)
	}
}

func TestRepository_DecideBooking_NotFound(t *testing.T) {
	repo := newTestRepository(t)

	_, err := repo.DecideBooking(context.Background(), "00000000-0000-0000-0000-000000000000", BookingStatusConfirmed, testVolunteerID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_DecideBooking_AlreadyDecided(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	booking, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	if _, err := repo.DecideBooking(ctx, booking.ID, BookingStatusConfirmed, testVolunteerID); err != nil {
		t.Fatalf("first DecideBooking: %v", err)
	}

	_, err = repo.DecideBooking(ctx, booking.ID, BookingStatusRejected, testVolunteerID)
	if !errors.Is(err, ErrBookingNotPending) {
		t.Fatalf("expected ErrBookingNotPending, got %v", err)
	}
}

func TestRepository_DecideBooking_Cancelled(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	booking, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}
	if err := testDB.Model(&Booking{}).Where("id = ?", booking.ID).Update("status", BookingStatusCancelled).Error; err != nil {
		t.Fatalf("cancel booking: %v", err)
	}

	_, err = repo.DecideBooking(ctx, booking.ID, BookingStatusConfirmed, testVolunteerID)
	if !errors.Is(err, ErrBookingNotPending) {
		t.Fatalf("expected ErrBookingNotPending, got %v", err)
	}
}

func TestRepository_CreateConfirmedBooking(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	created, err := repo.CreateConfirmedBooking(ctx, Booking{
		TemplateID:  tpl.ID,
		VolunteerID: testVolunteerID,
		Role:        BookingRoleDriver,
		Date:        thursday,
		StartTime:   tpl.StartTime,
		EndTime:     tpl.EndTime,
	}, testVolunteerID)
	if err != nil {
		t.Fatalf("CreateConfirmedBooking: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected a non-empty id")
	}
	if created.Status != BookingStatusConfirmed {
		t.Fatalf("expected status confirmed, got %s", created.Status)
	}
	if created.DecidedBy == nil || *created.DecidedBy != testVolunteerID {
		t.Fatalf("expected decided_by to be set, got %+v", created.DecidedBy)
	}
	if created.DecidedAt == nil {
		t.Fatal("expected decided_at to be set")
	}
}

func TestRepository_CreateConfirmedBooking_RejectsUnknownVolunteer(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	_, err := repo.CreateConfirmedBooking(ctx, Booking{
		TemplateID:  tpl.ID,
		VolunteerID: "00000000-0000-0000-0000-000000000000",
		Role:        BookingRoleDriver,
		Date:        thursday,
		StartTime:   tpl.StartTime,
		EndTime:     tpl.EndTime,
	}, testVolunteerID)
	if !errors.Is(err, ErrUnknownVolunteer) {
		t.Fatalf("expected ErrUnknownVolunteer, got %v", err)
	}
}

func TestRepository_CreateBookingsAtomic_CreatesAll(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	created, err := repo.CreateBookingsAtomic(ctx, []Booking{
		{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime},
		{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: thursday.AddDate(0, 0, 7), StartTime: tpl.StartTime, EndTime: tpl.EndTime},
		{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: thursday.AddDate(0, 0, 14), StartTime: tpl.StartTime, EndTime: tpl.EndTime},
	})
	if err != nil {
		t.Fatalf("CreateBookingsAtomic: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("expected 3 created bookings, got %d", len(created))
	}
	for i, b := range created {
		if b.ID == "" {
			t.Fatalf("booking %d: expected a non-empty id", i)
		}
		if b.Status != BookingStatusPending {
			t.Fatalf("booking %d: expected status pending, got %s", i, b.Status)
		}
	}

	var count int64
	if err := testDB.Model(&Booking{}).Where("template_id = ?", tpl.ID).Count(&count).Error; err != nil {
		t.Fatalf("count bookings: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 rows persisted, got %d", count)
	}
}

func TestRepository_CreateBookingsAtomic_RollsBackOnError(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	_, err := repo.CreateBookingsAtomic(ctx, []Booking{
		{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime},
		{TemplateID: tpl.ID, VolunteerID: "00000000-0000-0000-0000-000000000000", Role: BookingRoleDriver, Date: thursday.AddDate(0, 0, 7), StartTime: tpl.StartTime, EndTime: tpl.EndTime},
		{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Role: BookingRoleDriver, Date: thursday.AddDate(0, 0, 14), StartTime: tpl.StartTime, EndTime: tpl.EndTime},
	})
	if err == nil {
		t.Fatal("expected an error from the unknown volunteer_id in the middle of the list")
	}

	var count int64
	if err := testDB.Model(&Booking{}).Where("template_id = ?", tpl.ID).Count(&count).Error; err != nil {
		t.Fatalf("count bookings: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected the whole batch to roll back, got %d rows persisted", count)
	}
}
