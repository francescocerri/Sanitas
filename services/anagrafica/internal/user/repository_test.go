package user

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/francescocerri/sanitas/services/anagrafica/internal/testdb"
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

func newTestRepository(t *testing.T) *Repository {
	t.Helper()
	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), "TRUNCATE users, roles, user_roles, invite_tokens CASCADE"); err != nil {
			t.Fatalf("truncate: %v", err)
		}
	})
	return NewRepository(testPool)
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
	if !u.IsAdmin {
		t.Fatal("expected the bootstrapped user to be an admin")
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

	if err := repo.UpsertRole(ctx, "president", "Presidente"); err != nil {
		t.Fatalf("UpsertRole: %v", err)
	}
	if err := repo.UpsertRole(ctx, "shift_manager", "Responsabile turni"); err != nil {
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

func TestInviteToken_ActivateAndConsumeOnce(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	u, err := repo.CreatePendingUser(ctx, "mario@example.org", "mario")
	if err != nil {
		t.Fatalf("CreatePendingUser: %v", err)
	}
	raw, err := repo.CreateInviteToken(ctx, u.ID, "invite", time.Hour)
	if err != nil {
		t.Fatalf("CreateInviteToken: %v", err)
	}

	gotUserID, err := repo.ConsumeInviteToken(ctx, raw, "invite")
	if err != nil {
		t.Fatalf("ConsumeInviteToken: %v", err)
	}
	if gotUserID != u.ID {
		t.Fatalf("expected user id %s, got %s", u.ID, gotUserID)
	}

	// A token can only ever be redeemed once.
	if _, err := repo.ConsumeInviteToken(ctx, raw, "invite"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on reuse, got %v", err)
	}
}

func TestCreateActiveUser_DuplicateEmail(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if _, err := repo.CreateActiveUser(ctx, "dup@example.org", "user1", "hash", false); err != nil {
		t.Fatalf("first CreateActiveUser: %v", err)
	}
	_, err := repo.CreateActiveUser(ctx, "dup@example.org", "user2", "hash", false)
	if !errors.Is(err, ErrDuplicateUser) {
		t.Fatalf("expected ErrDuplicateUser, got %v", err)
	}
}
