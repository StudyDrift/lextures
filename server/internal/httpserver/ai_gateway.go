package httpserver

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
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

// logAIInferenceAllowed records a successful (non-blocked) inference after the external call completes.
func (d Deps) logAIInferenceAllowed(r *http.Request, userID uuid.UUID, feature, modelID, contentForHash string, dec aigateway.Decision) {
	if d.Pool == nil || !d.effectiveConfig().AiDisclosureEnabled {
		return
	}
	ctx := r.Context()
	var orgID *uuid.UUID
	if oid, err := organization.OrgIDForUser(ctx, d.Pool, userID); err == nil {
		orgID = &oid
	}
	if dec.UserIDHash == "" {
		dec.UserIDHash = aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, userID)
		dec.OptInConfirmed = true
	}
	_ = aigateway.LogInference(ctx, d.Pool, orgID, dec, feature, modelID, aigateway.ProviderOpenRouter, aigateway.ContentHash(contentForHash), false)
}
