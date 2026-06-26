package learnermodel

import (
	"time"

	"github.com/google/uuid"
)

type LearnerConceptStateRow struct {
	UserID    uuid.UUID
	CourseID  uuid.UUID
	ConceptID uuid.UUID
	Mastery   float64
	Engine    string
	UpdatedAt time.Time
}
