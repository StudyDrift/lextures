package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// interactiveQuizzesFeatureOff returns true when Live Quizzes are disabled for the course.
// Access is controlled only by the per-course interactiveQuizzesEnabled flag (no platform master switch).
func (d Deps) interactiveQuizzesFeatureOff(w http.ResponseWriter, r *http.Request, courseCode string) bool {
	crow, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || crow == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return true
	}
	if !crow.InteractiveQuizzesEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Live Quizzes are not enabled for this course.")
		return true
	}
	return false
}

func kitJSON(k quizgame.Kit) map[string]any {
	out := map[string]any{
		"id":            k.ID,
		"courseId":      nil,
		"title":         k.Title,
		"description":   k.Description,
		"slug":          k.Slug,
		"coverImageRef": k.CoverImageRef,
		"status":        k.Status,
		"visibility":    k.Visibility,
		"tags":          k.Tags,
		"questionCount": k.QuestionCount,
		"archived":      k.Archived,
		"createdBy":     k.CreatedBy,
		"createdAt":     k.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":     k.UpdatedAt.UTC().Format(time.RFC3339),
		"isTemplate":    k.IsTemplate,
		"attribution":   k.Attribution,
		"catalogStatus": k.CatalogStatus,
		"subject":       k.Subject,
		"gradeBand":     k.GradeBand,
		"language":      k.Language,
	}
	if k.CourseID != "" {
		out["courseId"] = k.CourseID
	}
	if k.TemplateScope != nil {
		out["templateScope"] = *k.TemplateScope
	} else {
		out["templateScope"] = nil
	}
	if k.DerivedFromKitID != nil {
		out["derivedFromKitId"] = *k.DerivedFromKitID
	} else {
		out["derivedFromKitId"] = nil
	}
	return out
}

// handleListQuizKits is GET /api/v1/courses/{course_code}/live-quizzes/kits.
func (d Deps) handleListQuizKits() http.HandlerFunc {
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
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		pageSize, _ := strconv.Atoi(q.Get("pageSize"))
		opts := quizgame.ListOpts{
			Query:           q.Get("q"),
			Tag:             q.Get("tag"),
			IncludeArchived: strings.EqualFold(q.Get("includeArchived"), "true"),
			Page:            page,
			PageSize:        pageSize,
		}
		result, err := quizgame.List(r.Context(), d.Pool, courseCode, opts)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list quiz kits.")
			return
		}
		out := make([]map[string]any, 0, len(result.Kits))
		for _, k := range result.Kits {
			out = append(out, kitJSON(k))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"kits":       out,
			"total":      result.Total,
			"page":       result.Page,
			"pageSize":   result.PageSize,
			"totalPages": result.TotalPages,
		})
	}
}

// handleCreateQuizKit is POST /api/v1/courses/{course_code}/live-quizzes/kits.
func (d Deps) handleCreateQuizKit() http.HandlerFunc {
	type reqBody struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := quizgame.CheckKitsPerCourseQuota(r.Context(), d.Pool, courseCode); err != nil {
			if errors.Is(err, quizgame.ErrKitsPerCourseQuota) {
				telemetry.RecordBusinessEvent("quizgame.quota.kits_rejected")
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Kit quota for this course has been reached.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not verify kit quotas.")
			return
		}
		created, err := quizgame.Create(r.Context(), d.Pool, courseCode, viewer, in.Title, in.Description, in.Tags)
		if err != nil {
			if strings.Contains(err.Error(), "title") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create quiz kit.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.kit.created")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(kitJSON(*created))
	}
}

// handleGetQuizKit is GET /api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}.
func (d Deps) handleGetQuizKit() http.HandlerFunc {
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
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		k, err := quizgame.Get(r.Context(), d.Pool, courseCode, kitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load quiz kit.")
			return
		}
		if k == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(kitJSON(*k))
	}
}

// handlePatchQuizKit is PATCH /api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}.
func (d Deps) handlePatchQuizKit() http.HandlerFunc {
	type reqBody struct {
		Title         *string   `json:"title"`
		Description   *string   `json:"description"`
		CoverImageRef *string   `json:"coverImageRef"`
		Status        *string   `json:"status"`
		Visibility    *string   `json:"visibility"`
		Tags          *[]string `json:"tags"`
		Archived      *bool     `json:"archived"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		updated, err := quizgame.Patch(r.Context(), d.Pool, courseCode, kitID, quizgame.PatchKitInput{
			Title:         in.Title,
			Description:   in.Description,
			CoverImageRef: in.CoverImageRef,
			Status:        in.Status,
			Visibility:    in.Visibility,
			Tags:          in.Tags,
			Archived:      in.Archived,
		})
		if err != nil {
			if strings.Contains(err.Error(), "title") || strings.Contains(err.Error(), "status") ||
				strings.Contains(err.Error(), "visibility") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update quiz kit.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		if in.Archived != nil && *in.Archived {
			telemetry.RecordBusinessEvent("quizgame.kit.archived")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(kitJSON(*updated))
	}
}

// handleDuplicateQuizKit is POST /api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/duplicate.
// Optional JSON body: { "targetCourseCode": "..." } for cross-course deep copy (IQ.8).
func (d Deps) handleDuplicateQuizKit() http.HandlerFunc {
	return d.handleDuplicateQuizKitV2()
}

// handleArchiveQuizKit is POST /api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/archive.
func (d Deps) handleArchiveQuizKit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		updated, err := quizgame.Archive(r.Context(), d.Pool, courseCode, kitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not archive quiz kit.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.kit.archived")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(kitJSON(*updated))
	}
}

// handleRestoreQuizKit is POST /api/v1/courses/{course_code}/live-quizzes/kits/{kit_id}/restore.
func (d Deps) handleRestoreQuizKit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		updated, err := quizgame.Restore(r.Context(), d.Pool, courseCode, kitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not restore quiz kit.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.kit.restored")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(kitJSON(*updated))
	}
}
