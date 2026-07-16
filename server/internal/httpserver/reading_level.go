package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	rlrepo "github.com/lextures/lextures/server/internal/repos/readinglevel"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/contentsimplificationai"
	readingsvc "github.com/lextures/lextures/server/internal/service/readinglevel"
)

const (
	readingLevelFeature        = aigateway.FeatureReadingLevelSimplification
	readingLevelSimplifyModel    = contentsimplificationai.DefaultModel
	defaultCourseGradeLevel      = 8
	readingLevelWarningOffset    = 2
)

func (d Deps) readingLevelEnabled() bool {
	return d.effectiveConfig().ReadingLevelEnabled
}

// enrichModuleItemResponse attaches FKGL metadata and optional simplified markdown for accommodated learners.
func (d Deps) enrichModuleItemResponse(r *http.Request, courseID, itemID uuid.UUID, itemType rlrepo.ItemType, viewer uuid.UUID, canEdit bool, resp *moduleAssignmentGetResponse) {
	if !d.readingLevelEnabled() || resp == nil {
		return
	}
	ctx := r.Context()
	stored, err := rlrepo.GetScore(ctx, d.Pool, itemID, itemType)
	if err == nil {
		resp.ReadingLevelFkgl = stored.FKGL
		resp.ReadingLevelFre = stored.FRE
	}
	if canEdit {
		return
	}
	override, err := rlrepo.EnrollmentReadingOverride(ctx, d.Pool, courseID, viewer)
	if err != nil || override == nil {
		return
	}
	cached, err := rlrepo.GetSimplified(ctx, d.Pool, itemID, itemType, *override)
	if err != nil || cached == nil {
		return
	}
	orig := resp.Markdown
	resp.OriginalMarkdown = &orig
	resp.Markdown = cached.SimplifiedText
	resp.SimplifiedForReadingLevel = true
	resp.ReadingLevelTargetFkgl = override
}

func (d Deps) requireReadingLevelEnabled(w http.ResponseWriter) bool {
	if !d.readingLevelEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Reading level is not enabled.")
		return false
	}
	return true
}

type readingLevelJSON struct {
	FKGL               *float64 `json:"fkgl,omitempty"`
	FRE                *float64 `json:"fre,omitempty"`
	Sufficient         bool     `json:"sufficient"`
	WordCount          int      `json:"wordCount,omitempty"`
	SimplifiedMarkdown *string  `json:"simplifiedMarkdown,omitempty"`
	SimplifiedNotice   bool     `json:"simplifiedNotice,omitempty"`
	TargetFKGL         *int     `json:"targetFkgl,omitempty"`
	AboveThreshold     bool     `json:"aboveThreshold,omitempty"`
}

func (d Deps) registerReadingLevelRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/items/{item_id}/reading-level", d.handleGetItemReadingLevel())
	r.Post("/api/v1/courses/{course_code}/items/{item_id}/simplify", d.handlePostItemSimplify())
	r.Get("/api/v1/courses/{course_code}/items/{item_id}/simplify/{grade}", d.handleGetItemSimplifyCached())
	r.Post("/api/v1/courses/{course_code}/bulk-reading-level", d.handlePostBulkReadingLevel())
	r.Patch("/api/v1/courses/{course_code}/enrollments/{enrollment_id}/reading-level", d.handlePatchEnrollmentReadingLevel())
}

func (d Deps) resolveCourseItem(w http.ResponseWriter, r *http.Request) (courseCode string, courseID uuid.UUID, itemID uuid.UUID, itemType rlrepo.ItemType, ok bool) {
	courseCode, _, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, "", false
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return "", uuid.Nil, uuid.Nil, "", false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, uuid.Nil, "", false
	}
	itemType, err = rlrepo.ResolveItemType(r.Context(), d.Pool, *cid, itemID)
	if errors.Is(err, pgx.ErrNoRows) {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Item not found.")
		return "", uuid.Nil, uuid.Nil, "", false
	}
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This item type does not support reading level.")
		return "", uuid.Nil, uuid.Nil, "", false
	}
	return courseCode, *cid, itemID, itemType, true
}

