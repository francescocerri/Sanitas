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
	"gorm.io/gorm"

	"github.com/francescocerri/sanitas/services/shifts/internal/authclient"
	"github.com/francescocerri/sanitas/services/shifts/internal/shift"
	"github.com/francescocerri/sanitas/services/shifts/internal/testdb"
)

var testDB *gorm.DB

// testVolunteerID is a real registry.users row seeded by testdb.StartPostgres
// — bookings.volunteer_id is an FK, and handleCreateBooking now writes
// claims.Subject there, so tests creating a booking need a token whose
// Subject is an id that actually exists (see tokenFor).
var testVolunteerID string

func TestMain(m *testing.M) {
	ctx := context.Background()
	db, volunteerID, cleanup, err := testdb.StartPostgres(ctx, shift.Migrate)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	testDB = db
	testVolunteerID = volunteerID

	os.Exit(m.Run())
}

// testIssuer stands in for registry in tests: shifts never issues tokens
// itself, only verifies ones signed by registry's key — see
// internal/authclient. Serves a JWKS document and signs tokens with the
// matching private key, so tests don't need the real (separate-module)
// registry service running.
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

// bigEndianExponent mirrors registry's user.jwt.go helper of the same
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

// token signs a JWT carrying the given permissions — the shape
// requirePermission actually checks (see docs/adr/0018). Subject is a
// fixed placeholder, fine for every test that doesn't write it anywhere
// (most of them); tests that do (e.g. creating a booking, where
// claims.Subject becomes volunteer_id — a real FK) need tokenFor instead.
func (iss *testIssuer) token(t *testing.T, permissions []string) string {
	t.Helper()
	return iss.tokenFor(t, "test-user", permissions)
}

// tokenFor is token but with a caller-chosen Subject — needed wherever the
// claims' subject ends up written to the database (see comment on token).
func (iss *testIssuer) tokenFor(t *testing.T, subject string, permissions []string) string {
	t.Helper()
	claims := authclient.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Username:    subject,
		Permissions: permissions,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = iss.kid
	signed, err := token.SignedString(iss.priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// newTestServerWithIssuer wires a real shift.Repository to the shared test
// database — no mock/interface, consistent with ADR-0010 (no layer
// introduced until the domain model needs one) — plus a test JWKS issuer so
// requireAuth/requirePermission have something real to verify against.
func newTestServerWithIssuer(t *testing.T) (*Server, *testIssuer) {
	t.Helper()
	t.Cleanup(func() {
		if err := testDB.Exec("TRUNCATE bookings, shift_templates").Error; err != nil {
			t.Fatalf("truncate: %v", err)
		}
	})
	repo := shift.NewRepository(testDB)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	issuer := newTestIssuer(t)
	authClient := authclient.New(issuer.server.URL)
	if err := authClient.Refresh(context.Background()); err != nil {
		t.Fatalf("authClient.Refresh: %v", err)
	}

	return NewServer(repo, authClient, "http://localhost:5173", logger), issuer
}

func newTestServer(t *testing.T) *Server {
	server, _ := newTestServerWithIssuer(t)
	return server
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

// requireAuth/requirePermission have no business route to exercise them
// against right now (see docs/adr/0025-modello-dati-turni.md — the old
// /v1/shifts* routes are gone, the new ones return in later backlog items),
// so these tests mount the middleware directly over a stub handler instead
// of going through Routes(). The middleware itself doesn't change with the
// domain model, so this coverage stays meaningful across the gap.
func TestRequireAuth(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /protected", server.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler := server.withLogging(server.withCORS(mux))

	noAuth := httptest.NewRequest(http.MethodGet, "/protected", nil)
	noAuthRec := httptest.NewRecorder()
	handler.ServeHTTP(noAuthRec, noAuth)
	if noAuthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no Authorization header, got %d: %s", noAuthRec.Code, noAuthRec.Body.String())
	}

	invalid := httptest.NewRequest(http.MethodGet, "/protected", nil)
	invalid.Header.Set("Authorization", "Bearer not-a-real-token")
	invalidRec := httptest.NewRecorder()
	handler.ServeHTTP(invalidRec, invalid)
	if invalidRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with an invalid token, got %d: %s", invalidRec.Code, invalidRec.Body.String())
	}

	valid := httptest.NewRequest(http.MethodGet, "/protected", nil)
	valid.Header.Set("Authorization", "Bearer "+issuer.token(t, nil))
	validRec := httptest.NewRecorder()
	handler.ServeHTTP(validRec, valid)
	if validRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with a valid token, got %d: %s", validRec.Code, validRec.Body.String())
	}
}

