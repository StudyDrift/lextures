package mail

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/lextures/lextures/server/internal/config"
)

// Provider names for EmailProvider config. Additional providers can be registered
// without changing call sites — only the factory needs to know about them.
const (
	ProviderSMTP = "smtp"
	ProviderSES  = "ses"
)

// Message is a provider-agnostic outbound email.
type Message struct {
	To              string
	Subject         string
	BodyText        string
	HTMLBody        string
	FromDisplayName string
	// ICSContent, when non-empty, attaches a calendar invite (raw MIME path).
	ICSContent  string
	ICSFilename string
}

// Provider delivers transactional email via a specific backend (SMTP, SES, …).
// Implementations must be safe for concurrent use.
type Provider interface {
	// Name returns the provider identifier (smtp, ses, …).
	Name() string
	// Configured reports whether this provider has enough settings to send.
	// When false, callers may log a dry-run and return nil (dev parity).
	Configured(cfg config.Config) bool
	// Send delivers msg. Callers should only invoke when Configured is true.
	Send(ctx context.Context, cfg config.Config, msg Message) error
}

// SelectProvider returns the active email delivery backend for cfg.
//
// Selection rules:
//  1. Normalize EmailProvider (default "smtp").
//  2. SES is only used when EmailProvider=ses AND FFEmailSES is true.
//  3. Unknown providers fall back to SMTP so misconfiguration does not drop mail.
//
// Future providers (SendGrid API, Mailgun, Postmark, …) plug in here.
func SelectProvider(cfg config.Config) Provider {
	name := NormalizeEmailProvider(cfg.EmailProvider)
	switch name {
	case ProviderSES:
		if !cfg.FFEmailSES {
			log.Printf("mail: emailProvider=ses ignored because ffEmailSes is disabled; using smtp")
			return smtpProvider{}
		}
		return sesProvider{}
	case ProviderSMTP:
		return smtpProvider{}
	default:
		log.Printf("mail: unknown emailProvider %q; using smtp", name)
		return smtpProvider{}
	}
}

// NormalizeEmailProvider returns a canonical provider name (smtp, ses, …).
func NormalizeEmailProvider(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ProviderSMTP
	}
	return s
}

// DeliveryConfigured reports whether the active provider can send mail.
func DeliveryConfigured(cfg config.Config) bool {
	return SelectProvider(cfg).Configured(cfg)
}

// EffectiveFromAddress returns the From address for the active provider.
// SES prefers SESFrom, then SMTPFrom; SMTP uses SMTPFrom.
func EffectiveFromAddress(cfg config.Config) string {
	p := SelectProvider(cfg)
	if p.Name() == ProviderSES {
		if v := strings.TrimSpace(cfg.SESFrom); v != "" {
			return v
		}
	}
	return strings.TrimSpace(cfg.SMTPFrom)
}

// deliver sends via the selected provider, or dry-runs when not configured.
func deliver(cfg config.Config, msg Message) error {
	if strings.TrimSpace(msg.To) == "" {
		return ErrInvalidToEmail
	}
	p := SelectProvider(cfg)
	if !p.Configured(cfg) {
		log.Printf("mail: would send via %s to %q subject=%q (provider not configured)", p.Name(), msg.To, msg.Subject)
		return nil
	}
	if err := p.Send(context.Background(), cfg, msg); err != nil {
		return fmt.Errorf("mail %s: %w", p.Name(), err)
	}
	return nil
}
