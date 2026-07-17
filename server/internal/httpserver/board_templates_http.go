package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/background"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func templateJSON(t board.Template) map[string]any {
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	def := json.RawMessage(`{}`)
	if len(t.Definition) > 0 {
		def = t.Definition
	}
	out := map[string]any{
		"id":          t.ID,
		"scope":       t.Scope,
		"title":       t.Title,
		"description": t.Description,
		"tags":        tags,
		"definition":  def,
		"createdAt":   t.CreatedAt.UTC().Format(time.RFC3339),
	}
	if t.CourseID != nil {
		out["courseId"] = *t.CourseID
	} else {
		out["courseId"] = nil
	}
	if t.OrgID != nil {
		out["orgId"] = *t.OrgID
	} else {
		out["orgId"] = nil
	}
	if t.CreatedBy != nil {
		out["createdBy"] = *t.CreatedBy
	} else {
		out["createdBy"] = nil
	}
	return out
}

func copyJobJSON(j board.CopyJob) map[string]any {
	out := map[string]any{
		"id":            j.ID,
		"sourceBoardId": j.SourceBoardID,
		"mode":          j.Mode,
		"title":         j.Title,
		"status":        j.Status,
		"progress":      j.Progress,
		"error":         j.Error,
		"createdAt":     j.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":     j.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if j.ResultBoardID != nil {
		out["resultBoardId"] = *j.ResultBoardID
	} else {
		out["resultBoardId"] = nil
	}
	return out
}

func requestLocale(r *http.Request) string {
	if q := strings.TrimSpace(r.URL.Query().Get("locale")); q != "" {
		return q
	}
	al := r.Header.Get("Accept-Language")
	if al == "" {
		return "en"
	}
	// Take first tag.
	part := strings.Split(al, ",")[0]
	return strings.TrimSpace(part)
}

// handleListBoardTemplates is GET /api/v1/board-templates.
func (d Deps) handleListBoardTemplates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		scope := strings.TrimSpace(r.URL.Query().Get("scope"))
		courseCode := strings.TrimSpace(r.URL.Query().Get("courseCode"))
		if courseCode == "" {
			courseCode = strings.TrimSpace(r.URL.Query().Get("course_code"))
		}
		if courseCode != "" {
			if !d.userHasCourseAccess(w, r, courseCode, viewer) {
				return
			}
			if d.visualBoardsFeatureOff(w, r, courseCode) {
				return
			}
		}
		orgID, _ := organization.OrgIDForUser(r.Context(), d.Pool, viewer)
		var orgPtr *uuid.UUID
		if orgID != uuid.Nil {
			orgPtr = &orgID
		}
		templates, err := board.ListTemplates(r.Context(), d.Pool, board.ListTemplatesFilter{
			Scope:      scope,
			CourseCode: courseCode,
			OrgID:      orgPtr,
			Query:      strings.TrimSpace(r.URL.Query().Get("q")),
			Locale:     requestLocale(r),
		})
		if err != nil {
			if strings.Contains(err.Error(), "scope") || strings.Contains(err.Error(), "courseCode") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list templates.")
			return
		}
		out := make([]map[string]any, 0, len(templates))
		for _, t := range templates {
			out = append(out, templateJSON(t))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"templates": out})
	}
}

// handleSaveBoardAsTemplate is POST .../boards/{board_id}/save-as-template.
func (d Deps) handleSaveBoardAsTemplate() http.HandlerFunc {
	type reqBody struct {
		Scope        string   `json:"scope"`
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		Tags         []string `json:"tags"`
		IncludePosts bool     `json:"includePosts"`
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
		if d.visualBoardsFeatureOff(w, r, courseCode) {
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
		scope, err := board.NormalizeTemplateScope(in.Scope)
		if err != nil || scope == board.TemplateScopeBuiltin {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "scope must be course or org.")
			return
		}
		var orgID *uuid.UUID
		if scope == board.TemplateScopeOrg {
			id, err := organization.OrgIDForUser(r.Context(), d.Pool, viewer)
			if err != nil || id == uuid.Nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization scope requires an organization membership.")
				return
			}
			courseOrg, err := board.CourseOrgID(r.Context(), d.Pool, courseCode)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course organization.")
				return
			}
			if courseOrg == nil || *courseOrg != id {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Board course is not in your organization.")
				return
			}
			orgID = &id
		}
		boardID := chi.URLParam(r, "board_id")
		title := strings.TrimSpace(in.Title)
		if title == "" {
			b, _ := board.Get(r.Context(), d.Pool, courseCode, boardID)
			if b != nil {
				title = b.Title
			}
		}
		tmpl, err := board.SaveAsTemplate(r.Context(), d.Pool, courseCode, boardID, viewer, board.SaveAsTemplateInput{
			Scope:        scope,
			Title:        title,
			Description:  in.Description,
			Tags:         in.Tags,
			IncludePosts: in.IncludePosts,
			OrgID:        orgID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "title") || strings.Contains(err.Error(), "scope") || strings.Contains(err.Error(), "org") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save template.")
			return
		}
		if tmpl == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.template.saved")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(templateJSON(*tmpl))
	}
}

// handleGetBoardCopyJob is GET .../board-copy-jobs/{job_id}.
func (d Deps) handleGetBoardCopyJob() http.HandlerFunc {
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
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		jobID := chi.URLParam(r, "job_id")
		j, err := board.GetCopyJobForCourse(r.Context(), d.Pool, courseCode, jobID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load copy job.")
			return
		}
		if j == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Copy job not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(copyJobJSON(*j))
	}
}