func (d Deps) buildReadingLevelResponse(ctx context.Context, courseID, itemID uuid.UUID, itemType rlrepo.ItemType, viewer uuid.UUID, markdown string) readingLevelJSON {
	out := readingLevelJSON{}
	if markdown != "" {
		sc := readingsvc.Analyze(readingsvc.PlainTextFromMarkdown(markdown))
		out.WordCount = sc.WordCount
		out.Sufficient = sc.Sufficient
		if sc.Sufficient {
			f, r := sc.FKGL, sc.FRE
			out.FKGL = &f
			out.FRE = &r
			warnAt := float64(defaultCourseGradeLevel + readingLevelWarningOffset)
			out.AboveThreshold = sc.FKGL > warnAt
		}
	}
	stored, _ := rlrepo.GetScore(ctx, d.Pool, itemID, itemType)
	if stored.FKGL != nil && out.FKGL == nil {
		out.FKGL = stored.FKGL
		out.FRE = stored.FRE
		out.Sufficient = true
	}
	override, _ := rlrepo.EnrollmentReadingOverride(ctx, d.Pool, courseID, viewer)
	if override != nil {
		if cached, _ := rlrepo.GetSimplified(ctx, d.Pool, itemID, itemType, *override); cached != nil {
			out.SimplifiedMarkdown = &cached.SimplifiedText
			out.SimplifiedNotice = true
			t := *override
			out.TargetFKGL = &t
		}
	}
	return out
}

// handleGetItemReadingLevel is GET /api/v1/courses/{course_code}/items/{item_id}/reading-level
func (d Deps) handleGetItemReadingLevel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireReadingLevelEnabled(w) {
			return
		}
		_, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		_, courseID, itemID, itemType, ok := d.resolveCourseItem(w, r)
		if !ok {
			return
		}
		md, err := rlrepo.GetMarkdown(r.Context(), d.Pool, itemID, itemType)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load content.")
			return
		}
		out := d.buildReadingLevelResponse(r.Context(), courseID, itemID, itemType, viewer, md)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

type simplifyRequest struct {
	TargetFKGL int    `json:"targetFkgl"`
	Text       string `json:"text"`
}

type simplifyResponse struct {
	Original   string  `json:"original"`
	Simplified string  `json:"simplified"`
	TargetFKGL int     `json:"targetFkgl"`
	ComputedFKGL *float64 `json:"computedFkgl,omitempty"`
	Cached     bool    `json:"cached"`
}

// handlePostItemSimplify is POST /api/v1/courses/{course_code}/items/{item_id}/simplify
func (d Deps) handlePostItemSimplify() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireReadingLevelEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to simplify content.")
			return
		}
		_, _, itemID, itemType, ok := d.resolveCourseItem(w, r)
		if !ok {
			return
		}
		var req simplifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if req.TargetFKGL < 0 || req.TargetFKGL > 12 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "targetFkgl must be between 0 and 12.")
			return
		}
		source := strings.TrimSpace(req.Text)
		if source == "" {
			var err error
			source, err = rlrepo.GetMarkdown(r.Context(), d.Pool, itemID, itemType)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load content.")
				return
			}
		}
		if cached, _ := rlrepo.GetSimplified(r.Context(), d.Pool, itemID, itemType, req.TargetFKGL); cached != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(simplifyResponse{
				Original:     source,
				Simplified:   cached.SimplifiedText,
				TargetFKGL:   req.TargetFKGL,
				ComputedFKGL: cached.ComputedFKGL,
				Cached:       true,
			})
			return
		}
		orgID := d.orgIDPtrForUser(r.Context(), userID)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		plain := readingsvc.PlainTextFromMarkdown(source)
		if !d.enforceAIGateway(w, r, userID, readingLevelFeature, readingLevelSimplifyModel, plain) {
			return
		}
		bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
		simplified, callMeta, err := contentsimplificationai.Simplify(r.Context(), bound, readingLevelSimplifyModel, plain, req.TargetFKGL)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Simplification failed.")
			return
		}
		sc := readingsvc.Analyze(simplified)
		var computed *float64
		if sc.Sufficient {
			v := sc.FKGL
			computed = &v
		}
		_ = rlrepo.UpsertSimplified(r.Context(), d.Pool, itemID, itemType, req.TargetFKGL, simplified, computed)
		dec := aigateway.Decision{OptInConfirmed: true}
		d.logAIInferenceAllowedWithProvider(r, userID, readingLevelFeature, readingLevelSimplifyModel, string(callMeta.Provider), plain, dec)
		d.recordAIProviderUsage(r.Context(), AIUsageMeta{UserID: userID, Feature: readingLevelFeature, Model: readingLevelSimplifyModel}, callMeta, true)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(simplifyResponse{
			Original:     source,
			Simplified:   simplified,
			TargetFKGL:   req.TargetFKGL,
			ComputedFKGL: computed,
			Cached:       false,
		})
	}
}

