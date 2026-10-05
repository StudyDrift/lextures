package questionbank

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QuestionWrite is an authored question-bank create or full update.
type QuestionWrite struct {
	QuestionType           string
	Stem                   string
	Options                json.RawMessage
	CorrectAnswer          json.RawMessage
	Explanation            *string
	Points                 float64
	Status                 string
	Shared                 bool
	Metadata               json.RawMessage
	ShuffleChoicesOverride *bool
	SRSEligible            bool
	ChangeNote             *string
}

// QuestionVersionSummary is one history row for the bank UI.
type QuestionVersionSummary struct {
	VersionNumber int32
	ChangeNote    *string
	ChangeSummary json.RawMessage
	CreatedBy     *uuid.UUID
	CreatedAt     time.Time
}

var (
	allowedQuestionTypes = map[string]struct{}{
		"mc_single": {}, "mc_multiple": {}, "true_false": {}, "short_answer": {},
		"numeric": {}, "matching": {}, "ordering": {}, "hotspot": {}, "formula": {},
		"code": {}, "file_upload": {}, "audio_response": {}, "video_response": {},
	}
	allowedQuestionStatuses = map[string]struct{}{
		"draft": {}, "active": {}, "retired": {},
	}
)

// ValidateQuestionWrite checks fields the question bank client already sends.
func ValidateQuestionWrite(in QuestionWrite, creating bool) error {
	if creating || strings.TrimSpace(in.Stem) != "" || in.QuestionType != "" {
		if strings.TrimSpace(in.Stem) == "" {
			return fmt.Errorf("stem is required")
		}
	}
	if in.QuestionType != "" {
		if _, ok := allowedQuestionTypes[in.QuestionType]; !ok {
			return fmt.Errorf("invalid question type")
		}
	}
	if in.Status != "" {
		if _, ok := allowedQuestionStatuses[in.Status]; !ok {
			return fmt.Errorf("invalid status")
		}
	}
	if in.Points < 0 {
		return fmt.Errorf("points must be zero or greater")
	}
	return nil
}

