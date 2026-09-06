package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/francescocerri/sanitas/services/registry/internal/user"
)

func TestRequestPasswordReset_SendsEmailForKnownUser(t *testing.T) {
	server, repo := newTestServerWithMailer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	rec := doJSON(t, server, http.MethodPost, "/v1/password/reset/request",
		requestPasswordResetRequest{Identifier: "admin"}, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	delivered, err := testMail.HasMessageTo(ctx, "admin@example.org")
	if err != nil {
		t.Fatalf("query mailpit: %v", err)
	}
	if !delivered {
		t.Fatal("expected mailpit to have received a password reset email addressed to admin@example.org")
	}
}

// Stessa identica risposta (204) di un identifier trovato: mai rivelare se
// un account esiste, vedi docs/adr/0024-recupero-password.md.
func TestRequestPasswordReset_NoOpForUnknownIdentifier(t *testing.T) {
	server, _ := newTestServer(t)

	rec := doJSON(t, server, http.MethodPost, "/v1/password/reset/request",
		requestPasswordResetRequest{Identifier: "nessuno"}, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 even for an unknown identifier, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestConfirmPasswordReset_SetsNewPasswordAndAllowsLogin(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	u, _, err := repo.GetByLogin(ctx, "admin")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}

	token, err := repo.CreateToken(ctx, u.ID, "password_reset", passwordResetTokenTTL)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	confirm := doJSON(t, server, http.MethodPost, "/v1/password/reset/confirm",
		confirmPasswordResetRequest{Token: token, Password: "nuovapassword"}, "")
	if confirm.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", confirm.Code, confirm.Body.String())
	}

	// Reusing the same token must fail.
	reuse := doJSON(t, server, http.MethodPost, "/v1/password/reset/confirm",
		confirmPasswordResetRequest{Token: token, Password: "altra"}, "")
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on token reuse, got %d", reuse.Code)
	}

	loginRec := doJSON(t, server, http.MethodPost, "/v1/login",
		loginRequest{Identifier: "admin", Password: "nuovapassword"}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login with the new password to succeed, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
}

func TestConfirmPasswordReset_RejectsInvalidToken(t *testing.T) {
	server, _ := newTestServer(t)

	rec := doJSON(t, server, http.MethodPost, "/v1/password/reset/confirm",
		confirmPasswordResetRequest{Token: "non-esiste", Password: "qualcosa"}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
