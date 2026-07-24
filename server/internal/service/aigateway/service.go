// Package aigateway enforces AI usage disclosure controls before external model calls (plan 10.17).
package aigateway

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	pkgai "github.com/lextures/lextures/server/internal/aidisclosure"
	repo "github.com/lextures/lextures/server/internal/repos/aidisclosure"
	gdprrepo "github.com/lextures/lextures/server/internal/repos/gdpr"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	coppaservice "github.com/lextures/lextures/server/internal/service/coppa"
)

const (
	ProviderOpenRouter  = "openrouter"
	ProviderAnthropic   = "anthropic"
	ProviderOpenAI      = "openai"
	ProviderAzureOpenAI = "azure_openai"
	ProviderBedrock     = "bedrock"
	ProviderVertex      = "vertex"
	ReadPermission      = "compliance:ai:read:*"

	FeatureAITutor                    = "ai_tutor"
	FeatureModulesAIAssistant         = "modules_ai_assistant"
	FeatureRAGNotebook                = "rag_notebook"
	FeatureSyllabusGeneration         = "syllabus_generation"
	FeatureOutcomesExtraction         = "outcomes_extraction"
	FeatureBadgesExtraction           = "badges_extraction"
	FeatureTranslation                = "translation"
	FeatureQuizGeneration             = "quiz_generation"
	FeatureLiveQuizKitGeneration      = "live_quiz_kit_generation"
	FeatureReadingLevelSimplification = "reading_level_simplification"
	FeatureContentTranslation         = "content_translation"
	FeatureAltTextSuggestion          = "alt_text_suggestion"
	FeatureVibeGeneration             = "vibe_generation"
	FeatureGraderAgent                = "grader_agent"
	FeatureLessonGeneration           = "lesson_generation"
	FeatureAIStudyBuddy               = "ai_study_buddy"
	FeatureReportCardComment          = "report_card_comment"
)

// BlockReason explains why a call was blocked.
type BlockReason string

const (
	BlockNone          BlockReason = ""
	BlockOptOut        BlockReason = "opt_out"
	BlockCoppaAI       BlockReason = "coppa_ai"
	BlockGDPRConsent   BlockReason = "gdpr_consent"
	BlockTenantFeature BlockReason = "tenant_feature"
	BlockTenantModel   BlockReason = "tenant_model"
	BlockServiceError  BlockReason = "service_error"
)

// Decision is the outcome of an AI gateway check.
type Decision struct {
	Allowed        bool
	Reason         BlockReason
	UserIDHash     string
	OptInConfirmed bool
	LogBlocked     bool
}

// Config holds runtime flags for gateway evaluation.
type Config struct {
	DisclosureEnabled bool
	GDPRModuleEnabled bool
	CoppaEnabled      bool
	HMACSecret        string
}

type optOutCacheEntry struct {
	optedOut bool
	expires  time.Time
}

var optOutCache sync.Map

const optOutCacheTTL = 30 * time.Second

// UserIDHash delegates to the shared aidisclosure helper.
func UserIDHash(secret string, userID uuid.UUID) string {
	return pkgai.UserIDHash(secret, userID)
}

// ContentHash delegates to the shared aidisclosure helper.
func ContentHash(content string) string {
	return pkgai.ContentHash(content)
}

// Evaluate checks whether an AI call may proceed. On any policy DB error, returns blocked (fail closed).
func Evaluate(ctx context.Context, pool *pgxpool.Pool, cfg Config, userID uuid.UUID, orgID *uuid.UUID, feature, modelID, contentHash string) (Decision, error) {
	hash := pkgai.UserIDHash(cfg.HMACSecret, userID)
	dec := Decision{UserIDHash: hash, Allowed: true}

	if !cfg.DisclosureEnabled {
		dec.OptInConfirmed = true
		return dec, nil
	}

	optedOut, err := cachedOptOut(ctx, pool, userID)
	if err != nil {
		dec.Allowed = false
		dec.Reason = BlockServiceError
		dec.LogBlocked = true
		return dec, err
	}
	if optedOut {
		dec.Allowed = false
		dec.Reason = BlockOptOut
		dec.LogBlocked = true
		return dec, nil
	}

	if cfg.CoppaEnabled {
		blocked, err := coppaservice.IsCoppaAIBlocked(ctx, pool, userID)
		if err != nil {
			dec.Allowed = false
			dec.Reason = BlockServiceError
			dec.LogBlocked = true
			return dec, err
		}
		if blocked {
			dec.Allowed = false
			dec.Reason = BlockCoppaAI
			dec.LogBlocked = true
			return dec, nil
		}
	}

	if cfg.GDPRModuleEnabled {
		active, err := gdprrepo.HasActiveConsent(ctx, pool, userID, "ai_processing")
		if err != nil {
			dec.Allowed = false
			dec.Reason = BlockServiceError
			dec.LogBlocked = true
			return dec, err
		}
		if !active {
			dec.Allowed = false
			dec.Reason = BlockGDPRConsent
			dec.LogBlocked = true
			return dec, nil
		}
		dec.OptInConfirmed = true
	} else {
		dec.OptInConfirmed = true
	}

	if orgID != nil {
		tc, err := repo.GetTenantConfig(ctx, pool, *orgID)
		if err != nil {
			dec.Allowed = false
			dec.Reason = BlockServiceError
			dec.LogBlocked = true
			return dec, err
		}
		if tc != nil {
			if disabled, ok := tc.FeaturesEnabled[feature]; ok && !disabled {
				dec.Allowed = false
				dec.Reason = BlockTenantFeature
				dec.LogBlocked = true
				return dec, nil
			}
			if len(tc.AllowedModels) > 0 && modelID != "" && !modelAllowed(tc.AllowedModels, modelID) {
				dec.Allowed = false
				dec.Reason = BlockTenantModel
				dec.LogBlocked = true
				return dec, nil
			}
		}
	}

	return dec, nil
}

