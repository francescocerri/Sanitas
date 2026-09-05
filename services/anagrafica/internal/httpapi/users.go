package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/francescocerri/sanitas/services/anagrafica/internal/user"
)

const inviteTokenTTL = 7 * 24 * time.Hour

type createUserRequest struct {
	Email    string   `json:"email"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

type createUserResponse struct {
	user.User
	// InviteURL is handed back directly in this response instead of being
	// emailed: sending it for real (Gmail SMTP) is a separate follow-up
	// activity, this phase only lays the token groundwork. The admin
	// forwards it to the volunteer by whatever channel they already use.
	InviteURL string `json:"invite_url"`
}

// @Summary	Create a new user (admin only)
// @Tags		users
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		utente	body		createUserRequest	true	"Email, username, roles (slug)"
// @Success	201		{object}	createUserResponse
// @Failure	400		"Invalid payload or unknown role"
// @Failure	403		"Admin permission required"
// @Failure	409		"Email or username already in use"
// @Router		/v1/users [post]
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload non valido")
		return
	}

	roleIDsBySlug, err := s.repo.RoleIDsBySlug(r.Context(), req.Roles)
	if err != nil {
		s.logger.Error("resolve roles", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}
	roleIDs := make([]string, 0, len(req.Roles))
	for _, slug := range req.Roles {
		id, ok := roleIDsBySlug[slug]
		if !ok {
			writeError(w, http.StatusBadRequest, "ruolo sconosciuto: "+slug)
			return
		}
		roleIDs = append(roleIDs, id)
	}

	created, err := s.repo.CreatePendingUser(r.Context(), req.Email, req.Username)
	if err != nil {
		if errors.Is(err, user.ErrDuplicateUser) {
			writeError(w, http.StatusConflict, "email o username già in uso")
			return
		}
		s.logger.Error("create pending user", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}

	if err := s.repo.AssignRoles(r.Context(), created.ID, roleIDs); err != nil {
		s.logger.Error("assign roles", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}
	created.Roles = req.Roles

	token, err := s.repo.CreateInviteToken(r.Context(), created.ID, "invite", inviteTokenTTL)
	if err != nil {
		s.logger.Error("create invite token", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}

	writeJSON(w, http.StatusCreated, createUserResponse{
		User:      created,
		InviteURL: s.inviteURLBase + "?token=" + token,
	})
}

type activateUserRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// @Summary	Activate an account via the invite token, setting the password
// @Tags		users
// @Accept		json
// @Produce	json
// @Param		attivazione	body	activateUserRequest	true	"Invite token and new password"
// @Success	204			"Account activated"
// @Failure	400			"Invalid payload"
// @Failure	401			"Invalid, expired, or already used token"
// @Router		/v1/users/activate [post]
func (s *Server) handleActivateUser(w http.ResponseWriter, r *http.Request) {
	var req activateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload non valido")
		return
	}

	userID, err := s.repo.ConsumeInviteToken(r.Context(), req.Token, "invite")
	if err != nil {
		if errors.Is(err, user.ErrInvalidToken) {
			writeError(w, http.StatusUnauthorized, "token non valido, scaduto o già usato")
			return
		}
		s.logger.Error("consume invite token", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}

	hash, err := user.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("hash password", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}
	if err := s.repo.SetPassword(r.Context(), userID, hash); err != nil {
		s.logger.Error("set password", "error", err)
		writeError(w, http.StatusInternalServerError, "errore interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
