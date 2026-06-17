// Package aiusage holds JSON DTOs for Intelligence AI usage reports.
package aiusage

import "time"

// ReportsPayload is GET /api/v1/settings/ai/reports.
type ReportsPayload struct {
	Range     DateRange         `json:"range"`
	Cost      CostReport        `json:"cost"`
	ByUser    []UserUsageRow    `json:"byUser"`
	ByCourse  []CourseUsageRow  `json:"byCourse"`
}

// DateRange is the resolved query window (RFC 3339, UTC).
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// CostReport aggregates platform AI spend.
type CostReport struct {
	Summary   CostSummary      `json:"summary"`
	ByDay     []DayCostBucket  `json:"byDay"`
	ByFeature []FeatureCostRow `json:"byFeature"`
}

// CostSummary is headline totals.
type CostSummary struct {
	TotalCostUSD float64 `json:"totalCostUsd"`
	TotalCalls   int64   `json:"totalCalls"`
	TotalTokens  int64   `json:"totalTokens"`
}

// DayCostBucket is one UTC calendar day.
type DayCostBucket struct {
	Day     string  `json:"day"`
	CostUSD float64 `json:"costUsd"`
	Calls   int64   `json:"calls"`
	Tokens  int64   `json:"tokens"`
}

// FeatureCostRow is spend for one AI feature key.
type FeatureCostRow struct {
	Feature string  `json:"feature"`
	CostUSD float64 `json:"costUsd"`
	Calls   int64   `json:"calls"`
	Tokens  int64   `json:"tokens"`
}

// UserUsageRow is per-user rollup.
type UserUsageRow struct {
	UserID           string  `json:"userId"`
	Email            string  `json:"email"`
	DisplayName      string  `json:"displayName"`
	Calls            int64   `json:"calls"`
	PromptTokens     int64   `json:"promptTokens"`
	CompletionTokens int64   `json:"completionTokens"`
	TotalTokens      int64   `json:"totalTokens"`
	CostUSD          float64 `json:"costUsd"`
}

// CourseUsageRow is per-course rollup.
type CourseUsageRow struct {
	CourseID   string  `json:"courseId"`
	CourseCode string  `json:"courseCode"`
	Title      string  `json:"title"`
	Calls      int64   `json:"calls"`
	TotalTokens int64  `json:"totalTokens"`
	CostUSD    float64 `json:"costUsd"`
}