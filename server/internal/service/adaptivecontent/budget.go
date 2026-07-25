package adaptivecontent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
)

// EventBudgetExhausted is written when generation is skipped due to monthly budget.
const EventBudgetExhausted = "budget_exhausted"

// DefaultEstimateTokens is a conservative pre-call token estimate for budget reservation.
const DefaultEstimateTokens int64 = 4000

// BudgetStatus is the instructor-facing budget snapshot (AC.4).
type BudgetStatus struct {
	MonthlyTokenBudget int64
	TokensUsedPeriod   int64
	BudgetRemaining    *int64 // nil when unlimited (budget == 0)
	PeriodStart        time.Time
	GenerationPaused   bool
	Unlimited          bool
}

// BudgetCheck is the result of CheckAndReserve.
type BudgetCheck struct {
	Allowed   bool
	Used      int64
	Budget    int64 // 0 = unlimited
	Remaining int64 // large when unlimited
	Period    time.Time
	Reason    string
}

// LoadBudgetStatus returns the course budget snapshot, reconciling period cache when stale.
func LoadBudgetStatus(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, now time.Time) (BudgetStatus, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	period := acrepo.PeriodStartUTC(now)
	s, err := acrepo.GetSettings(ctx, pool, courseID)
	if err != nil {
		return BudgetStatus{}, err
	}
	if s == nil {
		def := acrepo.DefaultSettings(courseID)
		return BudgetStatus{
			MonthlyTokenBudget: def.MonthlyTokenBudget,
			TokensUsedPeriod:   0,
			BudgetRemaining:    nil,
			PeriodStart:        period,
			GenerationPaused:   false,
			Unlimited:          true,
		}, nil
	}

	used := s.TokensUsedPeriod
	needReconcile := s.BudgetPeriodStart == nil ||
		s.BudgetPeriodStart.UTC().Year() != period.Year() ||
		s.BudgetPeriodStart.UTC().Month() != period.Month()
	if needReconcile {
		if n, rerr := acrepo.ReconcileTokensUsedPeriod(ctx, pool, courseID, now); rerr == nil {
			used = n
		}
	}

	out := BudgetStatus{
		MonthlyTokenBudget: s.MonthlyTokenBudget,
		TokensUsedPeriod:   used,
		PeriodStart:        period,
		GenerationPaused:   s.GenerationPaused,
		Unlimited:          s.MonthlyTokenBudget <= 0,
	}
	if !out.Unlimited {
		rem := s.MonthlyTokenBudget - used
		if rem < 0 {
			rem = 0
		}
		out.BudgetRemaining = &rem
	}
	return out, nil
}

// CheckAndReserve reports whether a projected model call of estTokens is within budget.
// It does not pre-debit; actual spend is applied via RecordTokenUsage after the call.
// monthlyTokenBudget <= 0 means unlimited.
func CheckAndReserve(budget, used, estTokens int64) BudgetCheck {
	if estTokens < 0 {
		estTokens = 0
	}
	if used < 0 {
		used = 0
	}
	// Unlimited
	if budget <= 0 {
		return BudgetCheck{
			Allowed:   true,
			Used:      used,
			Budget:    0,
			Remaining: 1<<62 - 1,
			Reason:    "unlimited",
		}
	}
	if used+estTokens > budget {
		rem := budget - used
		if rem < 0 {
			rem = 0
		}
		return BudgetCheck{
			Allowed:   false,
			Used:      used,
			Budget:    budget,
			Remaining: rem,
			Reason:    "budget_exhausted",
		}
	}
	return BudgetCheck{
		Allowed:   true,
		Used:      used,
		Budget:    budget,
		Remaining: budget - used,
		Reason:    "ok",
	}
}

// CheckCourseBudget loads course settings and checks estTokens against the period budget.
func CheckCourseBudget(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, estTokens int64, now time.Time) (BudgetCheck, error) {
	st, err := LoadBudgetStatus(ctx, pool, courseID, now)
	if err != nil {
		return BudgetCheck{}, err
	}
	c := CheckAndReserve(st.MonthlyTokenBudget, st.TokensUsedPeriod, estTokens)
	c.Period = st.PeriodStart
	return c, nil
}

// RecordTokenUsage updates the period cache after a real model call.
func RecordTokenUsage(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, tokens int64, now time.Time) error {
	if tokens <= 0 {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return acrepo.AddTokensUsedPeriod(ctx, pool, courseID, tokens, now)
}

// RecordBudgetExhaustedEvent writes an audit event for AC.4 budget blocks.
func RecordBudgetExhaustedEvent(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, unitID *uuid.UUID, detail map[string]any) {
	if detail == nil {
		detail = map[string]any{}
	}
	_ = acrepo.InsertEvent(ctx, pool, courseID, unitID, nil, nil, EventBudgetExhausted, detail)
}
