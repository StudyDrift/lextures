// Package instructorinsights computes weekly "What's Working" signals for instructors (plan 9.10).
package instructorinsights

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SignalItem is one entry in the working_well or needs_attention list.
type SignalItem struct {
	ItemID         string  `json:"itemId"`
	ItemTitle      string  `json:"itemTitle"`
	ItemType       string  `json:"itemType"`
	CompletionRate float64 `json:"completionRate"`
	AvgScore       float64 `json:"avgScore"`
	Composite      float64 `json:"composite"`
	Narrative      string  `json:"narrative"`
}

// ScatterPoint is one item on the difficulty-vs-engagement scatter plot.
type ScatterPoint struct {
	ItemID     string  `json:"itemId"`
	ItemTitle  string  `json:"itemTitle"`
	ItemType   string  `json:"itemType"`
	Difficulty float64 `json:"difficulty"` // 0–100; higher = harder
	Engagement float64 `json:"engagement"` // avg seconds on task
	Flag       string  `json:"flag,omitempty"` // "needs_redesign" when low engagement + high difficulty
}

// Insights holds the computed signals stored in analytics.instructor_insights.
type Insights struct {
	CourseID       string         `json:"courseId"`
	WeekOf         string         `json:"weekOf"` // "YYYY-MM-DD"
	WorkingWell    []SignalItem   `json:"workingWell"`
	NeedsAttention []SignalItem   `json:"needsAttention"`
	ScatterData    []ScatterPoint `json:"scatterData"`
	GeneratedAt    string         `json:"generatedAt"`
}

// CrossSectionRow holds comparison data for one section of a course.
type CrossSectionRow struct {
	SectionID   string  `json:"sectionId"`
	SectionName string  `json:"sectionName"`
	AvgGrade    float64 `json:"avgGrade"`    // avg quiz score_percent (0–100)
	AtRiskCount int     `json:"atRiskCount"` // enrolled students with no quiz attempts
}

