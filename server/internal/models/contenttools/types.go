// Package contenttools holds JSON shapes for Content Tools HTTP APIs (plan CT.1+).
package contenttools

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Settings is the course-scoped Content Tools configuration.
type Settings struct {
	AllowedToolIDs       []string  `json:"allowedToolIds"`
	StudentResetAllowed  bool      `json:"studentResetAllowed"`
	MaxInstancesPerItem  int16     `json:"maxInstancesPerItem"`
	MonthlyAITokenBudget int64     `json:"monthlyAiTokenBudget"`
	DailyAICallsPerUser  int       `json:"dailyAiCallsPerUser"`
	LinkIngestionMode    string    `json:"linkIngestionMode"`
	LinkHostAllowlist    []string  `json:"linkHostAllowlist"`
	GradeLinksAllowed    bool      `json:"gradeLinksAllowed"`
	UpdatedAt            time.Time `json:"updatedAt,omitempty"`
}

// ContextSegment is one grounded pack segment (CT.6).
type ContextSegment struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url,omitempty"`
	Lang   string `json:"lang,omitempty"`
	Text   string `json:"text"`
	Tokens int    `json:"tokens"`
}

// ContextPendingSource is a discovered source not yet ready.
type ContextPendingSource struct {
	URL    string `json:"url"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ContextPack is the instructor preview / dry-run pack (CT.6).
type ContextPack struct {
	InstanceID     uuid.UUID              `json:"instanceId"`
	Segments       []ContextSegment       `json:"segments"`
	PendingSources []ContextPendingSource `json:"pendingSources"`
	TotalTokens    int                    `json:"totalTokens"`
	VariantID      *uuid.UUID             `json:"variantId,omitempty"`
	// Optional dry-run AI fields when preview includes a query (CT.6 rails).
	DryRunAnswer    string          `json:"dryRunAnswer,omitempty"`
	DryRunCitations []CitationPublic `json:"dryRunCitations,omitempty"`
	RedactedQuery   string          `json:"redactedQuery,omitempty"`
}

// CitationPublic is a footnote handle returned to clients.
type CitationPublic struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
	Loc   string `json:"loc,omitempty"`
}

// ContextSource is one instructor-visible corpus row (CT.6).
type ContextSource struct {
	ID                uuid.UUID  `json:"id"`
	SourceID          *uuid.UUID `json:"sourceId,omitempty"`
	URL               string     `json:"url"`
	Title             string     `json:"title,omitempty"`
	Host              string     `json:"host,omitempty"`
	Origin            string     `json:"origin"`
	Status            string     `json:"status"`
	Error             string     `json:"error,omitempty"`
	FetchedAt         *time.Time `json:"fetchedAt,omitempty"`
	ByteSize          *int       `json:"byteSize,omitempty"`
	Excluded          bool       `json:"excluded"`
	ExtractedText     string     `json:"extractedText,omitempty"`
	ExtractionQuality string     `json:"extractionQuality,omitempty"`
}

// ContextSourcesResponse lists activity sources.
type ContextSourcesResponse struct {
	Items       []ContextSource `json:"items"`
	TotalTokens int             `json:"totalTokens"`
}

// ContextPreviewRequest is POST .../context/preview.
type ContextPreviewRequest struct {
	InstanceID uuid.UUID `json:"instanceId"`
	Query      string    `json:"query,omitempty"`
}

// ContextSourcePatch is PATCH .../context/sources/{id}.
type ContextSourcePatch struct {
	Excluded *bool `json:"excluded"`
}

// CatalogTool is one tool available in a course after flag/allowlist/role filtering.
type CatalogTool struct {
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Capabilities  []string `json:"capabilities"`
	I18nNamespace string   `json:"i18nNamespace"`
	UI            ToolUI   `json:"ui"`
}

// ToolUI is the manifest ui block exposed to clients.
type ToolUI struct {
	Renderer     string `json:"renderer"`
	Icon         string `json:"icon"`
	Group        string `json:"group"`
	CustomEditor string `json:"customEditor,omitempty"`
}

// ToolManifestPublic is a manifest returned by GET .../manifests/{tool_id}
// (sensitive schema annotations stripped).
type ToolManifestPublic struct {
	ID              string          `json:"id"`
	Version         string          `json:"version"`
	Name            string          `json:"name"`
	Category        string          `json:"category"`
	Capabilities    []string        `json:"capabilities"`
	ConfigSchema    json.RawMessage `json:"configSchema"`
	StateSchema     json.RawMessage `json:"stateSchema"`
	Scoring         Scoring         `json:"scoring"`
	AI              *AIBlock        `json:"ai,omitempty"`
	Network         *NetworkBlock   `json:"network,omitempty"`
	Storage         StorageBlock    `json:"storage"`
	Roles           RolesBlock      `json:"roles"`
	A11y            A11yBlock       `json:"a11y"`
	I18nNamespace   string          `json:"i18nNamespace"`
	UI              ToolUI          `json:"ui"`
	Actions         []ActionPublic  `json:"actions,omitempty"`
	ConflictPolicy  string          `json:"conflictPolicy,omitempty"`
	AutosaveMs      int             `json:"autosaveMs,omitempty"`
	RespectsDueDate bool            `json:"respectsDueDate,omitempty"`
	AllowsSelfReset bool            `json:"allowsSelfReset,omitempty"`
	// CT.5
	Sandbox             string `json:"sandbox,omitempty"`
	Deprecated          bool   `json:"deprecated,omitempty"`
	SunsetAt            string `json:"sunsetAt,omitempty"`
	Contract            int    `json:"contract,omitempty"`
	StateSchemaVersion  int    `json:"stateSchemaVersion,omitempty"`
	ConfigSchemaVersion int    `json:"configSchemaVersion,omitempty"`
}

// RosterStateRow is one learner on GET .../instances/{id}/states (CT.4).
type RosterStateRow struct {
	EnrollmentID     uuid.UUID  `json:"enrollmentId"`
	DisplayName      string     `json:"displayName"`
	Status           string     `json:"status"`
	Score            *ToolScore `json:"score"`
	InteractionCount int        `json:"interactionCount"`
	LastInteractedAt *time.Time `json:"lastInteractedAt"`
	ResetCount       int        `json:"resetCount"`
}

// RosterStatesResponse is the paginated roster.
type RosterStatesResponse struct {
	Items      []RosterStateRow `json:"items"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalCount int              `json:"totalCount"`
}

