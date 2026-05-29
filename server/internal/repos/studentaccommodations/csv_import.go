package studentaccommodations

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyCSVRow(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	accommodationType, value string,
	createdBy uuid.UUID,
) (bool, error) {
	existing, err := FindActiveGlobal(ctx, pool, userID)
	if err != nil {
		return false, err
	}
	w := AccommodationWrite{TimeMultiplier: 1}
	if existing != nil {
		w = rowToWrite(existing)
	}
	if err := applyCSVField(&w, accommodationType, value); err != nil {
		return false, err
	}
	if existing == nil {
		_, err = InsertRow(ctx, pool, userID, nil, w, createdBy)
		return true, err
	}
	_, err = UpdateRow(ctx, pool, existing.ID, userID, w, createdBy)
	return false, err
}

func rowToWrite(r *Row) AccommodationWrite {
	var ef, eu *time.Time
	if r.EffectiveFrom.Valid {
		t := r.EffectiveFrom.Time
		ef = &t
	}
	if r.EffectiveUntil.Valid {
		t := r.EffectiveUntil.Time
		eu = &t
	}
	return AccommodationWrite{
		TimeMultiplier:         r.TimeMultiplier,
		ExtraAttempts:          r.ExtraAttempts,
		HintsAlwaysEnabled:     r.HintsAlwaysEnabled,
		ReducedDistraction:     r.ReducedDistraction,
		SpeechToTextEnabled:    r.SpeechToTextEnabled,
		TTSEnabled:             r.TTSEnabled,
		DyslexiaDisplayEnabled: r.DyslexiaDisplayEnabled,
		HighContrastEnabled:    r.HighContrastEnabled,
		ReducedMotionEnabled:   r.ReducedMotionEnabled,
		SeparateSetting:        r.SeparateSetting,
		AlternativeFormat:      r.AlternativeFormat,
		EffectiveFrom:          ef,
		EffectiveUntil:         eu,
	}
}

func applyCSVField(w *AccommodationWrite, accommodationType, value string) error {
	typ := strings.ToLower(strings.TrimSpace(accommodationType))
	val := strings.TrimSpace(value)
	switch typ {
	case "time_extension", "extended_time", "time_multiplier":
		f, err := strconv.ParseFloat(val, 64)
		if err != nil || f < 1 || f > 99.99 {
			return errors.New("time_extension value must be a multiplier between 1.0 and 99.99")
		}
		w.TimeMultiplier = f
	case "extra_attempts":
		n, err := strconv.ParseInt(val, 10, 32)
		if err != nil || n < 0 || n > 500 {
			return errors.New("extra_attempts value must be between 0 and 500")
		}
		w.ExtraAttempts = int32(n)
	case "tts", "text_to_speech":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.TTSEnabled = b
	case "stt", "speech_to_text":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.SpeechToTextEnabled = b
	case "dyslexia_display":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.DyslexiaDisplayEnabled = b
	case "high_contrast":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.HighContrastEnabled = b
	case "reduced_motion":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.ReducedMotionEnabled = b
	case "reduced_distraction", "reduced_distraction_mode":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.ReducedDistraction = b
	case "always_allow_hints", "hints_always_enabled":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.HintsAlwaysEnabled = b
	case "separate_setting":
		b, err := parseBoolValue(val)
		if err != nil {
			return err
		}
		w.SeparateSetting = b
	default:
		return errors.New("unknown accommodation_type: " + typ)
	}
	return nil
}

func parseBoolValue(val string) (bool, error) {
	switch strings.ToLower(val) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true, nil
	case "0", "false", "no", "n", "off", "disabled":
		return false, nil
	default:
		return false, errors.New("boolean value must be true/false, 1/0, or yes/no")
	}
}

var _ = pgx.ErrNoRows
