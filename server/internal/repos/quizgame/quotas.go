package quizgame

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrConcurrentGamesQuota = errors.New("quizgame: concurrent games quota exceeded")
	ErrPlayersPerGameQuota  = errors.New("quizgame: players per game quota exceeded")
	ErrKitsPerCourseQuota   = errors.New("quizgame: kits per course quota exceeded")
	ErrAIGenerationQuota    = errors.New("quizgame: AI generation daily quota exceeded")
)

// CountConcurrentLiveGames returns lobby/running/paused sessions for an org (or tenant-wide when orgID is nil).
func CountConcurrentLiveGames(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID) (int, error) {
	var n int
	var err error
	if orgID == nil {
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*)::int
			FROM quizgame.sessions
			WHERE status IN ('lobby', 'running', 'paused')
		`).Scan(&n)
	} else {
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*)::int
			FROM quizgame.sessions s
			INNER JOIN course.courses c ON c.id = s.course_id
			WHERE c.org_id = $1 AND s.status IN ('lobby', 'running', 'paused')
		`, *orgID).Scan(&n)
	}
	return n, err
}

// CountKitsInCourse returns non-archived kits for a course.
func CountKitsInCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM quizgame.kits k
		INNER JOIN course.courses c ON c.id = k.course_id
		WHERE c.course_code = $1 AND k.archived = FALSE AND k.is_template = FALSE
	`, courseCode).Scan(&n)
	return n, err
}

// CountAIGenerationsToday returns generation jobs created today for an org.
func CountAIGenerationsToday(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM quizgame.generation_jobs j
		INNER JOIN course.courses c ON c.id = j.course_id
		WHERE c.org_id = $1
		  AND j.created_at >= date_trunc('day', NOW() AT TIME ZONE 'UTC')
	`, orgID).Scan(&n)
	return n, err
}

// CheckConcurrentGamesQuota refuses when at/over the effective concurrent-games cap.
func CheckConcurrentGamesQuota(ctx context.Context, pool *pgxpool.Pool, courseCode string) error {
	eff, err := ResolveEffectiveSettingsForCourse(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	if eff.MaxConcurrentGames == nil {
		return nil
	}
	orgID, err := OrgIDForCourse(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	var orgPtr *uuid.UUID
	if orgID != uuid.Nil {
		orgPtr = &orgID
	}
	n, err := CountConcurrentLiveGames(ctx, pool, orgPtr)
	if err != nil {
		return err
	}
	if n >= *eff.MaxConcurrentGames {
		return fmt.Errorf("%w: %d/%d", ErrConcurrentGamesQuota, n, *eff.MaxConcurrentGames)
	}
	return nil
}

// CheckPlayersPerGameQuota refuses when the session is at the player cap (rejoins excluded by caller).
func CheckPlayersPerGameQuota(ctx context.Context, pool *pgxpool.Pool, courseCode, sessionID string) error {
	eff, err := ResolveEffectiveSettingsForCourse(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	n, err := CountActivePlayers(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	if n >= eff.MaxPlayersPerGame {
		return fmt.Errorf("%w: %d/%d", ErrPlayersPerGameQuota, n, eff.MaxPlayersPerGame)
	}
	return nil
}

// CheckKitsPerCourseQuota refuses when the course is at the kit cap.
func CheckKitsPerCourseQuota(ctx context.Context, pool *pgxpool.Pool, courseCode string) error {
	eff, err := ResolveEffectiveSettingsForCourse(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	if eff.MaxKitsPerCourse == nil {
		return nil
	}
	n, err := CountKitsInCourse(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	if n >= *eff.MaxKitsPerCourse {
		return fmt.Errorf("%w: %d/%d", ErrKitsPerCourseQuota, n, *eff.MaxKitsPerCourse)
	}
	return nil
}

// CheckAIGenerationQuota refuses when the org has hit its daily AI generation budget.
func CheckAIGenerationQuota(ctx context.Context, pool *pgxpool.Pool, courseCode string) error {
	eff, err := ResolveEffectiveSettingsForCourse(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	if !eff.AIGenerationEnabled {
		// Section default off — still allow when platform ff is on; callers gate on FFIqAiGeneration.
		// Quota only applies when a daily budget is set.
	}
	if eff.AIGenerationsPerDay == nil {
		return nil
	}
	orgID, err := OrgIDForCourse(ctx, pool, courseCode)
	if err != nil {
		return err
	}
	n, err := CountAIGenerationsToday(ctx, pool, orgID)
	if err != nil {
		return err
	}
	if n >= *eff.AIGenerationsPerDay {
		return fmt.Errorf("%w: %d/%d", ErrAIGenerationQuota, n, *eff.AIGenerationsPerDay)
	}
	return nil
}
