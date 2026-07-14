package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/emailtemplates"
	emailtemplatesvc "github.com/lextures/lextures/server/internal/service/emailtemplates"
)

// registerPlatformEmailTemplateRoutes mounts system-scope email template APIs (ET-3).
func (d Deps) registerPlatformEmailTemplateRoutes(r chi.Router) {
	r.Get("/api/v1/settings/platform/email-templates", d.handlePlatformEmailTemplatesList())
	r.Get("/api/v1/settings/platform/email-templates/{slotId}", d.handlePlatformEmailTemplateGet())
	r.Put("/api/v1/settings/platform/email-templates/{slotId}", d.handlePlatformEmailTemplatePut())
	r.Get("/api/v1/settings/platform/email-templates/{slotId}/history", d.handlePlatformEmailTemplateHistory())
	r.Post("/api/v1/settings/platform/email-templates/{slotId}/restore", d.handlePlatformEmailTemplateRestore())
	r.Post("/api/v1/settings/platform/email-templates/{slotId}/reset", d.handlePlatformEmailTemplateReset())
	r.Post("/api/v1/settings/platform/email-templates/{slotId}/test", d.handlePlatformEmailTemplateTest())
	r.Post("/api/v1/settings/platform/email-templates/{slotId}/preview", d.handlePlatformEmailTemplatePreview())
}

type platformEmailTemplateSlotJSON struct {
	ID              string            `json:"id"`
	Description     string            `json:"description"`
	MergeFields     map[string]string `json:"mergeFields"`
	DefaultHTML     string            `json:"defaultHtml"`
	DefaultText     string            `json:"defaultText"`
	DefaultMarkdown string            `json:"defaultMarkdown"`
	HasCustom       bool              `json:"hasCustom"`
	ActiveID        *string           `json:"activeId,omitempty"`
	UpdatedAt       *string           `json:"updatedAt,omitempty"`
	ReplyTo         *string           `json:"replyTo,omitempty"`
	SenderName      *string           `json:"senderName,omitempty"`
	UnknownFields   []string          `json:"unknownFields,omitempty"`
}

type platformEmailTemplateVersionJSON struct {
	ID             string  `json:"id"`
	SlotID         string  `json:"slotId"`
	SourceMarkdown string  `json:"sourceMarkdown"`
	HTMLBody       string  `json:"htmlBody"`
	TextBody       *string `json:"textBody,omitempty"`
	ReplyTo        *string `json:"replyTo,omitempty"`
	SenderName     *string `json:"senderName,omitempty"`
	CreatedBy      *string `json:"createdBy,omitempty"`
	CreatedAt      string  `json:"createdAt"`
	IsActive       bool    `json:"isActive"`
}

func toSystemVersionJSON(v emailtemplates.SystemVersion) platformEmailTemplateVersionJSON {
	out := platformEmailTemplateVersionJSON{
		ID:             v.ID.String(),
		SlotID:         v.SlotID,
		SourceMarkdown: v.SourceMarkdown,
		HTMLBody:       v.HTMLBody,
		TextBody:       v.TextBody,
		ReplyTo:        v.ReplyTo,
		SenderName:     v.SenderName,
		CreatedAt:      v.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		IsActive:       v.IsActive,
	}
	if v.CreatedBy != nil {
		s := v.CreatedBy.String()
		out.CreatedBy = &s
	}
	return out
}

func (d Deps) platformEmailTemplateAccess(w http.ResponseWriter, r *http.Request) (actor uuid.UUID, ok bool) {
	if !d.effectiveConfig().EmailTemplateEditorEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Email template editor is not enabled.")
		return uuid.UUID{}, false
	}
	// Super-admin gate: same as platform settings (global rbac.manage).
	return d.adminRbacUser(w, r)
}

