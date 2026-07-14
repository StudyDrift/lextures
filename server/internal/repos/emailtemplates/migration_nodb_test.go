package emailtemplates

import (
	"strings"
	"testing"

	serverdata "github.com/lextures/lextures/server"
)

func TestMigration373_MarkdownSystemScope(t *testing.T) {
	up, err := serverdata.Migrations.ReadFile("migrations/373_email_templates_markdown_system_scope.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	sql := string(up)
	for _, want := range []string{
		"default_markdown",
		"source_markdown",
		"settings.system_email_templates",
		"system_email_templates_active",
		"'magic_link'",
		"'coppa_consent'",
		"'coppa_consent_confirmation'",
		"ON CONFLICT (id) DO NOTHING",
		"WHERE id = 'welcome' AND default_markdown = ''",
		"WHERE id = 'password_reset' AND default_markdown = ''",
		"WHERE id = 'grade_posted' AND default_markdown = ''",
		"WHERE id = 'assignment_created' AND default_markdown = ''",
		"WHERE id = 'assignment_due_reminder' AND default_markdown = ''",
		"WHERE id = 'discussion_reply' AND default_markdown = ''",
		"WHERE id = 'enrollment_confirmed' AND default_markdown = ''",
		"WHERE id = 'seat_utilization_alert' AND default_markdown = ''",
		"What we collect",
		"Third-party sharing",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("up migration missing %q", want)
		}
	}

	down, err := serverdata.Migrations.ReadFile("migrations/373_email_templates_markdown_system_scope.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	downSQL := string(down)
	for _, want := range []string{
		"DROP TABLE IF EXISTS settings.system_email_templates",
		"DROP COLUMN IF EXISTS source_markdown",
		"DROP COLUMN IF EXISTS default_markdown",
		"'magic_link'",
		"'coppa_consent'",
		"'coppa_consent_confirmation'",
	} {
		if !strings.Contains(downSQL, want) {
			t.Errorf("down migration missing %q", want)
		}
	}
	// Must not drop the original 18.5 tables.
	if strings.Contains(downSQL, "email_template_slots;") && strings.Contains(downSQL, "DROP TABLE IF EXISTS settings.email_template_slots") {
		t.Error("down migration must not drop email_template_slots table")
	}
	if strings.Contains(downSQL, "DROP TABLE IF EXISTS settings.org_email_templates") {
		t.Error("down migration must not drop org_email_templates table")
	}
}