// handleGetItemSimplifyCached is GET .../simplify/{grade} — students with accommodation only.
func (d Deps) handleGetItemSimplifyCached() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireReadingLevelEnabled(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		_, courseID, itemID, itemType, ok := d.resolveCourseItem(w, r)
		if !ok {
			return
		}
		grade, err := strconv.Atoi(chi.URLParam(r, "grade"))
		if err != nil || grade < 0 || grade > 12 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid grade level.")
			return
		}
		override, err := rlrepo.EnrollmentReadingOverride(r.Context(), d.Pool, courseID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify accommodation.")
			return
		}
		if override == nil || *override != grade {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Reading level accommodation not set for this grade.")
			return
		}
		cached, err := rlrepo.GetSimplified(r.Context(), d.Pool, itemID, itemType, grade)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load simplified content.")
			return
		}
		if cached == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Simplified version not available.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"simplified":   cached.SimplifiedText,
			"targetFkgl":   grade,
			"computedFkgl": cached.ComputedFKGL,
		})
	}
}

// handlePostBulkReadingLevel scores all scorable items in a course.
func (d Deps) handlePostBulkReadingLevel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireReadingLevelEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		ids, types, err := rlrepo.ListScorableItemIDs(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list items.")
			return
		}
		scored := 0
		for i, id := range ids {
			md, err := rlrepo.GetMarkdown(r.Context(), d.Pool, id, types[i])
			if err != nil {
				continue
			}
			if err := rlrepo.ScoreAndPersist(r.Context(), d.Pool, id, types[i], md); err == nil {
				scored++
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]int{"scored": scored, "total": len(ids)})
	}
}

type patchEnrollmentReadingLevelRequest struct {
	ReadingLevelOverride *int `json:"readingLevelOverride"`
}

func (d Deps) handlePatchEnrollmentReadingLevel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireReadingLevelEnabled(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		can, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":enrollments:update")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !can {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to manage enrollments.")
			return
		}
		eid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "enrollment_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		var body patchEnrollmentReadingLevelRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.ReadingLevelOverride != nil {
			v := *body.ReadingLevelOverride
			if v < 0 || v > 12 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "readingLevelOverride must be 0–12.")
				return
			}
		}
		var exists bool
		err = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
  SELECT 1 FROM course.course_enrollments ce
  INNER JOIN course.courses c ON c.id = ce.course_id
  WHERE ce.id = $1 AND c.course_code = $2 AND ce.active
)`, eid, courseCode).Scan(&exists)
		if err != nil || !exists {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
			return
		}
		if err := rlrepo.SetEnrollmentReadingOverride(r.Context(), d.Pool, eid, body.ReadingLevelOverride); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update accommodation.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
