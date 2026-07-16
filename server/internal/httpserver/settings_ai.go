package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/aiprovidercreds"
	"github.com/lextures/lextures/server/internal/repos/platformconfig"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// handleListAIModels is GET /api/v1/settings/ai/models?provider=&kind=text|image|vision
func (d Deps) handleListAIModels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		kind, err := aiprovider.ParseCatalogKind(r.URL.Query().Get("kind"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid kind (use text, image, or vision).")
			return
		}
		provider, configured, opts := d.resolveCatalogRequest(r.Context(), r.URL.Query().Get("provider"))
		models, err := aiprovider.ListCatalog(r.Context(), provider, kind, opts)
		if err != nil {
			// OpenRouter live failure should still surface; curated providers never fail solely for missing OpenRouter (AC-1).
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"Could not load models. Try again. ("+err.Error()+")")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"configured": configured,
			"provider":   string(provider),
			"models":     models,
		})
	}
}

// resolveCatalogRequest picks provider (query or active platform default) and catalog options.
func (d Deps) resolveCatalogRequest(ctx context.Context, providerParam string) (aiprovider.ProviderName, bool, aiprovider.CatalogOptions) {
	opts := aiprovider.CatalogOptions{}
	if p, ok := aiprovider.NormalizeProviderName(providerParam); ok && p != aiprovider.ProviderDryRun {
		configured := d.providerConfigured(ctx, p)
		if key := d.platformAPIKeyForCatalog(ctx, p); key != "" {
			opts.APIKey = key
		}
		return p, configured, opts
	}
	active := d.activePlatformProvider(ctx)
	configured := d.providerConfigured(ctx, active)
	if key := d.platformAPIKeyForCatalog(ctx, active); key != "" {
		opts.APIKey = key
	}
	return active, configured, opts
}

func (d Deps) activePlatformProvider(ctx context.Context) aiprovider.ProviderName {
	if d.Pool != nil {
		creds, err := aiprovidercreds.ListByScope(ctx, d.Pool, aiprovidercreds.ScopePlatform, nil)
		if err == nil {
			configured := map[string]bool{}
			for _, c := range creds {
				if c.Enabled && c.SecretConfigured {
					configured[c.Provider] = true
				}
			}
			cfg := d.effectiveConfig()
			if strings.TrimSpace(cfg.OpenRouterAPIKey) != "" {
				configured[string(aiprovider.ProviderOpenRouter)] = true
			}
			for _, name := range aiprovider.ListProviders() {
				if configured[string(name)] {
					return name
				}
			}
		}
	}
	cfg := d.effectiveConfig()
	if strings.TrimSpace(cfg.OpenRouterAPIKey) != "" {
		return aiprovider.ProviderOpenRouter
	}
	return aiprovider.ProviderOpenRouter
}

func (d Deps) providerConfigured(ctx context.Context, provider aiprovider.ProviderName) bool {
	if provider == aiprovider.ProviderOpenRouter && d.openRouterClient() != nil {
		return true
	}
	if d.Pool == nil {
		return false
	}
	ok, err := aiprovidercreds.SecretConfigured(ctx, d.Pool, aiprovidercreds.ScopePlatform, nil, string(provider))
	if err == nil && ok {
		return true
	}
	if provider == aiprovider.ProviderOpenRouter {
		return strings.TrimSpace(d.effectiveConfig().OpenRouterAPIKey) != ""
	}
	return false
}

func (d Deps) platformAPIKeyForCatalog(ctx context.Context, provider aiprovider.ProviderName) string {
	cfg := d.effectiveConfig()
	if d.Pool == nil {
		if provider == aiprovider.ProviderOpenRouter {
			return strings.TrimSpace(cfg.OpenRouterAPIKey)
		}
		return ""
	}
	key, _, enabled, err := aiprovidercreds.ResolveAPIKey(
		ctx, d.Pool, aiprovidercreds.ScopePlatform, nil, string(provider), cfg.PlatformSecretsKey, cfg.OpenRouterAPIKey,
	)
	if err != nil || !enabled {
		return ""
	}
	return key
}

// handleGetSettingsAI is GET /api/v1/settings/ai
func (d Deps) handleGetSettingsAI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		uid, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		img, err := user.GetImageModelID(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI settings.")
			return
		}
		course, err := user.GetCourseSetupModelID(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI settings.")
			return
		}
		flashcards, err := user.GetNotebookFlashcardsModelID(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI settings.")
			return
		}
		vibe, err := user.GetVibeActivityModelID(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI settings.")
			return
		}
		grader, err := user.GetGraderAgentModelID(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI settings.")
			return
		}
		cfg := d.effectiveConfig()
		active := d.activePlatformProvider(r.Context())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"imageModelId":              img,
			"courseSetupModelId":        course,
			"notebookFlashcardsModelId": flashcards,
			"vibeActivityModelId":       vibe,
			"graderAgentModelId":        grader,
			"activeProvider":            string(active),
			"openRouterApiKey":          maskSecret(cfg.OpenRouterAPIKey),
		})
	}
}

