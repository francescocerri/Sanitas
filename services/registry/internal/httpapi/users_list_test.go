package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/francescocerri/sanitas/services/registry/internal/user"
)

func TestListUsers_RequiresPermission(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if _, err := repo.UpsertRole(ctx, "base_volunteer", "Volontario base", nil); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}
	adminToken := login(t, server, "admin", "supersegreta")

	noAuth := doJSON(t, server, http.MethodGet, "/v1/users", nil, "")
	if noAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a token, got %d", noAuth.Code)
	}

	created := doJSON(t, server, http.MethodPost, "/v1/users",
		createUserRequest{Email: "luca@example.org", Username: "luca", Roles: []string{"base_volunteer"}}, adminToken)
	var createResp createUserResponse
	if err := json.Unmarshal(created.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	lucaToken := activateAndLogin(t, server, createResp.InviteURL, "luca", "nuovapassword")

	denied := doJSON(t, server, http.MethodGet, "/v1/users", nil, lucaToken)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a role without users:manage, got %d: %s", denied.Code, denied.Body.String())
	}

	granted := doJSON(t, server, http.MethodGet, "/v1/users", nil, adminToken)
	if granted.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", granted.Code, granted.Body.String())
	}
}

func TestListUsers_ReturnsUsersWithRoles(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if _, err := repo.UpsertRole(ctx, "president", "Presidente", nil); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}
	adminToken := login(t, server, "admin", "supersegreta")

	doJSON(t, server, http.MethodPost, "/v1/users",
		createUserRequest{Email: "mario@example.org", Username: "mario", Roles: []string{"president"}}, adminToken)

	rec := doJSON(t, server, http.MethodGet, "/v1/users", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var users []user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	var mario *user.User
	for i := range users {
		if users[i].Username == "mario" {
			mario = &users[i]
		}
	}
	if mario == nil {
		t.Fatal("expected mario in the list")
	}
	if len(mario.Roles) != 1 || mario.Roles[0] != "president" {
		t.Fatalf("unexpected roles for mario: %v", mario.Roles)
	}
}

func TestUpdateUserRoles_ReplacesRoles(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if _, err := repo.UpsertRole(ctx, "president", "Presidente", nil); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}
	if _, err := repo.UpsertRole(ctx, "base_volunteer", "Volontario base", nil); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}
	adminToken := login(t, server, "admin", "supersegreta")

	created := doJSON(t, server, http.MethodPost, "/v1/users",
		createUserRequest{Email: "mario@example.org", Username: "mario", Roles: []string{"president"}}, adminToken)
	var createResp createUserResponse
	if err := json.Unmarshal(created.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	rec := doJSON(t, server, http.MethodPatch, "/v1/users/"+createResp.ID+"/roles",
		updateUserRolesRequest{Roles: []string{"base_volunteer"}}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if len(updated.Roles) != 1 || updated.Roles[0] != "base_volunteer" {
		t.Fatalf("expected roles to be replaced, got %v", updated.Roles)
	}
}

func TestUpdateUserRoles_RejectsUnknownRoleSlug(t *testing.T) {
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

	rec := doJSON(t, server, http.MethodPatch, "/v1/users/"+createResp.ID+"/roles",
		updateUserRolesRequest{Roles: []string{"non-esiste"}}, adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateUserRoles_UnknownUserID(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	adminToken := login(t, server, "admin", "supersegreta")

	rec := doJSON(t, server, http.MethodPatch, "/v1/users/00000000-0000-0000-0000-000000000000/roles",
		updateUserRolesRequest{Roles: []string{}}, adminToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateUserRoles_RequiresPermission(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	if err := user.Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if _, err := repo.UpsertRole(ctx, "base_volunteer", "Volontario base", nil); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}
	adminToken := login(t, server, "admin", "supersegreta")

	created := doJSON(t, server, http.MethodPost, "/v1/users",
		createUserRequest{Email: "luca@example.org", Username: "luca", Roles: []string{"base_volunteer"}}, adminToken)
	var createResp createUserResponse
	if err := json.Unmarshal(created.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	lucaToken := activateAndLogin(t, server, createResp.InviteURL, "luca", "nuovapassword")

	rec := doJSON(t, server, http.MethodPatch, "/v1/users/"+createResp.ID+"/roles",
		updateUserRolesRequest{Roles: []string{}}, lucaToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}
