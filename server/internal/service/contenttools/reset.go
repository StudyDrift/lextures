package contenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	ferpaservice "github.com/lextures/lextures/server/internal/service/ferpa"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

const (
	// EnvAsyncResetThreshold controls when resets become async (default 200).
	EnvAsyncResetThreshold = "CONTENT_TOOLS_ASYNC_RESET_THRESHOLD"

	DefaultAsyncResetThreshold = 200
	ResetBatchSize             = 500
)

// ResetScope values (instructor + self).
var ValidResetScopes = map[string]struct{}{
	ctrepo.ResetScopeInstanceEnrollment: {},
	ctrepo.ResetScopeInstanceAll:        {},
	ctrepo.ResetScopeItemEnrollment:     {},
	ctrepo.ResetScopeItemAll:            {},
	ctrepo.ResetScopeCourseEnrollment:   {},
}

// ResetRequest is the service input for instructor/self reset.
type ResetRequest struct {
	CourseID       uuid.UUID
	CourseCode     string
	OrgID          uuid.UUID
	ActorID        uuid.UUID
	ActorRole      string // instructor|ta|student
	Scope          string
	InstanceID     *uuid.UUID
	ItemID         *uuid.UUID
	EnrollmentID   *uuid.UUID
	SectionIDs     []uuid.UUID // optional client filter
	TASectionIDs   []uuid.UUID // server-resolved TA narrowing (non-empty = apply)
	Reason         *string
	Notify         bool
	DryRun         bool
	IdempotencyKey *string
	InitialState   json.RawMessage // default {}
	ToolID         string          // for metrics; optional
	ActivityTitle  string          // for notifications
}

// GradeEffect describes gradebook side-effects for one enrollment (CT.7 stub).
type GradeEffect struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	Action       string    `json:"action"` // reverted|unchanged|blocked
	Reason       *string   `json:"reason,omitempty"`
}

// ResetSample is one learner in the dry-run / result sample.
type ResetSample struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	DisplayName  string    `json:"displayName"`
	Status       string    `json:"status"`
	Score        *float64  `json:"score"`
}

// ResetResult is returned by ExecuteReset.
type ResetResult struct {
	DryRun          bool
	AffectedCount   int
	Sample          []ResetSample
	BatchID         *uuid.UUID
	JobID           *uuid.UUID
	GradeEffects    []GradeEffect
	ScopeNarrowed   bool
	AppliedSections []uuid.UUID
	Async           bool
}

// AsyncResetThreshold returns the sync/async cutoff (default 200).
func AsyncResetThreshold() int {
	v := strings.TrimSpace(os.Getenv(EnvAsyncResetThreshold))
	if v == "" {
		return DefaultAsyncResetThreshold
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return DefaultAsyncResetThreshold
	}
	return n
}

// InitialStateForTool returns the tool's declared initial state document ({} when unset).
func InitialStateForTool(m *CompiledManifest) json.RawMessage {
	_ = m
	return json.RawMessage(`{}`)
}

// SummarizeState returns a short human-readable summary of learner state for instructors.
func SummarizeState(toolID string, state json.RawMessage, status string, scoreRaw, scoreMax *float64) string {
	parts := []string{fmt.Sprintf("status=%s", status)}
	if scoreRaw != nil && scoreMax != nil {
		parts = append(parts, fmt.Sprintf("score=%g/%g", *scoreRaw, *scoreMax))
	}
	if toolID == "noop_probe" && len(state) > 0 {
		var s struct {
			Response string `json:"response"`
			Attempts int    `json:"attempts"`
		}
		if json.Unmarshal(state, &s) == nil && (s.Response != "" || s.Attempts > 0) {
			resp := s.Response
			if len(resp) > 80 {
				resp = resp[:80] + "…"
			}
			parts = append(parts, fmt.Sprintf("response=%q attempts=%d", resp, s.Attempts))
		}
	}
	return strings.Join(parts, "; ")
}

// ClassifyGradeEffect stubs CT.7 gradebook bridge: scores are unchanged until passback exists.
func ClassifyGradeEffect(enrollmentID uuid.UUID, scoringMode string, hadScore bool) GradeEffect {
	_ = scoringMode
	_ = hadScore
	return GradeEffect{EnrollmentID: enrollmentID, Action: "unchanged"}
}