func (d Deps) handlePlatformEmailTemplatesList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.platformEmailTemplateAccess(w, r); !ok {
			return
		}
		slots, err := emailtemplates.ListSlots(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list email templates.")
			return
		}
		out := make([]platformEmailTemplateSlotJSON, 0, len(slots))
		for _, slot := range slots {
			item := platformEmailTemplateSlotJSON{
				ID:              slot.ID,
				Description:     slot.Description,
				MergeFields:     slot.MergeFields,
				DefaultHTML:     slot.DefaultHTML,
				DefaultText:     slot.DefaultText,
				DefaultMarkdown: slot.DefaultMarkdown,
			}
			active, err := emailtemplates.GetActiveSystem(r.Context(), d.Pool, slot.ID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list email templates.")
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

func (d Deps) handlePlatformEmailTemplateGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.platformEmailTemplateAccess(w, r); !ok {
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
			platformEmailTemplateSlotJSON
			Active *platformEmailTemplateVersionJSON `json:"active,omitempty"`
		}{
			platformEmailTemplateSlotJSON: platformEmailTemplateSlotJSON{
				ID:              slot.ID,
				Description:     slot.Description,
				MergeFields:     slot.MergeFields,
				DefaultHTML:     slot.DefaultHTML,
				DefaultText:     slot.DefaultText,
				DefaultMarkdown: slot.DefaultMarkdown,
			},
		}
		active, err := emailtemplates.GetActiveSystem(r.Context(), d.Pool, slotID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load email template.")
			return
		}
		if active != nil {
			v := toSystemVersionJSON(*active)
			resp.Active = &v
			resp.HasCustom = true
			id := active.ID.String()
			resp.ActiveID = &id
			ts := active.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
			resp.UpdatedAt = &ts
			resp.ReplyTo = active.ReplyTo
			resp.SenderName = active.SenderName
			resp.UnknownFields = emailtemplatesvc.ValidateUnknown(slot, active.HTMLBody, active.TextBody)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handlePlatformEmailTemplatePut() http.HandlerFunc {
	svc := d.emailTemplateService()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, ok := d.platformEmailTemplateAccess(w, r)
		if !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		var body struct {
			SourceMarkdown string  `json:"sourceMarkdown"`
			TextBody       *string `json:"textBody"`
			ReplyTo        *string `json:"replyTo"`
			SenderName     *string `json:"senderName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.SourceMarkdown) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "sourceMarkdown is required.")
			return
		}
		result, err := svc.SaveSystem(r.Context(), slotID, body.SourceMarkdown, body.TextBody, body.ReplyTo, body.SenderName, actor)
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
			platformEmailTemplateVersionJSON
			UnknownFields []string `json:"unknownFields,omitempty"`
		}{
			platformEmailTemplateVersionJSON: toSystemVersionJSON(result.Version),
			UnknownFields:                    result.UnknownFields,
		})
	}
}

func (d Deps) handlePlatformEmailTemplateHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.platformEmailTemplateAccess(w, r); !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		versions, err := emailtemplates.ListHistorySystem(r.Context(), d.Pool, slotID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load version history.")
			return
		}
		out := make([]platformEmailTemplateVersionJSON, 0, len(versions))
		for _, v := range versions {
			out = append(out, toSystemVersionJSON(v))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handlePlatformEmailTemplateRestore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.platformEmailTemplateAccess(w, r); !ok {
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
		version, err := emailtemplates.RestoreSystem(r.Context(), d.Pool, slotID, versionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Version not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to restore version.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toSystemVersionJSON(*version))
	}
}

func (d Deps) handlePlatformEmailTemplateReset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.platformEmailTemplateAccess(w, r); !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		if err := emailtemplates.ResetSystem(r.Context(), d.Pool, slotID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reset template.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handlePlatformEmailTemplateTest() http.HandlerFunc {
	svc := d.emailTemplateService()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, ok := d.platformEmailTemplateAccess(w, r)
		if !ok {
			return
		}
		slotID := strings.TrimSpace(chi.URLParam(r, "slotId"))
		if err := svc.SendSystemTestEmail(r.Context(), slotID, actor, nil); err != nil {
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

func (d Deps) handlePlatformEmailTemplatePreview() http.HandlerFunc {
	svc := d.emailTemplateService()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, ok := d.platformEmailTemplateAccess(w, r)
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
			TextBody       *string           `json:"textBody"`
			SampleData     map[string]string `json:"sampleData"`
		}
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		if strings.TrimSpace(body.SourceMarkdown) == "" {
			slot, slotErr := emailtemplates.GetSlot(r.Context(), d.Pool, slotID)
			if slotErr != nil || slot == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template slot not found.")
				return
			}
			active, activeErr := emailtemplates.GetActiveSystem(r.Context(), d.Pool, slotID)
			if activeErr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to preview template.")
				return
			}
			if active != nil && strings.TrimSpace(active.SourceMarkdown) != "" {
				body.SourceMarkdown = active.SourceMarkdown
				body.TextBody = active.TextBody
			} else {
				body.SourceMarkdown = slot.DefaultMarkdown
				t := slot.DefaultText
				body.TextBody = &t
			}
		}
		data := body.SampleData
		if len(data) == 0 {
			// Prefer actor org for realistic sample merge values when present.
			// SampleData tolerates uuid.Nil org.
			data, err = svc.SampleData(r.Context(), uuid.Nil, actor)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build preview data.")
				return
			}
		}
		preview := svc.Preview(body.SourceMarkdown, body.TextBody, data)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(preview)
	}
}
