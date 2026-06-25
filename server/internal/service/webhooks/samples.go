package webhooksvc

import (
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/webhooks"
)

// SampleDataForEvent returns synthetic data for test deliveries.
func SampleDataForEvent(eventType webhooks.EventType) map[string]any {
	switch eventType {
	case webhooks.EventGradePosted:
		return map[string]any{
			"courseId":      uuid.New().String(),
			"moduleItemId":  uuid.New().String(),
			"studentUserId": uuid.New().String(),
			"pointsEarned":  92.5,
		}
	case webhooks.EventEnrollmentCreated:
		return map[string]any{
			"courseId":      uuid.New().String(),
			"enrollmentId":  uuid.New().String(),
			"studentUserId": uuid.New().String(),
			"role":          "student",
		}
	case webhooks.EventAssignmentSubmitted:
		return map[string]any{
			"courseId":     uuid.New().String(),
			"moduleItemId": uuid.New().String(),
			"submissionId": uuid.New().String(),
			"submittedBy":  uuid.New().String(),
		}
	case webhooks.EventQuizCompleted:
		return map[string]any{
			"courseId":      uuid.New().String(),
			"moduleItemId":  uuid.New().String(),
			"attemptId":     uuid.New().String(),
			"studentUserId": uuid.New().String(),
			"pointsEarned":  8.0,
			"scorePercent":  80.0,
		}
	default:
		return map[string]any{"message": "test event"}
	}
}
