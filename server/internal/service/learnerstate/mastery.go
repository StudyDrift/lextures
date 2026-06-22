// Mastery updates from graded quiz responses (port of server/src/services/learner_state.rs).
package learnerstate

import (
	"context"
	"encoding/json"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/concepts"
	"github.com/lextures/lextures/server/internal/repos/hints"
	"github.com/lextures/lextures/server/internal/repos/learnermodel"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
)

// ConceptTouch is one concept update derived from a graded question.
type ConceptTouch struct {
	ConceptID          uuid.UUID
	Score              float64
	QuestionIndex      int32
	EMAAlphaMultiplier float64
}

// ApplyMasteryParams configures mastery application from saved quiz responses.
type ApplyMasteryParams struct {
	CourseID                      uuid.UUID
	UserID                        uuid.UUID
	AttemptID                     uuid.UUID
	Questions                     []coursemodulequiz.QuizQuestion
	Responses                     []quizattempts.ResponseRow
	HintScaffoldingEnabled        bool
	MisconceptionDetectionEnabled bool
	LearnerModelEnabled           bool
	EMAAlpha                      float64
}

// MasteryScaleForHintUses scales evidence weight when hints were used (Rust hint_service).
func MasteryScaleForHintUses(hintUses int64) float64 {
	if hintUses <= 0 {
		return 1.0
	}
	return math.Min(1, math.Max(0.35, 1.0/(1.0+0.12*float64(hintUses))))
}

// CollectConceptTouchesFromQuestion appends concept touches for one graded question.
func CollectConceptTouchesFromQuestion(
	q *coursemodulequiz.QuizQuestion,
	qIndex int32,
	pts, maxPts float64,
	extraConceptIDs []uuid.UUID,
	masteryScale, emaAlphaMultiplier float64,
	out *[]ConceptTouch,
) {
	denom := maxPts
	if denom <= 0 {
		denom = 1
	}
	score := math.Min(1, math.Max(0, pts/denom)) * math.Min(1, math.Max(0, masteryScale))
	mult := math.Min(2, math.Max(0.01, emaAlphaMultiplier))
	seen := make(map[[2]any]struct{})

	add := func(cid uuid.UUID) {
		key := [2]any{cid, qIndex}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		*out = append(*out, ConceptTouch{
			ConceptID:          cid,
			Score:              score,
			QuestionIndex:      qIndex,
			EMAAlphaMultiplier: mult,
		})
	}

	for _, sid := range q.ConceptIDs {
		cid, err := uuid.Parse(strings.TrimSpace(sid))
		if err != nil {
			continue
		}
		add(cid)
	}
	for _, cid := range extraConceptIDs {
		add(cid)
	}
}

// ApplyQuizGradesMastery applies all concept touches inside the grading transaction.
func ApplyQuizGradesMastery(
	ctx context.Context,
	tx pgx.Tx,
	courseID, userID, attemptID uuid.UUID,
	touches []ConceptTouch,
	learnerModelEnabled bool,
	emaAlpha float64,
) error {
	if !learnerModelEnabled {
		return nil
	}
	if emaAlpha <= 0 || emaAlpha > 1 {
		emaAlpha = 0.3
	}
	for _, touch := range touches {
		input := learnermodel.MasteryUpdateInput{
			UserID:             userID,
			AttemptID:          attemptID,
			CourseID:           courseID,
			ConceptID:          touch.ConceptID,
			Score:              touch.Score,
			QuestionIndex:      touch.QuestionIndex,
			EMAAlpha:           emaAlpha,
			EMAAlphaMultiplier: touch.EMAAlphaMultiplier,
		}
		if err := learnermodel.ApplyMasteryUpdateInTx(ctx, tx, input); err != nil {
			return err
		}
	}
	return nil
}

// ApplyMasteryFromSavedResponses updates mastery from stored quiz responses before finalize
// (Rust apply_mastery_from_saved_responses).
func ApplyMasteryFromSavedResponses(
	ctx context.Context,
	pool *pgxpool.Pool,
	tx pgx.Tx,
	p ApplyMasteryParams,
) error {
	if !p.LearnerModelEnabled {
		return nil
	}
	emaAlpha := p.EMAAlpha
	if emaAlpha <= 0 || emaAlpha > 1 {
		emaAlpha = 0.3
	}

	var hintCounts map[string]int64
	if p.HintScaffoldingEnabled {
		var err error
		hintCounts, err = hints.HintUseCountsForAttempt(ctx, pool, p.AttemptID)
		if err != nil {
			return err
		}
	} else {
		hintCounts = map[string]int64{}
	}

	byID := make(map[string]*coursemodulequiz.QuizQuestion, len(p.Questions))
	var qids []uuid.UUID
	for i := range p.Questions {
		q := &p.Questions[i]
		byID[q.ID] = q
		if qid, err := uuid.Parse(strings.TrimSpace(q.ID)); err == nil {
			qids = append(qids, qid)
		}
	}
	tagMap, err := concepts.ConceptIDsForQuestionIDs(ctx, pool, qids)
	if err != nil {
		return err
	}

	var touches []ConceptTouch
	for _, r := range p.Responses {
		qid := strings.TrimSpace(r.QuestionID)
		if qid == "" {
			continue
		}
		q, ok := byID[qid]
		if !ok {
			continue
		}
		max := r.MaxPoints
		pts := r.PointsAwarded
		var extra []uuid.UUID
		if quuid, err := uuid.Parse(qid); err == nil {
			extra = tagMap[quuid]
		}
		hintN := hintCounts[qid]
		ms := MasteryScaleForHintUses(hintN)
		emaMult := 1.0
		if p.MisconceptionDetectionEnabled && r.IsCorrect != nil && !*r.IsCorrect {
			emaMult = misconceptionEMAMultiplierForResponse(r.ResponseJSON)
		}
		CollectConceptTouchesFromQuestion(q, r.QuestionIndex, pts, max, extra, ms, emaMult, &touches)
	}
	return ApplyQuizGradesMastery(ctx, tx, p.CourseID, p.UserID, p.AttemptID, touches, true, emaAlpha)
}

// misconceptionEMAMultiplierForResponse returns a stronger EMA weight when a known misconception
// was selected. Full misconception recording is not yet ported in Go; this stays at 1.0 until then.
func misconceptionEMAMultiplierForResponse(_ json.RawMessage) float64 {
	return 1.0
}