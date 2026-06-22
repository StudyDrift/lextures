// Package peerreview defines domain types for plan 3.15 peer review & assessment.
package peerreview

import (
	"time"

	"github.com/google/uuid"
)

type AnonymityMode string

const (
	AnonymityDoubleBlind  AnonymityMode = "double_blind"
	AnonymityReviewerAnon AnonymityMode = "reviewer_anon"
	AnonymityNamed        AnonymityMode = "named"
)

type GradeMode string

const (
	GradeModeNone          GradeMode = "none"
	GradeModeScoreOnly     GradeMode = "score_only"
	GradeModeWeightedBlend GradeMode = "weighted_blend"
)

type Aggregation string

const (
	AggregationMean    Aggregation = "mean"
	AggregationMedian  Aggregation = "median"
	AggregationTrimmed Aggregation = "trimmed"
)

type AllocationStatus string

const (
	AllocationAssigned   AllocationStatus = "assigned"
	AllocationInProgress AllocationStatus = "in_progress"
	AllocationSubmitted  AllocationStatus = "submitted"
	AllocationExpired    AllocationStatus = "expired"
)

type Config struct {
	ID                 uuid.UUID
	AssignmentID       uuid.UUID
	ReviewsPerReviewer int
	Anonymity          AnonymityMode
	OpensAt            *time.Time
	ClosesAt           *time.Time
	GradeMode          GradeMode
	BlendWeight        float64
	Aggregation        Aggregation
	ExcludeSameGroup   bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Allocation struct {
	ID                   uuid.UUID
	ConfigID             uuid.UUID
	ReviewerEnrollmentID uuid.UUID
	TargetSubmissionID   uuid.UUID
	Status               AllocationStatus
	AssignedAt           time.Time
}

type Review struct {
	ID               uuid.UUID
	AllocationID     uuid.UUID
	Score            *float64
	RubricScoresJSON []byte
	Comments         *string
	SubmittedAt      time.Time
}

type TeamEvaluation struct {
	ID                  uuid.UUID
	GroupID             uuid.UUID
	RaterEnrollmentID   uuid.UUID
	RateeEnrollmentID   uuid.UUID
	ContributionScore   int
	Comment             *string
	SubmittedAt         time.Time
}
