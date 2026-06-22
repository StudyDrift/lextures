package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/gradecurve"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/gradeauditevents"
	"github.com/lextures/lextures/server/internal/repos/gradecurves"
	webhooksvc "github.com/lextures/lextures/server/internal/service/webhooks"
	"github.com/lextures/lextures/server/internal/webhooks"
)

type gradeCurveRequestBody struct {
	Method        string          `json:"method"`
	Params        json.RawMessage `json:"params"`
	AllowAboveMax bool            `json:"allowAboveMax"`
}

func (d Deps) requireGradeCurvingEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFGradeCurving {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Grade curving is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireGradeCurveAccess(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, courseID uuid.UUID, itemID uuid.UUID, ok bool) {
	if !d.requireGradeCurvingEnabled(w) {
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to curve grades.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	itemID, err = uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment ID.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return courseCode, viewer, *cid, itemID, true
}

func (d Deps) assignmentMaxPoints(ctx context.Context, courseID, itemID uuid.UUID) (float64, error) {
	row, err := coursemoduleassignments.GetForCourseItem(ctx, d.Pool, courseID, itemID)
	if err != nil {
		return 0, err
	}
	if row != nil {
		max := 100.0
		if row.PointsWorth != nil && *row.PointsWorth > 0 {
			max = float64(*row.PointsWorth)
		}
		return max, nil
	}
	var pts *int32
	err = d.Pool.QueryRow(ctx, `
SELECT m.points_worth
FROM course.module_quizzes m
WHERE m.structure_item_id = $1
`, itemID).Scan(&pts)
	if errors.Is(err, pgx.ErrNoRows) {
		return 100, nil
	}
	if err != nil {
		return 0, err
	}
	if pts != nil && *pts > 0 {
		return float64(*pts), nil
	}
	return 100, nil
}

func (d Deps) loadCurvePreviewInputs(ctx context.Context, courseID, itemID uuid.UUID) ([]gradecurve.ScoreInput, float64, error) {
	cells, err := gradecurves.ListCellsForAssignment(ctx, d.Pool, courseID, itemID)
	if err != nil {
		return nil, 0, err
	}
	_, rawByStudent, err := gradecurves.RawScoresForActiveCurve(ctx, d.Pool, itemID)
	if err != nil {
		return nil, 0, err
	}
	maxPts, err := d.assignmentMaxPoints(ctx, courseID, itemID)
	if err != nil {
		return nil, 0, err
	}
	return gradecurves.BuildScoreInputs(cells, rawByStudent), maxPts, nil
}

func (d Deps) parseCurveRequest(w http.ResponseWriter, r *http.Request, maxPts float64) (gradecurve.Options, error) {
	var body gradeCurveRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
		return gradecurve.Options{}, err
	}
	params, err := gradecurve.ParseParams(body.Params)
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid curve params.")
		return gradecurve.Options{}, err
	}
	opts := gradecurve.Options{
		MaxPoints:     maxPts,
		AllowAboveMax: body.AllowAboveMax,
		Method:        gradecurve.Method(body.Method),
		Params:        params,
	}
	if err := gradecurve.ValidateMethod(opts.Method, opts.Params); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
		return gradecurve.Options{}, err
	}
	return opts, nil
}

