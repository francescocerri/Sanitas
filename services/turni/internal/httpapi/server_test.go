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

// testVolontarioID is a real anagrafica.users row seeded by testdb.StartPostgres
// — turni.turni.volontario_id is now an FK, so tests need an existing user
// to reference instead of an arbitrary placeholder string.
var testVolontarioID string

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, volontarioID, cleanup, err := testdb.StartPostgres(ctx)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	testPool = pool
	testVolontarioID = volontarioID

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
		VolontarioID: testVolontarioID,
		Data:         "2026-09-10",
		OraInizio:    "08:00",
		OraFine:      "14:00",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/shifts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /v1/shifts: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created turno.Turno
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected a non-empty id in the create response")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/shifts/"+created.ID, nil)
	getRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/shifts/{id}: expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
}

func TestGetTurnoNotFound(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/shifts/00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body["error"] != "shift not found" {
		t.Fatalf("unexpected error body: %v", body)
	}
}

func TestCreateTurnoInvalidPayload(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/shifts", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateTurnoLogsBodyWithPIIRedacted(t *testing.T) {
	server := newTestServer(t)

	var logBuf bytes.Buffer
	server.logger = slog.New(slog.NewJSONHandler(&logBuf, nil))

	body, _ := json.Marshal(turno.Turno{
		VolontarioID: testVolontarioID,
		Data:         "2026-09-10",
		OraInizio:    "08:00",
		OraFine:      "14:00",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/shifts", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	logged := logBuf.String()
	if bytes.Contains(logBuf.Bytes(), []byte(testVolontarioID)) {
		t.Fatalf("PII leaked into the log: %s", logged)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("[redacted]")) {
		t.Fatalf("expected the request-handled log line to contain the redacted body, got: %s", logged)
	}
}