// CreateAuthoredQuestion inserts a bank question and its first version snapshot.
func CreateAuthoredQuestion(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, createdBy *uuid.UUID, in QuestionWrite) (*QuestionEntity, error) {
	if err := ValidateQuestionWrite(in, true); err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = "draft"
	}
	qType := in.QuestionType
	if qType == "" {
		qType = "mc_single"
	}
	meta := in.Metadata
	if len(meta) == 0 || string(meta) == "null" {
		meta = json.RawMessage(`{}`)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO course.questions (
	course_id, question_type, stem, options, correct_answer, explanation,
	points, status, shared, source, metadata, created_by, is_published,
	shuffle_choices_override, srs_eligible, version_number
)
VALUES (
	$1, $2::course.question_type, $3, $4, $5, $6,
	$7, $8::course.question_status, $9, 'authored', $10, $11, $12, $13, $14, 1
)
RETURNING id
`, courseID, qType, strings.TrimSpace(in.Stem), rawOrNil(in.Options), rawOrNil(in.CorrectAnswer), in.Explanation,
		in.Points, status, in.Shared, meta, createdBy, status == "active", in.ShuffleChoicesOverride, in.SRSEligible).Scan(&id)
	if err != nil {
		return nil, err
	}
	row, err := getQuestionForCourseTx(ctx, tx, courseID, id)
	if err != nil {
		return nil, err
	}
	note := in.ChangeNote
	if note == nil {
		s := "Created"
		note = &s
	}
	if err := insertQuestionVersion(ctx, tx, row, note, createdBy); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return row, nil
}

// UpdateAuthoredQuestion updates a bank question and appends a version snapshot.
func UpdateAuthoredQuestion(ctx context.Context, pool *pgxpool.Pool, courseID, questionID uuid.UUID, editor *uuid.UUID, in QuestionWrite) (*QuestionEntity, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cur, err := getQuestionForCourseTx(ctx, tx, courseID, questionID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, nil
	}
	next := *cur
	if t := strings.TrimSpace(in.QuestionType); t != "" {
		next.QuestionType = t
	}
	if s := strings.TrimSpace(in.Stem); s != "" {
		next.Stem = s
	}
	next.Options = in.Options
	next.CorrectAnswer = in.CorrectAnswer
	next.Explanation = in.Explanation
	next.Points = in.Points
	if in.Status != "" {
		next.Status = in.Status
	}
	next.Shared = in.Shared
	if len(in.Metadata) > 0 && string(in.Metadata) != "null" {
		next.Metadata = in.Metadata
	}
	next.ShuffleChoicesOverride = in.ShuffleChoicesOverride
	next.SRSEligible = in.SRSEligible
	if err := ValidateQuestionWrite(QuestionWrite{
		QuestionType: next.QuestionType,
		Stem:         next.Stem,
		Points:       next.Points,
		Status:       next.Status,
	}, true); err != nil {
		return nil, err
	}
	next.VersionNumber = cur.VersionNumber + 1
	next.IsPublished = next.Status == "active"
	meta := next.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	tag, err := tx.Exec(ctx, `
UPDATE course.questions
SET question_type = $3::course.question_type,
	stem = $4,
	options = $5,
	correct_answer = $6,
	explanation = $7,
	points = $8,
	status = $9::course.question_status,
	shared = $10,
	metadata = $11,
	shuffle_choices_override = $12,
	srs_eligible = $13,
	is_published = $14,
	version_number = $15,
	updated_at = NOW()
WHERE id = $2 AND course_id = $1
`, courseID, questionID, next.QuestionType, next.Stem, rawOrNil(next.Options), rawOrNil(next.CorrectAnswer), next.Explanation,
		next.Points, next.Status, next.Shared, meta, next.ShuffleChoicesOverride, next.SRSEligible, next.IsPublished, next.VersionNumber)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	row, err := getQuestionForCourseTx(ctx, tx, courseID, questionID)
	if err != nil {
		return nil, err
	}
	if err := insertQuestionVersion(ctx, tx, row, in.ChangeNote, editor); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return row, nil
}

// ListQuestionVersions returns history newest first.
func ListQuestionVersions(ctx context.Context, pool *pgxpool.Pool, courseID, questionID uuid.UUID) ([]QuestionVersionSummary, error) {
	rows, err := pool.Query(ctx, `
SELECT qv.version_number, qv.change_note, qv.change_summary, qv.created_by, qv.created_at
FROM course.question_versions qv
INNER JOIN course.questions q ON q.id = qv.question_id
WHERE qv.question_id = $1 AND q.course_id = $2
ORDER BY qv.version_number DESC
`, questionID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QuestionVersionSummary{}
	for rows.Next() {
		var s QuestionVersionSummary
		if err := rows.Scan(&s.VersionNumber, &s.ChangeNote, &s.ChangeSummary, &s.CreatedBy, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// RestoreQuestionVersion copies a historical snapshot onto the question as a new version.
// The returned int is the new version number. A nil entity means the question or version was missing.
func RestoreQuestionVersion(ctx context.Context, pool *pgxpool.Pool, courseID, questionID uuid.UUID, version int32, editor *uuid.UUID, changeNote *string) (*QuestionEntity, error) {
	if version < 1 {
		return nil, fmt.Errorf("invalid version")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cur, err := getQuestionForCourseTx(ctx, tx, courseID, questionID)
	if err != nil || cur == nil {
		return nil, err
	}
	var snap []byte
	err = tx.QueryRow(ctx, `
SELECT snapshot FROM course.question_versions
WHERE question_id = $1 AND version_number = $2
`, questionID, version).Scan(&snap)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var parsed versionSnapshot
	if err := json.Unmarshal(snap, &parsed); err != nil {
		return nil, err
	}
	if strings.TrimSpace(parsed.QuestionType) == "" || strings.TrimSpace(parsed.Stem) == "" {
		return nil, fmt.Errorf("version snapshot is incomplete")
	}
	nextVersion := cur.VersionNumber + 1
	status := parsed.Status
	if status == "" {
		status = cur.Status
	}
	meta := parsed.Metadata
	if len(meta) == 0 || string(meta) == "null" {
		meta = cur.Metadata
	}
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	published := status == "active"
	_, err = tx.Exec(ctx, `
UPDATE course.questions
SET question_type = $3::course.question_type,
	stem = $4,
	options = $5,
	correct_answer = $6,
	explanation = $7,
	points = $8,
	status = $9::course.question_status,
	shared = $10,
	metadata = $11,
	shuffle_choices_override = $12,
	srs_eligible = $13,
	is_published = $14,
	version_number = $15,
	updated_at = NOW()
WHERE id = $2 AND course_id = $1
`, courseID, questionID, parsed.QuestionType, parsed.Stem, rawOrNil(parsed.Options), rawOrNil(parsed.CorrectAnswer),
		parsed.Explanation, parsed.Points, status, parsed.Shared, meta, parsed.ShuffleChoicesOverride, parsed.SRSEligible,
		published, nextVersion)
	if err != nil {
		return nil, err
	}
	row, err := getQuestionForCourseTx(ctx, tx, courseID, questionID)
	if err != nil {
		return nil, err
	}
	note := changeNote
	if note == nil || strings.TrimSpace(*note) == "" {
		s := fmt.Sprintf("Restored version %d", version)
		note = &s
	}
	if err := insertQuestionVersion(ctx, tx, row, note, editor); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return row, nil
}

type versionSnapshot struct {
	QuestionType           string          `json:"question_type"`
	Stem                   string          `json:"stem"`
	Options                json.RawMessage `json:"options"`
	CorrectAnswer          json.RawMessage `json:"correct_answer"`
	Explanation            *string         `json:"explanation"`
	Points                 float64         `json:"points"`
	Status                 string          `json:"status"`
	Shared                 bool            `json:"shared"`
	Metadata               json.RawMessage `json:"metadata"`
	ShuffleChoicesOverride *bool           `json:"shuffle_choices_override"`
	SRSEligible            bool            `json:"srs_eligible"`
}

func rawOrNil(r json.RawMessage) any {
	if len(r) == 0 || string(r) == "null" {
		return nil
	}
	return []byte(r)
}

func getQuestionForCourseTx(ctx context.Context, tx pgx.Tx, courseID, questionID uuid.UUID) (*QuestionEntity, error) {
	var e QuestionEntity
	err := tx.QueryRow(ctx, `
SELECT id, course_id, question_type::text, stem, options, correct_answer, explanation,
       points::float8, status::text, shared, source, metadata, shuffle_choices_override,
       irt_a::float8, irt_b::float8, irt_c::float8,
       irt_status::text, irt_sample_n, irt_calibrated_at,
       created_by, created_at, updated_at,
       version_number, is_published, srs_eligible
FROM course.questions
WHERE id = $2 AND course_id = $1
`, courseID, questionID).Scan(
		&e.ID, &e.CourseID, &e.QuestionType, &e.Stem, &e.Options, &e.CorrectAnswer, &e.Explanation,
		&e.Points, &e.Status, &e.Shared, &e.Source, &e.Metadata, &e.ShuffleChoicesOverride,
		&e.IrtA, &e.IrtB, &e.IrtC, &e.IrtStatus, &e.IrtSampleN, &e.IrtCalibratedAt,
		&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt,
		&e.VersionNumber, &e.IsPublished, &e.SRSEligible,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func insertQuestionVersion(ctx context.Context, tx pgx.Tx, e *QuestionEntity, note *string, createdBy *uuid.UUID) error {
	if e == nil {
		return fmt.Errorf("missing question")
	}
	snap := versionSnapshot{
		QuestionType:           e.QuestionType,
		Stem:                   e.Stem,
		Options:                e.Options,
		CorrectAnswer:          e.CorrectAnswer,
		Explanation:            e.Explanation,
		Points:                 e.Points,
		Status:                 e.Status,
		Shared:                 e.Shared,
		Metadata:               e.Metadata,
		ShuffleChoicesOverride: e.ShuffleChoicesOverride,
		SRSEligible:            e.SRSEligible,
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO course.question_versions (question_id, version_number, snapshot, change_note, created_by)
VALUES ($1, $2, $3, $4, $5)
`, e.ID, e.VersionNumber, raw, note, createdBy)
	return err
}