// StateDetailResponse is GET .../states/{enrollment_id}.
type StateDetailResponse struct {
	EnrollmentID uuid.UUID         `json:"enrollmentId"`
	DisplayName  string            `json:"displayName"`
	Summary      string            `json:"summary"`
	State        ToolStateEnvelope `json:"state"`
}

// ResetRequest is POST .../state-resets.
type ResetRequest struct {
	Scope          string      `json:"scope"`
	InstanceID     *uuid.UUID  `json:"instanceId"`
	ItemID         *uuid.UUID  `json:"itemId"`
	EnrollmentID   *uuid.UUID  `json:"enrollmentId"`
	SectionIDs     []uuid.UUID `json:"sectionIds"`
	Reason         *string     `json:"reason"`
	Notify         *bool       `json:"notify"`
	DryRun         bool        `json:"dryRun"`
	IdempotencyKey *string     `json:"idempotencyKey"`
}

// ResetSampleLearner is one sample row in a reset response.
type ResetSampleLearner struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	DisplayName  string    `json:"displayName"`
	Status       string    `json:"status"`
	Score        *float64  `json:"score"`
}

// GradeEffect describes gradebook side-effects for a reset.
type GradeEffect struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	Action       string    `json:"action"`
	Reason       *string   `json:"reason,omitempty"`
}

// ResetResponse is the dry-run or sync reset result (202 adds jobId).
type ResetResponse struct {
	DryRun          bool                 `json:"dryRun"`
	AffectedCount   int                  `json:"affectedCount"`
	Sample          []ResetSampleLearner `json:"sample"`
	BatchID         *uuid.UUID           `json:"batchId,omitempty"`
	JobID           *uuid.UUID           `json:"jobId,omitempty"`
	GradeEffects    []GradeEffect        `json:"gradeEffects"`
	ScopeNarrowed   bool                 `json:"scopeNarrowed,omitempty"`
	AppliedSections []uuid.UUID          `json:"appliedSections,omitempty"`
}

// StateResetSnapshot is one restorable snapshot.
type StateResetSnapshot struct {
	ID            uuid.UUID  `json:"id"`
	InstanceID    uuid.UUID  `json:"instanceId"`
	EnrollmentID  uuid.UUID  `json:"enrollmentId"`
	ToolID        string     `json:"toolId"`
	Scope         string     `json:"scope"`
	Reason        *string    `json:"reason"`
	BatchID       *uuid.UUID `json:"batchId"`
	ResetBy       *uuid.UUID `json:"resetBy"`
	ResetAt       time.Time  `json:"resetAt"`
	RestoredAt    *time.Time `json:"restoredAt"`
	PurgeAfter    time.Time  `json:"purgeAfter"`
	PriorStatus   string     `json:"priorStatus"`
	PriorRevision int64      `json:"priorRevision"`
}

