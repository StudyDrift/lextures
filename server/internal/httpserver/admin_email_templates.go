package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/mail"
	"github.com/lextures/lextures/server/internal/repos/emailtemplates"
	"github.com/lextures/lextures/server/internal/repos/orgbranding"
	emailtemplatesvc "github.com/lextures/lextures/server/internal/service/emailtemplates"
)

func (d Deps) emailTemplateEditorEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().EmailTemplateEditorEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Email template editor is not enabled.")
		return false
	}
	if !d.adminConsoleEnabled(w) {
		return false
	}
	return true
}

func (d Deps) registerAdminEmailTemplateRoutes(r chi.Router) {
	r.Get("/api/v1/admin-console/email-templates", d.handleAdminEmailTemplatesList())
	r.Get("/api/v1/admin-console/email-templates/{slotId}", d.handleAdminEmailTemplateGet())
	r.Put("/api/v1/admin-console/email-templates/{slotId}", d.handleAdminEmailTemplatePut())
	r.Get("/api/v1/admin-console/email-templates/{slotId}/history", d.handleAdminEmailTemplateHistory())
	r.Post("/api/v1/admin-console/email-templates/{slotId}/restore", d.handleAdminEmailTemplateRestore())
	r.Post("/api/v1/admin-console/email-templates/{slotId}/reset", d.handleAdminEmailTemplateReset())
	r.Post("/api/v1/admin-console/email-templates/{slotId}/test", d.handleAdminEmailTemplateTest())
	r.Post("/api/v1/admin-console/email-templates/{slotId}/preview", d.handleAdminEmailTemplatePreview())
}

type adminEmailTemplateSlotJSON struct {
	ID            string            `json:"id"`
	Description   string            `json:"description"`
	MergeFields   map[string]string `json:"mergeFields"`
	DefaultHTML   string            `json:"defaultHtml"`
	DefaultText   string            `json:"defaultText"`
	HasCustom     bool              `json:"hasCustom"`
	ActiveID      *string           `json:"activeId,omitempty"`
	UpdatedAt     *string           `json:"updatedAt,omitempty"`
	ReplyTo       *string           `json:"replyTo,omitempty"`
	SenderName    *string           `json:"senderName,omitempty"`
	UnknownFields []string          `json:"unknownFields,omitempty"`
}

type adminEmailTemplateVersionJSON struct {
	ID         string  `json:"id"`
	OrgID      string  `json:"orgId"`
	SlotID     string  `json:"slotId"`
	HTMLBody   string  `json:"htmlBody"`
	TextBody   *string `json:"textBody,omitempty"`
	ReplyTo    *string `json:"replyTo,omitempty"`
	SenderName *string `json:"senderName,omitempty"`
	CreatedBy  *string `json:"createdBy,omitempty"`
	CreatedAt  string  `json:"createdAt"`
	IsActive   bool    `json:"isActive"`
}

func toVersionJSON(v emailtemplates.OrgVersion) adminEmailTemplateVersionJSON {
	out := adminEmailTemplateVersionJSON{
		ID:         v.ID.String(),
		OrgID:      v.OrgID.String(),
		SlotID:     v.SlotID,
		HTMLBody:   v.HTMLBody,
		TextBody:   v.TextBody,
		ReplyTo:    v.ReplyTo,
		SenderName: v.SenderName,
		CreatedAt:  v.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		IsActive:   v.IsActive,
	}
	if v.CreatedBy != nil {
		s := v.CreatedBy.String()
		out.CreatedBy = &s
	}
	return out
}

func (d Deps) emailTemplateService() emailtemplatesvc.Service {
	return emailtemplatesvc.Service{Pool: d.Pool, Cfg: d.effectiveConfig()}
}

