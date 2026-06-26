package canvassubmissionsyncqueue

import "github.com/google/uuid"

// QueueMessage is the RabbitMQ payload for one Canvas grade push (includes the Canvas token).
type QueueMessage struct {
	JobID             uuid.UUID          `json:"jobId"`
	UserID            uuid.UUID          `json:"userId"`
	CourseCode        string             `json:"courseCode"`
	// ItemKind selects the sync executor: empty/"assignment" pushes an assignment submission grade,
	// "quiz" pushes a quiz attempt's gradebook score. SubmissionID holds the assignment submission id
	// for assignments and the quiz attempt id for quizzes.
	ItemKind          string             `json:"itemKind,omitempty"`
	ItemID            uuid.UUID          `json:"itemId"`
	SubmissionID      uuid.UUID          `json:"submissionId"`
	CanvasBaseURL     string             `json:"canvasBaseUrl"`
	AccessToken       string             `json:"accessToken"`
	PointsEarned      *float64           `json:"pointsEarned,omitempty"`
	RubricScores      map[string]float64 `json:"rubricScores,omitempty"`
	InstructorComment *string            `json:"instructorComment,omitempty"`
}