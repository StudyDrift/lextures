// Command content-tools-schema-diff enforces CT.5 FR-4: a manifest's schema
// change must be covered by its semver bump versus the previous git revision.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func main() {
	root := findRepoRoot()
	toolsDir := filepath.Join(root, "server", "internal", "service", "contenttools", "tools")
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "content-tools-schema-diff: %v\n", err)
		os.Exit(1)
	}
	failed := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		toolID := e.Name()
		curPath := filepath.Join(toolsDir, toolID, "manifest.json")
		curRaw, err := os.ReadFile(curPath)
		if err != nil {
			continue
		}
		var cur ctsvc.Manifest
		if err := json.Unmarshal(curRaw, &cur); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %s: invalid manifest: %v\n", toolID, err)
			failed++
			continue
		}
		prevRaw, err := gitShow(root, fmt.Sprintf("HEAD~1:server/internal/service/contenttools/tools/%s/manifest.json", toolID))
		if err != nil || len(prevRaw) == 0 {
			fmt.Printf("OK: %s (no previous manifest to diff)\n", toolID)
			continue
		}
		var prev ctsvc.Manifest
		if err := json.Unmarshal(prevRaw, &prev); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %s: invalid previous manifest: %v\n", toolID, err)
			failed++
			continue
		}
		if prev.Version == cur.Version {
			cfgKind, _, _ := ctsvc.ClassifySchemaDiff(prev.ConfigSchema, cur.ConfigSchema)
			stKind, _, _ := ctsvc.ClassifySchemaDiff(prev.StateSchema, cur.StateSchema)
			if ctsvc.RequiredBumpRank(cfgKind) > 0 || ctsvc.RequiredBumpRank(stKind) > 0 {
				fmt.Fprintf(os.Stderr, "FAIL: %s schema changed without a version bump (config=%s, state=%s)\n", toolID, cfgKind, stKind)
				failed++
				continue
			}
			fmt.Printf("OK: %s@%s unchanged schemas\n", toolID, cur.Version)
			continue
		}
		if err := ctsvc.AssertVersionCoversSchemaDiff(prev.Version, cur.Version, prev.ConfigSchema, cur.ConfigSchema); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %s configSchema: %v\n", toolID, err)
			failed++
		}
		if err := ctsvc.AssertVersionCoversSchemaDiff(prev.Version, cur.Version, prev.StateSchema, cur.StateSchema); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %s stateSchema: %v\n", toolID, err)
			failed++
		}
		if failed == 0 {
			fmt.Printf("OK: %s %s → %s\n", toolID, prev.Version, cur.Version)
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
	fmt.Println("Tool schema-diff check passed.")
}

func gitShow(root, spec string) ([]byte, error) {
	cmd := exec.Command("git", "show", spec)
	cmd.Dir = root
	return cmd.Output()
}

func findRepoRoot() string {
	wd, _ := os.Getwd()
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "server", "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// When run from server/, go up one.
	if strings.HasSuffix(wd, "/server") {
		return filepath.Dir(wd)
	}
	return wd
}
