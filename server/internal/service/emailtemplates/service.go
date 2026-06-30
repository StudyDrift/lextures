package emailtemplates

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/mail"
	emailtemplatesrepo "github.com/lextures/lextures/server/internal/repos/emailtemplates"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// Service renders and manages org email templates (plan 18.5).
type Service struct {
	Pool *pgxpool.Pool
	Cfg  config.Config
}

// PreviewResult is merged HTML and text for preview APIs.
type PreviewResult struct {
	HTML string `json:"html"`
	Text string `json:"text"`
}

// SaveResult is the response from saving a template.
type SaveResult struct {
	Version       emailtemplatesrepo.OrgVersion `json:"version"`
	UnknownFields []string                      `json:"unknownFields,omitempty"`
}

// SampleData builds preview merge data for an admin user.
func (s *Service) SampleData(ctx context.Context, orgID, userID uuid.UUID) (map[string]string, error) {
	data := map[string]string{
		"link":             "https://example.edu/courses/demo",
		"unsubscribe_url":  "https://example.edu/settings/notifications",
		"expires_at":       "in 1 hour",
		"assignment.due_at": "Friday at 11:59 PM",
	}
	u, err := user.FindByID(ctx, s.Pool, userID)
	if err != nil {
		return nil, err
	}
	if u != nil {
		data["user.email"] = u.Email
		data["user.first_name"] = strOr(u.FirstName, "Alex")
		data["user.last_name"] = strOr(u.LastName, "Rivera")
	}
	org, err := organization.GetByID(ctx, s.Pool, orgID)
	if err != nil {
		return nil, err
	}
	if org != nil {
		data["org.name"] = org.Name
	}
	data["course.title"] = "Introduction to Biology"
	data["assignment.title"] = "Lab Report 2"
	data["discussion.title"] = "Week 3 discussion"
	return data, nil
}

func strOr(p *string, fallback string) string {
	if p != nil && strings.TrimSpace(*p) != "" {
		return strings.TrimSpace(*p)
	}
	return fallback
}

// Preview merges a template body with sample or provided data.
func (s *Service) Preview(htmlBody string, textBody *string, data map[string]string) PreviewResult {
	html := Merge(SanitizeHTML(htmlBody), data)
	text := ""
	if textBody != nil && strings.TrimSpace(*textBody) != "" {
		text = Merge(*textBody, data)
	} else {
		text = StripHTMLTags(html)
	}
	return PreviewResult{HTML: wrapPreviewHTML(html), Text: text}
}

func wrapPreviewHTML(body string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"/></head><body style="font-family:system-ui,sans-serif;line-height:1.5;padding:16px;">%s</body></html>`, body)
}

// ValidateUnknown returns unrecognized merge tokens for html and optional text.
func ValidateUnknown(slot *emailtemplatesrepo.Slot, htmlBody string, textBody *string) []string {
	allowed := slot.MergeFields
	unknown := FindUnknownTokens(htmlBody, allowed)
	if textBody != nil {
		for _, k := range FindUnknownTokens(*textBody, allowed) {
			found := false
			for _, u := range unknown {
				if u == k {
					found = true
					break
				}
			}
			if !found {
				unknown = append(unknown, k)
			}
		}
	}
	return unknown
}

// Save sanitizes, stores a new version, and returns warnings for unknown fields.
func (s *Service) Save(ctx context.Context, orgID uuid.UUID, slotID, htmlBody string, textBody, replyTo, senderName *string, actor uuid.UUID) (*SaveResult, error) {
	slot, err := emailtemplatesrepo.GetSlot(ctx, s.Pool, slotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, fmt.Errorf("unknown slot")
	}
	cleanHTML := SanitizeHTML(htmlBody)
	var cleanText *string
	if textBody != nil && strings.TrimSpace(*textBody) != "" {
		t := strings.TrimSpace(*textBody)
		cleanText = &t
	} else {
		generated := StripHTMLTags(cleanHTML)
		cleanText = &generated
	}
	unknown := ValidateUnknown(slot, cleanHTML, cleanText)
	version, err := emailtemplatesrepo.Save(ctx, s.Pool, emailtemplatesrepo.SaveInput{
		OrgID:      orgID,
		SlotID:     slotID,
		HTMLBody:   cleanHTML,
		TextBody:   cleanText,
		ReplyTo:    replyTo,
		SenderName: senderName,
		CreatedBy:  actor,
	})
	if err != nil {
		return nil, err
	}
	RecordSave()
	return &SaveResult{Version: *version, UnknownFields: unknown}, nil
}

