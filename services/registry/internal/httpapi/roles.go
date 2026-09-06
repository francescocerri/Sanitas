package httpapi

import (
	"github.com/francescocerri/sanitas/services/registry/internal/user"
	"net/http"
)

type listRolesResponse []user.Role

// @Summary List available roles (any authenticated user)
// @Tags roles
// @Produce json
// @Security BearerAuth
// @Success 200 {object} listRolesResponse
// @Failure 401 "Authentication required"
// @Failure 500 "Internal error"
// @Router /v1/roles [get]
func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := s.repo.ListRoles(r.Context())
	if err != nil {
		s.logger.Error("list roles", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, listRolesResponse(roles))
}
