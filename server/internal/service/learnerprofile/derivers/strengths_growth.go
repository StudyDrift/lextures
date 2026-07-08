package derivers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/learnerprofile"
)

// StrengthsGrowthDeriver derives cross-course strengths and growth areas from mastery and misconceptions (LP04).
type StrengthsGrowthDeriver struct {
	Pool *pgxpool.Pool
	Now  func() time.Time
}

func (d StrengthsGrowthDeriver) Key() string { return "strengths_growth" }

func (d StrengthsGrowthDeriver) Version() int { return strengthsGrowthDeriverVersion }

func (d StrengthsGrowthDeriver) MinSignals() int { return strengthsGrowthMinConcepts }

func (d StrengthsGrowthDeriver) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func (d StrengthsGrowthDeriver) Derive(ctx context.Context, userID uuid.UUID) (learnerprofile.FacetResult, error) {
	now := d.now()

	conceptRows, err := d.loadConceptStates(ctx, userID)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	misconceptions, err := d.loadRecurringMisconceptions(ctx, userID)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}

	summary, sufficient, signalCount := computeStrengthsGrowth(strengthsGrowthComputeInput{
		ConceptRows:    conceptRows,
		Misconceptions: misconceptions,
		Now:            now,
	})
	if !sufficient {
		return learnerprofile.FacetResult{
			State:           "insufficient_data",
			Summary:         json.RawMessage(`{}`),
			Confidence:      0,
			ComputedVersion: d.Version(),
		}, nil
	}

	summaryJSON, _ := json.Marshal(summary)
	confidence := strengthsGrowthConfidence(signalCount, summary)
	insights := buildStrengthsGrowthInsights(summary, conceptRows, misconceptions, now)

	return learnerprofile.FacetResult{
		State:           "ok",
		Summary:         summaryJSON,
		Confidence:      confidence,
		ComputedVersion: d.Version(),
		Insights:        insights,
	}, nil
}

