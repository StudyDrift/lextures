package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/service/managedlearners"
)

// resolvePurchaseLearner checks that buyer may enroll learnerID in the course.
// An empty learnerID means "buyer only" and returns (nil, true).
func (d Deps) resolvePurchaseLearner(w http.ResponseWriter, r *http.Request, buyer uuid.UUID, courseCode, learnerID string) (*uuid.UUID, bool) {
	learnerID = strings.TrimSpace(learnerID)
	if learnerID == "" {
		return nil, true
	}
	if !d.effectiveConfig().FFHomeschoolManagedLearners {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Managed learners are not enabled.")
		return nil, false
	}
	id, err := uuid.Parse(learnerID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid learnerUserId.")
		return nil, false
	}
	if id == buyer {
		return nil, true
	}
	orgID, err := repoCourse.CourseOrgID(r.Context(), d.Pool, courseCode)
	if err != nil || orgID == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return nil, false
	}
	can, err := managedlearners.CanEnrollManaged(r.Context(), d.Pool, buyer, id, *orgID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify learner.")
		return nil, false
	}
	if !can {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You cannot assign this course to that learner.")
		return nil, false
	}
	return &id, true
}

func (d Deps) handleAssignPurchase() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.courseMarketplaceOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		courseCode := strings.TrimSpace(chi.URLParam(r, "courseCode"))
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
			return
		}
		cid, err := repoCourse.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		owned, err := repoBilling.MarketplaceAccess(r.Context(), d.Pool, userID, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check ownership.")
			return
		}
		if !owned {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Purchase this course before assigning it.")
			return
		}
		var body struct {
			LearnerUserID string `json:"learnerUserId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		enrollID, ok := d.resolvePurchaseLearner(w, r, userID, courseCode, body.LearnerUserID)
		if !ok {
			return
		}
		if enrollID == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Choose a learner to assign this course to.")
			return
		}
		created, err := courseroles.EnrollStudentWithGrants(r.Context(), d.Pool, *cid, *enrollID, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to assign course.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enrolled":        true,
			"alreadyEnrolled": !created,
			"courseCode":      courseCode,
			"learnerUserId":   enrollID.String(),
		})
	}
}
