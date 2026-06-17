package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/service/catalogsearch"
	"github.com/lextures/lextures/server/internal/service/coursereviews"
)

func (d Deps) courseReviewsFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFCourseReviews {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course reviews are not enabled.")
		return true
	}
	return false
}

func reviewToJSON(rv coursereviews.Review) map[string]any {
	out := map[string]any{
		"id":                  rv.ID.String(),
		"courseId":            rv.CourseID.String(),
		"reviewerId":          rv.ReviewerID.String(),
		"rating":              rv.Rating,
		"reviewerDisplayName": rv.ReviewerDisplayName,
		"isFlagged":           rv.IsFlagged,
		"createdAt":           rv.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":           rv.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if rv.ReviewText != nil {
		out["reviewText"] = *rv.ReviewText
	}
	if rv.CreatorResponse != nil {
		out["creatorResponse"] = *rv.CreatorResponse
	}
	return out
}

func summaryToJSON(s coursereviews.Summary) map[string]any {
	dist := map[string]int{}
	for k, v := range s.Distribution {
		dist[strconv.Itoa(k)] = v
	}
	out := map[string]any{
		"ratingCount":  s.RatingCount,
		"distribution": dist,
	}
	if s.AverageRating != nil {
		out["averageRating"] = *s.AverageRating
	}
	return out
}

// handleCourseReviewsList returns paginated public reviews for a course (no auth required).
func (d Deps) handleCourseReviewsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.courseReviewsFeatureOff(w) {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
		limit := 10
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}

		result, err := coursereviews.List(r.Context(), d.Pool, *cid, cursor, limit)
		if err != nil {
			if errors.Is(err, coursereviews.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid cursor.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load reviews.")
			return
		}

		reviews := make([]map[string]any, 0, len(result.Reviews))
		for i := range result.Reviews {
			reviews = append(reviews, reviewToJSON(result.Reviews[i]))
		}
		out := map[string]any{
			"summary": summaryToJSON(result.Summary),
			"reviews": reviews,
		}
		if result.NextCursor != "" {
			out["nextCursor"] = result.NextCursor
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleCourseReviewsEligibility returns whether the viewer may submit a review.
func (d Deps) handleCourseReviewsEligibility() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.courseReviewsFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		elig, err := coursereviews.CheckEligibility(r.Context(), d.Pool, *cid, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check eligibility.")
			return
		}
		out := map[string]any{
			"eligible":        elig.Eligible,
			"progressPercent": elig.ProgressPercent,
			"hasReview":       elig.HasReview,
			"canEdit":         elig.CanEdit,
		}
		if elig.ReviewID != nil {
			out["reviewId"] = elig.ReviewID.String()
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleCourseReviewsSubmit creates or updates the viewer's review (idempotent).
func (d Deps) handleCourseReviewsSubmit() http.HandlerFunc {
	type body struct {
		Rating     int     `json:"rating"`
		ReviewText *string `json:"reviewText"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.courseReviewsFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		enrolled, err := enrollment.UserHasStudentEquivalentEnrollment(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify enrollment.")
			return
		}
		if !enrolled {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only enrolled learners may submit reviews.")
			return
		}

		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Failed to read request.")
			return
		}
		var b body
		if err := json.Unmarshal(raw, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}

		rv, err := coursereviews.Submit(r.Context(), d.Pool, coursereviews.SubmitInput{
			CourseID:   *cid,
			ReviewerID: viewer,
			Rating:     b.Rating,
			ReviewText: b.ReviewText,
		}, time.Now().UTC())
		if errors.Is(err, coursereviews.ErrInsufficientProgress) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Complete at least 10% of the course before reviewing.")
			return
		}
		if errors.Is(err, coursereviews.ErrEditWindowExpired) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Reviews can only be edited within 30 days of submission.")
			return
		}
		if errors.Is(err, coursereviews.ErrInvalidRating) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Rating must be between 1 and 5.")
			return
		}
		if errors.Is(err, coursereviews.ErrReviewTextTooLong) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Review text must be at most 2000 characters.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save review.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(reviewToJSON(*rv))
	}
}

// handleCourseReviewFlag flags a review for moderation.
func (d Deps) handleCourseReviewFlag() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.courseReviewsFeatureOff(w) {
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		reviewID, err := uuid.Parse(chi.URLParam(r, "review_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid review ID.")
			return
		}
		if err := coursereviews.Flag(r.Context(), d.Pool, reviewID); err != nil {
			if errors.Is(err, coursereviews.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Review not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to flag review.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleCourseReviewResponse lets course staff reply publicly to a review.
func (d Deps) handleCourseReviewResponse() http.HandlerFunc {
	type body struct {
		Response string `json:"response"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.courseReviewsFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		isStaff, err := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only course staff may respond to reviews.")
			return
		}

		reviewID, err := uuid.Parse(chi.URLParam(r, "review_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid review ID.")
			return
		}

		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Failed to read request.")
			return
		}
		var b body
		if err := json.Unmarshal(raw, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}

		if err := coursereviews.SetCreatorResponse(r.Context(), d.Pool, reviewID, b.Response); err != nil {
			if errors.Is(err, coursereviews.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Review not found.")
				return
			}
			if errors.Is(err, coursereviews.ErrEmptyResponse) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Response text is required.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save response.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleAdminReviewsList returns flagged reviews for the moderation queue.
func (d Deps) handleAdminReviewsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.courseReviewsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		reviews, err := coursereviews.ListFlagged(r.Context(), d.Pool, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load reviews.")
			return
		}
		out := make([]map[string]any, 0, len(reviews))
		for i := range reviews {
			out = append(out, reviewToJSON(reviews[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"reviews": out})
	}
}

// handleAdminReviewDelete soft-deletes a review and recomputes aggregates.
func (d Deps) handleAdminReviewDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.courseReviewsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		reviewID, err := uuid.Parse(chi.URLParam(r, "review_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid review ID.")
			return
		}
		if err := coursereviews.AdminRemove(r.Context(), d.Pool, reviewID); err != nil {
			if errors.Is(err, coursereviews.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Review not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to remove review.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handlePublicCatalogReviews lists reviews for a public catalog course by slug.
func (d Deps) handlePublicCatalogReviews() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicCatalogOff(w) || d.courseReviewsFeatureOff(w) {
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		svc := catalogsearch.New(d.Pool)
		c, err := svc.CourseBySlug(r.Context(), slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(c.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}
		cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
		limit := 10
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		result, err := coursereviews.List(r.Context(), d.Pool, courseID, cursor, limit)
		if err != nil {
			if errors.Is(err, coursereviews.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid cursor.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load reviews.")
			return
		}
		reviews := make([]map[string]any, 0, len(result.Reviews))
		for i := range result.Reviews {
			reviews = append(reviews, reviewToJSON(result.Reviews[i]))
		}
		out := map[string]any{
			"summary": summaryToJSON(result.Summary),
			"reviews": reviews,
		}
		if result.NextCursor != "" {
			out["nextCursor"] = result.NextCursor
		}
		publicCatalogCacheHeaders(w)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
