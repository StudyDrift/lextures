package migrate

import "testing"

func TestDemoChecksumRepairMigrations_Includes358(t *testing.T) {
	found := false
	for _, m := range demoChecksumRepairMigrations {
		if m.version == 358 && m.file == "358_learner_profile_core.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 358_learner_profile_core.sql in demoChecksumRepairMigrations")
	}
}
