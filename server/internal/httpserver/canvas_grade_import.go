package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// normalizedLexturesEmailGuessFromCanvasUserMap picks a plausible email-ish login from a Canvas User
// (course roster payloads often include primary email here; enrollment-embedded mini users omit it often).
func normalizedLexturesEmailGuessFromCanvasUserMap(u map[string]any) string {
	if u == nil {
		return ""
	}
	em := strings.ToLower(strings.TrimSpace(strAt(u, "email", "")))
	if strings.Contains(em, "@") {
		return em
	}
	lid := strings.ToLower(strings.TrimSpace(strAt(u, "login_id", "")))
	if strings.Contains(lid, "@") {
		return lid
	}
	sis := strings.ToLower(strings.TrimSpace(strAt(u, "sis_user_id", "")))
	if strings.Contains(sis, "@") {
		return sis
	}
	return ""
}

func lexturesUUIDForMatchedCanvasEmail(ctx context.Context, pool *pgxpool.Pool, emailGuess string) (uuid.UUID, bool) {
	if pool == nil || emailGuess == "" || !strings.Contains(emailGuess, "@") {
		return uuid.Nil, false
	}
	usr, ue := user.FindByEmailCI(ctx, pool, emailGuess)
	if ue != nil || usr == nil {
		return uuid.Nil, false
	}
	userID, pe := uuid.Parse(usr.ID)
	if pe != nil {
		return uuid.Nil, false
	}
	return userID, true
}

func canvasListCourseUsersByEnrollmentType(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	enrollmentType string,
) ([]map[string]any, error) {
	q := url.Values{}
	q.Add("enrollment_type[]", enrollmentType)
	path := fmt.Sprintf("courses/%d/users", canvasCourseID)
	return canvasGetArrayPaginated(ctx, client, canvasBase, accessToken, path, q)
}

// canvasListCourseStudentUsersForGradeMatch loads student roster emails for grading.
// Prefer enrollment_type=student; Canvas sites with bespoke student roles may return an empty list, so we fall back to enrollment_role=StudentEnrollment.
func canvasListCourseStudentUsersForGradeMatch(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
) ([]map[string]any, error) {
	path := fmt.Sprintf("courses/%d/users", canvasCourseID)
	if roster, err := canvasListCourseUsersByEnrollmentType(ctx, client, canvasBase, accessToken, canvasCourseID, "student"); err != nil {
		return nil, err
	} else if len(roster) > 0 {
		return roster, nil
	}
	q := url.Values{}
	q.Set("enrollment_role", "StudentEnrollment")
	return canvasGetArrayPaginated(ctx, client, canvasBase, accessToken, path, q)
}

func canvasMapCanvasUserToLextures(
	ctx context.Context,
	pool *pgxpool.Pool,
	out map[int64]uuid.UUID,
	canvasUID int64,
	email string,
) {
	if canvasUID <= 0 || out == nil {
		return
	}
	if _, dup := out[canvasUID]; dup {
		return
	}
	em := strings.TrimSpace(email)
	if em == "" {
		return
	}
	if userID, ok := lexturesUUIDForMatchedCanvasEmail(ctx, pool, em); ok {
		out[canvasUID] = userID
	}
}

// buildCanvasUserIDToLexturesUserID maps Canvas user ids → Lextures user ids using roster/enrollment emails.
func buildCanvasUserIDToLexturesUserID(
	ctx context.Context,
	pool *pgxpool.Pool,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	enrollmentRows []map[string]any,
	rosterEmailByCanvasUID map[int64]string,
) map[int64]uuid.UUID {
	out := make(map[int64]uuid.UUID)
	if pool == nil {
		return out
	}
	for canvasUID, email := range rosterEmailByCanvasUID {
		canvasMapCanvasUserToLextures(ctx, pool, out, canvasUID, email)
	}
	for _, e := range enrollmentRows {
		u := objAt(e, "user")
		canvasUID := int64At(u, "id")
		email := rosterEmailByCanvasUID[canvasUID]
		if email == "" {
			email = normalizedLexturesEmailGuessFromCanvasUserMap(u)
		}
		canvasMapCanvasUserToLextures(ctx, pool, out, canvasUID, email)
	}
	if client != nil && len(out) == 0 {
		roster, err := canvasListCourseStudentUsersForGradeMatch(ctx, client, canvasBase, accessToken, canvasCourseID)
		if err == nil {
			for _, u := range roster {
				canvasUID := int64At(u, "id")
				email := rosterEmailByCanvasUID[canvasUID]
				if email == "" {
					email = normalizedLexturesEmailGuessFromCanvasUserMap(u)
				}
				canvasMapCanvasUserToLextures(ctx, pool, out, canvasUID, email)
			}
		}
	}
	return out
}

