package context

import (
	stdctx "context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

// FeaturePrefix is the analytics.ai_usage_log feature prefix for content tools.
const FeaturePrefix = "content_tool"

// CheckBudgets enforces per-request, per-user daily, and per-course monthly caps (FR-14).
func CheckBudgets(
	ctx stdctx.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	userID uuid.UUID,
	settings *ctrepo.SettingsRow,
	requestTokens int,
) error {
	if requestTokens <= 0 {
		requestTokens = DefaultRequestContextTokens
	}
	if requestTokens > DefaultRequestContextTokens+DefaultRequestCompletionToks {
		observeBudgetDenial("request")
		return &BudgetError{
			Level:   "request",
			Message: fmt.Sprintf("per-request token budget exceeded (cap %d)", DefaultRequestContextTokens+DefaultRequestCompletionToks),
		}
	}
	daily := DefaultDailyCallsPerUser
	monthly := int64(0)
	if settings != nil {
		if settings.DailyAICallsPerUser > 0 {
			daily = settings.DailyAICallsPerUser
		}
		monthly = settings.MonthlyAITokenBudget
	}
	if daily > 0 {
		n, err := ctrepo.CountUserAICallsToday(ctx, pool, userID, FeaturePrefix)
		if err != nil {
			return err
		}
		if n >= daily {
			observeBudgetDenial("daily_user")
			return &BudgetError{
				Level:   "daily_user",
				Message: fmt.Sprintf("per-user daily call budget exceeded (cap %d)", daily),
			}
		}
	}
	if monthly > 0 {
		used, err := ctrepo.SumCourseAITokensMonth(ctx, pool, courseID, FeaturePrefix)
		if err != nil {
			return err
		}
		if used >= monthly {
			observeBudgetDenial("monthly_course")
			return &BudgetError{
				Level:   "monthly_course",
				Message: fmt.Sprintf("per-course monthly token budget exceeded (cap %d)", monthly),
			}
		}
	}
	return nil
}
