package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	svcadmindaudit "github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func kitJSONFull(k quizgame.Kit) map[string]any {
	return kitJSON(k)
}

func kitShareJSON(s quizgame.KitShare) map[string]any {
	return map[string]any{
		"id":          s.ID,
		"kitId":       s.KitID,
		"granteeType": s.GranteeType,
		"granteeId":   s.GranteeID,
		"permission":  s.Permission,
		"createdBy":   s.CreatedBy,
		"createdAt":   s.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (d Deps) auditKitShare(r *http.Request, actor uuid.UUID, eventType string, kitID string, after any) {
	orgID, _ := organization.OrgIDForUser(r.Context(), d.Pool, actor)
	var orgPtr *uuid.UUID
	if orgID != uuid.Nil {
		orgPtr = &orgID
	}
	targetType := "quizgame_kit"
	tid, err := uuid.Parse(kitID)
	if err != nil {
		return
	}
	var afterBytes []byte
	if after != nil {
		afterBytes, _ = json.Marshal(after)
	}
	_, _ = svcadmindaudit.Record(r.Context(), d.Pool, svcadmindaudit.RecordParams{
		OrgID:      orgPtr,
		EventType:  eventType,
		ActorID:    actor,
		TargetType: &targetType,
		TargetID:   &tid,
		AfterValue: afterBytes,
	})
}

// handleDuplicateQuizKit body may include targetCourseCode for cross-course copy.
func (d Deps) handleDuplicateQuizKitV2() http.HandlerFunc {
	type reqBody struct {
		TargetCourseCode string `json:"targetCourseCode"`
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
		var in reqBody
		if r.Body != nil && r.ContentLength != 0 {
			_ = json.NewDecoder(r.Body).Decode(&in)
		}
		target := strings.TrimSpace(in.TargetCourseCode)
		if target == "" {
			target = courseCode
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+target+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if target != courseCode {
			if d.interactiveQuizzesFeatureOff(w, r, target) {
				return
			}
		}
		kitID := chi.URLParam(r, "kit_id")
		created, err := quizgame.Duplicate(r.Context(), d.Pool, courseCode, kitID, viewer, target)
		if err != nil {
			if strings.Contains(err.Error(), "target course") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not duplicate quiz kit.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.kit.duplicated")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(kitJSONFull(*created))
	}
}

// handleSaveQuizKitAsTemplate is POST .../kits/{kit_id}/save-as-template.
func (d Deps) handleSaveQuizKitAsTemplate() http.HandlerFunc {
	type reqBody struct {
		Scope       string   `json:"scope"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
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
		scope, err := quizgame.NormalizeTemplateScope(in.Scope)
		if err != nil || scope == quizgame.TemplateScopeSystem {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "scope must be course or org.")
			return
		}
		var orgID *uuid.UUID
		if scope == quizgame.TemplateScopeOrg {
			id, err := organization.OrgIDForUser(r.Context(), d.Pool, viewer)
			if err != nil || id == uuid.Nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization scope requires an organization membership.")
				return
			}
			orgID = &id
		}
		kitID := chi.URLParam(r, "kit_id")
		tmpl, err := quizgame.SaveAsTemplate(r.Context(), d.Pool, courseCode, kitID, viewer, quizgame.SaveAsTemplateInput{
			Scope:       scope,
			Title:       in.Title,
			Description: in.Description,
			Tags:        in.Tags,
			OrgID:       orgID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "scope") || strings.Contains(err.Error(), "org") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save template.")
			return
		}
		if tmpl == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.template.saved")
		d.auditKitShare(r, viewer, "quizgame.kit.save_as_template", tmpl.ID, kitJSONFull(*tmpl))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(kitJSONFull(*tmpl))
	}
}

// handleListQuizTemplates is GET /api/v1/live-quizzes/templates.
func (d Deps) handleListQuizTemplates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
				return
			}
		}
		orgID, _ := quizgame.ResolveOrgIDForTemplates(r.Context(), d.Pool, viewer)
		templates, err := quizgame.ListTemplates(r.Context(), d.Pool, quizgame.ListTemplatesFilter{
			Scope:      scope,
			CourseCode: courseCode,
			OrgID:      orgID,
			Query:      strings.TrimSpace(r.URL.Query().Get("q")),
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
			out = append(out, kitJSONFull(t))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"templates": out})
	}
}

// handleCreateKitFromTemplate is POST /api/v1/live-quizzes/templates/{id}/create-kit.
func (d Deps) handleCreateKitFromTemplate() http.HandlerFunc {
	type reqBody struct {
		TargetCourseCode string `json:"targetCourseCode"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		target := strings.TrimSpace(in.TargetCourseCode)
		if target == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "targetCourseCode is required.")
			return
		}
		if !d.userHasCourseAccess(w, r, target, viewer) {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, target) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+target+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		templateID := chi.URLParam(r, "id")
		created, err := quizgame.CreateKitFromTemplate(r.Context(), d.Pool, templateID, target, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create kit from template.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.template.used")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(kitJSONFull(*created))
	}
}

// handleListQuizKitShares is GET .../kits/{kit_id}/shares.
func (d Deps) handleListQuizKitShares() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		shares, err := quizgame.ListShares(r.Context(), d.Pool, courseCode, kitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not list shares.")
			return
		}
		if shares == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		out := make([]map[string]any, 0, len(shares))
		for _, s := range shares {
			out = append(out, kitShareJSON(s))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"shares": out})
	}
}

