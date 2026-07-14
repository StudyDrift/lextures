package mail

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
)

// CoppaSendOpts carries optional org context for template resolution (ET-2).
type CoppaSendOpts struct {
	Branding *BrandingOpts
	OrgID    *uuid.UUID
	OrgName  string
	Context  context.Context
}

// SendCoppaConsentNotice sends the COPPA direct-notice email to a parent (16 CFR §312.4(c)).
// When SMTP is not configured it logs the URL and returns nil (dev parity).
func SendCoppaConsentNotice(c config.Config, parentEmail, studentName, consentURL string, branding *BrandingOpts) error {
	return SendCoppaConsentNoticeOpts(c, parentEmail, studentName, consentURL, &CoppaSendOpts{Branding: branding})
}

// SendCoppaConsentNoticeOpts is the opts-aware COPPA notice sender.
func SendCoppaConsentNoticeOpts(c config.Config, parentEmail, studentName, consentURL string, opts *CoppaSendOpts) error {
	if strings.TrimSpace(parentEmail) == "" {
		return ErrInvalidToEmail
	}
	if !strings.Contains(consentURL, "coppa") && !strings.Contains(consentURL, "consent") {
		// defensive: refuse to send a non-consent URL as a consent notice
		return fmt.Errorf("coppa: consentURL does not look like a consent URL: %q", consentURL)
	}

	if !DeliveryConfigured(c) {
		log.Printf("mail: coppa consent notice for %q student=%q (email delivery not configured) url=%q", parentEmail, studentName, consentURL)
		return nil
	}

	ctx := context.Background()
	var orgID *uuid.UUID
	var branding *BrandingOpts
	orgName := "Lextures"
	if opts != nil {
		if opts.Context != nil {
			ctx = opts.Context
		}
		orgID = opts.OrgID
		branding = opts.Branding
		if strings.TrimSpace(opts.OrgName) != "" {
			orgName = strings.TrimSpace(opts.OrgName)
		}
	}

	vars := map[string]string{
		"student.name": studentName,
		"org.name":     orgName,
		"link":         consentURL,
		"expires_at":   "in 72 hours",
	}
	rendered, err := RenderSlot(ctx, orgID, "coppa_consent", vars, branding)
	if err != nil || (rendered.HTMLBody == "" && rendered.BodyText == "") {
		rendered, err = renderCoppaConsentEmail(studentName, consentURL, branding)
		if err != nil {
			return err
		}
	}
	return SendMultipart(c, parentEmail, rendered.Subject, rendered.BodyText, rendered.HTMLBody, branding)
}

// SendCoppaConsentConfirmation sends a confirmation after parent approval.
func SendCoppaConsentConfirmation(c config.Config, parentEmail, studentName string, branding *BrandingOpts) error {
	return SendCoppaConsentConfirmationOpts(c, parentEmail, studentName, &CoppaSendOpts{Branding: branding})
}

// SendCoppaConsentConfirmationOpts is the opts-aware COPPA confirmation sender.
func SendCoppaConsentConfirmationOpts(c config.Config, parentEmail, studentName string, opts *CoppaSendOpts) error {
	if strings.TrimSpace(parentEmail) == "" {
		return ErrInvalidToEmail
	}
	if !DeliveryConfigured(c) {
		log.Printf("mail: coppa consent confirmed for parent=%q student=%q (email delivery not configured)", parentEmail, studentName)
		return nil
	}
	ctx := context.Background()
	var orgID *uuid.UUID
	var branding *BrandingOpts
	orgName := "Lextures"
	if opts != nil {
		if opts.Context != nil {
			ctx = opts.Context
		}
		orgID = opts.OrgID
		branding = opts.Branding
		if strings.TrimSpace(opts.OrgName) != "" {
			orgName = strings.TrimSpace(opts.OrgName)
		}
	}
	vars := map[string]string{
		"student.name": studentName,
		"org.name":     orgName,
	}
	rendered, err := RenderSlot(ctx, orgID, "coppa_consent_confirmation", vars, branding)
	if err != nil || (rendered.HTMLBody == "" && rendered.BodyText == "") {
		rendered, err = renderCoppaConfirmationEmail(studentName, branding)
		if err != nil {
			return err
		}
	}
	return SendMultipart(c, parentEmail, rendered.Subject, rendered.BodyText, rendered.HTMLBody, branding)
}

