package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/gradingredaction"
	"github.com/lextures/lextures/server/internal/models/peerreview"
	prrepo "github.com/lextures/lextures/server/internal/repos/peerreview"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	peerreviewsvc "github.com/lextures/lextures/server/internal/service/peerreview"
)

func (d Deps) peerReviewFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFPeerReview {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Peer review is not enabled.")
		return true
	}
	return false
}

func (d Deps) peerReviewService() peerreviewsvc.Service {
	return peerreviewsvc.New(d.Pool)
}

func blindLabelRank(id uuid.UUID) int {
	b := id[:]
	n := int(b[0])<<8 | int(b[1])
	if n < 1 {
		n = 1
	}
	return (n % 999) + 1
}

func (d Deps) registerPeerReviewRoutes(r chi.Router) {
	r.Put("/api/v1/courses/{course_code}/assignments/{item_id}/peer-review", d.handlePutAssignmentPeerReviewConfig())
	r.Post("/api/v1/courses/{course_code}/assignments/{item_id}/peer-review/allocate", d.handlePostPeerReviewAllocate())
	r.Get("/api/v1/courses/{course_code}/assignments/{item_id}/peer-review/received", d.handleGetPeerReviewReceived())
	r.Get("/api/v1/courses/{course_code}/assignments/{item_id}/peer-review/summary", d.handleGetPeerReviewSummary())
	r.Get("/api/v1/peer-review/assigned", d.handleGetPeerReviewAssigned())
	r.Post("/api/v1/peer-review/allocations/{allocation_id}", d.handlePostPeerReviewSubmit())
	r.Post("/api/v1/courses/{course_code}/groups/{group_id}/team-eval", d.handlePostTeamPeerEval())
}

type peerReviewConfigBody struct {
	ReviewsPerReviewer int     `json:"reviewsPerReviewer"`
	Anonymity          string  `json:"anonymity"`
	OpensAt            *string `json:"opensAt"`
	ClosesAt           *string `json:"closesAt"`
	GradeMode          string  `json:"gradeMode"`
	BlendWeight        float64 `json:"blendWeight"`
	Aggregation        string  `json:"aggregation"`
	ExcludeSameGroup   *bool   `json:"excludeSameGroup"`
}

func parsePeerReviewTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	utc := t.UTC()
	return &utc, nil
}

func peerReviewConfigToJSON(cfg *prrepo.ConfigRow) map[string]any {
	out := map[string]any{
		"id":                 cfg.ID.String(),
		"assignmentId":       cfg.AssignmentID.String(),
		"reviewsPerReviewer": cfg.ReviewsPerReviewer,
		"anonymity":          string(cfg.Anonymity),
		"gradeMode":          string(cfg.GradeMode),
		"blendWeight":        cfg.BlendWeight,
		"aggregation":        string(cfg.Aggregation),
		"excludeSameGroup":   cfg.ExcludeSameGroup,
	}
	if cfg.OpensAt != nil {
		out["opensAt"] = cfg.OpensAt.Format(time.RFC3339)
	}
	if cfg.ClosesAt != nil {
		out["closesAt"] = cfg.ClosesAt.Format(time.RFC3339)
	}
	return out
}

func (d Deps) handlePutAssignmentPeerReviewConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.peerReviewFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor permission required.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		var body peerReviewConfigBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.ReviewsPerReviewer < 1 || body.ReviewsPerReviewer > 20 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "reviewsPerReviewer must be 1–20.")
			return
		}
		anonymity := peerreview.AnonymityMode(body.Anonymity)
		switch anonymity {
		case peerreview.AnonymityDoubleBlind, peerreview.AnonymityReviewerAnon, peerreview.AnonymityNamed:
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid anonymity mode.")
			return
		}
		gradeMode := peerreview.GradeMode(body.GradeMode)
		switch gradeMode {
		case peerreview.GradeModeNone, peerreview.GradeModeScoreOnly, peerreview.GradeModeWeightedBlend:
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid gradeMode.")
			return
		}
		aggregation := peerreview.Aggregation(body.Aggregation)
		switch aggregation {
		case peerreview.AggregationMean, peerreview.AggregationMedian, peerreview.AggregationTrimmed:
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid aggregation.")
			return
		}
		opensAt, err := parsePeerReviewTime(body.OpensAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid opensAt.")
			return
		}
		closesAt, err := parsePeerReviewTime(body.ClosesAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid closesAt.")
			return
		}
		excludeSameGroup := true
		if body.ExcludeSameGroup != nil {
			excludeSameGroup = *body.ExcludeSameGroup
		}
		blendWeight := body.BlendWeight
		if blendWeight < 0 || blendWeight > 1 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "blendWeight must be 0–1.")
			return
		}
		svc := d.peerReviewService()
		cfg, err := svc.UpsertConfig(r.Context(), prrepo.UpsertConfigInput{
			AssignmentID:       itemID,
			ReviewsPerReviewer: body.ReviewsPerReviewer,
			Anonymity:          anonymity,
			OpensAt:            opensAt,
			ClosesAt:           closesAt,
			GradeMode:          gradeMode,
			BlendWeight:        blendWeight,
			Aggregation:        aggregation,
			ExcludeSameGroup:   excludeSameGroup,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save peer review config.")
			return
		}
		writeJSON(w, http.StatusOK, peerReviewConfigToJSON(cfg))
	}
}

