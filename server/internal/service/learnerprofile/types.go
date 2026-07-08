package learnerprofile

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FacetResult is the output of one FacetDeriver run.
type FacetResult struct {
	State           string
	Summary         json.RawMessage
	Confidence      float64
	ComputedVersion int
	Insights        []InsightResult
}

// InsightResult is one derived insight with provenance evidence.
type InsightResult struct {
	InsightKey   string
	LabelI18nKey string
	Value        json.RawMessage
	Confidence   float64
	Salience     int
	Evidence     []EvidenceResult
}

// EvidenceResult is aggregated provenance for one insight.
type EvidenceResult struct {
	SourceKind       string
	SourceTable      string
	CourseID         *uuid.UUID
	ObservationCount int
	WindowStart      *time.Time
	WindowEnd        *time.Time
	Contribution     *float64
	SampleRefs       json.RawMessage
}

// ProfileView is the read model returned by Get.
type ProfileView struct {
	Status         string
	LastComputedAt *time.Time
	Facets         []FacetSummary
}

// FacetSummary is a facet without drill-down insights.
type FacetSummary struct {
	FacetKey        string
	State           string
	Summary         json.RawMessage
	Confidence      float64
	ComputedVersion int
	UpdatedAt       time.Time
}

// FacetDetail includes insights for one facet.
type FacetDetail struct {
	Facet    FacetSummary
	Insights []InsightView
}

// InsightView is an insight with resolved label and evidence.
type InsightView struct {
	InsightKey string
	Label      string
	Value      json.RawMessage
	Confidence float64
	Salience   int
	Evidence   []EvidenceView
}

// EvidenceView is the API shape for provenance rows.
type EvidenceView struct {
	SourceKind       string
	SourceTable      string
	CourseID         *string
	ObservationCount int
	WindowStart      *string
	WindowEnd        *string
	Contribution     *float64
}