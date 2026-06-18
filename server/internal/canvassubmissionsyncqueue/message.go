package canvassubmissionsyncqueue

import "github.com/google/uuid"

// QueueMessage is the RabbitMQ payload for one Canvas grade push (includes the Canvas token).
type QueueMessage struct {
	JobID             uuid.UUID          `json:"jobId"`
	UserID            uuid.UUID          `json:"userId"`
	CourseCode        string             `json:"courseCode"`
	ItemID            uuid.UUID          `json:"itemId"`
	SubmissionID      uuid.UUID          `json:"submissionId"`
	CanvasBaseURL     string             `json:"canvasBaseUrl"`
	AccessToken       string             `json:"accessToken"`
	PointsEarned      *float64           `json:"pointsEarned,omitempty"`
	RubricScores      map[string]float64 `json:"rubricScores,omitempty"`
	InstructorComment *string            `json:"instructorComment,omitempty"`
}