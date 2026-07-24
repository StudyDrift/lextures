package coursemodulequizzes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

// PatchWrite holds optional quiz content/settings updates for PATCH /quizzes/{item_id}.
// Pointer fields: nil = omit. For **T fields: nil = omit, non-nil pointer to nil = clear, value = set.
type PatchWrite struct {
	Title                       *string
	Markdown                    *string
	Questions                   *[]coursemodulequiz.QuizQuestion
	DueAt                       **time.Time
	AvailableFrom               **time.Time
	AvailableUntil              **time.Time
	UnlimitedAttempts           *bool
	OneQuestionAtATime          *bool
	MaxAttempts                 *int32
	GradeAttemptPolicy          *string
	PassingScorePercent         **int32
	PointsWorth                 **int32
	LateSubmissionPolicy        *string
	LatePenaltyPercent          **int32
	TimeLimitMinutes            **int32
	TimerPauseWhenTabHidden     *bool
	PerQuestionTimeLimitSeconds **int32
	ShowScoreTiming             *string
	ReviewVisibility            *string
	ReviewWhen                  *string
	ShuffleQuestions            *bool
	ShuffleChoices              *bool
	AllowBackNavigation         *bool
	QuizAccessCode              **string
	AdaptiveDifficulty          *string
	AdaptiveTopicBalance        *bool
	AdaptiveStopRule            *string
	RandomQuestionPoolCount     **int32
	LockdownMode                *string
	FocusLossThreshold          **int32
	IsAdaptive                  *bool
	AdaptiveSystemPrompt        *string
	AdaptiveSourceItemIDs       *[]uuid.UUID
	AdaptiveQuestionCount       *int32
	AdaptiveDeliveryMode        *string
	NeverDrop                   *bool
	ReplaceWithFinal            *bool
}

func (w PatchWrite) hasAny() bool {
	return w.Title != nil ||
		w.Markdown != nil ||
		w.Questions != nil ||
		w.DueAt != nil ||
		w.AvailableFrom != nil ||
		w.AvailableUntil != nil ||
		w.UnlimitedAttempts != nil ||
		w.OneQuestionAtATime != nil ||
		w.MaxAttempts != nil ||
		w.GradeAttemptPolicy != nil ||
		w.PassingScorePercent != nil ||
		w.PointsWorth != nil ||
		w.LateSubmissionPolicy != nil ||
		w.LatePenaltyPercent != nil ||
		w.TimeLimitMinutes != nil ||
		w.TimerPauseWhenTabHidden != nil ||
		w.PerQuestionTimeLimitSeconds != nil ||
		w.ShowScoreTiming != nil ||
		w.ReviewVisibility != nil ||
		w.ReviewWhen != nil ||
		w.ShuffleQuestions != nil ||
		w.ShuffleChoices != nil ||
		w.AllowBackNavigation != nil ||
		w.QuizAccessCode != nil ||
		w.AdaptiveDifficulty != nil ||
		w.AdaptiveTopicBalance != nil ||
		w.AdaptiveStopRule != nil ||
		w.RandomQuestionPoolCount != nil ||
		w.LockdownMode != nil ||
		w.FocusLossThreshold != nil ||
		w.IsAdaptive != nil ||
		w.AdaptiveSystemPrompt != nil ||
		w.AdaptiveSourceItemIDs != nil ||
		w.AdaptiveQuestionCount != nil ||
		w.AdaptiveDeliveryMode != nil ||
		w.NeverDrop != nil ||
		w.ReplaceWithFinal != nil
}

