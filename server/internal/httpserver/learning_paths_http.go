package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoPaths "github.com/lextures/lextures/server/internal/repos/learningpaths"
	svcPaths "github.com/lextures/lextures/server/internal/service/learningpaths"
)

func (d Deps) learningPathsFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFLearningPaths {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Learning paths are not enabled.")
		return true
	}
	return false
}

type learningPathJSON struct {
	ID               string  `json:"id"`
	CreatorID        string  `json:"creatorId"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	Slug             *string `json:"slug,omitempty"`
	BundlePriceCents *int    `json:"bundlePriceCents,omitempty"`
	StripeProductID  *string `json:"stripeProductId,omitempty"`
	IsPublic         bool    `json:"isPublic"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
	CourseIDs        []string `json:"courseIds,omitempty"`
}

func pathToJSON(p *repoPaths.Path, courseIDs []uuid.UUID) learningPathJSON {
	out := learningPathJSON{
		ID:               p.ID.String(),
		CreatorID:        p.CreatorID.String(),
		Title:            p.Title,
		Description:      p.Description,
		Slug:             p.Slug,
		BundlePriceCents: p.BundlePriceCents,
		StripeProductID:  p.StripeProductID,
		IsPublic:         p.IsPublic,
		CreatedAt:        p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        p.UpdatedAt.UTC().Format(time.RFC3339),
	}
	for _, id := range courseIDs {
		out.CourseIDs = append(out.CourseIDs, id.String())
	}
	return out
}

type pathCourseJSON struct {
	CourseID        string   `json:"courseId"`
	Position        int      `json:"position"`
	CourseCode      string   `json:"courseCode"`
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	ListPriceCents  *int     `json:"listPriceCents,omitempty"`
	DurationMinutes int      `json:"durationMinutes"`
	SkillTags       []string `json:"skillTags"`
	Completed       bool     `json:"completed,omitempty"`
	Recommended     bool     `json:"recommended,omitempty"`
}

type catalogPathSummaryJSON struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Slug             string   `json:"slug"`
	BundlePriceCents *int     `json:"bundlePriceCents,omitempty"`
	CourseCount      int      `json:"courseCount"`
	TotalDurationMin int      `json:"totalDurationMinutes"`
	IndividualTotal  int      `json:"individualTotalCents"`
	SkillTags        []string `json:"skillTags"`
}

type pathProgressJSON struct {
	PathID           string           `json:"pathId"`
	PathTitle        string           `json:"pathTitle"`
	Slug             string           `json:"slug,omitempty"`
	TotalCourses     int              `json:"totalCourses"`
	CompletedCourses int              `json:"completedCourses"`
	Percent          int              `json:"percent"`
	ProgressLabel    string           `json:"progressLabel"`
	CompletedAt      *string          `json:"completedAt,omitempty"`
	JustCompleted    bool             `json:"justCompleted"`
	Courses          []pathCourseJSON `json:"courses"`
}

func (d Deps) registerLearningPathRoutes(r chi.Router) {
	r.Get("/api/v1/catalog/paths", d.handleCatalogPathsList())
	r.Get("/api/v1/catalog/paths/{slug}", d.handleCatalogPathDetail())

	r.Get("/api/v1/creator/learning-paths", d.handleCreatorListPaths())
	r.Post("/api/v1/creator/learning-paths", d.handleCreatorCreatePath())
	r.Patch("/api/v1/creator/learning-paths/{id}", d.handleCreatorPatchPath())
	r.Delete("/api/v1/creator/learning-paths/{id}", d.handleCreatorDeletePath())

	r.Get("/api/v1/me/paths", d.handleMeListPaths())
	r.Get("/api/v1/me/paths/{id}/progress", d.handleMePathProgress())
	r.Post("/api/v1/paths/{id}/enroll", d.handlePathEnroll())
}

func (d Deps) handleCatalogPathsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.learningPathsFeatureOff(w) {
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		sort := strings.TrimSpace(r.URL.Query().Get("sort"))
		rows, err := repoPaths.ListPublicPaths(r.Context(), d.Pool, q, sort)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load paths.")
			return
		}
		out := make([]catalogPathSummaryJSON, 0, len(rows))
		for _, row := range rows {
			out = append(out, catalogPathSummaryJSON{
				ID:               row.ID.String(),
				Title:            row.Title,
				Description:      row.Description,
				Slug:             row.Slug,
				BundlePriceCents: row.BundlePriceCents,
				CourseCount:      row.CourseCount,
				TotalDurationMin: row.TotalDurationMin,
				IndividualTotal:  row.IndividualTotal,
				SkillTags:        row.SkillTags,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"paths": out})
	}
}

