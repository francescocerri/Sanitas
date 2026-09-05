package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/francescocerri/sanitas/services/turni/internal/turno"
)

type Server struct {
	repo          *turno.Repository
	allowedOrigin string
	logger        *slog.Logger
}

func NewServer(repo *turno.Repository, allowedOrigin string, logger *slog.Logger) *Server {
	return &Server{repo: repo, allowedOrigin: allowedOrigin, logger: logger}
}

// v1 versions the resource endpoints only: /healthz and /docs/ are
// operational/meta, not part of the API contract that evolves, so they
// stay unversioned (see docs/adr/0010 — no global @BasePath in the swag
// annotations, each @Router spells out its real full path instead).
const v1 = "/v1"

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET "+v1+"/shifts", s.handleListTurni)
	mux.HandleFunc("POST "+v1+"/shifts", s.handleCreateTurno)
	mux.HandleFunc("GET "+v1+"/shifts/{id}", s.handleGetTurno)
	mux.Handle("GET /docs/", docsHandler())
	return s.withLogging(s.withCORS(mux))
}

// CORS is deliberately minimal (one configurable origin, GET/POST only): the
// only client today is the web/ frontend, and there is no cookie/credential
// use yet that would require a stricter policy.
func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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

// withLogging combines structured access logging and panic recovery —
// standard Go pattern: without recovery here, a panic in a handler would
// close the connection with no response and no log to diagnose it from.
// For POST requests it also logs the body (PII redacted, see
// redactJSONBody) for audit/debugging — reading it here and putting it
// back via io.NopCloser so the handler can still decode it normally.
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
				writeError(w, http.StatusInternalServerError, "internal error")
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

// piiJSONFields lists the JSON field names this service's request bodies
// may contain that identify a person — today just volontario_id (a text
// placeholder until the anagrafica service exists, but it stands in for a
// real person). Extend this list as new PII-bearing fields are added.
var piiJSONFields = map[string]bool{
	"volontario_id": true,
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
		writeError(w, http.StatusServiceUnavailable, "database unreachable")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// @Summary	List turni
// @Tags		turni
// @Produce	json
// @Success	200	{array}	turno.Turno
// @Router		/v1/shifts [get]
func (s *Server) handleListTurni(w http.ResponseWriter, r *http.Request) {
	turni, err := s.repo.List(r.Context())
	if err != nil {
		s.logger.Error("list turni", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, turni)
}

// @Summary	Create a new turno
// @Tags		turni
// @Accept		json
// @Produce	json
// @Param		turno	body		turno.Turno	true	"New turno (id and stato in the input are ignored)"
// @Success	201		{object}	turno.Turno
// @Failure	400		"Invalid payload"
// @Router		/v1/shifts [post]
func (s *Server) handleCreateTurno(w http.ResponseWriter, r *http.Request) {
	var input turno.Turno
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := s.repo.Create(r.Context(), input)
	if err != nil {
		s.logger.Error("create turno", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// @Summary	Get a turno by id
// @Tags		turni
// @Produce	json
// @Param		id	path		string	true	"Turno id (UUID)"
// @Success	200	{object}	turno.Turno
// @Failure	404	"Turno not found"
// @Router		/v1/shifts/{id} [get]
func (s *Server) handleGetTurno(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, turno.ErrNotFound) {
			writeError(w, http.StatusNotFound, "shift not found")
			return
		}
		s.logger.Error("get turno", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError wraps a client-facing message in the same JSON shape used
// everywhere else in this API — never the raw error passed to it, so an
// internal detail (a DB connection string, a query) can't leak to the caller.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
