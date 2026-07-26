package httpserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRouteInventory is the TD.1 route-inventory harness (FR-1–FR-4, FR-8–FR-9).
//
// It walks the chi router returned by NewHandler, probes each route unauthenticated
// to classify auth posture, and compares against the committed golden file.
//
// Regenerate intentionally:
//
//	UPDATE_GOLDEN=1 go test ./internal/httpserver/ -run TestRouteInventory -count=1
//	# or: make route-inventory-update
func TestRouteInventory(t *testing.T) {
	start := time.Now()
	got := buildRouteInventory(t)
	if len(got) == 0 {
		t.Fatal("route inventory is empty — NewHandler registered no routes")
	}

	goldenPath := filepath.Join("testdata", "route_inventory.golden")
	if updateGoldenRequested() {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		body := formatRouteInventory(got)
		if err := os.WriteFile(goldenPath, []byte(body), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated %s (%d routes) in %s", goldenPath, len(got), time.Since(start).Round(time.Millisecond))
		return
	}

	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v\nRun with UPDATE_GOLDEN=1 to create it.", goldenPath, err)
	}
	want, err := parseRouteInventory(string(raw))
	if err != nil {
		t.Fatalf("parse golden: %v", err)
	}

	added, removed, changed := diffRouteInventory(want, got)
	if len(added) == 0 && len(removed) == 0 && len(changed) == 0 {
		// Determinism check: format must be stable.
		if formatRouteInventory(got) != string(raw) {
			// Allow trailing whitespace drift only if line sets match — fail otherwise.
			t.Fatalf("inventory content differs from golden despite set equality (formatting drift).\nRegenerate with UPDATE_GOLDEN=1 if intentional.")
		}
		t.Logf("route inventory OK: %d routes (anonymous=%d session=%d) in %s",
			len(got), countAuth(got, authAnonymous), countAuth(got, authSession), time.Since(start).Round(time.Millisecond))
		// Budget: inventory must stay well under 5s (NFR).
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("route inventory took %s (> 5s budget)", elapsed)
		}
		return
	}

	t.Errorf("route inventory diverged from %s", goldenPath)
	t.Errorf("  live: %d routes | golden: %d routes", len(got), len(want))
	if len(added) > 0 {
		t.Errorf("  added (%d):", len(added))
		for _, line := range added {
			t.Errorf("    %s", line)
		}
	}
	if len(removed) > 0 {
		t.Errorf("  removed (%d):", len(removed))
		for _, line := range removed {
			t.Errorf("    %s", line)
		}
	}
	if len(changed) > 0 {
		t.Errorf("  changed (%d):", len(changed))
		for _, line := range changed {
			t.Errorf("    %s", line)
		}
	}
	t.Errorf("If the change is intentional, regenerate:\n  UPDATE_GOLDEN=1 go test ./internal/httpserver/ -run TestRouteInventory -count=1")
}

func countAuth(entries []routeInventoryEntry, auth string) int {
	n := 0
	for _, e := range entries {
		if e.Auth == auth {
			n++
		}
	}
	return n
}

// TestRouteInventory_KnownPostures pins the highest-value auth classifications
// so a regression cannot hide behind a mass golden update.
func TestRouteInventory_KnownPostures(t *testing.T) {
	h := inventoryTestHandler(t)
	cases := []struct {
		method, pattern, want string
	}{
		{httpMethodGet, "/health", authAnonymous},
		{httpMethodGet, "/health/live", authAnonymous},
		{httpMethodGet, "/api/openapi.json", authAnonymous},
		{httpMethodGet, "/api/v1/courses", authSession},
		{httpMethodGet, "/api/v1/platform/features", authSession},
		{httpMethodGet, "/health/detailed", authSession},
	}
	for _, tc := range cases {
		got := probeAuthPosture(h, tc.method, tc.pattern)
		if got != tc.want {
			t.Errorf("%s %s: auth=%s want %s", tc.method, tc.pattern, got, tc.want)
		}
	}
}

// Local constants avoid importing net/http solely for method names in table tests.
const (
	httpMethodGet = "GET"
)

// TestRouteInventory_Determinism ensures two consecutive builds are byte-identical (AC-7).
func TestRouteInventory_Determinism(t *testing.T) {
	a := formatRouteInventory(buildRouteInventory(t))
	b := formatRouteInventory(buildRouteInventory(t))
	if a != b {
		t.Fatal("route inventory is non-deterministic across consecutive builds")
	}
}

// TestRouteInventoryPrint prints the live inventory (used by `make route-inventory`).
// Not a golden assertion — always passes after printing.
func TestRouteInventoryPrint(t *testing.T) {
	got := buildRouteInventory(t)
	// Print via t.Log so -v shows lines; also write to stdout for make pipelines.
	text := formatRouteInventory(got)
	_, _ = os.Stdout.WriteString(text)
	t.Logf("%d routes", len(got))
}
