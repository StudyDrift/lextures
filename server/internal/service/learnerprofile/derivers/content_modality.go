package derivers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/learnerprofile"
)

// ContentModalityDeriver derives content & modality preferences from engagement events (LP03).
type ContentModalityDeriver struct {
	Pool *pgxpool.Pool
	Now  func() time.Time
}

func (d ContentModalityDeriver) Key() string { return "content_modality" }

func (d ContentModalityDeriver) Version() int { return contentModalityDeriverVersion }

func (d ContentModalityDeriver) MinSignals() int { return contentModalityMinDistinctItems }

func (d ContentModalityDeriver) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func (d ContentModalityDeriver) Derive(ctx context.Context, userID uuid.UUID) (learnerprofile.FacetResult, error) {
	now := d.now()
	windowEnd := now
	windowStart := now.AddDate(0, 0, -contentModalityWindowDays)

	rawEvents, err := d.loadEngagementEvents(ctx, userID, windowStart, windowEnd)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	items, err := d.buildItemEngagement(ctx, userID, rawEvents)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}

	summary, sufficient, modalityCounts := computeContentModality(items)
	if !sufficient {
		return learnerprofile.FacetResult{
			State:           "insufficient_data",
			Summary:         json.RawMessage(`{}`),
			Confidence:      0,
			ComputedVersion: d.Version(),
		}, nil
	}

	summaryJSON, _ := json.Marshal(summary)
	totalItems := len(items)
	modalityCount := len(modalityCounts)
	confidence := modalityConfidence(totalItems, modalityCount)

	ws := windowStart
	we := windowEnd
	evidenceBase := learnerprofile.EvidenceResult{
		SourceKind:       "engagement_event",
		SourceTable:      contentModalitySourceTable,
		ObservationCount: len(rawEvents),
		WindowStart:      &ws,
		WindowEnd:        &we,
	}

	insights := []learnerprofile.InsightResult{
		buildModalityAffinityInsight(summary, modalityCounts, evidenceBase),
		buildContentPacingInsight(summary, items, evidenceBase),
	}
	if comfortInsight := buildComplexityComfortInsight(summary, items, evidenceBase); comfortInsight != nil {
		insights = append(insights, *comfortInsight)
	}

	return learnerprofile.FacetResult{
		State:           "ok",
		Summary:         summaryJSON,
		Confidence:      confidence,
		ComputedVersion: d.Version(),
		Insights:        insights,
	}, nil
}

type rawEngagementEvent struct {
	ItemID    uuid.UUID
	ItemType  string
	EventType string
	Value     *float64
	CourseID  *uuid.UUID
}

type modalityItemAgg struct {
	itemEngagement
	heartbeatCount int
}

func (d ContentModalityDeriver) loadEngagementEvents(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]rawEngagementEvent, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT item_id, item_type, event_type, value, course_id
FROM analytics.engagement_events
WHERE user_id = $1
  AND occurred_at >= $2 AND occurred_at <= $3
  AND item_id IS NOT NULL
  AND item_type IS NOT NULL
  AND event_type IN ('video_progress', 'scroll_depth', 'heartbeat', 'time_on_task')
`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []rawEngagementEvent
	for rows.Next() {
		var ev rawEngagementEvent
		if err := rows.Scan(&ev.ItemID, &ev.ItemType, &ev.EventType, &ev.Value, &ev.CourseID); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (d ContentModalityDeriver) buildItemEngagement(ctx context.Context, userID uuid.UUID, events []rawEngagementEvent) ([]itemEngagement, error) {
	byKey := make(map[string]*modalityItemAgg)

	for _, ev := range events {
		mod := mapItemTypeToModality(ev.ItemType)
		if mod == "" {
			continue
		}
		key := ev.ItemID.String() + "|" + string(mod)
		a, ok := byKey[key]
		if !ok {
			courseKey := ""
			if ev.CourseID != nil {
				courseKey = ev.CourseID.String()
			}
			a = &modalityItemAgg{itemEngagement: itemEngagement{
				ItemKey:   ev.ItemID.String(),
				Modality:  mod,
				CourseKey: courseKey,
			}}
			switch mod {
			case modalityQuiz:
				a.ExpectedDurationSec = contentModalityDefaultQuizSec
			case modalityInteractive:
				a.ExpectedDurationSec = contentModalityDefaultInteractiveSec
			}
			byKey[key] = a
		}
		if ev.Value != nil {
			switch ev.EventType {
			case "video_progress":
				if *ev.Value > a.MaxVideoPct {
					a.MaxVideoPct = *ev.Value
				}
			case "scroll_depth":
				if *ev.Value > a.MaxScrollDepth {
					a.MaxScrollDepth = *ev.Value
				}
			case "time_on_task":
				a.TimeOnTaskSec += int(*ev.Value)
			}
		}
		if ev.EventType == "heartbeat" {
			a.heartbeatCount++
		}
	}

	readingIDs := make([]uuid.UUID, 0)
	quizIDs := make([]uuid.UUID, 0)
	for _, a := range byKey {
		if a.heartbeatCount > 0 {
			a.TimeOnTaskSec += a.heartbeatCount * contentModalityHeartbeatSec
		}
		switch a.Modality {
		case modalityReading:
			id, err := uuid.Parse(a.ItemKey)
			if err == nil {
				readingIDs = append(readingIDs, id)
			}
		case modalityQuiz:
			id, err := uuid.Parse(a.ItemKey)
			if err == nil {
				quizIDs = append(quizIDs, id)
			}
		}
	}

	if len(readingIDs) > 0 {
		if err := d.applyReadingMetadata(ctx, byKey, readingIDs); err != nil {
			return nil, err
		}
	}
	if len(quizIDs) > 0 {
		if err := d.applyQuizCompletions(ctx, userID, byKey, quizIDs); err != nil {
			return nil, err
		}
	}

	out := make([]itemEngagement, 0, len(byKey))
	for _, a := range byKey {
		out = append(out, a.itemEngagement)
	}
	return out, nil
}

func (d ContentModalityDeriver) applyReadingMetadata(ctx context.Context, byKey map[string]*modalityItemAgg, itemIDs []uuid.UUID) error {
	rows, err := d.Pool.Query(ctx, `
