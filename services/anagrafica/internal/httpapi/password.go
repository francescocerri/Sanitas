package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/francescocerri/sanitas/services/anagrafica/internal/user"
)

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// @Summary	Change the authenticated user's password
// @Tags		auth
// @Accept		json
// @Security	BearerAuth
// @Param		password	body	changePasswordRequest	true	"Old and new password"
// @Success	204			"Password changed"
// @Failure	400			"Invalid payload"
// @Failure	401			"Old password incorrect"
// @Router		/v1/password/change [post]
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload non valido")
		return
	}

	claims := claimsFromContext(r)
	currentHash, err := s.repo.GetPasswordHash(r.Context(), claims.Subject)
	if err != nil {
		s.logger.Error("get password hash", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}
	if !user.VerifyPassword(currentHash, req.OldPassword) {
		writeError(w, http.StatusUnauthorized, "vecchia password non corretta")
		return
	}

	newHash, err := user.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("hash password", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}
	if err := s.repo.SetPassword(r.Context(), claims.Subject, newHash); err != nil {
		s.logger.Error("set password", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
