package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/francescocerri/sanitas/services/shifts/internal/shift"
)

// newRegistryUser seeds a second registry.users row — used where a test
// needs two distinct volunteers (e.g. one whose booking gets confirmed,
// another whose competing request must then be rejected), since
// testVolunteerID alone only gives one.
func newRegistryUser(t *testing.T) string {
	t.Helper()
	var id string
	if err := testDB.Raw(`INSERT INTO registry.users DEFAULT VALUES RETURNING id`).Scan(&id).Error; err != nil {
		t.Fatalf("seed second registry user: %v", err)
	}
	return id
}

// futureThursday is a date guaranteed to be in the future relative to
// whenever the test suite runs, matching the fixture template's weekday
// (created with Weekday: 4, i.e. Thursday) — see createTestTemplate below.
func futureThursday(t *testing.T) time.Time {
	t.Helper()
	d := time.Now().UTC().AddDate(0, 0, 14)
	for d.Weekday() != time.Thursday {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

// createTestTemplate creates an active Thursday template via the real
// HTTP endpoint (not the repository directly) — keeps these tests
// exercising the same path a real shift manager would use.
func createTestTemplate(t *testing.T, server *Server, configureToken string) shift.ShiftTemplate {
	t.Helper()
	body, _ := json.Marshal(createTemplateRequest{Weekday: int(time.Thursday), StartTime: "20:00", EndTime: "08:00", Label: "Turno Serale"})
	req := httptest.NewRequest(http.MethodPost, "/v1/shift-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+configureToken)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("createTestTemplate: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var tpl shift.ShiftTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &tpl); err != nil {
		t.Fatalf("createTestTemplate: decode: %v", err)
	}
	return tpl
}

func TestCreateBooking_RequiresRequestPermission(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	tpl := createTestTemplate(t, server, issuer.token(t, []string{permShiftsConfigure}))

	body, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: futureThursday(t).Format(dateLayout)})
	req := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+issuer.tokenFor(t, testVolunteerID, []string{permShiftsRead}))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without shifts:request, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBooking_Success(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	tpl := createTestTemplate(t, server, issuer.token(t, []string{permShiftsConfigure}))
	volunteerToken := issuer.tokenFor(t, testVolunteerID, []string{permShiftsRequest})

	body, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: futureThursday(t).Format(dateLayout)})
	req := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+volunteerToken)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created shift.Booking
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.VolunteerID != testVolunteerID {
		t.Fatalf("expected volunteer_id to come from the token subject (%s), got %s", testVolunteerID, created.VolunteerID)
	}
	if created.Status != shift.BookingStatusPending {
		t.Fatalf("expected status pending, got %s", created.Status)
	}
}

func TestCreateBooking_RejectsPastDate(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	tpl := createTestTemplate(t, server, issuer.token(t, []string{permShiftsConfigure}))

	body, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: "2020-01-02"}) // a Thursday, but in the past
	req := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+issuer.tokenFor(t, testVolunteerID, []string{permShiftsRequest}))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBooking_RejectsUnknownTemplate(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)

	body, _ := json.Marshal(createBookingRequest{TemplateID: "00000000-0000-0000-0000-000000000000", Date: futureThursday(t).Format(dateLayout)})
	req := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+issuer.tokenFor(t, testVolunteerID, []string{permShiftsRequest}))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBooking_RejectsInactiveTemplate(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	configureToken := issuer.token(t, []string{permShiftsConfigure})
	tpl := createTestTemplate(t, server, configureToken)

	updateBody, _ := json.Marshal(updateTemplateRequest{Weekday: tpl.Weekday, StartTime: tpl.StartTime, EndTime: tpl.EndTime, Label: tpl.Label, Active: false})
	updateReq := httptest.NewRequest(http.MethodPatch, "/v1/shift-templates/"+tpl.ID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+configureToken)
	updateRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("deactivate template: expected 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	body, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: futureThursday(t).Format(dateLayout)})
	req := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+issuer.tokenFor(t, testVolunteerID, []string{permShiftsRequest}))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBooking_RejectsWeekdayMismatch(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	tpl := createTestTemplate(t, server, issuer.token(t, []string{permShiftsConfigure})) // Thursday

	notThursday := futureThursday(t).AddDate(0, 0, 1) // Friday
	body, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: notThursday.Format(dateLayout)})
	req := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+issuer.tokenFor(t, testVolunteerID, []string{permShiftsRequest}))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateBooking_RejectsWhenSlotAlreadyConfirmed(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	tpl := createTestTemplate(t, server, issuer.token(t, []string{permShiftsConfigure}))
	volunteerToken := issuer.tokenFor(t, testVolunteerID, []string{permShiftsRequest})
	date := futureThursday(t).Format(dateLayout)

	firstBody, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: date})
	firstReq := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(firstBody))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.Header.Set("Authorization", "Bearer "+volunteerToken)
	firstRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(firstRec, firstReq)
	var created shift.Booking
	if err := json.Unmarshal(firstRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode first booking: %v", err)
	}
	if err := testDB.Model(&shift.Booking{}).Where("id = ?", created.ID).Update("status", shift.BookingStatusConfirmed).Error; err != nil {
		t.Fatalf("confirm booking: %v", err)
	}

	secondBody, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: date})
	secondReq := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(secondBody))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("Authorization", "Bearer "+issuer.tokenFor(t, newRegistryUser(t), []string{permShiftsRequest}))
	secondRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", secondRec.Code, secondRec.Body.String())
	}
}

func TestCreateBooking_RejectsDuplicateOwnRequest(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	tpl := createTestTemplate(t, server, issuer.token(t, []string{permShiftsConfigure}))
	volunteerToken := issuer.tokenFor(t, testVolunteerID, []string{permShiftsRequest})
	date := futureThursday(t).Format(dateLayout)

	for i, wantCode := range []int{http.StatusCreated, http.StatusConflict} {
		body, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: date})
		req := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+volunteerToken)
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		if rec.Code != wantCode {
			t.Fatalf("attempt %d: expected %d, got %d: %s", i+1, wantCode, rec.Code, rec.Body.String())
		}
	}
}

func TestPendingBookingsCount_RequiresWritePermission(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/shift-bookings/pending-count", nil)
	req.Header.Set("Authorization", "Bearer "+issuer.token(t, []string{permShiftsRead}))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPendingBookingsCount_ReturnsCount(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	tpl := createTestTemplate(t, server, issuer.token(t, []string{permShiftsConfigure}))
	volunteerToken := issuer.tokenFor(t, testVolunteerID, []string{permShiftsRequest})

	body, _ := json.Marshal(createBookingRequest{TemplateID: tpl.ID, Date: futureThursday(t).Format(dateLayout)})
	req := httptest.NewRequest(http.MethodPost, "/v1/shift-bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+volunteerToken)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup booking: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	countReq := httptest.NewRequest(http.MethodGet, "/v1/shift-bookings/pending-count", nil)
	countReq.Header.Set("Authorization", "Bearer "+issuer.token(t, []string{permShiftsWrite}))
	countRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(countRec, countReq)
	if countRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", countRec.Code, countRec.Body.String())
	}
	var got pendingBookingsCountResponse
	if err := json.Unmarshal(countRec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.PendingCount != 1 {
		t.Fatalf("expected pending_count=1, got %d", got.PendingCount)
	}
}