// PatchForCourseItem merges patch fields into the quiz row; returns false when not found.
func PatchForCourseItem(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, w PatchWrite) (bool, error) {
	if !w.hasAny() {
		return false, errors.New("coursemodulequizzes: patch requires at least one field")
	}
	row, err := GetForCourseItem(ctx, pool, courseID, itemID)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}

	title := row.Title
	if w.Title != nil {
		title = strings.TrimSpace(*w.Title)
	}
	dueAt := row.DueAt
	if w.DueAt != nil {
		dueAt = *w.DueAt
	}
	markdown := row.Markdown
	if w.Markdown != nil {
		markdown = *w.Markdown
	}
	questions := row.Questions
	if w.Questions != nil {
		questions = *w.Questions
	}
	if questions == nil {
		questions = []coursemodulequiz.QuizQuestion{}
	}
	availableFrom := row.AvailableFrom
	if w.AvailableFrom != nil {
		availableFrom = *w.AvailableFrom
	}
	availableUntil := row.AvailableUntil
	if w.AvailableUntil != nil {
		availableUntil = *w.AvailableUntil
	}
	unlimitedAttempts := row.UnlimitedAttempts
	if w.UnlimitedAttempts != nil {
		unlimitedAttempts = *w.UnlimitedAttempts
	}
	oneQuestionAtATime := row.OneQuestionAtATime
	if w.OneQuestionAtATime != nil {
		oneQuestionAtATime = *w.OneQuestionAtATime
	}
	maxAttempts := row.MaxAttempts
	if w.MaxAttempts != nil {
		maxAttempts = *w.MaxAttempts
	}
	gradeAttemptPolicy := row.GradeAttemptPolicy
	if w.GradeAttemptPolicy != nil {
		gradeAttemptPolicy = strings.TrimSpace(*w.GradeAttemptPolicy)
	}
	passingScorePercent := row.PassingScorePercent
	if w.PassingScorePercent != nil {
		passingScorePercent = *w.PassingScorePercent
	}
	pointsWorth := row.PointsWorth
	if w.PointsWorth != nil {
		pointsWorth = *w.PointsWorth
	}
	lateSubmissionPolicy := row.LateSubmissionPolicy
	if w.LateSubmissionPolicy != nil {
		lateSubmissionPolicy = strings.TrimSpace(*w.LateSubmissionPolicy)
	}
	latePenaltyPercent := row.LatePenaltyPercent
	if w.LatePenaltyPercent != nil {
		latePenaltyPercent = *w.LatePenaltyPercent
	}
	timeLimitMinutes := row.TimeLimitMinutes
	if w.TimeLimitMinutes != nil {
		timeLimitMinutes = *w.TimeLimitMinutes
	}
	timerPauseWhenTabHidden := row.TimerPauseWhenTabHidden
	if w.TimerPauseWhenTabHidden != nil {
		timerPauseWhenTabHidden = *w.TimerPauseWhenTabHidden
	}
	perQuestionTimeLimitSeconds := row.PerQuestionTimeLimitSeconds
	if w.PerQuestionTimeLimitSeconds != nil {
		perQuestionTimeLimitSeconds = *w.PerQuestionTimeLimitSeconds
	}
	showScoreTiming := row.ShowScoreTiming
	if w.ShowScoreTiming != nil {
		showScoreTiming = strings.TrimSpace(*w.ShowScoreTiming)
	}
	reviewVisibility := row.ReviewVisibility
	if w.ReviewVisibility != nil {
		reviewVisibility = strings.TrimSpace(*w.ReviewVisibility)
	}
	reviewWhen := row.ReviewWhen
	if w.ReviewWhen != nil {
		reviewWhen = strings.TrimSpace(*w.ReviewWhen)
	}
	shuffleQuestions := row.ShuffleQuestions
	if w.ShuffleQuestions != nil {
		shuffleQuestions = *w.ShuffleQuestions
	}
	shuffleChoices := row.ShuffleChoices
	if w.ShuffleChoices != nil {
		shuffleChoices = *w.ShuffleChoices
	}
	allowBackNavigation := row.AllowBackNavigation
	if w.AllowBackNavigation != nil {
		allowBackNavigation = *w.AllowBackNavigation
	}
	quizAccessCode := row.QuizAccessCode
	if w.QuizAccessCode != nil {
		if *w.QuizAccessCode == nil {
			quizAccessCode = nil
		} else {
			s := strings.TrimSpace(**w.QuizAccessCode)
			if s == "" {
				quizAccessCode = nil
			} else {
				quizAccessCode = &s
			}
		}
	}
	adaptiveDifficulty := row.AdaptiveDifficulty
	if w.AdaptiveDifficulty != nil {
		adaptiveDifficulty = strings.TrimSpace(*w.AdaptiveDifficulty)
	}
	adaptiveTopicBalance := row.AdaptiveTopicBalance
	if w.AdaptiveTopicBalance != nil {
		adaptiveTopicBalance = *w.AdaptiveTopicBalance
	}
	adaptiveStopRule := row.AdaptiveStopRule
	if w.AdaptiveStopRule != nil {
		adaptiveStopRule = strings.TrimSpace(*w.AdaptiveStopRule)
	}
	randomQuestionPoolCount := row.RandomQuestionPoolCount
	if w.RandomQuestionPoolCount != nil {
		randomQuestionPoolCount = *w.RandomQuestionPoolCount
	}
	lockdownMode := row.LockdownMode
	if w.LockdownMode != nil {
		lockdownMode = strings.TrimSpace(*w.LockdownMode)
	}
	focusLossThreshold := row.FocusLossThreshold
	if w.FocusLossThreshold != nil {
		focusLossThreshold = *w.FocusLossThreshold
	}
	isAdaptive := row.IsAdaptive
	if w.IsAdaptive != nil {
		isAdaptive = *w.IsAdaptive
	}
	adaptiveSystemPrompt := row.AdaptiveSystemPrompt
	if w.AdaptiveSystemPrompt != nil {
		adaptiveSystemPrompt = *w.AdaptiveSystemPrompt
	}
	adaptiveSourceItemIDs := row.AdaptiveSourceItemIDs
	if w.AdaptiveSourceItemIDs != nil {
		adaptiveSourceItemIDs = append([]uuid.UUID(nil), (*w.AdaptiveSourceItemIDs)...)
	}
	if adaptiveSourceItemIDs == nil {
		adaptiveSourceItemIDs = []uuid.UUID{}
	}
	adaptiveQuestionCount := row.AdaptiveQuestionCount
	if w.AdaptiveQuestionCount != nil {
		adaptiveQuestionCount = *w.AdaptiveQuestionCount
	}
	adaptiveDeliveryMode := row.AdaptiveDeliveryMode
	if w.AdaptiveDeliveryMode != nil {
		adaptiveDeliveryMode = strings.TrimSpace(*w.AdaptiveDeliveryMode)
	}
	neverDrop := row.NeverDrop
	if w.NeverDrop != nil {
		neverDrop = *w.NeverDrop
	}
	replaceWithFinal := row.ReplaceWithFinal
	if w.ReplaceWithFinal != nil {
		replaceWithFinal = *w.ReplaceWithFinal
	}

	qJSON, err := json.Marshal(questions)
	if err != nil {
		return false, err
	}
	srcJSON, err := json.Marshal(adaptiveSourceItemIDs)
	if err != nil {
		return false, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
UPDATE course.course_structure_items
SET title = $3,
    due_at = $4,
    updated_at = NOW()
WHERE id = $2
  AND course_id = $1
  AND kind = 'quiz'
`, courseID, itemID, title, dueAt)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}

	tag, err = tx.Exec(ctx, `
UPDATE course.module_quizzes
SET markdown = $2,
    questions_json = $3::jsonb,
    available_from = $4,
    available_until = $5,
    unlimited_attempts = $6,
    max_attempts = $7,
    grade_attempt_policy = $8,
    passing_score_percent = $9,
    points_worth = $10,
    late_submission_policy = $11,
    late_penalty_percent = $12,
    time_limit_minutes = $13,
    timer_pause_when_tab_hidden = $14,
    per_question_time_limit_seconds = $15,
    show_score_timing = $16,
    review_visibility = $17,
    review_when = $18,
    one_question_at_a_time = $19,
    shuffle_questions = $20,
    shuffle_choices = $21,
    allow_back_navigation = $22,
    quiz_access_code = $23,
    adaptive_difficulty = $24,
    adaptive_topic_balance = $25,
    adaptive_stop_rule = $26,
    random_question_pool_count = $27,
    lockdown_mode = $28::course.lockdown_mode,
    focus_loss_threshold = $29,
    is_adaptive = $30,
    adaptive_system_prompt = $31,
    adaptive_source_item_ids = $32::jsonb,
    adaptive_question_count = $33,
    adaptive_delivery_mode = $34,
    never_drop = $35,
    replace_with_final = $36,
    updated_at = NOW()
WHERE structure_item_id = $1
`, itemID, markdown, qJSON, availableFrom, availableUntil, unlimitedAttempts, maxAttempts,
		gradeAttemptPolicy, passingScorePercent, pointsWorth, lateSubmissionPolicy, latePenaltyPercent,
		timeLimitMinutes, timerPauseWhenTabHidden, perQuestionTimeLimitSeconds, showScoreTiming,
		reviewVisibility, reviewWhen, oneQuestionAtATime, shuffleQuestions, shuffleChoices,
		allowBackNavigation, quizAccessCode, adaptiveDifficulty, adaptiveTopicBalance, adaptiveStopRule,
		randomQuestionPoolCount, lockdownMode, focusLossThreshold, isAdaptive, adaptiveSystemPrompt,
		srcJSON, adaptiveQuestionCount, adaptiveDeliveryMode, neverDrop, replaceWithFinal)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
