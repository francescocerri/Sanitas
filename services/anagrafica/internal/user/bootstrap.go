package user

import (
	"context"
	"fmt"
)

// bootstrapAdminRoleSlug is a technical role, not an organizational one —
// it never comes from config/<slug>/anagrafica/roles.json (see
// docs/adr/0012, which is about committee-specific role names) and isn't
// meant to be assigned to a real volunteer. It exists purely to solve the
// bootstrap chicken-and-egg problem: the first user has no roles yet to
// carry the permissions it needs, so it gets this one instead — see
// docs/adr/0018.
const bootstrapAdminRoleSlug = "bootstrap_admin"

// Bootstrap creates the first admin account from config-provided
// credentials, but only if the users table is empty. There is no public
// setup endpoint (see docs/adr/0013) — once at least one user exists this
// is a no-op, regardless of what INITIAL_ADMIN_* is set to.
//
// The new user is an ordinary account: its only distinguishing trait is
// being assigned the reserved bootstrapAdminRoleSlug role (created here,
// with every known permission), not a special flag on the user row — every
// authorization check in this codebase goes through roles/permissions,
// with no separate case for "is this the bootstrap user".
func Bootstrap(ctx context.Context, repo *Repository, email, username, password string) error {
	count, err := repo.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if email == "" || username == "" || password == "" {
		return fmt.Errorf("bootstrap: users table is empty but INITIAL_ADMIN_EMAIL/USERNAME/PASSWORD are not all set")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("bootstrap: hash password: %w", err)
	}
	u, err := repo.CreateActiveUser(ctx, email, username, hash)
	if err != nil {
		return err
	}
	roleID, err := repo.UpsertRole(ctx, bootstrapAdminRoleSlug, "Amministratore di bootstrap", AllPermissions)
	if err != nil {
		return fmt.Errorf("bootstrap: create technical admin role: %w", err)
	}
	if err := repo.AssignRoles(ctx, u.ID, []string{roleID}); err != nil {
		return fmt.Errorf("bootstrap: assign technical admin role: %w", err)
	}
	return nil
}
