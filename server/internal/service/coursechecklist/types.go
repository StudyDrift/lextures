package coursechecklist

import "context"

// ItemID is a stable checklist rule identifier (persisted contract).
type ItemID string

// CategoryID groups checklist items for display ordering.
type CategoryID string

// Tier controls badge weighting (essential drives the nav badge in CC.7).
type Tier string

const (
	TierEssential   Tier = "essential"
	TierRecommended Tier = "recommended"
)

// Status is the evaluation outcome for one checklist item.
type Status string

const (
	StatusDone          Status = "done"
	StatusTodo          Status = "todo"
	StatusInProgress    Status = "in_progress"
	StatusNotApplicable Status = "not_applicable"
	StatusUnknown       Status = "unknown"
)

// MaxEvidenceRows is the hard cap on evidence rows returned per finding (FR-5 / AC-4).
const MaxEvidenceRows = 200

// EngineVersionConst is the current evaluator contract version. Bump when Result
// shape or evaluation semantics change in a way that invalidates cached snapshots (CC.2).
const EngineVersionConst = 1

// NavTarget describes where the UI should navigate for an actionable item (CC.8).
type NavTarget struct {
	Surface   string `json:"surface"`             // "web" | "ios" | "android" | "all"
	Route     string `json:"route"`               // e.g. "/courses/{courseCode}/settings/general"
	Anchor    string `json:"anchor,omitempty"`    // focus token, e.g. "course.general.dates"
	EntityKey string `json:"entityKey,omitempty"` // optional; substituted from evidence row
}

// EvidenceShape names columns for the expandable evidence table (CC.7).
type EvidenceShape struct {
	Columns []string `json:"columns"`
}

// ActionKind identifies an optional assisted-fix primary action (CC.10).
// Unknown kinds are ignored by clients (backward compatible).
type ActionKind string

const (
	ActionKindSuggestOutcomeMappings ActionKind = "suggest_outcome_mappings"
	ActionKindBuildRubricAI          ActionKind = "build_rubric_ai"
	ActionKindDraftWelcome           ActionKind = "draft_welcome"
	ActionKindSuggestAltText         ActionKind = "suggest_alt_text"
)

// ItemAction is an optional primary action declared on a registry item (CC.10 FR-5).
// Endpoint is a relative API path template (e.g. "/api/v1/courses/{courseCode}/outcomes/suggest-links").
// Clients that do not recognise Kind render nothing.
type ItemAction struct {
	Kind      ActionKind `json:"kind"`
	LabelKey  string     `json:"labelKey"`
	Label     string     `json:"label"`
	Endpoint  string     `json:"endpoint"`
	RequiresAI bool      `json:"requiresAi"`
}

// EvidenceRow is one offending (or exemplary) entity in a finding.
// Privacy: Label/Sublabel may carry display name + opaque ID only — never email or DOB.
type EvidenceRow struct {
	Label          string     `json:"label"`
	Sublabel       string     `json:"sublabel,omitempty"`
	TargetOverride *NavTarget `json:"targetOverride,omitempty"`
	Status         Status     `json:"status,omitempty"`
}

// Progress is optional done/total counters for in_progress items.
type Progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// Finding is the evaluator output for one item.
type Finding struct {
	Status        Status         `json:"status"`
	DetailKey     string         `json:"detailKey,omitempty"`
	DetailDefault string         `json:"detailDefault,omitempty"`
	DetailFields  map[string]any `json:"detailFields,omitempty"`
	Progress      *Progress      `json:"progress,omitempty"`
	Evidence      []EvidenceRow  `json:"evidence,omitempty"`
	TruncatedAt   *int           `json:"truncatedAt,omitempty"`
}

// ItemDescriptor is one declarative checklist rule.
type ItemDescriptor struct {
	ID            ItemID
	Category      CategoryID
	TitleKey      string
	TitleDefault  string
	WhyKey        string
	WhyDefault    string
	HelpRef       string
	Tier          Tier
	Sources       []string
	DataNeeds     []DataNeed
	LazyNeeds     []LazyLoaderID
	Applies       func(CourseSnapshot) bool
	Evaluate      func(context.Context, CourseSnapshot) (Finding, error)
	Target        NavTarget
	EvidenceShape *EvidenceShape
	// Action is an optional assisted-fix primary action (CC.10). Nil when absent.
	Action *ItemAction
}

// EvaluateOptions controls a single Evaluate call.
type EvaluateOptions struct {
	// Only, when non-empty, restricts evaluation to those item IDs (after ResolveItemID).
	Only []ItemID
	// Registry overrides the process default registry (tests / custom packs).
	Registry *Registry
	// LazyLoaders supplies expensive data loaders keyed by LazyLoaderID.
	LazyLoaders map[LazyLoaderID]LazyLoader
}

// ItemResult is one evaluated checklist item in Result.
type ItemResult struct {
	ID            ItemID         `json:"id"`
	Category      CategoryID     `json:"category"`
	Tier          Tier           `json:"tier"`
	TitleKey      string         `json:"titleKey"`
	TitleDefault  string         `json:"titleDefault"`
	WhyKey        string         `json:"whyKey"`
	WhyDefault    string         `json:"whyDefault"`
	HelpRef       string         `json:"helpRef,omitempty"`
	Sources       []string       `json:"sources"`
	Target        NavTarget      `json:"target"`
	EvidenceShape *EvidenceShape `json:"evidenceShape,omitempty"`
	Action        *ItemAction    `json:"action,omitempty"`
	Finding       Finding        `json:"finding"`
}

// Counts aggregates checklist progress. Total is the progress denominator
// (done + todo + in_progress); not_applicable and unknown are excluded (AC-3, §18 Q3).
type Counts struct {
	Total                int `json:"total"`
	Done                 int `json:"done"`
	Todo                 int `json:"todo"`
	InProgress           int `json:"inProgress"`
	NotApplicable        int `json:"notApplicable"`
	Unknown              int `json:"unknown"`
	OutstandingEssential int `json:"outstandingEssential"`
}

// CategoryCounts is Counts for one category, preserving registry category order.
type CategoryCounts struct {
	Category CategoryID `json:"category"`
	Counts
}

// Result is the full evaluation outcome for a course.
type Result struct {
	Findings       []ItemResult     `json:"findings"`
	Counts         Counts           `json:"counts"`
	ByCategory     []CategoryCounts `json:"byCategory"`
	CatalogVersion string           `json:"catalogVersion"`
	EngineVersion  int              `json:"engineVersion"`
}
