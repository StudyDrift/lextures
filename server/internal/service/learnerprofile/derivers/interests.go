package derivers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/learnerprofile"
)

const globalStudentNotebookKey = "__lextures_global__"

// InterestsDeriver derives topic affinity from enrollments, notebooks, deep reads, and feed activity (LP05).
type InterestsDeriver struct {
	Pool *pgxpool.Pool
	Now  func() time.Time
}

func (d InterestsDeriver) Key() string { return "interests" }

func (d InterestsDeriver) Version() int { return interestsDeriverVersion }

func (d InterestsDeriver) MinSignals() int { return interestsMinTopics * interestsMinSignalsPerTopic }

func (d InterestsDeriver) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func (d InterestsDeriver) Derive(ctx context.Context, userID uuid.UUID) (learnerprofile.FacetResult, error) {
	now := d.now()
	windowEnd := now
	windowStart := now.AddDate(0, 0, -interestsWindowDays)

	courseTopics, err := d.loadEnrolledCourseTopics(ctx, userID)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	notebookSignals, err := d.loadNotebookSignals(ctx, userID, courseTopics)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	taskSignals, err := d.loadNotebookTaskSignals(ctx, userID, courseTopics)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	readingSignals, err := d.loadDeepReadingSignals(ctx, userID, windowStart, windowEnd, courseTopics)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	feedSignals, err := d.loadFeedSignals(ctx, userID, windowStart, windowEnd, courseTopics)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}

	signals := make([]rawInterestSignal, 0,
		len(courseTopics)+len(notebookSignals)+len(taskSignals)+len(readingSignals)+len(feedSignals))
	for _, row := range courseTopics {
		if row.Topic == "" {
			continue
		}
		signals = append(signals, rawInterestSignal{
			Topic:        row.Topic,
			Kind:         signalCourses,
			Weight:       interestsAssignedWeight,
			SelfDirected: false,
			CourseID:     row.CourseID,
			Ref:          row.CourseCode,
		})
	}
	signals = append(signals, notebookSignals...)
	signals = append(signals, taskSignals...)
	signals = append(signals, readingSignals...)
	signals = append(signals, feedSignals...)

	summary, sufficient := computeInterests(signals)
	if !sufficient {
		return learnerprofile.FacetResult{
			State:           "insufficient_data",
			Summary:         json.RawMessage(`{}`),
			Confidence:      0,
			ComputedVersion: d.Version(),
		}, nil
	}

	summaryJSON, _ := json.Marshal(summary)
	qualifiedTopics := countQualifiedTopics(signals)
	confidence := interestsConfidence(summary, qualifiedTopics)
	insights := buildInterestsInsights(summary, signals, windowStart, windowEnd)

	return learnerprofile.FacetResult{
		State:           "ok",
		Summary:         summaryJSON,
		Confidence:      confidence,
		ComputedVersion: d.Version(),
		Insights:        insights,
	}, nil
}

type courseTopicRow struct {
	CourseID   string
	CourseCode string
	Topic      string
}