// ResetJobStatus is GET .../reset-jobs/{job_id}.
type ResetJobStatus struct {
	ID            uuid.UUID       `json:"id"`
	Status        string          `json:"status"`
	Scope         string          `json:"scope"`
	TotalRows     int             `json:"totalRows"`
	ProcessedRows int             `json:"processedRows"`
	BatchID       *uuid.UUID      `json:"batchId,omitempty"`
	Error         *string         `json:"error,omitempty"`
	Result        json.RawMessage `json:"result,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	FinishedAt    *time.Time      `json:"finishedAt,omitempty"`
}

// ActionPublic is one server action declared on a manifest (CT.3).
type ActionPublic struct {
	Name            string `json:"name"`
	RateLimitPerMin int    `json:"rateLimitPerMin,omitempty"`
	RequiresAI      bool   `json:"requiresAi,omitempty"`
	Description     string `json:"description,omitempty"`
}

// Scoring is the manifest scoring block.
type Scoring struct {
	Mode     string   `json:"mode"`
	MaxScore *float64 `json:"maxScore,omitempty"`
}

// AIBlock declares AI usage for a tool.
type AIBlock struct {
	FeatureID string `json:"featureId"`
	Required  bool   `json:"required"`
}

// NetworkBlock declares allowed network hosts.
type NetworkBlock struct {
	AllowedHosts []string `json:"allowedHosts"`
}

// StorageBlock declares state size budget.
type StorageBlock struct {
	MaxStateBytes int `json:"maxStateBytes"`
}

// RolesBlock declares who may interact.
type RolesBlock struct {
	Interact []string `json:"interact"`
}

// A11yBlock is mandatory accessibility metadata.
type A11yBlock struct {
	KeyboardOperable bool   `json:"keyboardOperable"`
	SRPattern        string `json:"srPattern"`
	WCAGNotes        string `json:"wcagNotes,omitempty"`
}

// ToolInstance is a placed tool returned by instance CRUD.
type ToolInstance struct {
	ID              uuid.UUID          `json:"id"`
	ToolID          string             `json:"toolId"`
	ToolVersion     string             `json:"toolVersion"`
	HostKind        string             `json:"hostKind"`
	StructureItemID *uuid.UUID         `json:"structureItemId"`
	SectionKey      *string            `json:"sectionKey"`
	Title           *string            `json:"title"`
	Config          json.RawMessage    `json:"config"`
	Status          string             `json:"status"`
	UpdatedAt       time.Time          `json:"updatedAt"`
	State           *ToolStateEnvelope `json:"state,omitempty"`
	// CT.5 mount metadata (derived from registry + platform flag).
	SandboxMode string `json:"sandboxMode,omitempty"`
	Contract    int    `json:"contract,omitempty"`
	BreakerOpen bool   `json:"breakerOpen,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
	SunsetAt    string `json:"sunsetAt,omitempty"`
	// Capabilities copied from the registered manifest for host gating (e.g. AI disclosure).
	Capabilities []string `json:"capabilities,omitempty"`
	// CT.9: true when the org installation was revoked (read-only tombstone).
	Tombstone bool `json:"tombstone,omitempty"`
}

// ToolStateEnvelope is the CT.3 learner state contract.
type ToolStateEnvelope struct {
	InstanceID  uuid.UUID       `json:"instanceId"`
	Revision    int64           `json:"revision"`
	Status      string          `json:"status"`
	State       json.RawMessage `json:"state"`
	Score       *ToolScore      `json:"score"`
	UpdatedAt   *time.Time      `json:"updatedAt"`
	ResetCount  int             `json:"resetCount"`
	LastResetAt *time.Time      `json:"lastResetAt"`
	// Backward-compatible aliases for CT.1/CT.2 clients.
	StateJSON json.RawMessage `json:"stateJson,omitempty"`
	Scope     string          `json:"scope,omitempty"`
	// CT.5
	StateSchemaVersion int  `json:"stateSchemaVersion,omitempty"`
	Quarantined        bool `json:"quarantined,omitempty"`
}

// ToolScore is the optional scored outcome on a state row.
type ToolScore struct {
	Raw float64 `json:"raw"`
	Max float64 `json:"max"`
}

// SaveStateRequest is PUT .../state (CT.3).
type SaveStateRequest struct {
	Revision  int64           `json:"revision"`
	State     json.RawMessage `json:"state"`
	StateJSON json.RawMessage `json:"stateJson"` // CT.1/CT.2 alias
	Status    string          `json:"status,omitempty"`
}

