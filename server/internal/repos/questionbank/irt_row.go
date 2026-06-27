package questionbank

import (
	"github.com/google/uuid"
)

// IRTFields holds 2PL column subset used for CAT and calibration (port of Rust question bank IRT).
type IRTFields struct {
	ID        uuid.UUID
	CourseID  uuid.UUID
	IRTStatus string
	IRTA      *float64
	IRTB      *float64
}