// A valid token isn't enough on its own — the right permission must be
// among its claims, checked per action (see docs/adr/0018).
func TestRequirePermission(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /protected", server.requirePermission("some:permission", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler := server.withLogging(server.withCORS(mux))

	noPermissions := issuer.token(t, nil)
	noPermReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	noPermReq.Header.Set("Authorization", "Bearer "+noPermissions)
	noPermRec := httptest.NewRecorder()
	handler.ServeHTTP(noPermRec, noPermReq)
	if noPermRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with no permissions, got %d: %s", noPermRec.Code, noPermRec.Body.String())
	}

	withPermission := issuer.token(t, []string{"some:permission"})
	withPermReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	withPermReq.Header.Set("Authorization", "Bearer "+withPermission)
	withPermRec := httptest.NewRecorder()
	handler.ServeHTTP(withPermRec, withPermReq)
	if withPermRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with the right permission, got %d: %s", withPermRec.Code, withPermRec.Body.String())
	}
}

func TestCreateAndListShiftTemplates(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	configureToken := issuer.token(t, []string{permShiftsConfigure})
	readToken := issuer.token(t, []string{permShiftsRead})

	body, _ := json.Marshal(createTemplateRequest{Weekday: 4, StartTime: "20:00", EndTime: "08:00", Label: "Turno Serale"})
	createReq := httptest.NewRequest(http.MethodPost, "/v1/shift-templates", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+configureToken)
	createRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created shift.ShiftTemplate
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected a non-empty id")
	}
	if !created.Active {
		t.Fatal("expected a newly created template to be active")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/shift-templates", nil)
	listReq.Header.Set("Authorization", "Bearer "+readToken)
	listRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var listed []shift.ShiftTemplate
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("expected the created template in the list, got %+v", listed)
	}
}

func TestCreateShiftTemplate_RequiresConfigurePermission(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	body, _ := json.Marshal(createTemplateRequest{Weekday: 4, StartTime: "20:00", EndTime: "08:00", Label: "x"})

	readOnly := issuer.token(t, []string{permShiftsRead})
	req := httptest.NewRequest(http.MethodPost, "/v1/shift-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+readOnly)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with only shifts:read, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListShiftTemplates_RequiresReadPermission(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/shift-templates", nil)
	req.Header.Set("Authorization", "Bearer "+issuer.token(t, nil))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with no permissions, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateShiftTemplate_ValidatesFields(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	token := issuer.token(t, []string{permShiftsConfigure})

	for _, tc := range []struct {
		name string
		req  createTemplateRequest
	}{
		{"weekday too low", createTemplateRequest{Weekday: -1, StartTime: "08:00", EndTime: "14:00", Label: "x"}},
		{"weekday too high", createTemplateRequest{Weekday: 7, StartTime: "08:00", EndTime: "14:00", Label: "x"}},
		{"malformed start_time", createTemplateRequest{Weekday: 1, StartTime: "8:00", EndTime: "14:00", Label: "x"}},
		{"malformed end_time", createTemplateRequest{Weekday: 1, StartTime: "08:00", EndTime: "24:00", Label: "x"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/v1/shift-templates", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			server.Routes().ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestUpdateShiftTemplate(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	token := issuer.token(t, []string{permShiftsConfigure})

	createBody, _ := json.Marshal(createTemplateRequest{Weekday: 6, StartTime: "08:00", EndTime: "14:00", Label: "Turno Mattina"})
	createReq := httptest.NewRequest(http.MethodPost, "/v1/shift-templates", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRec, createReq)
	var created shift.ShiftTemplate
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	updateBody, _ := json.Marshal(updateTemplateRequest{Weekday: 0, StartTime: "09:00", EndTime: "13:00", Label: "Modificato", Active: false})
	updateReq := httptest.NewRequest(http.MethodPatch, "/v1/shift-templates/"+created.ID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	var updated shift.ShiftTemplate
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updated.Active {
		t.Fatal("expected Active to be false after the update")
	}
	if updated.Label != "Modificato" {
		t.Fatalf("expected the label to be updated, got %q", updated.Label)
	}
}

func TestUpdateShiftTemplate_NotFound(t *testing.T) {
	server, issuer := newTestServerWithIssuer(t)
	token := issuer.token(t, []string{permShiftsConfigure})

	body, _ := json.Marshal(updateTemplateRequest{Weekday: 1, StartTime: "08:00", EndTime: "14:00", Label: "x", Active: true})
	req := httptest.NewRequest(http.MethodPatch, "/v1/shift-templates/00000000-0000-0000-0000-000000000000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
