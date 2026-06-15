// Package advising provides degree-audit adapter integration (plan 14.14).
package advising

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const CacheTTL = 4 * time.Hour

const (
	ProviderNone        = "none"
	ProviderDegreeWorks = "degreeworks"
	ProviderStellic     = "stellic"
)

// RequirementGroup is one remaining degree requirement.
type RequirementGroup struct {
	Group            string `json:"group"`
	CoursesRemaining int    `json:"coursesRemaining"`
}

// DegreeProgressSummary is the cached degree audit payload.
type DegreeProgressSummary struct {
	CompletionPercent        int                `json:"completionPercent"`
	RemainingRequiredCount   int                `json:"remainingRequiredCount"`
	RemainingRequirements    []RequirementGroup `json:"remainingRequirements"`
	CourseRequirements       map[string][]string `json:"courseRequirements"`
	AtRisk                   bool               `json:"atRisk,omitempty"`
}

// AdapterConfig holds connection settings for a degree audit provider.
type AdapterConfig struct {
	Provider        string
	BaseURL         string
	CredentialsRef  string
	ExternalSISID   string
}

// DegreeAuditAdapter fetches degree audit summaries from external systems.
type DegreeAuditAdapter interface {
	Provider() string
	FetchSummary(ctx context.Context, cfg AdapterConfig, userID uuid.UUID) (DegreeProgressSummary, error)
}

// AdapterFor returns the adapter for a provider constant.
func AdapterFor(provider string) DegreeAuditAdapter {
	switch provider {
	case ProviderDegreeWorks:
		return degreeWorksAdapter{}
	case ProviderStellic:
		return stellicAdapter{}
	default:
		return nil
	}
}

type degreeWorksAdapter struct{}

func (degreeWorksAdapter) Provider() string { return ProviderDegreeWorks }

func (degreeWorksAdapter) FetchSummary(_ context.Context, cfg AdapterConfig, userID uuid.UUID) (DegreeProgressSummary, error) {
	if cfg.BaseURL == "" || cfg.CredentialsRef == "" {
		return stubSummary(userID, ProviderDegreeWorks), nil
	}
	// Live DegreeWorks REST integration requires institution credentials; stub until pilot.
	return stubSummary(userID, ProviderDegreeWorks), nil
}

type stellicAdapter struct{}

func (stellicAdapter) Provider() string { return ProviderStellic }

func (stellicAdapter) FetchSummary(_ context.Context, cfg AdapterConfig, userID uuid.UUID) (DegreeProgressSummary, error) {
	if cfg.BaseURL == "" {
		return stubSummary(userID, ProviderStellic), nil
	}
	return stubSummary(userID, ProviderStellic), nil
}

func stubSummary(userID uuid.UUID, source string) DegreeProgressSummary {
	// Deterministic demo data from user id for E2E/dev.
	seed := int(userID[0]) + int(userID[1])
	pct := 40 + (seed % 50)
	remaining := 12 - (pct / 10)
	if remaining < 1 {
		remaining = 1
	}
	code := fmt.Sprintf("%s%d", "MATH", 100+(seed%200))
	return DegreeProgressSummary{
		CompletionPercent:      pct,
		RemainingRequiredCount: remaining,
		RemainingRequirements: []RequirementGroup{
			{Group: "Core Mathematics", CoursesRemaining: 2},
			{Group: "General Education — Humanities", CoursesRemaining: 1},
			{Group: "Major Electives", CoursesRemaining: remaining - 3},
		},
		CourseRequirements: map[string][]string{
			code:       {"Core Mathematics"},
			"ENG101":   {"General Education — Humanities"},
			"CS201":    {"Major Electives"},
			"HIST110":  {"General Education — Humanities"},
		},
		AtRisk: seed%7 == 0,
	}
}

// SummaryToJSON marshals a summary for cache storage.
func SummaryToJSON(s DegreeProgressSummary) (json.RawMessage, error) {
	return json.Marshal(s)
}

// ParseSummaryJSON unmarshals cached audit data.
func ParseSummaryJSON(raw json.RawMessage) (DegreeProgressSummary, error) {
	var s DegreeProgressSummary
	if err := json.Unmarshal(raw, &s); err != nil {
		return DegreeProgressSummary{}, err
	}
	return s, nil
}

// CacheExpired reports whether cached data is older than TTL.
func CacheExpired(fetchedAt time.Time, now time.Time) bool {
	return now.Sub(fetchedAt) >= CacheTTL
}

// FulfillsRequirements returns requirement group names a catalog course code satisfies.
func FulfillsRequirements(summary *DegreeProgressSummary, courseCode string) []string {
	if summary == nil || summary.CourseRequirements == nil {
		return nil
	}
	return summary.CourseRequirements[courseCode]
}
