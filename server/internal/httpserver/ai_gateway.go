package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/aiusage"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/organization"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

func (d Deps) aiGatewayConfig() aigateway.Config {
	cfg := d.effectiveConfig()
	secret := cfg.JWTSecret
	if secret == "" {
		secret = "dev-ai-gateway-hmac"
	}
	return aigateway.Config{
		DisclosureEnabled: cfg.AiDisclosureEnabled,
		GDPRModuleEnabled: cfg.GDPRModuleEnabled,
		CoppaEnabled:      cfg.CoppaWorkflowEnabled,
		HMACSecret:        secret,
	}
}

// enforceAIGateway checks opt-out, COPPA, GDPR consent, and tenant policy before an AI call.
// Returns false after writing an HTTP error response. Always logs the attempt when disclosure is enabled.
func (d Deps) enforceAIGateway(
	w http.ResponseWriter,
	r *http.Request,
	userID uuid.UUID,
	feature, modelID, contentForHash string,
) bool {
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, aigateway.BlockMessage(aigateway.BlockServiceError))
		return false
	}
	ctx := r.Context()
	var orgID *uuid.UUID
	if oid, err := organization.OrgIDForUser(ctx, d.Pool, userID); err == nil {
		orgID = &oid
	}
	hash := aigateway.ContentHash(contentForHash)
	dec, err := aigateway.Evaluate(ctx, d.Pool, d.aiGatewayConfig(), userID, orgID, feature, modelID, hash)
	if err != nil {
		_ = aigateway.LogInference(ctx, d.Pool, orgID, dec, feature, modelID, aigateway.ProviderOpenRouter, hash, true)
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeAiProcessingDisabled, aigateway.BlockMessage(aigateway.BlockServiceError))
		return false
	}
	if !dec.Allowed {
		_ = aigateway.LogInference(ctx, d.Pool, orgID, dec, feature, modelID, aigateway.ProviderOpenRouter, hash, true)
		code := apierr.CodeAiProcessingDisabled
		if dec.Reason == aigateway.BlockTenantFeature || dec.Reason == aigateway.BlockTenantModel {
			code = apierr.CodeTenantAIPolicyDisabled
		}
		apierr.WriteJSON(w, http.StatusForbidden, code, aigateway.BlockMessage(dec.Reason))
		return false
	}
	return true
}

// evaluateAIGatewayBlock returns a user-facing block message when AI processing is disallowed.
func (d Deps) evaluateAIGatewayBlock(ctx context.Context, userID uuid.UUID, feature, modelID, contentForHash string) (string, bool) {
	if d.Pool == nil {
		return aigateway.BlockMessage(aigateway.BlockServiceError), true
	}
	var orgID *uuid.UUID
	if oid, err := organization.OrgIDForUser(ctx, d.Pool, userID); err == nil {
		orgID = &oid
	}
	hash := aigateway.ContentHash(contentForHash)
	dec, err := aigateway.Evaluate(ctx, d.Pool, d.aiGatewayConfig(), userID, orgID, feature, modelID, hash)
	if err != nil {
		_ = aigateway.LogInference(ctx, d.Pool, orgID, dec, feature, modelID, aigateway.ProviderOpenRouter, hash, true)
		return aigateway.BlockMessage(aigateway.BlockServiceError), true
	}
	if !dec.Allowed {
		_ = aigateway.LogInference(ctx, d.Pool, orgID, dec, feature, modelID, aigateway.ProviderOpenRouter, hash, true)
		return aigateway.BlockMessage(dec.Reason), true
	}
	return "", false
}

// AIUsageMeta identifies who triggered an AI call for analytics.ai_usage_log.
type AIUsageMeta struct {
	UserID     uuid.UUID
	CourseID   *uuid.UUID
	CourseCode string
	Feature    string
	Model      string
}

// recordAIUsage appends token/cost usage for Intelligence reports (best-effort).
// Prefer recordAIProviderResult when CallMeta is available so provider is accurate (AP.6).
func (d Deps) recordAIUsage(ctx context.Context, meta AIUsageMeta, usage openrouter.UsageInfo, succeeded bool) {
	if d.Pool == nil || !usage.HasData() {
		return
	}
	var userID *uuid.UUID
	if meta.UserID != uuid.Nil {
		uid := meta.UserID
		userID = &uid
	}
	var courseID *uuid.UUID
	if meta.CourseID != nil {
		courseID = meta.CourseID
	} else if code := strings.TrimSpace(meta.CourseCode); code != "" {
		if row, err := course.GetPublicByCourseCode(ctx, d.Pool, code); err == nil && row != nil {
			if cid, perr := uuid.Parse(row.ID); perr == nil {
				courseID = &cid
			}
		}
	}
	_ = aiusage.Insert(ctx, d.Pool, aiusage.EntryFromUsage(userID, courseID, meta.Feature, meta.Model, usage, succeeded))
}

// logAIInferenceAllowed records a successful (non-blocked) inference after the external call completes.
// Prefer logAIInferenceAllowedWithProvider when the backend provider is known (AP.6 FR-7).
func (d Deps) logAIInferenceAllowed(r *http.Request, userID uuid.UUID, feature, modelID, contentForHash string, dec aigateway.Decision) {
	d.logAIInferenceAllowedWithProvider(r, userID, feature, modelID, "", contentForHash, dec)
}
