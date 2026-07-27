package analytics

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

// SyncSummaryFromState projects and persists a summary for one enrollment-scoped state (FR-2).
// Preview-scope states are ignored. Staff roles are stored but filtered at aggregate time (FR-5).
func SyncSummaryFromState(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	toolID string,
	st *ctrepo.StateRow,
) error {
	if pool == nil || st == nil {
		return nil
	}
	if st.Scope != "" && st.Scope != ctrepo.ScopeEnrollment {
		return nil
	}
	role, err := ctrepo.EnrollmentRole(ctx, pool, st.EnrollmentID)
	if err != nil {
		return err
	}
	if role == "" {
		role = "student"
	}
	sum := Project(ProjectInput{
		ToolID:            toolID,
		StateJSON:         st.StateJSON,
		Status:            st.Status,
		ScoreRaw:          st.ScoreRaw,
		ScoreMax:          st.ScoreMax,
		FirstInteractedAt: st.FirstInteractedAt,
		LastInteractedAt:  st.LastInteractedAt,
		CompletedAt:       st.CompletedAt,
	})
	facets, _ := json.Marshal(sum.Facets)
	err = ctrepo.UpsertStateSummary(ctx, pool, ctrepo.StateSummaryRow{
		StateID:           st.ID,
		InstanceID:        st.InstanceID,
		CourseID:          courseID,
		EnrollmentID:      st.EnrollmentID,
		ToolID:            toolID,
		Role:              role,
		Engaged:           sum.Engaged,
		Completed:         sum.Completed,
		ScorePct:          sum.ScorePct,
		DurationMs:        sum.DurationMs,
		FacetsJSON:        facets,
		ProjectionVersion: sum.ProjectionVersion,
	})
	if err != nil {
		IncSummaryWrite(toolID, "error")
		return err
	}
	IncSummaryWrite(toolID, "ok")
	itemID := ""
	course, item, _, ierr := ctrepo.InstanceCourseAndItem(ctx, pool, st.InstanceID)
	if ierr == nil {
		if item != nil {
			itemID = item.String()
		}
		InvalidateForInstance(st.InstanceID.String(), course.String(), itemID)
	} else {
		InvalidateForInstance(st.InstanceID.String(), courseID.String(), "")
	}
	return nil
}

// SyncSummaryAfterReset clears the summary for a reset state and invalidates caches (FR-16, AC-10).
func SyncSummaryAfterReset(ctx context.Context, pool *pgxpool.Pool, st *ctrepo.StateRow, courseID uuid.UUID, toolID string) {
	if pool == nil || st == nil {
		return
	}
	// Re-project empty/reset state so parity holds rather than only nulling fields.
	if err := SyncSummaryFromState(ctx, pool, courseID, toolID, st); err != nil {
		slog.Warn("contenttools.analytics.reset_summary", "err", err, "state_id", st.ID)
		_ = ctrepo.ResetStateSummary(ctx, pool, st.ID)
		InvalidateForInstance(st.InstanceID.String(), courseID.String(), "")
	}
}

// RebuildSummaries recomputes summaries for states with missing/outdated projections (AC-9).
func RebuildSummaries(ctx context.Context, pool *pgxpool.Pool, toolID string, limit int) (int, error) {
	states, err := ctrepo.ListStatesNeedingSummary(ctx, pool, toolID, ProjectionVersion, limit)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range states {
		st := &states[i]
		courseID, _, tid, err := ctrepo.InstanceCourseAndItem(ctx, pool, st.InstanceID)
		if err != nil || courseID == uuid.Nil {
			continue
		}
		if toolID == "" {
			toolID = tid
		}
		useTool := tid
		if useTool == "" {
			useTool = toolID
		}
		if err := SyncSummaryFromState(ctx, pool, courseID, useTool, st); err != nil {
			slog.Warn("contenttools.analytics.rebuild", "err", err, "state_id", st.ID)
			continue
		}
		n++
	}
	return n, nil
}

// ToAggregateRows converts DB summary rows to aggregate input.
func ToAggregateRows(rows []ctrepo.StateSummaryRow) []SummaryRow {
	out := make([]SummaryRow, 0, len(rows))
	for _, r := range rows {
		facets := map[string]any{}
		if len(r.FacetsJSON) > 0 {
			_ = json.Unmarshal(r.FacetsJSON, &facets)
		}
		out = append(out, SummaryRow{
			EnrollmentID: r.EnrollmentID.String(),
			DisplayName:  r.DisplayName,
			Role:         r.Role,
			Engaged:      r.Engaged,
			Completed:    r.Completed,
			ScorePct:     r.ScorePct,
			DurationMs:   r.DurationMs,
			Facets:       facets,
			Status:       r.Status,
		})
	}
	return out
}

// MaybePushGradebook pushes a score when the instance is bridged and completed (FR-8).
func MaybePushGradebook(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, st *ctrepo.StateRow, toolID string) {
	if pool == nil || st == nil {
		return
	}
	if st.Status != "completed" && st.Status != "submitted" {
		return
	}
	link, err := ctrepo.GetGradeLink(ctx, pool, st.InstanceID)
	if err != nil || link == nil || !link.CountsForGrade || link.AssignmentItemID == nil {
		return
	}
	ptsPossible := 0.0
	if link.PointsPossible != nil {
		ptsPossible = *link.PointsPossible
	}
	if ptsPossible <= 0 {
		return
	}
	sum := Project(ProjectInput{
		ToolID:    toolID,
		StateJSON: st.StateJSON,
		Status:    st.Status,
		ScoreRaw:  st.ScoreRaw,
		ScoreMax:  st.ScoreMax,
	})
	pts := PointsFromScorePct(sum.ScorePct, ptsPossible)
	if err := PushGrade(ctx, pool, courseID, st.UserID, *link.AssignmentItemID, pts); err != nil {
		IncGradebookPush(toolID, "error")
		slog.Warn("contenttools.analytics.grade_push", "err", err, "instance_id", st.InstanceID)
		return
	}
	IncGradebookPush(toolID, "ok")
}
