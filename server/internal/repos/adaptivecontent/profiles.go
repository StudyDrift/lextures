package adaptivecontent

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProfileRow is a course.adaptation_profiles row (AC.1/AC.2).
type ProfileRow struct {
	ID               uuid.UUID
	UnitID           uuid.UUID
	EnrollmentID     uuid.UUID
	UserID           uuid.UUID
	ProfileSignature string
	EmphasisMode     *string
	PayloadJSON      json.RawMessage
	SourceAttemptID  *uuid.UUID
	TargetBloom      *string
	ReadingLevelPref *string
	ModalityPref     *string
	AxisSet          []string
	IsNeutral        bool
	CreatedAt        time.Time
}

// ProfileUpsert is the input for inserting/updating a profile.
type ProfileUpsert struct {
	UnitID           uuid.UUID
	EnrollmentID     uuid.UUID
	UserID           uuid.UUID
	ProfileSignature string
	EmphasisMode     string
	PayloadJSON      any
	SourceAttemptID  *uuid.UUID
	TargetBloom      *string
	ReadingLevelPref string
	ModalityPref     string
	AxisSet          []string
	IsNeutral        bool
}

// EmphasisCount is a cohort aggregate bucket.
type EmphasisCount struct {
	EmphasisMode string
	Count        int64
}

// SignatureCount is a cohort signature aggregate (no PII).
type SignatureCount struct {
	ProfileSignature string
	EmphasisMode     string
	Count            int64
}

// CohortDistribution is the instructor cohort view (no free-text PII).
type CohortDistribution struct {
	ByEmphasis  []EmphasisCount
	BySignature []SignatureCount
	Unprofiled  int64 // enrollments with no profile (filled by caller if needed)
}

// UpsertProfile inserts or replaces the profile for (unit, enrollment).
// Latest attempt wins: ON CONFLICT updates all decision fields.
func UpsertProfile(ctx context.Context, pool *pgxpool.Pool, in ProfileUpsert) (*ProfileRow, error) {
	payload, err := json.Marshal(in.PayloadJSON)
	if err != nil {
		return nil, err
	}
	if in.AxisSet == nil {
		in.AxisSet = []string{}
	}
	var emphasis *string
	if in.EmphasisMode != "" {
		e := in.EmphasisMode
		emphasis = &e
	}
	var reading *string
	if in.ReadingLevelPref != "" {
		r := in.ReadingLevelPref
		reading = &r
	}
	var modality *string
	if in.ModalityPref != "" {
		m := in.ModalityPref
		modality = &m
	}

	row := pool.QueryRow(ctx, `
INSERT INTO course.adaptation_profiles (
  unit_id, enrollment_id, user_id, profile_signature, emphasis_mode, payload_json,
  source_attempt_id, target_bloom, reading_level_pref, modality_pref, axis_set, is_neutral, created_at
) VALUES (
  $1, $2, $3, $4, $5, $6::jsonb,
  $7, $8::course.bloom_level, $9, $10, $11, $12, NOW()
)
ON CONFLICT (unit_id, enrollment_id) DO UPDATE SET
  profile_signature = EXCLUDED.profile_signature,
  emphasis_mode = EXCLUDED.emphasis_mode,
  payload_json = EXCLUDED.payload_json,
  source_attempt_id = EXCLUDED.source_attempt_id,
  target_bloom = EXCLUDED.target_bloom,
  reading_level_pref = EXCLUDED.reading_level_pref,
  modality_pref = EXCLUDED.modality_pref,
  axis_set = EXCLUDED.axis_set,
  is_neutral = EXCLUDED.is_neutral,
  user_id = EXCLUDED.user_id,
  created_at = NOW()
RETURNING id, unit_id, enrollment_id, user_id, profile_signature, emphasis_mode, payload_json,
          source_attempt_id, target_bloom::text, reading_level_pref, modality_pref, axis_set,
          is_neutral, created_at
`, in.UnitID, in.EnrollmentID, in.UserID, in.ProfileSignature, emphasis, string(payload),
		in.SourceAttemptID, in.TargetBloom, reading, modality, in.AxisSet, in.IsNeutral,
	)
	return scanProfile(row)
}