func (d Deps) handleCatalogPathDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.learningPathsFeatureOff(w) {
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		if slug == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing path slug.")
			return
		}
		detail, err := repoPaths.GetCatalogDetail(r.Context(), d.Pool, slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load path.")
			return
		}
		if detail == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Path not found.")
			return
		}
		courses := make([]pathCourseJSON, 0, len(detail.Courses))
		for _, c := range detail.Courses {
			courses = append(courses, pathCourseJSON{
				CourseID:        c.CourseID.String(),
				Position:        c.Position,
				CourseCode:      c.CourseCode,
				Title:           c.Title,
				Description:     c.Description,
				ListPriceCents:  c.ListPriceCents,
				DurationMinutes: c.DurationMinutes,
				SkillTags:       c.SkillTags,
			})
		}
		slugVal := ""
		if detail.Slug != nil {
			slugVal = *detail.Slug
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"path": pathToJSON(&detail.Path, nil),
			"courses": courses,
			"totalDurationMinutes": detail.TotalDurationMin,
			"individualTotalCents": detail.IndividualTotal,
			"skillTags":            detail.SkillTags,
			"slug":                 slugVal,
		})
	}
}

func (d Deps) handleCreatorListPaths() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.learningPathsFeatureOff(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		paths, err := repoPaths.ListPathsByCreator(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load paths.")
			return
		}
		out := make([]learningPathJSON, 0, len(paths))
		for i := range paths {
			courses, err := repoPaths.ListPathCourses(r.Context(), d.Pool, paths[i].ID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load path courses.")
				return
			}
			ids := make([]uuid.UUID, len(courses))
			for j, c := range courses {
				ids[j] = c.CourseID
			}
			out = append(out, pathToJSON(&paths[i], ids))
		}
		writeJSON(w, http.StatusOK, map[string]any{"paths": out})
	}
}

func (d Deps) handleCreatorCreatePath() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.learningPathsFeatureOff(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			Title            string   `json:"title"`
			Description      string   `json:"description"`
			CourseIDs        []string `json:"courseIds"`
			BundlePriceCents *int     `json:"bundlePriceCents"`
			IsPublic         bool     `json:"isPublic"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		title := strings.TrimSpace(body.Title)
		if title == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "title is required.")
			return
		}
		courseIDs, err := parseUUIDList(body.CourseIDs)
		if err != nil || len(courseIDs) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseIds must include at least one valid course id.")
			return
		}
		if err := svcPaths.ValidateCreatorCourses(r.Context(), d.Pool, uid, courseIDs); err != nil {
			if errors.Is(err, svcPaths.ErrCourseNotOwned) {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You may only add courses you teach to a learning path.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify courses.")
			return
		}
		p, err := repoPaths.CreatePath(r.Context(), d.Pool, uid, title, strings.TrimSpace(body.Description), courseIDs, body.BundlePriceCents, body.IsPublic)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, pathToJSON(p, courseIDs))
	}
}

func (d Deps) handleCreatorPatchPath() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.learningPathsFeatureOff(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pathID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid path id.")
			return
		}
		var body struct {
			Title            *string  `json:"title"`
			Description      *string  `json:"description"`
			CourseIDs        []string `json:"courseIds"`
			BundlePriceCents *int     `json:"bundlePriceCents"`
			IsPublic         *bool    `json:"isPublic"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var courseIDs []uuid.UUID
		if body.CourseIDs != nil {
			courseIDs, err = parseUUIDList(body.CourseIDs)
			if err != nil || len(courseIDs) == 0 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseIds must include at least one valid course id.")
				return
			}
			if err := svcPaths.ValidateCreatorCourses(r.Context(), d.Pool, uid, courseIDs); err != nil {
				if errors.Is(err, svcPaths.ErrCourseNotOwned) {
					apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You may only add courses you teach to a learning path.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify courses.")
				return
			}
		}
		bundleSet := body.BundlePriceCents != nil || r.Header.Get("X-Bundle-Price-Clear") == "1"
		p, err := repoPaths.UpdatePath(r.Context(), d.Pool, pathID, uid, body.Title, body.Description, body.BundlePriceCents, bundleSet, body.IsPublic, courseIDs)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if p == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Path not found.")
			return
		}
		courses, _ := repoPaths.ListPathCourses(r.Context(), d.Pool, p.ID)
		ids := make([]uuid.UUID, len(courses))
		for i, c := range courses {
			ids[i] = c.CourseID
		}
		writeJSON(w, http.StatusOK, pathToJSON(p, ids))
	}
}

