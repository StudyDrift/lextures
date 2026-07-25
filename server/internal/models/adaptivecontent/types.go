// Package adaptivecontent holds JSON shapes for Adaptive Content Engine HTTP APIs (plans AC.1–AC.7).
package adaptivecontent

import (
	"time"

	"github.com/google/uuid"
)

// DraftSection is a content-page section shape (mirrors contentpagegeneration.DraftSection).
type DraftSection struct {
	Heading  string `json:"heading"`
	Markdown string `json:"markdown"`
}

// Settings is the course-scoped ACE configuration returned by GET/PUT .../adaptive-content/settings.
type Settings struct {
	AllowedAxes               []string  `json:"allowedAxes"`
	DefaultStrategy           string    `json:"defaultStrategy"`
	HoldoutPercent            int16     `json:"holdoutPercent"`
	MonthlyTokenBudget        int64     `json:"monthlyTokenBudget"`
	RequireInstructorApproval bool      `json:"requireInstructorApproval"`
	StudentOptoutAllowed      bool      `json:"studentOptoutAllowed"`
	UpdatedAt                 time.Time `json:"updatedAt,omitempty"`
	// AC.4
	GenerationPaused   bool  `json:"generationPaused"`
	MaxPrewarmVariants int16 `json:"maxPrewarmVariants"`
}

// PatchSettingsRequest is PATCH .../adaptive-content/settings (AC.4 pipeline controls).
type PatchSettingsRequest struct {
	GenerationPaused   *bool  `json:"generationPaused"`
	MaxPrewarmVariants *int16 `json:"maxPrewarmVariants"`
}

// BudgetResponse is GET .../adaptive-content/budget (AC.4).
type BudgetResponse struct {
	MonthlyTokenBudget int64  `json:"monthlyTokenBudget"`
	TokensUsedPeriod   int64  `json:"tokensUsedPeriod"`
	BudgetRemaining    *int64 `json:"budgetRemaining"` // null when unlimited
	PeriodStart        string `json:"periodStart"`     // YYYY-MM-DD
	GenerationPaused   bool   `json:"generationPaused"`
	Unlimited          bool   `json:"unlimited"`
}

// PrewarmResponse is POST .../units/{id}/prewarm (AC.4).
type PrewarmResponse struct {
	Enqueued   int       `json:"enqueued"`
	QueueDepth int64     `json:"queueDepth"`
	UnitID     uuid.UUID `json:"unitId"`
}

// AdminAdaptiveContentResponse is GET/PATCH /api/v1/admin/adaptive-content (AC.4).
type AdminAdaptiveContentResponse struct {
	GenerationPaused bool  `json:"generationPaused"`
	QueueDepth       int64 `json:"queueDepth"`
	Inflight         int64 `json:"inflight"`
	KillSwitch       bool  `json:"killSwitch"`
}

// AdminAdaptiveContentPatch is PATCH /api/v1/admin/adaptive-content body.
type AdminAdaptiveContentPatch struct {
	GenerationPaused *bool `json:"generationPaused"`
}