// GetProfileForEnrollment returns the profile for a unit+enrollment, or nil.
func GetProfileForEnrollment(ctx context.Context, pool *pgxpool.Pool, unitID, enrollmentID uuid.UUID) (*ProfileRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, unit_id, enrollment_id, user_id, profile_signature, emphasis_mode, payload_json,
       source_attempt_id, target_bloom::text, reading_level_pref, modality_pref, axis_set,
       is_neutral, created_at
FROM course.adaptation_profiles
WHERE unit_id = $1 AND enrollment_id = $2
`, unitID, enrollmentID)
	p, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// GetProfileForUser returns the profile for a unit+user, or nil.
func GetProfileForUser(ctx context.Context, pool *pgxpool.Pool, unitID, userID uuid.UUID) (*ProfileRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, unit_id, enrollment_id, user_id, profile_signature, emphasis_mode, payload_json,
       source_attempt_id, target_bloom::text, reading_level_pref, modality_pref, axis_set,
       is_neutral, created_at
FROM course.adaptation_profiles
WHERE unit_id = $1 AND user_id = $2
`, unitID, userID)
	p, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// ListCohortDistribution returns aggregate counts per emphasis_mode and signature (no PII).
func ListCohortDistribution(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) (*CohortDistribution, error) {
	emRows, err := pool.Query(ctx, `
SELECT COALESCE(emphasis_mode, 'unknown') AS emphasis_mode, COUNT(*)::bigint
FROM course.adaptation_profiles
WHERE unit_id = $1
GROUP BY 1
ORDER BY 1
`, unitID)
	if err != nil {
		return nil, err
	}
	defer emRows.Close()
	var byEm []EmphasisCount
	for emRows.Next() {
		var e EmphasisCount
		if err := emRows.Scan(&e.EmphasisMode, &e.Count); err != nil {
			return nil, err
		}
		byEm = append(byEm, e)
	}
	if err := emRows.Err(); err != nil {
		return nil, err
	}

	sigRows, err := pool.Query(ctx, `
SELECT profile_signature, COALESCE(emphasis_mode, 'unknown'), COUNT(*)::bigint
FROM course.adaptation_profiles
WHERE unit_id = $1
GROUP BY profile_signature, emphasis_mode
ORDER BY COUNT(*) DESC, profile_signature ASC
`, unitID)
	if err != nil {
		return nil, err
	}
	defer sigRows.Close()
	var bySig []SignatureCount
	for sigRows.Next() {
		var s SignatureCount
		if err := sigRows.Scan(&s.ProfileSignature, &s.EmphasisMode, &s.Count); err != nil {
			return nil, err
		}
		bySig = append(bySig, s)
	}
	if err := sigRows.Err(); err != nil {
		return nil, err
	}
	if byEm == nil {
		byEm = []EmphasisCount{}
	}
	if bySig == nil {
		bySig = []SignatureCount{}
	}
	return &CohortDistribution{ByEmphasis: byEm, BySignature: bySig}, nil
}

