package mail

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/lextures/lextures/server/internal/config"
)

// BrandingOpts carries optional org branding for transactional email (plan 5.7).
type BrandingOpts struct {
	FromDisplayName *string
	LogoURL         *string
	PrimaryColor    string
}

// SendMultipart sends a multipart alternative email (plain + HTML) via the
// configured email provider (SMTP by default; SES when enabled and selected).
func SendMultipart(c config.Config, toEmail, subject, bodyText, htmlBody string, branding *BrandingOpts) error {
	if len(strings.TrimSpace(toEmail)) == 0 {
		return ErrInvalidToEmail
	}

	fromDisplay := ""
	if branding != nil && branding.FromDisplayName != nil {
		fromDisplay = strings.TrimSpace(*branding.FromDisplayName)
	}

	// Validate From early when delivery is configured so callers get clear errors.
	if DeliveryConfigured(c) {
		from := EffectiveFromAddress(c)
		if from == "" {
			return fmt.Errorf("from address is required when email delivery is configured")
		}
		if _, err := mail.ParseAddress(from); err != nil {
			return fmt.Errorf("parse from address: %w", err)
		}
	}

	color := "#4F46E5"
	logoURL := ""
	if branding != nil {
		if strings.TrimSpace(branding.PrimaryColor) != "" {
			color = strings.TrimSpace(branding.PrimaryColor)
		}
		if branding.LogoURL != nil {
			logoURL = absPublicURL(c, *branding.LogoURL)
		}
	}
	if htmlBody == "" {
		htmlBody = plainToSimpleHTML(bodyText, logoURL, color)
	}

	return deliver(c, Message{
		To:              toEmail,
		Subject:         subject,
		BodyText:        bodyText,
		HTMLBody:        htmlBody,
		FromDisplayName: fromDisplay,
	})
}

// SendMultipartWithICS sends plain+HTML email with an optional ICS calendar attachment (plan 13.12).
func SendMultipartWithICS(c config.Config, toEmail, subject, bodyText, htmlBody string, branding *BrandingOpts, icsContent, icsFilename string) error {
	if strings.TrimSpace(icsContent) == "" {
		return SendMultipart(c, toEmail, subject, bodyText, htmlBody, branding)
	}
	if len(strings.TrimSpace(toEmail)) == 0 {
		return ErrInvalidToEmail
	}

	fromDisplay := ""
	if branding != nil && branding.FromDisplayName != nil {
		fromDisplay = strings.TrimSpace(*branding.FromDisplayName)
	}

	if DeliveryConfigured(c) {
		from := EffectiveFromAddress(c)
		if from == "" {
			return fmt.Errorf("from address is required when email delivery is configured")
		}
		if _, err := mail.ParseAddress(from); err != nil {
			return fmt.Errorf("parse from address: %w", err)
		}
	}

	if htmlBody == "" {
		color := "#4F46E5"
		logoURL := ""
		if branding != nil {
			if strings.TrimSpace(branding.PrimaryColor) != "" {
				color = strings.TrimSpace(branding.PrimaryColor)
			}
			if branding.LogoURL != nil {
				logoURL = absPublicURL(c, *branding.LogoURL)
			}
		}
		htmlBody = plainToSimpleHTML(bodyText, logoURL, color)
	}

	return deliver(c, Message{
		To:              toEmail,
		Subject:         subject,
		BodyText:        bodyText,
		HTMLBody:        htmlBody,
		FromDisplayName: fromDisplay,
		ICSContent:      icsContent,
		ICSFilename:     icsFilename,
	})
}

// SendPlain delivers a text-only email (magic link, etc.) via the active provider.
func SendPlain(c config.Config, toEmail, subject, bodyText string) error {
	if len(strings.TrimSpace(toEmail)) == 0 {
		return ErrInvalidToEmail
	}
	return deliver(c, Message{
		To:       toEmail,
		Subject:  subject,
		BodyText: bodyText,
	})
}

func plainToSimpleHTML(bodyText, logoURL, _ string) string {
	escaped := strings.ReplaceAll(bodyText, "&", "&amp;")
	escaped = strings.ReplaceAll(escaped, "<", "&lt;")
	escaped = strings.ReplaceAll(escaped, "\n", "<br/>\n")
	logo := ""
	if logoURL != "" {
		logo = fmt.Sprintf(`<div style="margin-bottom:16px;"><img src="%s" alt="" width="180" style="max-width:100%%;height:auto;" /></div>`, logoURL)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:system-ui,sans-serif;line-height:1.5;color:#111827;font-size:14px;">
%s
<p style="color:#374151;">%s</p>
</body></html>`, logo, escaped)
}