// Unit is an authorable adaptive content unit.
type Unit struct {
	ID                   uuid.UUID  `json:"id"`
	CourseID             uuid.UUID  `json:"courseId"`
	TargetKind           string     `json:"targetKind"`
	TargetModuleItemID   *uuid.UUID `json:"targetModuleItemId,omitempty"`
	TargetOutcomeID      *uuid.UUID `json:"targetOutcomeId,omitempty"`
	BaseContentItemID    uuid.UUID  `json:"baseContentItemId"`
	PreAssessmentItemID  *uuid.UUID `json:"preAssessmentItemId,omitempty"`
	PostAssessmentItemID *uuid.UUID `json:"postAssessmentItemId,omitempty"`
	AllowedAxes          []string   `json:"allowedAxes"`
	Status               string     `json:"status"`
	CreatedBy            uuid.UUID  `json:"createdBy"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	// AC.2
	TriggerMode          string      `json:"triggerMode"`
	MasteryFreshnessDays int16       `json:"masteryFreshnessDays"`
	ConceptIDs           []uuid.UUID `json:"conceptIds,omitempty"`
	// AC.3
	ContentVersion int32   `json:"contentVersion"`
	MinFidelity    float64 `json:"minFidelity"`
	// AC.5 — coverage for authoring workspace list (optional; filled on list).
	VariantTotal         *int64 `json:"variantTotal,omitempty"`
	VariantApproved      *int64 `json:"variantApproved,omitempty"`
	VariantPendingReview *int64 `json:"variantPendingReview,omitempty"`
	VariantRejected      *int64 `json:"variantRejected,omitempty"`
	VariantAutoServed    *int64 `json:"variantAutoServed,omitempty"`
}

// CreateUnitRequest is the POST .../units body.
type CreateUnitRequest struct {
	TargetKind           string      `json:"targetKind"`
	TargetModuleItemID   *uuid.UUID  `json:"targetModuleItemId"`
	TargetOutcomeID      *uuid.UUID  `json:"targetOutcomeId"`
	BaseContentItemID    uuid.UUID   `json:"baseContentItemId"`
	PreAssessmentItemID  *uuid.UUID  `json:"preAssessmentItemId"`
	PostAssessmentItemID *uuid.UUID  `json:"postAssessmentItemId"`
	AllowedAxes          []string    `json:"allowedAxes"`
	Status               string      `json:"status"`
	TriggerMode          string      `json:"triggerMode"`
	MasteryFreshnessDays *int16      `json:"masteryFreshnessDays"`
	ConceptIDs           []uuid.UUID `json:"conceptIds"`
}

// PatchUnitRequest is the PATCH .../units/{id} body (all fields optional).
type PatchUnitRequest struct {
	TargetKind           *string     `json:"targetKind"`
	TargetModuleItemID   *uuid.UUID  `json:"targetModuleItemId"`
	TargetOutcomeID      *uuid.UUID  `json:"targetOutcomeId"`
	BaseContentItemID    *uuid.UUID  `json:"baseContentItemId"`
	PreAssessmentItemID  *uuid.UUID  `json:"preAssessmentItemId"`
	PostAssessmentItemID *uuid.UUID  `json:"postAssessmentItemId"`
	AllowedAxes          []string    `json:"allowedAxes"`
	Status               *string     `json:"status"`
	// ClearPreAssessment when true sets pre_assessment_item_id to NULL.
	ClearPreAssessment bool `json:"clearPreAssessment"`
	// ClearPostAssessment when true sets post_assessment_item_id to NULL.
	ClearPostAssessment bool `json:"clearPostAssessment"`
	// AC.2
	TriggerMode          *string     `json:"triggerMode"`
	MasteryFreshnessDays *int16      `json:"masteryFreshnessDays"`
	ConceptIDs           []uuid.UUID `json:"conceptIds"`
	// ClearConceptIDs when true replaces the unit concept set with empty.
	ClearConceptIDs bool `json:"clearConceptIds"`
	// AC.3
	MinFidelity *float64 `json:"minFidelity"`
}

// UnitsListResponse wraps a list of units.
type UnitsListResponse struct {
	Units []Unit `json:"units"`
}

// ConceptGap is one concept gap in an adaptation profile payload.
type ConceptGap struct {
	ConceptID uuid.UUID `json:"conceptId"`
	Gap       float64   `json:"gap"`
}

// AdaptationProfile is the student-facing profile (own only).
type AdaptationProfile struct {
	UnitID           uuid.UUID    `json:"unitId"`
	EmphasisMode     string       `json:"emphasisMode"`
	TargetBloom      *string      `json:"targetBloom,omitempty"`
	ProfileSignature string       `json:"profileSignature"`
	IsNeutral        bool         `json:"isNeutral"`
	ConceptGaps      []ConceptGap `json:"conceptGaps"`
	Misconceptions   []string     `json:"misconceptions"`
	ReadingLevelPref *string      `json:"readingLevelPref,omitempty"`
	ModalityPref     *string      `json:"modalityPref,omitempty"`
	AxisSet          []string     `json:"axisSet,omitempty"`
	SourceAttemptID  *uuid.UUID   `json:"sourceAttemptId,omitempty"`
	CreatedAt        time.Time    `json:"createdAt,omitempty"`
}

// EmphasisBucket is a cohort count for one emphasis mode.
type EmphasisBucket struct {
	EmphasisMode string `json:"emphasisMode"`
	Count        int64  `json:"count"`
}

// SignatureBucket is a cohort count for one profile signature (no PII).
type SignatureBucket struct {
	ProfileSignature string `json:"profileSignature"`
	EmphasisMode     string `json:"emphasisMode"`
	Count            int64  `json:"count"`
}

// CohortProfilesResponse is GET .../units/{id}/profiles (instructor).
type CohortProfilesResponse struct {
	ByEmphasis  []EmphasisBucket  `json:"byEmphasis"`
	BySignature []SignatureBucket `json:"bySignature"`
}

// GeneratePreCheckRequest is optional body for POST .../pre-check/generate.
type GeneratePreCheckRequest struct {
	Title         string `json:"title"`
	QuestionCount int32  `json:"questionCount"`
}

// GeneratePreCheckResponse returns the created quiz structure item id and updated unit.
type GeneratePreCheckResponse struct {
	PreAssessmentItemID uuid.UUID `json:"preAssessmentItemId"`
	Unit                Unit      `json:"unit"`
}

// SyntheticProfile is an instructor-supplied hypothetical learner for preview (AC.3).
type SyntheticProfile struct {
	EmphasisMode     string       `json:"emphasisMode"`
	TargetBloom      string       `json:"targetBloom"`
	ReadingLevelPref string       `json:"readingLevelPref"`
	ModalityPref     string       `json:"modalityPref"`
	ConceptGaps      []ConceptGap `json:"conceptGaps"`
	Misconceptions   []string     `json:"misconceptions"`
	AxisSet          []string     `json:"axisSet"`
}

// PreviewVariantRequest is POST .../units/{id}/variants/preview body.
type PreviewVariantRequest struct {
	// ProfileSignature loads a real cohort signature's axes from an existing profile if present.
	ProfileSignature string `json:"profileSignature"`
	// SyntheticProfile builds a hypothetical learner when ProfileSignature is empty.
	SyntheticProfile *SyntheticProfile `json:"syntheticProfile"`
	// Persist when true upserts the content_variants row (default true for real signatures; false for pure synthetic if omitted).
	Persist *bool `json:"persist"`
}

// ContentVariant is a stored or previewed content variant (AC.3 / AC.5).
type ContentVariant struct {
	ID               uuid.UUID      `json:"id,omitempty"`
	UnitID           uuid.UUID      `json:"unitId,omitempty"`
	ProfileSignature string         `json:"profileSignature"`
	AxesApplied      []string       `json:"axesApplied"`
	Sections         []DraftSection `json:"sections,omitempty"`
	VariantMarkdown  string         `json:"variantMarkdown"`
	Model            string         `json:"model,omitempty"`
	FidelityScore    *float64       `json:"fidelityScore,omitempty"`
	SafetyFlags      []string       `json:"safetyFlags"`
	A11yFlags        []string       `json:"a11yFlags"`
	Status           string         `json:"status"`
	PromptVersion    string         `json:"promptVersion,omitempty"`
	ContentVersion   int32          `json:"contentVersion,omitempty"`
	PromptTokens     int32          `json:"promptTokens,omitempty"`
	CompletionTokens int32          `json:"completionTokens,omitempty"`
	CreatedAt        time.Time      `json:"createdAt,omitempty"`
	Fallback         bool           `json:"fallback,omitempty"`
	FallbackReason   string         `json:"fallbackReason,omitempty"`
	CacheHit         bool           `json:"cacheHit,omitempty"`
	// AC.5 review metadata
	HumanEdited    bool       `json:"humanEdited,omitempty"`
	ReviewedBy     *uuid.UUID `json:"reviewedBy,omitempty"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty"`
	ReviewNote     *string    `json:"reviewNote,omitempty"`
	VariantVersion int32      `json:"variantVersion,omitempty"`
	ApprovedBy     *uuid.UUID `json:"approvedBy,omitempty"`
}

