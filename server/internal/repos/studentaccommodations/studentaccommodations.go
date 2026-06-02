// Package studentaccommodations is course.student_accommodations (server/src/repos/student_accommodations.rs).
package studentaccommodations

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is a single accommodation record.
type Row struct {
	ID                     uuid.UUID
	UserID                 uuid.UUID
	CourseID               *uuid.UUID
	TimeMultiplier         float64
	ExtraAttempts          int32
	HintsAlwaysEnabled     bool
	ReducedDistraction     bool
	SpeechToTextEnabled    bool
	TTSEnabled             bool
	DyslexiaDisplayEnabled bool
	HighContrastEnabled    bool
	ReducedMotionEnabled   bool
	SeparateSetting        bool
	Active                 bool
	AlternativeFormat      *string
	EffectiveFrom          sql.NullTime
	EffectiveUntil         sql.NullTime
	CreatedBy              uuid.UUID
	UpdatedBy              *uuid.UUID
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// AccommodationWrite is the mutable payload for insert/update.
type AccommodationWrite struct {
	TimeMultiplier         float64
	ExtraAttempts          int32
	HintsAlwaysEnabled     bool
	ReducedDistraction     bool
	SpeechToTextEnabled    bool
	TTSEnabled             bool
	DyslexiaDisplayEnabled bool
	HighContrastEnabled    bool
	ReducedMotionEnabled   bool
	SeparateSetting        bool
	AlternativeFormat      *string
	EffectiveFrom          *time.Time
	EffectiveUntil         *time.Time
}

// ListRow is a row joined to course code.
type ListRow struct {
	Row        Row
	CourseCode *string
}

const rowSelectCols = `
       (time_multiplier)::double precision,
       extra_attempts, hints_always_enabled, reduced_distraction_mode,
       speech_to_text_enabled,
       tts_enabled, dyslexia_display_enabled, high_contrast_enabled,
       reduced_motion_enabled, separate_setting, active,
       alternative_format, effective_from, effective_until,
       created_by, updated_by, created_at, updated_at`

// rowSelectColsSA qualifies columns for queries that JOIN course.courses (both have created_at).
const rowSelectColsSA = `
       (sa.time_multiplier)::double precision,
       sa.extra_attempts, sa.hints_always_enabled, sa.reduced_distraction_mode,
       sa.speech_to_text_enabled,
       sa.tts_enabled, sa.dyslexia_display_enabled, sa.high_contrast_enabled,
       sa.reduced_motion_enabled, sa.separate_setting, sa.active,
       sa.alternative_format, sa.effective_from, sa.effective_until,
       sa.created_by, sa.updated_by, sa.created_at, sa.updated_at`

// ListForUserWithCourse returns active rows for a user with optional course code.
func ListForUserWithCourse(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]ListRow, error) {
	const q = `
SELECT sa.id, sa.user_id, sa.course_id,` + rowSelectColsSA + `,
       c.course_code AS course_code
FROM course.student_accommodations sa
LEFT JOIN course.courses c ON c.id = sa.course_id
WHERE sa.user_id = $1 AND sa.active = true
ORDER BY sa.course_id NULLS LAST, sa.created_at ASC`
	rows, err := pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ListRow
	for rows.Next() {
		var r ListRow
		var courseID *uuid.UUID
		var updatedBy *uuid.UUID
		var alt sql.NullString
		var cc sql.NullString
		if err := rows.Scan(
			&r.Row.ID, &r.Row.UserID, &courseID,
			&r.Row.TimeMultiplier, &r.Row.ExtraAttempts, &r.Row.HintsAlwaysEnabled, &r.Row.ReducedDistraction,
			&r.Row.SpeechToTextEnabled,
			&r.Row.TTSEnabled, &r.Row.DyslexiaDisplayEnabled, &r.Row.HighContrastEnabled,
			&r.Row.ReducedMotionEnabled, &r.Row.SeparateSetting, &r.Row.Active,
			&alt, &r.Row.EffectiveFrom, &r.Row.EffectiveUntil,
			&r.Row.CreatedBy, &updatedBy, &r.Row.CreatedAt, &r.Row.UpdatedAt,
			&cc,
		); err != nil {
			return nil, err
		}
		r.Row.CourseID = courseID
		r.Row.UpdatedBy = updatedBy
		r.Row.AlternativeFormat = strptr(alt)
		if cc.Valid {
			s := cc.String
			r.CourseCode = &s
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// FindActiveForCourse is the course-specific or nil row active by server date.
func FindActiveForCourse(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (*Row, error) {
	return findOne(ctx, pool, `
SELECT id, user_id, course_id,`+rowSelectCols+`
FROM course.student_accommodations
WHERE user_id = $1 AND course_id = $2 AND active = true
  AND (effective_from IS NULL OR effective_from <= CURRENT_DATE)
  AND (effective_until IS NULL OR effective_until >= CURRENT_DATE)
LIMIT 1`, userID, courseID)
}

// FindActiveGlobal is the global (course_id IS NULL) row active by server date.
func FindActiveGlobal(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Row, error) {
	return findOne(ctx, pool, `
SELECT id, user_id, course_id,`+rowSelectCols+`
FROM course.student_accommodations
WHERE user_id = $1 AND course_id IS NULL AND active = true
  AND (effective_from IS NULL OR effective_from <= CURRENT_DATE)
  AND (effective_until IS NULL OR effective_until >= CURRENT_DATE)
LIMIT 1`, userID)
}

func findOne(ctx context.Context, pool *pgxpool.Pool, q string, args ...any) (*Row, error) {
	var r Row
	var courseID *uuid.UUID
	var updatedBy *uuid.UUID
	var alt sql.NullString
	var effF, effU sql.NullTime
	err := pool.QueryRow(ctx, q, args...).Scan(
		&r.ID, &r.UserID, &courseID,
		&r.TimeMultiplier, &r.ExtraAttempts, &r.HintsAlwaysEnabled, &r.ReducedDistraction,
		&r.SpeechToTextEnabled,
		&r.TTSEnabled, &r.DyslexiaDisplayEnabled, &r.HighContrastEnabled,
		&r.ReducedMotionEnabled, &r.SeparateSetting, &r.Active,
		&alt, &effF, &effU,
		&r.CreatedBy, &updatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.CourseID = courseID
	r.UpdatedBy = updatedBy
	r.EffectiveFrom = effF
	r.EffectiveUntil = effU
	r.AlternativeFormat = strptr(alt)
	return &r, nil
}

// InsertRow creates a new row. createdBy is the acting staff user id.
func InsertRow(
	ctx context.Context, pool *pgxpool.Pool,
	userID uuid.UUID,
	courseID *uuid.UUID,
	w AccommodationWrite,
	createdBy uuid.UUID,
) (*Row, error) {
	const q = `
INSERT INTO course.student_accommodations (
  user_id, course_id, time_multiplier, extra_attempts,
  hints_always_enabled, reduced_distraction_mode, speech_to_text_enabled,
  tts_enabled, dyslexia_display_enabled, high_contrast_enabled,
  reduced_motion_enabled, separate_setting, active,
  alternative_format, effective_from, effective_until, created_by, updated_by
) VALUES ($1, $2, $3::numeric, $4, $5, $6, $7, $8, $9, $10, $11, $12, true, $13, $14, $15, $16, $16)
RETURNING id, user_id, course_id,` + rowSelectCols
	var r Row
	var courseOut *uuid.UUID
	var updatedBy *uuid.UUID
	var alt sql.NullString
	var effF, effU sql.NullTime
	err := pool.QueryRow(ctx, q,
		userID, courseID, w.TimeMultiplier, w.ExtraAttempts,
		w.HintsAlwaysEnabled, w.ReducedDistraction, w.SpeechToTextEnabled,
		w.TTSEnabled, w.DyslexiaDisplayEnabled, w.HighContrastEnabled,
		w.ReducedMotionEnabled, w.SeparateSetting,
		w.AlternativeFormat, w.EffectiveFrom, w.EffectiveUntil, createdBy,
	).Scan(
		&r.ID, &r.UserID, &courseOut, &r.TimeMultiplier, &r.ExtraAttempts, &r.HintsAlwaysEnabled, &r.ReducedDistraction,
		&r.SpeechToTextEnabled,
		&r.TTSEnabled, &r.DyslexiaDisplayEnabled, &r.HighContrastEnabled,
		&r.ReducedMotionEnabled, &r.SeparateSetting, &r.Active,
		&alt, &effF, &effU, &r.CreatedBy, &updatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.CourseID = courseOut
	r.UpdatedBy = updatedBy
	r.EffectiveFrom = effF
	r.EffectiveUntil = effU
	r.AlternativeFormat = strptr(alt)
	return &r, nil
}

// UpdateRow updates if id matches user (learner) id.
func UpdateRow(
	ctx context.Context, pool *pgxpool.Pool,
	id, userID uuid.UUID,
	w AccommodationWrite,
	updatedBy uuid.UUID,
) (*Row, error) {
	const q = `
UPDATE course.student_accommodations
SET time_multiplier = $3::numeric,
    extra_attempts = $4,
    hints_always_enabled = $5,
    reduced_distraction_mode = $6,
    speech_to_text_enabled = $7,
    tts_enabled = $8,
    dyslexia_display_enabled = $9,
    high_contrast_enabled = $10,
    reduced_motion_enabled = $11,
    separate_setting = $12,
    alternative_format = $13,
    effective_from = $14,
    effective_until = $15,
    updated_by = $16,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND active = true
RETURNING id, user_id, course_id,` + rowSelectCols
	var r Row
	var courseID *uuid.UUID
	var uby *uuid.UUID
	var alt sql.NullString
	var effF, effU sql.NullTime
	err := pool.QueryRow(ctx, q,
		id, userID, w.TimeMultiplier, w.ExtraAttempts,
		w.HintsAlwaysEnabled, w.ReducedDistraction, w.SpeechToTextEnabled,
		w.TTSEnabled, w.DyslexiaDisplayEnabled, w.HighContrastEnabled,
		w.ReducedMotionEnabled, w.SeparateSetting,
		w.AlternativeFormat, w.EffectiveFrom, w.EffectiveUntil, updatedBy,
	).Scan(
		&r.ID, &r.UserID, &courseID, &r.TimeMultiplier, &r.ExtraAttempts, &r.HintsAlwaysEnabled, &r.ReducedDistraction,
		&r.SpeechToTextEnabled,
		&r.TTSEnabled, &r.DyslexiaDisplayEnabled, &r.HighContrastEnabled,
		&r.ReducedMotionEnabled, &r.SeparateSetting, &r.Active,
		&alt, &effF, &effU, &r.CreatedBy, &uby, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.CourseID = courseID
	r.UpdatedBy = uby
	r.EffectiveFrom = effF
	r.EffectiveUntil = effU
	r.AlternativeFormat = strptr(alt)
	return &r, nil
}

// DeleteRow soft-deletes a row; returns true if a row was deactivated.
func DeleteRow(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE course.student_accommodations
SET active = false, updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND active = true
`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func strptr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}
