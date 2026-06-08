package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	repo "github.com/lextures/lextures/server/internal/repos/incompletegrades"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

func sweepIncompleteReminders(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.FFIncompleteGradeWorkflow || !cfg.EmailNotificationsEnabled || pool == nil {
		return
	}
	candidates, err := repo.ListDueReminders(ctx, pool, now)
	if err != nil {
		slog.Warn("incomplete_reminders.list", "err", err)
		return
	}
	ns := &notifications.Service{Pool: pool, Config: cfg}
	for _, c := range candidates {
		ns.NotifyIncompleteReminder(
			ctx,
			c.StudentUserID,
			c.InstructorIDs,
			c.CourseTitle,
			c.CourseCode,
			c.StudentName,
			c.DaysRemaining,
			c.ExtensionDeadline,
		)
		if err := repo.MarkReminderSent(ctx, pool, c.ID, c.ReminderKind, now); err != nil {
			slog.Warn("incomplete_reminders.mark_sent", "err", err, "record_id", c.ID)
		}
	}
}

func sweepIncompleteLapse(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.FFIncompleteGradeWorkflow || pool == nil {
		return
	}
	n, err := repo.LapseOverdue(ctx, pool, now)
	if err != nil {
		slog.Warn("incomplete_lapse", "err", err)
		return
	}
	if n > 0 {
		slog.Info("incomplete_lapse", "count", n)
	}
}