type putSettingsAIBody struct {
	ImageModelID              string  `json:"imageModelId"`
	CourseSetupModelID        string  `json:"courseSetupModelId"`
	NotebookFlashcardsModelID string  `json:"notebookFlashcardsModelId"`
	VibeActivityModelID       string  `json:"vibeActivityModelId"`
	GraderAgentModelID        string  `json:"graderAgentModelId"`
	// OpenRouterAPIKey is deprecated (AP.9); prefer PUT /api/v1/settings/ai/providers/openrouter.
	OpenRouterAPIKey *string `json:"openRouterApiKey"`
	// ClearOpenRouterAPIKey is deprecated (AP.9); prefer DELETE /api/v1/settings/ai/providers/openrouter.
	ClearOpenRouterAPIKey bool `json:"clearOpenRouterApiKey"`
}

// handlePutSettingsAI is PUT /api/v1/settings/ai
func (d Deps) handlePutSettingsAI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		uid, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var in putSettingsAIBody
		if err := json.Unmarshal(b, &in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		img := strings.TrimSpace(in.ImageModelID)
		if img == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Choose an image model.")
			return
		}
		course := strings.TrimSpace(in.CourseSetupModelID)
		if course == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Choose a course setup model.")
			return
		}
		flashcards := strings.TrimSpace(in.NotebookFlashcardsModelID)
		if flashcards == "" {
			flashcards = user.DefaultNotebookFlashcardsModelID
		}
		vibe := strings.TrimSpace(in.VibeActivityModelID)
		if vibe == "" {
			vibe = user.DefaultVibeActivityModelID
		}
		grader := strings.TrimSpace(in.GraderAgentModelID)
		if grader == "" {
			grader = user.DefaultGraderAgentModelID
		}
		if err := d.applyOpenRouterAPIKeyUpdate(r.Context(), in.OpenRouterAPIKey, in.ClearOpenRouterAPIKey); err != nil {
			if err == errOpenRouterAPIKeyConflict {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Cannot set openRouterApiKey and clearOpenRouterApiKey together.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save OpenRouter API key.")
			return
		}

		imgOut, courseOut, flashcardsOut, vibeOut, graderOut, err := user.UpsertAISettings(r.Context(), d.Pool, uid, img, course, flashcards, vibe, grader)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save AI settings.")
			return
		}
		cfg := d.effectiveConfig()
		active := d.activePlatformProvider(r.Context())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"imageModelId":              imgOut,
			"courseSetupModelId":        courseOut,
			"notebookFlashcardsModelId": flashcardsOut,
			"vibeActivityModelId":       vibeOut,
			"graderAgentModelId":        graderOut,
			"activeProvider":            string(active),
			"openRouterApiKey":          maskSecret(cfg.OpenRouterAPIKey),
		})
	}
}

var errOpenRouterAPIKeyConflict = errOpenRouterKeyConflict{}

type errOpenRouterKeyConflict struct{}

func (errOpenRouterKeyConflict) Error() string {
	return "openrouter api key conflict"
}

func (d Deps) applyOpenRouterAPIKeyUpdate(ctx context.Context, key *string, clear bool) error {
	if d.Pool == nil {
		return nil
	}
	if key == nil && !clear {
		return nil
	}

	wr := &platformconfig.Write{}
	if key != nil {
		s := strings.TrimSpace(*key)
		if s != "" && s != placeholderSecretResponse {
			wr.OpenRouterAPIKey = &s
		}
	}
	if clear && wr.OpenRouterAPIKey != nil && strings.TrimSpace(*wr.OpenRouterAPIKey) != "" {
		return errOpenRouterAPIKeyConflict
	}
	if clear {
		if err := platformconfig.ClearOpenRouterAPIKey(ctx, d.Pool); err != nil {
			return err
		}
		_ = aiprovidercreds.ClearSecret(ctx, d.Pool, aiprovidercreds.ScopePlatform, nil, string(aiprovider.ProviderOpenRouter))
	}
	if wr.OpenRouterAPIKey == nil {
		if clear {
			d.reloadPlatformAIClients()
		}
		return nil
	}
	dbRow, err := platformconfig.Upsert(ctx, d.Pool, wr)
	if err != nil {
		return err
	}
	// Dual-write into encrypted multi-provider store when secrets key is configured (AP.2).
	if secretsKey := d.effectiveConfig().PlatformSecretsKey; len(secretsKey) == 32 {
		_ = aiprovidercreds.Upsert(ctx, d.Pool, aiprovidercreds.ScopePlatform, nil, string(aiprovider.ProviderOpenRouter), aiprovidercreds.UpsertInput{})
		_ = aiprovidercreds.StoreSecret(ctx, d.Pool, aiprovidercreds.ScopePlatform, nil, string(aiprovider.ProviderOpenRouter), secretsKey, *wr.OpenRouterAPIKey)
		aiprovider.RecordCredentialConfigured(aiprovidercreds.ScopePlatform, string(aiprovider.ProviderOpenRouter), true)
	}
	merged := platformconfig.Merge(d.Config, dbRow)
	if err := merged.Validate(); err != nil {
		return err
	}
	if d.Platform != nil {
		d.Platform.Reload(merged)
	}
	return nil
}
