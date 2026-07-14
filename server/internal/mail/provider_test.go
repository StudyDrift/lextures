package mail

import (
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestNormalizeEmailProvider(t *testing.T) {
	if got := NormalizeEmailProvider(""); got != ProviderSMTP {
		t.Fatalf("empty: got %q", got)
	}
	if got := NormalizeEmailProvider(" SES "); got != ProviderSES {
		t.Fatalf("ses: got %q", got)
	}
	if got := NormalizeEmailProvider("SMTP"); got != ProviderSMTP {
		t.Fatalf("smtp: got %q", got)
	}
}

func TestSelectProvider_DefaultSMTP(t *testing.T) {
	p := SelectProvider(config.Config{})
	if p.Name() != ProviderSMTP {
		t.Fatalf("got %q", p.Name())
	}
}

func TestSelectProvider_SESRequiresFlag(t *testing.T) {
	p := SelectProvider(config.Config{EmailProvider: "ses", FFEmailSES: false})
	if p.Name() != ProviderSMTP {
		t.Fatalf("expected smtp when flag off, got %q", p.Name())
	}
	p = SelectProvider(config.Config{EmailProvider: "ses", FFEmailSES: true})
	if p.Name() != ProviderSES {
		t.Fatalf("expected ses when flag on, got %q", p.Name())
	}
}

func TestSelectProvider_UnknownFallsBackToSMTP(t *testing.T) {
	p := SelectProvider(config.Config{EmailProvider: "sendgrid-api"})
	if p.Name() != ProviderSMTP {
		t.Fatalf("got %q", p.Name())
	}
}

func TestDeliveryConfigured_SMTP(t *testing.T) {
	if DeliveryConfigured(config.Config{}) {
		t.Fatal("empty config should not be configured")
	}
	if !DeliveryConfigured(config.Config{SMTPHost: "localhost", SMTPFrom: "a@b.c"}) {
		t.Fatal("SMTP host should configure delivery")
	}
}

func TestDeliveryConfigured_SES(t *testing.T) {
	cfg := config.Config{EmailProvider: "ses", FFEmailSES: true, SESFrom: "a@b.c"}
	if !DeliveryConfigured(cfg) {
		t.Fatal("SES with from should be configured")
	}
	cfg.SESFrom = ""
	cfg.SMTPFrom = "fallback@b.c"
	if !DeliveryConfigured(cfg) {
		t.Fatal("SES should fall back to SMTPFrom")
	}
	cfg.SMTPFrom = ""
	if DeliveryConfigured(cfg) {
		t.Fatal("SES without from should not be configured")
	}
}

func TestEffectiveFromAddress(t *testing.T) {
	cfg := config.Config{
		EmailProvider: "ses",
		FFEmailSES:    true,
		SESFrom:       "ses@x.com",
		SMTPFrom:      "smtp@x.com",
	}
	if got := EffectiveFromAddress(cfg); got != "ses@x.com" {
		t.Fatalf("got %q", got)
	}
	cfg.SESFrom = ""
	if got := EffectiveFromAddress(cfg); got != "smtp@x.com" {
		t.Fatalf("fallback got %q", got)
	}
}

func TestSendPlain_NotConfigured(t *testing.T) {
	if err := SendPlain(config.Config{}, "u@x.com", "subj", "body"); err != nil {
		t.Fatal(err)
	}
}

func TestSendMultipart_MissingFromWhenConfigured(t *testing.T) {
	cfg := config.Config{SMTPHost: "localhost"}
	if err := SendMultipart(cfg, "u@x.com", "s", "t", "", nil); err == nil {
		t.Fatal("expected missing from error")
	}
}
