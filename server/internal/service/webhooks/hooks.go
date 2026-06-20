package webhooksvc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	webhooksrepo "github.com/lextures/lextures/server/internal/repos/webhooks"
	"github.com/lextures/lextures/server/internal/webhooks"
)

// GradePostedData is the grade.posted event payload (no PII by default).
type GradePostedData struct {
	CourseID      string  `json:"courseId"`
	CourseCode    string  `json:"courseCode,omitempty"`
	ModuleItemID  string  `json:"moduleItemId"`
	StudentUserID string  `json:"studentUserId"`
	PointsEarned  float64 `json:"pointsEarned"`
}

// EnrollmentCreatedData is the enrollment.created event payload.
type EnrollmentCreatedData struct {
	CourseID     string `json:"courseId"`
	CourseCode   string `json:"courseCode,omitempty"`
	EnrollmentID string `json:"enrollmentId"`
	StudentUserID string `json:"studentUserId"`
	Role         string `json:"role"`
}

// AssignmentSubmittedData is the assignment.submitted event payload.
type AssignmentSubmittedData struct {
	CourseID     string `json:"courseId"`
	CourseCode   string `json:"courseCode,omitempty"`
	ModuleItemID string `json:"moduleItemId"`
	SubmissionID string `json:"submissionId"`
	SubmittedBy  string `json:"submittedBy"`
}

// EmitGradePosted notifies subscribers when a grade is posted.
func EmitGradePosted(pool *pgxpool.Pool, cfg config.Config, orgID uuid.UUID, data GradePostedData) {
	EmitAsync(pool, cfg, orgID, webhooks.EventGradePosted, data)
}

// EmitEnrollmentCreated notifies subscribers when an enrollment is created.
func EmitEnrollmentCreated(pool *pgxpool.Pool, cfg config.Config, orgID uuid.UUID, data EnrollmentCreatedData) {
	EmitAsync(pool, cfg, orgID, webhooks.EventEnrollmentCreated, data)
}

// EmitAssignmentSubmitted notifies subscribers when an assignment is submitted.
func EmitAssignmentSubmitted(pool *pgxpool.Pool, cfg config.Config, orgID uuid.UUID, data AssignmentSubmittedData) {
	EmitAsync(pool, cfg, orgID, webhooks.EventAssignmentSubmitted, data)
}

// PurgeRetention deletes delivery log entries older than 90 days.
func PurgeRetention(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.FFWebhooks || pool == nil {
		return
	}
	before := now.Add(-90 * 24 * time.Hour)
	if n, err := webhooksrepo.PurgeOldDeliveries(ctx, pool, before); err != nil {
		_ = err
	} else if n > 0 {
		_ = n
	}
}
