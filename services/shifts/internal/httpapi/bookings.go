package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/francescocerri/sanitas/services/shifts/internal/shift"
)

type createBookingRequest struct {
	TemplateID string `json:"template_id"`
	Date       string `json:"date"`
}

// @Summary	Request a booking for an open slot (requires the shifts:request permission)
// @Tags		shift-bookings
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		booking	body		createBookingRequest	true	"Template id and date (YYYY-MM-DD)"
// @Success	201		{object}	shift.Booking
// @Failure	400		"Invalid payload, past date, inactive template, or weekday mismatch"
// @Failure	401		"Authentication required"
// @Failure	403		"Missing required permission: shifts:request"
// @Failure	404		"Template not found"
// @Failure	409		"Slot already confirmed, or the caller already has a pending/confirmed booking for it"
// @Router		/v1/shift-bookings [post]
func (s *Server) handleCreateBooking(w http.ResponseWriter, r *http.Request) {
	var req createBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	date, ok := parseAndValidateBookingDate(w, req.Date)
	if !ok {
		return
	}
	template, ok := s.getActiveTemplateForDate(w, r, req.TemplateID, date)
	if !ok {
		return
	}

	volunteerID := claimsFromContext(r).Subject
	if !s.checkSlotAvailability(w, r, template.ID, date, volunteerID,
		"you already have a pending or confirmed booking for this slot") {
		return
	}

	created, err := s.repo.CreateBooking(r.Context(), shift.Booking{
		TemplateID:  template.ID,
		VolunteerID: volunteerID,
		Date:        date,
		StartTime:   template.StartTime,
		EndTime:     template.EndTime,
	})
	if err != nil {
		s.logger.Error("create booking", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// parseAndValidateBookingDate parses raw and checks it's not in the past,
// writing the appropriate 400 and returning ok=false on either failure —
// shared by handleCreateBooking and handleCreateDirectBooking.
func parseAndValidateBookingDate(w http.ResponseWriter, raw string) (date time.Time, ok bool) {
	date, err := time.Parse(dateLayout, raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "date must be a valid date in YYYY-MM-DD format")
		return time.Time{}, false
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if date.Before(today) {
		writeError(w, http.StatusBadRequest, "date must not be in the past")
		return time.Time{}, false
	}
	return date, true
}

// getActiveTemplateForDate fetches templateID, checks it's active and that
// date falls on its weekday, writing the appropriate error and returning
// ok=false on any failure — shared by handleCreateBooking and
// handleCreateDirectBooking.
func (s *Server) getActiveTemplateForDate(w http.ResponseWriter, r *http.Request, templateID string, date time.Time) (template shift.ShiftTemplate, ok bool) {
	template, err := s.repo.GetTemplate(r.Context(), templateID)
	if err != nil {
		if errors.Is(err, shift.ErrNotFound) {
			writeError(w, http.StatusNotFound, "template not found")
			return shift.ShiftTemplate{}, false
		}
		s.logger.Error("get shift template", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return shift.ShiftTemplate{}, false
	}
	if !template.Active {
		writeError(w, http.StatusBadRequest, "template is not active")
		return shift.ShiftTemplate{}, false
	}
	if int(date.Weekday()) != template.Weekday {
		writeError(w, http.StatusBadRequest, "date does not fall on the template's weekday")
		return shift.ShiftTemplate{}, false
	}
	return template, true
}

// checkSlotAvailability rejects a booking attempt that would land on an
// already-confirmed slot, or duplicate a pending/confirmed booking
// volunteerID already has for it — shared by handleCreateBooking and
// handleCreateDirectBooking (different duplicateMessage: "you already
// have..." for a volunteer's own request vs "the chosen volunteer already
// has..." for a manager's direct booking).
func (s *Server) checkSlotAvailability(w http.ResponseWriter, r *http.Request, templateID string, date time.Time, volunteerID, duplicateMessage string) bool {
	ctx := r.Context()
	confirmed, err := s.repo.HasConfirmedBooking(ctx, templateID, date)
	if err != nil {
		s.logger.Error("check confirmed booking", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return false
	}
	if confirmed {
		writeError(w, http.StatusConflict, "this slot is already confirmed")
		return false
	}

	hasBooking, err := s.repo.HasBookingForVolunteer(ctx, templateID, date, volunteerID)
	if err != nil {
		s.logger.Error("check volunteer booking", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return false
	}
	if hasBooking {
		writeError(w, http.StatusConflict, duplicateMessage)
		return false
	}
	return true
}

type createDirectBookingRequest struct {
	TemplateID  string `json:"template_id"`
	VolunteerID string `json:"volunteer_id"`
	Date        string `json:"date"`
}

// @Summary	Book a slot directly for a chosen volunteer, already confirmed (requires the shifts:write permission)
// @Tags		shift-bookings
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		booking	body		createDirectBookingRequest	true	"Template id, volunteer id, and date (YYYY-MM-DD)"
// @Success	201		{object}	shift.Booking
// @Failure	400		"Invalid payload, empty/unknown volunteer_id, past date, inactive template, or weekday mismatch"
// @Failure	401		"Authentication required"
// @Failure	403		"Missing required permission: shifts:write"
// @Failure	404		"Template not found"
// @Failure	409		"Slot already confirmed, or the chosen volunteer already has a pending/confirmed booking for it"
// @Router		/v1/shift-bookings/direct [post]
func (s *Server) handleCreateDirectBooking(w http.ResponseWriter, r *http.Request) {
	var req createDirectBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.VolunteerID == "" {
		writeError(w, http.StatusBadRequest, "volunteer_id is required")
		return
	}
	date, ok := parseAndValidateBookingDate(w, req.Date)
	if !ok {
		return
	}
	template, ok := s.getActiveTemplateForDate(w, r, req.TemplateID, date)
	if !ok {
		return
	}
	if !s.checkSlotAvailability(w, r, template.ID, date, req.VolunteerID,
		"the chosen volunteer already has a pending or confirmed booking for this slot") {
		return
	}

	decidedBy := claimsFromContext(r).Subject
	created, err := s.repo.CreateConfirmedBooking(r.Context(), shift.Booking{
		TemplateID:  template.ID,
		VolunteerID: req.VolunteerID,
		Date:        date,
		StartTime:   template.StartTime,
		EndTime:     template.EndTime,
	}, decidedBy)
	if err != nil {
		if errors.Is(err, shift.ErrUnknownVolunteer) {
			writeError(w, http.StatusBadRequest, "unknown volunteer_id")
			return
		}
		s.logger.Error("create direct booking", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

type decideBookingRequest struct {
	Status string `json:"status"`
}

// @Summary	Approve or reject a pending booking (requires the shifts:write permission)
// @Tags		shift-bookings
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		id		path		string					true	"Booking id (UUID)"
// @Param		decision	body		decideBookingRequest	true	"status: confirmed or rejected"
// @Success	200		{object}	shift.Booking
// @Failure	400		"Invalid payload, or status is not confirmed/rejected"
// @Failure	401		"Authentication required"
// @Failure	403		"Missing required permission: shifts:write"
// @Failure	404		"Booking not found"
// @Failure	409		"Booking is no longer pending"
// @Router		/v1/shift-bookings/{id} [patch]
func (s *Server) handleDecideBooking(w http.ResponseWriter, r *http.Request) {
	var req decideBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	var status shift.BookingStatus
	switch req.Status {
	case string(shift.BookingStatusConfirmed):
		status = shift.BookingStatusConfirmed
	case string(shift.BookingStatusRejected):
		status = shift.BookingStatusRejected
	default:
		writeError(w, http.StatusBadRequest, `status must be "confirmed" or "rejected"`)
		return
	}

	id := r.PathValue("id")
	decidedBy := claimsFromContext(r).Subject
	decided, err := s.repo.DecideBooking(r.Context(), id, status, decidedBy)
	if err != nil {
		if errors.Is(err, shift.ErrNotFound) {
			writeError(w, http.StatusNotFound, "booking not found")
			return
		}
		if errors.Is(err, shift.ErrBookingNotPending) {
			writeError(w, http.StatusConflict, "booking is no longer pending")
			return
		}
		s.logger.Error("decide booking", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, decided)
}

type pendingBookingsCountResponse struct {
	PendingCount int `json:"pending_count"`
}

// @Summary	Count pending booking requests (requires the shifts:write permission)
// @Tags		shift-bookings
// @Produce	json
// @Security	BearerAuth
// @Success	200	{object}	pendingBookingsCountResponse
// @Failure	401	"Authentication required"
// @Failure	403	"Missing required permission: shifts:write"
// @Router		/v1/shift-bookings/pending-count [get]
func (s *Server) handlePendingBookingsCount(w http.ResponseWriter, r *http.Request) {
	count, err := s.repo.CountPendingBookings(r.Context())
	if err != nil {
		s.logger.Error("count pending bookings", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, pendingBookingsCountResponse{PendingCount: count})
}
