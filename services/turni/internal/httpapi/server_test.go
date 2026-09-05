package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/francescocerri/sanitas/services/turni/internal/authclient"
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

// testIssuer stands in for anagrafica in tests: turni never issues tokens
// itself, only verifies ones signed by anagrafica's key — see
// internal/authclient. Serves a JWKS document and signs tokens with the
// matching private key, so tests don't need the real (separate-module)
// anagrafica service running.
type testIssuer struct {
	server *httptest.Server
	priv   *rsa.PrivateKey
	kid    string
}

func newTestIssuer(t *testing.T) *testIssuer {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	iss := &testIssuer{priv: priv, kid: "test-kid"}
	iss.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pub := priv.PublicKey
		body := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"use": "sig",
					"alg": "RS256",
					"kid": iss.kid,
					"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(bigEndianExponent(pub.E)),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(iss.server.Close)
	return iss
}

// bigEndianExponent mirrors anagrafica's user.jwt.go helper of the same
// name (and internal/authclient's test copy) — needed only to build a
// realistic test JWKS document.
func bigEndianExponent(e int) []byte {
	b := []byte{byte(e >> 16), byte(e >> 8), byte(e)}
	i := 0
	for i < len(b)-1 && b[i] == 0 {
		i++
	}
	return b[i:]
}

func (iss *testIssuer) token(t *testing.T) string {
	t.Helper()
	claims := authclient.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Username: "test-user",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = iss.kid
	signed, err := token.SignedString(iss.priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// newTestServer wires a real turno.Repository to the shared test database —
// no mock/interface, consistent with ADR-0010 (no layer introduced until
// the domain model needs one) — plus a test JWKS issuer so requireAuth has
// something real to verify against. Returns a ready-to-use valid bearer
// token: authorization (roles/is_admin) isn't checked in turni yet, so every
// test needing a logged-in caller can share the same one.
func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), "TRUNCATE turni"); err != nil {
			t.Fatalf("truncate turni: %v", err)
		}
	})
	repo := turno.NewRepository(testPool)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	issuer := newTestIssuer(t)
	authClient := authclient.New(issuer.server.URL)
	if err := authClient.Refresh(context.Background()); err != nil {
		t.Fatalf("authClient.Refresh: %v", err)
	}

	server := NewServer(repo, authClient, "http://localhost:5173", logger)
	return server, issuer.token(t)
}

func TestHealthz(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateAndGetTurno(t *testing.T) {
	server, token := newTestServer(t)

	body, _ := json.Marshal(turno.Turno{
		VolontarioID: testVolontarioID,
		Data:         "2026-09-10",
		OraInizio:    "08:00",
		OraFine:      "14:00",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/shifts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
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
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/shifts/{id}: expected 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
}

func TestGetTurnoNotFound(t *testing.T) {
	server, token := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/shifts/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
	server, token := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/shifts", bytes.NewReader([]byte("not json")))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestShifts_RequireAuth(t *testing.T) {
	server, token := newTestServer(t)

	noAuth := httptest.NewRequest(http.MethodGet, "/v1/shifts", nil)
	noAuthRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(noAuthRec, noAuth)
	if noAuthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no Authorization header, got %d: %s", noAuthRec.Code, noAuthRec.Body.String())
	}

	invalid := httptest.NewRequest(http.MethodGet, "/v1/shifts", nil)
	invalid.Header.Set("Authorization", "Bearer not-a-real-token")
	invalidRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(invalidRec, invalid)
	if invalidRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with an invalid token, got %d: %s", invalidRec.Code, invalidRec.Body.String())
	}

	valid := httptest.NewRequest(http.MethodGet, "/v1/shifts", nil)
	valid.Header.Set("Authorization", "Bearer "+token)
	validRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(validRec, valid)
	if validRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with a valid token, got %d: %s", validRec.Code, validRec.Body.String())
	}
}

func TestCreateTurnoLogsBodyWithPIIRedacted(t *testing.T) {
	server, token := newTestServer(t)

	var logBuf bytes.Buffer
	server.logger = slog.New(slog.NewJSONHandler(&logBuf, nil))

	body, _ := json.Marshal(turno.Turno{
		VolontarioID: testVolontarioID,
		Data:         "2026-09-10",
		OraInizio:    "08:00",
		OraFine:      "14:00",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/shifts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
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
