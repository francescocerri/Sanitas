package user

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type roleSeed struct {
	Slug        string   `json:"slug"`
	DisplayName string   `json:"display_name"`
	Permissions []string `json:"permissions"`
}

// SeedRoles upserts every role listed in the file at path into the roles
// table. Roles (and which permissions each gets) are per-committee data,
// not a Go enum, precisely so a fork only has to replace this file — see
// docs/adr/0012. The permission slugs themselves are NOT per-committee
// (docs/adr/0018): an unknown one is almost certainly a typo, rejected
// here rather than silently granting nothing — same fail-fast principle
// already used for DATABASE_URL/the JWT key.
func SeedRoles(ctx context.Context, repo *Repository, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("seed roles: read %s: %w", path, err)
	}
	var seeds []roleSeed
	if err := json.Unmarshal(raw, &seeds); err != nil {
		return fmt.Errorf("seed roles: parse %s: %w", path, err)
	}
	for _, s := range seeds {
		for _, p := range s.Permissions {
			if !isKnownPermission(p) {
				return fmt.Errorf("seed roles: role %q: unknown permission %q", s.Slug, p)
			}
		}
		if _, err := repo.UpsertRole(ctx, s.Slug, s.DisplayName, s.Permissions); err != nil {
			return err
		}
	}
	return nil
}
