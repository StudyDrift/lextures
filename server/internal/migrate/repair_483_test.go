package migrate

import "testing"

func TestDemoChecksumRepairMigrationsIncludes483(t *testing.T) {
	found := false
	for _, m := range demoChecksumRepairMigrations {
		if m.version == 483 && m.file == "483_marketing_content_route_hints.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 483_marketing_content_route_hints.sql in demoChecksumRepairMigrations")
	}
}

func TestDemoChecksumRepairMigrationsIncludes485(t *testing.T) {
	found := false
	for _, m := range demoChecksumRepairMigrations {
		if m.version == 485 && m.file == "485_marketing_content_seed.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 485_marketing_content_seed.sql in demoChecksumRepairMigrations")
	}
}