// PreviewVariantResponse is the instructor preview payload (AC.3).
type PreviewVariantResponse struct {
	Variant          ContentVariant `json:"variant"`
	FidelityScore    float64        `json:"fidelityScore"`
	A11yFlags        []string       `json:"a11yFlags"`
	SafetyFlags      []string       `json:"safetyFlags"`
	PromptTokens     int            `json:"promptTokens"`
	CompletionTokens int            `json:"completionTokens"`
	BaseMarkdown     string         `json:"baseMarkdown,omitempty"`
}

// VariantsListResponse wraps variants for GET .../variants.
type VariantsListResponse struct {
	Variants []ContentVariant `json:"variants"`
}

// KeyTerm is an instructor-marked fidelity anchor.
type KeyTerm struct {
	ID         uuid.UUID `json:"id"`
	UnitID     uuid.UUID `json:"unitId"`
	Term       string    `json:"term"`
	MustAppear bool      `json:"mustAppear"`
	CreatedAt  time.Time `json:"createdAt,omitempty"`
}

// KeyTermsListResponse wraps key terms.
type KeyTermsListResponse struct {
	KeyTerms []KeyTerm `json:"keyTerms"`
}

// PutKeyTermsRequest replaces all key terms for a unit.
type PutKeyTermsRequest struct {
	Terms []struct {
		Term       string `json:"term"`
		MustAppear *bool  `json:"mustAppear"`
	} `json:"terms"`
}

