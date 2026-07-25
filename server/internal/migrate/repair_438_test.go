package migrate

import "testing"

func TestDemoChecksumRepairMigrations_Includes438(t *testing.T) {
	found := false
	for _, m := range demoChecksumRepairMigrations {
		if m.version == 438 && m.file == "438_course_grade_levels_array.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 438_course_grade_levels_array.sql in demoChecksumRepairMigrations")
	}
}
