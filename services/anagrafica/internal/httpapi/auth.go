package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/francescocerri/sanitas/services/anagrafica/internal/user"
)

type claimsContextKey struct{}

// requireAuth verifies the Bearer JWT (signature checked against this
// service's own key pair — see internal/user.KeyPair) and makes the claims
// available to the handler via claimsFromContext. A future service that
// only holds the public key (fetched from GET /.well-known/jwks.json)
// would do the same verification without ever calling back here.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		claims, err := s.keys.Verify(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), claimsContextKey{}, claims)
		next(w, r.WithContext(ctx))
	}
}

// requireAdmin builds on requireAuth: the system permission to manage
// accounts (distinct from the organizational roles — see docs/adr/0012).
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := r.Context().Value(claimsContextKey{}).(*user.Claims)
		if claims == nil || !claims.IsAdmin {
			writeError(w, http.StatusForbidden, "admin permission required")
			return
		}
		next(w, r)
	})
}

func claimsFromContext(r *http.Request) *user.Claims {
	claims, _ := r.Context().Value(claimsContextKey{}).(*user.Claims)
	return claims
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// @Summary	Login
// @Tags		auth
// @Accept		json
// @Produce	json
// @Param		credenziali	body		loginRequest	true	"Email or username + password"
// @Success	200			{object}	map[string]string
// @Failure	400			"Invalid payload"
// @Failure	401			"Invalid credentials"
// @Router		/v1/login [post]
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	u, hash, err := s.repo.GetByLogin(r.Context(), req.Identifier)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !user.VerifyPassword(hash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := s.keys.IssueToken(u)
	if err != nil {
		s.logger.Error("issue token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// @Summary	Get the authenticated user's profile
// @Tags		auth
// @Produce	json
// @Security	BearerAuth
// @Success	200	{object}	user.User
// @Failure	401	"Authentication required"
// @Router		/v1/me [get]
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r)
	u, err := s.repo.GetByID(r.Context(), claims.Subject)
	if err != nil {
		s.logger.Error("get me", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// @Summary	Public key to verify JWTs (JWKS format)
// @Tags		system
// @Produce	json
// @Success	200	{object}	map[string]any
// @Router		/.well-known/jwks.json [get]
func (s *Server) handleJWKS(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.keys.JWKS())
}
