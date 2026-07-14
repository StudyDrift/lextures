package mail

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
)

// MagicLinkOpts carries optional context for template resolution (ET-2).
type MagicLinkOpts struct {
	// FirstName is optional merge data ({{user.first_name}}).
	FirstName string
	// OrgID, when set, prefers the per-org override before system/code defaults.
	OrgID *uuid.UUID
	// Context for DB lookups; Background is used when nil.
	Context context.Context
	// Branding is optional org branding for multipart HTML.
	Branding *BrandingOpts
}

// SendMagicLinkEmail sends a one-time login link when email delivery is configured;
// otherwise dry-runs (parity with password reset). Renders via the template layer
// (system → code default) when wired (ET-2).
func SendMagicLinkEmail(c config.Config, toEmail, magicURL string) error {
	return SendMagicLinkEmailOpts(c, toEmail, magicURL, nil)
}

// SendMagicLinkEmailOpts is the opts-aware magic-link sender.
func SendMagicLinkEmailOpts(c config.Config, toEmail, magicURL string, opts *MagicLinkOpts) error {
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
		branding = opts.Branding
		firstName = opts.FirstName
	}
	vars := map[string]string{
		"link":             magicURL,
		"expires_at":       "in 15 minutes",
		"user.first_name":  firstName,
	}
	rendered, err := RenderSlot(ctx, orgID, "magic_link", vars, branding)
	if err != nil || (rendered.HTMLBody == "" && rendered.BodyText == "") {
		// Hard fallback matching pre-ET-2 plain body.
		body := fmt.Sprintf(`Sign in to your StudyDrift account without a password.

Open this link within 15 minutes (it works only once):

%s

If you did not request this, you can ignore this message.
`, magicURL)
		return SendPlain(c, toEmail, "Your StudyDrift sign-in link", body)
	}
	subject := rendered.Subject
	if subject == "" {
		subject = "Your StudyDrift sign-in link"
	}
	if rendered.HTMLBody != "" {
		return SendMultipart(c, toEmail, subject, rendered.BodyText, rendered.HTMLBody, branding)
	}
	return SendPlain(c, toEmail, subject, rendered.BodyText)
}
