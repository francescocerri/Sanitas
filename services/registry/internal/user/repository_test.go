package user

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/francescocerri/sanitas/services/registry/internal/testdb"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	ctx := context.Background()
	db, cleanup, err := testdb.StartPostgres(ctx, Migrate)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	testDB = db

	os.Exit(m.Run())
}

func newTestRepository(t *testing.T) *Repository {
	t.Helper()
	t.Cleanup(func() {
		if err := testDB.Exec("TRUNCATE users, roles, user_roles, tokens CASCADE").Error; err != nil {
			t.Fatalf("truncate: %v", err)
		}
	})
	return NewRepository(testDB)
}

func TestBootstrap_CreatesFirstAdminOnEmptyDB(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if err := Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	u, hash, err := repo.GetByLogin(ctx, "admin")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if !slices.Contains(u.Permissions, PermUsersManage) {
		t.Fatalf("expected the bootstrapped user to have %s, got permissions: %v", PermUsersManage, u.Permissions)
	}
	if !VerifyPassword(hash, "supersegreta") {
		t.Fatal("expected the bootstrapped password to verify")
	}
}

func TestBootstrap_NoOpWhenUsersExist(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if err := Bootstrap(ctx, repo, "admin@example.org", "admin", "supersegreta"); err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}
	// Second call must not fail nor create a second user, even with empty creds.
	if err := Bootstrap(ctx, repo, "", "", ""); err != nil {
		t.Fatalf("second Bootstrap should be a no-op, got: %v", err)
	}
	count, err := repo.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 user, got %d", count)
	}
}

func TestCreatePendingUser_RolesAssignedCorrectly(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if _, err := repo.UpsertRole(ctx, "president", "Presidente", nil); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}
	if _, err := repo.UpsertRole(ctx, "shift_manager", "Responsabile turni", nil); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}

	u, err := repo.CreatePendingUser(ctx, "mario@example.org", "mario")
	if err != nil {
		t.Fatalf("CreatePendingUser: %v", err)
	}

	ids, err := repo.RoleIDsBySlug(ctx, []string{"president", "shift_manager"})
	if err != nil {
		t.Fatalf("RoleIDsBySlug: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 role ids, got %d", len(ids))
	}
	if err := repo.AssignRoles(ctx, u.ID, []string{ids["president"], ids["shift_manager"]}); err != nil {
		t.Fatalf("AssignRoles: %v", err)
	}

	roles, err := repo.GetRolesForUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetRolesForUser: %v", err)
	}
	if len(roles) != 2 || roles[0] != "president" || roles[1] != "shift_manager" {
		t.Fatalf("unexpected roles: %v", roles)
	}
}

// A permission granted by more than one of a user's roles must still show
// up once — see docs/adr/0018.
func TestGetPermissionsForUser_DedupsAcrossOverlappingRoles(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	presidentID, err := repo.UpsertRole(ctx, "president", "Presidente", []string{PermUsersManage, PermShiftsRead})
	if err != nil {
		t.Fatalf("UpsertRole president: %v", err)
	}
	shiftManagerID, err := repo.UpsertRole(ctx, "shift_manager", "Responsabile turni", []string{PermShiftsRead, PermShiftsWrite})
	if err != nil {
		t.Fatalf("UpsertRole shift_manager: %v", err)
	}

	u, err := repo.CreatePendingUser(ctx, "mario@example.org", "mario")
	if err != nil {
		t.Fatalf("CreatePendingUser: %v", err)
	}
	if err := repo.AssignRoles(ctx, u.ID, []string{presidentID, shiftManagerID}); err != nil {
		t.Fatalf("AssignRoles: %v", err)
	}

	permissions, err := repo.GetPermissionsForUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetPermissionsForUser: %v", err)
	}
	want := []string{PermShiftsRead, PermShiftsWrite, PermUsersManage} // alphabetical, matches ORDER BY
	if len(permissions) != len(want) {
		t.Fatalf("expected %v (shifts:read deduped across both roles), got %v", want, permissions)
	}
	for i, p := range want {
		if permissions[i] != p {
			t.Fatalf("expected %v, got %v", want, permissions)
		}
	}
}

func TestInviteToken_ActivateAndConsumeOnce(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	u, err := repo.CreatePendingUser(ctx, "mario@example.org", "mario")
	if err != nil {
		t.Fatalf("CreatePendingUser: %v", err)
	}
	raw, err := repo.CreateToken(ctx, u.ID, "invite", time.Hour)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	gotUserID, err := repo.ConsumeToken(ctx, raw, "invite")
	if err != nil {
		t.Fatalf("ConsumeToken: %v", err)
	}
	if gotUserID != u.ID {
		t.Fatalf("expected user id %s, got %s", u.ID, gotUserID)
	}

	// A token can only ever be redeemed once.
	if _, err := repo.ConsumeToken(ctx, raw, "invite"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on reuse, got %v", err)
	}
}

func TestCreateActiveUser_DuplicateEmail(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if _, err := repo.CreateActiveUser(ctx, "dup@example.org", "user1", "hash"); err != nil {
		t.Fatalf("first CreateActiveUser: %v", err)
	}
	_, err := repo.CreateActiveUser(ctx, "dup@example.org", "user2", "hash")
	if !errors.Is(err, ErrDuplicateUser) {
		t.Fatalf("expected ErrDuplicateUser, got %v", err)
	}
}
