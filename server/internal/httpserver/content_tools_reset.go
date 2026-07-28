package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/config"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	"github.com/lextures/lextures/server/internal/ratelimit"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func (d Deps) handleContentToolsStateReset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		var body ctmodel.ResetRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		limit := 20
		if body.DryRun {
			limit = 60
		}
		if !d.contentToolsRateLimit(w, r, viewer, "ct_reset", limit) {
			return
		}
		if _, ok := ctsvc.ValidResetScopes[body.Scope]; !ok {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid reset scope.")
			return
		}
		if body.IdempotencyKey != nil && *body.IdempotencyKey != "" && !body.DryRun {
			if existing, err := ctrepo.GetResetJobByIdempotency(r.Context(), d.Pool, *body.IdempotencyKey); err == nil && existing != nil {
				writeResetJobAccepted(w, existing)
				return
			}
		}
		taSections, err := d.contentToolsTASectionFilter(r.Context(), courseID, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve section scope.")
			return
		}
		notify := true
		if body.Notify != nil {
			notify = *body.Notify
		}
		orgID, _ := course.CourseOrgID(r.Context(), d.Pool, courseCode)
		org := uuid.Nil
		if orgID != nil {
			org = *orgID
		}
		toolID := ""
		activityTitle := ""
		if body.InstanceID != nil {
			inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, *body.InstanceID)
			if err != nil || inst == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
				return
			}
			toolID = inst.ToolID
			if inst.Title != nil {
				activityTitle = *inst.Title
			}
		}
		req := ctsvc.ResetRequest{
			CourseID:       courseID,
			CourseCode:     courseCode,
			OrgID:          org,
			ActorID:        viewer,
			ActorRole:      "instructor",
			Scope:          body.Scope,
			InstanceID:     body.InstanceID,
			ItemID:         body.ItemID,
			EnrollmentID:   body.EnrollmentID,
			SectionIDs:     body.SectionIDs,
			TASectionIDs:   taSections,
			Reason:         body.Reason,
			Notify:         notify,
			DryRun:         body.DryRun,
			IdempotencyKey: body.IdempotencyKey,
			ToolID:         toolID,
			ActivityTitle:  activityTitle,
		}
		if body.PostHandling != nil {
			req.PostHandling = strings.TrimSpace(*body.PostHandling)
		}
		if body.SchedulingHandling != nil {
			req.SchedulingHandling = strings.TrimSpace(*body.SchedulingHandling)
		}
		if len(taSections) > 0 {
			req.ActorRole = "ta"
		}
		result, err := ctsvc.ExecuteReset(r.Context(), d.Pool, req)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if result.Async && !body.DryRun {
			target, _ := json.Marshal(map[string]any{
				"instanceId":   body.InstanceID,
				"itemId":       body.ItemID,
				"enrollmentId": body.EnrollmentID,
				"sectionIds":   taSections,
			})
			job, reused, err := ctrepo.InsertResetJob(
				r.Context(), d.Pool, courseID, viewer, body.Scope, target, body.Reason, notify, body.IdempotencyKey, result.AffectedCount,
			)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enqueue reset job.")
				return
			}
			if !reused {
				reqCopy := req
				jobID := job.ID
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
					defer cancel()
					if err := ctsvc.RunAsyncResetJob(ctx, d.Pool, jobID, reqCopy); err != nil {
						slog.Warn("content_tools.reset_job", "job_id", jobID.String(), "err", err)
					}
				}()
			}
			writeResetJobAccepted(w, job)
			return
		}
		writeResetResponse(w, http.StatusOK, result)
	}
}

