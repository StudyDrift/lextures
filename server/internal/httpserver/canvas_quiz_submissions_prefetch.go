package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/google/uuid"
)

func canvasQuizIDsFromMap(m map[int64]uuid.UUID) []int64 {
	if len(m) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	return ids
}

// canvasBackfillQuizSubmissionsByUser fetches per-learner quiz submissions when the course-wide
// list endpoint returns none (common when Canvas omits user_id without include[]=user).
func canvasBackfillQuizSubmissionsByUser(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	canvasQuizToItem map[int64]uuid.UUID,
	canvasUserToLocal map[int64]uuid.UUID,
	quizSubsByQuiz map[int64][]map[string]any,
) error {
	if len(canvasQuizToItem) == 0 || len(canvasUserToLocal) == 0 {
		return nil
	}
	canvasUserIDs := make([]int64, 0, len(canvasUserToLocal))
	for canvasUID := range canvasUserToLocal {
		if canvasUID > 0 {
			canvasUserIDs = append(canvasUserIDs, canvasUID)
		}
	}
	for canvasQID := range canvasQuizToItem {
		if len(quizSubsByQuiz[canvasQID]) > 0 {
			continue
		}
		backfill := make([]map[string]any, 0)
		for _, canvasUID := range canvasUserIDs {
			q := url.Values{}
			q.Set("user_id", strconv.FormatInt(canvasUID, 10))
			subs, err := canvasGetQuizSubmissionsPaginated(ctx, client, canvasBase, accessToken, canvasCourseID, canvasQID, q)
			if err != nil {
				return fmt.Errorf("Canvas quiz %d submission for user %d: %w", canvasQID, canvasUID, err)
			}
			for _, raw := range subs {
				if canvasQuizSubmissionImportable(raw) {
					backfill = append(backfill, raw)
				}
			}
		}
		if len(backfill) > 0 {
			quizSubsByQuiz[canvasQID] = backfill
		}
	}
	return nil
}

func canvasFetchQuizSubmissionsParallel(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	quizIDs []int64,
) (map[int64][]map[string]any, error) {
	out := make(map[int64][]map[string]any, len(quizIDs))
	if len(quizIDs) == 0 {
		return out, nil
	}
	var mu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	g, gctx := canvasImportParallelGroup(ctx, len(quizIDs))
	for _, quizID := range quizIDs {
		quizID := quizID
		g.Go(func() error {
			subs, err := canvasGetQuizSubmissionsPaginated(gctx, client, canvasBase, accessToken, canvasCourseID, quizID, nil)
			if err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("Canvas quiz %d submissions: %w", quizID, err) })
				return err
			}
			mu.Lock()
			out[quizID] = subs
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, err
	}
	return out, nil
}

func canvasFetchAssignmentSubmissionsParallel(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	canvasAssignToItem map[int64]uuid.UUID,
) (map[int64][]map[string]any, error) {
	if len(canvasAssignToItem) == 0 {
		return map[int64][]map[string]any{}, nil
	}
	assignmentSubsQuery := url.Values{}
	assignmentSubsQuery.Add("include[]", "submission_history")
	assignmentSubsQuery.Add("include[]", "submission_comments")
	assignmentSubsQuery.Add("include[]", "submission_html_comments")
	assignmentSubsQuery.Add("include[]", "rubric_assessment")

	out := make(map[int64][]map[string]any, len(canvasAssignToItem))
	var mu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	g, gctx := canvasImportParallelGroup(ctx, len(canvasAssignToItem))
	for canvasAID := range canvasAssignToItem {
		canvasAID := canvasAID
		g.Go(func() error {
			subs, err := canvasGetArrayPaginated(gctx, client, canvasBase, accessToken,
				fmt.Sprintf("courses/%d/assignments/%d/submissions", canvasCourseID, canvasAID), assignmentSubsQuery)
			if err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("Canvas assignment %d submissions: %w", canvasAID, err) })
				return err
			}
			mu.Lock()
			out[canvasAID] = subs
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, err
	}
	return out, nil
}