// ReviewVariantRequest is POST .../variants/{vid}/approve|reject|revoke body (AC.5).
type ReviewVariantRequest struct {
	// ExpectedVariantVersion for optimistic concurrency (0 = skip check).
	ExpectedVariantVersion int32 `json:"expectedVariantVersion"`
	// Note is an optional reason (reject) or comment (approve/revoke).
	Note string `json:"note"`
	// OverrideGate allows approving soft gate failures (not hard key-term misses).
	OverrideGate bool `json:"overrideGate"`
}

// EditAndApproveVariantRequest is PUT .../variants/{vid} (edit body then approve).
type EditAndApproveVariantRequest struct {
	ExpectedVariantVersion int32  `json:"expectedVariantVersion"`
	VariantMarkdown        string `json:"variantMarkdown"`
	Note                   string `json:"note"`
	// OverrideGate allows approving soft gate failures after re-check.
	OverrideGate bool `json:"overrideGate"`
}

// BulkVariantsRequest is POST .../units/{id}/variants/bulk.
type BulkVariantsRequest struct {
	Action                 string      `json:"action"` // approve | reject
	VariantIDs             []uuid.UUID `json:"variantIds"`
	ExpectedVariantVersion int32       `json:"expectedVariantVersion"` // applied to each when > 0 (usually 0 for bulk)
	Note                   string      `json:"note"`
	OverrideGate           bool        `json:"overrideGate"`
}

// BulkVariantsResponse summarizes bulk approve/reject results.
type BulkVariantsResponse struct {
	Succeeded int              `json:"succeeded"`
	Failed    int              `json:"failed"`
	Results   []BulkVariantRow `json:"results"`
}

// BulkVariantRow is one bulk action outcome.
type BulkVariantRow struct {
	VariantID uuid.UUID `json:"variantId"`
	OK        bool      `json:"ok"`
	Error     string    `json:"error,omitempty"`
	Status    string    `json:"status,omitempty"`
}

