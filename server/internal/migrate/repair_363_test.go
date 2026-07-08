package migrate

import "testing"

func TestDemoChecksumRepairMigrations_Includes363(t *testing.T) {
	found := false
	for _, m := range demoChecksumRepairMigrations {
		if m.version == 363 && m.file == "363_lp_adaptivity_flags.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 363_lp_adaptivity_flags.sql in demoChecksumRepairMigrations")
	}
}