func (d Deps) handleCreatorDeletePath() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.learningPathsFeatureOff(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pathID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid path id.")
			return
		}
		okDel, err := repoPaths.DeletePath(r.Context(), d.Pool, pathID, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete path.")
			return
		}
		if !okDel {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Path not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleMeListPaths() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.learningPathsFeatureOff(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		enrollments, err := repoPaths.ListPathEnrollmentsByUser(r.Context(), d.Pool, uid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load path enrollments.")
			return
		}
		out := make([]pathProgressJSON, 0, len(enrollments))
		for _, e := range enrollments {
			prog, err := svcPaths.GetProgress(r.Context(), d.Pool, uid, e.PathID)
			if err != nil || prog == nil {
				continue
			}
			out = append(out, progressToJSON(prog))
		}
		writeJSON(w, http.StatusOK, map[string]any{"paths": out})
	}
}

func (d Deps) handleMePathProgress() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.learningPathsFeatureOff(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pathID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid path id.")
			return
		}
		prog, err := svcPaths.GetProgress(r.Context(), d.Pool, uid, pathID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load path progress.")
			return
		}
		writeJSON(w, http.StatusOK, progressToJSON(prog))
	}
}

func (d Deps) handlePathEnroll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.learningPathsFeatureOff(w) {
			return
		}
		uid, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		pathID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid path id.")
			return
		}
		enrollment, err := svcPaths.EnrollUser(r.Context(), d.Pool, pathID, uid)
		if err != nil {
			switch {
			case errors.Is(err, svcPaths.ErrPathNotFound):
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Path not found.")
			case errors.Is(err, svcPaths.ErrAlreadyEnrolled):
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Already enrolled in this path.")
			case errors.Is(err, svcPaths.ErrEntitlementRequired):
				apierr.WriteJSON(w, http.StatusPaymentRequired, apierr.CodeForbidden, "Purchase this path bundle before enrolling.")
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enroll in path.")
			}
			return
		}
		prog, _ := svcPaths.GetProgress(r.Context(), d.Pool, uid, pathID)
		writeJSON(w, http.StatusCreated, map[string]any{
			"enrollmentId": enrollment.ID.String(),
			"progress":     progressToJSON(prog),
		})
	}
}

func progressToJSON(prog *svcPaths.PathProgress) pathProgressJSON {
	if prog == nil {
		return pathProgressJSON{}
	}
	label := formatPathProgressLabel(prog.CompletedCourses, prog.TotalCourses, prog.Percent)
	out := pathProgressJSON{
		PathID:           prog.PathID.String(),
		PathTitle:        prog.PathTitle,
		Slug:             prog.Slug,
		TotalCourses:     prog.TotalCourses,
		CompletedCourses: prog.CompletedCourses,
		Percent:          prog.Percent,
		ProgressLabel:    label,
		JustCompleted:    prog.JustCompleted,
	}
	if prog.CompletedAt != nil {
		s := prog.CompletedAt.UTC().Format(time.RFC3339)
		out.CompletedAt = &s
	}
	for _, c := range prog.Courses {
		out.Courses = append(out.Courses, pathCourseJSON{
			CourseID:    c.CourseID.String(),
			Position:    c.Position,
			CourseCode:  c.CourseCode,
			Title:       c.Title,
			Completed:   c.Completed,
			Recommended: c.Recommended,
		})
	}
	return out
}

func formatPathProgressLabel(completed, total, percent int) string {
	if total == 0 || completed == 0 {
		return "0% — Start Your Journey"
	}
	return fmt.Sprintf("%d%% — %d of %d complete", percent, completed, total)
}
