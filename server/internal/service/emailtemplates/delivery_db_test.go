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
	"github.com/lextures/lextures/server/internal/mail"
	"github.com/lextures/lextures/server/internal/migrate"
	emailtemplatesrepo "github.com/lextures/lextures/server/internal/repos/emailtemplates"
	"github.com/lextures/lextures/server/internal/repos/organization"
	emailtemplatesvc "github.com/lextures/lextures/server/internal/service/emailtemplates"
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
		"et2-"+uuid.NewString()[:8]+"@example.com",
	).Scan(&id)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	return id
}

func TestSaveSystem_AndRenderSystemForDelivery(t *testing.T) {
	ctx, pool := testPool(t)
	emailtemplatesvc.DeliveryOverridesEnabled = func() bool { return true }
	t.Cleanup(func() { emailtemplatesvc.DeliveryOverridesEnabled = nil })

	if _, err := pool.Exec(ctx, `DELETE FROM settings.system_email_templates WHERE slot_id = 'magic_link'`); err != nil {
		t.Fatal(err)
	}
	actor := makeUser(t, ctx, pool)
	svc := emailtemplatesvc.Service{Pool: pool}

	md := "Custom sign-in for **you**: [Go]({{link}}) expires {{expires_at}}."
	_, err := svc.SaveSystem(ctx, "magic_link", md, nil, nil, nil, actor)
	if err != nil {
		t.Fatal(err)
	}

	rendered, err := emailtemplatesvc.RenderSystemForDelivery(ctx, pool, "magic_link", map[string]string{
		"link":       "https://app.test/magic",
		"expires_at": "soon",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.HTMLBody, "https://app.test/magic") {
		t.Fatalf("html missing merged link: %q", rendered.HTMLBody)
	}
	if strings.Contains(rendered.HTMLBody, "{{link}}") {
		t.Fatalf("token not merged: %q", rendered.HTMLBody)
	}
	if !strings.Contains(rendered.HTMLBody, "Custom sign-in") && !strings.Contains(rendered.BodyText, "Custom sign-in") {
		t.Fatalf("override body missing: html=%q text=%q", rendered.HTMLBody, rendered.BodyText)
	}
}

func TestRenderForDelivery_orgBeatsSystem(t *testing.T) {
	ctx, pool := testPool(t)
	emailtemplatesvc.DeliveryOverridesEnabled = func() bool { return true }
	t.Cleanup(func() { emailtemplatesvc.DeliveryOverridesEnabled = nil })

	actor := makeUser(t, ctx, pool)
	orgID, err := organization.OrgIDForUser(ctx, pool, actor)
	if err != nil || orgID == uuid.Nil {
		// Create a dedicated org if user has none.
		err = pool.QueryRow(ctx, `
INSERT INTO tenant.organizations (slug, name, status)
VALUES ($1, 'ET2 Org', 'active') RETURNING id`, "et2-"+uuid.NewString()[:8]).Scan(&orgID)
		if err != nil {
			t.Fatalf("org: %v", err)
		}
	}

	if _, err := pool.Exec(ctx, `DELETE FROM settings.system_email_templates WHERE slot_id = 'password_reset'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM settings.org_email_templates WHERE org_id = $1 AND slot_id = 'password_reset'`, orgID); err != nil {
		t.Fatal(err)
	}

	svc := emailtemplatesvc.Service{Pool: pool}
	if _, err := svc.SaveSystem(ctx, "password_reset", "SYSTEM override [sys]({{link}})", nil, nil, nil, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Save(ctx, orgID, "password_reset", "ORG override [org]({{link}})", nil, nil, nil, actor); err != nil {
		t.Fatal(err)
	}

	rendered, err := emailtemplatesvc.RenderForDelivery(ctx, pool, orgID, "password_reset", map[string]string{
		"link": "https://app.test/reset",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.HTMLBody, "ORG override") && !strings.Contains(rendered.BodyText, "ORG override") {
		t.Fatalf("expected org override, got html=%q text=%q", rendered.HTMLBody, rendered.BodyText)
	}
	if strings.Contains(rendered.HTMLBody, "SYSTEM override") || strings.Contains(rendered.BodyText, "SYSTEM override") {
		t.Fatal("system override should not win over org")
	}
}

func TestRenderSystemForDelivery_emptyOverrideFallsBack(t *testing.T) {
	ctx, pool := testPool(t)
	emailtemplatesvc.DeliveryOverridesEnabled = func() bool { return true }
	t.Cleanup(func() { emailtemplatesvc.DeliveryOverridesEnabled = nil })

	if _, err := pool.Exec(ctx, `DELETE FROM settings.system_email_templates WHERE slot_id = 'magic_link'`); err != nil {
		t.Fatal(err)
	}
	actor := makeUser(t, ctx, pool)
	// Insert an active row with empty-ish body that fails override rendering after strip.
	// HTML body is a single space after we force empty via repo save bypass.
	_, err := emailtemplatesrepo.SaveSystem(ctx, pool, emailtemplatesrepo.SaveSystemInput{
		SlotID:         "magic_link",
		SourceMarkdown: "x",
		HTMLBody:       "   ", // whitespace only → treated as empty
		CreatedBy:      actor,
	})
	if err != nil {
		t.Fatal(err)
	}

	before := emailtemplatesvc.FallbackTotal()
	rendered, err := emailtemplatesvc.RenderSystemForDelivery(ctx, pool, "magic_link", map[string]string{
		"link": "https://app.test/magic",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if emailtemplatesvc.FallbackTotal() <= before {
		t.Fatal("expected fallback metric increment for empty override")
	}
	// Built-in default should still produce a body.
	if !strings.Contains(rendered.BodyText, "https://app.test/magic") && !strings.Contains(rendered.HTMLBody, "https://app.test/magic") {
		t.Fatalf("fallback body missing link: %+v", rendered)
	}
}

func TestSendMagicLink_usesSystemOverride(t *testing.T) {
	ctx, pool := testPool(t)
	emailtemplatesvc.DeliveryOverridesEnabled = func() bool { return true }
	t.Cleanup(func() {
		emailtemplatesvc.DeliveryOverridesEnabled = nil
		mail.SetSlotRenderer(nil)
	})
	emailtemplatesvc.WireMailSlotRenderer(pool, func() bool { return true })

	if _, err := pool.Exec(ctx, `DELETE FROM settings.system_email_templates WHERE slot_id = 'magic_link'`); err != nil {
		t.Fatal(err)
	}
	actor := makeUser(t, ctx, pool)
	svc := emailtemplatesvc.Service{Pool: pool}
	const marker = "ET2-MAGIC-MARKER"
	if _, err := svc.SaveSystem(ctx, "magic_link", marker+" [Sign in]({{link}})", nil, nil, nil, actor); err != nil {
		t.Fatal(err)
	}

	// Capture via slot renderer path (RenderSlot) without SMTP.
	rendered, err := mail.RenderSlot(ctx, nil, "magic_link", map[string]string{
		"link":       "https://app.test/m",
		"expires_at": "soon",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.HTMLBody, marker) && !strings.Contains(rendered.BodyText, marker) {
		t.Fatalf("expected system override in render: html=%q text=%q", rendered.HTMLBody, rendered.BodyText)
	}
	if !strings.Contains(rendered.HTMLBody, "https://app.test/m") {
		t.Fatalf("link not merged: %q", rendered.HTMLBody)
	}
}
