// Package readingprefs manages per-user reading preferences: dyslexia display (12.6), TTS (12.8), STT (12.9), accommodations overrides (12.10).
package readingprefs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	acsvc "github.com/lextures/lextures/server/internal/service/accommodations"
)

const defaultSTTLanguage = "en-US"

// Row holds a user's reading preferences.
type Row struct {
	FontFace               string    `json:"fontFace"`
	LetterSpacing          string    `json:"letterSpacing"`
	WordSpacing            string    `json:"wordSpacing"`
	LineHeight             string    `json:"lineHeight"`
	RulerEnabled           bool      `json:"rulerEnabled"`
	RulerColor             string    `json:"rulerColor"`
	TTSEnabled             bool      `json:"ttsEnabled"`
	TTSSpeed               float64   `json:"ttsSpeed"`
	TTSVoiceName           *string   `json:"ttsVoiceName,omitempty"`
	STTEnabled             bool      `json:"sttEnabled"`
	STTLanguage            string    `json:"sttLanguage"`
	DyslexiaDisplayEnabled bool      `json:"dyslexiaDisplayEnabled"`
	HighContrastEnabled    bool      `json:"highContrastEnabled"`
	ReducedMotionEnabled   bool      `json:"reducedMotionEnabled"`
	// UIModeOverride stores the admin-set override; nil means derive from grade_level (plan 13.11).
	UIModeOverride *string   `json:"uiModeOverride,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// AccommodationOverrides marks fields forced by an active accommodation plan.
type AccommodationOverrides struct {
	TTSEnabled             bool `json:"ttsEnabled,omitempty"`
	DyslexiaDisplayEnabled bool `json:"dyslexiaDisplayEnabled,omitempty"`
	HighContrastEnabled    bool `json:"highContrastEnabled,omitempty"`
	ReducedMotionEnabled   bool `json:"reducedMotionEnabled,omitempty"`
	STTEnabled             bool `json:"sttEnabled,omitempty"`
}

func defaults() Row {
	return Row{
		FontFace:      "default",
		LetterSpacing: "normal",
		WordSpacing:   "normal",
		LineHeight:    "normal",
		RulerEnabled:  false,
		RulerColor:    "yellow",
		TTSEnabled:    false,
		TTSSpeed:      1.0,
		STTEnabled:    false,
		STTLanguage:   defaultSTTLanguage,
		UpdatedAt:     time.Now(),
	}
}

// Get returns the user's reading preferences, falling back to defaults when no row exists.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Row, error) {
	r := defaults()
	var voice *string
	err := pool.QueryRow(ctx, `
SELECT font_face, letter_spacing, word_spacing, line_height, ruler_enabled, ruler_color,
       tts_enabled, (tts_speed)::double precision, tts_voice_name,
       stt_enabled, stt_language,
       dyslexia_display_enabled, high_contrast_enabled, reduced_motion_enabled,
       ui_mode_override,
       updated_at
FROM settings.user_reading_preferences
WHERE user_id = $1
`, userID).Scan(
		&r.FontFace, &r.LetterSpacing, &r.WordSpacing, &r.LineHeight,
		&r.RulerEnabled, &r.RulerColor,
		&r.TTSEnabled, &r.TTSSpeed, &voice,
		&r.STTEnabled, &r.STTLanguage,
		&r.DyslexiaDisplayEnabled, &r.HighContrastEnabled, &r.ReducedMotionEnabled,
		&r.UIModeOverride,
		&r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &r, nil
		}
		return nil, err
	}
	r.TTSVoiceName = voice
	return &r, nil
}

// Patch holds optional field updates; nil fields are unchanged.
type Patch struct {
	FontFace               *string
	LetterSpacing          *string
	WordSpacing            *string
	LineHeight             *string
	RulerEnabled           *bool
	RulerColor             *string
	TTSEnabled             *bool
	TTSSpeed               *float64
	TTSVoiceName           **string
	STTEnabled             *bool
	STTLanguage            *string
	DyslexiaDisplayEnabled *bool
	HighContrastEnabled    *bool
	ReducedMotionEnabled   *bool
}

var validFontFace = map[string]bool{
	"default": true, "open-dyslexic": true, "atkinson": true, "system": true,
}
var validSpacing = map[string]bool{
	"normal": true, "wide": true, "wider": true,
}
var validLineHeight = map[string]bool{
	"normal": true, "tall": true, "taller": true,
}
var validRulerColor = map[string]bool{
	"yellow": true, "grey": true,
}

// Validate returns a non-nil error if p contains an out-of-range value.
func (p Patch) Validate() error {
	if p.FontFace != nil && !validFontFace[*p.FontFace] {
		return errors.New("fontFace must be one of: default, open-dyslexic, atkinson, system")
	}
	if p.LetterSpacing != nil && !validSpacing[*p.LetterSpacing] {
		return errors.New("letterSpacing must be one of: normal, wide, wider")
	}
	if p.WordSpacing != nil && !validSpacing[*p.WordSpacing] {
		return errors.New("wordSpacing must be one of: normal, wide, wider")
	}
	if p.LineHeight != nil && !validLineHeight[*p.LineHeight] {
		return errors.New("lineHeight must be one of: normal, tall, taller")
	}
	if p.RulerColor != nil && !validRulerColor[*p.RulerColor] {
		return errors.New("rulerColor must be one of: yellow, grey")
	}
	if p.TTSSpeed != nil {
		s := *p.TTSSpeed
		if s < 0.75 || s > 2.0 {
			return fmt.Errorf("ttsSpeed must be between 0.75 and 2.0")
		}
	}
	return nil
}

// Upsert creates or merges reading preferences for the user and returns the saved row.
func Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, p Patch) (*Row, error) {
	current, err := Get(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	if p.FontFace != nil {
		current.FontFace = *p.FontFace
	}
	if p.LetterSpacing != nil {
		current.LetterSpacing = *p.LetterSpacing
	}
	if p.WordSpacing != nil {
		current.WordSpacing = *p.WordSpacing
	}
	if p.LineHeight != nil {
		current.LineHeight = *p.LineHeight
	}
	if p.RulerEnabled != nil {
		current.RulerEnabled = *p.RulerEnabled
	}
	if p.RulerColor != nil {
		current.RulerColor = *p.RulerColor
	}
	if p.TTSEnabled != nil {
		current.TTSEnabled = *p.TTSEnabled
	}
	if p.TTSSpeed != nil {
		current.TTSSpeed = *p.TTSSpeed
	}
	if p.TTSVoiceName != nil {
		current.TTSVoiceName = *p.TTSVoiceName
	}
	if p.STTEnabled != nil {
		current.STTEnabled = *p.STTEnabled
	}
	if p.STTLanguage != nil {
		lang := *p.STTLanguage
		if lang == "" {
			lang = defaultSTTLanguage
		}
		current.STTLanguage = lang
	}
	if p.DyslexiaDisplayEnabled != nil {
		current.DyslexiaDisplayEnabled = *p.DyslexiaDisplayEnabled
	}
	if p.HighContrastEnabled != nil {
		current.HighContrastEnabled = *p.HighContrastEnabled
	}
	if p.ReducedMotionEnabled != nil {
		current.ReducedMotionEnabled = *p.ReducedMotionEnabled
	}

	var out Row
	var voice *string
	err = pool.QueryRow(ctx, `
INSERT INTO settings.user_reading_preferences (
    user_id, font_face, letter_spacing, word_spacing, line_height,
    ruler_enabled, ruler_color, tts_enabled, tts_speed, tts_voice_name,
    stt_enabled, stt_language,
    dyslexia_display_enabled, high_contrast_enabled, reduced_motion_enabled,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::numeric, $10, $11, $12, $13, $14, $15, now())
ON CONFLICT (user_id) DO UPDATE
    SET font_face                = excluded.font_face,
        letter_spacing           = excluded.letter_spacing,
        word_spacing             = excluded.word_spacing,
        line_height              = excluded.line_height,
        ruler_enabled            = excluded.ruler_enabled,
        ruler_color              = excluded.ruler_color,
        tts_enabled              = excluded.tts_enabled,
        tts_speed                = excluded.tts_speed,
        tts_voice_name           = excluded.tts_voice_name,
        stt_enabled              = excluded.stt_enabled,
        stt_language             = excluded.stt_language,
        dyslexia_display_enabled = excluded.dyslexia_display_enabled,
        high_contrast_enabled    = excluded.high_contrast_enabled,
        reduced_motion_enabled   = excluded.reduced_motion_enabled,
        updated_at               = now()
RETURNING font_face, letter_spacing, word_spacing, line_height, ruler_enabled, ruler_color,
          tts_enabled, (tts_speed)::double precision, tts_voice_name,
          stt_enabled, stt_language,
          dyslexia_display_enabled, high_contrast_enabled, reduced_motion_enabled,
          ui_mode_override,
          updated_at
`, userID,
		current.FontFace, current.LetterSpacing, current.WordSpacing, current.LineHeight,
		current.RulerEnabled, current.RulerColor,
		current.TTSEnabled, current.TTSSpeed, current.TTSVoiceName,
		current.STTEnabled, current.STTLanguage,
		current.DyslexiaDisplayEnabled, current.HighContrastEnabled, current.ReducedMotionEnabled,
	).Scan(
		&out.FontFace, &out.LetterSpacing, &out.WordSpacing, &out.LineHeight,
		&out.RulerEnabled, &out.RulerColor,
		&out.TTSEnabled, &out.TTSSpeed, &voice,
		&out.STTEnabled, &out.STTLanguage,
		&out.DyslexiaDisplayEnabled, &out.HighContrastEnabled, &out.ReducedMotionEnabled,
		&out.UIModeOverride,
		&out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	out.TTSVoiceName = voice
	return &out, nil
}

// MergeAccommodationOverrides applies active accommodation display settings over user prefs.
func MergeAccommodationOverrides(row *Row, eff acsvc.Effective) (*Row, AccommodationOverrides) {
	out := *row
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
	return &out, overrides
}

// EffectiveForCourse loads prefs and merges course-scoped accommodations.
func EffectiveForCourse(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (*Row, AccommodationOverrides, error) {
	row, err := Get(ctx, pool, userID)
	if err != nil {
		return nil, AccommodationOverrides{}, err
	}
	eff := acsvc.ResolveEffectiveOrDefault(ctx, pool, userID, courseID)
	merged, overrides := MergeAccommodationOverrides(row, eff)
	return merged, overrides, nil
}
