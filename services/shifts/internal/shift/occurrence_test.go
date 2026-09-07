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

	got, err := repo.ListOccurrences(ctx, thursday, thursday)
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

	got, err := repo.ListOccurrences(ctx, thursday, thursday)
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

	got, err := repo.ListOccurrences(ctx, thursday, thursday)
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

	got, err := repo.ListOccurrences(ctx, thursday, thursday)
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

	got, err := repo.ListOccurrences(ctx, thursday, thursday.AddDate(0, 0, 3))
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
	got, err := repo.ListOccurrences(ctx, friday, friday)
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
