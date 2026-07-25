package adaptivecontent

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notifevents"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/atrisk"
	"github.com/lextures/lextures/server/internal/repos/learnermodel"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

// Named effectiveness thresholds (AC.7) — documented, tunable constants.
const (
	// MinNPerArm is the minimum sample size per treatment/holdout arm before a causal verdict.
	MinNPerArm = 10
	// HelpingMarginPts is the minimum treatment−holdout lift (percentage points) for "helping".
	HelpingMarginPts = 5.0
	// RegressingMarginPts is the (negative) margin below which treatment is "regressing".
	RegressingMarginPts = -5.0
	// SmallCellMinN suppresses group means when n < k (re-identification guard, FR-9 / AC-7).
	SmallCellMinN = 5

	VerdictHelping          = "helping"
	VerdictNoEffect         = "no_effect"
	VerdictInsufficientData = "insufficient_data"
	VerdictRegressing       = "regressing"

	EventOutcomeRecorded = "outcome_recorded"
	EventVerdictChanged  = "effectiveness_verdict_changed"
)

// PostAssessmentAttempt is the minimal attempt payload for the post-submit hook.
type PostAssessmentAttempt struct {
	AttemptID       uuid.UUID
	CourseID        uuid.UUID
	StructureItemID uuid.UUID
	StudentUserID   uuid.UUID
}

// EffectivenessNotifyDeps wires optional instructor alerts on regressing verdicts.
type EffectivenessNotifyDeps struct {
	Pool   *pgxpool.Pool
	Config config.Config
	SSEHub *notifevents.Hub
}

// GroupStats is difference-in-means summary for one arm.
type GroupStats struct {
	N    int
	Mean float64
	Var  float64 // sample variance (n-1); 0 when n < 2
}

// DiffInMeansResult is treatment vs holdout comparison.
type DiffInMeansResult struct {
	Treatment         GroupStats
	Holdout           GroupStats
	Diff              float64 // treatment mean − holdout mean
	StdError          float64
	Verdict           string
	MeanMasteryTreat  *float64
	MeanMasteryHoldout *float64
}

// ComputeLift returns post − pre when both are present.
func ComputeLift(pre, post *float32) *float32 {
	if pre == nil || post == nil {
		return nil
	}
	v := *post - *pre
	return &v
}