// Compute derives and persists instructor insights for a course.
func Compute(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*Insights, error) {
	weekOf := weekStart(time.Now().UTC())
	items, err := loadItemMetrics(ctx, pool, courseID)
	if err != nil {
		return nil, fmt.Errorf("instructorinsights: load items: %w", err)
	}

	dismissed, err := loadDismissedKeys(ctx, pool, courseID)
	if err != nil {
		return nil, fmt.Errorf("instructorinsights: load dismissed: %w", err)
	}

	active := filterDismissed(items, dismissed)
	sorted := sortByComposite(active)

	working := topN(sorted, 3, true)
	attention := topN(sorted, 3, false)
	scatter := buildScatter(items)

	ins := &Insights{
		CourseID:       courseID.String(),
		WeekOf:         weekOf.Format("2006-01-02"),
		WorkingWell:    toSignalSlice(working),
		NeedsAttention: toSignalSlice(attention),
		ScatterData:    scatter,
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	if err := persist(ctx, pool, courseID, weekOf, ins); err != nil {
		return nil, fmt.Errorf("instructorinsights: persist: %w", err)
	}
	return ins, nil
}

// Load returns the most recent stored insights for a course, computing if none exist.
func Load(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*Insights, error) {
	var ww, na, sc []byte
	var generatedAt time.Time
	var weekOf string

	err := pool.QueryRow(ctx, `
SELECT week_of::text, working_well, needs_attention, scatter_data, generated_at
FROM analytics.instructor_insights
WHERE course_id = $1
ORDER BY week_of DESC
LIMIT 1
`, courseID).Scan(&weekOf, &ww, &na, &sc, &generatedAt)

	if err != nil {
		// No data yet — compute fresh.
		return Compute(ctx, pool, courseID)
	}

	var working []SignalItem
	var attention []SignalItem
	var scatter []ScatterPoint
	_ = json.Unmarshal(ww, &working)
	_ = json.Unmarshal(na, &attention)
	_ = json.Unmarshal(sc, &scatter)

	if working == nil {
		working = []SignalItem{}
	}
	if attention == nil {
		attention = []SignalItem{}
	}
	if scatter == nil {
		scatter = []ScatterPoint{}
	}

	return &Insights{
		CourseID:       courseID.String(),
		WeekOf:         weekOf,
		WorkingWell:    working,
		NeedsAttention: attention,
		ScatterData:    scatter,
		GeneratedAt:    generatedAt.Format(time.RFC3339),
	}, nil
}

// DismissSignal records a dismissed signal for a course.
func DismissSignal(ctx context.Context, pool *pgxpool.Pool, courseID, dismissedBy uuid.UUID, signalKey, reason string) error {
	var reasonArg any
	if reason != "" {
		reasonArg = reason
	}
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.dismissed_signals (course_id, signal_key, dismissed_by, reason)
VALUES ($1, $2, $3, $4)
ON CONFLICT (course_id, signal_key) DO UPDATE
    SET dismissed_by = EXCLUDED.dismissed_by,
        reason       = EXCLUDED.reason,
        dismissed_at = now()
`, courseID, signalKey, dismissedBy, reasonArg)
	return err
}

// LoadCrossSection returns section comparison data for a course.
func LoadCrossSection(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]CrossSectionRow, error) {
	rows, err := pool.Query(ctx, `
SELECT
    cs.id::text,
    COALESCE(cs.name, cs.section_code)         AS section_name,
    COALESCE(AVG(qa.score_percent), 0)::float8  AS avg_grade,
    COUNT(DISTINCT ce.user_id)::int             AS at_risk_count
FROM course.course_sections cs
JOIN course.course_enrollments ce ON ce.course_id = cs.course_id AND ce.active = true
LEFT JOIN course.quiz_attempts qa
    ON qa.course_id = $1
   AND qa.student_user_id = ce.user_id
   AND qa.status = 'submitted'
   AND qa.score_percent IS NOT NULL
WHERE cs.course_id = $1
  AND cs.status = 'active'
GROUP BY cs.id, cs.name, cs.section_code
ORDER BY section_name
`, courseID)
	if err != nil {
		return nil, fmt.Errorf("instructorinsights: cross-section: %w", err)
	}
	defer rows.Close()

	var result []CrossSectionRow
	for rows.Next() {
		var r CrossSectionRow
		if err := rows.Scan(&r.SectionID, &r.SectionName, &r.AvgGrade, &r.AtRiskCount); err != nil {
			continue
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []CrossSectionRow{}
	}
	return result, nil
}

// ---- internal helpers ----

type itemMetric struct {
	ItemID         uuid.UUID
	ItemTitle      string
	ItemType       string // "quiz" | "assignment" | "content_page"
	CompletionRate float64
	AvgScore       float64 // 0–100; 0 for non-scored items
	Engagement     float64 // avg heartbeat seconds per enrolled student
	Difficulty     float64 // 0–100; higher = harder
	HasScore       bool
	Narrative      string
}

func (m itemMetric) composite() float64 {
	if m.HasScore {
		return m.CompletionRate * (m.AvgScore / 100.0)
	}
	return m.CompletionRate * 0.5
}

func (m itemMetric) signalKey() string {
	return m.ItemID.String()
}

func loadItemMetrics(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]itemMetric, error) {
	var enrolled int64
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_enrollments
WHERE course_id = $1 AND active = true
`, courseID).Scan(&enrolled); err != nil {
		return nil, err
	}
	if enrolled == 0 {
		return nil, nil
	}

	rows, err := pool.Query(ctx, `
SELECT csi.id, csi.title, csi.kind
FROM course.course_structure_items csi
WHERE csi.course_id = $1
  AND csi.kind IN ('assignment', 'quiz', 'content_page')
  AND csi.published
  AND NOT csi.archived
ORDER BY csi.sort_order
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rawItem struct {
		id    uuid.UUID
		title string
		kind  string
	}
	var rawItems []rawItem
	for rows.Next() {
		var ri rawItem
		if err := rows.Scan(&ri.id, &ri.title, &ri.kind); err != nil {
			continue
		}
		rawItems = append(rawItems, ri)
	}
	_ = rows.Err()

	metrics := make([]itemMetric, 0, len(rawItems))
	for _, ri := range rawItems {
		m := itemMetric{
			ItemID:    ri.id,
			ItemTitle: ri.title,
			ItemType:  ri.kind,
		}

		switch ri.kind {
		case "quiz":
			var attemptors int64
			_ = pool.QueryRow(ctx, `
SELECT COUNT(DISTINCT student_user_id)
FROM course.quiz_attempts
WHERE course_id = $1 AND structure_item_id = $2 AND status = 'submitted'
`, courseID, ri.id).Scan(&attemptors)
			m.CompletionRate = safeDiv(float64(attemptors), float64(enrolled))

			var avgScore *float64
			_ = pool.QueryRow(ctx, `
SELECT AVG(score_percent)
FROM course.quiz_attempts
WHERE course_id = $1
  AND structure_item_id = $2
  AND status = 'submitted'
  AND score_percent IS NOT NULL
`, courseID, ri.id).Scan(&avgScore)
			if avgScore != nil {
				m.AvgScore = *avgScore
				m.HasScore = true
			}

			// Difficulty from item_stats; quiz_id = structure_item_id.
			var avgP *float64
			_ = pool.QueryRow(ctx, `
SELECT AVG(p_value)
FROM analytics.item_stats
WHERE quiz_id = $1 AND p_value IS NOT NULL
`, ri.id).Scan(&avgP)
			if avgP != nil {
				m.Difficulty = (1.0 - *avgP) * 100.0
			} else if m.HasScore {
				m.Difficulty = 100.0 - m.AvgScore
			} else {
				m.Difficulty = 50.0
			}

		case "assignment":
			var submitters int64
			_ = pool.QueryRow(ctx, `
SELECT COUNT(DISTINCT submitted_by)
FROM course.module_assignment_submissions
WHERE module_item_id = $1
`, ri.id).Scan(&submitters)
			m.CompletionRate = safeDiv(float64(submitters), float64(enrolled))
			m.Difficulty = (1.0 - m.CompletionRate) * 100.0

		case "content_page":
			var openers int64
			_ = pool.QueryRow(ctx, `
SELECT COUNT(DISTINCT user_id)
FROM "user".user_audit
WHERE course_id = $1
  AND structure_item_id = $2
  AND event_kind = 'content_open'
`, courseID, ri.id).Scan(&openers)
			m.CompletionRate = safeDiv(float64(openers), float64(enrolled))
			m.Difficulty = 50.0
		}

		// Engagement from heartbeat events for this item.
		var heartbeats int64
		_ = pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM analytics.engagement_events
WHERE course_id = $1 AND item_id = $2 AND event_type = 'heartbeat'
`, courseID, ri.id).Scan(&heartbeats)
		m.Engagement = safeDiv(float64(heartbeats)*30.0, float64(enrolled))

		m.Narrative = buildNarrative(m)
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func loadDismissedKeys(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `
SELECT signal_key FROM analytics.dismissed_signals WHERE course_id = $1
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	dismissed := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			dismissed[k] = true
		}
	}
	return dismissed, rows.Err()
}

func filterDismissed(items []itemMetric, dismissed map[string]bool) []itemMetric {
	out := make([]itemMetric, 0, len(items))
	for _, it := range items {
		if !dismissed[it.signalKey()] {
			out = append(out, it)
		}
	}
	return out
}

func sortByComposite(items []itemMetric) []itemMetric {
	cp := make([]itemMetric, len(items))
	copy(cp, items)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].composite() > cp[j].composite()
	})
	return cp
}

// topN returns the top (best) or bottom (worst) n items from the sorted slice.
func topN(sorted []itemMetric, n int, best bool) []itemMetric {
	if len(sorted) == 0 {
		return nil
	}
	if best {
		if len(sorted) <= n {
			return sorted
		}
		return sorted[:n]
	}
	// Bottom n: reverse the slice.
	reversed := make([]itemMetric, len(sorted))
	for i, v := range sorted {
		reversed[len(sorted)-1-i] = v
	}
	if len(reversed) <= n {
		return reversed
	}
	return reversed[:n]
}

func buildScatter(items []itemMetric) []ScatterPoint {
	pts := make([]ScatterPoint, 0, len(items))
	for _, m := range items {
		pt := ScatterPoint{
			ItemID:     m.ItemID.String(),
			ItemTitle:  m.ItemTitle,
			ItemType:   m.ItemType,
			Difficulty: roundTo1(m.Difficulty),
			Engagement: roundTo1(m.Engagement),
		}
		if m.Difficulty > 65 && m.Engagement < 120 {
			pt.Flag = "needs_redesign"
		}
		pts = append(pts, pt)
	}
	return pts
}

func toSignalSlice(items []itemMetric) []SignalItem {
	out := make([]SignalItem, 0, len(items))
	for _, m := range items {
		out = append(out, SignalItem{
			ItemID:         m.ItemID.String(),
			ItemTitle:      m.ItemTitle,
			ItemType:       m.ItemType,
			CompletionRate: roundTo3(m.CompletionRate),
			AvgScore:       roundTo1(m.AvgScore),
			Composite:      roundTo3(m.composite()),
			Narrative:      m.Narrative,
		})
	}
	if out == nil {
		return []SignalItem{}
	}
	return out
}

func buildNarrative(m itemMetric) string {
	pct := int(math.Round(m.CompletionRate * 100))
	if m.HasScore {
		return fmt.Sprintf(
			"%d%% of students completed this item with an average score of %.0f%%.",
			pct, m.AvgScore,
		)
	}
	return fmt.Sprintf("%d%% of students completed this item.", pct)
}

func persist(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, weekOf time.Time, ins *Insights) error {
	ww, _ := json.Marshal(ins.WorkingWell)
	na, _ := json.Marshal(ins.NeedsAttention)
	sc, _ := json.Marshal(ins.ScatterData)
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.instructor_insights (course_id, week_of, working_well, needs_attention, scatter_data)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (course_id, week_of) DO UPDATE
    SET working_well    = EXCLUDED.working_well,
        needs_attention = EXCLUDED.needs_attention,
        scatter_data    = EXCLUDED.scatter_data,
        generated_at    = now()
`, courseID, weekOf.Format("2006-01-02"), ww, na, sc)
	return err
}

func weekStart(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return t.AddDate(0, 0, -(wd - 1)).Truncate(24 * time.Hour)
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	v := a / b
	if v > 1.0 {
		return 1.0
	}
	return v
}

func roundTo1(v float64) float64 { return math.Round(v*10) / 10 }
func roundTo3(v float64) float64 { return math.Round(v*1000) / 1000 }