func (d Deps) userHasCourseAccess(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID) bool {
	has, err := enrollment.UserHasAccess(r.Context(), d.Pool, courseCode, viewer)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course access.")
		return false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return false
	}
	return true
}

// createBoardFromSelector handles ?from=template:… and ?from=board:…&mode=…
func (d Deps) createBoardFromSelector(
	w http.ResponseWriter,
	r *http.Request,
	targetCourseCode string,
	viewer uuid.UUID,
	title, description, from string,
) bool {
	from = strings.TrimSpace(from)
	if from == "" {
		return false
	}
	locale := requestLocale(r)
	switch {
	case strings.HasPrefix(from, "template:"):
		templateID := strings.TrimSpace(strings.TrimPrefix(from, "template:"))
		tmpl, err := board.GetTemplate(r.Context(), d.Pool, templateID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load template.")
			return true
		}
		if tmpl == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
			return true
		}
		board.ApplyBuiltinLocale(tmpl, locale)
		courseID, err := board.CourseIDByCode(r.Context(), d.Pool, targetCourseCode)
		if err != nil || courseID == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return true
		}
		orgID, _ := board.CourseOrgID(r.Context(), d.Pool, targetCourseCode)
		if !board.TemplateVisible(*tmpl, courseID, orgID) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Template is not available for this course.")
			return true
		}
		def, err := board.ParseDefinition(tmpl.Definition)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return true
		}
		if strings.TrimSpace(title) == "" {
			title = tmpl.Title
		}
		created, err := board.InstantiateFromDefinition(r.Context(), d.Pool, targetCourseCode, viewer, def, board.InstantiateOpts{
			Title:       title,
			Description: description,
			Locale:      locale,
			AuthorID:    viewer,
			BlobCopier:  d.boardBlobCopier(),
		})
		if err != nil {
			if strings.Contains(err.Error(), "title") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return true
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create board from template.")
			return true
		}
		if pol, ok := d.orgBoardPoliciesForCourse(w, r, targetCourseCode); ok {
			created = d.applyCreatePolicies(r, targetCourseCode, created, pol)
		}
		telemetry.RecordBusinessEvent("board.created.from_template")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(boardJSON(*created))
		return true

	case strings.HasPrefix(from, "board:"):
		sourceBoardID := strings.TrimSpace(strings.TrimPrefix(from, "board:"))
		mode := r.URL.Query().Get("mode")
		if mode == "" {
			mode = board.CopyModeStructure
		}
		normMode, err := board.NormalizeCopyMode(mode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return true
		}
		sourceCourseCode, err := board.GetBoardCourseCode(r.Context(), d.Pool, sourceBoardID)
		if err != nil || sourceCourseCode == "" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Source board not found.")
			return true
		}
		// Viewer must be able to view the source board.
		srcBoard, caps, ok := d.loadBoardWithAccess(w, r, sourceCourseCode, viewer, sourceBoardID)
		if !ok {
			return true
		}
		if !caps.CanView {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You cannot access the source board.")
			return true
		}
		_ = srcBoard

		// Target create permission already checked by caller; re-check for clarity (AC-5).
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+targetCourseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to create boards in the target course.")
			return true
		}

		if strings.TrimSpace(title) == "" {
			title = srcBoard.Title + " (copy)"
		}

		attCount := 0
		if normMode == board.CopyModeFull {
			attCount, _ = board.CountBoardAttachments(r.Context(), d.Pool, sourceCourseCode, sourceBoardID)
		}
		async := normMode == board.CopyModeFull &&
			attCount >= board.FullCopyAsyncThreshold &&
			d.effectiveConfig().BackgroundJobsEnabled

		if async {
			targetCourseID, err := board.CourseIDByCode(r.Context(), d.Pool, targetCourseCode)
			if err != nil || targetCourseID == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
				return true
			}
			srcID, err := uuid.Parse(sourceBoardID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid board id.")
				return true
			}
			job, err := board.CreateCopyJob(r.Context(), d.Pool, *targetCourseID, srcID, viewer, normMode, title)
			if err != nil || job == nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not enqueue board copy.")
				return true
			}
			if _, err := background.EnqueueBoardCopy(r.Context(), d.Pool, background.BoardCopyPayload{
				JobID:            job.ID,
				SourceCourseCode: sourceCourseCode,
				SourceBoardID:    sourceBoardID,
				TargetCourseCode: targetCourseCode,
				CreatedBy:        viewer.String(),
				Mode:             normMode,
				Title:            title,
				Description:      description,
			}); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not enqueue board copy.")
				return true
			}
			telemetry.RecordBusinessEvent("board.copy.enqueued")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"job": copyJobJSON(*job),
			})
			return true
		}

		created, err := board.CopyBoard(r.Context(), d.Pool, sourceCourseCode, sourceBoardID, targetCourseCode, viewer, board.CopyBoardOpts{
			Mode:        normMode,
			Title:       title,
			Description: description,
			AuthorID:    viewer,
			BlobCopier:  d.boardBlobCopier(),
		})
		if err != nil {
			if strings.Contains(err.Error(), "title") || strings.Contains(err.Error(), "mode") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return true
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not duplicate board.")
			return true
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Source board not found.")
			return true
		}
		telemetry.RecordBusinessEvent("board.duplicated")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(boardJSON(*created))
		return true

	default:
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "from must be template:{id} or board:{id}.")
		return true
	}
}