// handlePostAssignmentCurvePreview is POST .../assignments/{item_id}/curve/preview.
func (d Deps) handlePostAssignmentCurvePreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, _, courseID, itemID, ok := d.requireGradeCurveAccess(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		inputs, maxPts, err := d.loadCurvePreviewInputs(ctx, courseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grades.")
			return
		}
		opts, err := d.parseCurveRequest(w, r, maxPts)
		if err != nil {
			return
		}
		preview, err := gradecurve.Preview(inputs, opts)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		active, _ := gradecurves.GetActiveForAssignment(ctx, d.Pool, itemID)
		type resp struct {
			Preview      gradecurve.PreviewSummary `json:"preview"`
			ActiveCurve  *string                   `json:"activeCurveId,omitempty"`
			MaxPoints    float64                   `json:"maxPoints"`
		}
		out := resp{Preview: preview, MaxPoints: maxPts}
		if active != nil {
			s := active.ID.String()
			out.ActiveCurve = &s
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handlePostAssignmentCurve is POST .../assignments/{item_id}/curve.
func (d Deps) handlePostAssignmentCurve() http.HandlerFunc {
	type resp struct {
		CurveID string `json:"curveId"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, courseID, itemID, ok := d.requireGradeCurveAccess(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		inputs, maxPts, err := d.loadCurvePreviewInputs(ctx, courseID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grades.")
			return
		}
		opts, err := d.parseCurveRequest(w, r, maxPts)
		if err != nil {
			return
		}
		preview, err := gradecurve.Preview(inputs, opts)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if preview.EligibleCount == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No graded, non-excused submissions to curve.")
			return
		}

		paramsJSON, _ := json.Marshal(opts.Params)
		adjs := make([]gradecurves.AdjustmentRow, 0, len(preview.Results))
		for _, res := range preview.Results {
			adjs = append(adjs, gradecurves.AdjustmentRow{
				StudentID:     res.StudentID,
				RawScore:      res.RawScore,
				AdjustedScore: res.AdjustedScore,
			})
		}

		curveID, err := gradecurves.Apply(ctx, d.Pool, gradecurves.ApplyInput{
			CourseID:      courseID,
			ModuleItemID:  itemID,
			Method:        opts.Method,
			ParamsJSON:    paramsJSON,
			AllowAboveMax: opts.AllowAboveMax,
			AppliedBy:     viewer,
			Adjustments:   adjs,
		}, func(ctx context.Context, tx pgx.Tx, studentID uuid.UUID, points float64) error {
			_, err := tx.Exec(ctx, `
UPDATE course.course_grades
SET points_earned = $1, updated_at = NOW()
WHERE course_id = $2 AND student_user_id = $3 AND module_item_id = $4
`, points, courseID, studentID, itemID)
			return err
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to apply curve.")
			return
		}

		reason := gradecurve.AuditReasonJSON(opts.Method, opts.Params, curveID)
		changedBy := viewer
		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to write audit trail.")
			return
		}
		for _, res := range preview.Results {
			if !res.Changed {
				continue
			}
			prev := res.RawScore
			next := res.AdjustedScore
			if err := gradeauditevents.Insert(ctx, tx, courseID, itemID, res.StudentID, &changedBy, "curved", &prev, &next, nil, nil, &reason); err != nil {
				_ = tx.Rollback(ctx)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to write audit trail.")
				return
			}
		}
		if err := tx.Commit(ctx); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to write audit trail.")
			return
		}

		cfg := d.effectiveConfig()
		var orgID uuid.UUID
		if err := d.Pool.QueryRow(ctx, `SELECT org_id FROM course.courses WHERE id = $1`, courseID).Scan(&orgID); err == nil && orgID != uuid.Nil {
			webhooksvc.EmitAsync(d.Pool, cfg, orgID, webhooks.EventGradeCurveApplied, webhooksvc.GradeCurveAppliedData{
				CourseID:     courseID.String(),
				CourseCode:   courseCode,
				ModuleItemID: itemID.String(),
				CurveID:      curveID.String(),
				Method:       string(opts.Method),
				AppliedBy:    viewer.String(),
				AppliedAt:    time.Now().UTC().Format(time.RFC3339),
				Affected:     len(preview.Results),
			})
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{CurveID: curveID.String()})
	}
}

// handleDeleteGradeCurve is DELETE /api/v1/curves/{curve_id}.
func (d Deps) handleDeleteGradeCurve() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireGradeCurvingEnabled(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		curveID, err := uuid.Parse(chi.URLParam(r, "curve_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid curve ID.")
			return
		}
		ctx := r.Context()
		curve, err := gradecurves.GetByID(ctx, d.Pool, curveID)
		if err != nil || curve == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Curve not found.")
			return
		}
		courseCodePtr, err := course.GetCourseCodeByID(ctx, d.Pool, curve.CourseID)
		if err != nil || courseCodePtr == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		has, err := courseroles.UserHasPermission(ctx, d.Pool, viewer, "course:"+*courseCodePtr+":item:create")
		if err != nil || !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to revert curves.")
			return
		}
		if curve.ModuleItemID == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Cannot revert this curve type.")
			return
		}
		itemID := *curve.ModuleItemID
		adjs, err := gradecurves.ListAdjustmentsForCurve(ctx, d.Pool, curveID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load curve adjustments.")
			return
		}

		if err := gradecurves.Revert(ctx, d.Pool, curveID, func(ctx context.Context, tx pgx.Tx, studentID uuid.UUID, points float64) error {
			_, err := tx.Exec(ctx, `
UPDATE course.course_grades
SET points_earned = $1, updated_at = NOW()
WHERE course_id = $2 AND student_user_id = $3 AND module_item_id = $4
`, points, curve.CourseID, studentID, itemID)
			return err
		}); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to revert curve.")
			return
		}

		reason := gradecurve.AuditReasonJSON(curve.Method, gradecurve.Params{}, curveID)
		changedBy := viewer
		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to write audit trail.")
			return
		}
		for _, adj := range adjs {
			prev := adj.AdjustedScore
			next := adj.RawScore
			if err := gradeauditevents.Insert(ctx, tx, curve.CourseID, itemID, adj.StudentID, &changedBy, "curve_reverted", &prev, &next, nil, nil, &reason); err != nil {
				_ = tx.Rollback(ctx)
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to write audit trail.")
				return
			}
		}
		if err := tx.Commit(ctx); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to write audit trail.")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
