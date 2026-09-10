package httpapi

import (
	"net/http"
	"time"

	"github.com/francescocerri/sanitas/services/shifts/internal/shift"
)

const dateLayout = "2006-01-02"

// maxOccurrenceRangeDays bounds how large a single from/to request can be —
// a real calendar view never needs more than a few months at once, and
// without a cap an unbounded range could generate an unreasonably large
// response (every active template × every day in the range).
const maxOccurrenceRangeDays = 90

type listOccurrencesResponse []shift.Occurrence

// @Summary	List shift occurrences with coverage status in a date range (requires the shifts:read permission)
// @Tags		shift-occurrences
// @Produce	json
// @Security	BearerAuth
// @Param		from	query		string	true	"Start date, inclusive (YYYY-MM-DD)"
// @Param		to		query		string	true	"End date, inclusive (YYYY-MM-DD)"
// @Success	200		{object}	listOccurrencesResponse
// @Failure	400		"Missing or invalid from/to, or range too wide"
// @Failure	401		"Authentication required"
// @Failure	403		"Missing required permission: shifts:read"
// @Router		/v1/shift-occurrences [get]
func (s *Server) handleListOccurrences(w http.ResponseWriter, r *http.Request) {
	from, err := time.Parse(dateLayout, r.URL.Query().Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "from must be a valid date in YYYY-MM-DD format")
		return
	}
	to, err := time.Parse(dateLayout, r.URL.Query().Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "to must be a valid date in YYYY-MM-DD format")
		return
	}
	if to.Before(from) {
		writeError(w, http.StatusBadRequest, "to must not be before from")
		return
	}
	if to.Sub(from) > maxOccurrenceRangeDays*24*time.Hour {
		writeError(w, http.StatusBadRequest, "the range between from and to must not exceed 90 days")
		return
	}

	callerID := claimsFromContext(r).Subject
	occurrences, err := s.repo.ListOccurrences(r.Context(), from, to, callerID)
	if err != nil {
		s.logger.Error("list shift occurrences", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, listOccurrencesResponse(occurrences))
}
