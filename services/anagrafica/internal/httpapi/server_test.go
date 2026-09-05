package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/francescocerri/sanitas/services/anagrafica/internal/testdb"
	"github.com/francescocerri/sanitas/services/anagrafica/internal/user"
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

func newTestKeyPair(t *testing.T) *user.KeyPair {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	path := filepath.Join(t.TempDir(), "jwt_private_key.pem")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	defer f.Close()
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	if err := pem.Encode(f, block); err != nil {
		t.Fatalf("write key: %v", err)
	}
	kp, err := user.LoadKeyPair(path)
	if err != nil {
		t.Fatalf("LoadKeyPair: %v", err)
	}
	return kp
}

// newTestServer wires a real repository and a real (freshly generated) key
// pair to the shared test database — no mock/interface, same choice as
// services/turni (ADR-0010).
func newTestServer(t *testing.T) (*Server, *user.Repository) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), "TRUNCATE users, roles, user_roles, tokens CASCADE"); err != nil {
			t.Fatalf("truncate: %v", err)
		}
	})
	repo := user.NewRepository(testPool)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := NewServer(repo, newTestKeyPair(t), "http://localhost:5173", "http://localhost:5173/attiva", logger)
	return server, repo
}

func doJSON(t *testing.T, server *Server, method, path string, body any, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	server, _ := newTestServer(t)
	rec := doJSON(t, server, http.MethodGet, "/healthz", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestJWKS(t *testing.T) {
	server, _ := newTestServer(t)
	rec := doJSON(t, server, http.MethodGet, "/.well-known/jwks.json", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode jwks: %v", err)
	}
	if _, ok := body["keys"]; !ok {
		t.Fatalf("expected a keys field, got: %s", rec.Body.String())
	}
}

func TestLogin_CorrectAndWrongCredentials(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	ok := doJSON(t, server, http.MethodPost, "/v1/login", loginRequest{Identifier: "admin", Password: "supersegreta"}, "")
	if ok.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", ok.Code, ok.Body.String())
	}
	var resp authTokens
	if err := json.Unmarshal(ok.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected a non-empty token")
	}
	if resp.RefreshToken == "" {
		t.Fatal("expected a non-empty refresh_token")
	}

	wrong := doJSON(t, server, http.MethodPost, "/v1/login", loginRequest{Identifier: "admin", Password: "sbagliata"}, "")
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", wrong.Code)
	}
}

// requireAuth requires the standard "Bearer <token>" header — a bare token
// (no prefix) must be rejected, even though it's what Swagger UI's Authorize
// dialog would send if you paste the token without typing "Bearer " first.
func TestRequireAuth_RequiresBearerPrefix(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	token := login(t, server, "admin", "supersegreta")

	withPrefix := doJSON(t, server, http.MethodGet, "/v1/me", nil, token)
	if withPrefix.Code != http.StatusOK {
		t.Fatalf("expected 200 with the standard 'Bearer <token>' header, got %d: %s", withPrefix.Code, withPrefix.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a bare token without the 'Bearer ' prefix, got %d: %s", rec.Code, rec.Body.String())
	}
}

// loginTokens is like login but returns both tokens, for tests exercising
// refresh/logout — most tests only need the access token (see login below).
func loginTokens(t *testing.T, server *Server, identifier, password string) authTokens {
	t.Helper()
	rec := doJSON(t, server, http.MethodPost, "/v1/login", loginRequest{Identifier: identifier, Password: password}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d: %s", rec.Code, rec.Body.String())
	}
	var resp authTokens
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	return resp
}

func TestRefresh_RotatesTokenAndRejectsReuse(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	first := loginTokens(t, server, "admin", "supersegreta")

	refreshed := doJSON(t, server, http.MethodPost, "/v1/refresh", refreshRequest{RefreshToken: first.RefreshToken}, "")
	if refreshed.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", refreshed.Code, refreshed.Body.String())
	}
	var second authTokens
	if err := json.Unmarshal(refreshed.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if second.Token == "" || second.RefreshToken == "" {
		t.Fatal("expected both a new access token and a new refresh token")
	}
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("expected refresh to rotate the refresh token, got the same one back")
	}

	// The first refresh token was single use — reusing it must fail.
	reuse := doJSON(t, server, http.MethodPost, "/v1/refresh", refreshRequest{RefreshToken: first.RefreshToken}, "")
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on refresh token reuse, got %d", reuse.Code)
	}
}

func TestRefresh_UnknownToken(t *testing.T) {
	server, _ := newTestServer(t)
	rec := doJSON(t, server, http.MethodPost, "/v1/refresh", refreshRequest{RefreshToken: "not-a-real-token"}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogout_InvalidatesRefreshToken(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	tokens := loginTokens(t, server, "admin", "supersegreta")

	logout := doJSON(t, server, http.MethodPost, "/v1/logout", refreshRequest{RefreshToken: tokens.RefreshToken}, "")
	if logout.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", logout.Code, logout.Body.String())
	}

	afterLogout := doJSON(t, server, http.MethodPost, "/v1/refresh", refreshRequest{RefreshToken: tokens.RefreshToken}, "")
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("expected refresh to fail after logout, got %d", afterLogout.Code)
	}
}

