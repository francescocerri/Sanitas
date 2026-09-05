package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/francescocerri/sanitas/services/anagrafica/internal/user"
)

type Server struct {
	repo          *user.Repository
	keys          *user.KeyPair
	allowedOrigin string
	inviteURLBase string
	logger        *slog.Logger
}

func NewServer(repo *user.Repository, keys *user.KeyPair, allowedOrigin, inviteURLBase string, logger *slog.Logger) *Server {
	return &Server{repo: repo, keys: keys, allowedOrigin: allowedOrigin, inviteURLBase: inviteURLBase, logger: logger}
}

// v1 versions the resource endpoints only: /healthz, /docs/ and
// /.well-known/jwks.json are operational/meta, not part of the API
// contract that evolves — same convention as services/turni.
const v1 = "/v1"

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /.well-known/jwks.json", s.handleJWKS)
	mux.HandleFunc("POST "+v1+"/login", s.handleLogin)
	mux.HandleFunc("GET "+v1+"/me", s.requireAuth(s.handleMe))
	mux.HandleFunc("POST "+v1+"/utenti", s.requireAdmin(s.handleCreateUser))
	mux.HandleFunc("POST "+v1+"/utenti/attiva", s.handleActivateUser)
	mux.HandleFunc("POST "+v1+"/password/cambia", s.requireAuth(s.handleChangePassword))
	mux.Handle("GET /docs/", docsHandler())
	return s.withLogging(s.withCORS(mux))
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// statusRecorder captures the status code written by the handler, so it can
// be included in the access log (http.ResponseWriter doesn't expose it).
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

// piiJSONFields lists the JSON field names this service's request bodies
// may contain that identify a person or a credential — wider than turni's
// denylist since this service's whole job is handling identity. Extend as
// new PII-bearing fields are added.
var piiJSONFields = map[string]bool{
	"email":        true,
	"username":     true,
	"password":     true,
	"old_password": true,
	"new_password": true,
	// Not personal data, but a bearer credential: whoever reads the logs
	// could activate the account before the real invitee does.
	"token": true,
}

// withLogging combines structured access logging and panic recovery, and
// for POST requests also logs the body with PII redacted — same pattern as
// services/turni (see docs/adr/0010), just a wider denylist here.
func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		var loggedBody string
		if r.Method == http.MethodPost {
			raw, err := io.ReadAll(r.Body)
			if err == nil {
				r.Body = io.NopCloser(bytes.NewReader(raw))
				loggedBody = redactJSONBody(raw)
			}
		}

		defer func() {
			if err := recover(); err != nil {
				s.logger.Error("panic while handling request",
					"method", r.Method, "path", r.URL.Path, "panic", err)
				writeError(w, http.StatusInternalServerError, "errore interno")
			}
		}()

		next.ServeHTTP(rec, r)

		attrs := []any{
			"method", r.Method, "path", r.URL.Path,
			"status", rec.status, "duration_ms", time.Since(start).Milliseconds(),
		}
		if loggedBody != "" {
			attrs = append(attrs, "body", loggedBody)
		}
		s.logger.Info("request handled", attrs...)
	})
}

// redactJSONBody returns raw parsed as generic JSON with any piiJSONFields
// values replaced, for safe logging. Never returns raw bytes on a parse
// failure — logging unparsed input could leak PII we failed to recognize.
func redactJSONBody(raw []byte) string {
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "<unparsable body>"
	}
	for key := range parsed {
		if piiJSONFields[key] {
			parsed[key] = "[redacted]"
		}
	}
	redacted, err := json.Marshal(parsed)
	if err != nil {
		return "<unloggable body>"
	}
	return string(redacted)
}

// @Summary	Liveness check
// @Tags		system
// @Success	200	"Service healthy"
// @Failure	503	"Database unreachable"
// @Router		/healthz [get]
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if err := s.repo.Ping(r.Context()); err != nil {
		s.logger.Error("healthz: database unreachable", "error", err)
		writeError(w, http.StatusServiceUnavailable, "db non raggiungibile")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