// CountDistinctSignatures returns how many distinct profile signatures exist for a unit.
func CountDistinctSignatures(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(DISTINCT profile_signature)::bigint
FROM course.adaptation_profiles
WHERE unit_id = $1
`, unitID).Scan(&n)
	return n, err
}

// ListUnitConceptIDs returns explicit concept ids linked to a unit.
func ListUnitConceptIDs(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT concept_id
FROM course.adaptive_content_unit_concepts
WHERE unit_id = $1
ORDER BY concept_id ASC
`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ReplaceUnitConcepts replaces the explicit concept set for a unit.
func ReplaceUnitConcepts(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, conceptIDs []uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM course.adaptive_content_unit_concepts WHERE unit_id = $1`, unitID); err != nil {
		return err
	}
	for _, cid := range conceptIDs {
		if cid == uuid.Nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO course.adaptive_content_unit_concepts (unit_id, concept_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, unitID, cid); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListActiveUnitsByPreAssessment returns active units in a course whose pre_assessment_item_id matches.
func ListActiveUnitsByPreAssessment(ctx context.Context, pool *pgxpool.Pool, courseID, preAssessmentItemID uuid.UUID) ([]UnitRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, course_id, target_kind, target_module_item_id, target_outcome_id,
       base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
       allowed_axes, status, created_by, created_at, updated_at,
       trigger_mode, mastery_freshness_days, content_version, min_fidelity,
       quarantined, quarantined_reason
FROM course.adaptive_content_units
WHERE course_id = $1
  AND pre_assessment_item_id = $2
  AND status = 'active'
ORDER BY created_at ASC
`, courseID, preAssessmentItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UnitRow
	for rows.Next() {
		u, err := scanUnit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ListMisconceptionIDsForAttempt returns distinct misconception ids recorded for an attempt.
func ListMisconceptionIDsForAttempt(ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT misconception_id
FROM course.misconception_events
WHERE attempt_id = $1
ORDER BY misconception_id ASC
`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ConceptIDsTaggedOnQuizQuestions returns distinct concept ids tagged on questions belonging to the quiz item.
// For adaptive quizzes with empty questions_json this may return empty; unit concepts are the primary source.
func ConceptIDsTaggedOnQuizQuestions(ctx context.Context, pool *pgxpool.Pool, courseID, quizItemID uuid.UUID) ([]uuid.UUID, error) {
	// Prefer concept tags on questions that appear in the quiz's questions_json ids.
	// Also include tags on bank questions linked via concept_question_tags where the question is in the course.
	rows, err := pool.Query(ctx, `
WITH qids AS (
  SELECT DISTINCT (elem->>'id')::uuid AS question_id
  FROM course.module_quizzes m
  CROSS JOIN LATERAL jsonb_array_elements(COALESCE(m.questions_json, '[]'::jsonb)) AS elem
  WHERE m.structure_item_id = $1
    AND elem->>'id' IS NOT NULL
    AND elem->>'id' ~ '^[0-9a-fA-F-]{36}$'
)
SELECT DISTINCT t.concept_id
FROM course.concept_question_tags t
INNER JOIN qids q ON q.question_id = t.question_id
ORDER BY t.concept_id ASC
`, quizItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// StructureItemIsQuiz reports whether itemID is a quiz in courseID.
func StructureItemIsQuiz(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM course.course_structure_items
  WHERE id = $1 AND course_id = $2 AND kind = 'quiz'
)`, itemID, courseID).Scan(&ok)
	return ok, err
}

// AdaptiveContentEnabledForCourse returns the course flag value.
func AdaptiveContentEnabledForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (bool, error) {
	var enabled bool
	err := pool.QueryRow(ctx, `
SELECT adaptive_content_enabled FROM course.courses WHERE id = $1
`, courseID).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return enabled, err
}

// GetEnrollmentIDForUser returns an active student-equivalent enrollment id, or nil.
func GetEnrollmentIDForUser(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT ce.id
FROM course.course_enrollments ce
JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.course_id = $1 AND ce.user_id = $2 AND ce.active
LIMIT 1
`, courseID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// ParentModuleID returns the parent module id for a structure item (or the item itself if it is a module).
func ParentModuleID(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (*uuid.UUID, error) {
	var kind string
	var parent *uuid.UUID
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id, kind, parent_id
FROM course.course_structure_items
WHERE course_id = $1 AND id = $2
`, courseID, itemID).Scan(&id, &kind, &parent)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if kind == "module" {
		return &id, nil
	}
	if parent == nil {
		return nil, nil
	}
	return parent, nil
}

// ConfigureQuizAsAdaptivePreCheck marks a quiz as adaptive and seeds source items + question count.
func ConfigureQuizAsAdaptivePreCheck(
	ctx context.Context,
	pool *pgxpool.Pool,
	quizItemID uuid.UUID,
	sourceItemIDs []uuid.UUID,
	questionCount int32,
	systemPrompt string,
) error {
	if questionCount <= 0 {
		questionCount = 5
	}
	srcJSON, err := json.Marshal(sourceItemIDs)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
UPDATE course.module_quizzes
SET is_adaptive = TRUE,
    adaptive_source_item_ids = $2::jsonb,
    adaptive_question_count = $3,
    adaptive_system_prompt = $4,
    points_worth = COALESCE(points_worth, $3),
    show_score_timing = 'immediate',
    review_when = 'after_submit',
    updated_at = NOW()
WHERE structure_item_id = $1
`, quizItemID, string(srcJSON), questionCount, systemPrompt)
	return err
}

func scanProfile(row scannable) (*ProfileRow, error) {
	var p ProfileRow
	var payload []byte
	err := row.Scan(
		&p.ID, &p.UnitID, &p.EnrollmentID, &p.UserID, &p.ProfileSignature, &p.EmphasisMode, &payload,
		&p.SourceAttemptID, &p.TargetBloom, &p.ReadingLevelPref, &p.ModalityPref, &p.AxisSet,
		&p.IsNeutral, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if payload == nil {
		payload = []byte("{}")
	}
	p.PayloadJSON = payload
	if p.AxisSet == nil {
		p.AxisSet = []string{}
	}
	return &p, nil
}
