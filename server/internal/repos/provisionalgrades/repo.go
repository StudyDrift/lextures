package provisionalgrades

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ProvisionalGradeRow struct {
	ID           uuid.UUID
	SubmissionID uuid.UUID
	GraderID     uuid.UUID
	Score        float64
	RubricData   json.RawMessage
	SubmittedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