func (d Deps) handleAdminEmailTemplatesList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, orgID, _, ok := d.adminEmailTemplateAccess(w, r, false)
		if !ok {
			return
		}
		slots, err := emailtemplates.ListSlots(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load email templates.")
			return
		}
		out := make([]adminEmailTemplateSlotJSON, 0, len(slots))
		for _, slot := range slots {
			item := adminEmailTemplateSlotJSON{
				ID:          slot.ID,
				Description: slot.Description,
				MergeFields: slot.MergeFields,
				DefaultHTML: slot.DefaultHTML,
				DefaultText: slot.DefaultText,
			}
			active, err := emailtemplates.GetActive(r.Context(), d.Pool, orgID, slot.ID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load email templates.")
				return
			}
			if active != nil {
				item.HasCustom = true
				id := active.ID.String()
				item.ActiveID = &id
				ts := active.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
				item.UpdatedAt = &ts
				item.ReplyTo = active.ReplyTo
				item.SenderName = active.SenderName
				item.UnknownFields = emailtemplatesvc.ValidateUnknown(&slot, active.HTMLBody, active.TextBody)
			}
			out = append(out, item)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handleAdminEmailTemplateGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, orgID, _, ok := d.adminEmailTemplateAccess(w, r, false)
		if !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		slot, err := emailtemplates.GetSlot(r.Context(), d.Pool, slotID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load email template.")
			return
		}
		if slot == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template slot not found.")
			return
		}
		resp := struct {
			adminEmailTemplateSlotJSON
			Active *adminEmailTemplateVersionJSON `json:"active,omitempty"`
		}{
			adminEmailTemplateSlotJSON: adminEmailTemplateSlotJSON{
				ID:          slot.ID,
				Description: slot.Description,
				MergeFields: slot.MergeFields,
				DefaultHTML: slot.DefaultHTML,
				DefaultText: slot.DefaultText,
			},
		}
		active, err := emailtemplates.GetActive(r.Context(), d.Pool, orgID, slotID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load email template.")
			return
		}
		if active != nil {
			resp.HasCustom = true
			id := active.ID.String()
			resp.ActiveID = &id
			ts := active.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
			resp.UpdatedAt = &ts
			resp.ReplyTo = active.ReplyTo
			resp.SenderName = active.SenderName
			v := toVersionJSON(*active)
			resp.Active = &v
			resp.UnknownFields = emailtemplatesvc.ValidateUnknown(slot, active.HTMLBody, active.TextBody)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handleAdminEmailTemplatePut() http.HandlerFunc {
	svc := d.emailTemplateService()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, _, ok := d.adminEmailTemplateAccess(w, r, true)
		if !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		var body struct {
			SourceMarkdown string  `json:"sourceMarkdown"`
			HTMLBody       string  `json:"htmlBody"`
			TextBody       *string `json:"textBody"`
			ReplyTo        *string `json:"replyTo"`
			SenderName     *string `json:"senderName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var result *emailtemplatesvc.SaveResult
		var err error
		switch {
		case strings.TrimSpace(body.SourceMarkdown) != "":
			// ET-2: Markdown is canonical.
			result, err = svc.Save(r.Context(), orgID, slotID, body.SourceMarkdown, body.TextBody, body.ReplyTo, body.SenderName, actor)
		case strings.TrimSpace(body.HTMLBody) != "":
			// 18.5 / legacy HTML editor until ET-3 lands.
			result, err = svc.SaveHTML(r.Context(), orgID, slotID, body.HTMLBody, body.TextBody, body.ReplyTo, body.SenderName, actor)
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "sourceMarkdown or htmlBody is required.")
			return
		}
		if err != nil {
			if err.Error() == "unknown slot" {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template slot not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save email template.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			adminEmailTemplateVersionJSON
			UnknownFields []string `json:"unknownFields,omitempty"`
		}{
			adminEmailTemplateVersionJSON: toVersionJSON(result.Version),
			UnknownFields:                 result.UnknownFields,
		})
	}
}

func (d Deps) handleAdminEmailTemplateHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, orgID, _, ok := d.adminEmailTemplateAccess(w, r, false)
		if !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		versions, err := emailtemplates.ListHistory(r.Context(), d.Pool, orgID, slotID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load version history.")
			return
		}
		out := make([]adminEmailTemplateVersionJSON, 0, len(versions))
		for _, v := range versions {
			out = append(out, toVersionJSON(v))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handleAdminEmailTemplateRestore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, orgID, _, ok := d.adminEmailTemplateAccess(w, r, true)
		if !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		var body struct {
			VersionID string `json:"versionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		versionID, err := uuid.Parse(strings.TrimSpace(body.VersionID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid versionId.")
			return
		}
		version, err := emailtemplates.Restore(r.Context(), d.Pool, orgID, slotID, versionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Version not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to restore version.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toVersionJSON(*version))
	}
}

