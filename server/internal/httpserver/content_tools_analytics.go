package httpserver

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	ctanalytics "github.com/lextures/lextures/server/internal/service/contenttools/analytics"
)

func (d Deps) registerContentToolsAnalyticsRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/analytics", d.handleContentToolsInstanceAnalytics())
	r.Get("/api/v1/courses/{course_code}/content-tools/analytics", d.handleContentToolsItemAnalytics())
	r.Get("/api/v1/courses/{course_code}/content-tools/analytics/course", d.handleContentToolsCourseAnalytics())
	r.Get("/api/v1/courses/{course_code}/content-tools/analytics/export", d.handleContentToolsAnalyticsExport())
	r.Get("/api/v1/courses/{course_code}/content-tools/my-progress", d.handleContentToolsMyProgress())
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/grade-link", d.handleContentToolsGradeLinkGet())
	r.Put("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/grade-link", d.handleContentToolsGradeLinkPut())
	r.Delete("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/grade-link", d.handleContentToolsGradeLinkDelete())
}

func (d Deps) handleContentToolsInstanceAnalytics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		if d.contentToolsAnalyticsRateLimited(w, r, viewer) {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		start := time.Now()
		out, err := d.buildInstanceAnalytics(r.Context(), courseID, instanceID, false)
		ctanalytics.ObserveAggregate("instance", time.Since(start))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load analytics.")
			return
		}
		if out == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (d Deps) handleContentToolsItemAnalytics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		if d.contentToolsAnalyticsRateLimited(w, r, viewer) {
			return
		}
		itemStr := strings.TrimSpace(r.URL.Query().Get("itemId"))
		itemID, err := uuid.Parse(itemStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "itemId is required.")
			return
		}
		start := time.Now()
		cacheKey := ctanalytics.CacheKeyItem(itemID.String())
		if cached, ok := ctanalytics.DefaultCache().Get(cacheKey); ok {
			if ov, ok := cached.(ctmodel.PageToolsOverview); ok {
				ctanalytics.ObserveAggregate("item", time.Since(start))
				writeJSON(w, http.StatusOK, ov)
				return
			}
		}
		instances, err := ctrepo.ListInstances(r.Context(), d.Pool, courseID, &itemID, "", false)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list instances.")
			return
		}
		ov := ctmodel.PageToolsOverview{ItemID: itemID, Instances: []ctmodel.InstanceAnalytics{}}
		for _, inst := range instances {
			ia, err := d.buildInstanceAnalytics(r.Context(), courseID, inst.ID, true)
			if err != nil || ia == nil {
				continue
			}
			ov.Instances = append(ov.Instances, *ia)
			ov.Totals.Instances++
			ov.Totals.Learners += ia.Learners
			ov.Totals.Engaged += ia.Engaged
			ov.Totals.Completed += ia.Completed
		}
		ctanalytics.DefaultCache().Set(cacheKey, ov)
		ctanalytics.ObserveAggregate("item", time.Since(start))
		writeJSON(w, http.StatusOK, ov)
	}
}

