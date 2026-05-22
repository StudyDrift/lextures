// Package masteryheatmap queries the analytics.mastery_heatmap materialized view
// (migration 169) to power the mastery heatmap API (plan 9.3).
package masteryheatmap

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConceptSummary is per-concept aggregate statistics for a course.
type ConceptSummary struct {
	ConceptID   uuid.UUID `json:"conceptId"`
	ConceptName string    `json:"conceptName"`
	MeanMastery float64   `json:"meanMastery"`
	PctMastered float64   `json:"pctMastered"` // fraction with mastery >= 0.8
	PctAtRisk   float64   `json:"pctAtRisk"`   // fraction with mastery < 0.4
}

// HeatmapCell is one cell in the matrix (per enrollment × concept).
type HeatmapCell struct {
	ConceptID    uuid.UUID  `json:"conceptId"`
	MasteryScore *float64   `json:"masteryScore"`
	Assessed     bool       `json:"assessed"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
}

// HeatmapRow is one student row in the matrix.
type HeatmapRow struct {
	EnrollmentID uuid.UUID     `json:"enrollmentId"`
	UserID       uuid.UUID     `json:"userId"`
	DisplayName  *string       `json:"displayName"`
	Cells        []HeatmapCell `json:"cells"`
}

// ConceptMeta is lightweight concept metadata for the heatmap header.
type ConceptMeta struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// HeatmapResult is the full heatmap response for a course.
type HeatmapResult struct {
	Concepts    []ConceptMeta    `json:"concepts"`
	Rows        []HeatmapRow     `json:"rows"`
	Summary     []ConceptSummary `json:"summary"`
	RefreshedAt *time.Time       `json:"refreshedAt"`
}

// DrillDownStudent is a student entry for the per-concept drill-down.
type DrillDownStudent struct {
	EnrollmentID uuid.UUID  `json:"enrollmentId"`
	UserID       uuid.UUID  `json:"userId"`
	DisplayName  *string    `json:"displayName"`
	MasteryScore *float64   `json:"masteryScore"`
	Assessed     bool       `json:"assessed"`
}

// StudentMasteryRow is one student's mastery vector for all concepts in a course.
type StudentMasteryRow struct {
	EnrollmentID uuid.UUID     `json:"enrollmentId"`
	UserID       uuid.UUID     `json:"userId"`
	Concepts     []ConceptMeta `json:"concepts"`
	Cells        []HeatmapCell `json:"cells"`
}

// HeatmapForCourse returns the full heatmap matrix for a course.
// If the materialized view is empty (§1 not yet deployed) the result has empty rows and
// a nil RefreshedAt to signal the empty-state.
func HeatmapForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*HeatmapResult, error) {
	// 1. Concepts for this course (ordered alphabetically).
	concepts, err := listConceptsForCourse(ctx, pool, courseID)
	if err != nil {
		return nil, err
	}

	// 2. Active student enrollments.
	type enrollmentRow struct {
		id          uuid.UUID
		userID      uuid.UUID
		displayName *string
	}
	erows, err := pool.Query(ctx, `
SELECT ce.id, ce.user_id, u.display_name
FROM course.course_enrollments ce
JOIN "user".users u ON u.id = ce.user_id
JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.course_id = $1 AND ce.active = true
ORDER BY COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	var enrollments []enrollmentRow
	for erows.Next() {
		var er enrollmentRow
		if err := erows.Scan(&er.id, &er.userID, &er.displayName); err != nil {
			erows.Close()
			return nil, err
		}
		enrollments = append(enrollments, er)
	}
	erows.Close()
	if err := erows.Err(); err != nil {
		return nil, err
	}

	if len(concepts) == 0 || len(enrollments) == 0 {
		// Use a non-nil empty slice so JSON encodes as [] not null.
		metas := make([]ConceptMeta, 0, len(concepts))
		metas = append(metas, concepts...)
		return &HeatmapResult{
			Concepts: metas,
			Rows:     []HeatmapRow{},
			Summary:  []ConceptSummary{},
		}, nil
	}

	// 3. Pull heatmap data for all enrolled users.
	userIDs := make([]uuid.UUID, len(enrollments))
	for i, e := range enrollments {
		userIDs[i] = e.userID
	}
	type stateKey struct {
		userID    uuid.UUID
		conceptID uuid.UUID
	}
	type stateVal struct {
		mastery   float64
		updatedAt time.Time
	}
	stateMap := make(map[stateKey]stateVal)

	rows, err := pool.Query(ctx, `
SELECT h.user_id, h.concept_id, h.mastery_score, h.state_updated_at
FROM analytics.mastery_heatmap h
WHERE h.course_id = $1 AND h.user_id = ANY($2::uuid[])
`, courseID, userIDs)
	if err != nil {
		// Fall through to base tables if materialized view doesn't exist yet.
		rows, err = pool.Query(ctx, `
SELECT lcs.user_id, lcs.concept_id, (lcs.mastery)::float8, lcs.updated_at
FROM course.learner_concept_states lcs
JOIN course.concepts c ON c.id = lcs.concept_id
WHERE c.course_id = $1 AND lcs.user_id = ANY($2::uuid[])
`, courseID, userIDs)
		if err != nil {
			return nil, err
		}
	}
	var refreshedAt *time.Time
	for rows.Next() {
		var uid, cid uuid.UUID
		var mastery float64
		var updatedAt time.Time
		if err := rows.Scan(&uid, &cid, &mastery, &updatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		stateMap[stateKey{userID: uid, conceptID: cid}] = stateVal{mastery: mastery, updatedAt: updatedAt}
		if refreshedAt == nil || updatedAt.After(*refreshedAt) {
			t := updatedAt
			refreshedAt = &t
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 4. Build concept index and summary accumulators.
	conceptMetas := make([]ConceptMeta, len(concepts))
	copy(conceptMetas, concepts)

	type summaryAccum struct {
		total    float64
		count    int
		mastered int
		atRisk   int
	}
	summaryMap := make(map[uuid.UUID]*summaryAccum, len(concepts))
	for _, c := range concepts {
		summaryMap[c.ID] = &summaryAccum{}
	}

	// 5. Build rows.
	heatmapRows := make([]HeatmapRow, 0, len(enrollments))
	for _, e := range enrollments {
		cells := make([]HeatmapCell, len(concepts))
		for i, c := range concepts {
			key := stateKey{userID: e.userID, conceptID: c.ID}
			if sv, ok := stateMap[key]; ok {
				score := sv.mastery
				t := sv.updatedAt
				cells[i] = HeatmapCell{
					ConceptID:    c.ID,
					MasteryScore: &score,
					Assessed:     true,
					UpdatedAt:    &t,
				}
				acc := summaryMap[c.ID]
				acc.total += score
				acc.count++
				if score >= 0.8 {
					acc.mastered++
				}
				if score < 0.4 {
					acc.atRisk++
				}
			} else {
				cells[i] = HeatmapCell{
					ConceptID: c.ID,
					Assessed:  false,
				}
			}
		}
		heatmapRows = append(heatmapRows, HeatmapRow{
			EnrollmentID: e.id,
			UserID:       e.userID,
			DisplayName:  e.displayName,
			Cells:        cells,
		})
	}

	// 6. Build summary slice.
	summary := make([]ConceptSummary, len(concepts))
	totalStudents := len(enrollments)
	for i, c := range concepts {
		acc := summaryMap[c.ID]
		var mean, pctMastered, pctAtRisk float64
		if acc.count > 0 {
			mean = acc.total / float64(acc.count)
		}
		if totalStudents > 0 {
			pctMastered = float64(acc.mastered) / float64(totalStudents)
			pctAtRisk = float64(acc.atRisk) / float64(totalStudents)
		}
		summary[i] = ConceptSummary{
			ConceptID:   c.ID,
			ConceptName: c.Name,
			MeanMastery: mean,
			PctMastered: pctMastered,
			PctAtRisk:   pctAtRisk,
		}
	}

	return &HeatmapResult{
		Concepts:    conceptMetas,
		Rows:        heatmapRows,
		Summary:     summary,
		RefreshedAt: refreshedAt,
	}, nil
}

// DrillDownForConcept returns the list of students for a specific concept with their mastery.
func DrillDownForConcept(ctx context.Context, pool *pgxpool.Pool, courseID, conceptID uuid.UUID) ([]DrillDownStudent, error) {
	rows, err := pool.Query(ctx, `
SELECT
    ce.id AS enrollment_id,
    ce.user_id,
    u.display_name,
    h.mastery_score
FROM course.course_enrollments ce
JOIN "user".users u ON u.id = ce.user_id
JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
LEFT JOIN analytics.mastery_heatmap h
    ON h.user_id = ce.user_id AND h.course_id = ce.course_id AND h.concept_id = $2
WHERE ce.course_id = $1 AND ce.active = true
ORDER BY COALESCE(h.mastery_score, -1) ASC,
         COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) ASC
`, courseID, conceptID)
	if err != nil {
		// Fall back to base tables.
		rows, err = pool.Query(ctx, `
SELECT
    ce.id AS enrollment_id,
    ce.user_id,
    u.display_name,
    (lcs.mastery)::float8
FROM course.course_enrollments ce
JOIN "user".users u ON u.id = ce.user_id
JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
LEFT JOIN course.learner_concept_states lcs
    ON lcs.user_id = ce.user_id AND lcs.concept_id = $2
WHERE ce.course_id = $1 AND ce.active = true
ORDER BY COALESCE((lcs.mastery)::float8, -1) ASC,
         COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) ASC
`, courseID, conceptID)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()
	var out []DrillDownStudent
	for rows.Next() {
		var eid, uid uuid.UUID
		var displayName *string
		var mastery *float64
		if err := rows.Scan(&eid, &uid, &displayName, &mastery); err != nil {
			return nil, err
		}
		out = append(out, DrillDownStudent{
			EnrollmentID: eid,
			UserID:       uid,
			DisplayName:  displayName,
			MasteryScore: mastery,
			Assessed:     mastery != nil,
		})
	}
	return out, rows.Err()
}

// StudentMastery returns one student's mastery vector for a course identified by enrollment ID.
func StudentMastery(ctx context.Context, pool *pgxpool.Pool, courseID, enrollmentID uuid.UUID) (*StudentMasteryRow, error) {
	// Resolve enrollment → user.
	var userID uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT user_id FROM course.course_enrollments
WHERE id = $1 AND course_id = $2 AND active = true
`, enrollmentID, courseID).Scan(&userID)
	if err != nil {
		return nil, err
	}

	concepts, err := listConceptsForCourse(ctx, pool, courseID)
	if err != nil {
		return nil, err
	}

	conceptMetas := make([]ConceptMeta, len(concepts))
	copy(conceptMetas, concepts)

	rows, err := pool.Query(ctx, `
SELECT h.concept_id, h.mastery_score, h.state_updated_at
FROM analytics.mastery_heatmap h
WHERE h.course_id = $1 AND h.user_id = $2
`, courseID, userID)
	if err != nil {
		rows, err = pool.Query(ctx, `
SELECT lcs.concept_id, (lcs.mastery)::float8, lcs.updated_at
FROM course.learner_concept_states lcs
JOIN course.concepts c ON c.id = lcs.concept_id
WHERE c.course_id = $1 AND lcs.user_id = $2
`, courseID, userID)
		if err != nil {
			return nil, err
		}
	}
	type sv struct {
		mastery   float64
		updatedAt time.Time
	}
	stateMap := make(map[uuid.UUID]sv)
	for rows.Next() {
		var cid uuid.UUID
		var mastery float64
		var updatedAt time.Time
		if err := rows.Scan(&cid, &mastery, &updatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		stateMap[cid] = sv{mastery: mastery, updatedAt: updatedAt}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	cells := make([]HeatmapCell, len(concepts))
	for i, c := range concepts {
		if s, ok := stateMap[c.ID]; ok {
			score := s.mastery
			t := s.updatedAt
			cells[i] = HeatmapCell{
				ConceptID:    c.ID,
				MasteryScore: &score,
				Assessed:     true,
				UpdatedAt:    &t,
			}
		} else {
			cells[i] = HeatmapCell{ConceptID: c.ID, Assessed: false}
		}
	}

	return &StudentMasteryRow{
		EnrollmentID: enrollmentID,
		UserID:       userID,
		Concepts:     conceptMetas,
		Cells:        cells,
	}, nil
}

// RefreshMaterializedView triggers a concurrent refresh of the heatmap cache.
// Returns without error if the materialized view does not exist yet.
func RefreshMaterializedView(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY analytics.mastery_heatmap`)
	return err
}

// listConceptsForCourse returns ordered concepts for a course.
func listConceptsForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]ConceptMeta, error) {
	rows, err := pool.Query(ctx, `
SELECT id, name FROM course.concepts WHERE course_id = $1 ORDER BY name ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ConceptMeta
	for rows.Next() {
		var m ConceptMeta
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