func (d Deps) handlePostPeerReviewAllocate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.peerReviewFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor permission required.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		svc := d.peerReviewService()
		result, err := svc.Allocate(r.Context(), *cid, itemID)
		if err != nil {
			switch err {
			case peerreviewsvc.ErrNoConfig:
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Peer review is not configured for this assignment.")
			case peerreviewsvc.ErrNotEnoughPeers:
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput, "Not enough submissions to allocate peer reviews.")
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Allocation failed.")
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"allocationsCreated": result.AllocationsCreated})
	}
}

func (d Deps) handleGetPeerReviewAssigned() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.peerReviewFeatureOff(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		// Cross-course list: find all reviewer allocations for this user's enrollments.
		rows, err := d.Pool.Query(r.Context(), `
SELECT ce.id
FROM course.course_enrollments ce
JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.user_id = $1 AND ce.active
`, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollments.")
			return
		}
		defer rows.Close()
		allocations := make([]map[string]any, 0)
		for rows.Next() {
			var enrollmentID uuid.UUID
			if err := rows.Scan(&enrollmentID); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollments.")
				return
			}
			allocs, err := prrepo.ListAllocationsForReviewer(r.Context(), d.Pool, enrollmentID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load allocations.")
				return
			}
			for _, a := range allocs {
				cfg, err := prrepo.GetConfigByAssignment(r.Context(), d.Pool, a.CourseID, a.AssignmentID)
				if err != nil || cfg == nil {
					continue
				}
				allocations = append(allocations, d.allocationToJSON(a, cfg, viewer))
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"allocations": allocations})
	}
}

func (d Deps) allocationToJSON(a prrepo.AllocationRow, cfg *prrepo.ConfigRow, viewer uuid.UUID) map[string]any {
	out := map[string]any{
		"id":               a.ID.String(),
		"configId":         a.ConfigID.String(),
		"assignmentId":     a.AssignmentID.String(),
		"courseId":         a.CourseID.String(),
		"courseCode":       a.CourseCode,
		"targetSubmissionId": a.TargetSubmissionID.String(),
		"status":           string(a.Status),
		"assignedAt":       a.AssignedAt.Format(time.RFC3339),
		"anonymity":        string(cfg.Anonymity),
	}
	labelRank := blindLabelRank(a.TargetSubmissionID)
	switch cfg.Anonymity {
	case peerreview.AnonymityDoubleBlind, peerreview.AnonymityReviewerAnon:
		out["targetLabel"] = gradingredaction.BlindStudentLabel(labelRank)
	case peerreview.AnonymityNamed:
		if a.ReviewerUserID == viewer {
			out["targetUserId"] = a.TargetUserID.String()
		}
	}
	return out
}

type peerReviewSubmitBody struct {
	Score        *float64           `json:"score"`
	RubricScores map[string]float64 `json:"rubricScores"`
	Comments     *string            `json:"comments"`
}