// ExecuteReset performs dry-run or real reset (sync path). Caller handles async enqueue when Async is set.
func ExecuteReset(ctx context.Context, pool *pgxpool.Pool, req ResetRequest) (*ResetResult, error) {
	if _, ok := ValidResetScopes[req.Scope]; !ok && req.Scope != ctrepo.ResetScopeSelf {
		return nil, fmt.Errorf("invalid scope")
	}

	sectionFilter := req.TASectionIDs
	if len(req.SectionIDs) > 0 {
		if len(sectionFilter) == 0 {
			sectionFilter = req.SectionIDs
		} else {
			sectionFilter = intersectUUIDs(sectionFilter, req.SectionIDs)
		}
	}
	scopeNarrowed := len(req.TASectionIDs) > 0

	affected, err := ctrepo.ResolveAffectedStates(
		ctx, pool, req.CourseID, req.Scope, req.InstanceID, req.ItemID, req.EnrollmentID, sectionFilter,
	)
	if err != nil {
		return nil, err
	}

	sample := make([]ResetSample, 0, min(10, len(affected)))
	gradeEffects := make([]GradeEffect, 0, len(affected))
	for i, a := range affected {
		var score *float64
		if a.State.ScoreRaw != nil {
			score = a.State.ScoreRaw
		}
		if i < 10 {
			sample = append(sample, ResetSample{
				EnrollmentID: a.EnrollmentID,
				DisplayName:  a.DisplayName,
				Status:       a.State.Status,
				Score:        score,
			})
		}
		gradeEffects = append(gradeEffects, ClassifyGradeEffect(a.EnrollmentID, "", a.State.ScoreRaw != nil))
	}

	result := &ResetResult{
		DryRun:          req.DryRun,
		AffectedCount:   len(affected),
		Sample:          sample,
		GradeEffects:    gradeEffects,
		ScopeNarrowed:   scopeNarrowed,
		AppliedSections: append([]uuid.UUID{}, sectionFilter...),
	}

	if req.DryRun {
		return result, nil
	}

	if len(affected) == 0 {
		batchID := uuid.New()
		result.BatchID = &batchID
		return result, nil
	}

	threshold := AsyncResetThreshold()
	if len(affected) > threshold {
		result.Async = true
		return result, nil
	}

	batchID := uuid.New()
	result.BatchID = &batchID
	initial := req.InitialState
	if len(initial) == 0 {
		initial = json.RawMessage(`{}`)
	}
	retentionDays, err := ctrepo.OrgRetentionDays(ctx, pool, req.CourseID)
	if err != nil {
		retentionDays = ctrepo.DefaultRetentionDays
	}
	purgeAfter := time.Now().UTC().Add(time.Duration(retentionDays) * 24 * time.Hour)

	if err := applyResetBatch(ctx, pool, req, affected, batchID, initial, purgeAfter); err != nil {
		return nil, err
	}

	IncResets(req.ToolID, req.Scope, req.ActorRole)
	IncResetRows(req.Scope, len(affected))
	return result, nil
}

func applyResetBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	req ResetRequest,
	affected []ctrepo.AffectedState,
	batchID uuid.UUID,
	initial json.RawMessage,
	purgeAfter time.Time,
) error {
	for start := 0; start < len(affected); start += ResetBatchSize {
		end := start + ResetBatchSize
		if end > len(affected) {
			end = len(affected)
		}
		chunk := affected[start:end]
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		for _, a := range chunk {
			reason := req.Reason
			snap := ctrepo.StateResetRow{
				InstanceID:     a.InstanceID,
				EnrollmentID:   a.EnrollmentID,
				CourseID:       req.CourseID,
				ToolID:         a.ToolID,
				Scope:          req.Scope,
				Reason:         reason,
				PriorStateJSON: a.State.StateJSON,
				PriorStatus:    a.State.Status,
				PriorScoreRaw:  a.State.ScoreRaw,
				PriorScoreMax:  a.State.ScoreMax,
				PriorRevision:  a.State.Revision,
				BatchID:        &batchID,
				ResetBy:        &req.ActorID,
				PurgeAfter:     purgeAfter,
			}
			if _, err := ctrepo.InsertStateReset(ctx, tx, snap); err != nil {
				_ = tx.Rollback(ctx)
				return err
			}
			if _, err := ctrepo.ClearStateForReset(ctx, tx, a.State.ID, initial, req.ActorID); err != nil {
				_ = tx.Rollback(ctx)
				return err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		// Events + notifications outside the short transaction.
		for _, a := range chunk {
			enr := a.EnrollmentID
			actor := req.ActorID
			inst := a.InstanceID
			_ = ctrepo.InsertEvent(ctx, pool, req.CourseID, &inst, &enr, &actor, a.ToolID, EventStateReset, map[string]any{
				"scope":     req.Scope,
				"batchId":   batchID.String(),
				"reason":    req.Reason,
				"notify":    req.Notify,
				"actorRole": req.ActorRole,
			})
			if req.Notify && a.UserID != req.ActorID {
				title := "Activity reset"
				body := "Your instructor reset your work on this activity. You can start again."
				if req.ActivityTitle != "" {
					body = fmt.Sprintf("Your instructor reset your work on “%s”. You can start again.", req.ActivityTitle)
				}
				actionURL := fmt.Sprintf("/courses/%s", req.CourseCode)
				push := &notifications.PushService{Pool: pool}
				_ = push.Enqueue(ctx, a.UserID, notifications.EventContentToolStateReset, title, body, actionURL)
			}
		}
	}

	targetType := "content_tool_reset_batch"
	targetID := batchID
	before, _ := json.Marshal(map[string]any{"affectedCount": len(affected), "scope": req.Scope})
	after, _ := json.Marshal(map[string]any{"batchId": batchID.String(), "dryRun": false})
	_, _ = adminaudit.Record(ctx, pool, adminaudit.RecordParams{
		OrgID:       &req.OrgID,
		EventType:   adminaudit.EventContentToolStateReset,
		ActorID:     req.ActorID,
		TargetType:  &targetType,
		TargetID:    &targetID,
		BeforeValue: before,
		AfterValue:  after,
	})
	return nil
}

// RunAsyncResetJob executes a queued reset job.
func RunAsyncResetJob(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, req ResetRequest) error {
	start := time.Now()
	if err := ctrepo.MarkResetJobRunning(ctx, pool, jobID); err != nil {
		return err
	}
	affected, err := ctrepo.ResolveAffectedStates(
		ctx, pool, req.CourseID, req.Scope, req.InstanceID, req.ItemID, req.EnrollmentID, req.TASectionIDs,
	)
	if err != nil {
		msg := err.Error()
		_ = ctrepo.FinishResetJob(ctx, pool, jobID, "failed", nil, nil, &msg)
		return err
	}
	batchID := uuid.New()
	initial := req.InitialState
	if len(initial) == 0 {
		initial = json.RawMessage(`{}`)
	}
	retentionDays, _ := ctrepo.OrgRetentionDays(ctx, pool, req.CourseID)
	if retentionDays < ctrepo.MinRetentionDays {
		retentionDays = ctrepo.DefaultRetentionDays
	}
	purgeAfter := time.Now().UTC().Add(time.Duration(retentionDays) * 24 * time.Hour)

	for startIdx := 0; startIdx < len(affected); startIdx += ResetBatchSize {
		end := startIdx + ResetBatchSize
		if end > len(affected) {
			end = len(affected)
		}
		if err := applyResetBatch(ctx, pool, req, affected[startIdx:end], batchID, initial, purgeAfter); err != nil {
			msg := err.Error()
			_ = ctrepo.FinishResetJob(ctx, pool, jobID, "failed", &batchID, nil, &msg)
			return err
		}
		_ = ctrepo.UpdateResetJobProgress(ctx, pool, jobID, end)
	}

	resultJSON, _ := json.Marshal(map[string]any{
		"affectedCount": len(affected),
		"batchId":       batchID.String(),
	})
	if err := ctrepo.FinishResetJob(ctx, pool, jobID, "succeeded", &batchID, resultJSON, nil); err != nil {
		return err
	}
	IncResets(req.ToolID, req.Scope, req.ActorRole)
	IncResetRows(req.Scope, len(affected))
	ObserveResetJobDuration(time.Since(start).Seconds())
	return nil
}

// RestoreReset restores a single snapshot and audits.
func RestoreReset(ctx context.Context, pool *pgxpool.Pool, orgID, courseID, actorID, resetID uuid.UUID) (*ctrepo.StateRow, *ctrepo.StateResetRow, error) {
	st, snap, err := ctrepo.RestoreStateFromReset(ctx, pool, resetID, actorID)
	if err != nil {
		return nil, nil, err
	}
	if snap == nil {
		return nil, nil, nil
	}
	enr := snap.EnrollmentID
	inst := snap.InstanceID
	actor := actorID
	_ = ctrepo.InsertEvent(ctx, pool, courseID, &inst, &enr, &actor, snap.ToolID, EventStateResetRestored, map[string]any{
		"resetId": resetID.String(),
	})
	targetType := "content_tool_state_reset"
	_, _ = adminaudit.Record(ctx, pool, adminaudit.RecordParams{
		OrgID:      &orgID,
		EventType:  adminaudit.EventContentToolStateRestore,
		ActorID:    actorID,
		TargetType: &targetType,
		TargetID:   &resetID,
	})
	IncResetRestores()
	return st, snap, nil
}

// LogStateDetailAccess writes a FERPA disclosure when an instructor views another learner's state.
func LogStateDetailAccess(ctx context.Context, pool *pgxpool.Pool, orgID, accessorID, studentID uuid.UUID) error {
	return ferpaservice.LogDisclosure(ctx, pool, orgID, accessorID, studentID, "content_tool_state", "school_official", nil)
}

func intersectUUIDs(a, b []uuid.UUID) []uuid.UUID {
	set := make(map[uuid.UUID]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	out := make([]uuid.UUID, 0)
	for _, id := range b {
		if _, ok := set[id]; ok {
			out = append(out, id)
		}
	}
	return out
}
