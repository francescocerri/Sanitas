package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/francescocerri/sanitas/services/anagrafica/internal/user"
)

// refreshTokenTTL: long-lived compared to the access token (24h, see
// user.KeyPair), so a session survives without re-entering credentials.
// Rotated on every use (see handleRefresh) — easy to shorten later if 30
// days turns out too generous, not a decision that blocks anything else.
const refreshTokenTTL = 30 * 24 * time.Hour

type claimsContextKey struct{}

// requireAuth verifies the Bearer JWT (signature checked against this
// service's own key pair — see internal/user.KeyPair) and makes the claims
// available to the handler via claimsFromContext. A future service that
// only holds the public key (fetched from GET /.well-known/jwks.json)
// would do the same verification without ever calling back here.
//
// Requires the standard "Bearer <token>" header — Swagger UI's Authorize
// dialog for this scheme can't auto-prepend it (a Swagger 2.0 apiKey
// limitation, see @securityDefinitions.apikey in cmd/server/main.go), so
// type "Bearer " yourself before pasting the token there.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
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

// authTokens is the response shape shared by login and refresh: an access
// token (short-lived JWT) plus a refresh token (long-lived, opaque, single
// use — see handleRefresh).
type authTokens struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// issueTokenPair issues a fresh access token for u and a fresh refresh
// token (stored as an invite_tokens row with purpose "refresh" — see
// docs/adr/0016, the table already generalizes on purpose).
func (s *Server) issueTokenPair(ctx context.Context, u user.User) (authTokens, error) {
	token, err := s.keys.IssueToken(u)
	if err != nil {
		return authTokens{}, err
	}
	refreshToken, err := s.repo.CreateInviteToken(ctx, u.ID, "refresh", refreshTokenTTL)
	if err != nil {
		return authTokens{}, err
	}
	return authTokens{Token: token, RefreshToken: refreshToken}, nil
}

// @Summary	Login
// @Tags		auth
// @Accept		json
// @Produce	json
// @Param		credenziali	body		loginRequest	true	"Email or username + password"
// @Success	200			{object}	authTokens
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

	tokens, err := s.issueTokenPair(r.Context(), u)
	if err != nil {
		s.logger.Error("issue token pair", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// @Summary	Refresh the access token
// @Description	Consumes the given refresh token (single use — see docs/adr/0016) and
// @Description	returns a new access token plus a new refresh token.
// @Tags		auth
// @Accept		json
// @Produce	json
// @Param		refresh	body		refreshRequest	true	"Refresh token"
// @Success	200		{object}	authTokens
// @Failure	400		"Invalid payload"
// @Failure	401		"Invalid, expired, or already used token"
// @Router		/v1/refresh [post]
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	userID, err := s.repo.ConsumeInviteToken(r.Context(), req.RefreshToken, "refresh")
	if err != nil {
		if errors.Is(err, user.ErrInvalidToken) {
			writeError(w, http.StatusUnauthorized, "invalid, expired, or already used token")
			return
		}
		s.logger.Error("consume refresh token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	u, err := s.repo.GetByID(r.Context(), userID)
	if err != nil {
		s.logger.Error("get user for refresh", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	tokens, err := s.issueTokenPair(r.Context(), u)
	if err != nil {
		s.logger.Error("issue token pair", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

// @Summary	Logout
// @Description	Invalidates the given refresh token. An access token already issued stays
// @Description	valid until its own expiry (24h) — no revocation mechanism for those yet,
// @Description	see docs/adr/0013.
// @Tags		auth
// @Accept		json
// @Param		refresh	body	refreshRequest	true	"Refresh token"
// @Success	204		"Logged out"
// @Failure	400		"Invalid payload"
// @Failure	401		"Invalid, expired, or already used token"
// @Router		/v1/logout [post]
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if _, err := s.repo.ConsumeInviteToken(r.Context(), req.RefreshToken, "refresh"); err != nil {
		if errors.Is(err, user.ErrInvalidToken) {
			writeError(w, http.StatusUnauthorized, "invalid, expired, or already used token")
			return
		}
		s.logger.Error("consume refresh token on logout", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
