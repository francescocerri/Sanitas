package shift

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type templateSeed struct {
	Weekday   int    `json:"weekday"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Label     string `json:"label"`
}

// SeedTemplates creates every shift template listed in the file at path,
// but only if the table is empty — a one-time seed, not a reconciliation
// re-applied on every boot like registry's SeedRoles. Shift templates are
// meant to be edited by the shift manager from the app after this initial
// seed (see docs/adr/0025-modello-dati-turni.md); re-imposing the file on
// every restart would silently wipe those edits. Same "only if empty"
// principle as registry.Bootstrap, not SeedRoles.
//
// path == "" is a no-op: seeding is optional, a fork that doesn't configure
// it just starts with no templates (the shift manager creates them from
// the app instead).
func SeedTemplates(ctx context.Context, repo *Repository, path string) error {
	if path == "" {
		return nil
	}
	count, err := repo.CountTemplates(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("seed templates: read %s: %w", path, err)
	}
	var seeds []templateSeed
	if err := json.Unmarshal(raw, &seeds); err != nil {
		return fmt.Errorf("seed templates: parse %s: %w", path, err)
	}
	for _, s := range seeds {
		if _, err := repo.CreateTemplate(ctx, ShiftTemplate{
			Weekday:   s.Weekday,
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
			Label:     s.Label,
		}); err != nil {
			return fmt.Errorf("seed templates: create %q: %w", s.Label, err)
		}
	}
	return nil
}
