package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/canvassubmissionsyncqueue"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
	"github.com/lextures/lextures/server/internal/models/gradecomment"
	"github.com/lextures/lextures/server/internal/repos/canvasimportjobs"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// handleGetCourseCanvasLink is GET /api/v1/courses/{course_code}/canvas-link.
func (d Deps) handleGetCourseCanvasLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		job, err := canvasimportjobs.LatestLinkedForCourse(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Canvas link.")
			return
		}
		out := map[string]any{"linked": false, "gradeSyncEnabled": false}
		if job != nil {
			out["linked"] = true
			out["canvasBaseUrl"] = job.CanvasBaseURL
			out["canvasCourseId"] = job.CanvasCourseID
		}
		courseRow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if courseRow != nil {
			out["gradeSyncEnabled"] = courseRow.CanvasGradeSyncEnabled
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handlePatchCourseCanvasLink is PATCH /api/v1/courses/{course_code}/canvas-link.
func (d Deps) handlePatchCourseCanvasLink() http.HandlerFunc {
	type body struct {
		GradeSyncEnabled *bool `json:"gradeSyncEnabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		has, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to update Canvas settings.")
			return
		}
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if b.GradeSyncEnabled == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "gradeSyncEnabled is required.")
			return
		}
		link, err := canvasimportjobs.LatestLinkedForCourse(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Canvas link.")
			return
		}
		if link == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This course was not imported from Canvas.")
			return
		}
		out, err := course.SetCanvasGradeSyncEnabled(r.Context(), d.Pool, courseCode, *b.GradeSyncEnabled)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update Canvas grade sync.")
			return
		}
		if out == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		resp := map[string]any{
			"linked":           true,
			"canvasBaseUrl":    link.CanvasBaseURL,
			"canvasCourseId":   link.CanvasCourseID,
			"gradeSyncEnabled": out.CanvasGradeSyncEnabled,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// handlePostSubmissionSyncCanvas is POST /api/v1/courses/{course_code}/assignments/{item_id}/submissions/{submission_id}/sync-canvas.
// Queues a background job that pushes the Lextures grade for this submission to the linked Canvas course.
func (d Deps) handlePostSubmissionSyncCanvas() http.HandlerFunc {
	type body struct {
		CanvasBaseURL     string             `json:"canvasBaseUrl"`
		AccessToken       string             `json:"accessToken"`
		PointsEarned      *float64           `json:"pointsEarned"`
		RubricScores      map[string]float64 `json:"rubricScores"`
		InstructorComment *string            `json:"instructorComment"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.CanvasSubmissionSyncQueue == nil || d.CanvasSubmissionSyncJobs == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		has, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to sync grades.")
			return
		}
		itemID, submissionID, _, _, _, ok := d.loadSubmissionGradeContext(w, r, courseCode)
		if !ok {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var b body
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &b); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}
		link, err := canvasimportjobs.LatestLinkedForCourse(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load Canvas link.")
			return
		}
		if link == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This course was not imported from Canvas.")
			return
		}
		courseRow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if courseRow == nil || !courseRow.CanvasGradeSyncEnabled {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Canvas grade sync is not enabled for this course.")
			return
		}
		canvasBaseRaw := strings.TrimSpace(b.CanvasBaseURL)
		if canvasBaseRaw == "" {
			canvasBaseRaw = link.CanvasBaseURL
		}
		canvasBase, err := normalizeCanvasBaseURL(canvasBaseRaw, d.effectiveConfig().CanvasAllowedHostSuffixes)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		token := strings.TrimSpace(b.AccessToken)
		if token == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Canvas access token is required.")
			return
		}

		jobID := d.CanvasSubmissionSyncJobs.Create(viewer)
		msg := canvassubmissionsyncqueue.QueueMessage{
			JobID:             jobID,
			UserID:            viewer,
			CourseCode:        courseCode,
			ItemID:            itemID,
			SubmissionID:      submissionID,
			CanvasBaseURL:     canvasBase,
			AccessToken:       token,
			PointsEarned:      b.PointsEarned,
			RubricScores:      b.RubricScores,
			InstructorComment: b.InstructorComment,
		}
		if err := d.CanvasSubmissionSyncQueue.Publish(r.Context(), msg); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enqueue Canvas sync.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"jobId":   jobID.String(),
			"message": "Canvas sync queued. You can keep grading — we will notify you when it finishes.",
		})
	}
}

