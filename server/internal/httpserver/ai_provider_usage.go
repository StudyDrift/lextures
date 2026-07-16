package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/aiusage"
	"github.com/lextures/lextures/server/internal/repos/organization"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

func (d Deps) logAIInferenceAllowedWithProvider(
	r *http.Request,
	userID uuid.UUID,
	feature, modelID, provider, contentForHash string,
	dec aigateway.Decision,
) {
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
	// Empty provider → LogInference stores "unknown" (AP.6 FR-7); do not invent openrouter.
	_ = aigateway.LogInference(ctx, d.Pool, orgID, dec, feature, modelID, provider, aigateway.ContentHash(contentForHash), false)
}

func (d Deps) recordAIProviderUsage(
	ctx context.Context,
	meta AIUsageMeta,
	callMeta aiprovider.CallMeta,
	succeeded bool,
) {
	if d.Pool == nil || !callMeta.Usage.HasData() {
		return
	}
	d.recordAIProviderResult(ctx, meta, callMeta, aiprovider.ChatResult{Usage: callMeta.Usage}, succeeded)
}

func (d Deps) recordAIProviderResult(ctx context.Context, meta AIUsageMeta, callMeta aiprovider.CallMeta, result aiprovider.ChatResult, succeeded bool) {
	if d.Pool == nil || !result.Usage.HasData() {
		return
	}
	var userID *uuid.UUID
	if meta.UserID != uuid.Nil {
		uid := meta.UserID
		userID = &uid
	}
	_ = aiusage.Insert(ctx, d.Pool, aiusage.EntryFromCallMeta(
		userID, meta.CourseID, meta.Feature, callMeta, result.Usage, succeeded,
	))
}
