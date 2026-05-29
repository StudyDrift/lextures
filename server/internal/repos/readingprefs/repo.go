// Package readingprefs stores per-user reading and accessibility preferences (plan 12.6+).
package readingprefs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	acsvc "github.com/lextures/lextures/server/internal/service/accommodations"
)

// Row is the reading and accessibility preference slice for a user.
type Row struct {
	UserID                 uuid.UUID
	STTEnabled             bool
	STTLanguage            string
	TTSEnabled             bool
	DyslexiaDisplayEnabled bool
	HighContrastEnabled    bool
	ReducedMotionEnabled   bool
	UpdatedAt              time.Time
}

// AccommodationOverrides marks fields forced by an active accommodation plan.
type AccommodationOverrides struct {
	TTSEnabled             bool `json:"ttsEnabled,omitempty"`
	DyslexiaDisplayEnabled bool `json:"dyslexiaDisplayEnabled,omitempty"`
	HighContrastEnabled    bool `json:"highContrastEnabled,omitempty"`
	ReducedMotionEnabled   bool `json:"reducedMotionEnabled,omitempty"`
	STTEnabled             bool `json:"sttEnabled,omitempty"`
}

const defaultSTTLanguage = "en-US"

// Get returns preferences for a user, creating defaults when missing.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (Row, error) {
	const q = `
SELECT user_id, stt_enabled, stt_language, tts_enabled, dyslexia_display_enabled,
       high_contrast_enabled, reduced_motion_enabled, updated_at
FROM settings.user_reading_preferences
WHERE user_id = $1`
	var r Row
	err := pool.QueryRow(ctx, q, userID).Scan(
		&r.UserID, &r.STTEnabled, &r.STTLanguage,
		&r.TTSEnabled, &r.DyslexiaDisplayEnabled, &r.HighContrastEnabled, &r.ReducedMotionEnabled,
		&r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{
			UserID:      userID,
			STTEnabled:  false,
			STTLanguage: defaultSTTLanguage,
			UpdatedAt:   time.Now().UTC(),
		}, nil
	}
	if err != nil {
		return Row{}, err
	}
	return r, nil
}

// Patch updates one or more fields; upserts the row when absent.
func Patch(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	sttEnabled *bool,
	sttLanguage *string,
	ttsEnabled *bool,
	dyslexiaDisplay *bool,
	highContrast *bool,
	reducedMotion *bool,
) (Row, error) {
	cur, err := Get(ctx, pool, userID)
	if err != nil {
		return Row{}, err
	}
	enabled := cur.STTEnabled
	lang := cur.STTLanguage
	tts := cur.TTSEnabled
	dys := cur.DyslexiaDisplayEnabled
	hc := cur.HighContrastEnabled
	rm := cur.ReducedMotionEnabled
	if sttEnabled != nil {
		enabled = *sttEnabled
	}
	if sttLanguage != nil {
		trimmed := *sttLanguage
		if trimmed == "" {
			trimmed = defaultSTTLanguage
		}
		lang = trimmed
	}
	if ttsEnabled != nil {
		tts = *ttsEnabled
	}
	if dyslexiaDisplay != nil {
		dys = *dyslexiaDisplay
	}
	if highContrast != nil {
		hc = *highContrast
	}
	if reducedMotion != nil {
		rm = *reducedMotion
	}
	const q = `
INSERT INTO settings.user_reading_preferences (
  user_id, stt_enabled, stt_language, tts_enabled, dyslexia_display_enabled,
  high_contrast_enabled, reduced_motion_enabled, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (user_id) DO UPDATE
SET stt_enabled = EXCLUDED.stt_enabled,
    stt_language = EXCLUDED.stt_language,
    tts_enabled = EXCLUDED.tts_enabled,
    dyslexia_display_enabled = EXCLUDED.dyslexia_display_enabled,
    high_contrast_enabled = EXCLUDED.high_contrast_enabled,
    reduced_motion_enabled = EXCLUDED.reduced_motion_enabled,
    updated_at = NOW()
RETURNING user_id, stt_enabled, stt_language, tts_enabled, dyslexia_display_enabled,
          high_contrast_enabled, reduced_motion_enabled, updated_at`
	var r Row
	err = pool.QueryRow(ctx, q, userID, enabled, lang, tts, dys, hc, rm).Scan(
		&r.UserID, &r.STTEnabled, &r.STTLanguage,
		&r.TTSEnabled, &r.DyslexiaDisplayEnabled, &r.HighContrastEnabled, &r.ReducedMotionEnabled,
		&r.UpdatedAt,
	)
	return r, err
}

// MergeAccommodationOverrides applies active accommodation display settings over user prefs.
func MergeAccommodationOverrides(row Row, eff acsvc.Effective) (Row, AccommodationOverrides) {
	out := row
	var overrides AccommodationOverrides
	if eff.TTSEnabled {
		out.TTSEnabled = true
		overrides.TTSEnabled = true
	}
	if eff.DyslexiaDisplay {
		out.DyslexiaDisplayEnabled = true
		overrides.DyslexiaDisplayEnabled = true
	}
	if eff.HighContrast {
		out.HighContrastEnabled = true
		overrides.HighContrastEnabled = true
	}
	if eff.ReducedMotion {
		out.ReducedMotionEnabled = true
		overrides.ReducedMotionEnabled = true
	}
	if eff.SpeechToTextEnabled {
		out.STTEnabled = true
		overrides.STTEnabled = true
	}
	return out, overrides
}

// EffectiveForCourse loads prefs and merges course-scoped accommodations.
func EffectiveForCourse(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (Row, AccommodationOverrides, error) {
	row, err := Get(ctx, pool, userID)
	if err != nil {
		return Row{}, AccommodationOverrides{}, err
	}
	eff := acsvc.ResolveEffectiveOrDefault(ctx, pool, userID, courseID)
	merged, overrides := MergeAccommodationOverrides(row, eff)
	return merged, overrides, nil
}