type submissionSyncCanvasInput struct {
	CourseCode        string
	ItemID            uuid.UUID
	SubmissionID      uuid.UUID
	CanvasBaseURL     string
	AccessToken       string
	PointsEarned      *float64
	RubricScores      map[string]float64
	InstructorComment *string
}

func (d Deps) loadSubmissionSyncContext(
	ctx context.Context,
	courseCode string,
	itemID, submissionID uuid.UUID,
) (*uuid.UUID, *coursemoduleassignments.CourseItemAssignmentRow, *moduleassignmentsubmissions.SubmissionRow, error) {
	cid, assignRow, err := d.loadAssignmentForSubmissionsByIDs(ctx, courseCode, itemID)
	if err != nil {
		return nil, nil, nil, err
	}
	if cid == nil || assignRow == nil {
		return nil, nil, nil, errors.New("Assignment not found.")
	}
	subRow, err := moduleassignmentsubmissions.GetByIDForCourse(ctx, d.Pool, *cid, submissionID)
	if err != nil {
		return nil, nil, nil, errors.New("Failed to load submission.")
	}
	if subRow == nil || subRow.ModuleItemID != itemID {
		return nil, nil, nil, errors.New("Submission not found.")
	}
	return cid, assignRow, subRow, nil
}

func (d Deps) loadAssignmentForSubmissionsByIDs(
	ctx context.Context,
	courseCode string,
	itemID uuid.UUID,
) (*uuid.UUID, *coursemoduleassignments.CourseItemAssignmentRow, error) {
	if d.Pool == nil {
		return nil, nil, errors.New("server misconfiguration")
	}
	return loadAssignmentForSubmissionsByIDs(ctx, d.Pool, courseCode, itemID)
}

func (d Deps) executeSubmissionSyncToCanvas(ctx context.Context, in submissionSyncCanvasInput) (map[string]any, error) {
	if d.Pool == nil {
		return nil, errors.New("server misconfiguration")
	}
	cid, assignRow, subRow, err := d.loadSubmissionSyncContext(ctx, in.CourseCode, in.ItemID, in.SubmissionID)
	if err != nil {
		return nil, err
	}

	link, err := canvasimportjobs.LatestLinkedForCourse(ctx, d.Pool, in.CourseCode)
	if err != nil {
		return nil, errors.New("Failed to load Canvas link.")
	}
	if link == nil {
		return nil, errors.New("This course was not imported from Canvas.")
	}
	canvasBase, err := normalizeCanvasBaseURL(in.CanvasBaseURL, d.effectiveConfig().CanvasAllowedHostSuffixes)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(in.AccessToken)
	if token == "" {
		return nil, errors.New("Canvas access token is required.")
	}
	canvasCourseID, err := strconv.ParseInt(strings.TrimSpace(link.CanvasCourseID), 10, 64)
	if err != nil || canvasCourseID <= 0 {
		return nil, errors.New("Canvas course id is invalid for this import.")
	}

	var itemTitle string
	var storedCanvasAssignID *int64
	if err := d.Pool.QueryRow(ctx, `
		SELECT title, canvas_assignment_id FROM course.course_structure_items
		WHERE id = $1 AND course_id = $2 AND kind = 'assignment' AND archived = false`,
		in.ItemID, *cid).Scan(&itemTitle, &storedCanvasAssignID); err != nil {
		return nil, errors.New("Assignment not found.")
	}

	student, err := user.FindByID(ctx, d.Pool, subRow.SubmittedBy)
	if err != nil || student == nil {
		return nil, errors.New("Failed to load student.")
	}

	rubricDef, _ := parseAssignmentRubricJSON(assignRow.RubricJSON)
	pushGrade, gradeErr := resolveLexturesGradeForCanvasPush(
		ctx, d.Pool, *cid, subRow.SubmittedBy, in.ItemID, rubricDef, in.PointsEarned, in.RubricScores, in.InstructorComment,
	)
	if gradeErr != nil {
		return nil, gradeErr
	}

	client := canvasHTTPClient()
	// Prefer the Canvas assignment id captured at import time; fall back to title matching only for
	// items imported before ids were persisted (titles are not unique, so this is best-effort).
	var canvasAssignID int64
	if storedCanvasAssignID != nil && *storedCanvasAssignID > 0 {
		canvasAssignID = *storedCanvasAssignID
	} else {
		canvasAssignID, err = canvasFindAssignmentIDByTitle(ctx, client, canvasBase, token, canvasCourseID, itemTitle)
		if err != nil {
			return nil, err
		}
	}
	canvasUserID, err := canvasFindCanvasUserIDForEmail(ctx, client, canvasBase, token, canvasCourseID, student.Email)
	if err != nil {
		return nil, err
	}

	canvasAssign, err := canvasFetchAssignmentForGradePush(ctx, client, canvasBase, token, canvasCourseID, canvasAssignID)
	if err != nil {
		return nil, err
	}

	form := canvasBuildCanvasGradePushForm(pushGrade, rubricDef, canvasAssign)
	if len(form) == 0 {
		return nil, errors.New("No grade data to push to Canvas.")
	}

	path := fmt.Sprintf("courses/%d/assignments/%d/submissions/%d", canvasCourseID, canvasAssignID, canvasUserID)
	if _, err := canvasPutForm(ctx, client, canvasBase, token, path, form); err != nil {
		return nil, err
	}

	posting := strings.TrimSpace(assignRow.PostingPolicy)
	if posting == "" {
		posting = "automatic"
	}
	out := map[string]any{
		"submissionId":   in.SubmissionID.String(),
		"pointsEarned":   pushGrade.points,
		"maxPoints":      assignRow.PointsWorth,
		"posted":         posting == "automatic",
		"excused":        pushGrade.excused,
		"syncedToCanvas": true,
	}
	if pushGrade.comment != nil {
		out["instructorComment"] = *pushGrade.comment
	}
	if len(pushGrade.rubricScores) > 0 {
		out["rubricScores"] = pushGrade.rubricScores
	}
	return out, nil
}

