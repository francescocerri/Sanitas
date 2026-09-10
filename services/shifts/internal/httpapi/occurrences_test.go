package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/francescocerri/sanitas/services/shifts/internal/shift"
)

func TestListOccurrences_RequiresReadPermission(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/shift-occurrences?from=2026-09-10&to=2026-09-10", nil)
	req.Header.Set("Authorization", "Bearer "+issuer.token(t, nil))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with no permissions, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListOccurrences_ValidatesParams(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	token := issuer.token(t, []string{permShiftsRead})

	for _, tc := range []struct {
		name  string
		query string
	}{
		{"missing from", "to=2026-09-10"},
		{"missing to", "from=2026-09-10"},
		{"malformed from", "from=10-09-2026&to=2026-09-10"},
		{"malformed to", "from=2026-09-10&to=not-a-date"},
		{"to before from", "from=2026-09-10&to=2026-09-01"},
		{"range too wide", "from=2026-01-01&to=2026-12-31"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/shift-occurrences?"+tc.query, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			server.Routes().ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestListOccurrences_ReturnsCoverageForRange(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	configureToken := issuer.token(t, []string{permShiftsConfigure})
	readToken := issuer.token(t, []string{permShiftsRead})

	// Thursday 2026-09-10 (see internal/shift's own fixture convention).
	createBody, _ := json.Marshal(createTemplateRequest{Weekday: 4, StartTime: "20:00", EndTime: "08:00", Label: "Turno Serale"})
	createReq := httptest.NewRequest(http.MethodPost, "/v1/shift-templates", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+configureToken)
	createRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRec, createReq)
	var created shift.ShiftTemplate
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/shift-occurrences?from=2026-09-10&to=2026-09-10", nil)
	req.Header.Set("Authorization", "Bearer "+readToken)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var occurrences []shift.Occurrence
	if err := json.Unmarshal(rec.Body.Bytes(), &occurrences); err != nil {
		t.Fatalf("decode occurrences response: %v", err)
	}
	if len(occurrences) != 1 {
		t.Fatalf("expected 1 occurrence, got %d: %+v", len(occurrences), occurrences)
	}
	if occurrences[0].TemplateID != created.ID || occurrences[0].Status != shift.OccurrenceStatusFree {
		t.Fatalf("unexpected occurrence: %+v", occurrences[0])
	}
	if occurrences[0].MyBookingStatus != nil {
		t.Fatalf("expected my_booking_status nil for a free slot, got %+v", occurrences[0].MyBookingStatus)
	}
}

func TestListOccurrences_MyBookingStatusReflectsCaller(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	configureToken := issuer.token(t, []string{permShiftsConfigure})
	tpl := createTestTemplate(t, server, configureToken)
	volunteerToken := issuer.tokenFor(t, testVolunteerID, []string{permShiftsRequest, permShiftsRead})

	bookingBody, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: futureThursday(t).Format(dateLayout)})
	bookingReq := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(bookingBody))
	bookingReq.Header.Set("Content-Type", "application/json")
	bookingReq.Header.Set("Authorization", "Bearer "+volunteerToken)
	bookingRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(bookingRec, bookingReq)
	if bookingRec.Code != http.StatusCreated {
		t.Fatalf("setup booking: expected 201, got %d: %s", bookingRec.Code, bookingRec.Body.String())
	}

	date := futureThursday(t).Format(dateLayout)
	req := httptest.NewRequest(http.MethodGet, "/v1/shift-occurrences?from="+date+"&to="+date, nil)
	req.Header.Set("Authorization", "Bearer "+volunteerToken)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var occurrences []shift.Occurrence
	if err := json.Unmarshal(rec.Body.Bytes(), &occurrences); err != nil {
		t.Fatalf("decode occurrences response: %v", err)
	}
	if len(occurrences) != 1 || occurrences[0].MyBookingStatus == nil || *occurrences[0].MyBookingStatus != shift.BookingStatusPending {
		t.Fatalf("expected my_booking_status=pending for the requesting volunteer, got %+v", occurrences)
	}

	otherReq := httptest.NewRequest(http.MethodGet, "/v1/shift-occurrences?from="+date+"&to="+date, nil)
	otherReq.Header.Set("Authorization", "Bearer "+issuer.token(t, []string{permShiftsRead}))
	otherRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(otherRec, otherReq)
	var otherOccurrences []shift.Occurrence
	if err := json.Unmarshal(otherRec.Body.Bytes(), &otherOccurrences); err != nil {
		t.Fatalf("decode occurrences response: %v", err)
	}
	if len(otherOccurrences) != 1 || otherOccurrences[0].MyBookingStatus != nil {
		t.Fatalf("expected my_booking_status nil for a different caller, got %+v", otherOccurrences)
	}
}