func optionalPointsWorthFromCanvas(m map[string]any, key string) *int {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	var f float64
	switch x := v.(type) {
	case float64:
		f = x
	case int64:
		f = float64(x)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return nil
		}
		f = parsed
	default:
		return nil
	}
	if math.IsNaN(f) || math.IsInf(f, 0) || f < 0 {
		return nil
	}
	i := int(math.Round(f))
	if i > 1000000 {
		i = 1000000
	}
	return &i
}

// coerceCanvasJSONNumber parses Canvas JSON numeric fields (normally float64 from encoding/json).
func coerceCanvasJSONNumber(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return 0, false
		}
		return x, true
	case int64:
		return float64(x), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, false
		}
		return parsed, true
	case json.Number:
		parsed, err := x.Float64()
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

// canvasSubmissionIsGradedForImport reports whether Canvas considers this submission graded
// (as opposed to submitted and awaiting grading). Resubmissions keep workflow_state=submitted
// even when submission_history still carries the prior attempt's score.
func canvasSubmissionIsGradedForImport(sub map[string]any) bool {
	if sub == nil {
		return false
	}
	if boolAt(sub, "excused", false) {
		return true
	}
	state := strings.ToLower(strings.TrimSpace(strAt(sub, "workflow_state", "")))
	switch state {
	case "graded":
		return true
	case "submitted", "unsubmitted", "deleted":
		return false
	case "pending_review":
		// Moderated grading: instructor entered a score awaiting release.
		_, _, hasScore := canvasSubmissionTopLevelNumericScore(sub)
		return hasScore
	case "":
		// Some Canvas list payloads omit workflow_state; infer from top-level score only.
		_, _, hasScore := canvasSubmissionTopLevelNumericScore(sub)
		return hasScore
	default:
		return false
	}
}

func canvasSubmissionTopLevelNumericScore(sub map[string]any) (excused bool, score float64, hasScore bool) {
	if sub == nil {
		return false, 0, false
	}
	if excused = boolAt(sub, "excused", false); excused {
		return true, 0, false
	}
	if sc, ok := coerceCanvasJSONNumber(sub["score"]); ok {
		return false, sc, true
	}
	if sc, ok := coerceCanvasJSONNumber(sub["entered_score"]); ok {
		return false, sc, true
	}
	if assessment := objAt(sub, "rubric_assessment"); assessment != nil {
		if sc, ok := coerceCanvasJSONNumber(assessment["score"]); ok {
			return false, sc, true
		}
	}
	return false, 0, false
}

func submissionScoreAndExcused(sub map[string]any) (excused bool, score float64, hasScore bool) {
	if sub == nil {
		return false, 0, false
	}
	if exc, sc, ok := canvasSubmissionTopLevelNumericScore(sub); ok || exc {
		return exc, sc, ok
	}
	// Only fall back to submission_history when Canvas marks the current attempt graded.
	if !canvasSubmissionIsGradedForImport(sub) {
		return false, 0, false
	}
	if hist, ok := sub["submission_history"].([]any); ok && len(hist) > 0 {
		for i := len(hist) - 1; i >= 0; i-- {
			hm, ok := hist[i].(map[string]any)
			if !ok || hm == nil {
				continue
			}
			if boolAt(hm, "excused", false) {
				continue
			}
			if sc, ok := coerceCanvasJSONNumber(hm["score"]); ok {
				return false, sc, true
			}
		}
	}
	return false, 0, false
}

