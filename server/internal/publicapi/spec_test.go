package publicapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenAPI31DocumentMatchesCommittedSnapshot(t *testing.T) {
	t.Parallel()
	root := findModuleRoot(t)
	snapshotPath := filepath.Join(root, "openapi", "v1.json")
	committed, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	got := strings.TrimSpace(OpenAPI31Document)
	want := strings.TrimSpace(string(committed))
	if got != want {
		t.Fatalf("OpenAPI spec drifted from %s; regenerate with: go test ./internal/publicapi -run TestOpenAPI31DocumentMatchesCommittedSnapshot -update (or copy OpenAPI31Document to openapi/v1.json)", snapshotPath)
	}
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