func (d StrengthsGrowthDeriver) loadConceptStates(ctx context.Context, userID uuid.UUID) ([]conceptCourseRow, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT
    c.slug,
    c.name,
    c.id,
    c.course_id,
    (c.decay_lambda)::float8,
    (s.mastery)::float8,
    s.attempt_count,
    s.last_seen_at,
    s.needs_review_at
FROM course.learner_concept_states s
INNER JOIN course.concepts c ON c.id = s.concept_id
INNER JOIN course.courses co ON co.id = c.course_id
INNER JOIN course.course_enrollments ce ON ce.course_id = co.id AND ce.user_id = s.user_id
WHERE s.user_id = $1
  AND co.archived = false
  AND (ce.active OR ce.state = 'active')
ORDER BY c.slug, c.name
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []conceptCourseRow
	for rows.Next() {
		var row conceptCourseRow
		if err := rows.Scan(
			&row.Slug, &row.Name, &row.ConceptID, &row.CourseID,
			&row.DecayLambda, &row.StoredMastery, &row.AttemptCount,
			&row.LastSeenAt, &row.NeedsReviewAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (d StrengthsGrowthDeriver) loadRecurringMisconceptions(ctx context.Context, userID uuid.UUID) ([]misconceptionRow, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT
    m.id,
    m.name,
    m.description,
    c.name,
    m.course_id,
    COUNT(*)::bigint AS trigger_count
FROM course.misconception_events e
INNER JOIN course.misconceptions m ON m.id = e.misconception_id
LEFT JOIN course.concepts c ON c.id = m.concept_id
INNER JOIN course.course_enrollments ce ON ce.course_id = e.course_id AND ce.user_id = e.user_id
WHERE e.user_id = $1
  AND (ce.active OR ce.state = 'active')
GROUP BY m.id, m.name, m.description, c.name, m.course_id
HAVING COUNT(*) >= $2
ORDER BY trigger_count DESC
`, userID, strengthsGrowthMisconceptionMinTriggers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []misconceptionRow
	for rows.Next() {
		var row misconceptionRow
		if err := rows.Scan(
			&row.MisconceptionID, &row.Name, &row.Description,
			&row.ConceptName, &row.CourseID, &row.TriggerCount,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func buildStrengthsGrowthInsights(
	summary StrengthsGrowthSummary,
	conceptRows []conceptCourseRow,
	misconceptions []misconceptionRow,
	now time.Time,
) []learnerprofile.InsightResult {
	var insights []learnerprofile.InsightResult
	baseEvidence := learnerprofile.EvidenceResult{
		SourceKind:       "learner_concept_state",
		SourceTable:      strengthsGrowthSourceTable,
		ObservationCount: len(conceptRows),
	}

	if len(summary.Strengths) > 0 {
		value, _ := json.Marshal(map[string]any{"strengths": summary.Strengths})
		insights = append(insights, learnerprofile.InsightResult{
			InsightKey:   "top_strengths",
			LabelI18nKey: "learner_profile.strengths_growth.top_strengths",
			Value:        value,
			Confidence:   summary.Strengths[0].Mastery,
			Salience:     100,
			Evidence:     conceptEvidenceForSummary(conceptRows, baseEvidence, now),
		})
	}
	if len(summary.Growth) > 0 {
		value, _ := json.Marshal(map[string]any{"growth": summary.Growth})
		conf := 0.0
		for _, item := range summary.Growth {
			if item.Mastery != nil {
				conf = 1 - *item.Mastery
				break
			}
		}
		evidence := conceptEvidenceForSummary(conceptRows, baseEvidence, now)
		evidence = append(evidence, misconceptionEvidence(misconceptions)...)
		insights = append(insights, learnerprofile.InsightResult{
			InsightKey:   "growth_areas",
			LabelI18nKey: "learner_profile.strengths_growth.growth_areas",
			Value:        value,
			Confidence:   round2(conf),
			Salience:     90,
			Evidence:     evidence,
		})
	}
	if len(summary.NeedsReview) > 0 {
		value, _ := json.Marshal(map[string]any{"needsReview": summary.NeedsReview})
		insights = append(insights, learnerprofile.InsightResult{
			InsightKey:   "needs_review",
			LabelI18nKey: "learner_profile.strengths_growth.needs_review",
			Value:        value,
			Confidence:   0.75,
			Salience:     80,
			Evidence:     conceptEvidenceForSummary(conceptRows, baseEvidence, now),
		})
	}
	return insights
}

func conceptEvidenceForSummary(rows []conceptCourseRow, base learnerprofile.EvidenceResult, now time.Time) []learnerprofile.EvidenceResult {
	evidence := []learnerprofile.EvidenceResult{base}
	if len(rows) == 0 {
		return evidence
	}
	totalAttempts := int32(0)
	for _, row := range rows {
		totalAttempts += row.AttemptCount
	}
	byCourse := make(map[uuid.UUID]int32)
	for _, row := range rows {
		byCourse[row.CourseID] += row.AttemptCount
	}
	for courseID, attempts := range byCourse {
		ev := base
		ev.CourseID = &courseID
		ev.ObservationCount = int(attempts)
		contrib := 1.0
		if totalAttempts > 0 {
			contrib = round2(float64(attempts) / float64(totalAttempts))
		}
		ev.Contribution = &contrib
		if ws, we := conceptWindowBounds(rows, now); ws != nil && we != nil {
			ev.WindowStart = ws
			ev.WindowEnd = we
		}
		evidence = append(evidence, ev)
	}
	return evidence
}

func misconceptionEvidence(rows []misconceptionRow) []learnerprofile.EvidenceResult {
	if len(rows) == 0 {
		return nil
	}
	total := int64(0)
	for _, row := range rows {
		total += row.TriggerCount
	}
	out := make([]learnerprofile.EvidenceResult, 0, len(rows))
	for _, row := range rows {
		ev := learnerprofile.EvidenceResult{
			SourceKind:       "misconception_event",
			SourceTable:      strengthsGrowthMisconceptionTable,
			CourseID:         &row.CourseID,
			ObservationCount: int(row.TriggerCount),
		}
		contrib := 1.0
		if total > 0 {
			contrib = round2(float64(row.TriggerCount) / float64(total))
		}
		ev.Contribution = &contrib
		sample, _ := json.Marshal(map[string]string{
			"misconceptionId": row.MisconceptionID.String(),
			"name":              row.Name,
		})
		ev.SampleRefs = sample
		out = append(out, ev)
	}
	return out
}

func conceptWindowBounds(rows []conceptCourseRow, now time.Time) (*time.Time, *time.Time) {
	var earliest, latest *time.Time
	for _, row := range rows {
		if row.LastSeenAt == nil {
			continue
		}
		t := row.LastSeenAt.UTC()
		if earliest == nil || t.Before(*earliest) {
			earliest = &t
		}
		if latest == nil || t.After(*latest) {
			latest = &t
		}
	}
	if earliest == nil {
		return nil, nil
	}
	end := now.UTC()
	return earliest, &end
}