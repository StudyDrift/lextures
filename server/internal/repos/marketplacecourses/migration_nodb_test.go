package marketplacecourses

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigration_MarketplaceCourseProvisioningPresent(t *testing.T) {
	root := filepath.Join("..", "..", "..", "migrations")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var foundUp, foundDown bool
	for _, e := range entries {
		name := e.Name()
		if strings.Contains(name, "marketplace_course_provisioning") {
			if strings.HasSuffix(name, ".down.sql") {
				foundDown = true
			} else if strings.HasSuffix(name, ".sql") {
				foundUp = true
			}
		}
	}
	if !foundUp || !foundDown {
		t.Fatalf("expected 369_marketplace_course_provisioning.sql (+ .down.sql), up=%v down=%v", foundUp, foundDown)
	}
}
