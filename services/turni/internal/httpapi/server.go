package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/francescocerri/sanitas/services/turni/internal/turno"
)

type Server struct {
	repo          *turno.Repository
	allowedOrigin string
}

func NewServer(repo *turno.Repository, allowedOrigin string) *Server {
	return &Server{repo: repo, allowedOrigin: allowedOrigin}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /turni", s.handleListTurni)
	mux.HandleFunc("POST /turni", s.handleCreateTurno)
	mux.HandleFunc("GET /turni/{id}", s.handleGetTurno)
	mux.Handle("GET /docs/", docsHandler())
	return s.withCORS(mux)
}

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

// @Summary	Liveness check
// @Tags		sistema
// @Success	200	"Servizio operativo"
// @Failure	503	"Database non raggiungibile"
// @Router		/healthz [get]
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if err := s.repo.Ping(r.Context()); err != nil {
		http.Error(w, "db non raggiungibile", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// @Summary	Elenca i turni
// @Tags		turni
// @Produce	json
// @Success	200	{array}	turno.Turno
// @Router		/turni [get]
func (s *Server) handleListTurni(w http.ResponseWriter, r *http.Request) {
	turni, err := s.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, turni)
}

// @Summary	Crea un nuovo turno
// @Tags		turni
// @Accept		json
// @Produce	json
// @Param		turno	body		turno.Turno	true	"Nuovo turno (id e stato in input vengono ignorati)"
// @Success	201		{object}	turno.Turno
// @Failure	400		"Payload non valido"
// @Router		/turni [post]
func (s *Server) handleCreateTurno(w http.ResponseWriter, r *http.Request) {
	var input turno.Turno
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "payload non valido", http.StatusBadRequest)
		return
	}
	created, err := s.repo.Create(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// @Summary	Recupera un turno per id
// @Tags		turni
// @Produce	json
// @Param		id	path		string	true	"ID del turno (UUID)"
// @Success	200	{object}	turno.Turno
// @Failure	404	"Turno non trovato"
// @Router		/turni/{id} [get]
func (s *Server) handleGetTurno(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "turno non trovato", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
