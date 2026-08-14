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
	Prompt          string                       `json:"prompt"`
	Kind            string                       `json:"kind"`
	ExistingTitle   string                       `json:"existingTitle"`
	ExistingBody    string                       `json:"existingBodyMd"`
	Mode            string                       `json:"mode"`
	Description     string                       `json:"description"`
	PrimaryQuestion string                       `json:"primaryQuestion"`
	Cluster         string                       `json:"cluster"`
	Pillar          string                       `json:"pillar"`
	Keywords        []string                     `json:"keywords"`
	KnownPaths      []string                     `json:"knownPaths"`
	Findings        []marketingarticleai.Finding `json:"findings"`
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
		title := strings.TrimSpace(body.ExistingTitle)
		existing := strings.TrimSpace(body.ExistingBody)
		mode := strings.ToLower(strings.TrimSpace(body.Mode))
		metadataOnly := mode == "metadata"
		repairMode := mode == "repair"
		if repairMode {
			if title == "" && existing == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title or article body is required.")
				return
			}
			if len(body.Findings) == 0 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "At least one finding is required.")
				return
			}
		} else if !metadataOnly && prompt == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Prompt is required.")
			return
		} else if metadataOnly && title == "" && existing == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title or article body is required.")
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

		promptMaterial := prompt + title + existing
		if repairMode {
			promptMaterial += findingsPromptMaterial(body.Findings)
		}
		if !d.enforceAIGateway(w, r, actor, aigateway.FeatureMarketingArticleGeneration, model, promptMaterial) {
			return
		}
		gwDec := aigateway.Decision{
			UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, actor),
			OptInConfirmed: true,
		}

		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		var draft marketingarticleai.Draft
		var callMeta aiprovider.CallMeta
		if repairMode {
			draft, callMeta, err = marketingarticleai.RepairFromFindings(r.Context(), bound, model, marketingarticleai.RepairInput{
				Kind:            kind,
				Title:           title,
				BodyMD:          existing,
				Description:     body.Description,
				PrimaryQuestion: body.PrimaryQuestion,
				Cluster:         body.Cluster,
				Pillar:          body.Pillar,
				Keywords:        body.Keywords,
				KnownPaths:      body.KnownPaths,
				Findings:        body.Findings,
			})
		} else if metadataOnly {
			draft, callMeta, err = marketingarticleai.GenerateMetadataFromContent(
				r.Context(), bound, model, kind, title, existing,
			)
		} else {
			draft, callMeta, err = marketingarticleai.GenerateFromPrompt(
				r.Context(), bound, model, marketingarticleai.DefaultSystemPrompt, kind, prompt, body.ExistingTitle, body.ExistingBody,
			)
		}
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

func findingsPromptMaterial(findings []marketingarticleai.Finding) string {
	var b strings.Builder
	for _, finding := range findings {
		b.WriteString(finding.Rule)
		b.WriteString(finding.Message)
		b.WriteString(finding.Path)
	}
	return b.String()
}
