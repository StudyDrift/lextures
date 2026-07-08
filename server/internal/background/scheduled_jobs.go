package background

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/publicapi"
	"github.com/lextures/lextures/server/internal/repos/apitokens"
	subrepo "github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	tutorsessionrepo "github.com/lextures/lextures/server/internal/repos/tutorsession"
	webhooksrepo "github.com/lextures/lextures/server/internal/repos/webhooks"
	"github.com/lextures/lextures/server/internal/scheduler"
	learnerprofilesvc "github.com/lextures/lextures/server/internal/service/learnerprofile"
	introcourseservice "github.com/lextures/lextures/server/internal/service/introcourse"
)

// dueReminderWindow is how far ahead the due_date_reminder job looks for
// assignments coming due (plan 17.4 FR-4). The daily schedule plus a 24h window
// gives every student one reminder the day before.
const dueReminderWindow = 24 * time.Hour

// inactiveIntegrationThreshold is the activity gap after which a webhook
// subscription is flagged as inactive (plan 17.4 FR-4 inactive_integration_alert).
const inactiveIntegrationThreshold = 12 * time.Hour

// registerScheduledJobs registers handlers for the built-in scheduled job types
// (plan 17.4 FR-4). The scheduler enqueues these onto the durable queue; the
// handlers run here under the same retry/dead-letter machinery as any other job.
func registerScheduledJobs(r *Registry, pool *pgxpool.Pool, cfgSrc ConfigSource) {
	cfg := cfgSrc.Config()
	r.Register(scheduler.JobTypeLateSubmissionSweep, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		n, err := subrepo.MarkOverdueLate(ctx, pool, time.Now().UTC())
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.late_submission_sweep", "marked", n)
		}
		return nil
	}))

	r.Register(scheduler.JobTypeExpiredTokenCleanup, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		n, err := apitokens.DeleteExpiredAndRevoked(ctx, pool, time.Now().UTC())
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.expired_token_cleanup", "deleted", n)
		}
		return nil
	}))

	r.Register(scheduler.JobTypeRequestLogRetention, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		cutoff := time.Now().UTC().Add(-publicapi.RequestLogRetention)
		n, err := publicapi.DeleteRequestLogsOlderThan(ctx, pool, cutoff)
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.request_log_retention", "deleted", n, "cutoff", cutoff)
		}
		return nil
	}))

	r.Register(scheduler.JobTypeDueDateReminder, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		return runDueDateReminder(ctx, pool, cfg, time.Now().UTC())
	}))

	r.Register(scheduler.JobTypeInactiveIntegration, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		rows, err := webhooksrepo.ListInactiveSubscriptions(ctx, pool, inactiveIntegrationThreshold, time.Now().UTC())
		if err != nil {
			return err
		}
		for _, sub := range rows {
			slog.Warn("scheduled.inactive_integration_alert",
				"subscription_id", sub.ID, "org_id", sub.OrgID, "label", sub.Label, "last_activity", sub.LastActivity)
		}
		if len(rows) > 0 {
			slog.Info("scheduled.inactive_integration_alert", "flagged", len(rows))
		}
		return nil
	}))

	r.Register(scheduler.JobTypeLearnerProfileFull, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		if !cfg.LearnerProfileEnabled {
			return nil
		}
		_, err := learnerprofilesvc.EnqueueFull(ctx, pool)
		return err
	}))

	r.Register(scheduler.JobTypeLearnerProfileRetention, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		if !cfg.LearnerProfileEnabled {
			return nil
		}
		n, err := learnerprofilesvc.PurgeInactiveProfiles(ctx, pool, learnerprofilesvc.DefaultRetentionDays)
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.learner_profile_retention", "deleted", n)
		}
		return nil
	}))

	r.Register(scheduler.JobTypeIntroCourseBackfill, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		live := cfgSrc.Config()
		if !live.IntroCourseEnabled {
			return nil
		}
		_, err := introcourseservice.EnqueueBackfillIfNeeded(ctx, pool, live)
		return err
	}))

	r.Register(scheduler.JobTypeIntroCourseCompletionSweep, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		live := cfgSrc.Config()
		if !live.IntroCourseEnabled || pool == nil {
			return nil
		}
		svc := introcourseservice.New(pool)
		courseID, ok, err := svc.CourseID(ctx)
		if err != nil {
			return err
		}
		if !ok || courseID == uuid.Nil {
			return nil
		}
		n, err := introcourseservice.SweepIncompleteCompletions(ctx, pool, live, courseID)
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.intro_course_completion_sweep", "newly_completed", n)
		}
		return nil
	}))

	r.Register(scheduler.JobTypeTutorSessionRetention, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		orgs, err := tutorsessionrepo.ListOrgRetentionDays(ctx, pool)
		if err != nil {
			return err
		}
		var total int64
		for _, org := range orgs {
			n, err := tutorsessionrepo.PurgeExpiredSessions(ctx, pool, org.OrgID, org.RetentionDays)
			if err != nil {
				return err
			}
			total += n
		}
		if total > 0 {
			slog.Info("scheduled.tutor_session_retention", "deleted", total)
		}
		return nil
	}))
}

// runDueDateReminder enqueues an email reminder for each student with an
// assignment coming due who has not yet submitted (plan 17.4 FR-4). It is a
// no-op when email notifications are disabled. Each reminder is deduped per
// student+item+day via the queue unique_key so re-running the sweep does not
// double-send.
func runDueDateReminder(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) error {
	if !cfg.EmailNotificationsEnabled {
		return nil
	}
	targets, err := subrepo.ListUpcomingDueReminders(ctx, pool, now, dueReminderWindow, 1000)
	if err != nil {
		return err
	}
	day := now.Format("2006-01-02")
	enqueued := 0
	for _, t := range targets {
		uniqueKey := fmt.Sprintf("due_reminder:%s:%s:%s", t.ModuleItemID, t.StudentUserID, day)
		_, err := EnqueueEmail(ctx, pool, EmailDeliveryPayload{
			RecipientID: t.StudentUserID,
			EventType:   "assignment_due_reminder",
			Template:    "assignment_due_reminder",
			TemplateVars: map[string]string{
				"courseName":     t.CourseTitle,
				"assignmentName": t.AssignmentTitle,
				"dueAt":          t.DueAt.Format("Jan 2, 2006 15:04 MST"),
			},
		}, uniqueKey)
		if err != nil {
			slog.Warn("scheduled.due_date_reminder.enqueue", "student", t.StudentUserID, "item", t.ModuleItemID, "err", err)
			continue
		}
		enqueued++
	}
	if enqueued > 0 {
		slog.Info("scheduled.due_date_reminder", "enqueued", enqueued)
	}
	return nil
}