func deleteCourseGradeFromCanvasImport(
	ctx context.Context,
	tx pgx.Tx,
	studentID, moduleItemID uuid.UUID,
) error {
	_, err := tx.Exec(ctx, `
DELETE FROM course.course_grades
WHERE student_user_id = $1 AND module_item_id = $2
`, studentID, moduleItemID)
	return err
}

func upsertCourseGradeFromCanvas(
	ctx context.Context,
	tx pgx.Tx,
	courseID, studentID, moduleItemID uuid.UUID,
	pointsEarned float64,
	excused bool,
	instructorComment *string,
	instructorCommentsJSON []byte,
	rubricJSON []byte,
	markPosted bool,
) error {
	if pointsEarned < 0 {
		pointsEarned = 0
	}
	if pointsEarned > 1e9 {
		pointsEarned = 1e9
	}
	var rubric any
	if len(rubricJSON) > 0 {
		rubric = rubricJSON
	}
	var comments any
	if len(instructorCommentsJSON) > 0 {
		comments = instructorCommentsJSON
	}
	query := `
INSERT INTO course.course_grades (
	course_id, student_user_id, module_item_id, points_earned, rubric_scores_json, instructor_comment, instructor_comments_json, updated_at, posted_at, excused
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NULL, $8)
ON CONFLICT (student_user_id, module_item_id) DO UPDATE SET
	course_id = EXCLUDED.course_id,
	points_earned = EXCLUDED.points_earned,
	rubric_scores_json = EXCLUDED.rubric_scores_json,
	instructor_comment = EXCLUDED.instructor_comment,
	instructor_comments_json = EXCLUDED.instructor_comments_json,
	updated_at = NOW(),
	posted_at = NULL,
	excused = EXCLUDED.excused`
	if markPosted {
		query = `
INSERT INTO course.course_grades (
	course_id, student_user_id, module_item_id, points_earned, rubric_scores_json, instructor_comment, instructor_comments_json, updated_at, posted_at, excused
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), $8)
ON CONFLICT (student_user_id, module_item_id) DO UPDATE SET
	course_id = EXCLUDED.course_id,
	points_earned = EXCLUDED.points_earned,
	rubric_scores_json = EXCLUDED.rubric_scores_json,
	instructor_comment = EXCLUDED.instructor_comment,
	instructor_comments_json = EXCLUDED.instructor_comments_json,
	updated_at = NOW(),
	posted_at = COALESCE(course.course_grades.posted_at, NOW()),
	excused = EXCLUDED.excused`
	}
	_, err := tx.Exec(ctx, query, courseID, studentID, moduleItemID, pointsEarned, rubric, instructorComment, comments, excused)
	return err
}

func canvasSyncedGradeHasImportableFeedback(synced canvasSyncedGrade, graded bool) bool {
	if graded && (synced.hasNumericScore || synced.excused || len(synced.rubricJSON) > 0) {
		return true
	}
	return synced.comment != nil || len(synced.commentsJSON) > 0
}