func (d Deps) handlePostPeerReviewSubmit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.peerReviewFeatureOff(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		allocationID, err := uuid.Parse(chi.URLParam(r, "allocation_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid allocation id.")
			return
		}
		alloc, err := prrepo.GetAllocationByID(r.Context(), d.Pool, allocationID)
		if err != nil || alloc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Allocation not found.")
			return
		}
		if alloc.ReviewerUserID != viewer {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You are not assigned to review this submission.")
			return
		}
		var body peerReviewSubmitBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		svc := d.peerReviewService()
		rev, err := svc.SubmitReview(r.Context(), allocationID, body.Score, body.RubricScores, body.Comments)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to submit review.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":           rev.ID.String(),
			"allocationId": rev.AllocationID.String(),
			"score":        rev.Score,
			"submittedAt":  rev.SubmittedAt.Format(time.RFC3339),
		})
	}
}

func (d Deps) handleGetPeerReviewReceived() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.peerReviewFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		cfg, err := prrepo.GetConfigByAssignment(r.Context(), d.Pool, *cid, itemID)
		if err != nil || cfg == nil {
			writeJSON(w, http.StatusOK, map[string]any{"reviews": []any{}})
			return
		}
		reviews, err := prrepo.ListReceivedReviewsForUser(r.Context(), d.Pool, cfg.ID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load reviews.")
			return
		}
		out := make([]map[string]any, 0, len(reviews))
		for i, rev := range reviews {
			item := map[string]any{
				"id":          rev.ID.String(),
				"score":       rev.Score,
				"comments":    rev.Comments,
				"submittedAt": rev.SubmittedAt.Format(time.RFC3339),
			}
			if cfg.Anonymity != peerreview.AnonymityNamed {
				item["reviewerLabel"] = gradingredaction.BlindStudentLabel(i + 1)
			}
			out = append(out, item)
		}
		writeJSON(w, http.StatusOK, map[string]any{"reviews": out})
	}
}

func (d Deps) handleGetPeerReviewSummary() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.peerReviewFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":gradebook:view"
		canView, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil || !canView {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Gradebook permission required.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		cfg, err := prrepo.GetConfigByAssignment(r.Context(), d.Pool, *cid, itemID)
		if err != nil || cfg == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Peer review is not configured.")
			return
		}
		svc := d.peerReviewService()
		summary, err := svc.BuildInstructorSummary(r.Context(), cfg)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build summary.")
			return
		}
		students := make([]map[string]any, 0, len(summary.SubmissionSummaries))
		for _, s := range summary.SubmissionSummaries {
			students = append(students, map[string]any{
				"submissionId":  s.SubmissionID.String(),
				"studentUserId": s.StudentUserID.String(),
				"peerAggregate": s.PeerAggregate,
				"reviewCount":   s.ReviewCount,
			})
		}
		incomplete := make([]string, 0, len(summary.IncompleteReviewers))
		for _, id := range summary.IncompleteReviewers {
			incomplete = append(incomplete, id.String())
		}
		outliers := make([]string, 0, len(summary.OutlierReviewers))
		for _, id := range summary.OutlierReviewers {
			outliers = append(outliers, id.String())
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"config":              peerReviewConfigToJSON(cfg),
			"totalAllocations":    summary.TotalAllocations,
			"completedReviews":    summary.CompletedReviews,
			"incompleteReviewers": incomplete,
			"outlierReviewers":    outliers,
			"submissions":         students,
		})
	}
}

type teamEvalBody struct {
	RateeEnrollmentID string  `json:"rateeEnrollmentId"`
	ContributionScore int     `json:"contributionScore"`
	Comment           *string `json:"comment"`
}

func (d Deps) handlePostTeamPeerEval() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.peerReviewFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		groupID, err := uuid.Parse(chi.URLParam(r, "group_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid group id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		raterEnrollmentID, err := prrepo.GetEnrollmentIDForUser(r.Context(), d.Pool, *cid, viewer)
		if err != nil || raterEnrollmentID == nil {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Student enrollment required.")
			return
		}
		var body teamEvalBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		rateeID, err := uuid.Parse(body.RateeEnrollmentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid rateeEnrollmentId.")
			return
		}
		if body.ContributionScore < 1 || body.ContributionScore > 5 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "contributionScore must be 1–5.")
			return
		}
		if *raterEnrollmentID == rateeID {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Cannot rate yourself.")
			return
		}
		if err := prrepo.UpsertTeamEvaluation(r.Context(), d.Pool, groupID, *raterEnrollmentID, rateeID, body.ContributionScore, body.Comment); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save team evaluation.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