// SendTestEmail sends the current template to the requesting admin only.
func (s *Service) SendTestEmail(ctx context.Context, orgID uuid.UUID, slotID string, actor uuid.UUID, branding *mail.BrandingOpts) error {
	slot, err := emailtemplatesrepo.GetSlot(ctx, s.Pool, slotID)
	if err != nil {
		return err
	}
	if slot == nil {
		return fmt.Errorf("unknown slot")
	}
	data, err := s.SampleData(ctx, orgID, actor)
	if err != nil {
		return err
	}
	active, err := emailtemplatesrepo.GetActive(ctx, s.Pool, orgID, slotID)
	if err != nil {
		return err
	}
	htmlBody := slot.DefaultHTML
	textBody := slot.DefaultText
	if active != nil {
		htmlBody = active.HTMLBody
		if active.TextBody != nil {
			textBody = *active.TextBody
		}
	}
	preview := s.Preview(htmlBody, &textBody, data)
	to, err := user.FindByID(ctx, s.Pool, actor)
	if err != nil {
		return err
	}
	if to == nil {
		return fmt.Errorf("user not found")
	}
	subject := fmt.Sprintf("[Test] %s", slot.Description)
	if err := mail.SendMultipart(s.Cfg, to.Email, subject, preview.Text, preview.HTML, branding); err != nil {
		return err
	}
	RecordTestSend()
	return nil
}

// RenderForDelivery tries org override then falls back to built-in mail templates.
func RenderForDelivery(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, slotName string, vars map[string]string, branding *mail.BrandingOpts) (mail.RenderedEmail, error) {
	if pool != nil {
		active, err := emailtemplatesrepo.GetActive(ctx, pool, orgID, slotName)
		if err == nil && active != nil {
			slot, slotErr := emailtemplatesrepo.GetSlot(ctx, pool, slotName)
			if slotErr == nil && slot != nil {
				data := MapJobVars(vars)
				html := Merge(active.HTMLBody, data)
				text := ""
				if active.TextBody != nil {
					text = Merge(*active.TextBody, data)
				} else {
					text = StripHTMLTags(html)
				}
				subject := slot.Description
				wrapped, wrapErr := renderWrappedHTML(html, branding, vars["unsubscribeUrl"])
				if wrapErr != nil {
					slog.Warn("email_template.render_wrap_failed", "slot", slotName, "err", wrapErr)
				} else {
					html = wrapped
				}
				return mail.RenderedEmail{Subject: subject, BodyText: text, HTMLBody: html}, nil
			}
		}
	}
	rendered, err := mail.RenderTemplate(slotName, vars, branding)
	if err != nil {
		slog.Warn("email_template.render_fallback_failed", "slot", slotName, "err", err)
	}
	return rendered, err
}

func renderWrappedHTML(body string, branding *mail.BrandingOpts, unsubscribeURL string) (string, error) {
	logo := ""
	if branding != nil {
		if branding.LogoURL != nil {
			logo = strings.TrimSpace(*branding.LogoURL)
		}
	}
	logoBlock := ""
	if logo != "" {
		logoBlock = fmt.Sprintf(`<div style="margin-bottom:16px;"><img src="%s" alt="" width="180" style="max-width:100%%;height:auto;" /></div>`, template.HTMLEscapeString(logo))
	}
	footer := ""
	if unsubscribeURL != "" {
		footer = fmt.Sprintf(`<p style="margin-top:24px;font-size:12px;color:#6b7280;"><a href="%s" style="color:#6b7280;">Unsubscribe from this notification type</a></p>`, template.HTMLEscapeString(unsubscribeURL))
	}
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"/></head><body style="font-family:system-ui,sans-serif;line-height:1.5;color:#111827;font-size:14px;">%s%s%s</body></html>`, logoBlock, body, footer), nil
}