func (d Deps) handleContentToolsCourseAnalytics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		if d.contentToolsAnalyticsRateLimited(w, r, viewer) {
			return
		}
		start := time.Now()
		cacheKey := ctanalytics.CacheKeyCourse(courseID.String())
		if cached, ok := ctanalytics.DefaultCache().Get(cacheKey); ok {
			if body, ok := cached.(ctmodel.CourseToolsAnalytics); ok {
				ctanalytics.ObserveAggregate("course", time.Since(start))
				writeJSON(w, http.StatusOK, body)
				return
			}
		}
		instances, err := ctrepo.ListInstances(r.Context(), d.Pool, courseID, nil, "", false)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list instances.")
			return
		}
		// Touch course-wide summaries so rebuild/atrisk plumbing stays live (FR-4 / FR-12).
		if courseSums, err := ctrepo.ListSummariesForCourse(r.Context(), d.Pool, courseID); err == nil {
			ctanalytics.ApplyToolDisengageSignal(ctanalytics.ToAggregateRows(courseSums), func(float32) {})
		}
		body := ctmodel.CourseToolsAnalytics{
			CourseID:  courseID.String(),
			ByTool:    []ctmodel.CourseToolRollup{},
			Instances: []ctmodel.InstanceAnalytics{},
		}
		byTool := map[string]*ctmodel.CourseToolRollup{}
		for _, inst := range instances {
			ia, err := d.buildInstanceAnalytics(r.Context(), courseID, inst.ID, true)
			if err != nil || ia == nil {
				continue
			}
			body.Instances = append(body.Instances, *ia)
			rt, ok := byTool[ia.ToolID]
			if !ok {
				rt = &ctmodel.CourseToolRollup{ToolID: ia.ToolID}
				byTool[ia.ToolID] = rt
			}
			rt.Instances++
			rt.Learners += ia.Learners
			rt.Engaged += ia.Engaged
			rt.Completed += ia.Completed
		}
		for _, rt := range byTool {
			body.ByTool = append(body.ByTool, *rt)
		}
		ctanalytics.DefaultCache().Set(cacheKey, body)
		ctanalytics.ObserveAggregate("course", time.Since(start))
		writeJSON(w, http.StatusOK, body)
	}
}

func (d Deps) handleContentToolsAnalyticsExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		if d.contentToolsExportRateLimited(w, r, viewer) {
			return
		}
		itemStr := strings.TrimSpace(r.URL.Query().Get("itemId"))
		itemID, err := uuid.Parse(itemStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "itemId is required.")
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "json"
		}
		if format != "json" && format != "csv" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "format must be csv or json.")
			return
		}
		summaries, err := ctrepo.ListSummariesForItem(r.Context(), d.Pool, courseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to export analytics.")
			return
		}
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, nil, nil, &viewer, "_export", "analytics_export", map[string]any{
			"itemId": itemID.String(), "format": format, "courseCode": courseCode, "rows": len(summaries),
		})
		if format == "csv" {
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=content-tool-analytics.csv")
			cw := csv.NewWriter(w)
			_ = cw.Write([]string{"enrollment_id", "instance_id", "tool_id", "role", "engaged", "completed", "score_pct", "duration_ms"})
			_ = cw.Write([]string{"enrollmentId", "instanceId", "toolId", "role", "engaged", "completed", "scorePct", "durationMs"})
			for _, s := range summaries {
				score := ""
				if s.ScorePct != nil {
					score = strconv.FormatFloat(*s.ScorePct, 'f', -1, 64)
				}
				dur := ""
				if s.DurationMs != nil {
					dur = strconv.Itoa(*s.DurationMs)
				}
				_ = cw.Write([]string{
					s.EnrollmentID.String(), s.InstanceID.String(), s.ToolID, s.Role,
					strconv.FormatBool(s.Engaged), strconv.FormatBool(s.Completed), score, dur,
				})
			}
			cw.Flush()
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"itemId": itemID, "rows": summaries})
	}
}