func canvasImportAssignmentGrades(
	ctx context.Context,
	tx pgx.Tx,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	courseID uuid.UUID,
	canvasAssignToItem map[int64]uuid.UUID,
	canvasUserToLocal map[int64]uuid.UUID,
	submissionDeps *canvasAssignmentSubmissionImportDeps,
	importGrades bool,
) error {
	// #region agent log
	assignCanvasIDs := int64(len(canvasAssignToItem))
	var totalSubs int64
	var skipBadCanvasUID int64
	var skipNoLocalUser int64
	var skipNoScore int64
	var upserts int64
	var submissionRowsUpserted int64
	var submissionContentStored int64
	// #endregion agent log

	subsByAssignment, err := canvasFetchAssignmentSubmissionsParallel(ctx, client, canvasBase, accessToken, canvasCourseID, canvasAssignToItem)
	if err != nil {
		return err
	}
	for canvasAID, itemID := range canvasAssignToItem {
		subs := subsByAssignment[canvasAID]
		// #region agent log
		totalSubs += int64(len(subs))
		// #endregion agent log
		prefetchedAttachments := canvasPrefetchSubmissionAttachmentsParallel(ctx, client, accessToken, subs)
		for _, raw := range subs {
			canvasUserID := int64At(raw, "user_id")
			if canvasUserID <= 0 {
				// #region agent log
				skipBadCanvasUID++
				// #endregion agent log
				continue
			}
			studentID, ok := canvasUserToLocal[canvasUserID]
			if !ok {
				// #region agent log
				skipNoLocalUser++
				// #endregion agent log
				continue
			}
			if importGrades {
				graded := canvasSubmissionIsGradedForImport(raw)
				synced, parseErr := canvasGradeFromSubmissionPayload(raw, nil, canvasUserToLocal)
				if parseErr != nil {
					return fmt.Errorf("parse grade for assignment canvas id %d: %w", canvasAID, parseErr)
				}
				if !canvasSyncedGradeHasImportableFeedback(synced, graded) {
					if err := deleteCourseGradeFromCanvasImport(ctx, tx, studentID, itemID); err != nil {
						return fmt.Errorf("clear stale grade for assignment canvas id %d: %w", canvasAID, err)
					}
					// #region agent log
					skipNoScore++
					// #endregion agent log
				} else {
					hasGrade := graded && (synced.hasNumericScore || synced.excused || len(synced.rubricJSON) > 0)
					pts := 0.0
					if graded && synced.hasNumericScore {
						pts = synced.points
					}
					excused := graded && synced.excused
					rubricJSON := synced.rubricJSON
					if !graded {
						rubricJSON = nil
					}
					if err := upsertCourseGradeFromCanvas(
						ctx, tx, courseID, studentID, itemID, pts, excused,
						synced.comment, synced.commentsJSON, rubricJSON, hasGrade,
					); err != nil {
						return fmt.Errorf("save grade for assignment canvas id %d: %w", canvasAID, err)
					}
					// #region agent log
					upserts++
					// #endregion agent log
				}
			}
			if submissionDeps != nil {
				hadContent := canvasSubmissionHasContent(raw)
				prefetched := prefetchedAttachments[canvasUserID]
				if err := canvasImportOneAssignmentSubmission(
					ctx, tx, client, accessToken, *submissionDeps, courseID, itemID, studentID, raw, prefetched,
				); err != nil {
					log.Printf("canvas-import: skip submission for assignment canvas id %d user %s: %v", canvasAID, studentID, err)
					continue
				}
				if canvasAssignmentSubmissionImportable(raw) {
					submissionRowsUpserted++
					if hadContent {
						submissionContentStored++
					}
				}
			}
		}
	}

	// #region agent log
	canvasAgentDebugLog("canvas-import", "H2-H4", "canvas_grade_import.go:canvasImportAssignmentGrades", "assignment submission import counters", map[string]any{
		"assignmentCanvasIDs":       assignCanvasIDs,
		"totalSubmissionRows":     totalSubs,
		"skipBadCanvasUserID":     skipBadCanvasUID,
		"skipNoMappedUser":        skipNoLocalUser,
		"skipNoScore":             skipNoScore,
		"gradesUpserted":          upserts,
		"submissionRowsUpserted":  submissionRowsUpserted,
		"submissionContentStored": submissionContentStored,
	})
	// #endregion agent log
	return nil
}

