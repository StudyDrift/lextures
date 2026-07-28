package flashcards

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	srsrepo "github.com/lextures/lextures/server/internal/repos/srs"
	"github.com/lextures/lextures/server/internal/service/srs"
)

// Fixed namespace for deterministic synthetic question ids (instance + card + side).
var questionNamespace = uuid.MustParse("c7f23a10-9b4e-4d8a-a1c2-0e6f4b8d2a31")

// QuestionIDFor returns a stable UUID for a deck card side.
func QuestionIDFor(instanceID uuid.UUID, cardID, side string) uuid.UUID {
	key := instanceID.String() + "|" + strings.TrimSpace(cardID) + "|" + strings.TrimSpace(side)
	return uuid.NewSHA1(questionNamespace, []byte(key))
}

// CourseSRSEnabled reports whether the course has spaced repetition turned on.
func CourseSRSEnabled(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (bool, error) {
	if pool == nil {
		return false, nil
	}
	var on bool
	err := pool.QueryRow(ctx, `SELECT srs_enabled FROM course.courses WHERE id = $1`, courseID).Scan(&on)
	if err != nil {
		return false, err
	}
	return on, nil
}

// SRSActive is true when both platform and course flags allow SRS writes.
func SRSActive(platformEnabled, courseEnabled bool) bool {
	return platformEnabled && courseEnabled
}

// EnsureSyntheticQuestion upserts a short_answer question-bank row for SRS.
func EnsureSyntheticQuestion(ctx context.Context, pool *pgxpool.Pool, courseID, instanceID uuid.UUID, card Card, side string) (uuid.UUID, error) {
	qid := QuestionIDFor(instanceID, card.ID, side)
	stem, answer := card.Front, card.Back
	if side == SideReverse {
		stem, answer = card.Back, card.Front
	}
	correct, _ := json.Marshal(map[string]any{"acceptedAnswers": []string{answer}})
	meta, _ := json.Marshal(map[string]any{
		"contentTool": "flashcards",
		"instanceId":  instanceID.String(),
		"cardId":      card.ID,
		"side":        side,
	})
	expl := answer
	_, err := pool.Exec(ctx, `
INSERT INTO course.questions (
	id, course_id, question_type, stem, options, correct_answer, explanation,
	points, status, shared, source, metadata, created_by, is_published, srs_eligible
)
VALUES (
	$1, $2, 'short_answer'::course.question_type, $3, NULL, $4, $5,
	0, 'active'::course.question_status, FALSE, $6, $7, NULL, TRUE, TRUE
)
ON CONFLICT (id) DO UPDATE SET
	stem = EXCLUDED.stem,
	correct_answer = EXCLUDED.correct_answer,
	explanation = EXCLUDED.explanation,
	metadata = EXCLUDED.metadata,
	srs_eligible = TRUE,
	is_published = TRUE,
	status = 'active'::course.question_status,
	updated_at = NOW()
`, qid, courseID, stem, correct, expl, Source, meta)
	if err != nil {
		return uuid.Nil, err
	}
	return qid, nil
}

// SubmitCardRating writes an SRS review when enabled. Returns nextDueAt when successful.
// Failures are returned but callers must still persist tool-state ratings.
func SubmitCardRating(
	ctx context.Context,
	pool *pgxpool.Pool,
	platformEnabled bool,
	courseEnabled bool,
	courseID uuid.UUID,
	instanceID uuid.UUID,
	userID uuid.UUID,
	card Card,
	side string,
	rating Rating,
) (*time.Time, error) {
	if pool == nil || !SRSActive(platformEnabled, courseEnabled) {
		return nil, nil
	}
	if !ValidSide(side) {
		side = SideForward
	}
	qid, err := EnsureSyntheticQuestion(ctx, pool, courseID, instanceID, card, side)
	if err != nil {
		return nil, err
	}
	resp, err := srs.SubmitReview(ctx, pool, platformEnabled, userID, userID, srs.SubmitReviewBody{
		QuestionID: qid,
		Grade:      string(rating),
	})
	if err != nil {
		return nil, err
	}
	t := resp.NextReviewAt
	return &t, nil
}

// LoadDueMap loads SRS scheduling info for every card/side in the deck.
func LoadDueMap(
	ctx context.Context,
	pool *pgxpool.Pool,
	platformEnabled bool,
	courseEnabled bool,
	instanceID uuid.UUID,
	userID uuid.UUID,
	cfg Config,
) (map[string]CardDueInfo, error) {
	out := map[string]CardDueInfo{}
	if pool == nil || !SRSActive(platformEnabled, courseEnabled) || userID == uuid.Nil {
		return out, nil
	}
	now := time.Now().UTC()
	for _, c := range cfg.Cards {
		sides := []string{SideForward}
		if cfg.ReversePractice {
			sides = append(sides, SideReverse)
		}
		for _, side := range sides {
			qid := QuestionIDFor(instanceID, c.ID, side)
			key := c.ID + "|" + side
			row, err := getStateForUserQuestion(ctx, pool, userID, qid)
			if err != nil {
				return nil, err
			}
			if row == nil {
				out[key] = CardDueInfo{CardID: c.ID, Side: side, IsNew: true}
				continue
			}
			info := CardDueInfo{CardID: c.ID, Side: side, NextDueAt: &row.NextReviewAt}
			if !row.NextReviewAt.After(now) {
				info.IsDue = true
			}
			out[key] = info
		}
	}
	return out, nil
}

func getStateForUserQuestion(ctx context.Context, pool *pgxpool.Pool, userID, questionID uuid.UUID) (*srsrepo.ItemStateRow, error) {
	var r srsrepo.ItemStateRow
	err := pool.QueryRow(ctx, `
SELECT
	s.id, s.user_id, s.question_id, s.algorithm::text,
	(s.interval_days)::float8, s.repetition, (s.easiness_factor)::float8,
	s.next_review_at, s.due_count, s.suppressed_until
FROM course.srs_item_states s
WHERE s.user_id = $1 AND s.question_id = $2
`, userID, questionID).Scan(
		&r.ID, &r.UserID, &r.QuestionID, &r.Algorithm, &r.IntervalDays, &r.Repetition, &r.EasinessFactor, &r.NextReviewAt, &r.DueCount, &r.SuppressedUntil,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ClearSchedulingForDeck deletes SRS states and review events for this deck's cards for one user.
func ClearSchedulingForDeck(ctx context.Context, pool *pgxpool.Pool, instanceID, userID uuid.UUID, cfg Config) error {
	if pool == nil || userID == uuid.Nil {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(cfg.Cards)*2)
	for _, c := range cfg.Cards {
		ids = append(ids, QuestionIDFor(instanceID, c.ID, SideForward))
		if cfg.ReversePractice {
			ids = append(ids, QuestionIDFor(instanceID, c.ID, SideReverse))
		}
	}
	if len(ids) == 0 {
		return nil
	}
	_, err := pool.Exec(ctx, `
DELETE FROM course.srs_review_events
WHERE user_id = $1 AND question_id = ANY($2::uuid[])
`, userID, ids)
	if err != nil {
		return fmt.Errorf("clear srs events: %w", err)
	}
	_, err = pool.Exec(ctx, `
DELETE FROM course.srs_item_states
WHERE user_id = $1 AND question_id = ANY($2::uuid[])
`, userID, ids)
	if err != nil {
		return fmt.Errorf("clear srs states: %w", err)
	}
	return nil
}

// Grade mapping lives in service/srs; adapter only submits reviews.
