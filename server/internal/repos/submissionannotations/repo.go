package submissionannotations

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AnnotationRow struct {
	ID           uuid.UUID
	SubmissionID uuid.UUID
	AnnotatorID  uuid.UUID
	ClientID     string
	Page         int32
	ToolType     string
	Colour       string
	CoordsJSON   json.RawMessage
	Body         *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type AnnotationUpsertWrite struct {
	SubmissionID uuid.UUID
	AnnotatorID  uuid.UUID
	ClientID     string
	Page         int32
	ToolType     string
	Colour       string
	CoordsJSON   json.RawMessage
	Body         *string
}
