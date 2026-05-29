// Package readingprefs manages per-user dyslexia-friendly and STT reading preferences (plans 12.6, 12.9).
package readingprefs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultSTTLanguage = "en-US"

// Row holds a user's reading preferences.
type Row struct {
	FontFace      string    `json:"fontFace"`
	LetterSpacing string    `json:"letterSpacing"`
	WordSpacing   string    `json:"wordSpacing"`
	LineHeight    string    `json:"lineHeight"`
	RulerEnabled  bool      `json:"rulerEnabled"`
	RulerColor    string    `json:"rulerColor"`
	STTEnabled    bool      `json:"sttEnabled"`
	STTLanguage   string    `json:"sttLanguage"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func defaults() Row {
	return Row{
		FontFace:      "default",
		LetterSpacing: "normal",
		WordSpacing:   "normal",
		LineHeight:    "normal",
		RulerEnabled:  false,
		RulerColor:    "yellow",
		STTEnabled:    false,
		STTLanguage:   defaultSTTLanguage,
		UpdatedAt:     time.Now(),
	}
}

// Get returns the user's reading preferences, falling back to defaults when no row exists.
func Get(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Row, error) {
	r := defaults()
	err := pool.QueryRow(ctx, `
SELECT font_face, letter_spacing, word_spacing, line_height, ruler_enabled, ruler_color,
       stt_enabled, stt_language, updated_at
FROM settings.user_reading_preferences
WHERE user_id = $1
`, userID).Scan(
		&r.FontFace, &r.LetterSpacing, &r.WordSpacing, &r.LineHeight,
		&r.RulerEnabled, &r.RulerColor, &r.STTEnabled, &r.STTLanguage, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &r, nil
		}
		return nil, err
	}
	return &r, nil
}

// Patch holds optional field updates; nil fields are unchanged.
type Patch struct {
	FontFace      *string
	LetterSpacing *string
	WordSpacing   *string
	LineHeight    *string
	RulerEnabled  *bool
	RulerColor    *string
	STTEnabled    *bool
	STTLanguage   *string
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

	var out Row
	err = pool.QueryRow(ctx, `
INSERT INTO settings.user_reading_preferences (
    user_id, font_face, letter_spacing, word_spacing, line_height,
    ruler_enabled, ruler_color, stt_enabled, stt_language, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
ON CONFLICT (user_id) DO UPDATE
    SET font_face      = excluded.font_face,
        letter_spacing = excluded.letter_spacing,
        word_spacing   = excluded.word_spacing,
        line_height    = excluded.line_height,
        ruler_enabled  = excluded.ruler_enabled,
        ruler_color    = excluded.ruler_color,
        stt_enabled    = excluded.stt_enabled,
        stt_language   = excluded.stt_language,
        updated_at     = now()
RETURNING font_face, letter_spacing, word_spacing, line_height, ruler_enabled, ruler_color,
          stt_enabled, stt_language, updated_at
`, userID,
		current.FontFace, current.LetterSpacing, current.WordSpacing, current.LineHeight,
		current.RulerEnabled, current.RulerColor, current.STTEnabled, current.STTLanguage,
	).Scan(
		&out.FontFace, &out.LetterSpacing, &out.WordSpacing, &out.LineHeight,
		&out.RulerEnabled, &out.RulerColor, &out.STTEnabled, &out.STTLanguage, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