// handleCreateQuizKitShare is POST .../kits/{kit_id}/shares.
func (d Deps) handleCreateQuizKitShare() http.HandlerFunc {
	type reqBody struct {
		GranteeType string  `json:"granteeType"`
		GranteeID   *string `json:"granteeId"`
		Permission  string  `json:"permission"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		share, err := quizgame.CreateShare(r.Context(), d.Pool, courseCode, kitID, viewer, quizgame.CreateShareInput{
			GranteeType: in.GranteeType,
			GranteeID:   in.GranteeID,
			Permission:  in.Permission,
		})
		if err != nil {
			if strings.Contains(err.Error(), "quizgame:") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create share.")
			return
		}
		if share == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.kit.shared")
		d.auditKitShare(r, viewer, "quizgame.kit.share_created", kitID, kitShareJSON(*share))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(kitShareJSON(*share))
	}
}

// handleDeleteQuizKitShare is DELETE .../kits/{kit_id}/shares/{share_id}.
func (d Deps) handleDeleteQuizKitShare() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		shareID := chi.URLParam(r, "share_id")
		okDel, err := quizgame.DeleteShare(r.Context(), d.Pool, courseCode, kitID, shareID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not revoke share.")
			return
		}
		if !okDel {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Share not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.kit.unshared")
		d.auditKitShare(r, viewer, "quizgame.kit.share_revoked", kitID, map[string]any{"shareId": shareID})
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleQuizLibrarySearch is GET /api/v1/live-quizzes/library.
func (d Deps) handleQuizLibrarySearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		pageSize, _ := strconv.Atoi(q.Get("pageSize"))
		includePublic := d.effectiveConfig().FFIqPublicKitCatalog
		result, err := quizgame.SearchLibrary(r.Context(), d.Pool, quizgame.LibraryOpts{
			Query:                q.Get("q"),
			Subject:              q.Get("subject"),
			GradeBand:            firstNonEmpty(q.Get("grade"), q.Get("gradeBand")),
			Language:             firstNonEmpty(q.Get("lang"), q.Get("language")),
			Tag:                  q.Get("tag"),
			IncludePublicCatalog: includePublic,
			Viewer:               viewer,
			Page:                 page,
			PageSize:             pageSize,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not search library.")
			return
		}
		out := make([]map[string]any, 0, len(result.Kits))
		for _, k := range result.Kits {
			out = append(out, kitJSONFull(k))
		}
		telemetry.RecordBusinessEvent("quizgame.library.view")
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// handleQuizLibraryPreview is GET /api/v1/live-quizzes/library/{kit_id}/preview.
func (d Deps) handleQuizLibraryPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		kit, questions, err := quizgame.PreviewKit(r.Context(), d.Pool, kitID, viewer)
		if err != nil {
			if strings.Contains(err.Error(), "not allowed") {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have access to this kit.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not preview kit.")
			return
		}
		if kit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		qs := make([]map[string]any, 0, len(questions))
		for _, q := range questions {
			qs = append(qs, questionJSON(q))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"kit":       kitJSONFull(*kit),
			"questions": qs,
		})
	}
}

// handleQuizLibraryImport is POST /api/v1/live-quizzes/library/{kit_id}/import.
func (d Deps) handleQuizLibraryImport() http.HandlerFunc {
	type reqBody struct {
		TargetCourseCode string `json:"targetCourseCode"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var in reqBody
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		target := strings.TrimSpace(in.TargetCourseCode)
		if target == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "targetCourseCode is required.")
			return
		}
		if !d.userHasCourseAccess(w, r, target, viewer) {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, target) {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+target+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		created, vr, err := quizgame.ImportFromLibrary(r.Context(), d.Pool, kitID, target, viewer)
		if err != nil {
			if strings.Contains(err.Error(), "not allowed") {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to import this kit.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not import kit.")
			return
		}
		if created == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.library.import")
		out := kitJSONFull(*created)
		if vr != nil {
			out["validation"] = vr
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleSubmitQuizKitToCatalog is POST .../kits/{kit_id}/submit-to-catalog.
func (d Deps) handleSubmitQuizKitToCatalog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		if !d.effectiveConfig().FFIqPublicKitCatalog {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Public kit catalog is not enabled.")
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		updated, err := quizgame.SubmitToCatalog(r.Context(), d.Pool, courseCode, kitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not submit kit to catalog.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz kit not found.")
			return
		}
		if _, qerr := quizgame.EnqueueCatalogSubmission(r.Context(), d.Pool, kitID, viewer, map[string]any{
			"courseCode": courseCode,
			"title":      updated.Title,
		}); qerr != nil {
			telemetry.RecordBusinessEvent("quizgame.catalog.review_enqueue_failed")
		} else {
			telemetry.RecordBusinessEvent("quizgame.catalog.review_enqueued")
		}
		telemetry.RecordBusinessEvent("quizgame.catalog.submit")
		d.auditKitShare(r, viewer, "quizgame.kit.catalog_submit", kitID, kitJSONFull(*updated))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(kitJSONFull(*updated))
	}
}
