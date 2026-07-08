package learnerprofile

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Profile row in learner.profiles.
type Profile struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Status         string
	LastComputedAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Facet row in learner.profile_facets.
type Facet struct {
	ID              uuid.UUID
	ProfileID       uuid.UUID
	FacetKey        string
	State           string
	Summary         json.RawMessage
	Confidence      float64
	ComputedVersion int
	UpdatedAt       time.Time
}

// Insight row in learner.profile_insights.
type Insight struct {
	ID           uuid.UUID
	FacetID      uuid.UUID
	InsightKey   string
	LabelI18nKey string
	Value        json.RawMessage
	Confidence   float64
	Salience     int
	CreatedAt    time.Time
}

// Evidence row in learner.profile_evidence.
type Evidence struct {
	ID               uuid.UUID
	InsightID        uuid.UUID
	SourceKind       string
	SourceTable      string
	CourseID         *uuid.UUID
	ObservationCount int
	WindowStart      *time.Time
	WindowEnd        *time.Time
	Contribution     *float64
	SampleRefs       json.RawMessage
	CreatedAt        time.Time
}

// InsightWrite is the write model for one insight and its evidence.
type InsightWrite struct {
	InsightKey   string
	LabelI18nKey string
	Value        json.RawMessage
	Confidence   float64
	Salience     int
	Evidence     []EvidenceWrite
}

// EvidenceWrite is the write model for aggregated evidence.
type EvidenceWrite struct {
	SourceKind       string
	SourceTable      string
	CourseID         *uuid.UUID
	ObservationCount int
	WindowStart      *time.Time
	WindowEnd        *time.Time
	Contribution     *float64
	SampleRefs       json.RawMessage
}

// FacetWrite is the atomic write payload for one facet derivation.
type FacetWrite struct {
	State           string
	Summary         json.RawMessage
	Confidence      float64
	ComputedVersion int
	Insights        []InsightWrite
}