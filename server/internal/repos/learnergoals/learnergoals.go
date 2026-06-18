// Package learnergoals stores self-learner onboarding goals and placement results (plan 15.11).
package learnergoals

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is the persisted learner goals record for a user.
type Row struct {
	ID                      uuid.UUID  `json:"id"`
	UserID                  uuid.UUID  `json:"userId"`
	Topic                   string     `json:"topic"`
	GoalText                *string    `json:"goalText,omitempty"`
	TargetDate              *time.Time `json:"targetDate,omitempty"`
	DailyMinutes            int        `json:"dailyMinutes"`
	PriorKnowledgeLevel     string     `json:"priorKnowledgeLevel"`
	DiagnosticScore         *float64   `json:"diagnosticScore,omitempty"`
	DiagnosticSkipped       bool       `json:"diagnosticSkipped"`
	OnboardingStep          int        `json:"onboardingStep"`
	OnboardingCompleted     bool       `json:"onboardingCompleted"`
	ReminderOptIn           bool       `json:"reminderOptIn"`
	ReminderTime            *string    `json:"reminderTime,omitempty"`
	RecommendedCourseCode   *string    `json:"recommendedCourseCode,omitempty"`
	RecommendedCourseTitle  *string    `json:"recommendedCourseTitle,omitempty"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

const selectCols = `
SELECT id, user_id, topic, goal_text, target_date, daily_minutes,
       prior_knowledge_level, diagnostic_score, diagnostic_skipped,
       onboarding_step, onboarding_completed, reminder_opt_in,
       to_char(reminder_time, 'HH24:MI') AS reminder_time,
       recommended_course_code, recommended_course_title,
       created_at, updated_at
FROM "user".learner_goals
`

func scanRow(row pgx.Row) (*Row, error) {
	var r Row
	var goalText, courseCode, courseTitle, reminderTime *string
	var targetDate *time.Time
	var diagnosticScore *float64
	err := row.Scan(
		&r.ID, &r.UserID, &r.Topic, &goalText, &targetDate, &r.DailyMinutes,
		&r.PriorKnowledgeLevel, &diagnosticScore, &r.DiagnosticSkipped,
		&r.OnboardingStep, &r.OnboardingCompleted, &r.ReminderOptIn,
		&reminderTime, &courseCode, &courseTitle,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.GoalText = goalText
	r.TargetDate = targetDate
	r.DiagnosticScore = diagnosticScore
	r.ReminderTime = reminderTime
	r.RecommendedCourseCode = courseCode
	r.RecommendedCourseTitle = courseTitle
	return &r, nil
}

// Get returns goals for a user, or nil when none exist.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Row, error) {
	row, err := scanRow(pool.QueryRow(ctx, selectCols+` WHERE user_id = $1`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// Ensure creates an empty goals row when missing.
func Ensure(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Row, error) {
	existing, err := Get(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	_, err = pool.Exec(ctx, `
INSERT INTO "user".learner_goals (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO NOTHING
`, userID)
	if err != nil {
		return nil, err
	}
	return Get(ctx, pool, userID)
}

// StepPatch holds incremental onboarding step updates.
type StepPatch struct {
	Step                *int
	Topic               *string
	GoalText            *string
	ClearGoalText       bool
	TargetDate          *time.Time
	ClearTargetDate     bool
	DailyMinutes        *int
	PriorKnowledgeLevel *string
	DiagnosticScore     *float64
	DiagnosticSkipped   *bool
	ReminderOptIn       *bool
	ReminderTime        *string
	ClearReminderTime   bool
	OnboardingCompleted *bool
	RecommendedCourse   *RecommendedCourse
}

// RecommendedCourse stores the placement recommendation.
type RecommendedCourse struct {
	Code  string
	Title string
}

// ApplyStep merges onboarding step data into the user's goals row.
func ApplyStep(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, p StepPatch) (*Row, error) {
	if _, err := Ensure(ctx, pool, userID); err != nil {
		return nil, err
	}
	sets := []string{"updated_at = NOW()"}
	args := []any{userID}
	n := 2

	add := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, n))
		args = append(args, val)
		n++
	}

	if p.Step != nil {
		add("onboarding_step", *p.Step)
	}
	if p.Topic != nil {
		add("topic", strings.TrimSpace(*p.Topic))
	}
	if p.GoalText != nil {
		add("goal_text", strings.TrimSpace(*p.GoalText))
	}
	if p.ClearGoalText {
		add("goal_text", nil)
	}
	if p.TargetDate != nil {
		add("target_date", *p.TargetDate)
	}
	if p.ClearTargetDate {
		add("target_date", nil)
	}
	if p.DailyMinutes != nil {
		add("daily_minutes", *p.DailyMinutes)
	}
	if p.PriorKnowledgeLevel != nil {
		level := strings.TrimSpace(*p.PriorKnowledgeLevel)
		if level != "beginner" && level != "intermediate" && level != "advanced" {
			return nil, fmt.Errorf("invalid prior knowledge level")
		}
		add("prior_knowledge_level", level)
	}
	if p.DiagnosticScore != nil {
		add("diagnostic_score", *p.DiagnosticScore)
	}
	if p.DiagnosticSkipped != nil {
		add("diagnostic_skipped", *p.DiagnosticSkipped)
	}
	if p.ReminderOptIn != nil {
		add("reminder_opt_in", *p.ReminderOptIn)
	}
	if p.ReminderTime != nil {
		rt := strings.TrimSpace(*p.ReminderTime)
		if rt == "" {
			add("reminder_time", nil)
		} else {
			add("reminder_time", rt)
		}
	}
	if p.ClearReminderTime {
		add("reminder_time", nil)
	}
	if p.OnboardingCompleted != nil {
		add("onboarding_completed", *p.OnboardingCompleted)
	}
	if p.RecommendedCourse != nil {
		add("recommended_course_code", p.RecommendedCourse.Code)
		add("recommended_course_title", p.RecommendedCourse.Title)
	}

	q := fmt.Sprintf(`UPDATE "user".learner_goals SET %s WHERE user_id = $1`, strings.Join(sets, ", "))
	if _, err := pool.Exec(ctx, q, args...); err != nil {
		return nil, err
	}
	return Get(ctx, pool, userID)
}

// GoalsPatch holds post-onboarding goal updates.
type GoalsPatch struct {
	Topic               *string
	GoalText            *string
	ClearGoalText       bool
	TargetDate          *time.Time
	ClearTargetDate     bool
	DailyMinutes        *int
	PriorKnowledgeLevel *string
	ReminderOptIn       *bool
	ReminderTime        *string
	ClearReminderTime   bool
}

// PatchGoals updates learner goals after onboarding.
func PatchGoals(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, p GoalsPatch) (*Row, error) {
	return ApplyStep(ctx, pool, userID, StepPatch{
		Topic:               p.Topic,
		GoalText:            p.GoalText,
		TargetDate:          p.TargetDate,
		DailyMinutes:        p.DailyMinutes,
		PriorKnowledgeLevel: p.PriorKnowledgeLevel,
		ReminderOptIn:       p.ReminderOptIn,
		ReminderTime:        p.ReminderTime,
	})
}