func (d Deps) handleContentToolsMyProgress() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		itemStr := strings.TrimSpace(r.URL.Query().Get("itemId"))
		itemID, err := uuid.Parse(itemStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "itemId is required.")
			return
		}
		enrollPtr, err := enrollment.GetActiveEnrollmentID(r.Context(), d.Pool, courseID, viewer)
		if err != nil || enrollPtr == nil {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Active enrollment required.")
			return
		}
		enrollID := *enrollPtr
		_ = courseCode
		instances, err := ctrepo.ListInstances(r.Context(), d.Pool, courseID, &itemID, "", false)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load tools.")
			return
		}
		ids := make([]uuid.UUID, 0, len(instances))
		for _, inst := range instances {
			ids = append(ids, inst.ID)
		}
		sums, err := ctrepo.ListSummariesForEnrollment(r.Context(), d.Pool, enrollID, ids)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load progress.")
			return
		}
		byInst := map[uuid.UUID]ctrepo.StateSummaryRow{}
		for _, s := range sums {
			byInst[s.InstanceID] = s
		}
		links, _ := ctrepo.ListGradeLinksForInstances(r.Context(), d.Pool, ids)
		out := ctmodel.StudentToolProgress{ItemID: itemID, Tools: []ctmodel.StudentToolProgressRow{}}
		for _, inst := range instances {
			row := ctmodel.StudentToolProgressRow{
				InstanceID: inst.ID,
				ToolID:     inst.ToolID,
				Title:      inst.Title,
			}
			if s, ok := byInst[inst.ID]; ok {
				row.Engaged = s.Engaged
				row.Completed = s.Completed
				row.ScorePct = s.ScorePct
			}
			if link, ok := links[inst.ID]; ok && link.CountsForGrade {
				row.CountsForGrade = true
			}
			out.Tools = append(out.Tools, row)
			out.Total++
			if row.Completed {
				out.Completed++
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (d Deps) handleContentToolsGradeLinkGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Course members (including students) may read the link so FR-9 can show
		// the "counts for a grade" badge before interaction. Mutations stay gated.
		_, _, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		link, err := ctrepo.GetGradeLink(r.Context(), d.Pool, instanceID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grade link.")
			return
		}
		if link == nil {
			writeJSON(w, http.StatusOK, ctmodel.GradeLink{InstanceID: instanceID, LatePolicy: "accept"})
			return
		}
		writeJSON(w, http.StatusOK, gradeLinkToAPI(link))
	}
}

func (d Deps) handleContentToolsGradeLinkPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		settings, _ := ctrepo.GetSettings(r.Context(), d.Pool, courseID)
		if settings != nil && !settings.GradeLinksAllowed {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Grade links are disabled for this course.")
			return
		}
		var body ctmodel.GradeLinkPutRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		policy := strings.TrimSpace(body.LatePolicy)
		if policy == "" {
			policy = "accept"
		}
		if policy != "accept" && policy != "accept_marked" && policy != "reject" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid latePolicy.")
			return
		}
		if body.AssignmentItemID != nil {
			okItem, err := ctrepo.StructureItemInCourse(r.Context(), d.Pool, courseID, *body.AssignmentItemID)
			if err != nil || !okItem {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "assignmentItemId must belong to this course.")
				return
			}
		}
		link, err := ctrepo.UpsertGradeLink(r.Context(), d.Pool, ctrepo.GradeLinkRow{
			InstanceID:       instanceID,
			AssignmentItemID: body.AssignmentItemID,
			OutcomeID:        body.OutcomeID,
			PointsPossible:   body.PointsPossible,
			CountsForGrade:   body.CountsForGrade,
			LatePolicy:       policy,
			EnabledBy:        &viewer,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save grade link.")
			return
		}
		ctanalytics.InvalidateForInstance(instanceID.String(), courseID.String(), "")
		writeJSON(w, http.StatusOK, gradeLinkToAPI(link))
	}
}

func (d Deps) handleContentToolsGradeLinkDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		if err := ctrepo.DeleteGradeLink(r.Context(), d.Pool, instanceID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete grade link.")
			return
		}
		ctanalytics.InvalidateForInstance(instanceID.String(), courseID.String(), "")
		w.WriteHeader(http.StatusNoContent)
	}
}

func gradeLinkToAPI(link *ctrepo.GradeLinkRow) ctmodel.GradeLink {
	if link == nil {
		return ctmodel.GradeLink{LatePolicy: "accept"}
	}
	en := link.EnabledAt
	return ctmodel.GradeLink{
		InstanceID:       link.InstanceID,
		AssignmentItemID: link.AssignmentItemID,
		OutcomeID:        link.OutcomeID,
		PointsPossible:   link.PointsPossible,
		CountsForGrade:   link.CountsForGrade,
		LatePolicy:       link.LatePolicy,
		EnabledAt:        &en,
	}
}
