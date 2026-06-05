package readingprefs

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UI mode constants (plan 13.11).
const (
	UIModeK2        = "k2"
	UIModeElementary = "elementary"
	UIModeStandard   = "standard"
)

var validUIMode = map[string]bool{
	UIModeK2:        true,
	UIModeElementary: true,
	UIModeStandard:   true,
}

// ValidUIMode returns true for k2, elementary, or standard.
func ValidUIMode(m string) bool { return validUIMode[m] }

// GradeToUIMode derives the effective UI mode from a grade_level string.
// K/1/2 → k2; 3/4/5 → elementary; null or any other value → standard.
func GradeToUIMode(gradeLevel *string) string {
	if gradeLevel == nil {
		return UIModeStandard
	}
	switch *gradeLevel {
	case "K", "1", "2":
		return UIModeK2
	case "3", "4", "5":
		return UIModeElementary
	default:
		return UIModeStandard
	}
}

// EffectiveUIMode returns the active UI mode.
// A non-nil, non-empty override beats grade-level derivation.
func EffectiveUIMode(gradeLevel *string, override *string) string {
	if override != nil && validUIMode[*override] {
		return *override
	}
	return GradeToUIMode(gradeLevel)
}

// SetUIModeOverride upserts the ui_mode_override for the given student.
// Pass nil to clear the override (restores grade-level derivation).
func SetUIModeOverride(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, mode *string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO settings.user_reading_preferences (user_id, ui_mode_override, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (user_id) DO UPDATE
    SET ui_mode_override = EXCLUDED.ui_mode_override,
        updated_at       = now()
`, studentID, mode)
	return err
}

// GetUIModeOverride returns the stored override for a user, or nil if none.
func GetUIModeOverride(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*string, error) {
	var override *string
	err := pool.QueryRow(ctx, `
SELECT ui_mode_override FROM settings.user_reading_preferences WHERE user_id = $1
`, userID).Scan(&override)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return override, nil
}
