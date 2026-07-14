package emailtemplates_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/emailtemplates"
)

func testPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return ctx, pool
}

func makeUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO "user".users (email, password_hash) VALUES ($1, 'x') RETURNING id`,
		"et1-"+uuid.NewString()[:8]+"@example.com",
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func resetSystemSlot(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slotID string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `DELETE FROM settings.system_email_templates WHERE slot_id = $1`, slotID); err != nil {
		t.Fatalf("cleanup system templates: %v", err)
	}
}

func TestListSlots_DefaultMarkdownNonEmpty(t *testing.T) {
	ctx, pool := testPool(t)
	slots, err := emailtemplates.ListSlots(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) < 10 {
		t.Fatalf("expected at least 10 slots (7 original + 3 system), got %d", len(slots))
	}
	byID := map[string]emailtemplates.Slot{}
	for _, s := range slots {
		byID[s.ID] = s
		if s.DefaultMarkdown == "" {
			t.Errorf("slot %q has empty default_markdown", s.ID)
		}
	}
	for _, id := range []string{"magic_link", "coppa_consent", "coppa_consent_confirmation", "welcome", "password_reset"} {
		s, ok := byID[id]
		if !ok {
			t.Errorf("missing slot %q", id)
			continue
		}
		if s.Description == "" {
			t.Errorf("slot %q missing description", id)
		}
		if len(s.MergeFields) == 0 {
			t.Errorf("slot %q missing merge_fields", id)
		}
		if s.DefaultHTML == "" || s.DefaultText == "" {
			t.Errorf("slot %q missing default html/text", id)
		}
	}
	// COPPA must retain required disclosures in markdown.
	coppa := byID["coppa_consent"]
	for _, frag := range []string{"What we collect", "How we use it", "Third-party sharing", "{{link}}", "{{student.name}}"} {
		if !strings.Contains(coppa.DefaultMarkdown, frag) {
			t.Errorf("coppa_consent default_markdown missing %q", frag)
		}
	}
	// magic_link catalog tokens.
	ml := byID["magic_link"]
	for _, tok := range []string{"link", "expires_at", "user.first_name"} {
		if _, ok := ml.MergeFields[tok]; !ok {
			t.Errorf("magic_link merge_fields missing %q", tok)
		}
	}
}

func TestGetActiveSystem_EmptyReturnsNilNil(t *testing.T) {
	ctx, pool := testPool(t)
	resetSystemSlot(t, ctx, pool, "magic_link")
	got, err := emailtemplates.GetActiveSystem(ctx, pool, "magic_link")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil,nil with no override, got %+v", got)
	}
}

func TestSaveSystem_ActiveFlipAndHistory(t *testing.T) {
	ctx, pool := testPool(t)
	resetSystemSlot(t, ctx, pool, "magic_link")
	actor := makeUser(t, ctx, pool)
	text1 := "text v1"
	v1, err := emailtemplates.SaveSystem(ctx, pool, emailtemplates.SaveSystemInput{
		SlotID:         "magic_link",
		SourceMarkdown: "Hello **v1** {{link}}",
		HTMLBody:       "<p>Hello <strong>v1</strong></p>",
		TextBody:       &text1,
		CreatedBy:      actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !v1.IsActive || v1.SourceMarkdown == "" {
		t.Fatalf("v1 unexpected: %+v", v1)
	}

	text2 := "text v2"
	v2, err := emailtemplates.SaveSystem(ctx, pool, emailtemplates.SaveSystemInput{
		SlotID:         "magic_link",
		SourceMarkdown: "Hello **v2** {{link}}",
		HTMLBody:       "<p>Hello <strong>v2</strong></p>",
		TextBody:       &text2,
		CreatedBy:      actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !v2.IsActive {
		t.Fatal("v2 should be active")
	}

	active, err := emailtemplates.GetActiveSystem(ctx, pool, "magic_link")
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != v2.ID {
		t.Fatalf("active want %s got %+v", v2.ID, active)
	}

	// Only one active row allowed.
	var activeCount int
	if err := pool.QueryRow(ctx, `
SELECT count(*) FROM settings.system_email_templates WHERE slot_id = 'magic_link' AND is_active = true
`).Scan(&activeCount); err != nil {
		t.Fatal(err)
	}
	if activeCount != 1 {
		t.Fatalf("want 1 active row, got %d", activeCount)
	}

	hist, err := emailtemplates.ListHistorySystem(ctx, pool, "magic_link")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) < 2 {
		t.Fatalf("want at least 2 history rows, got %d", len(hist))
	}
}

func TestRestoreSystem_AndResetSystem(t *testing.T) {
	ctx, pool := testPool(t)
	resetSystemSlot(t, ctx, pool, "password_reset")
	actor := makeUser(t, ctx, pool)

	v1, err := emailtemplates.SaveSystem(ctx, pool, emailtemplates.SaveSystemInput{
		SlotID:         "password_reset",
		SourceMarkdown: "md-v1",
		HTMLBody:       "<p>v1</p>",
		CreatedBy:      actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	v2, err := emailtemplates.SaveSystem(ctx, pool, emailtemplates.SaveSystemInput{
		SlotID:         "password_reset",
		SourceMarkdown: "md-v2",
		HTMLBody:       "<p>v2</p>",
		CreatedBy:      actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = v2

	restored, err := emailtemplates.RestoreSystem(ctx, pool, "password_reset", v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.SourceMarkdown != "md-v1" || !restored.IsActive {
		t.Fatalf("restore unexpected: %+v", restored)
	}
	if restored.ID == v1.ID {
		t.Fatal("restore should insert a new version row")
	}

	if err := emailtemplates.ResetSystem(ctx, pool, "password_reset"); err != nil {
		t.Fatal(err)
	}
	active, err := emailtemplates.GetActiveSystem(ctx, pool, "password_reset")
	if err != nil {
		t.Fatal(err)
	}
	if active != nil {
		t.Fatalf("after reset expected nil active, got %+v", active)
	}
}

func TestSystemEmailTemplates_PartialUniqueActive(t *testing.T) {
	ctx, pool := testPool(t)
	resetSystemSlot(t, ctx, pool, "welcome")
	actor := makeUser(t, ctx, pool)

	_, err := emailtemplates.SaveSystem(ctx, pool, emailtemplates.SaveSystemInput{
		SlotID:         "welcome",
		SourceMarkdown: "a",
		HTMLBody:       "<p>a</p>",
		CreatedBy:      actor,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Bypass SaveSystem and try to insert a second active row — unique index must reject.
	_, err = pool.Exec(ctx, `
INSERT INTO settings.system_email_templates (slot_id, source_markdown, html_body, created_by, is_active)
VALUES ('welcome', 'b', '<p>b</p>', $1, true)
`, actor)
	if err == nil {
		t.Fatal("expected unique partial index violation for second active row")
	}
}

func TestSystemEmailTemplates_NoOrgIDColumn(t *testing.T) {
	ctx, pool := testPool(t)
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM information_schema.columns
  WHERE table_schema = 'settings' AND table_name = 'system_email_templates' AND column_name = 'org_id'
)`).Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("system_email_templates must not have org_id")
	}
}

