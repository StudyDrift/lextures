package coachingtips

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notificationevents"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	repo "github.com/lextures/lextures/server/internal/repos/studyreflection"
	"github.com/lextures/lextures/server/internal/repos/userai"
	"github.com/lextures/lextures/server/internal/service/notifications"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

// Deliver stores a weekly tip and notifies the student (in-app + optional email).
func Deliver(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, userID uuid.UUID, weekOf time.Time, tipText string) error {
	if err := repo.UpsertCoachingTip(ctx, pool, userID, weekOf, tipText); err != nil {
		return err
	}
	link := strings.TrimRight(strings.TrimSpace(cfg.PublicWebOrigin), "/") + "/me/study-insights"
	if strings.TrimSpace(cfg.PublicWebOrigin) == "" {
		link = "http://localhost:5173/me/study-insights"
	}
	_, _ = notificationsinbox.Insert(ctx, pool, userID, notificationevents.CoachingTipWeekly,
		"Your weekly study tip", tipText, link)
	ns := &notifications.Service{Pool: pool, Config: cfg}
	_ = ns.EnqueueEmail(ctx, userID, notificationevents.CoachingTipWeekly, "coaching_tip", map[string]string{
		"subject": "Your weekly study coaching tip",
		"tipText": tipText,
		"link":    link,
	}, nil)
	return nil
}

// RunWeeklyBatch generates tips for opted-in users missing a tip this week.
func RunWeeklyBatch(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, or *openrouter.Client, now time.Time) (int, error) {
	if !cfg.SelfReflectionEnabled || pool == nil {
		return 0, nil
	}
	weekOf := WeekOfMonday(now)
	ids, err := repo.ListOptedInUserIDs(ctx, pool, 500)
	if err != nil {
		return 0, err
	}
	generated := 0
	for _, uid := range ids {
		has, err := repo.HasCoachingTipForWeek(ctx, pool, uid, weekOf)
		if err != nil || has {
			continue
		}
		model := "openai/gpt-4o-mini"
		if m, err := userai.GetCourseSetupModelID(ctx, pool, uid); err == nil && strings.TrimSpace(m) != "" {
			model = strings.TrimSpace(m)
		}
		tip, _, _, err := GenerateTip(ctx, pool, or, model, uid, now)
		if err != nil || strings.TrimSpace(tip) == "" {
			continue
		}
		if err := Deliver(ctx, pool, cfg, uid, weekOf, tip); err != nil {
			continue
		}
		generated++
	}
	return generated, nil
}
