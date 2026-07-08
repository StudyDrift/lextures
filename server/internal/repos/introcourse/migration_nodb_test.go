package introcourse

import (
	"strings"
	"testing"

	serverdata "github.com/lextures/lextures/server"
)

func TestMigration359_IncludesSystemAccountType(t *testing.T) {
	b, err := serverdata.Migrations.ReadFile("migrations/359_intro_course_core.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(b)
	if !strings.Contains(sql, "'system'") {
		t.Fatal("migration must extend account_type to include system")
	}
	if !strings.Contains(sql, SystemUserID.String()) {
		t.Fatal("migration must seed the guide system user")
	}
	if !strings.Contains(sql, "intro_course_enabled") {
		t.Fatal("migration must add intro_course_enabled flag column")
	}
}

func TestMigration361_IncludesIntroCourseItems(t *testing.T) {
	b, err := serverdata.Migrations.ReadFile("migrations/361_intro_course_items.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(b)
	if !strings.Contains(sql, "settings.intro_course_items") {
		t.Fatal("migration must create intro_course_items table")
	}
	if !strings.Contains(sql, "content_version") {
		t.Fatal("migration must track content_version")
	}
}

func TestMigration362_IncludesGradePolicy(t *testing.T) {
	b, err := serverdata.Migrations.ReadFile("migrations/362_intro_course_grading.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(b)
	if !strings.Contains(sql, "grade_policy") {
		t.Fatal("migration must add grade_policy column")
	}
}

func TestMigration364_IncludesCompletions(t *testing.T) {
	b, err := serverdata.Migrations.ReadFile("migrations/364_intro_course_completion.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(b)
	if !strings.Contains(sql, "settings.intro_course_completions") {
		t.Fatal("migration must create intro_course_completions table")
	}
	if !strings.Contains(sql, "event_sent") {
		t.Fatal("migration must track event_sent for dedup")
	}
}

func TestMigration360_IncludesBackfillState(t *testing.T) {
	b, err := serverdata.Migrations.ReadFile("migrations/360_intro_course_backfill_state.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(b)
	if !strings.Contains(sql, "settings.intro_course_backfill") {
		t.Fatal("migration must create intro_course_backfill table")
	}
	if !strings.Contains(sql, "last_user_id") {
		t.Fatal("migration must track backfill resume cursor")
	}
}