package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/aidisclosure"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	pkgai "github.com/lextures/lextures/server/internal/aidisclosure"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
)

func (d Deps) aiDisclosureEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().AiDisclosureEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "AI disclosure controls are not enabled.")
		return false
	}
	return true
}

func (d Deps) registerAIDisclosureRoutes(r chi.Router) {
	r.Get("/api/v1/public/ai-disclosure", d.handlePublicAIDisclosure())
	r.Get("/api/v1/settings/ai-opt-out", d.handleGetAIOptOut())
	r.Put("/api/v1/settings/ai-opt-out", d.handlePutAIOptOut())
	r.Get("/api/v1/settings/ai-disclosure/acknowledgements", d.handleListAIFeatureAcks())
	r.Post("/api/v1/settings/ai-disclosure/acknowledgements", d.handlePostAIFeatureAck())
	r.Get("/api/v1/admin/ai-config", d.handleGetAdminAIConfig())
	r.Put("/api/v1/admin/ai-config", d.handlePutAdminAIConfig())
	r.Get("/api/v1/compliance/ai-inference-log", d.handleGetAIInferenceLog())
}

func (d Deps) handlePublicAIDisclosure() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(pkgai.PublicDisclosureJSON)
	}
}

func (d Deps) handleGetAIOptOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiDisclosureEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		optedOut, err := repo.GetOptOut(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load AI settings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"aiProcessingOptOut": optedOut,
			"disclosureUrl":      "/ai-disclosure",
		})
	}
}

func (d Deps) handlePutAIOptOut() http.HandlerFunc {
	type body struct {
		OptOut *bool `json:"aiProcessingOptOut"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiDisclosureEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.OptOut == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "aiProcessingOptOut is required.")
			return
		}
		if err := repo.SetOptOut(r.Context(), d.Pool, userID, *b.OptOut); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update AI settings.")
			return
		}
		aigateway.InvalidateOptOutCache(userID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"aiProcessingOptOut": *b.OptOut})
	}
}

func (d Deps) handleListAIFeatureAcks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiDisclosureEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		keys, err := repo.ListFeatureAcks(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load acknowledgements.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"features": keys})
	}
}

func (d Deps) handlePostAIFeatureAck() http.HandlerFunc {
	type body struct {
		FeatureKey string `json:"featureKey"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiDisclosureEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil || strings.TrimSpace(b.FeatureKey) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "featureKey is required.")
			return
		}
		if err := repo.AcknowledgeFeature(r.Context(), d.Pool, userID, strings.TrimSpace(b.FeatureKey)); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save acknowledgement.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) requireOrgAdminForAIConfig(w http.ResponseWriter, r *http.Request) (userID, orgID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	oid, err := organization.OrgIDForUser(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not resolve organization.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	isAdmin, err := orgroles.UserHasRole(r.Context(), d.Pool, uid, oid, orgroles.RoleOrgAdmin)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	if !isAdmin {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Organization admin role required.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return uid, oid, true
}

func (d Deps) handleGetAdminAIConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiDisclosureEnabled(w) {
			return
		}
		_, orgID, ok := d.requireOrgAdminForAIConfig(w, r)
		if !ok {
			return
		}
		tc, err := repo.GetTenantConfig(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load AI config.")
			return
		}
		resp := map[string]any{
			"orgId":           orgID.String(),
			"featuresEnabled": map[string]bool{},
			"allowedModels":   []string(nil),
		}
		if tc != nil {
			resp["featuresEnabled"] = tc.FeaturesEnabled
			if tc.AllowedModels != nil {
				resp["allowedModels"] = tc.AllowedModels
			}
			resp["updatedAt"] = tc.UpdatedAt.UTC().Format(time.RFC3339)
			if tc.UpdatedBy != nil {
				resp["updatedBy"] = tc.UpdatedBy.String()
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handlePutAdminAIConfig() http.HandlerFunc {
	type body struct {
		FeaturesEnabled map[string]bool `json:"featuresEnabled"`
		AllowedModels   []string        `json:"allowedModels"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiDisclosureEnabled(w) {
			return
		}
		actorID, orgID, ok := d.requireOrgAdminForAIConfig(w, r)
		if !ok {
			return
		}
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if b.FeaturesEnabled == nil {
			b.FeaturesEnabled = map[string]bool{}
		}
		before, _ := repo.GetTenantConfig(r.Context(), d.Pool, orgID)
		if err := repo.UpsertTenantConfig(r.Context(), d.Pool, orgID, actorID, b.FeaturesEnabled, b.AllowedModels); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save AI config.")
			return
		}
		afterJSON, _ := json.Marshal(b)
		beforeJSON, _ := json.Marshal(before)
		orgPtr := &orgID
		_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
			OrgID:       orgPtr,
			EventType:   auditservice.EventAIConfigChange,
			ActorID:     actorID,
			TargetType:  aiConfigTargetType(),
			TargetID:    &orgID,
			BeforeValue: beforeJSON,
			AfterValue:  afterJSON,
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"orgId":           orgID.String(),
			"featuresEnabled": b.FeaturesEnabled,
			"allowedModels":   b.AllowedModels,
		})
	}
}

func (d Deps) handleGetAIInferenceLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiDisclosureEnabled(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		can, err := rbac.UserHasPermission(r.Context(), d.Pool, uid, aigateway.ReadPermission)
		if err != nil || !can {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view the AI inference log.")
			return
		}
		q := r.URL.Query()
		var orgID *uuid.UUID
		if s := strings.TrimSpace(q.Get("orgId")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
				return
			}
			orgID = &id
		}
		userHash := strings.TrimSpace(q.Get("userIdHash"))
		rows, err := repo.QueryLogs(r.Context(), d.Pool, orgID, userHash, 200)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load inference log.")
			return
		}
		type row struct {
			ID             string  `json:"id"`
			OrgID          *string `json:"orgId,omitempty"`
			UserIDHash     string  `json:"userIdHash"`
			FeatureName    string  `json:"featureName"`
			ModelID        string  `json:"modelId"`
			Provider       string  `json:"provider"`
			ContentHash    string  `json:"contentHash"`
			OptInConfirmed bool    `json:"optInConfirmed"`
			Blocked        bool    `json:"blocked"`
			Timestamp      string  `json:"timestamp"`
		}
		out := make([]row, 0, len(rows))
		for _, e := range rows {
			var orgStr *string
			if e.OrgID != nil {
				s := e.OrgID.String()
				orgStr = &s
			}
			out = append(out, row{
				ID:             e.ID.String(),
				OrgID:          orgStr,
				UserIDHash:     e.UserIDHash,
				FeatureName:    e.FeatureName,
				ModelID:        e.ModelID,
				Provider:       e.Provider,
				ContentHash:    e.ContentHash,
				OptInConfirmed: e.OptInConfirmed,
				Blocked:        e.Blocked,
				Timestamp:      e.Timestamp.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})
	}
}

func aiConfigTargetType() *string {
	s := "tenant_ai_config"
	return &s
}
