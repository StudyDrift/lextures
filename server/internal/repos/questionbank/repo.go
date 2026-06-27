package questionbank

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type QuestionRow struct {
	ID            uuid.UUID
	CourseID      uuid.UUID
	QuestionType  string
	Stem          string
	Options       json.RawMessage
	CorrectAnswer json.RawMessage
	Explanation   *string
	Points        float64
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