func renderCoppaConsentEmail(studentName, consentURL string, branding *BrandingOpts) (RenderedEmail, error) {
	color := "#4F46E5"
	logo := ""
	if branding != nil {
		if strings.TrimSpace(branding.PrimaryColor) != "" {
			color = strings.TrimSpace(branding.PrimaryColor)
		}
		if branding.LogoURL != nil {
			logo = *branding.LogoURL
		}
	}

	escapedName := template.HTMLEscapeString(studentName)
	escapedURL := template.HTMLEscapeString(consentURL)

	subject := fmt.Sprintf("Action required: Review privacy notice for %s", studentName)
	bodyText := fmt.Sprintf(`A school account has been created for %s on the Lextures learning platform.

Under the Children's Online Privacy Protection Act (COPPA), we need your permission before activating their account.

What we collect: first name, school ID, course progress, and quiz responses.
How we use it: to deliver coursework and track learning progress.
Third-party sharing: none without your consent.

To review the full privacy notice and give your permission, open this link (expires in 72 hours):

%s

If you did not expect this message, you can ignore it — no account will be activated without your approval.
`, studentName, consentURL)

	html, err := renderLayout("Parent permission required", fmt.Sprintf(`
<p>A school account has been created for <strong>%s</strong> on the Lextures learning platform.</p>
<p>Under the <strong>Children's Online Privacy Protection Act (COPPA)</strong>, we need your permission before we can activate their account.</p>
<table style="border-collapse:collapse;width:100%%;margin:16px 0;">
  <tr><th style="text-align:left;padding:6px 12px;background:#f3f4f6;border:1px solid #e5e7eb;">What we collect</th>
      <td style="padding:6px 12px;border:1px solid #e5e7eb;">First name, school ID, course progress, quiz responses</td></tr>
  <tr><th style="text-align:left;padding:6px 12px;background:#f3f4f6;border:1px solid #e5e7eb;">How we use it</th>
      <td style="padding:6px 12px;border:1px solid #e5e7eb;">Deliver coursework and track learning progress</td></tr>
  <tr><th style="text-align:left;padding:6px 12px;background:#f3f4f6;border:1px solid #e5e7eb;">Third-party sharing</th>
      <td style="padding:6px 12px;border:1px solid #e5e7eb;">None without your consent</td></tr>
</table>
<p style="margin-top:24px;">
  <a href="%s" style="display:inline-block;background-color:%s;color:#fff;padding:12px 24px;border-radius:6px;font-weight:600;text-decoration:none;">
    Review &amp; Give Permission
  </a>
</p>
<p style="margin-top:16px;font-size:12px;color:#6b7280;">This link expires in 72 hours. If you did not expect this message, you can safely ignore it.</p>`,
		escapedName, escapedURL, template.HTMLEscapeString(color),
	), logo, "")
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}

func renderCoppaConfirmationEmail(studentName string, branding *BrandingOpts) (RenderedEmail, error) {
	color := "#4F46E5"
	logo := ""
	if branding != nil {
		if strings.TrimSpace(branding.PrimaryColor) != "" {
			color = strings.TrimSpace(branding.PrimaryColor)
		}
		if branding.LogoURL != nil {
			logo = *branding.LogoURL
		}
	}
	_ = color

	escapedName := template.HTMLEscapeString(studentName)
	subject := fmt.Sprintf("Permission confirmed for %s", studentName)
	bodyText := fmt.Sprintf(`You have successfully given permission for %s to use Lextures.

Their account is now active. You can manage privacy settings, view collected data, or revoke permission at any time by contacting your school.
`, studentName)

	html, err := renderLayout("Permission confirmed", fmt.Sprintf(`
<p>You have successfully given permission for <strong>%s</strong> to use Lextures.</p>
<p>Their account is now active. You can manage privacy settings, view collected data, or revoke permission at any time by contacting your school.</p>`,
		escapedName,
	), logo, "")
	if err != nil {
		return RenderedEmail{}, err
	}
	return RenderedEmail{Subject: subject, BodyText: bodyText, HTMLBody: html}, nil
}