// MeanOf returns the arithmetic mean, or nil when empty.
func MeanOf(vals []float64) *float64 {
	if len(vals) == 0 {
		return nil
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	m := sum / float64(len(vals))
	return &m
}

// SampleVariance returns unbiased sample variance (n-1). Zero when n < 2.
func SampleVariance(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	m := *MeanOf(vals)
	var ss float64
	for _, v := range vals {
		d := v - m
		ss += d * d
	}
	return ss / float64(len(vals)-1)
}

// DiffInMeans computes treatment−holdout difference, SE, and verdict.
func DiffInMeans(treatmentLifts, holdoutLifts []float64, masteryTreat, masteryHoldout []float64) DiffInMeansResult {
	tStats := GroupStats{N: len(treatmentLifts)}
	hStats := GroupStats{N: len(holdoutLifts)}
	if tMean := MeanOf(treatmentLifts); tMean != nil {
		tStats.Mean = *tMean
		tStats.Var = SampleVariance(treatmentLifts)
	}
	if hMean := MeanOf(holdoutLifts); hMean != nil {
		hStats.Mean = *hMean
		hStats.Var = SampleVariance(holdoutLifts)
	}

	diff := tStats.Mean - hStats.Mean
	var se float64
	if tStats.N > 0 {
		se += tStats.Var / float64(tStats.N)
	}
	if hStats.N > 0 {
		se += hStats.Var / float64(hStats.N)
	}
	se = math.Sqrt(se)

	verdict := VerdictInsufficientData
	if tStats.N >= MinNPerArm && hStats.N >= MinNPerArm {
		switch {
		case diff >= HelpingMarginPts:
			verdict = VerdictHelping
		case diff <= RegressingMarginPts:
			verdict = VerdictRegressing
		default:
			verdict = VerdictNoEffect
		}
	}

	return DiffInMeansResult{
		Treatment:          tStats,
		Holdout:            hStats,
		Diff:               diff,
		StdError:           se,
		Verdict:            verdict,
		MeanMasteryTreat:   MeanOf(masteryTreat),
		MeanMasteryHoldout: MeanOf(masteryHoldout),
	}
}

// SuppressSmallCell returns nil mean when n < SmallCellMinN (cohort de-identification).
func SuppressSmallCell(n int, mean *float32) *float32 {
	if n < SmallCellMinN {
		return nil
	}
	return mean
}

// ModeAggregate is per-emphasis-mode lift.
type ModeAggregate struct {
	EmphasisMode string
	N            int
	MeanLift     *float32
}

// VariantAggregate is per-variant lift.
type VariantAggregate struct {
	VariantID *uuid.UUID
	N         int
	MeanLift  *float32
}

// AggregateByMode groups samples by emphasis mode (treatment only for mode comparison).
func AggregateByMode(samples []acrepo.OutcomeLiftSample) []ModeAggregate {
	sums := map[string]float64{}
	counts := map[string]int{}
	for _, s := range samples {
		if s.WasHoldout {
			continue // mode effectiveness is about adapted variants
		}
		mode := s.EmphasisMode
		if mode == "" {
			mode = "unknown"
		}
		sums[mode] += float64(s.Lift)
		counts[mode]++
	}
	modes := make([]string, 0, len(counts))
	for m := range counts {
		modes = append(modes, m)
	}
	sort.Strings(modes)
	out := make([]ModeAggregate, 0, len(modes))
	for _, m := range modes {
		n := counts[m]
		mean := float32(sums[m] / float64(n))
		out = append(out, ModeAggregate{
			EmphasisMode: m,
			N:            n,
			MeanLift:     SuppressSmallCell(n, &mean),
		})
	}
	return out
}

// AggregateByVariant groups samples by variant_id.
func AggregateByVariant(samples []acrepo.OutcomeLiftSample) []VariantAggregate {
	type key struct {
		has bool
		id  uuid.UUID
	}
	sums := map[key]float64{}
	counts := map[key]int{}
	for _, s := range samples {
		k := key{}
		if s.VariantID != nil {
			k.has = true
			k.id = *s.VariantID
		}
		sums[k] += float64(s.Lift)
		counts[k]++
	}
	keys := make([]key, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] != counts[keys[j]] {
			return counts[keys[i]] > counts[keys[j]]
		}
		if keys[i].has != keys[j].has {
			return keys[i].has
		}
		return keys[i].id.String() < keys[j].id.String()
	})
	out := make([]VariantAggregate, 0, len(keys))
	for _, k := range keys {
		n := counts[k]
		mean := float32(sums[k] / float64(n))
		var vid *uuid.UUID
		if k.has {
			id := k.id
			vid = &id
		}
		out = append(out, VariantAggregate{
			VariantID: vid,
			N:         n,
			MeanLift:  SuppressSmallCell(n, &mean),
		})
	}
	return out
}

// meanMasteryFromGaps derives mastery ≈ 1 − meanGap from a profile payload.
func meanMasteryFromGaps(payload json.RawMessage) *float32 {
	if len(payload) == 0 {
		return nil
	}
	var p ProfilePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil
	}
	if len(p.ConceptGaps) == 0 && p.MeanGap == 0 && !p.PriorRecord {
		return nil
	}
	m := float32(1 - p.MeanGap)
	if m < 0 {
		m = 0
	}
	if m > 1 {
		m = 1
	}
	return &m
}

// meanMasteryNow averages current learner_concept_states for unit concepts.
func meanMasteryNow(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, conceptIDs []uuid.UUID) *float32 {
	if len(conceptIDs) == 0 {
		return nil
	}
	states, err := learnermodel.ListConceptStatesForUser(ctx, pool, userID, conceptIDs)
	if err != nil || len(states) == 0 {
		return nil
	}
	var sum float64
	for _, st := range states {
		sum += st.MasteryEffective
	}
	m := float32(sum / float64(len(states)))
	return &m
}

