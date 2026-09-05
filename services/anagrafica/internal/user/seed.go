package user

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type roleSeed struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
}

// SeedRoles upserts every role listed in the file at path into the roles
// table. Roles are per-committee data, not a Go enum, precisely so a fork
// only has to replace this file — see docs/adr/0012.
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
		if err := repo.UpsertRole(ctx, s.Slug, s.DisplayName); err != nil {
			return err
		}
	}
	return nil
}
