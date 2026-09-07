package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/francescocerri/sanitas/services/shifts/internal/shift"
)

// timeOfDayPattern matches a 24h "HH:MM" time of day — the same shape
// ShiftTemplate.StartTime/EndTime are stored in (see internal/shift/template.go).
var timeOfDayPattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func validWeekday(w int) bool { return w >= 0 && w <= 6 }

// validateTemplateFields checks the fields the DB's own CHECK constraint
// and NOT NULL columns would otherwise reject — validating here turns a
// malformed request into a clean 400 instead of a raw constraint-violation
// 500.
func validateTemplateFields(weekday int, startTime, endTime string) error {
	if !validWeekday(weekday) {
		return fmt.Errorf("weekday must be between 0 (Sunday) and 6 (Saturday)")
	}
	if !timeOfDayPattern.MatchString(startTime) {
		return fmt.Errorf("start_time must be in HH:MM format")
	}
	if !timeOfDayPattern.MatchString(endTime) {
		return fmt.Errorf("end_time must be in HH:MM format")
	}
	return nil
}

type createTemplateRequest struct {
	Weekday   int    `json:"weekday"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Label     string `json:"label"`
}

// @Summary	Create a shift template (requires the shifts:configure permission)
// @Tags		shift-templates
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		template	body		createTemplateRequest	true	"Weekday (0=Sunday..6=Saturday), start/end time (HH:MM), label"
// @Success	201			{object}	shift.ShiftTemplate
// @Failure	400			"Invalid payload"
// @Failure	401			"Authentication required"
// @Failure	403			"Missing required permission: shifts:configure"
// @Router		/v1/shift-templates [post]
func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req createTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validateTemplateFields(req.Weekday, req.StartTime, req.EndTime); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := s.repo.CreateTemplate(r.Context(), shift.ShiftTemplate{
		Weekday:   req.Weekday,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Label:     req.Label,
	})
	if err != nil {
		s.logger.Error("create shift template", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

type listShiftTemplatesResponse []shift.ShiftTemplate

// @Summary	List shift templates (requires the shifts:read permission)
// @Tags		shift-templates
// @Produce	json
// @Security	BearerAuth
// @Success	200	{object}	listShiftTemplatesResponse
// @Failure	401	"Authentication required"
// @Failure	403	"Missing required permission: shifts:read"
// @Router		/v1/shift-templates [get]
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.repo.ListTemplates(r.Context())
	if err != nil {
		s.logger.Error("list shift templates", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, listShiftTemplatesResponse(templates))
}

type updateTemplateRequest struct {
	Weekday   int    `json:"weekday"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Label     string `json:"label"`
	Active    bool   `json:"active"`
}

// @Summary	Replace a shift template (requires the shifts:configure permission)
// @Tags		shift-templates
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		id			path		string					true	"Template id (UUID)"
// @Param		template	body		updateTemplateRequest	true	"Full new set of fields"
// @Success	200			{object}	shift.ShiftTemplate
// @Failure	400			"Invalid payload"
// @Failure	401			"Authentication required"
// @Failure	403			"Missing required permission: shifts:configure"
// @Failure	404			"Template not found"
// @Router		/v1/shift-templates/{id} [patch]
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req updateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validateTemplateFields(req.Weekday, req.StartTime, req.EndTime); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := s.repo.UpdateTemplate(r.Context(), id, shift.ShiftTemplate{
		Weekday:   req.Weekday,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Label:     req.Label,
		Active:    req.Active,
	})
	if err != nil {
		if errors.Is(err, shift.ErrNotFound) {
			writeError(w, http.StatusNotFound, "template not found")
			return
		}
		s.logger.Error("update shift template", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