// OnPostAssessmentSubmitted records lift against the student's serving (best-effort).
func OnPostAssessmentSubmitted(ctx context.Context, pool *pgxpool.Pool, attempt PostAssessmentAttempt) {
	if pool == nil || attempt.AttemptID == uuid.Nil {
		return
	}
	start := time.Now()
	defer func() {
		ObserveOutcomeRecord(float64(time.Since(start).Milliseconds()))
	}()

	if KillSwitchEngaged() {
		return
	}
	enabled, err := acrepo.AdaptiveContentEnabledForCourse(ctx, pool, attempt.CourseID)
	if err != nil || !ActiveForCourse(enabled) {
		return
	}

	units, err := acrepo.ListActiveUnitsByPostAssessment(ctx, pool, attempt.CourseID, attempt.StructureItemID)
	if err != nil {
		slog.Error("adaptivecontent: list units by post-assessment failed", "err", err)
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
		return
	}

	postScore, err := acrepo.QuizAttemptScorePct(ctx, pool, attempt.AttemptID)
	if err != nil {
		slog.Error("adaptivecontent: load post score failed", "attempt_id", attempt.AttemptID, "err", err)
		return
	}

	attemptID := attempt.AttemptID
	for _, unit := range units {
		serving, err := acrepo.GetLatestServingForEnrollment(ctx, pool, unit.ID, *enrollmentID)
		if err != nil {
			slog.Error("adaptivecontent: load serving failed", "unit_id", unit.ID, "err", err)
			continue
		}
		if serving == nil {
			// No exposure yet — cannot bind outcome to a serving_id.
			slog.Info("adaptivecontent: post-assessment without serving; skipping outcome",
				"unit_id", unit.ID, "enrollment_id", *enrollmentID)
			continue
		}

		var preScore *float32
		var emphasis *string
		var masteryBefore *float32
		profile, err := acrepo.GetProfileForEnrollment(ctx, pool, unit.ID, *enrollmentID)
		if err != nil {
			slog.Error("adaptivecontent: load profile for outcome failed", "unit_id", unit.ID, "err", err)
		} else if profile != nil {
			emphasis = profile.EmphasisMode
			masteryBefore = meanMasteryFromGaps(profile.PayloadJSON)
			if profile.SourceAttemptID != nil {
				preScore, err = acrepo.QuizAttemptScorePct(ctx, pool, *profile.SourceAttemptID)
				if err != nil {
					slog.Error("adaptivecontent: load pre score failed", "attempt_id", *profile.SourceAttemptID, "err", err)
					preScore = nil
				}
			}
		}

		conceptIDs, err := ResolveUnitConceptIDs(ctx, pool, unit.ID, unit.PreAssessmentItemID, unit.CourseID)
		if err != nil {
			slog.Error("adaptivecontent: resolve concepts for mastery_after failed", "unit_id", unit.ID, "err", err)
			conceptIDs = nil
		}
		masteryAfter := meanMasteryNow(ctx, pool, attempt.StudentUserID, conceptIDs)
		lift := ComputeLift(preScore, postScore)

		_, err = acrepo.UpsertOutcome(ctx, pool, acrepo.OutcomeUpsert{
			ServingID:     serving.ID,
			PreScorePct:   preScore,
			PostScorePct:  postScore,
			MasteryBefore: masteryBefore,
			MasteryAfter:  masteryAfter,
			Lift:          lift,
			EmphasisMode:  emphasis,
			WasHoldout:    serving.WasHoldout,
			PostAttemptID: &attemptID,
		})
		if err != nil {
			slog.Error("adaptivecontent: upsert outcome failed",
				"unit_id", unit.ID, "serving_id", serving.ID, "err", err)
			continue
		}
		IncOutcomesRecorded()
		actor := attempt.StudentUserID
		subject := attempt.StudentUserID
		uid := unit.ID
		_ = acrepo.InsertEvent(ctx, pool, unit.CourseID, &uid, &actor, &subject, EventOutcomeRecorded, map[string]any{
			"unitId":        unit.ID,
			"servingId":     serving.ID,
			"wasHoldout":    serving.WasHoldout,
			"lift":          lift,
			"postAttemptId": attemptID,
		})
	}
}

