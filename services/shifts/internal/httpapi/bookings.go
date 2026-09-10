package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	date, verr := parseAndValidateBookingDate(req.Date)
	if verr != nil {
		writeError(w, verr.status, verr.message)
		return
	}
	template, verr := s.getActiveTemplateForDate(r.Context(), req.TemplateID, date)
	if verr != nil {
		writeError(w, verr.status, verr.message)
		return
	}

	volunteerID := claimsFromContext(r).Subject
	if verr := s.checkSlotAvailability(r.Context(), template.ID, date, volunteerID,
		"you already have a pending or confirmed booking for this slot"); verr != nil {
		writeError(w, verr.status, verr.message)
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

// validationError carries the HTTP status/message a failed validation step
// should produce, without writing to a ResponseWriter directly — lets the
// same validation helpers serve both a single-item handler (writes the
// error immediately) and the bulk handler (needs to prefix it with which
// item in the list failed before writing anything).
type validationError struct {
	status  int
	message string
}

func (e *validationError) Error() string { return e.message }

// parseAndValidateBookingDate parses raw and checks it's not in the past —
// shared by handleCreateBooking, handleCreateDirectBooking and
// handleCreateBulkBooking.
func parseAndValidateBookingDate(raw string) (time.Time, *validationError) {
	date, err := time.Parse(dateLayout, raw)
	if err != nil {
		return time.Time{}, &validationError{http.StatusBadRequest, "date must be a valid date in YYYY-MM-DD format"}
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if date.Before(today) {
		return time.Time{}, &validationError{http.StatusBadRequest, "date must not be in the past"}
	}
	return date, nil
}

// getActiveTemplateForDate fetches templateID, checks it's active and that
// date falls on its weekday — shared by handleCreateBooking,
// handleCreateDirectBooking and handleCreateBulkBooking.
func (s *Server) getActiveTemplateForDate(ctx context.Context, templateID string, date time.Time) (shift.ShiftTemplate, *validationError) {
	template, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		if errors.Is(err, shift.ErrNotFound) {
			return shift.ShiftTemplate{}, &validationError{http.StatusNotFound, "template not found"}
		}
		s.logger.Error("get shift template", "error", err)
		return shift.ShiftTemplate{}, &validationError{http.StatusInternalServerError, "internal error"}
	}
	if !template.Active {
		return shift.ShiftTemplate{}, &validationError{http.StatusBadRequest, "template is not active"}
	}
	if int(date.Weekday()) != template.Weekday {
		return shift.ShiftTemplate{}, &validationError{http.StatusBadRequest, "date does not fall on the template's weekday"}
	}
	return template, nil
}

// checkSlotAvailability rejects a booking attempt that would land on an
// already-confirmed slot, or duplicate a pending/confirmed booking
// volunteerID already has for it — shared by handleCreateBooking,
// handleCreateDirectBooking and handleCreateBulkBooking (different
// duplicateMessage: "you already have..." for a volunteer's own request vs
// "the chosen volunteer already has..." for a manager's direct booking).
func (s *Server) checkSlotAvailability(ctx context.Context, templateID string, date time.Time, volunteerID, duplicateMessage string) *validationError {
	confirmed, err := s.repo.HasConfirmedBooking(ctx, templateID, date)
	if err != nil {
		s.logger.Error("check confirmed booking", "error", err)
		return &validationError{http.StatusInternalServerError, "internal error"}
	}
	if confirmed {
		return &validationError{http.StatusConflict, "this slot is already confirmed"}
	}

	hasBooking, err := s.repo.HasBookingForVolunteer(ctx, templateID, date, volunteerID)
	if err != nil {
		s.logger.Error("check volunteer booking", "error", err)
		return &validationError{http.StatusInternalServerError, "internal error"}
	}
	if hasBooking {
		return &validationError{http.StatusConflict, duplicateMessage}
	}
	return nil
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
	date, verr := parseAndValidateBookingDate(req.Date)
	if verr != nil {
		writeError(w, verr.status, verr.message)
		return
	}
	template, verr := s.getActiveTemplateForDate(r.Context(), req.TemplateID, date)
	if verr != nil {
		writeError(w, verr.status, verr.message)
		return
	}
	if verr := s.checkSlotAvailability(r.Context(), template.ID, date, req.VolunteerID,
		"the chosen volunteer already has a pending or confirmed booking for this slot"); verr != nil {
		writeError(w, verr.status, verr.message)
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

type createBulkBookingItem struct {
	TemplateID string `json:"template_id"`
	Date       string `json:"date"`
}

type createBulkBookingRequest struct {
	Bookings []createBulkBookingItem `json:"bookings"`
}

// @Summary	Request bookings for multiple open slots in one all-or-nothing call (requires the shifts:request permission)
// @Tags		shift-bookings
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		bookings	body		createBulkBookingRequest	true	"Non-empty list of template id + date (YYYY-MM-DD) pairs, no duplicates"
// @Success	201			{array}		shift.Booking
// @Failure	400			"Invalid payload, empty/duplicate list, or the Nth item has a past/malformed date, inactive template, or weekday mismatch"
// @Failure	401			"Authentication required"
// @Failure	403			"Missing required permission: shifts:request"
// @Failure	404			"The Nth item's template was not found"
// @Failure	409			"The Nth item's slot is already confirmed, or the caller already has a pending/confirmed booking for it"
// @Router		/v1/shift-bookings/bulk [post]
func (s *Server) handleCreateBulkBooking(w http.ResponseWriter, r *http.Request) {
	var req createBulkBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if len(req.Bookings) == 0 {
		writeError(w, http.StatusBadRequest, "at least one booking is required")
		return
	}

	// A duplicate (template_id, date) pair within the same request would
	// otherwise pass per-item validation twice — neither one exists in the
	// DB yet at check time — and end up inserted twice.
	type slot struct{ templateID, date string }
	seen := make(map[slot]bool, len(req.Bookings))
	for _, item := range req.Bookings {
		key := slot{item.TemplateID, item.Date}
		if seen[key] {
			writeError(w, http.StatusBadRequest, "duplicate template_id/date pair in request")
			return
		}
		seen[key] = true
	}

	ctx := r.Context()
	volunteerID := claimsFromContext(r).Subject
	toCreate := make([]shift.Booking, 0, len(req.Bookings))
	for i, item := range req.Bookings {
		date, verr := parseAndValidateBookingDate(item.Date)
		if verr != nil {
			writeError(w, verr.status, fmt.Sprintf("booking %d: %s", i+1, verr.message))
			return
		}
		template, verr := s.getActiveTemplateForDate(ctx, item.TemplateID, date)
		if verr != nil {
			writeError(w, verr.status, fmt.Sprintf("booking %d: %s", i+1, verr.message))
			return
		}
		if verr := s.checkSlotAvailability(ctx, template.ID, date, volunteerID,
			"you already have a pending or confirmed booking for this slot"); verr != nil {
			writeError(w, verr.status, fmt.Sprintf("booking %d: %s", i+1, verr.message))
			return
		}
		toCreate = append(toCreate, shift.Booking{
			TemplateID:  template.ID,
			VolunteerID: volunteerID,
			Date:        date,
			StartTime:   template.StartTime,
			EndTime:     template.EndTime,
		})
	}

	created, err := s.repo.CreateBookingsAtomic(ctx, toCreate)
	if err != nil {
		s.logger.Error("create bulk bookings", "error", err)
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