type lexturesGradeForCanvasPush struct {
	points        float64
	rubricScores  map[string]float64
	rubricJSON    []byte
	comment       *string
	excused       bool
	hasNumeric    bool
}

func resolveLexturesGradeForCanvasPush(
	ctx context.Context,
	pool *pgxpool.Pool,
	cid, userID, itemID uuid.UUID,
	rubricDef *assignmentrubric.RubricDefinition,
	bodyPoints *float64,
	bodyRubric map[string]float64,
	bodyComment *string,
) (lexturesGradeForCanvasPush, error) {
	var out lexturesGradeForCanvasPush
	cell, err := coursegrades.GetCell(ctx, pool, cid, userID, itemID)
	if err != nil {
		return out, fmt.Errorf("failed to load grade")
	}

	if bodyComment != nil {
		t := strings.TrimSpace(*bodyComment)
		if t != "" {
			out.comment = &t
		}
	} else if cell != nil {
		comments := gradecomment.ResolveList(cell.InstructorCommentsJSON, cell.InstructorComment)
		out.comment = gradecomment.LatestBody(comments)
	}

	if cell != nil {
		out.excused = cell.Excused
	}

	if len(bodyRubric) > 0 {
		if rubricDef == nil {
			return out, fmt.Errorf("this assignment has no rubric in Lextures")
		}
		scores := make(map[uuid.UUID]float64, len(bodyRubric))
		for k, v := range bodyRubric {
			id, perr := uuid.Parse(strings.TrimSpace(k))
			if perr != nil {
				return out, fmt.Errorf("invalid rubric criterion id")
			}
			scores[id] = v
		}
		total, verr := assignmentrubric.ValidateRubricScoresForGrade(rubricDef, scores)
		if verr != nil {
			return out, verr
		}
		out.points = total
		out.hasNumeric = true
		out.rubricScores = bodyRubric
		out.rubricJSON, _ = json.Marshal(bodyRubric)
	} else if cell != nil {
		if scores, perr := coursegrades.ParseRubricScoresMap(cell.RubricScoresJSON); perr == nil && len(scores) > 0 {
			out.rubricScores = scores
			out.rubricJSON = cell.RubricScoresJSON
			if rubricDef != nil {
				uuidScores := parseUUIDScoreMap(scores)
				if total, verr := assignmentrubric.ValidateRubricScoresForGrade(rubricDef, uuidScores); verr == nil {
					out.points = total
					out.hasNumeric = true
				}
			}
		}
	}

	if bodyPoints != nil {
		out.points = *bodyPoints
		out.hasNumeric = true
	} else if !out.hasNumeric && cell != nil && cell.PointsEarned != nil {
		out.points = *cell.PointsEarned
		out.hasNumeric = true
	}

	if out.excused {
		out.hasNumeric = true
		return out, nil
	}
	if !out.hasNumeric {
		return out, fmt.Errorf("save a grade in Lextures before syncing to Canvas")
	}
	if out.points < 0 || out.points > 1e9 {
		return out, fmt.Errorf("invalid points value")
	}
	return out, nil
}