// RefreshUnit recomputes and caches effectiveness for one unit. Returns the new verdict.
func RefreshUnit(ctx context.Context, pool *pgxpool.Pool, unit acrepo.UnitRow, notify *EffectivenessNotifyDeps) (string, error) {
	samples, err := acrepo.ListOutcomeLiftSamples(ctx, pool, unit.ID)
	if err != nil {
		return "", err
	}

	var treatLifts, holdLifts []float64
	var treatMastery, holdMastery []float64
	for _, s := range samples {
		if s.WasHoldout {
			holdLifts = append(holdLifts, float64(s.Lift))
			if s.MasteryDelta != nil {
				holdMastery = append(holdMastery, float64(*s.MasteryDelta))
			}
		} else {
			treatLifts = append(treatLifts, float64(s.Lift))
			if s.MasteryDelta != nil {
				treatMastery = append(treatMastery, float64(*s.MasteryDelta))
			}
		}
	}

	diff := DiffInMeans(treatLifts, holdLifts, treatMastery, holdMastery)

	prev, _ := acrepo.GetPreviousVerdict(ctx, pool, unit.ID)

	var meanTreat, meanHold, diffPtr, sePtr, mTreat, mHold *float32
	if diff.Treatment.N > 0 {
		v := float32(diff.Treatment.Mean)
		meanTreat = &v
	}
	if diff.Holdout.N > 0 {
		v := float32(diff.Holdout.Mean)
		meanHold = &v
	}
	if diff.Treatment.N > 0 || diff.Holdout.N > 0 {
		v := float32(diff.Diff)
		diffPtr = &v
		se := float32(diff.StdError)
		sePtr = &se
	}
	if diff.MeanMasteryTreat != nil {
		v := float32(*diff.MeanMasteryTreat)
		mTreat = &v
	}
	if diff.MeanMasteryHoldout != nil {
		v := float32(*diff.MeanMasteryHoldout)
		mHold = &v
	}

	if err := acrepo.UpsertEffectivenessCache(ctx, pool, acrepo.EffectivenessCacheRow{
		UnitID:                    unit.ID,
		CourseID:                  unit.CourseID,
		NTreatment:                diff.Treatment.N,
		NHoldout:                  diff.Holdout.N,
		MeanLiftTreatment:         meanTreat,
		MeanLiftHoldout:           meanHold,
		TreatmentMinusHoldout:     diffPtr,
		DiffStdError:              sePtr,
		MeanMasteryDeltaTreatment: mTreat,
		MeanMasteryDeltaHoldout:   mHold,
		Verdict:                   diff.Verdict,
	}); err != nil {
		return "", err
	}

	modes := AggregateByMode(samples)
	modeRows := make([]acrepo.ModeEffectivenessRow, 0, len(modes))
	for _, m := range modes {
		modeRows = append(modeRows, acrepo.ModeEffectivenessRow{
			UnitID: unit.ID, EmphasisMode: m.EmphasisMode, N: m.N, MeanLift: m.MeanLift,
		})
	}
	if err := acrepo.ReplaceModeEffectiveness(ctx, pool, unit.ID, modeRows); err != nil {
		return "", err
	}

	variants := AggregateByVariant(samples)
	varRows := make([]acrepo.VariantEffectivenessRow, 0, len(variants))
	for _, v := range variants {
		varRows = append(varRows, acrepo.VariantEffectivenessRow{
			UnitID: unit.ID, VariantID: v.VariantID, N: v.N, MeanLift: v.MeanLift,
		})
	}
	if err := acrepo.ReplaceVariantEffectiveness(ctx, pool, unit.ID, varRows); err != nil {
		return "", err
	}

	if meanTreat != nil {
		SetUnitMeanLift(float64(*meanTreat))
	}
	if diffPtr != nil {
		SetTreatmentMinusHoldout(float64(*diffPtr))
	}

	if diff.Verdict == VerdictRegressing && prev != VerdictRegressing {
		IncVerdictRegressing()
		notifyRegressing(ctx, pool, unit, notify)
		actor := unit.CreatedBy
		uid := unit.ID
		_ = acrepo.InsertEvent(ctx, pool, unit.CourseID, &uid, &actor, nil, EventVerdictChanged, map[string]any{
			"unitId":       unit.ID,
			"verdict":      diff.Verdict,
			"previous":     prev,
			"nTreatment":   diff.Treatment.N,
			"nHoldout":     diff.Holdout.N,
			"treatmentMinusHoldout": diff.Diff,
		})
	}

	return diff.Verdict, nil
}

