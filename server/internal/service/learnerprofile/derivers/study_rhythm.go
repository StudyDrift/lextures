package derivers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	userrepo "github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/learnerprofile"
)

// StudyRhythmDeriver derives the study_rhythm facet from engagement events and login audit (LP02).
type StudyRhythmDeriver struct {
	Pool *pgxpool.Pool
	Now  func() time.Time
}

func (d StudyRhythmDeriver) Key() string { return "study_rhythm" }

func (d StudyRhythmDeriver) Version() int { return studyRhythmDeriverVersion }

func (d StudyRhythmDeriver) MinSignals() int { return studyRhythmMinActiveDays }

func (d StudyRhythmDeriver) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func (d StudyRhythmDeriver) Derive(ctx context.Context, userID uuid.UUID) (learnerprofile.FacetResult, error) {
	now := d.now()
	windowEnd := now
	windowStart := now.AddDate(0, 0, -studyRhythmWindowDays)

	loc, err := d.loadTimezone(ctx, userID)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}

	events, err := d.loadRhythmEvents(ctx, userID, windowStart, windowEnd)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	activeDays, err := d.loadActiveDays(ctx, userID, windowStart, loc)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}

	if len(activeDays) < studyRhythmMinActiveDays {
		return learnerprofile.FacetResult{
			State:           "insufficient_data",
			Summary:         json.RawMessage(`{}`),
			Confidence:      0,
			ComputedVersion: d.Version(),
		}, nil
	}

	summary, eventCount, sessionCount := computeStudyRhythm(rhythmComputeInput{
		Events:      events,
		ActiveDays:  activeDays,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Now:         now,
		Loc:         loc,
	})
	summaryJSON, _ := json.Marshal(summary)
	confidence := rhythmConfidence(len(activeDays), eventCount)

	ws := windowStart
	we := windowEnd
	evidenceBase := learnerprofile.EvidenceResult{
		SourceKind:       "engagement_event",
		SourceTable:      studyRhythmSourceTable,
		ObservationCount: eventCount,
		WindowStart:      &ws,
		WindowEnd:        &we,
	}

	activeDayCount := len(activeDays)
	insights := []learnerprofile.InsightResult{
		buildPeakWindowInsight(summary, events, evidenceBase),
		buildConsistencyInsight(summary, activeDayCount, evidenceBase),
		buildStreakInsight(summary, activeDays, evidenceBase),
		buildSessionInsight(summary, sessionCount, activeDayCount, evidenceBase),
	}

	return learnerprofile.FacetResult{
		State:           "ok",
		Summary:         summaryJSON,
		Confidence:      confidence,
		ComputedVersion: d.Version(),
		Insights:        insights,
	}, nil
}

func (d StudyRhythmDeriver) loadTimezone(ctx context.Context, userID uuid.UUID) (*time.Location, error) {
	tz, err := userrepo.GetTimezone(ctx, d.Pool, userID)
	if err != nil {
		return nil, err
	}
	if tz == nil || *tz == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(*tz)
	if err != nil {
		return time.UTC, nil
	}
	return loc, nil
}

func (d StudyRhythmDeriver) loadRhythmEvents(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]rhythmEvent, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT occurred_at, course_id::text
FROM analytics.engagement_events
WHERE user_id = $1
  AND occurred_at >= $2 AND occurred_at <= $3
  AND event_type IN ('heartbeat', 'time_on_task')
ORDER BY occurred_at
`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []rhythmEvent
	for rows.Next() {
		var ev rhythmEvent
		var courseID *string
		if err := rows.Scan(&ev.At, &courseID); err != nil {
			return nil, err
		}
		ev.At = ev.At.UTC()
		ev.CourseID = courseID
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (d StudyRhythmDeriver) loadActiveDays(ctx context.Context, userID uuid.UUID, since time.Time, loc *time.Location) ([]time.Time, error) {
	tzName := loc.String()
	rows, err := d.Pool.Query(ctx, `
SELECT DISTINCT (occurred_at AT TIME ZONE $3)::date
FROM analytics.engagement_events
WHERE user_id = $1
  AND occurred_at >= $2
  AND event_type IN ('heartbeat', 'time_on_task')
