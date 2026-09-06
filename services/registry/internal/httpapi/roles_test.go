package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/francescocerri/sanitas/services/registry/internal/user"
)

func TestListRoles(t *testing.T) {
	server, repo := newTestServer(t)
	ctx := context.Background()
	managerToken, err := server.keys.IssueToken(user.User{ID: "test-manager", Permissions: []string{user.PermUsersManage}})
	if err != nil {
		t.Fatal(err)
	}
	readerToken, err := server.keys.IssueToken(user.User{ID: "test-reader", Permissions: []string{user.PermShiftsRead}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, token string
		status      int
	}{
		{"anonymous", "", http.StatusUnauthorized},
		{"invalid token", "invalid", http.StatusUnauthorized},
		{"without permission", readerToken, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := doJSON(t, server, http.MethodGet, "/v1/roles", nil, tc.token)
			if response.Code != tc.status {
				t.Fatalf("expected %d, got %d", tc.status, response.Code)
			}
		})
	}
	empty := doJSON(t, server, http.MethodGet, "/v1/roles", nil, managerToken)
	if empty.Code != http.StatusOK || strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Fatalf("expected empty array, got %d: %s", empty.Code, empty.Body.String())
	}
	for _, role := range []struct{ slug, name string }{{"custom_z", "Zeta"}, {"custom_b", "Alpha"}, {"custom_a", "Alpha"}} {
		if _, err := repo.UpsertRole(ctx, role.slug, role.name, nil); err != nil {
			t.Fatal(err)
		}
	}
	response := doJSON(t, server, http.MethodGet, "/v1/roles", nil, managerToken)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var roles []user.Role
	if err := json.Unmarshal(response.Body.Bytes(), &roles); err != nil {
		t.Fatal(err)
	}
	if len(roles) != 3 {
		t.Fatalf("expected 3 roles, got %v", roles)
	}
	for i, slug := range []string{"custom_a", "custom_b", "custom_z"} {
		if roles[i].Slug != slug || roles[i].ID == "" || roles[i].DisplayName == "" {
			t.Fatalf("unexpected role at %d: %+v", i, roles[i])
		}
	}
}
