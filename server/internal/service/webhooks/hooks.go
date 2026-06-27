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
	CourseID      string `json:"courseId"`
	CourseCode    string `json:"courseCode,omitempty"`
	EnrollmentID  string `json:"enrollmentId"`
	StudentUserID string `json:"studentUserId"`
	Role          string `json:"role"`
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

// AssignmentCreatedData is the assignment.created event payload.
type AssignmentCreatedData struct {
	CourseID        string `json:"courseId"`
	CourseCode      string `json:"courseCode,omitempty"`
	StructureItemID string `json:"structureItemId"`
	Title           string `json:"title"`
	DueAt           string `json:"dueAt,omitempty"`
	URL             string `json:"url,omitempty"`
}

// AssignmentDueSoonData is the assignment.due_soon event payload.
type AssignmentDueSoonData struct {
	CourseID        string `json:"courseId"`
	CourseCode      string `json:"courseCode,omitempty"`
	StructureItemID string `json:"structureItemId"`
	Title           string `json:"title"`
	DueAt           string `json:"dueAt"`
	StudentUserID   string `json:"studentUserId,omitempty"`
	URL             string `json:"url,omitempty"`
}

// GradeReleasedData is the grade.released event payload.
type GradeReleasedData struct {
	CourseID      string  `json:"courseId"`
	CourseCode    string  `json:"courseCode,omitempty"`
	ModuleItemID  string  `json:"moduleItemId"`
	StudentUserID string  `json:"studentUserId"`
	PointsEarned  float64 `json:"pointsEarned"`
	URL           string  `json:"url,omitempty"`
}

// GradeCurveAppliedData is the grade.curve.applied event payload (plan 3.17).
type GradeCurveAppliedData struct {
	CourseID     string `json:"courseId"`
	CourseCode   string `json:"courseCode,omitempty"`
	ModuleItemID string `json:"moduleItemId"`
	CurveID      string `json:"curveId"`
	Method       string `json:"method"`
	AppliedBy    string `json:"appliedBy"`
	AppliedAt    string `json:"appliedAt"`
	Affected     int    `json:"affectedCount"`
}

// AnnouncementCreatedData is the announcement.created event payload.
type AnnouncementCreatedData struct {
	CourseID   string `json:"courseId"`
	CourseCode string `json:"courseCode,omitempty"`
	Title      string `json:"title"`
	Body       string `json:"body,omitempty"`
	URL        string `json:"url,omitempty"`
}

// EmitAssignmentDueSoon notifies subscribers before an assignment is due.
func EmitAssignmentDueSoon(pool *pgxpool.Pool, cfg config.Config, orgID uuid.UUID, data AssignmentDueSoonData) {
	EmitAsync(pool, cfg, orgID, webhooks.EventAssignmentDueSoon, data)
}

// EmitGradeReleased notifies subscribers when a grade is released to a student.
func EmitGradeReleased(pool *pgxpool.Pool, cfg config.Config, orgID uuid.UUID, data GradeReleasedData) {
	EmitAsync(pool, cfg, orgID, webhooks.EventGradeReleased, data)
}

// QuizCompletedData is the quiz.completed event payload.
type QuizCompletedData struct {
	CourseID      string  `json:"courseId"`
	CourseCode    string  `json:"courseCode,omitempty"`
	ModuleItemID  string  `json:"moduleItemId"`
	AttemptID     string  `json:"attemptId"`
	StudentUserID string  `json:"studentUserId"`
	PointsEarned  float64 `json:"pointsEarned"`
	ScorePercent  float64 `json:"scorePercent"`
}

// EmitQuizCompleted notifies subscribers when a learner submits a quiz.
func EmitQuizCompleted(pool *pgxpool.Pool, cfg config.Config, orgID uuid.UUID, data QuizCompletedData) {
	EmitAsync(pool, cfg, orgID, webhooks.EventQuizCompleted, data)
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
