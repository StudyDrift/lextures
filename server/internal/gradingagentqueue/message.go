package gradingagentqueue

import "github.com/google/uuid"

// QueueMessage is the payload for one grading-agent batch item.
type QueueMessage struct {
	RunID        uuid.UUID `json:"runId"`
	ConfigID     uuid.UUID `json:"configId"`
	SubmissionID uuid.UUID `json:"submissionId"`
	CourseID     uuid.UUID `json:"courseId"`
	ItemID       uuid.UUID `json:"itemId"`
	CourseCode   string    `json:"courseCode"`
}