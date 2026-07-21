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

// Service renders and manages org/system email templates (plan 18.5 / ET-2).
type Service struct {
	Pool *pgxpool.Pool
	Cfg  config.Config
}

// PreviewResult is merged HTML and text for preview APIs.
type PreviewResult struct {
	HTML string `json:"html"`
	Text string `json:"text"`
}

// SaveResult is the response from saving an org template.
type SaveResult struct {
	Version       emailtemplatesrepo.OrgVersion `json:"version"`
	UnknownFields []string                      `json:"unknownFields,omitempty"`
}

// SystemSaveResult is the response from saving a system template.
type SystemSaveResult struct {
	Version       emailtemplatesrepo.SystemVersion `json:"version"`
	UnknownFields []string                         `json:"unknownFields,omitempty"`
}

// DeliveryOverridesEnabled, when set, gates org/system override lookup.
// When nil or returning false, Render* uses built-in code defaults only.
// App wiring sets this from EmailTemplateEditorEnabled.
var DeliveryOverridesEnabled func() bool

func overridesEnabled() bool {
	if DeliveryOverridesEnabled == nil {
		// Unset: allow overrides so unit/integration tests exercise the chain.
		return true
	}
	return DeliveryOverridesEnabled()
}

// SampleData builds preview merge data for an admin user.
func (s *Service) SampleData(ctx context.Context, orgID, userID uuid.UUID) (map[string]string, error) {
	data := map[string]string{
		"link":              "https://example.edu/courses/demo",
		"unsubscribe_url":   "https://example.edu/settings/notifications",
		"expires_at":        "in 1 hour",
		"assignment.due_at": "Friday at 11:59 PM",
		"student.name":      "Jamie",
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
	} else {
		data["org.name"] = "Example School"
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

// Preview compiles Markdown (when provided as source), sanitizes, merges sample
// data, and returns wrapped HTML + plain text. When textBody is nil/empty, text
// is derived from the compiled HTML.
func (s *Service) Preview(markdown string, textBody *string, data map[string]string) PreviewResult {
	htmlBody, err := Compile(markdown)
	if err != nil {
		// Fall back to treating input as pre-compiled HTML (legacy preview).
		htmlBody = SanitizeHTML(markdown)
	}
	html := Merge(htmlBody, data)
	text := ""
	if textBody != nil && strings.TrimSpace(*textBody) != "" {
		text = Merge(*textBody, data)
	} else {
		text = StripHTMLTags(html)
	}
	return PreviewResult{HTML: wrapPreviewHTML(html), Text: text}
}

// PreviewHTML merges pre-authored HTML (18.5 editor path) without Markdown compile.
func (s *Service) PreviewHTML(htmlBody string, textBody *string, data map[string]string) PreviewResult {
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
	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"/></head><body style="font-family:system-ui,sans-serif;line-height:1.5;padding:16px;">%s</body></html>`, body)
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
	// Also flag unknown tokens in markdown source catalogs (same body text).
	return unknown
}

// ValidateUnknownMarkdown checks markdown source and optional text against the slot catalog.
func ValidateUnknownMarkdown(slot *emailtemplatesrepo.Slot, markdown string, textBody *string) []string {
	unknown := FindUnknownTokens(markdown, slot.MergeFields)
	if textBody != nil {
		for _, k := range FindUnknownTokens(*textBody, slot.MergeFields) {
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

// Save compiles Markdown, sanitizes HTML, derives text, and stores a new active org version.
func (s *Service) Save(ctx context.Context, orgID uuid.UUID, slotID, markdown string, textBody, replyTo, senderName *string, actor uuid.UUID) (*SaveResult, error) {
	slot, err := emailtemplatesrepo.GetSlot(ctx, s.Pool, slotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, fmt.Errorf("unknown slot")
	}
	cleanHTML, err := Compile(markdown)
	if err != nil {
		return nil, err
	}
	cleanText := deriveText(textBody, cleanHTML)
	md := strings.TrimSpace(markdown)
	unknown := ValidateUnknownMarkdown(slot, md, cleanText)
	// Also check compiled HTML tokens (should match).
	for _, k := range ValidateUnknown(slot, cleanHTML, cleanText) {
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
	version, err := emailtemplatesrepo.Save(ctx, s.Pool, emailtemplatesrepo.SaveInput{
		OrgID:          orgID,
		SlotID:         slotID,
		SourceMarkdown: &md,
		HTMLBody:       cleanHTML,
		TextBody:       cleanText,
		ReplyTo:        replyTo,
		SenderName:     senderName,
		CreatedBy:      actor,
	})
	if err != nil {
		return nil, err
	}
	RecordSave()
	return &SaveResult{Version: *version, UnknownFields: unknown}, nil
}

// SaveHTML stores a pre-authored HTML body (18.5 / legacy admin editor). source_markdown is left nil.
func (s *Service) SaveHTML(ctx context.Context, orgID uuid.UUID, slotID, htmlBody string, textBody, replyTo, senderName *string, actor uuid.UUID) (*SaveResult, error) {
	slot, err := emailtemplatesrepo.GetSlot(ctx, s.Pool, slotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, fmt.Errorf("unknown slot")
	}
	cleanHTML := SanitizeHTML(htmlBody)
	if strings.TrimSpace(cleanHTML) == "" {
		return nil, fmt.Errorf("empty html body")
	}
	cleanText := deriveText(textBody, cleanHTML)
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

// SaveSystem compiles Markdown and stores a new active system (platform) version.
func (s *Service) SaveSystem(ctx context.Context, slotID, markdown string, textBody, replyTo, senderName *string, actor uuid.UUID) (*SystemSaveResult, error) {
	slot, err := emailtemplatesrepo.GetSlot(ctx, s.Pool, slotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, fmt.Errorf("unknown slot")
	}
	cleanHTML, err := Compile(markdown)
	if err != nil {
		return nil, err
	}
	cleanText := deriveText(textBody, cleanHTML)
	md := strings.TrimSpace(markdown)
	unknown := ValidateUnknownMarkdown(slot, md, cleanText)
	version, err := emailtemplatesrepo.SaveSystem(ctx, s.Pool, emailtemplatesrepo.SaveSystemInput{
		SlotID:         slotID,
		SourceMarkdown: md,
		HTMLBody:       cleanHTML,
		TextBody:       cleanText,
		ReplyTo:        replyTo,
		SenderName:     senderName,
		CreatedBy:      actor,
	})
	if err != nil {
		return nil, err
	}
	RecordSave()
	return &SystemSaveResult{Version: *version, UnknownFields: unknown}, nil
}

func deriveText(textBody *string, cleanHTML string) *string {
	if textBody != nil && strings.TrimSpace(*textBody) != "" {
		t := strings.TrimSpace(*textBody)
		return &t
	}
	generated := StripHTMLTags(cleanHTML)
	return &generated
}

// SendTestEmail sends the current org template to the requesting admin only.
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
	var preview PreviewResult
	if active != nil {
		if active.SourceMarkdown != nil && strings.TrimSpace(*active.SourceMarkdown) != "" {
			preview = s.Preview(*active.SourceMarkdown, active.TextBody, data)
		} else {
			preview = s.PreviewHTML(active.HTMLBody, active.TextBody, data)
		}
	} else if strings.TrimSpace(slot.DefaultMarkdown) != "" {
		t := slot.DefaultText
		preview = s.Preview(slot.DefaultMarkdown, &t, data)
	} else {
		t := slot.DefaultText
		preview = s.PreviewHTML(slot.DefaultHTML, &t, data)
	}
	return s.sendTestToActor(ctx, actor, slot.Description, preview, branding)
}

// SendSystemTestEmail sends the current system-scope template to the requesting admin only (ET-3).
func (s *Service) SendSystemTestEmail(ctx context.Context, slotID string, actor uuid.UUID, branding *mail.BrandingOpts) error {
	slot, err := emailtemplatesrepo.GetSlot(ctx, s.Pool, slotID)
	if err != nil {
		return err
	}
	if slot == nil {
		return fmt.Errorf("unknown slot")
	}
	// Prefer actor's org for sample merge data when available.
	orgID, _ := organization.OrgIDForUser(ctx, s.Pool, actor)
	data, err := s.SampleData(ctx, orgID, actor)
	if err != nil {
		return err
	}
	active, err := emailtemplatesrepo.GetActiveSystem(ctx, s.Pool, slotID)
	if err != nil {
		return err
	}
	var preview PreviewResult
	if active != nil {
		if strings.TrimSpace(active.SourceMarkdown) != "" {
			preview = s.Preview(active.SourceMarkdown, active.TextBody, data)
		} else {
			preview = s.PreviewHTML(active.HTMLBody, active.TextBody, data)
		}
	} else if strings.TrimSpace(slot.DefaultMarkdown) != "" {
		t := slot.DefaultText
		preview = s.Preview(slot.DefaultMarkdown, &t, data)
	} else {
		t := slot.DefaultText
		preview = s.PreviewHTML(slot.DefaultHTML, &t, data)
	}
	return s.sendTestToActor(ctx, actor, slot.Description, preview, branding)
}

func (s *Service) sendTestToActor(ctx context.Context, actor uuid.UUID, description string, preview PreviewResult, branding *mail.BrandingOpts) error {
	to, err := user.FindByID(ctx, s.Pool, actor)
	if err != nil {
		return err
	}
	if to == nil {
		return fmt.Errorf("user not found")
	}
	subject := fmt.Sprintf("[Test] %s", description)
	if err := mail.SendMultipart(s.Cfg, to.Email, subject, preview.Text, preview.HTML, branding); err != nil {
		return err
	}
	RecordTestSend()
	return nil
}

// RenderForDelivery resolves org → system → built-in code default.
func RenderForDelivery(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, slotName string, vars map[string]string, branding *mail.BrandingOpts) (mail.RenderedEmail, error) {
	if pool != nil && overridesEnabled() {
		if orgID != uuid.Nil {
			active, err := emailtemplatesrepo.GetActive(ctx, pool, orgID, slotName)
			if err == nil && active != nil {
				if strings.TrimSpace(active.HTMLBody) == "" {
					slog.Warn("email_template.override_empty_html", "slot", slotName, "scope", "org")
					RecordFallback()
				} else if rendered, ok := renderOverride(ctx, pool, slotName, active.HTMLBody, active.TextBody, vars, branding, "org"); ok {
					return rendered, nil
				}
				// renderOverride already recorded fallback when it returns false
			}
		}
		sys, err := emailtemplatesrepo.GetActiveSystem(ctx, pool, slotName)
		if err == nil && sys != nil {
			if strings.TrimSpace(sys.HTMLBody) == "" {
				slog.Warn("email_template.override_empty_html", "slot", slotName, "scope", "system")
				RecordFallback()
			} else if rendered, ok := renderOverride(ctx, pool, slotName, sys.HTMLBody, sys.TextBody, vars, branding, "system"); ok {
				return rendered, nil
			}
		}
	}
	slog.Debug("email_template.render_code_default", "slot", slotName, "scope", "code")
	rendered, err := mail.RenderTemplate(slotName, vars, branding)
	if err != nil {
		slog.Warn("email_template.render_fallback_failed", "slot", slotName, "err", err)
	}
	return rendered, err
}

// RenderSystemForDelivery resolves system → built-in code default (no org context).
func RenderSystemForDelivery(ctx context.Context, pool *pgxpool.Pool, slotName string, vars map[string]string, branding *mail.BrandingOpts) (mail.RenderedEmail, error) {
	if pool != nil && overridesEnabled() {
		sys, err := emailtemplatesrepo.GetActiveSystem(ctx, pool, slotName)
		if err == nil && sys != nil {
			if strings.TrimSpace(sys.HTMLBody) == "" {
				slog.Warn("email_template.override_empty_html", "slot", slotName, "scope", "system")
				RecordFallback()
			} else if rendered, ok := renderOverride(ctx, pool, slotName, sys.HTMLBody, sys.TextBody, vars, branding, "system"); ok {
				return rendered, nil
			}
		}
	}
	slog.Debug("email_template.render_code_default", "slot", slotName, "scope", "code")
	rendered, err := mail.RenderTemplate(slotName, vars, branding)
	if err != nil {
		slog.Warn("email_template.render_fallback_failed", "slot", slotName, "err", err)
	}
	return rendered, err
}

func renderOverride(ctx context.Context, pool *pgxpool.Pool, slotName, htmlBody string, textBody *string, vars map[string]string, branding *mail.BrandingOpts, scope string) (mail.RenderedEmail, bool) {
	slot, slotErr := emailtemplatesrepo.GetSlot(ctx, pool, slotName)
	if slotErr != nil || slot == nil {
		slog.Warn("email_template.override_slot_missing", "slot", slotName, "scope", scope, "err", slotErr)
		RecordFallback()
		return mail.RenderedEmail{}, false
	}
	if strings.TrimSpace(htmlBody) == "" {
		slog.Warn("email_template.override_empty_html", "slot", slotName, "scope", scope)
		RecordFallback()
		return mail.RenderedEmail{}, false
	}
	data := MapJobVars(vars)
	html := Merge(htmlBody, data)
	if strings.TrimSpace(html) == "" {
		slog.Warn("email_template.override_empty_after_merge", "slot", slotName, "scope", scope)
		RecordFallback()
		return mail.RenderedEmail{}, false
	}
	text := ""
	if textBody != nil {
		text = Merge(*textBody, data)
	} else {
		text = StripHTMLTags(html)
	}
	subject := subjectForSlot(slot, vars)
	unsub := vars["unsubscribeUrl"]
	if unsub == "" {
		unsub = vars["unsubscribe_url"]
	}
	wrapped, wrapErr := renderWrappedHTML(html, branding, unsub)
	if wrapErr != nil {
		slog.Warn("email_template.render_wrap_failed", "slot", slotName, "scope", scope, "err", wrapErr)
	} else {
		html = wrapped
	}
	return mail.RenderedEmail{Subject: subject, BodyText: text, HTMLBody: html}, true
}

func subjectForSlot(slot *emailtemplatesrepo.Slot, vars map[string]string) string {
	if s := strings.TrimSpace(vars["subject"]); s != "" {
		return s
	}
	// Preserve historical subjects for system emails (FR-10).
	switch slot.ID {
	case "magic_link":
		return "Your StudyDrift sign-in link"
	case "password_reset":
		return "Reset your StudyDrift password"
	case "parent_guardian_invite":
		if name := strings.TrimSpace(vars["student.name"]); name != "" {
			return fmt.Sprintf("Activate your parent account for %s", name)
		}
		return "Activate your parent account"
	case "coppa_consent":
		if name := strings.TrimSpace(vars["student.name"]); name != "" {
			return fmt.Sprintf("Action required: Review privacy notice for %s", name)
		}
		return "Action required: Review privacy notice"
	case "coppa_consent_confirmation":
		if name := strings.TrimSpace(vars["student.name"]); name != "" {
			return fmt.Sprintf("Permission confirmed for %s", name)
		}
		return "Permission confirmed"
	default:
		if slot.Description != "" {
			return slot.Description
		}
		return "Notification from StudyDrift"
	}
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
	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"/></head><body style="font-family:system-ui,sans-serif;line-height:1.5;color:#111827;font-size:14px;">%s%s%s</body></html>`, logoBlock, body, footer), nil
}
