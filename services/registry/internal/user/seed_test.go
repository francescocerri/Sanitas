package user

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeSeedFile(t *testing.T, seeds []roleSeed) string {
	t.Helper()
	raw, err := json.Marshal(seeds)
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "roles.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	return path
}

func TestSeedRoles_AppliesPermissions(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	path := writeSeedFile(t, []roleSeed{
		{Slug: "president", DisplayName: "Presidente", Permissions: []string{PermUsersManage, PermShiftsRead}},
		{Slug: "base_volunteer", DisplayName: "Volontario base"}, // no permissions field: must not crash (see UpsertRole nil handling)
	})

	if err := SeedRoles(ctx, repo, path); err != nil {
		t.Fatalf("SeedRoles: %v", err)
	}

	u, err := repo.CreatePendingUser(ctx, "mario@example.org", "mario")
	if err != nil {
		t.Fatalf("CreatePendingUser: %v", err)
	}
	ids, err := repo.RoleIDsBySlug(ctx, []string{"president"})
	if err != nil {
		t.Fatalf("RoleIDsBySlug: %v", err)
	}
	if err := repo.AssignRoles(ctx, u.ID, []string{ids["president"]}); err != nil {
		t.Fatalf("AssignRoles: %v", err)
	}

	permissions, err := repo.GetPermissionsForUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetPermissionsForUser: %v", err)
	}
	if len(permissions) != 2 {
		t.Fatalf("expected 2 permissions for president, got %v", permissions)
	}
}

// An unknown permission slug in the seed file is almost certainly a typo —
// SeedRoles must reject it rather than silently seeding a role that grants
// nothing (or ignoring the typo) — see docs/adr/0018.
func TestSeedRoles_RejectsUnknownPermission(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	path := writeSeedFile(t, []roleSeed{
		{Slug: "president", DisplayName: "Presidente", Permissions: []string{"users:mange"}}, // typo
	})

	if err := SeedRoles(ctx, repo, path); err == nil {
		t.Fatal("expected SeedRoles to reject an unknown permission slug")
	}
}
