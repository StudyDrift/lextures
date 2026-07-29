package coursemodulequizzes

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

// ImportQuizBody is the settings + questions payload for course JSON import.
type ImportQuizBody struct {
	Markdown                    string
	Questions                   []coursemodulequiz.QuizQuestion
	AvailableFrom               *time.Time
	AvailableUntil              *time.Time
	UnlimitedAttempts           bool
	MaxAttempts                 int32
	GradeAttemptPolicy          string
	PassingScorePercent         *int32
	PointsWorth                 *int32
	LateSubmissionPolicy        string
	LatePenaltyPercent          *int32
	TimeLimitMinutes            *int32
	TimerPauseWhenTabHidden     bool
	PerQuestionTimeLimitSeconds *int32
	ShowScoreTiming             string
	ReviewVisibility            string
	ReviewWhen                  string
	OneQuestionAtATime          bool
	ShuffleQuestions            bool
	ShuffleChoices              bool
	AllowBackNavigation         bool
	QuizAccessCode              *string
	AdaptiveDifficulty          string
	AdaptiveTopicBalance        bool
	AdaptiveStopRule            string
	RandomQuestionPoolCount     *int32
	LockdownMode                string
	FocusLossThreshold          *int32
	IsAdaptive                  bool
	AdaptiveSystemPrompt        string
	AdaptiveSourceItemIDs       []uuid.UUID
	AdaptiveQuestionCount       int32
	AdaptiveDeliveryMode        string
}

// UpsertImportBody inserts or updates a quiz body for course import.
func UpsertImportBody(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, body ImportQuizBody) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	if body.Questions == nil {
		body.Questions = []coursemodulequiz.QuizQuestion{}
	}
	if body.AdaptiveSourceItemIDs == nil {
		body.AdaptiveSourceItemIDs = []uuid.UUID{}
	}
	if body.GradeAttemptPolicy == "" {
		body.GradeAttemptPolicy = "latest"
	}
	if body.LateSubmissionPolicy == "" {
		body.LateSubmissionPolicy = "allow"
	}
	if body.ShowScoreTiming == "" {
		body.ShowScoreTiming = "immediate"
	}
	if body.ReviewVisibility == "" {
		body.ReviewVisibility = "full"
	}
	if body.ReviewWhen == "" {
		body.ReviewWhen = "always"
	}
	if body.AdaptiveDifficulty == "" {
		body.AdaptiveDifficulty = "standard"
	}
	if body.AdaptiveStopRule == "" {
		body.AdaptiveStopRule = "fixed_count"
	}
	if body.LockdownMode == "" {
		body.LockdownMode = "standard"
	}
	if body.AdaptiveDeliveryMode == "" {
		body.AdaptiveDeliveryMode = "ai"
	}
	if body.MaxAttempts <= 0 {
		body.MaxAttempts = 1
	}

	qJSON, err := json.Marshal(body.Questions)
	if err != nil {
		return err
	}
	srcJSON, err := json.Marshal(body.AdaptiveSourceItemIDs)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
INSERT INTO course.module_quizzes AS m (
	structure_item_id, markdown, questions_json, updated_at,
	available_from, available_until, unlimited_attempts, max_attempts,
	grade_attempt_policy, passing_score_percent, points_worth, late_submission_policy, late_penalty_percent,
	time_limit_minutes, timer_pause_when_tab_hidden, per_question_time_limit_seconds,
	show_score_timing, review_visibility, review_when,
	one_question_at_a_time, shuffle_questions, shuffle_choices, allow_back_navigation,
	quiz_access_code, adaptive_difficulty, adaptive_topic_balance, adaptive_stop_rule,
	random_question_pool_count,
	lockdown_mode, focus_loss_threshold,
	is_adaptive, adaptive_system_prompt, adaptive_source_item_ids, adaptive_question_count,
	adaptive_delivery_mode
)
SELECT c.id, $3, $4::jsonb, NOW(),
	$5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28,
	$29::course.lockdown_mode, $30,
	$31, $32, $33::jsonb, $34, $35
FROM course.course_structure_items c
WHERE c.id = $1 AND c.course_id = $2 AND c.kind = 'quiz'
ON CONFLICT (structure_item_id) DO UPDATE SET
	markdown = EXCLUDED.markdown,
	questions_json = EXCLUDED.questions_json,
	settings_version = m.settings_version + 1,
	updated_at = NOW(),
	available_from = EXCLUDED.available_from,
	available_until = EXCLUDED.available_until,
	unlimited_attempts = EXCLUDED.unlimited_attempts,
	max_attempts = EXCLUDED.max_attempts,
	grade_attempt_policy = EXCLUDED.grade_attempt_policy,
	passing_score_percent = EXCLUDED.passing_score_percent,
	points_worth = EXCLUDED.points_worth,
	late_submission_policy = EXCLUDED.late_submission_policy,
	late_penalty_percent = EXCLUDED.late_penalty_percent,
	time_limit_minutes = EXCLUDED.time_limit_minutes,
	timer_pause_when_tab_hidden = EXCLUDED.timer_pause_when_tab_hidden,
	per_question_time_limit_seconds = EXCLUDED.per_question_time_limit_seconds,
	show_score_timing = EXCLUDED.show_score_timing,
	review_visibility = EXCLUDED.review_visibility,
	review_when = EXCLUDED.review_when,
	one_question_at_a_time = EXCLUDED.one_question_at_a_time,
	shuffle_questions = EXCLUDED.shuffle_questions,
	shuffle_choices = EXCLUDED.shuffle_choices,
	allow_back_navigation = EXCLUDED.allow_back_navigation,
	quiz_access_code = EXCLUDED.quiz_access_code,
	adaptive_difficulty = EXCLUDED.adaptive_difficulty,
	adaptive_topic_balance = EXCLUDED.adaptive_topic_balance,
	adaptive_stop_rule = EXCLUDED.adaptive_stop_rule,
	random_question_pool_count = EXCLUDED.random_question_pool_count,
	lockdown_mode = EXCLUDED.lockdown_mode,
	focus_loss_threshold = EXCLUDED.focus_loss_threshold,
	is_adaptive = EXCLUDED.is_adaptive,
	adaptive_system_prompt = EXCLUDED.adaptive_system_prompt,
	adaptive_source_item_ids = EXCLUDED.adaptive_source_item_ids,
	adaptive_question_count = EXCLUDED.adaptive_question_count,
	adaptive_delivery_mode = EXCLUDED.adaptive_delivery_mode
`, itemID, courseID, body.Markdown, qJSON,
		body.AvailableFrom, body.AvailableUntil, body.UnlimitedAttempts, body.MaxAttempts,
		body.GradeAttemptPolicy, body.PassingScorePercent, body.PointsWorth, body.LateSubmissionPolicy, body.LatePenaltyPercent,
		body.TimeLimitMinutes, body.TimerPauseWhenTabHidden, body.PerQuestionTimeLimitSeconds,
		body.ShowScoreTiming, body.ReviewVisibility, body.ReviewWhen,
		body.OneQuestionAtATime, body.ShuffleQuestions, body.ShuffleChoices, body.AllowBackNavigation,
		body.QuizAccessCode, body.AdaptiveDifficulty, body.AdaptiveTopicBalance, body.AdaptiveStopRule,
		body.RandomQuestionPoolCount,
		body.LockdownMode, body.FocusLossThreshold,
		body.IsAdaptive, body.AdaptiveSystemPrompt, srcJSON, body.AdaptiveQuestionCount,
		body.AdaptiveDeliveryMode)
	return err
}