func modelAllowed(allowed []string, modelID string) bool {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	for _, m := range allowed {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if strings.EqualFold(m, modelID) {
			return true
		}
		// Match aliases against resolved provider ids and dual-read OpenRouter ids (AP.3 FR-8).
		if modelAllowedViaAlias(m, modelID) {
			return true
		}
	}
	return false
}

func modelAllowedViaAlias(allowedEntry, modelID string) bool {
	// allowedEntry is an alias that resolves to modelID for any provider.
	for _, p := range aiprovider.ListProviders() {
		if id, err := aiprovider.ResolveModelID(allowedEntry, p); err == nil && strings.EqualFold(id, modelID) {
			return true
		}
		if id, err := aiprovider.ResolveModelID(modelID, p); err == nil && strings.EqualFold(id, allowedEntry) {
			return true
		}
	}
	// Both sides dual-read to the same alias.
	if a, ok := aiprovider.AliasForOpenRouterID(allowedEntry); ok {
		if b, ok2 := aiprovider.AliasForOpenRouterID(modelID); ok2 && a == b {
			return true
		}
		if strings.EqualFold(string(a), modelID) {
			return true
		}
	}
	if a, ok := aiprovider.AliasForOpenRouterID(modelID); ok && strings.EqualFold(string(a), allowedEntry) {
		return true
	}
	return false
}

func cachedOptOut(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	key := userID.String()
	if v, ok := optOutCache.Load(key); ok {
		e := v.(optOutCacheEntry)
		if time.Now().Before(e.expires) {
			return e.optedOut, nil
		}
	}
	optedOut, err := repo.GetOptOut(ctx, pool, userID)
	if err != nil {
		return false, err
	}
	optOutCache.Store(key, optOutCacheEntry{optedOut: optedOut, expires: time.Now().Add(optOutCacheTTL)})
	return optedOut, nil
}

// InvalidateOptOutCache clears cached opt-out state after a user updates settings.
func InvalidateOptOutCache(userID uuid.UUID) {
	optOutCache.Delete(userID.String())
}

// LogInference records an allowed or blocked inference attempt (best-effort for allowed path after success too).
func LogInference(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, dec Decision, feature, modelID, provider, contentHash string, blocked bool) error {
	if pool == nil {
		return errors.New("aigateway: nil pool")
	}
	if modelID == "" {
		modelID = "unknown"
	}
	if provider == "" {
		provider = "unknown"
	}
	if contentHash == "" {
		contentHash = pkgai.ContentHash("")
	}
	return repo.InsertLog(ctx, pool, repo.InferenceLogEntry{
		OrgID:          orgID,
		UserIDHash:     dec.UserIDHash,
		FeatureName:    feature,
		ModelID:        modelID,
		Provider:       provider,
		ContentHash:    contentHash,
		OptInConfirmed: dec.OptInConfirmed,
		Blocked:        blocked,
	})
}

// BlockMessage returns a user-facing error string for HTTP responses.
func BlockMessage(reason BlockReason) string {
	switch reason {
	case BlockOptOut:
		return "AI processing is disabled for this account."
	case BlockCoppaAI:
		return "AI features require parental consent for this account."
	case BlockGDPRConsent:
		return "AI processing requires your consent. Visit the Privacy Center to grant consent for AI features."
	case BlockTenantFeature:
		return "This AI feature is disabled by your organization."
	case BlockTenantModel:
		return "This AI model is not approved by your organization."
	case BlockServiceError:
		return "AI processing is temporarily unavailable."
	default:
		return "AI processing is not permitted."
	}
}
