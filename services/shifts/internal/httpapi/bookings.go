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
	date, err := time.Parse(dateLayout, req.Date)
	if err != nil {
		writeError(w, http.StatusBadRequest, "date must be a valid date in YYYY-MM-DD format")
		return
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if date.Before(today) {
		writeError(w, http.StatusBadRequest, "date must not be in the past")
		return
	}

	ctx := r.Context()
	template, err := s.repo.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		if errors.Is(err, shift.ErrNotFound) {
			writeError(w, http.StatusNotFound, "template not found")
			return
		}
		s.logger.Error("get shift template", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !template.Active {
		writeError(w, http.StatusBadRequest, "template is not active")
		return
	}
	if int(date.Weekday()) != template.Weekday {
		writeError(w, http.StatusBadRequest, "date does not fall on the template's weekday")
		return
	}

	confirmed, err := s.repo.HasConfirmedBooking(ctx, template.ID, date)
	if err != nil {
		s.logger.Error("check confirmed booking", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if confirmed {
		writeError(w, http.StatusConflict, "this slot is already confirmed")
		return
	}

	volunteerID := claimsFromContext(r).Subject
	alreadyRequested, err := s.repo.HasBookingForVolunteer(ctx, template.ID, date, volunteerID)
	if err != nil {
		s.logger.Error("check own booking", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if alreadyRequested {
		writeError(w, http.StatusConflict, "you already have a pending or confirmed booking for this slot")
		return
	}

	created, err := s.repo.CreateBooking(ctx, shift.Booking{
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
