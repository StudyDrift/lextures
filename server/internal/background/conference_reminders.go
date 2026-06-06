package background

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/conferences"
	"github.com/lextures/lextures/server/internal/service/icsgenerator"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

func sweepConferenceReminders(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.FFConferenceScheduling || !cfg.EmailNotificationsEnabled || pool == nil {
		return
	}
	candidates, err := conferences.ListDueReminders(ctx, pool, now)
	if err != nil {
		slog.Warn("conference_reminders.list", "err", err)
		return
	}
	ns := &notifications.Service{Pool: pool, Config: cfg}
	for _, c := range candidates {
		when := c.StartAt.UTC().Format("Mon Jan 2, 2006 3:04 PM MST")
		location := c.Location
		if c.VideoLink != "" {
			location = c.VideoLink
		}
		summary := fmt.Sprintf("Parent-Teacher Conference with %s (%s)", c.Teacher, c.ChildName)
		ics := icsgenerator.BuildEvent(icsgenerator.Event{
			UID:       icsgenerator.ConferenceUID(c.SlotID.String()),
			Summary:   summary,
			Location:  location,
			Start:     c.StartAt,
			End:       c.EndAt,
			Organizer: "School",
		})
		vars := map[string]string{
			"when":        when,
			"summary":     summary,
			"location":    location,
			"icsContent":  ics,
			"icsFilename": fmt.Sprintf("conference-%s.ics", c.SlotID.String()[:8]),
		}
		if err := ns.EnqueueEmail(ctx, c.ParentID, notifications.EventConferenceReminder, "conference_reminder", vars, nil); err != nil {
			slog.Warn("conference_reminders.parent", "err", err, "slot_id", c.SlotID)
			continue
		}
		if err := ns.EnqueueEmail(ctx, c.TeacherID, notifications.EventConferenceReminder, "conference_reminder", vars, nil); err != nil {
			slog.Warn("conference_reminders.teacher", "err", err, "slot_id", c.SlotID)
			continue
		}
		if err := conferences.MarkReminderSent(ctx, pool, c.SlotID, now); err != nil {
			slog.Warn("conference_reminders.mark_sent", "err", err, "slot_id", c.SlotID)
		}
	}
}
