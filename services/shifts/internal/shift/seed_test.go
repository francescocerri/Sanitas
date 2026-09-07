package shift

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeSeedFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "templates.json")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	return path
}

func TestSeedTemplates_CreatesFromFileWhenEmpty(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	path := writeSeedFile(t, `[
		{"weekday": 4, "start_time": "20:00", "end_time": "08:00", "label": "Turno Serale"},
		{"weekday": 6, "start_time": "08:00", "end_time": "14:00", "label": "Turno Mattina"}
	]`)

	if err := SeedTemplates(ctx, repo, path); err != nil {
		t.Fatalf("SeedTemplates: %v", err)
	}

	got, err := repo.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 seeded templates, got %d", len(got))
	}
}

// The whole point of seeding "only if empty": a shift manager's edits
// (or deactivations) made from the app must survive a service restart,
// not get silently overwritten by re-applying the seed file.
func TestSeedTemplates_NoOpWhenNotEmpty(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()
	newTestTemplate(t, repo)

	path := writeSeedFile(t, `[{"weekday": 1, "start_time": "08:00", "end_time": "14:00", "label": "Should not be created"}]`)
	if err := SeedTemplates(ctx, repo, path); err != nil {
		t.Fatalf("SeedTemplates: %v", err)
	}

	got, err := repo.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected the pre-existing template to be the only one, got %d", len(got))
	}
}

func TestSeedTemplates_NoOpWhenPathEmpty(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	if err := SeedTemplates(ctx, repo, ""); err != nil {
		t.Fatalf("SeedTemplates: %v", err)
	}

	n, err := repo.CountTemplates(ctx)
	if err != nil {
		t.Fatalf("CountTemplates: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected no templates, got %d", n)
	}
}
