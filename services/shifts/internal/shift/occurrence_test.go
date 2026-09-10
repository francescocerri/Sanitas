package shift

import (
	"context"
	"testing"
	"time"
)

// thursday is the same fixture date used by newTestTemplate's booking
// tests: a Thursday, pairing with the Thursday template created there.
var thursday = time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)

func TestListOccurrences_FreeWhenNoBookings(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	got, err := repo.ListOccurrences(ctx, thursday, thursday, testVolunteerID)
	if err != nil {
		t.Fatalf("ListOccurrences: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 occurrence, got %d: %+v", len(got), got)
	}
	if got[0].TemplateID != tpl.ID || got[0].Status != OccurrenceStatusFree {
		t.Fatalf("unexpected occurrence: %+v", got[0])
	}
}

func TestListOccurrences_PendingWhenBookingPending(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	if _, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime}); err != nil {
		t.Fatalf("CreateBooking: %v", err)
	}

	got, err := repo.ListOccurrences(ctx, thursday, thursday, testVolunteerID)
	if err != nil {
		t.Fatalf("ListOccurrences: %v", err)
	}
	if len(got) != 1 || got[0].Status != OccurrenceStatusPending {
		t.Fatalf("expected a single pending occurrence, got %+v", got)
	}
}

func TestListOccurrences_ConfirmedWinsOverPending(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	if _, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime}); err != nil {
		t.Fatalf("CreateBooking pending: %v", err)
	}
	confirmed, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: testVolunteerID, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime})
	if err != nil {
		t.Fatalf("CreateBooking to confirm: %v", err)
	}
	if err := testDB.Model(&Booking{}).Where("id = ?", confirmed.ID).Update("status", BookingStatusConfirmed).Error; err != nil {
		t.Fatalf("confirm booking: %v", err)
	}

	got, err := repo.ListOccurrences(ctx, thursday, thursday, testVolunteerID)
	if err != nil {
		t.Fatalf("ListOccurrences: %v", err)
	}
	if len(got) != 1 || got[0].Status != OccurrenceStatusConfirmed {
		t.Fatalf("expected confirmed to win over pending, got %+v", got)
	}
}

func TestListOccurrences_ExcludesInactiveTemplates(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)

	if _, err := repo.UpdateTemplate(ctx, tpl.ID, ShiftTemplate{
		Weekday: tpl.Weekday, StartTime: tpl.StartTime, EndTime: tpl.EndTime, Label: tpl.Label, Active: false,
	}); err != nil {
		t.Fatalf("UpdateTemplate (deactivate): %v", err)
	}

	got, err := repo.ListOccurrences(ctx, thursday, thursday, testVolunteerID)
	if err != nil {
		t.Fatalf("ListOccurrences: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no occurrences for an inactive template, got %+v", got)
	}
}

func TestListOccurrences_OnlyMatchingWeekday(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	// A Saturday template; thursday..thursday+3 spans Thu/Fri/Sat/Sun.
	saturdayTpl, err := repo.CreateTemplate(ctx, ShiftTemplate{Weekday: int(time.Saturday), StartTime: "08:00", EndTime: "14:00", Label: "Turno Mattina"})
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	got, err := repo.ListOccurrences(ctx, thursday, thursday.AddDate(0, 0, 3), testVolunteerID)
	if err != nil {
		t.Fatalf("ListOccurrences: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 occurrence (the Saturday), got %d: %+v", len(got), got)
	}
	if got[0].TemplateID != saturdayTpl.ID || got[0].Weekday != int(time.Saturday) {
		t.Fatalf("unexpected occurrence: %+v", got[0])
	}
}

func TestListOccurrences_EmptyWhenNoTemplateMatchesRange(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	newTestTemplate(t, repo) // Thursday

	// A single Friday: no Thursday template matches.
	friday := thursday.AddDate(0, 0, 1)
	got, err := repo.ListOccurrences(ctx, friday, friday, testVolunteerID)
	if err != nil {
		t.Fatalf("ListOccurrences: %v", err)
	}
	if got == nil {
		t.Fatal("expected an empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected no occurrences, got %+v", got)
	}
}

// newTestVolunteer seeds a second registry.users row — occurrence_test.go's
// own equivalent of httpapi's newRegistryUser, needed here to tell "my
// booking" apart from "someone else's" on the same occurrence.
func newTestVolunteer(t *testing.T) string {
	t.Helper()
	var id string
	if err := testDB.Raw(`INSERT INTO registry.users DEFAULT VALUES RETURNING id`).Scan(&id).Error; err != nil {
		t.Fatalf("seed second registry user: %v", err)
	}
	return id
}

func TestListOccurrences_MyBookingStatusOnlyForCaller(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	tpl := newTestTemplate(t, repo)
	other := newTestVolunteer(t)

	if _, err := repo.CreateBooking(ctx, Booking{TemplateID: tpl.ID, VolunteerID: other, Date: thursday, StartTime: tpl.StartTime, EndTime: tpl.EndTime}); err != nil {
		t.Fatalf("CreateBooking (other volunteer): %v", err)
	}

	gotAsOther, err := repo.ListOccurrences(ctx, thursday, thursday, other)
	if err != nil {
		t.Fatalf("ListOccurrences (as other): %v", err)
	}
	if len(gotAsOther) != 1 || gotAsOther[0].MyBookingStatus == nil || *gotAsOther[0].MyBookingStatus != BookingStatusPending {
		t.Fatalf("expected my_booking_status=pending for the booking's own volunteer, got %+v", gotAsOther)
	}

	gotAsCaller, err := repo.ListOccurrences(ctx, thursday, thursday, testVolunteerID)
	if err != nil {
		t.Fatalf("ListOccurrences (as testVolunteerID): %v", err)
	}
	if len(gotAsCaller) != 1 || gotAsCaller[0].MyBookingStatus != nil {
		t.Fatalf("expected my_booking_status=nil for a caller with no booking on this occurrence, got %+v", gotAsCaller)
	}
	// The aggregate status is unaffected by whose booking it is.
	if gotAsCaller[0].Status != OccurrenceStatusPending {
		t.Fatalf("expected aggregate status pending regardless of caller, got %s", gotAsCaller[0].Status)
	}
}
