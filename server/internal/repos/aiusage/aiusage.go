// Package aiusage persists AI token/cost rows for Intelligence reports (multi-provider).
package aiusage

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

const defaultProviderUnknown = "unknown"

// Entry is one analytics.ai_usage_log row.
type Entry struct {
	UserID           *uuid.UUID
	CourseID         *uuid.UUID
	Feature          string
	Model            string
	ModelAlias       string
	Provider         string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
	CostEstimated    bool
	Succeeded        bool
}

// Insert appends a usage row (best-effort; callers may ignore errors).
// When Provider is empty, stores "unknown" — never silently defaults to openrouter (AP.6 FR-1/FR-7).
func Insert(ctx context.Context, pool *pgxpool.Pool, e Entry) error {
	if pool == nil {
		return nil
	}
	feature := strings.TrimSpace(e.Feature)
	if feature == "" {
		feature = "unknown"
	}
	model := strings.TrimSpace(e.Model)
	if model == "" {
		model = "unknown"
	}
	provider := strings.TrimSpace(e.Provider)
	if provider == "" {
		provider = defaultProviderUnknown
	}
	total := e.TotalTokens
	if total == 0 {
		total = e.PromptTokens + e.CompletionTokens
	}
	var modelAlias any
	if s := strings.TrimSpace(e.ModelAlias); s != "" {
		modelAlias = s
	}
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.ai_usage_log
  (user_id, course_id, feature, model, model_alias, provider, prompt_tokens, completion_tokens, total_tokens, cost_usd, cost_estimated, succeeded)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`, e.UserID, e.CourseID, feature, model, modelAlias, provider, e.PromptTokens, e.CompletionTokens, total, e.CostUSD, e.CostEstimated, e.Succeeded)
	return err
}

// EntryFromUsage builds a log entry from an OpenRouter result (legacy helper).
func EntryFromUsage(userID, courseID *uuid.UUID, feature, model string, usage openrouter.UsageInfo, succeeded bool) Entry {
	return EntryFromProviderUsage(userID, courseID, feature, model, "openrouter", usage, succeeded)
}

// EntryFromProviderUsage builds a log entry with an explicit provider label.
func EntryFromProviderUsage(userID, courseID *uuid.UUID, feature, model, provider string, usage openrouter.UsageInfo, succeeded bool) Entry {
	return Entry{
		UserID:           userID,
		CourseID:         courseID,
		Feature:          feature,
		Model:            model,
		Provider:         provider,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CostUSD:          usage.CostUSD,
		Succeeded:        succeeded,
	}
}

// EntryFromCallMeta builds a log entry from aiprovider CallMeta, applying cost estimation when needed (AP.6).
func EntryFromCallMeta(userID, courseID *uuid.UUID, feature string, meta aiprovider.CallMeta, usage aiprovider.UsageInfo, succeeded bool) Entry {
	u := usage
	estimated := u.CostEstimated
	if !estimated && aiprovider.ApplyCostEstimate(meta.Provider, meta.ModelID, &u) {
		estimated = true
	}
	provider := strings.TrimSpace(string(meta.Provider))
	model := strings.TrimSpace(meta.ModelID)
	if model == "" {
		model = strings.TrimSpace(meta.ModelAlias)
	}
	return Entry{
		UserID:           userID,
		CourseID:         courseID,
		Feature:          feature,
		Model:            model,
		ModelAlias:       strings.TrimSpace(meta.ModelAlias),
		Provider:         provider,
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		TotalTokens:      u.TotalTokens,
		CostUSD:          u.CostUSD,
		CostEstimated:    estimated,
		Succeeded:        succeeded,
	}
}