// RunActionRequest is POST .../actions/{action}.
type RunActionRequest struct {
	Input          json.RawMessage `json:"input"`
	IdempotencyKey string          `json:"idempotencyKey,omitempty"`
}

// RunActionResponse is the action dispatch result.
type RunActionResponse struct {
	Result map[string]any    `json:"result"`
	State  ToolStateEnvelope `json:"state"`
}

// RevisionConflictBody is HTTP 409 for stale revision saves.
type RevisionConflictBody struct {
	Error   string            `json:"error"`
	Current ToolStateEnvelope `json:"current"`
}

// StateTooLargeBody is HTTP 413 for oversized state.
type StateTooLargeBody struct {
	Error    string `json:"error"`
	MaxBytes int    `json:"maxBytes"`
}

// SchemaInvalidBody is HTTP 422 for state schema failures (CT.3 shape).
type SchemaInvalidBody struct {
	Error  string       `json:"error"`
	Errors []FieldError `json:"errors"`
}

// CreateInstanceRequest is POST .../instances.
type CreateInstanceRequest struct {
	ToolID          string          `json:"toolId"`
	HostKind        string          `json:"hostKind"`
	StructureItemID *uuid.UUID      `json:"structureItemId"`
	SectionKey      *string         `json:"sectionKey"`
	Title           *string         `json:"title"`
	Config          json.RawMessage `json:"config"`
}

// PatchInstanceRequest is PATCH .../instances/{id}.
type PatchInstanceRequest struct {
	Title      *string          `json:"title"`
	SectionKey *string          `json:"sectionKey"`
	Config     *json.RawMessage `json:"config"`
	Status     *string          `json:"status"`
}

// ToolInstanceUsage is GET .../instances/{id}/usage (plan CT.2 delete confirmation).
type ToolInstanceUsage struct {
	InstanceID        uuid.UUID `json:"instanceId"`
	LearnersWithState int       `json:"learnersWithState"`
	LearnersCompleted int       `json:"learnersCompleted"`
	LastInteractionAt *string   `json:"lastInteractionAt"`
	ReferencedInBody  bool      `json:"referencedInBody"`
}

// FieldError is one JSON Schema validation failure (HTTP 422).
type FieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ValidationErrorBody is the 422 error payload for config validation.
type ValidationErrorBody struct {
	Error struct {
		Code    string       `json:"code"`
		Message string       `json:"message"`
		Errors  []FieldError `json:"errors"`
	} `json:"error"`
}

// --- CT.7 analytics / gradebook ---

// ScoreDistributionBucket is one histogram bin for scores.
type ScoreDistributionBucket struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// FacetValue is one facet value with its count.
type FacetValue struct {
	Value   string `json:"value"`
	Count   int    `json:"count"`
	Correct *bool  `json:"correct,omitempty"`
}

// FacetAggregate is aggregated facet counts for one key.
type FacetAggregate struct {
	Key    string       `json:"key"`
	Label  string       `json:"label"`
	Values []FacetValue `json:"values"`
}

// NeedsAttention is one learner flagged for instructor follow-up.
type NeedsAttention struct {
	EnrollmentID string `json:"enrollmentId"`
	DisplayName  string `json:"displayName"`
	Reason       string `json:"reason"`
}

// InstanceScoreStats is optional score aggregate block.
type InstanceScoreStats struct {
	Mean         float64                   `json:"mean"`
	Median       float64                   `json:"median"`
	Distribution []ScoreDistributionBucket `json:"distribution"`
}

// InstanceAnalytics is GET .../instances/{id}/analytics (CT.7).
type InstanceAnalytics struct {
	InstanceID       uuid.UUID           `json:"instanceId"`
	ToolID           string              `json:"toolId"`
	Title            *string             `json:"title"`
	Learners         int                 `json:"learners"`
	Engaged          int                 `json:"engaged"`
	Completed        int                 `json:"completed"`
	Suppressed       bool                `json:"suppressed"`
	Score            *InstanceScoreStats `json:"score"`
	MedianDurationMs *int                `json:"medianDurationMs"`
	Facets           []FacetAggregate    `json:"facets"`
	NeedsAttention   []NeedsAttention    `json:"needsAttention"`
	CountsForGrade   bool                `json:"countsForGrade"`
	// AskQuestionsThemes is the CT.10 clustered question themes (anonymized).
	AskQuestionsThemes []AskQuestionsTheme `json:"askQuestionsThemes,omitempty"`
	TotalQuestions     *int                `json:"totalQuestions,omitempty"`
	// CalibrationMatrix is the CT.12 confidence × correctness matrix.
	CalibrationMatrix []CalibrationCell `json:"calibrationMatrix,omitempty"`
}

