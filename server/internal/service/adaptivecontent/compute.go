package adaptivecontent

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/learnermodel"
	"github.com/lextures/lextures/server/internal/service/adaptivepath"
)

// PreAssessmentAttempt is the minimal attempt payload for the submit hook.
type PreAssessmentAttempt struct {
	AttemptID       uuid.UUID
	CourseID        uuid.UUID
	StructureItemID uuid.UUID
	StudentUserID   uuid.UUID
}

// ResolveUnitConceptIDs returns the concept set for a unit:
// explicit unit concepts ∪ pre-assessment question tags (when preAssessmentItemID is set).
func ResolveUnitConceptIDs(
	ctx context.Context,
	pool *pgxpool.Pool,
	unitID uuid.UUID,
	preAssessmentItemID *uuid.UUID,
	courseID uuid.UUID,
) ([]uuid.UUID, error) {
	explicit, err := acrepo.ListUnitConceptIDs(ctx, pool, unitID)
	if err != nil {
		return nil, err
	}
	seen := make(map[uuid.UUID]struct{}, len(explicit))
	out := make([]uuid.UUID, 0, len(explicit)+8)
	for _, id := range explicit {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if preAssessmentItemID != nil && *preAssessmentItemID != uuid.Nil {
		tagged, err := acrepo.ConceptIDsTaggedOnQuizQuestions(ctx, pool, courseID, *preAssessmentItemID)
		if err != nil {
			return nil, err
		}
		for _, id := range tagged {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out, nil
}

// EffectiveAxes returns unit axes or falls back to course settings.
func EffectiveAxes(ctx context.Context, pool *pgxpool.Pool, unit acrepo.UnitRow) ([]string, error) {
	if len(unit.AllowedAxes) > 0 {
		return NormalizeAxes(unit.AllowedAxes), nil
	}
	s, err := acrepo.GetSettings(ctx, pool, unit.CourseID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return NormalizeAxes(acrepo.DefaultSettings(unit.CourseID).AllowedAxes), nil
	}
	return NormalizeAxes(s.AllowedAxes), nil
}

// ComputeAndUpsertProfile loads mastery + misconceptions, runs pure rules, and upserts the profile.
// On load errors it writes a neutral profile (FR-8) so the learner is never blocked.
func ComputeAndUpsertProfile(
	ctx context.Context,
	pool *pgxpool.Pool,
	unit acrepo.UnitRow,
	enrollmentID, userID uuid.UUID,
	sourceAttemptID *uuid.UUID,
	misconceptionIDs []uuid.UUID,
) (*acrepo.ProfileRow, ProfileResult, error) {
	start := time.Now()
	defer func() {
		ObserveProfileCompute(float64(time.Since(start).Milliseconds()))
	}()

	axes, err := EffectiveAxes(ctx, pool, unit)
	if err != nil {
		slog.Error("adaptivecontent: load axes failed; writing neutral profile",
			"unit_id", unit.ID, "err", err)
		return writeNeutral(ctx, pool, unit, enrollmentID, userID, sourceAttemptID, nil, "default", "default")
	}

	conceptIDs, err := ResolveUnitConceptIDs(ctx, pool, unit.ID, unit.PreAssessmentItemID, unit.CourseID)
	if err != nil {
		slog.Error("adaptivecontent: resolve concepts failed; writing neutral profile",
			"unit_id", unit.ID, "err", err)
		return writeNeutral(ctx, pool, unit, enrollmentID, userID, sourceAttemptID, axes, "default", "default")
	}

	masteryMap := map[uuid.UUID]float64{}
	if len(conceptIDs) > 0 {
		states, err := learnermodel.ListConceptStatesForUser(ctx, pool, userID, conceptIDs)
		if err != nil {
			slog.Error("adaptivecontent: load concept states failed; writing neutral profile",
				"unit_id", unit.ID, "user_id", userID, "err", err)
			return writeNeutral(ctx, pool, unit, enrollmentID, userID, sourceAttemptID, axes, "default", "default")
		}
		freshness := int(unit.MasteryFreshnessDays)
		if unit.TriggerMode == TriggerMasterySnapshot {
			now := time.Now().UTC()
			for _, st := range states {
				if MasteryIsFresh(st.LastSeenAt, freshness, now) {
					masteryMap[st.ConceptID] = st.MasteryEffective
				}
			}
		} else {
			for _, st := range states {
				masteryMap[st.ConceptID] = st.MasteryEffective
			}
		}
	}

	if misconceptionIDs == nil {
		misconceptionIDs = []uuid.UUID{}
	}

	result := ComputeProfile(ProfileInput{
		UnitID:           unit.ID,
		ConceptIDs:       conceptIDs,
		ConceptMastery:   masteryMap,
		MisconceptionIDs: misconceptionIDs,
		AxisSet:          axes,
		ReadingLevelPref: "default",
		ModalityPref:     "default",
	})

	var bloomPtr *string
	if result.TargetBloom != "" {
		b := result.TargetBloom
		bloomPtr = &b
	}

	row, err := acrepo.UpsertProfile(ctx, pool, acrepo.ProfileUpsert{
		UnitID:           unit.ID,
		EnrollmentID:     enrollmentID,
		UserID:           userID,
		ProfileSignature: result.ProfileSignature,
		EmphasisMode:     result.EmphasisMode,
		PayloadJSON:      result.Payload,
		SourceAttemptID:  sourceAttemptID,
		TargetBloom:      bloomPtr,
		ReadingLevelPref: result.ReadingLevelPref,
		ModalityPref:     result.ModalityPref,
		AxisSet:          result.AxisSet,
		IsNeutral:        result.IsNeutral,
	})
	if err != nil {
		return nil, result, err
	}

	// Audit event (FR-10).
	actor := userID
	subject := userID
	uid := unit.ID
	_ = acrepo.InsertEvent(ctx, pool, unit.CourseID, &uid, &actor, &subject, EventProfileComputed, map[string]any{
		"unitId":           unit.ID,
		"enrollmentId":     enrollmentID,
		"emphasisMode":     result.EmphasisMode,
		"profileSignature": result.ProfileSignature,
		"isNeutral":        result.IsNeutral,
		"sourceAttemptId":  sourceAttemptID,
		"payload":          result.Payload,
	})
	IncProfileEmphasis(result.EmphasisMode)
	return row, result, nil
}

func writeNeutral(
	ctx context.Context,
	pool *pgxpool.Pool,
	unit acrepo.UnitRow,
	enrollmentID, userID uuid.UUID,
	sourceAttemptID *uuid.UUID,
	axes []string,
	reading, modality string,
) (*acrepo.ProfileRow, ProfileResult, error) {
	result := NeutralProfile(unit.ID, axes, reading, modality)
	bloom := result.TargetBloom
	row, err := acrepo.UpsertProfile(ctx, pool, acrepo.ProfileUpsert{
		UnitID:           unit.ID,
		EnrollmentID:     enrollmentID,
		UserID:           userID,
		ProfileSignature: result.ProfileSignature,
		EmphasisMode:     result.EmphasisMode,
		PayloadJSON:      result.Payload,
		SourceAttemptID:  sourceAttemptID,
		TargetBloom:      &bloom,
		ReadingLevelPref: result.ReadingLevelPref,
		ModalityPref:     result.ModalityPref,
		AxisSet:          result.AxisSet,
		IsNeutral:        true,
	})
	if err != nil {
		return nil, result, err
	}
	actor := userID
	subject := userID
	uid := unit.ID
	_ = acrepo.InsertEvent(ctx, pool, unit.CourseID, &uid, &actor, &subject, EventProfileComputed, map[string]any{
		"unitId":           unit.ID,
		"enrollmentId":     enrollmentID,
		"emphasisMode":     result.EmphasisMode,
		"profileSignature": result.ProfileSignature,
		"isNeutral":        true,
		"sourceAttemptId":  sourceAttemptID,
		"fallback":         true,
	})
	IncProfileEmphasis(result.EmphasisMode)
	return row, result, nil
}

// OnPreAssessmentSubmitted is the post-submit hook for quiz delivery.
// It is best-effort: failures are logged and never fail the quiz submit.
func OnPreAssessmentSubmitted(ctx context.Context, pool *pgxpool.Pool, attempt PreAssessmentAttempt) {
	if pool == nil || attempt.AttemptID == uuid.Nil {
		return
	}
	if KillSwitchEngaged() {
		return
	}
	enabled, err := acrepo.AdaptiveContentEnabledForCourse(ctx, pool, attempt.CourseID)
	if err != nil || !ActiveForCourse(enabled) {
		return
	}

	units, err := acrepo.ListActiveUnitsByPreAssessment(ctx, pool, attempt.CourseID, attempt.StructureItemID)
	if err != nil {
		slog.Error("adaptivecontent: list units by pre-assessment failed", "err", err)
		return
	}
	if len(units) == 0 {
		return
	}

	enrollmentID, err := acrepo.GetEnrollmentIDForUser(ctx, pool, attempt.CourseID, attempt.StudentUserID)
	if err != nil {
		slog.Error("adaptivecontent: resolve enrollment failed", "err", err)
		return
	}
	if enrollmentID == nil {
		// Instructor/staff taking the quiz — skip.
		return
	}

	misIDs, err := acrepo.ListMisconceptionIDsForAttempt(ctx, pool, attempt.AttemptID)
	if err != nil {
		slog.Error("adaptivecontent: load misconception events failed", "attempt_id", attempt.AttemptID, "err", err)
		misIDs = nil
	}

	attemptID := attempt.AttemptID
	for _, unit := range units {
		// Only pre_quiz (and diagnostic when bound as pre-assessment) recompute on submit.
		mode := NormalizeTriggerMode(unit.TriggerMode)
		if mode != TriggerPreQuiz && mode != TriggerDiagnosticFirstVisit {
			continue
		}
		row, result, err := ComputeAndUpsertProfile(ctx, pool, unit, *enrollmentID, attempt.StudentUserID, &attemptID, misIDs)
		if err != nil {
			slog.Error("adaptivecontent: profile compute failed",
				"unit_id", unit.ID, "attempt_id", attempt.AttemptID, "err", err)
			continue
		}
		// AC.4: enqueue generation for new signatures (dedupe handles thundering herd).
		if row != nil {
			MaybeEnqueueAfterProfile(ctx, pool, unit, result)
		}
	}
}

// EnsureMasterySnapshotProfile computes a profile from current mastery when missing
// and the unit uses mastery_snapshot (or diagnostic_first_visit without a pre-quiz attempt).
func EnsureMasterySnapshotProfile(
	ctx context.Context,
	pool *pgxpool.Pool,
	unit acrepo.UnitRow,
	enrollmentID, userID uuid.UUID,
) (*acrepo.ProfileRow, error) {
	existing, err := acrepo.GetProfileForEnrollment(ctx, pool, unit.ID, enrollmentID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	row, _, err := ComputeAndUpsertProfile(ctx, pool, unit, enrollmentID, userID, nil, nil)
	return row, err
}

// ValidateConceptsBelongToCourse ensures concept ids are usable in the course
// (reuses adaptive path concept validation).
func ValidateConceptsBelongToCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, conceptIDs []uuid.UUID) error {
	if len(conceptIDs) == 0 {
		return nil
	}
	return adaptivepath.ValidateConceptsForCourse(ctx, pool, courseID, conceptIDs)
}
