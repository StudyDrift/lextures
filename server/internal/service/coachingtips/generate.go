package coachingtips

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

const systemPrompt = `You are a supportive study-skills coach for a college student.
Write one short, actionable weekly coaching tip (2-3 sentences, under 280 characters).
Use only the aggregate metrics provided — never invent grades, names, or assignments.
Be encouraging, not judgmental. Mention optimal study days when provided.
Do not use markdown or bullet points.`

// GenerateTip produces a coaching tip via AI completion or fallback pool.
func GenerateTip(ctx context.Context, pool *pgxpool.Pool, ai aiprovider.ScopedCompleter, model string, userID uuid.UUID, now time.Time) (tipText string, contextLine string, usedLLM bool, err error) {
	agg, err := LoadAggregateContext(ctx, pool, userID, now)
	if err != nil {
		return "", "", false, err
	}
	contextLine = agg.String()
	weekSeed := fmt.Sprintf("%s:%s", userID.String(), now.Format("2006-01-02"))

	if ai == nil || strings.TrimSpace(model) == "" {
		return PickFallback(weekSeed), contextLine, false, nil
	}

	userMsg := fmt.Sprintf(
		"Metrics: %s\nWrite a personalized weekly study tip.",
		contextLine,
	)
	text, _, genErr := ai.Complete(ctx, model, []aiprovider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	})
	if genErr != nil || strings.TrimSpace(text.Text) == "" {
		return PickFallback(weekSeed), contextLine, false, genErr
	}
	textStr := strings.TrimSpace(text.Text)
	if len(textStr) > 500 {
		textStr = textStr[:500]
	}
	return textStr, contextLine, true, nil
}

// WeekOfMonday returns the Monday date for the week containing t.
func WeekOfMonday(t time.Time) time.Time {
	start, _ := weekBounds(t)
	return start
}

func weekBounds(t time.Time) (time.Time, time.Time) {
	utc := t.UTC()
	wd := int(utc.Weekday())
	if wd == 0 {
		wd = 7
	}
	start := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	start = start.AddDate(0, 0, -(wd - 1))
	end := start.AddDate(0, 0, 7).Add(-time.Nanosecond)
	return start, end
}
