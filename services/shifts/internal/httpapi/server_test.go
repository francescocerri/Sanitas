package httpapi

import (
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

func TestMain(m *testing.M) {
	ctx := context.Background()
	db, _, cleanup, err := testdb.StartPostgres(ctx, shift.Migrate)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	testDB = db

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
// requirePermission actually checks (see docs/adr/0018).
func (iss *testIssuer) token(t *testing.T, permissions []string) string {
	t.Helper()
	claims := authclient.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Username:    "test-user",
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
