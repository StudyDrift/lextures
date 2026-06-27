package sbg

import (
	"time"

	"github.com/google/uuid"
)

type CourseStandardRow struct {
	ID          uuid.UUID
	CourseID    uuid.UUID
	ExternalID  *string
	Description string
	Subject     *string
	GradeLevel  *string
	Position    int32
}

type ProficiencyRow struct {
	CourseID      uuid.UUID
	StudentUserID uuid.UUID
	StandardID    uuid.UUID
	Proficiency   float64
	UpdatedAt     time.Time
}