// Caught by the same reasoning as TestActivateUserLogsTokenRedacted: a
// refresh token is a bearer credential, not personal data, but must still
// never appear in the clear in the access log.
func TestRefreshLogsTokenRedacted(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	tokens := loginTokens(t, server, "admin", "supersegreta")

	var logBuf bytes.Buffer
	server.logger = slog.New(slog.NewJSONHandler(&logBuf, nil))

	rec := doJSON(t, server, http.MethodPost, "/v1/refresh", refreshRequest{RefreshToken: tokens.RefreshToken}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	logged := logBuf.String()
	if strings.Contains(logged, tokens.RefreshToken) {
		t.Fatalf("refresh token leaked into the log: %s", logged)
	}
	if !strings.Contains(logged, "[redacted]") {
		t.Fatalf("expected the request-handled log line to contain the redacted body, got: %s", logged)
	}
}

func login(t *testing.T, server *Server, identifier, password string) string {
	t.Helper()
	rec := doJSON(t, server, http.MethodPost, "/v1/login", loginRequest{Identifier: identifier, Password: password}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	return resp["token"]
}

func TestCreateUser_RequiresAdmin(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if err := repo.UpsertRole(ctx, "president", "Presidente"); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}

	noAuth := doJSON(t, server, http.MethodPost, "/v1/users", createUserRequest{Email: "mario@example.org", Username: "mario"}, "")
	if noAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a token, got %d", noAuth.Code)
	}

	adminToken := login(t, server, "admin", "supersegreta")
	created := doJSON(t, server, http.MethodPost, "/v1/users",
		createUserRequest{Email: "mario@example.org", Username: "mario", Roles: []string{"president"}}, adminToken)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", created.Code, created.Body.String())
	}
	var resp createUserResponse
	if err := json.Unmarshal(created.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if resp.InviteURL == "" {
		t.Fatal("expected a non-empty invite_url")
	}
	if len(resp.Roles) != 1 || resp.Roles[0] != "president" {
		t.Fatalf("unexpected roles: %v", resp.Roles)
	}
}

func TestActivateUser_ThenLogin(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	adminToken := login(t, server, "admin", "supersegreta")

	created := doJSON(t, server, http.MethodPost, "/v1/users",
		createUserRequest{Email: "mario@example.org", Username: "mario"}, adminToken)
	var createResp createUserResponse
	if err := json.Unmarshal(created.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	parsed, err := url.Parse(createResp.InviteURL)
	if err != nil {
		t.Fatalf("parse invite url: %v", err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatal("expected a token query param in the invite url")
	}

	activate := doJSON(t, server, http.MethodPost, "/v1/users/activate",
		activateUserRequest{Token: token, Password: "nuovapassword"}, "")
	if activate.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", activate.Code, activate.Body.String())
	}

	// Reusing the same token must fail.
	reuse := doJSON(t, server, http.MethodPost, "/v1/users/activate",
		activateUserRequest{Token: token, Password: "altra"}, "")
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on token reuse, got %d", reuse.Code)
	}

	loginRec := doJSON(t, server, http.MethodPost, "/v1/login", loginRequest{Identifier: "mario", Password: "nuovapassword"}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected the newly activated user to log in, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
}

func TestChangePassword(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "vecchiapassword"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	token := login(t, server, "admin", "vecchiapassword")

	change := doJSON(t, server, http.MethodPost, "/v1/password/change",
		changePasswordRequest{OldPassword: "vecchiapassword", NewPassword: "nuovapassword"}, token)
	if change.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", change.Code, change.Body.String())
	}

	oldFails := doJSON(t, server, http.MethodPost, "/v1/login", loginRequest{Identifier: "admin", Password: "vecchiapassword"}, "")
	if oldFails.Code != http.StatusUnauthorized {
		t.Fatalf("expected the old password to stop working, got %d", oldFails.Code)
	}
	newWorks := doJSON(t, server, http.MethodPost, "/v1/login", loginRequest{Identifier: "admin", Password: "nuovapassword"}, "")
	if newWorks.Code != http.StatusOK {
		t.Fatalf("expected the new password to work, got %d", newWorks.Code)
	}
}

func TestCreateUserLogsBodyWithPIIRedacted(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	adminToken := login(t, server, "admin", "supersegreta")

	var logBuf bytes.Buffer
	server.logger = slog.New(slog.NewJSONHandler(&logBuf, nil))

	created := doJSON(t, server, http.MethodPost, "/v1/users",
		createUserRequest{Email: "riservato@example.org", Username: "riservato"}, adminToken)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", created.Code, created.Body.String())
	}

	logged := logBuf.String()
	if strings.Contains(logged, "riservato@example.org") || strings.Contains(logged, "\"riservato\"") {
		t.Fatalf("PII leaked into the log: %s", logged)
	}
	if !strings.Contains(logged, "[redacted]") {
		t.Fatalf("expected the request-handled log line to contain the redacted body, got: %s", logged)
	}
}

// Caught live while manually verifying the service: the invite token isn't
// personal data, but it's a bearer credential — logging it in the clear
// would let whoever reads the logs activate the account first.
func TestActivateUserLogsTokenRedacted(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	adminToken := login(t, server, "admin", "supersegreta")

	created := doJSON(t, server, http.MethodPost, "/v1/users",
		createUserRequest{Email: "mario@example.org", Username: "mario"}, adminToken)
	var createResp createUserResponse
	if err := json.Unmarshal(created.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	parsed, err := url.Parse(createResp.InviteURL)
	if err != nil {
		t.Fatalf("parse invite url: %v", err)
	}
	rawToken := parsed.Query().Get("token")

	var logBuf bytes.Buffer
	server.logger = slog.New(slog.NewJSONHandler(&logBuf, nil))

	activate := doJSON(t, server, http.MethodPost, "/v1/users/activate",
		activateUserRequest{Token: rawToken, Password: "nuovapassword"}, "")
	if activate.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", activate.Code, activate.Body.String())
	}

	logged := logBuf.String()
	if strings.Contains(logged, rawToken) {
		t.Fatalf("invite token leaked into the log: %s", logged)
	}
	if !strings.Contains(logged, "[redacted]") {
		t.Fatalf("expected the request-handled log line to contain the redacted body, got: %s", logged)
	}
}