// ReviewQueueResponse is GET .../adaptive-content/review-queue (course-wide pending).
type ReviewQueueResponse struct {
	Variants []ContentVariant `json:"variants"`
	Total    int64            `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

// ServingMeta is embedded on content-page GET when the page is an active ACE unit base (AC.6).
type ServingMeta struct {
	UnitID                uuid.UUID  `json:"unitId"`
	IsAdapted             bool       `json:"isAdapted"`
	ServedVariantID       *uuid.UUID `json:"servedVariantId,omitempty"`
	AxesApplied           []string   `json:"axesApplied"`
	CanViewOriginal       bool       `json:"canViewOriginal"`
	OptedOut              bool       `json:"optedOut"`
	IsHoldout             bool       `json:"isHoldout"`
	WasFallback           bool       `json:"wasFallback,omitempty"`
	AdaptationReason      string     `json:"adaptationReason,omitempty"`
	PreAssessmentItemID   *uuid.UUID `json:"preAssessmentItemId,omitempty"`
	RequiresPreAssessment bool       `json:"requiresPreAssessment,omitempty"`
	// OptoutAllowed surfaces whether the course currently allows student opt-out.
	OptoutAllowed bool `json:"optoutAllowed,omitempty"`
}

// OptoutResponse is GET/PUT .../adaptive-content/optout (AC.6).
type OptoutResponse struct {
	OptedOut      bool `json:"optedOut"`
	OptoutAllowed bool `json:"optoutAllowed"`
}

// OptoutPutRequest is PUT .../adaptive-content/optout body.
type OptoutPutRequest struct {
	OptedOut bool `json:"optedOut"`
}

// ViewedOriginalResponse is POST .../units/{id}/viewed-original (AC.6).
type ViewedOriginalResponse struct {
	ViewOriginalClicks int32 `json:"viewOriginalClicks"`
}

// ModeEffectiveness is one emphasis-mode effectiveness bucket (AC.7).
type ModeEffectiveness struct {
	EmphasisMode string   `json:"emphasisMode"`
	N            int      `json:"n"`
	MeanLift     *float32 `json:"meanLift"` // null when suppressed (n < k)
}

// VariantEffectiveness is one variant effectiveness bucket (AC.7).
type VariantEffectiveness struct {
	VariantID *uuid.UUID `json:"variantId"`
	N         int        `json:"n"`
	MeanLift  *float32   `json:"meanLift"` // null when suppressed (n < k)
}

// UnitEffectiveness is GET .../units/{id}/effectiveness (AC.7).
type UnitEffectiveness struct {
	UnitID                    uuid.UUID              `json:"unitId"`
	NTreatment                int                    `json:"nTreatment"`
	NHoldout                  int                    `json:"nHoldout"`
	MeanLiftTreatment         *float32               `json:"meanLiftTreatment"`
	MeanLiftHoldout           *float32               `json:"meanLiftHoldout"`
	TreatmentMinusHoldout     *float32               `json:"treatmentMinusHoldout"`
	DiffStdError              *float32               `json:"diffStdError"`
	MeanMasteryDeltaTreatment *float32               `json:"meanMasteryDeltaTreatment,omitempty"`
	MeanMasteryDeltaHoldout   *float32               `json:"meanMasteryDeltaHoldout,omitempty"`
	Verdict                   string                 `json:"verdict"`
	ByMode                    []ModeEffectiveness    `json:"byMode"`
	ByVariant                 []VariantEffectiveness `json:"byVariant"`
	RefreshedAt               *time.Time             `json:"refreshedAt,omitempty"`
	SmallCellMinN             int                    `json:"smallCellMinN"`
	MinNPerArm                int                    `json:"minNPerArm"`
}

// CourseEffectivenessResponse is GET .../adaptive-content/effectiveness (AC.7).
type CourseEffectivenessResponse struct {
	Units []UnitEffectiveness `json:"units"`
}

// EffectivenessRefreshResponse is POST .../effectiveness/refresh (AC.7).
type EffectivenessRefreshResponse struct {
	RefreshedUnits int `json:"refreshedUnits"`
}