func canvasFetchAssignmentForGradePush(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID, canvasAssignID int64,
) (map[string]any, error) {
	assign, err := canvasGetObject(ctx, client, canvasBase, accessToken,
		fmt.Sprintf("courses/%d/assignments/%d", canvasCourseID, canvasAssignID),
		url.Values{"include[]": {"rubric"}})
	if err != nil || assign == nil {
		return assign, err
	}
	_ = canvasEnrichAssignmentWithRubric(ctx, client, canvasBase, accessToken, canvasCourseID, assign)
	return assign, nil
}

func canvasBuildCanvasGradePushForm(
	grade lexturesGradeForCanvasPush,
	lexRubric *assignmentrubric.RubricDefinition,
	canvasAssign map[string]any,
) url.Values {
	form := url.Values{}
	if grade.excused {
		form.Set("submission[excuse]", "true")
		return form
	}
	if len(grade.rubricScores) > 0 && lexRubric != nil {
		canvasCriteria := arrAt(canvasAssign, "rubric")
		titleToLexScore := make(map[string]float64, len(grade.rubricScores))
		for _, c := range lexRubric.Criteria {
			if pts, ok := grade.rubricScores[c.ID.String()]; ok {
				titleToLexScore[normalizeLinkMatchTitle(c.Title)] = pts
			}
		}
		rubricMapped := false
		for _, crit := range canvasCriteria {
			if crit == nil {
				continue
			}
			canvasCritID := strings.TrimSpace(strAt(crit, "id", ""))
			if canvasCritID == "" {
				continue
			}
			title := strings.TrimSpace(strAt(crit, "description", ""))
			if title == "" {
				title = strings.TrimSpace(strAt(crit, "title", ""))
			}
			pts, ok := titleToLexScore[normalizeLinkMatchTitle(title)]
			if !ok {
				continue
			}
			form.Set(fmt.Sprintf("rubric_assessment[%s][points]", canvasCritID), formatCanvasPostedGrade(pts))
			if ratingID := canvasBestRatingIDForPoints(crit, pts); ratingID != "" {
				form.Set(fmt.Sprintf("rubric_assessment[%s][rating_id]", canvasCritID), ratingID)
			}
			rubricMapped = true
		}
		if rubricMapped {
			if grade.comment != nil {
				form.Set("comment[text_comment]", *grade.comment)
			}
			return form
		}
	}
	if grade.hasNumeric {
		form.Set("submission[posted_grade]", formatCanvasPostedGrade(grade.points))
	}
	if grade.comment != nil {
		form.Set("comment[text_comment]", *grade.comment)
	}
	return form
}

func formatCanvasPostedGrade(points float64) string {
	return strconv.FormatFloat(points, 'f', -1, 64)
}

func canvasBestRatingIDForPoints(crit map[string]any, points float64) string {
	ratings := arrAt(crit, "ratings")
	var bestID string
	bestDiff := math.MaxFloat64
	for _, rating := range ratings {
		if rating == nil {
			continue
		}
		pts, ok := coerceCanvasJSONNumber(rating["points"])
		if !ok {
			continue
		}
		diff := math.Abs(pts - points)
		if diff < bestDiff {
			bestDiff = diff
			bestID = strings.TrimSpace(strAt(rating, "id", ""))
		}
	}
	return bestID
}

