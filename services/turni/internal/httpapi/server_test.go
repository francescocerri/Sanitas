package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/francescocerri/sanitas/services/turni/internal/testdb"
	"github.com/francescocerri/sanitas/services/turni/internal/turno"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, cleanup, err := testdb.StartPostgres(ctx)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	testPool = pool

	os.Exit(m.Run())
}

// newTestServer wires a real turno.Repository to the shared test database —
// no mock/interface, consistent with ADR-0010 (no layer introduced until
// the domain model needs one).
func newTestServer(t *testing.T) *Server {
	t.Helper()
	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), "TRUNCATE turni"); err != nil {
			t.Fatalf("truncate turni: %v", err)
		}
	})
	repo := turno.NewRepository(testPool)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(repo, "http://localhost:5173", logger)
}

func TestHealthz(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateAndGetTurno(t *testing.T) {
	server := newTestServer(t)

	body, _ := json.Marshal(turno.Turno{
		VolontarioID: "v1",
		Data:         "2026-09-10",
		OraInizio:    "08:00",
		OraFine:      "14:00",
	})
	req := httptest.NewRequest(http.MethodPost, "/turni", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /turni: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created turno.Turno
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected a non-empty id in the create response")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/turni/"+created.ID, nil)
	getRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /turni/{id}: expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
}

func TestGetTurnoNotFound(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/turni/00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body["error"] != "turno non trovato" {
		t.Fatalf("unexpected error body: %v", body)
	}
}

func TestCreateTurnoInvalidPayload(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/turni", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
