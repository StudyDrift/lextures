package gamification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
)

// ActivityParams describes one server-side gamification activity event.
type ActivityParams struct {
	UserID       uuid.UUID
	ActivityType string
	SourceID     *uuid.UUID
	CourseID     *uuid.UUID
}

// EmitActivityAsync awards XP and updates streak when gamification is enabled.
func EmitActivityAsync(pool *pgxpool.Pool, cfg config.Config, params ActivityParams) {
	if !cfg.FFGamification || pool == nil {
		return
	}
	go func() {
		ctx := context.Background()
		if params.CourseID != nil {
			enabled, err := CourseGamificationEnabled(ctx, pool, *params.CourseID)
			if err != nil || !enabled {
				return
			}
		}
		tz, _ := userrepo.GetTimezone(ctx, pool, params.UserID)
		key := idempotencyKey(params)
		_, _ = AwardXP(ctx, pool, params.UserID, params.ActivityType, params.SourceID, params.CourseID, key, time.Now().UTC(), tz)
	}()
}

func idempotencyKey(p ActivityParams) string {
	src := "none"
	if p.SourceID != nil {
		src = p.SourceID.String()
	}
	return fmt.Sprintf("%s:%s:%s", p.ActivityType, p.UserID.String(), src)
}

// EmitModuleItemCompleted awards XP when a learner completes a module item.
func EmitModuleItemCompleted(pool *pgxpool.Pool, cfg config.Config, userID, courseID, itemID uuid.UUID) {
	cid := courseID
	iid := itemID
	EmitActivityAsync(pool, cfg, ActivityParams{
		UserID:       userID,
		ActivityType: ActivityModuleItemViewed,
		SourceID:     &iid,
		CourseID:     &cid,
	})
}

// EmitCourseCompleted awards XP when a learner finishes a self-paced course.
func EmitCourseCompleted(pool *pgxpool.Pool, cfg config.Config, userID, courseID uuid.UUID) {
	cid := courseID
	EmitActivityAsync(pool, cfg, ActivityParams{
		UserID:       userID,
		ActivityType: ActivityCourseCompleted,
		SourceID:     &cid,
		CourseID:     &cid,
	})
}

// EmitQuizPassed awards XP when a learner passes a quiz.
func EmitQuizPassed(pool *pgxpool.Pool, cfg config.Config, userID, courseID, attemptID uuid.UUID) {
	cid := courseID
	aid := attemptID
	EmitActivityAsync(pool, cfg, ActivityParams{
		UserID:       userID,
		ActivityType: ActivityQuizPassed,
		SourceID:     &aid,
		CourseID:     &cid,
	})
}

// EmitPathCompleted awards XP when a learner completes a learning path.
func EmitPathCompleted(pool *pgxpool.Pool, cfg config.Config, userID, pathID uuid.UUID) {
	pid := pathID
	EmitActivityAsync(pool, cfg, ActivityParams{
		UserID:       userID,
		ActivityType: ActivityPathCompleted,
		SourceID:     &pid,
		CourseID:     nil,
	})
}