// Quiz submission list responses wrap rows in {"quiz_submissions":[...]} (not a bare JSON array).
func canvasUnpackQuizSubmissionResponse(v any) []map[string]any {
	switch t := v.(type) {
	case map[string]any:
		raw, ok := t["quiz_submissions"].([]any)
		if !ok || len(raw) == 0 {
			return nil
		}
		out := make([]map[string]any, 0, len(raw))
		for _, it := range raw {
			if m, ok := it.(map[string]any); ok && m != nil {
				out = append(out, m)
			}
		}
		return out
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]any); ok && m != nil {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func canvasGetQuizSubmissionsPaginated(
	ctx context.Context,
	client *http.Client,
	base, token string,
	canvasCourseID, quizID int64,
	q url.Values,
) ([]map[string]any, error) {
	out := make([]map[string]any, 0)
	for page := 1; ; page++ {
		qp := cloneQuery(q)
		qp.Add("include[]", "user")
		qp.Add("include[]", "submission")
		qp.Set("per_page", "100")
		qp.Set("page", strconv.Itoa(page))
		v, err := canvasGetJSON(ctx, client, base, token,
			fmt.Sprintf("courses/%d/quizzes/%d/submissions", canvasCourseID, quizID), qp)
		if err != nil {
			return nil, err
		}
		chunk := canvasUnpackQuizSubmissionResponse(v)
		if len(chunk) == 0 {
			break
		}
		out = append(out, chunk...)
		if len(chunk) < 100 {
			break
		}
	}
	return out, nil
}

// quizSubmissionImportRank assigns a sortable priority when Canvas returns multiple attempts per learner.
func quizSubmissionImportRank(m map[string]any) int64 {
	if m == nil {
		return -1
	}
	state := strings.ToLower(strings.TrimSpace(strAt(m, "workflow_state", "")))
	att := int64At(m, "attempt")
	switch state {
	case "complete":
		return 1_000_000 + att
	case "pending_review":
		return 500_000 + att
	default:
		return att
	}
}

// canvasQuizSubmissionIsGradedForImport reports whether Canvas considers a quiz submission fully graded.
// pending_review means the learner submitted but manual question grading is still outstanding.
func canvasQuizSubmissionIsGradedForImport(raw map[string]any) bool {
	if raw == nil {
		return false
	}
	if boolAt(raw, "excused", false) {
		return true
	}
	state := strings.ToLower(strings.TrimSpace(strAt(raw, "workflow_state", "")))
	switch state {
	case "complete", "graded":
		return true
	case "pending_review", "submitted", "unsubmitted", "deleted":
		return false
	default:
		return false
	}
}

func pickPreferredQuizSubmissionForUser(existing, candidate map[string]any) map[string]any {
	if existing == nil {
		return candidate
	}
	if quizSubmissionImportRank(candidate) >= quizSubmissionImportRank(existing) {
		return candidate
	}
	return existing
}

func canvasImportQuizGrades(
	ctx context.Context,
	tx pgx.Tx,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	courseID uuid.UUID,
	canvasQuizToItem map[int64]uuid.UUID,
	canvasUserToLocal map[int64]uuid.UUID,
	quizSubsByQuiz map[int64][]map[string]any,
) error {
	// #region agent log
	quizCanvasIDs := int64(len(canvasQuizToItem))
	var totalQuizSubs int64
	var quizSkipNotInLocalMap int64
	var quizMergedLearners int64
	var quizSkipNoScore int64
	var quizUpserts int64
	// #endregion agent log

	for canvasQID, itemID := range canvasQuizToItem {
		subs := quizSubsByQuiz[canvasQID]
		// #region agent log
		totalQuizSubs += int64(len(subs))
		// #endregion agent log
		byCanvasUser := make(map[int64]map[string]any)
		for _, raw := range subs {
			canvasUserID := canvasCanvasUserIDFromMap(raw)
			if canvasUserID <= 0 {
				continue
			}
			if _, wants := canvasUserToLocal[canvasUserID]; !wants {
				// #region agent log
				quizSkipNotInLocalMap++
				// #endregion agent log
				continue
			}
			prev := byCanvasUser[canvasUserID]
			byCanvasUser[canvasUserID] = pickPreferredQuizSubmissionForUser(prev, raw)
		}
		// #region agent log
		quizMergedLearners += int64(len(byCanvasUser))
		// #endregion agent log
		for canvasUserID, raw := range byCanvasUser {
			studentID, ok := canvasUserToLocal[canvasUserID]
			if !ok {
				continue
			}
			if !canvasQuizSubmissionIsGradedForImport(raw) {
				if err := deleteCourseGradeFromCanvasImport(ctx, tx, studentID, itemID); err != nil {
					return fmt.Errorf("clear stale grade for quiz canvas id %d: %w", canvasQID, err)
				}
				// #region agent log
				quizSkipNoScore++
				// #endregion agent log
				continue
			}
			exc := boolAt(raw, "excused", false)
			score := 0.0
			hasScore := false
			if !exc {
				if v, ok := raw["kept_score"]; ok {
					if n, ok2 := coerceCanvasJSONNumber(v); ok2 {
						score, hasScore = n, true
					}
				}
				if !hasScore {
					if v, ok := raw["score"]; ok {
						if n, ok2 := coerceCanvasJSONNumber(v); ok2 {
							score, hasScore = n, true
						}
					}
				}
				if !hasScore {
					_, sc, okSc := submissionScoreAndExcused(raw)
					if okSc {
						score, hasScore = sc, true
					}
				}
			}
			if !exc && !hasScore {
				if err := deleteCourseGradeFromCanvasImport(ctx, tx, studentID, itemID); err != nil {
					return fmt.Errorf("clear stale grade for quiz canvas id %d: %w", canvasQID, err)
				}
				// #region agent log
				quizSkipNoScore++
				// #endregion agent log
				continue
			}
			pts := 0.0
			if hasScore {
				pts = score
			}
			if err := upsertCourseGradeFromCanvas(ctx, tx, courseID, studentID, itemID, pts, exc, nil, nil, nil, true); err != nil {
				return fmt.Errorf("save grade for quiz canvas id %d: %w", canvasQID, err)
			}
			// #region agent log
			quizUpserts++
			// #endregion agent log
		}
	}

	// #region agent log
	canvasAgentDebugLog("canvas-import", "H2-H4", "canvas_grade_import.go:canvasImportQuizGrades", "quiz submission import counters", map[string]any{
		"quizCanvasIDs":          quizCanvasIDs,
		"totalQuizSubmissionRaw": totalQuizSubs,
		"skippedRawNoLocalMatch": quizSkipNotInLocalMap,
		"mergedLearners":         quizMergedLearners,
		"skippedNoScore":         quizSkipNoScore,
		"gradesUpserted":         quizUpserts,
	})
	// #endregion agent log
	return nil
}

func canvasImportAllCanvasGrades(
	ctx context.Context,
	tx pgx.Tx,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	courseID uuid.UUID,
	canvasAssignToItem map[int64]uuid.UUID,
	canvasQuizToItem map[int64]uuid.UUID,
	canvasQuizToQuestions map[int64][]coursemodulequiz.QuizQuestion,
	canvasQuizToAssignmentID map[int64]int64,
	canvasUserToLocal map[int64]uuid.UUID,
	submissionDeps *canvasAssignmentSubmissionImportDeps,
) error {
	// #region agent log
	canvasAgentDebugLog("canvas-import", "H2,H5", "canvas_grade_import.go:canvasImportAllCanvasGrades", "entering aggregated grade import", map[string]any{
		"mappedCanvasUsers": len(canvasUserToLocal),
		"assignmentIDs":     len(canvasAssignToItem),
		"quizIDs":           len(canvasQuizToItem),
	})
	// #endregion agent log
	if len(canvasUserToLocal) == 0 {
		// #region agent log
		canvasAgentDebugLog("canvas-import", "H5", "canvas_grade_import.go:canvasImportAllCanvasGrades", "early exit — no Canvas users mapped to Lex accounts", map[string]any{})
		// #endregion agent log
		return nil
	}
	quizSubsByQuiz, err := canvasFetchQuizSubmissionsParallel(ctx, client, canvasBase, accessToken, canvasCourseID, canvasQuizIDsFromMap(canvasQuizToItem))
	if err != nil {
		return err
	}
	if err := canvasBackfillQuizSubmissionsByUser(ctx, client, canvasBase, accessToken, canvasCourseID, canvasQuizToItem, canvasUserToLocal, quizSubsByQuiz); err != nil {
		return err
	}
	if err := canvasImportAssignmentGrades(ctx, tx, client, canvasBase, accessToken, canvasCourseID, courseID, canvasAssignToItem, canvasUserToLocal, submissionDeps, true); err != nil {
		return err
	}
	if err := canvasImportQuizGrades(ctx, tx, client, canvasBase, accessToken, canvasCourseID, courseID, canvasQuizToItem, canvasUserToLocal, quizSubsByQuiz); err != nil {
		return err
	}
	if err := canvasImportQuizAttempts(ctx, tx, client, canvasBase, accessToken, canvasCourseID, courseID, canvasQuizToItem, canvasQuizToQuestions, canvasQuizToAssignmentID, canvasUserToLocal, quizSubsByQuiz, submissionDeps); err != nil {
		return err
	}
	return nil
}
