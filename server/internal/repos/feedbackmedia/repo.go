package feedbackmedia

import (
	"time"

	"github.com/google/uuid"
)

type FeedbackMediaRow struct {
	ID               uuid.UUID
	SubmissionID     uuid.UUID
	CourseID         uuid.UUID
	ModuleItemID     uuid.UUID
	UploaderID       uuid.UUID
	MediaType        string
	MimeType         string
	StorageKey       string
	ByteSize         int64
	DurationSecs     *int32
	CaptionStatus    string
	CaptionKey       *string
	UploadComplete   bool
	ExpectedByteSize *int64
	BytesReceived    int64
	CreatedAt        time.Time
	DeletedAt        *time.Time
}
