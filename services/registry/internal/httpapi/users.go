package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/francescocerri/sanitas/services/registry/internal/user"
)

const inviteTokenTTL = 7 * 24 * time.Hour

// passwordResetTokenTTL è volutamente molto più breve di inviteTokenTTL: un
// reset è un'operazione sensibile e ripetibile su un account già esistente,
// non ha senso un link valido per giorni come per un primo invito — vedi
// docs/adr/0024-recupero-password.md.
const passwordResetTokenTTL = 1 * time.Hour

type createUserRequest struct {
	Email    string   `json:"email"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

type createUserResponse struct {
	user.User
	// InviteURL is always returned regardless of EmailSent, as a fallback
	// the admin can still copy/forward by hand — e.g. if SMTP isn't
	// configured for this deployment, or the send itself failed.
	InviteURL string `json:"invite_url"`
	// EmailSent is best-effort: the user is already created by the time we
	// know this, so a failed send never turns the request into an error —
	// see docs/adr/0023-invio-email-invito-smtp.md.
	EmailSent bool `json:"email_sent"`
}

// @Summary	Create a new user (requires the users:manage permission)
// @Tags		users
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		utente	body		createUserRequest	true	"Email, username, roles (slug)"
// @Success	201		{object}	createUserResponse
// @Failure	400		"Invalid payload or unknown role"
// @Failure	403		"Missing required permission: users:manage"
// @Failure	409		"Email or username already in use"
// @Router		/v1/users [post]
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	roleIDsBySlug, err := s.repo.RoleIDsBySlug(r.Context(), req.Roles)
	if err != nil {
		s.logger.Error("resolve roles", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	roleIDs := make([]string, 0, len(req.Roles))
	for _, slug := range req.Roles {
		id, ok := roleIDsBySlug[slug]
		if !ok {
			writeError(w, http.StatusBadRequest, "unknown role: "+slug)
			return
		}
		roleIDs = append(roleIDs, id)
	}

	created, err := s.repo.CreatePendingUser(r.Context(), req.Email, req.Username)
	if err != nil {
		if errors.Is(err, user.ErrDuplicateUser) {
			writeError(w, http.StatusConflict, "email or username already in use")
			return
		}
		s.logger.Error("create pending user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := s.repo.AssignRoles(r.Context(), created.ID, roleIDs); err != nil {
		s.logger.Error("assign roles", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	created.Roles = req.Roles

	token, err := s.repo.CreateToken(r.Context(), created.ID, "invite", inviteTokenTTL)
	if err != nil {
		s.logger.Error("create invite token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	inviteURL := s.inviteURLBase + "?token=" + token
	emailSent := false
	if s.mailer != nil {
		// Un timeout tutto suo, più corto del WriteTimeout del server HTTP
		// (10s, vedi cmd/server/main.go): un SMTP lento o mal configurato
		// non deve mai poter bloccare la risposta abbastanza a lungo da far
		// scadere il client, che altrimenti riproverebbe con la stessa
		// email/username già creati con successo, trovandoli "duplicati".
		emailCtx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
		defer cancel()
		if err := s.mailer.SendInviteEmail(emailCtx, created.Email, created.Username, inviteURL); err != nil {
			// Non facciamo fallire la richiesta: l'utente è già stato
			// creato con successo, l'invio è un effetto collaterale
			// best-effort. L'admin ha comunque l'InviteURL come ripiego.
			s.logger.Error("send invite email", "error", err, "user_id", created.ID)
		} else {
			emailSent = true
		}
	}

	writeJSON(w, http.StatusCreated, createUserResponse{
		User:      created,
		InviteURL: inviteURL,
		EmailSent: emailSent,
	})
}

type listUsersResponse []user.User

// @Summary	List all users (requires the users:manage permission)
// @Tags		users
// @Produce	json
// @Security	BearerAuth
// @Success	200	{object}	listUsersResponse
// @Failure	403	"Missing required permission: users:manage"
// @Router		/v1/users [get]
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.repo.ListUsers(r.Context())
	if err != nil {
		s.logger.Error("list users", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, listUsersResponse(users))
}

type updateUserRolesRequest struct {
	Roles []string `json:"roles"`
}

// @Summary	Replace a user's roles (requires the users:manage permission)
// @Tags		users
// @Accept		json
// @Produce	json
// @Security	BearerAuth
// @Param		id		path		string					true	"User ID"
// @Param		ruoli	body		updateUserRolesRequest	true	"Full new set of role slugs"
// @Success	200		{object}	user.User
// @Failure	400		"Invalid payload or unknown role"
// @Failure	403		"Missing required permission: users:manage"
// @Failure	404		"User not found"
// @Router		/v1/users/{id}/roles [patch]
func (s *Server) handleUpdateUserRoles(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := s.repo.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		s.logger.Error("get user by id", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var req updateUserRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	roleIDsBySlug, err := s.repo.RoleIDsBySlug(r.Context(), req.Roles)
	if err != nil {
		s.logger.Error("resolve roles", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	roleIDs := make([]string, 0, len(req.Roles))
	for _, slug := range req.Roles {
		roleID, ok := roleIDsBySlug[slug]
		if !ok {
			writeError(w, http.StatusBadRequest, "unknown role: "+slug)
			return
		}
		roleIDs = append(roleIDs, roleID)
	}

	if err := s.repo.ReplaceRoles(r.Context(), id, roleIDs); err != nil {
		s.logger.Error("replace roles", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Rilettura fresca invece di riusare i dati della richiesta: i
	// permessi derivano dai ruoli via config, non c'è altro modo corretto
	// di calcolarli per la risposta.
	updated, err := s.repo.GetByID(r.Context(), id)
	if err != nil {
		s.logger.Error("get user by id", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
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
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	userID, err := s.repo.ConsumeToken(r.Context(), req.Token, "invite")
	if err != nil {
		if errors.Is(err, user.ErrInvalidToken) {
			writeError(w, http.StatusUnauthorized, "invalid, expired, or already used token")
			return
		}
		s.logger.Error("consume invite token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	hash, err := user.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("hash password", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := s.repo.SetPassword(r.Context(), userID, hash); err != nil {
		s.logger.Error("set password", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