func (d Deps) handleAdminEmailTemplateReset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, orgID, _, ok := d.adminEmailTemplateAccess(w, r, true)
		if !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		if err := emailtemplates.Reset(r.Context(), d.Pool, orgID, slotID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reset template.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleAdminEmailTemplateTest() http.HandlerFunc {
	svc := d.emailTemplateService()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, _, ok := d.adminEmailTemplateAccess(w, r, true)
		if !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		branding := d.mailBrandingForOrg(r.Context(), orgID)
		if err := svc.SendTestEmail(r.Context(), orgID, slotID, actor, branding); err != nil {
			if err.Error() == "unknown slot" {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template slot not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to send test email.")
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func (d Deps) handleAdminEmailTemplatePreview() http.HandlerFunc {
	svc := d.emailTemplateService()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, _, ok := d.adminEmailTemplateAccess(w, r, false)
		if !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}
		var body struct {
			SourceMarkdown string            `json:"sourceMarkdown"`
			HTMLBody       string            `json:"htmlBody"`
			TextBody       *string           `json:"textBody"`
			SampleData     map[string]string `json:"sampleData"`
		}
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		useMarkdown := strings.TrimSpace(body.SourceMarkdown) != ""
		if !useMarkdown && strings.TrimSpace(body.HTMLBody) == "" {
			slot, slotErr := emailtemplates.GetSlot(r.Context(), d.Pool, slotID)
			if slotErr != nil || slot == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template slot not found.")
				return
			}
			active, activeErr := emailtemplates.GetActive(r.Context(), d.Pool, orgID, slotID)
			if activeErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to preview template.")
				return
			}
			if active != nil {
				if active.SourceMarkdown != nil && strings.TrimSpace(*active.SourceMarkdown) != "" {
					body.SourceMarkdown = *active.SourceMarkdown
					useMarkdown = true
				} else {
					body.HTMLBody = active.HTMLBody
				}
				body.TextBody = active.TextBody
			} else if strings.TrimSpace(slot.DefaultMarkdown) != "" {
				body.SourceMarkdown = slot.DefaultMarkdown
				useMarkdown = true
				t := slot.DefaultText
				body.TextBody = &t
			} else {
				body.HTMLBody = slot.DefaultHTML
				t := slot.DefaultText
				body.TextBody = &t
			}
		}
		data := body.SampleData
		if len(data) == 0 {
			data, err = svc.SampleData(r.Context(), orgID, actor)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build preview data.")
				return
			}
		}
		var preview emailtemplatesvc.PreviewResult
		if useMarkdown {
			preview = svc.Preview(body.SourceMarkdown, body.TextBody, data)
		} else {
			preview = svc.PreviewHTML(body.HTMLBody, body.TextBody, data)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(preview)
	}
}

func (d Deps) adminEmailTemplateAccess(w http.ResponseWriter, r *http.Request, wantManage bool) (actor uuid.UUID, targetOrg uuid.UUID, globalAdmin bool, ok bool) {
	if !d.emailTemplateEditorEnabled(w) {
		return uuid.UUID{}, uuid.UUID{}, false, false
	}
	return d.adminConsoleAccess(w, r, wantManage)
}

func (d Deps) mailBrandingForOrg(ctx context.Context, orgID uuid.UUID) *mail.BrandingOpts {
	row, err := orgbranding.Get(ctx, d.Pool, orgID)
	if err != nil || row == nil {
		return nil
	}
	return &mail.BrandingOpts{
		FromDisplayName: row.CustomEmailDisplayName,
		LogoURL:         row.LogoURL,
		PrimaryColor:    row.PrimaryColor,
	}
}
