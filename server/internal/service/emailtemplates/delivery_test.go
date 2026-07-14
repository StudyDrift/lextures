package emailtemplates

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/mail"
)

func TestRenderForDelivery_codeDefaultWhenNoPool(t *testing.T) {
	before := FallbackTotal()
	rendered, err := RenderForDelivery(context.Background(), nil, uuid.Nil, "password_reset", map[string]string{
		"resetUrl": "https://example.edu/reset",
		"link":     "https://example.edu/reset",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rendered.Subject == "" {
		t.Fatal("expected subject")
	}
	if !strings.Contains(rendered.BodyText, "https://example.edu/reset") {
		t.Fatalf("body=%q", rendered.BodyText)
	}
	// No override present → no fallback metric.
	if FallbackTotal() != before {
		t.Fatalf("fallback should not increment for clean code default")
	}
}

func TestRenderSystemForDelivery_codeDefaultMagicLink(t *testing.T) {
	rendered, err := RenderSystemForDelivery(context.Background(), nil, "magic_link", map[string]string{
		"link":       "https://example.edu/magic",
		"expires_at": "in 15 minutes",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rendered.Subject != "Your StudyDrift sign-in link" {
		t.Fatalf("subject=%q", rendered.Subject)
	}
	if !strings.Contains(rendered.BodyText, "https://example.edu/magic") {
		t.Fatalf("body=%q", rendered.BodyText)
	}
}

func TestRenderSystemForDelivery_coppaBuiltIn(t *testing.T) {
	rendered, err := RenderSystemForDelivery(context.Background(), nil, "coppa_consent", map[string]string{
		"student.name": "Alex",
		"link":         "https://example.edu/coppa/consent?token=x",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.BodyText, "Alex") {
		t.Fatalf("body=%q", rendered.BodyText)
	}
	if !strings.Contains(rendered.HTMLBody, "What we collect") {
		t.Fatalf("html missing disclosure: %q", rendered.HTMLBody)
	}
}

func TestMailRenderSlot_withoutWireUsesBuiltIn(t *testing.T) {
	// Ensure slot renderer can be cleared for isolation.
	mail.SetSlotRenderer(nil)
	r, err := mail.RenderSlot(context.Background(), nil, "magic_link", map[string]string{
		"link": "https://x.test/m",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.BodyText, "https://x.test/m") {
		t.Fatalf("body=%q", r.BodyText)
	}
}
