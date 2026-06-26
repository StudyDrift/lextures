package submissionversions

import (
	"time"

	"github.com/google/uuid"
)

type SubmissionVersionRow struct {
	ID               uuid.UUID
	VersionNumber    int32
	AttachmentFileID *uuid.UUID
	SubmittedAt      time.Time
}
