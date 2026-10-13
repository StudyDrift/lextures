package parentprogress

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notificationevents"
	"github.com/lextures/lextures/server/internal/repos/emailjobs"
	"github.com/lextures/lextures/server/internal/repos/notificationprefs"
	repo "github.com/lextures/lextures/server/internal/repos/parentprogress"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

// Run sends the weekly parent progress email to parents whose local weekday and hour match.
func Run(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) (int, error) {
	if pool == nil || !cfg.EmailNotificationsEnabled {
		return 0, nil
	}
	weekday := NormalizeWeekday(cfg.ParentWeeklyProgressWeekday)
	hour := NormalizeHour(cfg.ParentWeeklyProgressHour)
	emptyMode := NormalizeEmptyMode(cfg.ParentWeeklyProgressEmpty)
	parents, err := repo.ListParents(ctx, pool, 2000)
	if err != nil {
		return 0, err
	}
	ns := &notifications.Service{Pool: pool, Config: cfg}
	origin := strings.TrimRight(strings.TrimSpace(cfg.PublicWebOrigin), "/")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	sent := 0
	for _, parent := range parents {
		if user.IsManagedEmail(parent.Email) {
			continue
		}
		local := LocalNow(now, parent.Timezone)
		if !ShouldSend(local, weekday, hour) {
			continue
		}
		pref, err := notificationprefs.Get(ctx, pool, parent.ID, notificationevents.ParentWeeklyProgress)
		if err != nil {
			slog.Warn("parent_weekly_progress.prefs", "err", err, "user_id", parent.ID)
			continue
		}
		if !pref.EmailEnabled || pref.DigestMode == "off" {
			continue
		}
		weekStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
		claimed, err := repo.Claim(ctx, pool, parent.ID, weekStart)
		if err != nil || !claimed {
			continue
		}
		msg, err := buildMessage(ctx, pool, parent, local)
		if err != nil {
			_ = repo.Release(ctx, pool, parent.ID, weekStart)
			slog.Warn("parent_weekly_progress.build", "err", err, "user_id", parent.ID)
			continue
		}
		if msg.NoteOnly && emptyMode != EmptyNote {
			continue
		}
		if !msg.NoteOnly && msg.BodyText == "" {
			continue
		}
		vars := map[string]string{
			"subject":        msg.Subject,
			"bodyText":       msg.BodyText,
			"bodyHtml":       msg.BodyHTML,
			"link":           origin + "/parent",
			"unsubscribeUrl": ns.UnsubscribeURL(parent.ID, notificationevents.ParentWeeklyProgress),
		}
		if _, err := emailjobs.Enqueue(ctx, pool, parent.ID, notificationevents.ParentWeeklyProgress, msg.Subject, "parent_weekly_progress", vars); err != nil {
			_ = repo.Release(ctx, pool, parent.ID, weekStart)
			slog.Warn("parent_weekly_progress.enqueue", "err", err, "user_id", parent.ID)
			continue
		}
		body := "Your weekly progress summary is ready."
		if msg.NoteOnly {
			body = "No learning activity this week."
		}
		if err := (&notifications.PushService{Pool: pool, Config: cfg}).Enqueue(ctx, parent.ID, notificationevents.ParentWeeklyProgress, msg.Subject, body, "/parent"); err != nil {
			slog.Warn("parent_weekly_progress.in_app", "err", err, "user_id", parent.ID)
		}
		sent++
	}
	return sent, nil
}

func buildMessage(ctx context.Context, pool *pgxpool.Pool, parent repo.Parent, local time.Time) (Message, error) {
	learners, err := repo.ListLearners(ctx, pool, parent.ID)
	if err != nil {
		return Message{}, err
	}
	pastStart, now, nextEnd := WeekWindow(local)
	sections := make([]Section, 0, len(learners))
	for _, learner := range learners {
		completed, err := repo.CompletedItems(ctx, pool, learner.ID, pastStart, now)
		if err != nil {
			return Message{}, err
		}
		due, err := repo.DueItems(ctx, pool, learner.ID, now, nextEnd)
		if err != nil {
			return Message{}, err
		}
		grades, err := repo.GradeChanges(ctx, pool, learner.ID, pastStart, now)
		if err != nil {
			return Message{}, err
		}
		comments, err := repo.RecentComments(ctx, pool, learner.ID, pastStart)
		if err != nil {
			return Message{}, err
		}
		seconds, err := repo.TimeSpentSeconds(ctx, pool, learner.ID, pastStart, now)
		if err != nil {
			return Message{}, err
		}
		sections = append(sections, BuildSection(learner.Name, completed, due, grades, comments, seconds, pastStart))
	}
	if !AnyActivity(sections) {
		return Compose(parent.DisplayName, nil, true), nil
	}
	return Compose(parent.DisplayName, sections, false), nil
}
