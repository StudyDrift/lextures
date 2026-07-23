package migrate

import "testing"

func TestDemoChecksumRepairMigrations_Includes431(t *testing.T) {
	found := false
	for _, m := range demoChecksumRepairMigrations {
		if m.version == 431 && m.file == "431_parent_link_assign.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 431_parent_link_assign.sql in demoChecksumRepairMigrations")
	}
}
