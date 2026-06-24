package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/aiusage"
	"github.com/lextures/lextures/server/internal/repos/organization"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/openrouter"
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
	if provider == "" {
		provider = aigateway.ProviderOpenRouter
	}
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
	usage := openrouter.UsageInfo{
		PromptTokens:     result.Usage.PromptTokens,
		CompletionTokens: result.Usage.CompletionTokens,
		TotalTokens:      result.Usage.TotalTokens,
		CostUSD:          result.Usage.CostUSD,
	}
	_ = aiusage.Insert(ctx, d.Pool, aiusage.EntryFromProviderUsage(
		userID, meta.CourseID, meta.Feature, meta.Model, string(callMeta.Provider), usage, succeeded,
	))
}