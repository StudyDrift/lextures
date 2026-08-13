package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	userai "github.com/lextures/lextures/server/internal/repos/user"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/marketingarticleai"
)

func (d Deps) registerMarketingArticleAIRoutes(r chi.Router) {
	r.Post("/api/v1/admin/marketing/articles/generate", d.handleGenerateMarketingArticle())
}

type generateMarketingArticleRequest struct {
	Prompt        string `json:"prompt"`
	Kind          string `json:"kind"`
	ExistingTitle string `json:"existingTitle"`
	ExistingBody  string `json:"existingBodyMd"`
}

// handleGenerateMarketingArticle is POST /api/v1/admin/marketing/articles/generate.
// Returns a draft article only; does not persist.
func (d Deps) handleGenerateMarketingArticle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAuthor)
		if !ok {
			return
		}
		var body generateMarketingArticleRequest
		if !readMarketingJSON(w, r, &body) {
			return
		}
		prompt := strings.TrimSpace(body.Prompt)
		if prompt == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Prompt is required.")
			return
		}
		kind := body.Kind
		if kind != "doc" {
			kind = "blog"
		}

		orgID := d.orgIDPtrForUser(r.Context(), actor)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}

		model, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, actor)
		if err != nil {
			model = userai.DefaultCourseSetupModelID
		}

		promptMaterial := prompt + strings.TrimSpace(body.ExistingTitle) + strings.TrimSpace(body.ExistingBody)
		if !d.enforceAIGateway(w, r, actor, aigateway.FeatureMarketingArticleGeneration, model, promptMaterial) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, actor),
			OptInConfirmed: true,
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		draft, callMeta, err := marketingarticleai.GenerateFromPrompt(
			r.Context(), bound, model, marketingarticleai.DefaultSystemPrompt, kind, prompt, body.ExistingTitle, body.ExistingBody,
		)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "parse marketing article JSON") {
				writeAIGenerationFailed(w, r, "AI did not return a valid article draft: "+msg, err)
				return
			}
			if strings.Contains(msg, "too long") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
				return
			}
			writeAIGenerationFailed(w, r, "AI generation failed: "+msg, err)
			return
		}

		d.logAIInferenceAllowedWithProvider(r, actor, aigateway.FeatureMarketingArticleGeneration, model, string(callMeta.Provider), promptMaterial, gwDec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{
			UserID: actor, Feature: aigateway.FeatureMarketingArticleGeneration, Model: model,
		}, callMeta, true)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(draft)
	}
}