func writeResetResponse(w http.ResponseWriter, status int, result *ctsvc.ResetResult) {
	sample := make([]ctmodel.ResetSampleLearner, 0, len(result.Sample))
	for _, s := range result.Sample {
		sample = append(sample, ctmodel.ResetSampleLearner{
			EnrollmentID: s.EnrollmentID,
			DisplayName:  s.DisplayName,
			Status:       s.Status,
			Score:        s.Score,
		})
	}
	effects := make([]ctmodel.GradeEffect, 0, len(result.GradeEffects))
	for _, g := range result.GradeEffects {
		effects = append(effects, ctmodel.GradeEffect{
			EnrollmentID: g.EnrollmentID,
			Action:       g.Action,
			Reason:       g.Reason,
		})
	}
	resp := ctmodel.ResetResponse{
		DryRun:          result.DryRun,
		AffectedCount:   result.AffectedCount,
		Sample:          sample,
		BatchID:         result.BatchID,
		JobID:           result.JobID,
		GradeEffects:    effects,
		ScopeNarrowed:   result.ScopeNarrowed,
		AppliedSections: result.AppliedSections,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func writeResetJobAccepted(w http.ResponseWriter, job *ctrepo.ResetJobRow) {
	id := job.ID
	resp := ctmodel.ResetResponse{
		DryRun:        false,
		AffectedCount: job.TotalRows,
		Sample:        []ctmodel.ResetSampleLearner{},
		JobID:         &id,
		GradeEffects:  []ctmodel.GradeEffect{},
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Location", "/api/v1/courses/_/content-tools/reset-jobs/"+job.ID.String())
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

func (d Deps) handleContentToolsStateResetsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		var instanceID, enrollmentID *uuid.UUID
		if s := strings.TrimSpace(r.URL.Query().Get("instanceId")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instanceId.")
				return
			}
			instanceID = &id
		}
		if s := strings.TrimSpace(r.URL.Query().Get("enrollmentId")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollmentId.")
				return
			}
			enrollmentID = &id
		}
		rows, err := ctrepo.ListStateResets(r.Context(), d.Pool, courseID, instanceID, enrollmentID, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list resets.")
			return
		}
		items := make([]ctmodel.StateResetSnapshot, 0, len(rows))
		for _, row := range rows {
			items = append(items, ctmodel.StateResetSnapshot{
				ID:            row.ID,
				InstanceID:    row.InstanceID,
				EnrollmentID:  row.EnrollmentID,
				ToolID:        row.ToolID,
				Scope:         row.Scope,
				Reason:        row.Reason,
				BatchID:       row.BatchID,
				ResetBy:       row.ResetBy,
				ResetAt:       row.ResetAt,
				RestoredAt:    row.RestoredAt,
				PurgeAfter:    row.PurgeAfter,
				PriorStatus:   row.PriorStatus,
				PriorRevision: row.PriorRevision,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	}
}

func (d Deps) handleContentToolsStateResetRestore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		if !d.contentToolsRateLimit(w, r, viewer, "ct_reset", 20) {
			return
		}
		resetID, err := uuid.Parse(chi.URLParam(r, "reset_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid reset id.")
			return
		}
		snap, err := ctrepo.GetStateReset(r.Context(), d.Pool, resetID)
		if err != nil || snap == nil || snap.CourseID != courseID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Reset snapshot not found.")
			return
		}
		orgID, _ := course.CourseOrgID(r.Context(), d.Pool, courseCode)
		org := uuid.Nil
		if orgID != nil {
			org = *orgID
		}
		st, restored, err := ctsvc.RestoreReset(r.Context(), d.Pool, org, courseID, viewer, resetID)
		if err != nil {
			switch err.Error() {
			case "already_restored":
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "This snapshot was already restored.")
			case "expired":
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "This snapshot has expired and cannot be restored.")
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to restore reset.")
			}
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"reset": ctmodel.StateResetSnapshot{
				ID:            restored.ID,
				InstanceID:    restored.InstanceID,
				EnrollmentID:  restored.EnrollmentID,
				ToolID:        restored.ToolID,
				Scope:         restored.Scope,
				Reason:        restored.Reason,
				BatchID:       restored.BatchID,
				ResetBy:       restored.ResetBy,
				ResetAt:       restored.ResetAt,
				RestoredAt:    restored.RestoredAt,
				PurgeAfter:    restored.PurgeAfter,
				PriorStatus:   restored.PriorStatus,
				PriorRevision: restored.PriorRevision,
			},
			"state": contentToolsStateEnvelope(restored.InstanceID, st),
		})
	}
}

func (d Deps) handleContentToolsResetJobGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		jobID, err := uuid.Parse(chi.URLParam(r, "job_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid job id.")
			return
		}
		job, err := ctrepo.GetResetJob(r.Context(), d.Pool, jobID)
		if err != nil || job == nil || job.CourseID != courseID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Reset job not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(ctmodel.ResetJobStatus{
			ID:            job.ID,
			Status:        job.Status,
			Scope:         job.Scope,
			TotalRows:     job.TotalRows,
			ProcessedRows: job.ProcessedRows,
			BatchID:       job.BatchID,
			Error:         job.Error,
			Result:        job.ResultJSON,
			CreatedAt:     job.CreatedAt,
			FinishedAt:    job.FinishedAt,
		})
	}
}

func (d Deps) handleContentToolsSelfReset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		if !d.contentToolsRateLimit(w, r, viewer, "ct_reset", 20) {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		settings, err := ctrepo.GetSettings(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load settings.")
			return
		}
		if settings == nil || !settings.StudentResetAllowed {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Student reset is not allowed for this course.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		m := ctsvc.MustDefault().Get(inst.ToolID)
		if m == nil || !m.AllowsSelfReset {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This tool does not allow student self-reset.")
			return
		}
		enrIDPtr, err := enrollment.GetActiveEnrollmentID(r.Context(), d.Pool, courseID, viewer)
		if err != nil || enrIDPtr == nil {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "No active enrollment.")
			return
		}
		orgID, _ := course.CourseOrgID(r.Context(), d.Pool, courseCode)
		org := uuid.Nil
		if orgID != nil {
			org = *orgID
		}
		enrID := *enrIDPtr
		instID := instanceID
		notify := false
		req := ctsvc.ResetRequest{
			CourseID:      courseID,
			CourseCode:    courseCode,
			OrgID:         org,
			ActorID:       viewer,
			ActorRole:     "student",
			Scope:         ctrepo.ResetScopeSelf,
			InstanceID:    &instID,
			EnrollmentID:  &enrID,
			Notify:        notify,
			DryRun:        false,
			InitialState:  ctsvc.InitialStateForTool(m),
			ToolID:        inst.ToolID,
			ActivityTitle: "",
		}
		if inst.Title != nil {
			req.ActivityTitle = *inst.Title
		}
		result, err := ctsvc.ExecuteReset(r.Context(), d.Pool, req)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		writeResetResponse(w, http.StatusOK, result)
	}
}

func (d Deps) contentToolsRateLimit(w http.ResponseWriter, r *http.Request, viewer uuid.UUID, bucket string, perMin int) bool {
	limiter := d.buildRateLimiter()
	rule := config.RateLimitRule{Limit: perMin, Window: time.Minute}
	key := limiter.UserKey(viewer.String(), bucket)
	dec := limiter.Allow(r.Context(), key, rule, ratelimit.LimitTypeToken)
	if dec.Allowed {
		return true
	}
	ratelimit.RecordExceeded("content_tool_"+bucket, ratelimit.LimitTypeToken)
	w.Header().Set("Retry-After", strconv.Itoa(dec.RetryAfter))
	apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Rate limit exceeded.")
	return false
}

func enrollmentInSections(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, sectionIDs []uuid.UUID) (bool, error) {
	if len(sectionIDs) == 0 {
		return true, nil
	}
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM course.course_enrollments
  WHERE id = $1 AND section_id = ANY($2::uuid[])
)
`, enrollmentID, sectionIDs).Scan(&ok)
	return ok, err
}
