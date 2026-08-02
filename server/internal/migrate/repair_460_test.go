package migrate

import "testing"

func TestDemoChecksumRepairMigrationsIncludes460(t *testing.T) {
	found := false
	for _, m := range demoChecksumRepairMigrations {
		if m.version == 460 && m.file == "460_mobile_link_handling.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 460_mobile_link_handling.sql in demoChecksumRepairMigrations")
	}
}
