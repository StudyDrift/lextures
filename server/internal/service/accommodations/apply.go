package accommodations

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/accommodationaudit"
)

func AppliedTimeLimit(baseSeconds int32, multiplier float64) int32 {
	if baseSeconds <= 0 {
		return 0
	}
	m := multiplier
	if m < 1 {
		m = 1
	}
	return int32(math.Ceil(float64(baseSeconds) * m))
}

func auditValueJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

func AuditQuizStart(
	ctx context.Context,
	pool *pgxpool.Pool,
	studentID, attemptID uuid.UUID,
	auditEnabled bool,
	eff Effective,
) error {
	if !auditEnabled || pool == nil {
		return nil
	}
	ctxID := attemptID
	if eff.TimeMultiplier > 1.000001 {
		if err := accommodationaudit.Insert(ctx, pool, studentID, "time_extension",
			auditValueJSON(map[string]any{"multiplier": eff.TimeMultiplier}), "quiz_attempt", &ctxID); err != nil {
			return err
		}
	}
	if eff.ReducedDistraction {
		if err := accommodationaudit.Insert(ctx, pool, studentID, "reduced_distraction",
			auditValueJSON(map[string]any{"enabled": true}), "quiz_attempt", &ctxID); err != nil {
			return err
		}
	}
	if eff.HintsAlwaysEnabled {
		if err := accommodationaudit.Insert(ctx, pool, studentID, "always_allow_hints",
			auditValueJSON(map[string]any{"enabled": true}), "quiz_attempt", &ctxID); err != nil {
			return err
		}
	}
	if eff.ExtraAttempts > 0 {
		if err := accommodationaudit.Insert(ctx, pool, studentID, "extra_attempts",
			auditValueJSON(map[string]any{"extraAttempts": eff.ExtraAttempts}), "quiz_attempt", &ctxID); err != nil {
			return err
		}
	}
	return nil
}

func AuditContentView(
	ctx context.Context,
	pool *pgxpool.Pool,
	studentID uuid.UUID,
	contentID *uuid.UUID,
	auditEnabled bool,
	eff Effective,
) error {
	if !auditEnabled || pool == nil {
		return nil
	}
	type entry struct {
		typ   string
		value any
	}
	var entries []entry
	if eff.TTSEnabled {
		entries = append(entries, entry{"tts", map[string]any{"enabled": true}})
	}
	if eff.DyslexiaDisplay {
		entries = append(entries, entry{"dyslexia_display", map[string]any{"enabled": true}})
	}
	if eff.HighContrast {
		entries = append(entries, entry{"high_contrast", map[string]any{"enabled": true}})
	}
	if eff.ReducedMotion {
		entries = append(entries, entry{"reduced_motion", map[string]any{"enabled": true}})
	}
	if eff.SpeechToTextEnabled {
		entries = append(entries, entry{"stt", map[string]any{"enabled": true}})
	}
	for _, e := range entries {
		if err := accommodationaudit.Insert(ctx, pool, studentID, e.typ,
			auditValueJSON(e.value), "content_view", contentID); err != nil {
			return fmt.Errorf("audit %s: %w", e.typ, err)
		}
	}
	return nil
}