// AskQuestionsTheme is one anonymized question cluster for instructor insights (CT.10 FR-11).
type AskQuestionsTheme struct {
	Theme                  string   `json:"theme"`
	Count                  int      `json:"count"`
	RepresentativeExamples []string `json:"representativeExamples"`
}

// CalibrationCell is one confidence × correctness cell (CT.12 FR-8).
type CalibrationCell struct {
	ConfidenceBucket string `json:"confidenceBucket"`
	Correct          bool   `json:"correct"`
	Count            int    `json:"count"`
	Highlight        bool   `json:"highlight"`
}

// PageToolsOverview is GET .../content-tools/analytics?itemId=.
type PageToolsOverview struct {
	ItemID    uuid.UUID               `json:"itemId"`
	Instances []InstanceAnalytics     `json:"instances"`
	Totals    PageToolsOverviewTotals `json:"totals"`
}

// PageToolsOverviewTotals rolls up instance counts on a page.
type PageToolsOverviewTotals struct {
	Instances int `json:"instances"`
	Learners  int `json:"learners"`
	Engaged   int `json:"engaged"`
	Completed int `json:"completed"`
}

// CourseToolsAnalytics is GET .../content-tools/analytics/course.
type CourseToolsAnalytics struct {
	CourseID  string                       `json:"courseId"`
	ByTool    []CourseToolRollup           `json:"byTool"`
	Instances []InstanceAnalytics          `json:"instances"`
}

// CourseToolRollup is per-tool_id rollup within a course.
type CourseToolRollup struct {
	ToolID    string `json:"toolId"`
	Instances int    `json:"instances"`
	Learners  int    `json:"learners"`
	Engaged   int    `json:"engaged"`
	Completed int    `json:"completed"`
}

// StudentToolProgress is GET .../content-tools/my-progress?itemId=.
type StudentToolProgress struct {
	ItemID    uuid.UUID                `json:"itemId"`
	Completed int                      `json:"completed"`
	Total     int                      `json:"total"`
	Tools     []StudentToolProgressRow `json:"tools"`
}

// StudentToolProgressRow is one tool on the student's progress strip.
type StudentToolProgressRow struct {
	InstanceID     uuid.UUID `json:"instanceId"`
	ToolID         string    `json:"toolId"`
	Title          *string   `json:"title"`
	Engaged        bool      `json:"engaged"`
	Completed      bool      `json:"completed"`
	ScorePct       *float64  `json:"scorePct,omitempty"`
	CountsForGrade bool      `json:"countsForGrade"`
}

// GradeLink is PUT/GET .../instances/{id}/grade-link.
type GradeLink struct {
	InstanceID       uuid.UUID  `json:"instanceId"`
	AssignmentItemID *uuid.UUID `json:"assignmentItemId,omitempty"`
	OutcomeID        *uuid.UUID `json:"outcomeId,omitempty"`
	PointsPossible   *float64   `json:"pointsPossible,omitempty"`
	CountsForGrade   bool       `json:"countsForGrade"`
	LatePolicy       string     `json:"latePolicy"`
	EnabledAt        *time.Time `json:"enabledAt,omitempty"`
}

// GradeLinkPutRequest is PUT .../grade-link body.
type GradeLinkPutRequest struct {
	AssignmentItemID *uuid.UUID `json:"assignmentItemId"`
	OutcomeID        *uuid.UUID `json:"outcomeId"`
	PointsPossible   *float64   `json:"pointsPossible"`
	CountsForGrade   bool       `json:"countsForGrade"`
	LatePolicy       string     `json:"latePolicy"`
}

// AdminToolTelemetry is GET /api/v1/admin/content-tools/telemetry.
type AdminToolTelemetry struct {
	From  string                    `json:"from"`
	To    string                    `json:"to"`
	Tools []AdminToolTelemetryRow   `json:"tools"`
}

// AdminToolTelemetryRow is one tool's cross-course telemetry (no free-text).
type AdminToolTelemetryRow struct {
	ToolID       string   `json:"toolId"`
	Instances    int      `json:"instances"`
	Learners     int      `json:"learners"`
	Engagements  int      `json:"engagements"`
	Completions  int      `json:"completions"`
	MeanScorePct *float64 `json:"meanScorePct,omitempty"`
	AITokens     int64    `json:"aiTokens"`
	AICostUSD    float64  `json:"aiCostUsd"`
	RenderErrors int      `json:"renderErrors"`
}
