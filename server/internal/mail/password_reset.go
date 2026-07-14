package mail

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
)

var (
	ErrInvalidToEmail = errors.New("invalid to email address")
)

// PasswordResetOpts carries optional org branding for transactional email (plan 5.7 / ET-2).
type PasswordResetOpts struct {
	FromDisplayName *string
	LogoURL         *string
	PrimaryColor    string
	// OrgID enables per-org template override resolution (ET-2).
	OrgID *uuid.UUID
	// Context for template DB lookups; Background when nil.
	Context context.Context
	// FirstName optional merge field.
	FirstName string
}

func absPublicURL(cfg config.Config, pathOrURL string) string {
	s := strings.TrimSpace(pathOrURL)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.LTIAPIBaseURL), "/")
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return base + s
}

// SendPasswordResetEmail sends a reset link when email delivery is configured;
// otherwise logs a dry-run and returns nil (Rust parity). Renders via the
// template layer (org → system → code) when wired (ET-2).
func SendPasswordResetEmail(c config.Config, toEmail, resetURL string, opts *PasswordResetOpts) error {
	if len(toEmail) == 0 {
		return ErrInvalidToEmail
	}

	ctx := context.Background()
	var orgID *uuid.UUID
	var branding *BrandingOpts
	firstName := ""
	if opts != nil {
		if opts.Context != nil {
			ctx = opts.Context
		}
		orgID = opts.OrgID
		firstName = opts.FirstName
		branding = &BrandingOpts{
			FromDisplayName: opts.FromDisplayName,
			LogoURL:         opts.LogoURL,
			PrimaryColor:    opts.PrimaryColor,
		}
	}

	vars := map[string]string{
		"link":            resetURL,
		"resetUrl":        resetURL,
		"expires_at":      "in one hour",
		"user.first_name": firstName,
	}
	rendered, err := RenderSlot(ctx, orgID, "password_reset", vars, branding)
	if err != nil || (rendered.HTMLBody == "" && rendered.BodyText == "") {
		// Inline fallback (pre-ET-2 body).
		return sendPasswordResetInline(c, toEmail, resetURL, opts)
	}
	subject := rendered.Subject
	if subject == "" {
		subject = "Reset your StudyDrift password"
	}
	return SendMultipart(c, toEmail, subject, rendered.BodyText, rendered.HTMLBody, branding)
}

func sendPasswordResetInline(c config.Config, toEmail, resetURL string, opts *PasswordResetOpts) error {
	subject := "Reset your StudyDrift password"
	bodyText := fmt.Sprintf(`You requested a password reset for your StudyDrift account.

Open this link to choose a new password (it expires in one hour):

%s

If you did not request this, you can ignore this message.
`, resetURL)

	logoURL := ""
	if opts != nil && opts.LogoURL != nil {
		logoURL = absPublicURL(c, *opts.LogoURL)
	}
	color := "#4F46E5"
	if opts != nil && strings.TrimSpace(opts.PrimaryColor) != "" {
		color = strings.TrimSpace(opts.PrimaryColor)
	}

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><body style="font-family:system-ui,sans-serif;line-height:1.5;color:#111827;">
%s
<p>You requested a password reset for your StudyDrift account.</p>
<p><a href="%s" style="color:%s;font-weight:600;">Choose a new password</a> (expires in one hour).</p>
<p style="font-size:13px;color:#6b7280;">If you did not request this, you can ignore this message.</p>
</body></html>`,
		func() string {
			if logoURL == "" {
				return ""
			}
			return fmt.Sprintf(`<div style="margin-bottom:16px;"><img src="%s" alt="" width="180" style="max-width:100%%;height:auto;" /></div>`, logoURL)
		}(),
		resetURL,
		color,
	)

	var branding *BrandingOpts
	if opts != nil {
		branding = &BrandingOpts{
			FromDisplayName: opts.FromDisplayName,
			LogoURL:         opts.LogoURL,
			PrimaryColor:    opts.PrimaryColor,
		}
	}
	return SendMultipart(c, toEmail, subject, bodyText, htmlBody, branding)
}
