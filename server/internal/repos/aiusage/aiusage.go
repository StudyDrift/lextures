// Package aiusage persists OpenRouter token/cost rows for Intelligence reports.
package aiusage

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/openrouter"
)

// Entry is one analytics.ai_usage_log row.
type Entry struct {
	UserID           *uuid.UUID
	CourseID         *uuid.UUID
	Feature          string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
	Succeeded        bool
}

// Insert appends a usage row (best-effort; callers may ignore errors).
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
	total := e.TotalTokens
	if total == 0 {
		total = e.PromptTokens + e.CompletionTokens
	}
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.ai_usage_log
  (user_id, course_id, feature, model, prompt_tokens, completion_tokens, total_tokens, cost_usd, succeeded)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`, e.UserID, e.CourseID, feature, model, e.PromptTokens, e.CompletionTokens, total, e.CostUSD, e.Succeeded)
	return err
}

// EntryFromUsage builds a log entry from an OpenRouter result.
func EntryFromUsage(userID, courseID *uuid.UUID, feature, model string, usage openrouter.UsageInfo, succeeded bool) Entry {
	return Entry{
		UserID:           userID,
		CourseID:         courseID,
		Feature:          feature,
		Model:            model,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CostUSD:          usage.CostUSD,
		Succeeded:        succeeded,
	}
}