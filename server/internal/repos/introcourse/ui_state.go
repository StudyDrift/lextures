package introcourse

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// UIState holds per-user intro course onboarding surface flags (IC06).
type UIState struct {
	WelcomeBannerDismissed bool
	CelebrationSeen        bool
}

// GetUIState returns dismiss/seen flags for userID.
func GetUIState(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, userID uuid.UUID) (UIState, error) {
	var bannerDismissed, celebrationSeen *time.Time
	err := q.QueryRow(ctx, `
SELECT welcome_banner_dismissed_at, celebration_seen_at
FROM settings.intro_course_user_ui_state
WHERE user_id = $1
`, userID).Scan(&bannerDismissed, &celebrationSeen)
	if err == pgx.ErrNoRows {
		return UIState{}, nil
	}
	if err != nil {
		return UIState{}, err
	}
	return UIState{
		WelcomeBannerDismissed: bannerDismissed != nil,
		CelebrationSeen:        celebrationSeen != nil,
	}, nil
}

// SetWelcomeBannerDismissed records that the first-login welcome banner was dismissed.
func SetWelcomeBannerDismissed(ctx context.Context, q interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}, userID uuid.UUID, now time.Time) error {
	_, err := q.Exec(ctx, `
INSERT INTO settings.intro_course_user_ui_state (user_id, welcome_banner_dismissed_at, updated_at)
VALUES ($1, $2, $2)
ON CONFLICT (user_id) DO UPDATE SET
    welcome_banner_dismissed_at = COALESCE(settings.intro_course_user_ui_state.welcome_banner_dismissed_at, EXCLUDED.welcome_banner_dismissed_at),
    updated_at = EXCLUDED.updated_at
`, userID, now)
	return err
}

// SetCelebrationSeen records that the completion celebration was shown and dismissed.
func SetCelebrationSeen(ctx context.Context, q interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}, userID uuid.UUID, now time.Time) error {
	_, err := q.Exec(ctx, `
INSERT INTO settings.intro_course_user_ui_state (user_id, celebration_seen_at, updated_at)
VALUES ($1, $2, $2)
ON CONFLICT (user_id) DO UPDATE SET
    celebration_seen_at = COALESCE(settings.intro_course_user_ui_state.celebration_seen_at, EXCLUDED.celebration_seen_at),
    updated_at = EXCLUDED.updated_at
`, userID, now)
	return err
}