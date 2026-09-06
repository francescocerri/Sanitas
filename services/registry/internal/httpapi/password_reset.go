package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/francescocerri/sanitas/services/registry/internal/user"
)

type requestPasswordResetRequest struct {
	Identifier string `json:"identifier"`
}

// @Summary	Request a password reset email
// @Tags		auth
// @Accept		json
// @Param		richiesta	body	requestPasswordResetRequest	true	"Email or username"
// @Success	204			"Always returned, whether or not the identifier matches an account"
// @Failure	400			"Invalid payload"
// @Router		/v1/password/reset/request [post]
func (s *Server) handleRequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req requestPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	u, _, err := s.repo.GetByLogin(r.Context(), req.Identifier)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			// Stessa risposta di un identifier trovato: mai rivelare se un
			// account esiste (vedi docs/adr/0024-recupero-password.md).
			w.WriteHeader(http.StatusNoContent)
			return
		}
		s.logger.Error("get user by login", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	token, err := s.repo.CreateToken(r.Context(), u.ID, "password_reset", passwordResetTokenTTL)
	if err != nil {
		s.logger.Error("create password reset token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if s.mailer != nil {
		resetURL := s.passwordResetURLBase + "?token=" + token
		// Stesso timeout indipendente già usato per l'invito (users.go): un
		// SMTP lento non deve poter ritardare la risposta oltre il
		// WriteTimeout del server HTTP.
		emailCtx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
		defer cancel()
		if err := s.mailer.SendPasswordResetEmail(emailCtx, u.Email, u.Username, resetURL); err != nil {
			// Best-effort, come per l'invito: mai far fallire la richiesta,
			// e comunque non c'è nulla di utile da restituire al chiamante
			// (la risposta è sempre 204, vedi sopra).
			s.logger.Error("send password reset email", "error", err, "user_id", u.ID)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

type confirmPasswordResetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// @Summary	Confirm a password reset via the emailed token, setting a new password
// @Tags		auth
// @Accept		json
// @Param		conferma	body	confirmPasswordResetRequest	true	"Reset token and new password"
// @Success	204			"Password reset"
// @Failure	400			"Invalid payload"
// @Failure	401			"Invalid, expired, or already used token"
// @Router		/v1/password/reset/confirm [post]
func (s *Server) handleConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req confirmPasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	userID, err := s.repo.ConsumeToken(r.Context(), req.Token, "password_reset")
	if err != nil {
		if errors.Is(err, user.ErrInvalidToken) {
			writeError(w, http.StatusUnauthorized, "invalid, expired, or already used token")
			return
		}
		s.logger.Error("consume password reset token", "error", err)
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
