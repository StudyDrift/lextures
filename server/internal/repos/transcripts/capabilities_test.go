package transcripts

import "testing"

func TestParseDeliveryMethod(t *testing.T) {
	m, ok := ParseDeliveryMethod(" Postal_Mail ")
	if !ok || m != DeliveryPostalMail {
		t.Fatalf("got %q ok=%v", m, ok)
	}
	if _, ok := ParseDeliveryMethod("carrier_pigeon"); ok {
		t.Fatal("expected reject")
	}
}

func TestMethodAllowedByCapabilities(t *testing.T) {
	caps := []string{"electronic_pdf", "postal_mail"}
	if !MethodAllowedByCapabilities(DeliveryPostalMail, caps) {
		t.Fatal("expected postal_mail allowed")
	}
	if MethodAllowedByCapabilities(DeliveryElectronicPESC, caps) {
		t.Fatal("expected pesc rejected")
	}
}

func TestNormalizeCapabilitiesDropsUnknown(t *testing.T) {
	out := NormalizeCapabilities([]string{"electronic_pdf", "nope", "ELECTRONIC_PDF", "postal_mail"})
	if len(out) != 2 || out[0] != "electronic_pdf" || out[1] != "postal_mail" {
		t.Fatalf("got %#v", out)
	}
}

func TestCanonicalKeyFromName(t *testing.T) {
	if got := CanonicalKeyFromName("  State University "); got != "name:state university" {
		t.Fatalf("got %q", got)
	}
}

func TestOrgEnabledDeliveryMethods(t *testing.T) {
	cfg := &Config{}
	enabled := OrgEnabledDeliveryMethods(cfg)
	if !enabled[DeliveryPostalMail] || !enabled[DeliverySecureLink] {
		t.Fatal("expected baseline methods")
	}
	if enabled[DeliveryElectronicPESC] {
		t.Fatal("pesc should require webhook")
	}
	url := "https://example.com/hook"
	cfg.WebhookURL = &url
	enabled = OrgEnabledDeliveryMethods(cfg)
	if !enabled[DeliveryElectronicPESC] {
		t.Fatal("expected pesc with webhook")
	}
}