func canvasPutForm(
	ctx context.Context,
	client *http.Client,
	base, accessToken, path string,
	form url.Values,
) (map[string]any, error) {
	u := fmt.Sprintf("%s/api/v1/%s", strings.TrimRight(base, "/"), strings.TrimLeft(path, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.New("Failed to build Canvas request.")
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Could not reach Canvas (network error). Check the base URL and try again.")
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("Canvas rejected the access token (401). Create a token with permission to update grades and try again.")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("Canvas rejected the grade update (403). Your token may lack permission to manage grades in this course.")
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return nil, fmt.Errorf("Canvas API error HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("Canvas API error HTTP %d: %s", resp.StatusCode, msg)
	}
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var out any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, errors.New("Canvas returned invalid JSON.")
	}
	m, ok := out.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return m, nil
}

func canvasFindAssignmentIDByTitle(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	title string,
) (int64, error) {
	want := normalizeLinkMatchTitle(title)
	if want == "" {
		return 0, fmt.Errorf("assignment title is empty")
	}
	rows, err := canvasGetArrayPaginated(ctx, client, canvasBase, accessToken,
		fmt.Sprintf("courses/%d/assignments", canvasCourseID), nil)
	if err != nil {
		return 0, err
	}
	var matches []int64
	for _, row := range rows {
		if normalizeLinkMatchTitle(strAt(row, "name", "")) != want {
			continue
		}
		if id := int64At(row, "id"); id > 0 {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("could not find a matching Canvas assignment for %q", title)
	case 1:
		return matches[0], nil
	default:
		return 0, fmt.Errorf("multiple Canvas assignments match %q — rename for a unique title", title)
	}
}

func canvasFindCanvasUserIDForEmail(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	email string,
) (int64, error) {
	want := strings.ToLower(strings.TrimSpace(email))
	if want == "" || !strings.Contains(want, "@") {
		return 0, fmt.Errorf("student email is missing")
	}
	roster, err := canvasRosterEmailsByCanvasUserID(ctx, client, canvasBase, accessToken, canvasCourseID)
	if err != nil {
		return 0, err
	}
	var matches []int64
	for canvasUID, em := range roster {
		if strings.EqualFold(strings.TrimSpace(em), want) {
			matches = append(matches, canvasUID)
		}
	}
	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("could not find this student in the Canvas course roster (email %s)", email)
	case 1:
		return matches[0], nil
	default:
		return 0, fmt.Errorf("multiple Canvas users match this student's email")
	}
}

type canvasSyncedGrade struct {
	points          float64
	rubricJSON      []byte
	comment         *string
	commentsJSON    []byte
	excused         bool
	hasNumericScore bool
}

func canvasGradeFromSubmissionPayload(
	sub map[string]any,
	lexturesRubric *assignmentrubric.RubricDefinition,
	canvasUserToLocal map[int64]uuid.UUID,
) (canvasSyncedGrade, error) {
	var out canvasSyncedGrade
	if sub == nil {
		return out, fmt.Errorf("empty Canvas submission")
	}
	out.excused = boolAt(sub, "excused", false)
	if out.excused {
		out.hasNumericScore = true
	}

	gradedForImport := canvasSubmissionIsGradedForImport(sub)
	if gradedForImport && !out.excused {
		rubricScores, rubricTotal, hasRubric := canvasMapRubricAssessmentScores(sub, lexturesRubric)
		if hasRubric {
			out.points = rubricTotal
			out.hasNumericScore = true
			if len(rubricScores) > 0 {
				raw, err := json.Marshal(rubricScores)
				if err != nil {
					return out, err
				}
				out.rubricJSON = raw
			}
		}

		if exc, score, hasScore := canvasSubmissionEffectiveScore(sub); hasScore {
			if exc {
				out.excused = true
				out.hasNumericScore = true
			} else if !out.hasNumericScore {
				out.points = score
				out.hasNumericScore = true
			}
		}
	}

	comments := canvasSubmissionCommentsFromPayload(sub, canvasUserToLocal)
	if len(comments) > 0 {
		raw, err := gradecomment.MarshalList(comments)
		if err != nil {
			return out, err
		}
		out.commentsJSON = raw
		flat := gradecomment.Flatten(comments)
		if flat != "" {
			out.comment = &flat
		}
	}
	return out, nil
}

func canvasMapRubricAssessmentScores(
	sub map[string]any,
	lexturesRubric *assignmentrubric.RubricDefinition,
) (map[string]float64, float64, bool) {
	if lexturesRubric == nil || len(lexturesRubric.Criteria) == 0 {
		return nil, 0, false
	}
	assessment := objAt(sub, "rubric_assessment")
	if assessment == nil {
		return nil, 0, false
	}
	data := objAt(assessment, "data")
	if len(data) == 0 {
		if sc, ok := coerceCanvasJSONNumber(assessment["score"]); ok {
			return nil, sc, true
		}
		return nil, 0, false
	}

	canvasCriteria := canvasRubricCriteriaFromSubmission(sub)
	titleByCanvasCritID := canvasRubricCriterionTitles(canvasCriteria)
	lexByTitle := make(map[string]uuid.UUID, len(lexturesRubric.Criteria))
	for _, c := range lexturesRubric.Criteria {
		key := normalizeLinkMatchTitle(c.Title)
		if key != "" {
			lexByTitle[key] = c.ID
		}
	}

	scores := make(map[string]float64)
	for critKey, raw := range data {
		row, ok := raw.(map[string]any)
		if !ok || row == nil {
			continue
		}
		pts, ok := coerceCanvasJSONNumber(row["points"])
		if !ok {
			continue
		}
		title := titleByCanvasCritID[critKey]
		if title == "" {
			title = critKey
		}
		lexID, ok := lexByTitle[normalizeLinkMatchTitle(title)]
		if !ok {
			continue
		}
		scores[lexID.String()] = pts
	}
	if len(scores) == 0 {
		if sc, ok := coerceCanvasJSONNumber(assessment["score"]); ok {
			return nil, sc, true
		}
		return nil, 0, false
	}
	total, err := assignmentrubric.ValidateRubricScoresForGrade(lexturesRubric, parseUUIDScoreMap(scores))
	if err != nil {
		if sc, ok := coerceCanvasJSONNumber(assessment["score"]); ok {
			return scores, sc, true
		}
		var sum float64
		for _, v := range scores {
			sum += v
		}
		return scores, sum, true
	}
	return scores, total, true
}

func parseUUIDScoreMap(m map[string]float64) map[uuid.UUID]float64 {
	out := make(map[uuid.UUID]float64, len(m))
	for k, v := range m {
		id, err := uuid.Parse(k)
		if err != nil {
			continue
		}
		out[id] = v
	}
	return out
}

func canvasRubricCriteriaFromSubmission(sub map[string]any) []map[string]any {
	if sub == nil {
		return nil
	}
	if criteria := arrAt(sub, "rubric"); len(criteria) > 0 {
		return criteria
	}
	if assign := objAt(sub, "assignment"); assign != nil {
		if criteria := arrAt(assign, "rubric"); len(criteria) > 0 {
			return criteria
		}
	}
	return nil
}

// canvasSubmissionEffectiveScore reads Canvas score fields used in SpeedGrader.
// entered_score is set when a grade exists but has not been posted to the student yet.
func canvasSubmissionEffectiveScore(sub map[string]any) (excused bool, score float64, hasScore bool) {
	if sub == nil {
		return false, 0, false
	}
	if exc, sc, ok := submissionScoreAndExcused(sub); ok || exc {
		return exc, sc, ok
	}
	if !canvasSubmissionIsGradedForImport(sub) {
		return false, 0, false
	}
	if grade := strings.TrimSpace(strAt(sub, "grade", "")); grade != "" {
		if sc, ok := coerceCanvasJSONNumber(grade); ok {
			return false, sc, true
		}
	}
	if grade := strings.TrimSpace(strAt(sub, "entered_grade", "")); grade != "" {
		if sc, ok := coerceCanvasJSONNumber(grade); ok {
			return false, sc, true
		}
	}
	return false, 0, false
}

func canvasRubricCriterionTitles(criteria []map[string]any) map[string]string {
	out := make(map[string]string, len(criteria))
	for _, crit := range criteria {
		if crit == nil {
			continue
		}
		id := strings.TrimSpace(strAt(crit, "id", ""))
		if id == "" {
			continue
		}
		title := strings.TrimSpace(strAt(crit, "description", ""))
		if title == "" {
			title = strings.TrimSpace(strAt(crit, "title", ""))
		}
		if title != "" {
			out[id] = title
		}
	}
	return out
}

func canvasSubmissionCommentAuthorLabel(c map[string]any, studentCanvasUID int64) string {
	if c == nil {
		return "Comment"
	}
	if name := strings.TrimSpace(strAt(c, "author_name", "")); name != "" {
		return name
	}
	if author := objAt(c, "author"); author != nil {
		for _, k := range []string{"display_name", "name", "short_name", "sortable_name"} {
			if n := strings.TrimSpace(strAt(author, k, "")); n != "" {
				return n
			}
		}
	}
	authorID := int64At(c, "author_id")
	if studentCanvasUID > 0 && authorID == studentCanvasUID {
		return "Student"
	}
	if authorID > 0 {
		return fmt.Sprintf("User %d", authorID)
	}
	return "Comment"
}

func canvasNormalizeCanvasSubmissionCommentText(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	if strings.Contains(text, "<") && strings.Contains(text, ">") {
		if plain := strings.TrimSpace(htmlToPlainText(text)); plain != "" {
			return plain
		}
	}
	return text
}

// canvasInstructorCommentFromSubmission builds legacy flat text for tests and callers.
func canvasInstructorCommentFromSubmission(sub map[string]any) string {
	comments := canvasSubmissionCommentsFromPayload(sub, nil)
	flat := gradecomment.Flatten(comments)
	if len(flat) > maxInstructorCommentLen {
		flat = flat[:maxInstructorCommentLen]
	}
	return flat
}

// canvasSubmissionCommentsFromPayload builds structured SpeedGrader comments for import.
func canvasSubmissionCommentsFromPayload(
	sub map[string]any,
	canvasUserToLocal map[int64]uuid.UUID,
) []gradecomment.Comment {
	studentCanvasUID := int64At(sub, "user_id")
	type commentRow struct {
		created string
		author  string
		text    string
		key     string
		userID  *string
		avatar  *string
		source  string
	}
	rows := make([]commentRow, 0)
	appendCanvasSubmissionComments := func(comments []map[string]any) {
		for _, c := range comments {
			if c == nil {
				continue
			}
			text := canvasNormalizeCanvasSubmissionCommentText(strAt(c, "comment", ""))
			if text == "" {
				continue
			}
			author := canvasSubmissionCommentAuthorLabel(c, studentCanvasUID)
			created := strAt(c, "created_at", "")
			canvasCommentID := int64At(c, "id")
			key := fmt.Sprintf("%s|%s|%s|%d", created, author, text, canvasCommentID)
			var userID *string
			authorCanvasID := int64At(c, "author_id")
			if authorCanvasID > 0 && canvasUserToLocal != nil {
				if localID, ok := canvasUserToLocal[authorCanvasID]; ok {
					s := localID.String()
					userID = &s
				}
			}
			var avatar *string
			if authorObj := objAt(c, "author"); authorObj != nil {
				if u := strings.TrimSpace(strAt(authorObj, "avatar_url", "")); u != "" {
					avatar = &u
				}
			}
			rows = append(rows, commentRow{
				created: created,
				author:  author,
				text:    text,
				key:     key,
				userID:  userID,
				avatar:  avatar,
				source:  "canvas",
			})
		}
	}
	appendCanvasSubmissionComments(arrAt(sub, "submission_comments"))
	appendCanvasSubmissionComments(arrAt(sub, "submission_html_comments"))
	for _, hist := range arrAt(sub, "submission_history") {
		appendCanvasSubmissionComments(arrAt(hist, "submission_comments"))
		appendCanvasSubmissionComments(arrAt(hist, "submission_html_comments"))
	}
	for _, text := range canvasRubricCriterionComments(sub) {
		rows = append(rows, commentRow{
			author: "Rubric",
			text:   text,
			key:    "rubric|" + text,
			source: "rubric",
		})
	}
	if len(rows) == 0 {
		return nil
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].created < rows[j].created
	})
	out := make([]gradecomment.Comment, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, dup := seen[row.key]; dup {
			continue
		}
		seen[row.key] = struct{}{}
		id := row.key
		if strings.HasPrefix(row.source, "canvas") {
			id = "canvas-" + row.key
		}
		out = append(out, gradecomment.Comment{
			ID:          id,
			UserID:      row.userID,
			DisplayName: row.author,
			AvatarURL:   row.avatar,
			Body:        row.text,
			CreatedAt:   row.created,
			Source:      row.source,
		})
	}
	return out
}

func canvasRubricCriterionComments(sub map[string]any) []string {
	assessment := objAt(sub, "rubric_assessment")
	if assessment == nil {
		return nil
	}
	data := objAt(assessment, "data")
	if data == nil {
		return nil
	}
	titleByID := canvasRubricCriterionTitles(canvasRubricCriteriaFromSubmission(sub))
	out := make([]string, 0)
	for critKey, raw := range data {
		row, ok := raw.(map[string]any)
		if !ok || row == nil {
			continue
		}
		text := strings.TrimSpace(strAt(row, "comments", ""))
		if text == "" {
			continue
		}
		if label := titleByID[critKey]; label != "" {
			text = label + ": " + text
		}
		out = append(out, text)
	}
	return out
}