func (d InterestsDeriver) loadEnrolledCourseTopics(ctx context.Context, userID uuid.UUID) ([]courseTopicRow, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT
    c.id::text,
    c.course_code,
    COALESCE(
        NULLIF(TRIM(c.catalog_category), ''),
        NULLIF(TRIM(cs.subject), ''),
        NULLIF(TRIM(cs.department), '')
    ) AS topic
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
LEFT JOIN catalog.catalog_sections cs ON cs.lms_course_id = c.id AND cs.status = 'active'
WHERE ce.user_id = $1
  AND (ce.active OR ce.state = 'active')
  AND c.archived = false
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []courseTopicRow
	for rows.Next() {
		var row courseTopicRow
		if err := rows.Scan(&row.CourseID, &row.CourseCode, &row.Topic); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type notebookPage struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	ParentID  *string `json:"parentId"`
	Kind      string  `json:"kind"`
	ContentMd string  `json:"contentMd"`
}

type notebookStore struct {
	Pages []notebookPage `json:"pages"`
}

func (d InterestsDeriver) loadNotebookSignals(ctx context.Context, userID uuid.UUID, courseTopics []courseTopicRow) ([]rawInterestSignal, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT course_code, data
FROM analytics.student_notebooks
WHERE user_id = $1
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	courseTopicByCode := make(map[string]courseTopicRow, len(courseTopics))
	for _, row := range courseTopics {
		courseTopicByCode[row.CourseCode] = row
	}

	var out []rawInterestSignal
	for rows.Next() {
		var courseCode string
		var data []byte
		if err := rows.Scan(&courseCode, &data); err != nil {
			return nil, err
		}
		global := courseCode == globalStudentNotebookKey
		weight := interestsNotebookCourseWeight
		if global {
			weight = interestsNotebookGlobalWeight
		}

		var store notebookStore
		if err := json.Unmarshal(data, &store); err != nil {
			continue
		}
		if len(store.Pages) == 0 {
			continue
		}

		groupTopics := make(map[string]string)
		for _, page := range store.Pages {
			if page.Kind == "group" {
				title := normalizeTopicLabel(page.Title)
				if title != "" {
					groupTopics[page.ID] = title
				}
			}
		}

		fallbackTopic := ""
		if !global {
			if row, ok := courseTopicByCode[courseCode]; ok && row.Topic != "" {
				fallbackTopic = row.Topic
			}
		}

		for _, page := range store.Pages {
			if page.Kind == "group" {
				topic := groupTopics[page.ID]
				if topic == "" {
					continue
				}
				out = append(out, rawInterestSignal{
					Topic:        topic,
					Kind:         signalNotebooks,
					Weight:       weight,
					SelfDirected: true,
					Ref:          courseCode + ":" + page.ID,
				})
				continue
			}
			topic := topicForNotebookPage(page, store.Pages, groupTopics, fallbackTopic)
			if topic == "" {
				continue
			}
			if strings.TrimSpace(page.ContentMd) == "" {
				continue
			}
			courseID := ""
			if row, ok := courseTopicByCode[courseCode]; ok {
				courseID = row.CourseID
			}
			out = append(out, rawInterestSignal{
				Topic:        topic,
				Kind:         signalNotebooks,
				Weight:       weight,
				SelfDirected: true,
				CourseID:     courseID,
				Ref:          courseCode + ":" + page.ID,
			})
		}
	}
	return out, rows.Err()
}

func topicForNotebookPage(page notebookPage, pages []notebookPage, groupTopics map[string]string, fallbackTopic string) string {
	if page.ParentID != nil {
		if topic := ancestorGroupTopic(*page.ParentID, pages, groupTopics); topic != "" {
			return topic
		}
	}
	return fallbackTopic
}

func ancestorGroupTopic(parentID string, pages []notebookPage, groupTopics map[string]string) string {
	byID := make(map[string]notebookPage, len(pages))
	for _, p := range pages {
		byID[p.ID] = p
	}
	cur := parentID
	for cur != "" {
		if topic, ok := groupTopics[cur]; ok && topic != "" {
			return topic
		}
		parent, ok := byID[cur]
		if !ok || parent.ParentID == nil {
			return ""
		}
		cur = *parent.ParentID
	}
	return ""
}

func (d InterestsDeriver) loadNotebookTaskSignals(ctx context.Context, userID uuid.UUID, courseTopics []courseTopicRow) ([]rawInterestSignal, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT course_code, notebook_page_id
FROM analytics.student_notebook_tasks
WHERE user_id = $1
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notebookData := make(map[string]notebookStore)
	notebookRows, err := d.Pool.Query(ctx, `
SELECT course_code, data FROM analytics.student_notebooks WHERE user_id = $1
`, userID)
	if err != nil {
		return nil, err
	}
	for notebookRows.Next() {
		var courseCode string
		var data []byte
		if err := notebookRows.Scan(&courseCode, &data); err != nil {
			notebookRows.Close()
			return nil, err
		}
		var store notebookStore
		if err := json.Unmarshal(data, &store); err == nil {
			notebookData[courseCode] = store
		}
	}
	notebookRows.Close()
	if err := notebookRows.Err(); err != nil {
		return nil, err
	}

	courseTopicByCode := make(map[string]courseTopicRow, len(courseTopics))
	for _, row := range courseTopics {
		courseTopicByCode[row.CourseCode] = row
	}

	var out []rawInterestSignal
	for rows.Next() {
		var courseCode, pageID string
		if err := rows.Scan(&courseCode, &pageID); err != nil {
			return nil, err
		}
		global := courseCode == globalStudentNotebookKey
		weight := interestsTaskWeight
		store := notebookData[courseCode]
		groupTopics := make(map[string]string)
		for _, page := range store.Pages {
			if page.Kind == "group" {
				title := normalizeTopicLabel(page.Title)
				if title != "" {
					groupTopics[page.ID] = title
				}
			}
		}
		fallbackTopic := ""
		if !global {
			if row, ok := courseTopicByCode[courseCode]; ok {
				fallbackTopic = row.Topic
			}
		}
		topic := ""
		for _, page := range store.Pages {
			if page.ID == pageID {
				topic = topicForNotebookPage(page, store.Pages, groupTopics, fallbackTopic)
				break
			}
		}
		if topic == "" {
			continue
		}
		courseID := ""
		if row, ok := courseTopicByCode[courseCode]; ok {
			courseID = row.CourseID
		}
		out = append(out, rawInterestSignal{
			Topic:        topic,
			Kind:         signalTasks,
			Weight:       weight,
			SelfDirected: true,
			CourseID:     courseID,
			Ref:          courseCode + ":" + pageID,
		})
	}
	return out, rows.Err()
}

func (d InterestsDeriver) loadDeepReadingSignals(
	ctx context.Context,
	userID uuid.UUID,
	windowStart, windowEnd time.Time,
	courseTopics []courseTopicRow,
) ([]rawInterestSignal, error) {
	modality := ContentModalityDeriver{Pool: d.Pool, Now: d.Now}
	rawEvents, err := modality.loadEngagementEvents(ctx, userID, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}
	items, err := modality.buildItemEngagement(ctx, userID, rawEvents)
	if err != nil {
		return nil, err
	}

	topicByCourseID := make(map[string]string, len(courseTopics))
	for _, row := range courseTopics {
		if row.Topic != "" {
			topicByCourseID[row.CourseID] = row.Topic
		}
	}

	var out []rawInterestSignal
	for _, item := range items {
		if item.Modality != modalityReading {
			continue
		}
		if item.engagementScore() < contentModalityThoroughThreshold {
			continue
		}
		topic := topicByCourseID[item.CourseKey]
		if topic == "" {
			continue
		}
		out = append(out, rawInterestSignal{
			Topic:        topic,
			Kind:         signalReading,
			Weight:       interestsReadingWeight,
			SelfDirected: true,
			CourseID:     item.CourseKey,
			Ref:          item.ItemKey,
		})
	}
	return out, nil
}

func (d InterestsDeriver) loadFeedSignals(
	ctx context.Context,
	userID uuid.UUID,
	windowStart, windowEnd time.Time,
	courseTopics []courseTopicRow,
) ([]rawInterestSignal, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT fm.id::text, fc.course_id::text
FROM course.feed_messages fm
INNER JOIN course.feed_channels fc ON fc.id = fm.channel_id
INNER JOIN course.course_enrollments ce ON ce.course_id = fc.course_id AND ce.user_id = fm.author_user_id
WHERE fm.author_user_id = $1
  AND fm.created_at >= $2
  AND fm.created_at <= $3
  AND (ce.active OR ce.state = 'active')
`, userID, windowStart, windowEnd)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topicByCourseID := make(map[string]string, len(courseTopics))
	for _, row := range courseTopics {
		if row.Topic != "" {
			topicByCourseID[row.CourseID] = row.Topic
		}
	}

	var out []rawInterestSignal
	for rows.Next() {
		var messageID, courseID string
		if err := rows.Scan(&messageID, &courseID); err != nil {
			return nil, err
		}
		topic := topicByCourseID[courseID]
		if topic == "" {
			continue
		}
		out = append(out, rawInterestSignal{
			Topic:        topic,
			Kind:         signalFeed,
			Weight:       interestsFeedWeight,
			SelfDirected: true,
			CourseID:     courseID,
			Ref:          messageID,
		})
	}
	return out, rows.Err()
}

func countQualifiedTopics(signals []rawInterestSignal) int {
	counts := make(map[string]int)
	for _, sig := range signals {
		label := normalizeTopicLabel(sig.Topic)
		if label == "" {
			continue
		}
		counts[topicKey(label)]++
	}
	n := 0
	for _, count := range counts {
		if count >= interestsMinSignalsPerTopic {
			n++
		}
	}
	return n
}

func buildInterestsInsights(summary InterestsSummary, signals []rawInterestSignal, windowStart, windowEnd time.Time) []learnerprofile.InsightResult {
	byTopicKey := aggregateSignalsByTopic(signals)
	ws := windowStart
	we := windowEnd
	insights := make([]learnerprofile.InsightResult, 0, len(summary.Topics))
	for i, topic := range summary.Topics {
		key := topicKey(topic.Topic)
		acc := byTopicKey[key]
		value, _ := json.Marshal(topic)
		salience := 100 - i*5
		if salience < 10 {
			salience = 10
		}
		insights = append(insights, learnerprofile.InsightResult{
			InsightKey:   "topic_" + slugifyTopicKey(key),
			LabelI18nKey: "learner_profile.interests.topic",
			Value:        value,
			Confidence:   topic.Affinity,
			Salience:     salience,
			Evidence:     topicEvidenceRows(acc, ws, we),
		})
	}
	return insights
}

func aggregateSignalsByTopic(signals []rawInterestSignal) map[string]*topicAccumulator {
	out := make(map[string]*topicAccumulator)
	for _, sig := range signals {
		label := normalizeTopicLabel(sig.Topic)
		if label == "" {
			continue
		}
		key := topicKey(label)
		acc, ok := out[key]
		if !ok {
			acc = &topicAccumulator{
				Label:          label,
				EvidenceByKind: make(map[interestSignalKind]topicEvidenceAcc),
			}
			out[key] = acc
		}
		acc.WeightedScore += sig.Weight
		ev := acc.EvidenceByKind[sig.Kind]
		ev.Count++
		if ev.CourseIDs == nil {
			ev.CourseIDs = make(map[string]struct{})
		}
		if sig.CourseID != "" {
			ev.CourseIDs[sig.CourseID] = struct{}{}
		}
		if sig.Ref != "" && len(ev.Refs) < 8 {
			ev.Refs = append(ev.Refs, sig.Ref)
		}
		acc.EvidenceByKind[sig.Kind] = ev
	}
	return out
}

func topicEvidenceRows(acc *topicAccumulator, windowStart, windowEnd time.Time) []learnerprofile.EvidenceResult {
	if acc == nil {
		contrib := 1.0
		return []learnerprofile.EvidenceResult{{
			SourceKind:       "interests_signal",
			SourceTable:      interestsSourceTable,
			ObservationCount: 0,
			Contribution:     &contrib,
			WindowStart:      &windowStart,
			WindowEnd:        &windowEnd,
		}}
	}
	kinds := []interestSignalKind{signalNotebooks, signalReading, signalFeed, signalCourses, signalTasks}
	out := make([]learnerprofile.EvidenceResult, 0, len(kinds))
	total := 0
	for _, kind := range kinds {
		total += acc.EvidenceByKind[kind].Count
	}
	for _, kind := range kinds {
		evAcc := acc.EvidenceByKind[kind]
		if evAcc.Count == 0 {
			continue
		}
		sourceTable := interestsSourceTable
		switch kind {
		case signalCourses:
			sourceTable = "course.course_enrollments"
		case signalNotebooks:
			sourceTable = "analytics.student_notebooks"
		case signalReading:
			sourceTable = contentModalitySourceTable
		case signalFeed:
			sourceTable = "course.feed_messages"
		case signalTasks:
			sourceTable = "analytics.student_notebook_tasks"
		}
		row := learnerprofile.EvidenceResult{
			SourceKind:       string(kind),
			SourceTable:      sourceTable,
			ObservationCount: evAcc.Count,
			WindowStart:      &windowStart,
			WindowEnd:        &windowEnd,
		}
		contrib := 1.0
		if total > 0 {
			contrib = round2(float64(evAcc.Count) / float64(total))
		}
		row.Contribution = &contrib
		if len(evAcc.CourseIDs) > 0 || len(evAcc.Refs) > 0 {
			sample := map[string]any{}
			if len(evAcc.CourseIDs) > 0 {
				courses := make([]string, 0, len(evAcc.CourseIDs))
				for id := range evAcc.CourseIDs {
					courses = append(courses, id)
				}
				sample["courseIds"] = courses
			}
			if len(evAcc.Refs) > 0 {
				sample["refs"] = evAcc.Refs
			}
			sample["topic"] = acc.Label
			sampleBytes, _ := json.Marshal(sample)
			row.SampleRefs = sampleBytes
		}
		if len(evAcc.CourseIDs) == 1 {
			for id := range evAcc.CourseIDs {
				if parsed, err := uuid.Parse(id); err == nil {
					row.CourseID = &parsed
				}
			}
		}
		out = append(out, row)
	}
	if len(out) == 0 {
		contrib := 1.0
		out = append(out, learnerprofile.EvidenceResult{
			SourceKind:       "interests_signal",
			SourceTable:      interestsSourceTable,
			ObservationCount: 0,
			Contribution:     &contrib,
			WindowStart:      &windowStart,
			WindowEnd:        &windowEnd,
		})
	}
	return out
}

func slugifyTopicKey(key string) string {
	key = strings.ReplaceAll(key, " ", "_")
	var b strings.Builder
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "unknown"
	}
	return out
}