UNION
SELECT DISTINCT (occurred_at AT TIME ZONE $3)::date
FROM "user".user_audit
WHERE user_id = $1 AND occurred_at >= $2
`, userID, since, tzName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc))
	}
	return out, rows.Err()
}

func buildPeakWindowInsight(summary RhythmSummary, events []rhythmEvent, base learnerprofile.EvidenceResult) learnerprofile.InsightResult {
	value, _ := json.Marshal(map[string]any{"peakWindows": summary.PeakWindows})
	evidence := []learnerprofile.EvidenceResult{base}
	if base.ObservationCount > 0 {
		for courseKey, count := range courseEventCounts(events) {
			if count == 0 || courseKey == "" {
				continue
			}
			id, err := uuid.Parse(courseKey)
			if err != nil {
				continue
			}
			ev := base
			ev.ObservationCount = count
			ev.CourseID = &id
			contrib := round2(float64(count) / float64(base.ObservationCount))
			ev.Contribution = &contrib
			evidence = append(evidence, ev)
		}
	}
	confidence := 0.0
	if len(summary.PeakWindows) > 0 {
		confidence = summary.PeakWindows[0].Share
	}
	return learnerprofile.InsightResult{
		InsightKey:   "peak_study_window",
		LabelI18nKey: "learner_profile.study_rhythm.peak_study_window",
		Value:        value,
		Confidence:   confidence,
		Salience:     100,
		Evidence:     evidence,
	}
}

func buildConsistencyInsight(summary RhythmSummary, activeDayCount int, base learnerprofile.EvidenceResult) learnerprofile.InsightResult {
	value, _ := json.Marshal(map[string]any{
		"consistencyScore":  summary.ConsistencyScore,
		"activeDaysPerWeek": summary.ActiveDaysPerWeek,
	})
	ev := base
	ev.ObservationCount = activeDayCount
	ev.SourceKind = "engagement_event"
	contrib := 1.0
	ev.Contribution = &contrib
	return learnerprofile.InsightResult{
		InsightKey:   "study_consistency",
		LabelI18nKey: "learner_profile.study_rhythm.study_consistency",
		Value:        value,
		Confidence:   summary.ConsistencyScore,
		Salience:     80,
		Evidence:     []learnerprofile.EvidenceResult{ev},
	}
}

func buildStreakInsight(summary RhythmSummary, activeDays []time.Time, base learnerprofile.EvidenceResult) learnerprofile.InsightResult {
	value, _ := json.Marshal(map[string]any{
		"currentStreakDays": summary.CurrentStreakDays,
		"longestStreakDays": summary.LongestStreakDays,
	})
	ev := base
	ev.SourceKind = "engagement_event"
	ev.SourceTable = studyRhythmSourceTable
	ev.ObservationCount = len(activeDays)
	contrib := 1.0
	ev.Contribution = &contrib
	auditEv := base
	auditEv.SourceKind = "login_audit"
	auditEv.SourceTable = studyRhythmAuditSourceTable
	auditEv.ObservationCount = len(activeDays)
	auditEv.Contribution = &contrib
	return learnerprofile.InsightResult{
		InsightKey:   "study_streak",
		LabelI18nKey: "learner_profile.study_rhythm.study_streak",
		Value:        value,
		Confidence:   rhythmConfidence(len(activeDays), base.ObservationCount),
		Salience:     70,
		Evidence:     []learnerprofile.EvidenceResult{ev, auditEv},
	}
}

func buildSessionInsight(summary RhythmSummary, sessionCount, activeDayCount int, base learnerprofile.EvidenceResult) learnerprofile.InsightResult {
	value, _ := json.Marshal(map[string]any{
		"medianSessionMin":      summary.MedianSessionMin,
		"sessionsPerActiveWeek": summary.SessionsPerActiveWeek,
	})
	ev := base
	ev.ObservationCount = sessionCount
	contrib := 1.0
	ev.Contribution = &contrib
	return learnerprofile.InsightResult{
		InsightKey:   "session_shape",
		LabelI18nKey: "learner_profile.study_rhythm.session_shape",
		Value:        value,
		Confidence:   rhythmConfidence(activeDayCount, sessionCount),
		Salience:     60,
		Evidence:     []learnerprofile.EvidenceResult{ev},
	}
}