SELECT structure_item_id, markdown, reading_level_fkgl
FROM course.module_content_pages
WHERE structure_item_id = ANY($1)
`, itemIDs)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var itemID uuid.UUID
		var markdown string
		var fkgl *float64
		if err := rows.Scan(&itemID, &markdown, &fkgl); err != nil {
			return err
		}
		key := itemID.String() + "|" + string(modalityReading)
		if a, ok := byKey[key]; ok {
			a.ReadingLevelFKGL = fkgl
			a.ExpectedDurationSec = estimateReadSeconds(countWords(markdown))
		}
	}
	return rows.Err()
}

func (d ContentModalityDeriver) applyQuizCompletions(ctx context.Context, userID uuid.UUID, byKey map[string]*modalityItemAgg, itemIDs []uuid.UUID) error {
	rows, err := d.Pool.Query(ctx, `
SELECT structure_item_id, bool_or(status = 'submitted') AS completed
FROM course.quiz_attempts
WHERE student_user_id = $1 AND structure_item_id = ANY($2)
GROUP BY structure_item_id
`, userID, itemIDs)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var itemID uuid.UUID
		var completed bool
		if err := rows.Scan(&itemID, &completed); err != nil {
			return err
		}
		key := itemID.String() + "|" + string(modalityQuiz)
		if a, ok := byKey[key]; ok {
			a.QuizCompleted = completed
		}
	}
	return rows.Err()
}

func buildModalityAffinityInsight(summary ModalitySummary, modalityCounts map[string]int, base learnerprofile.EvidenceResult) learnerprofile.InsightResult {
	value, _ := json.Marshal(map[string]any{"modalityAffinity": summary.ModalityAffinity})
	evidence := []learnerprofile.EvidenceResult{base}
	for mod, count := range modalityCounts {
		if count == 0 {
			continue
		}
		ev := base
		ev.ObservationCount = count
		contrib := round2(float64(count) / float64(base.ObservationCount))
		if base.ObservationCount == 0 {
			contrib = 1
		}
		ev.Contribution = &contrib
		sample, _ := json.Marshal(map[string]string{"modality": mod})
		ev.SampleRefs = sample
		evidence = append(evidence, ev)
	}
	confidence := 0.0
	if affinity, ok := summary.ModalityAffinity[string(modalityVideo)]; ok {
		confidence = affinity
	}
	return learnerprofile.InsightResult{
		InsightKey:   "modality_affinity",
		LabelI18nKey: "learner_profile.content_modality.modality_affinity",
		Value:        value,
		Confidence:   confidence,
		Salience:     100,
		Evidence:     evidence,
	}
}

func buildComplexityComfortInsight(summary ModalitySummary, items []itemEngagement, base learnerprofile.EvidenceResult) *learnerprofile.InsightResult {
	if summary.ComplexityComfort == nil {
		return nil
	}
	value, _ := json.Marshal(summary.ComplexityComfort)
	readingCount := 0
	for _, item := range items {
		if item.Modality == modalityReading && item.ReadingLevelFKGL != nil {
			readingCount++
		}
	}
	ev := base
	ev.SourceTable = contentModalityReadingSourceTable
	ev.ObservationCount = readingCount
	contrib := summary.ComplexityComfort.Confidence
	ev.Contribution = &contrib
	return &learnerprofile.InsightResult{
		InsightKey:   "complexity_comfort",
		LabelI18nKey: "learner_profile.content_modality.complexity_comfort",
		Value:        value,
		Confidence:   summary.ComplexityComfort.Confidence,
		Salience:     80,
		Evidence:     []learnerprofile.EvidenceResult{ev},
	}
}

func buildContentPacingInsight(summary ModalitySummary, items []itemEngagement, base learnerprofile.EvidenceResult) learnerprofile.InsightResult {
	value, _ := json.Marshal(map[string]string{"pacing": summary.Pacing})
	ev := base
	ev.ObservationCount = len(items)
	contrib := 1.0
	ev.Contribution = &contrib
	return learnerprofile.InsightResult{
		InsightKey:   "content_pacing",
		LabelI18nKey: "learner_profile.content_modality.content_pacing",
		Value:        value,
		Confidence:   modalityConfidence(len(items), len(modalityItemCounts(items))),
		Salience:     70,
		Evidence:     []learnerprofile.EvidenceResult{ev},
	}
}