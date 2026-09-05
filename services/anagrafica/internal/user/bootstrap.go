package user

import (
	"context"
	"fmt"
)

// Bootstrap creates the first admin account from config-provided
// credentials, but only if the users table is empty. There is no public
// setup endpoint (see docs/adr/0013) — once at least one user exists this
// is a no-op, regardless of what INITIAL_ADMIN_* is set to.
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
	_, err = repo.CreateActiveUser(ctx, email, username, hash, true)
	return err
}