func notifyRegressing(ctx context.Context, pool *pgxpool.Pool, unit acrepo.UnitRow, notify *EffectivenessNotifyDeps) {
	if notify == nil || notify.Pool == nil {
		return
	}
	instructors, err := atrisk.ListInstructorUserIDs(ctx, pool, unit.CourseID)
	if err != nil {
		slog.Error("adaptivecontent: list instructors for regressing alert failed", "err", err)
		return
	}
	code, _ := acrepo.CourseCodeForID(ctx, pool, unit.CourseID)
	actionURL := "/courses"
	if code != "" {
		actionURL = "/courses/" + code + "/settings?tab=adaptive-content"
	}
	title := "Adaptive unit may be hurting learning"
	body := "An adaptive content unit is underperforming the holdout group. Review variants and consider pausing the unit."
	push := &notifications.PushService{Pool: notify.Pool, Config: notify.Config, SSEHub: notify.SSEHub}
	for _, uid := range instructors {
		if err := push.Enqueue(ctx, uid, notifications.EventAdaptiveContentRegressing, title, body, actionURL); err != nil {
			slog.Warn("adaptivecontent: regressing notify failed", "user_id", uid, "err", err)
		}
	}
}

// RefreshCourse refreshes effectiveness for all units with a post-assessment in the course.
func RefreshCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, notify *EffectivenessNotifyDeps) (int, error) {
	if KillSwitchEngaged() {
		return 0, nil
	}
	enabled, err := acrepo.AdaptiveContentEnabledForCourse(ctx, pool, courseID)
	if err != nil {
		return 0, err
	}
	if !ActiveForCourse(enabled) {
		return 0, nil
	}
	units, err := acrepo.ListUnitsWithPostAssessment(ctx, pool, courseID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, u := range units {
		if _, err := RefreshUnit(ctx, pool, u, notify); err != nil {
			slog.Error("adaptivecontent: refresh unit effectiveness failed", "unit_id", u.ID, "err", err)
			continue
		}
		n++
	}
	// AC.9: keep coverage + course report matview in sync with effectiveness refresh.
	if err := RefreshCourseReport(ctx, pool, courseID); err != nil {
		slog.Warn("adaptivecontent: course report refresh failed", "course_id", courseID, "err", err)
	}
	return n, nil
}

// RefreshAll refreshes effectiveness for every ACE-enabled course (scheduled job).
func RefreshAll(ctx context.Context, pool *pgxpool.Pool, notify *EffectivenessNotifyDeps) (int, error) {
	if KillSwitchEngaged() {
		return 0, nil
	}
	ids, err := acrepo.ListCourseIDsWithAdaptiveContentEnabled(ctx, pool)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, id := range ids {
		n, err := RefreshCourse(ctx, pool, id, notify)
		if err != nil {
			slog.Error("adaptivecontent: refresh course effectiveness failed", "course_id", id, "err", err)
			continue
		}
		total += n
	}
	return total, nil
}
