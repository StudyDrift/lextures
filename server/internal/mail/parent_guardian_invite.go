package mail

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
)

// ParentGuardianInviteOpts carries org branding for parent invite email (PP.1).
type ParentGuardianInviteOpts struct {
	FromDisplayName *string
	LogoURL         *string
	PrimaryColor    string
	OrgID           *uuid.UUID
	Context         context.Context
}

// SendParentGuardianInviteEmail sends an activate link for a pending parent/guardian invite.
// Does not include grades or education records — name of student + school + CTA only.
func SendParentGuardianInviteEmail(
	c config.Config,
	toEmail, activateURL, studentName, orgName, firstName string,
	opts *ParentGuardianInviteOpts,
) error {
	if len(toEmail) == 0 {
		return ErrInvalidToEmail
	}

	ctx := context.Background()
	var orgID *uuid.UUID
	var branding *BrandingOpts
	if opts != nil {
		if opts.Context != nil {
			ctx = opts.Context
		}
		orgID = opts.OrgID
		branding = &BrandingOpts{
			FromDisplayName: opts.FromDisplayName,
			LogoURL:         opts.LogoURL,
			PrimaryColor:    opts.PrimaryColor,
		}
	}
	if strings.TrimSpace(firstName) == "" {
		firstName = "there"
	}
	if strings.TrimSpace(studentName) == "" {
		studentName = "your student"
	}
	if strings.TrimSpace(orgName) == "" {
		orgName = "Your school"
	}

	vars := map[string]string{
		"link":            activateURL,
		"expires_at":      "in one hour",
		"user.first_name": firstName,
		"user.email":      toEmail,
		"student.name":    studentName,
		"org.name":        orgName,
	}
	rendered, err := RenderSlot(ctx, orgID, "parent_guardian_invite", vars, branding)
	if err != nil || (rendered.HTMLBody == "" && rendered.BodyText == "") {
		return sendParentGuardianInviteInline(c, toEmail, activateURL, studentName, orgName, firstName, opts)
	}
	subject := rendered.Subject
	if subject == "" {
		subject = fmt.Sprintf("Activate your parent account for %s", studentName)
	}
	return SendMultipart(c, toEmail, subject, rendered.BodyText, rendered.HTMLBody, branding)
}

func sendParentGuardianInviteInline(
	c config.Config,
	toEmail, activateURL, studentName, orgName, firstName string,
	opts *ParentGuardianInviteOpts,
) error {
	subject := fmt.Sprintf("Activate your parent account for %s", studentName)
	bodyText := fmt.Sprintf(`Hi %s,

%s invited you to connect as a parent/guardian of %s.

Activate your account (expires in one hour):
%s

Grades and other records are not included in this email.
`, firstName, orgName, studentName, activateURL)

	color := "#4F46E5"
	logoURL := ""
	if opts != nil {
		if strings.TrimSpace(opts.PrimaryColor) != "" {
			color = strings.TrimSpace(opts.PrimaryColor)
		}
		if opts.LogoURL != nil {
			logoURL = absPublicURL(c, *opts.LogoURL)
		}
	}
	logoBlock := ""
	if logoURL != "" {
		logoBlock = fmt.Sprintf(`<div style="margin-bottom:16px;"><img src="%s" alt="" width="180" style="max-width:100%%;height:auto;" /></div>`, logoURL)
	}
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><body style="font-family:system-ui,sans-serif;line-height:1.5;color:#111827;">
%s
<p>Hi %s,</p>
<p>%s invited you to connect as a parent/guardian of <strong>%s</strong>.</p>
<p><a href="%s" style="color:%s;font-weight:600;">Activate your account</a> (expires in one hour).</p>
<p style="font-size:13px;color:#6b7280;">Grades and other records are not included in this email.</p>
</body></html>`, logoBlock, firstName, orgName, studentName, activateURL, color)

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
