package adaptivecontent

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/google/uuid"
)

// IsHoldout returns true when the enrollment is deterministically assigned to the
// control group for this unit. Assignment is stable across visits: hash(enrollment|unit)
// mod 100 falls in [0, holdoutPercent). holdoutPercent is clamped to [0, 50].
func IsHoldout(enrollmentID, unitID uuid.UUID, holdoutPercent int16) bool {
	if holdoutPercent <= 0 {
		return false
	}
	if holdoutPercent > 50 {
		holdoutPercent = 50
	}
	// Stable bytes: enrollment UUID + separator + unit UUID.
	sum := sha256.Sum256([]byte(enrollmentID.String() + "|" + unitID.String() + "|ace-holdout-v1"))
	// First 8 bytes → uint64 → bucket 0..99
	n := binary.BigEndian.Uint64(sum[:8])
	bucket := int(n % 100)
	return bucket < int(holdoutPercent)
}

// HoldoutBucket returns the 0..99 bucket used for holdout decisions (for tests/debug).
func HoldoutBucket(enrollmentID, unitID uuid.UUID) int {
	sum := sha256.Sum256([]byte(enrollmentID.String() + "|" + unitID.String() + "|ace-holdout-v1"))
	n := binary.BigEndian.Uint64(sum[:8])
	return int(n % 100)
}
