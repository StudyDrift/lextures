package scheduler

// Job type identifiers for the built-in scheduled jobs. The scheduler enqueues a
// jobs.queue row with one of these as job_type; a handler registered in the
// background worker performs the actual work (plan 17.4 FR-3, FR-4).
const (
	JobTypeLateSubmissionSweep = "scheduled.late_submission_sweep"
	JobTypeExpiredTokenCleanup = "scheduled.expired_token_cleanup"
	JobTypeRequestLogRetention = "scheduled.request_log_retention"
	JobTypeDueDateReminder     = "scheduled.due_date_reminder"
	JobTypeInactiveIntegration = "scheduled.inactive_integration_alert"
	JobTypeTutorSessionRetention = "scheduled.tutor_session_retention"
)

// ScheduledJob is one configuration-driven entry in the schedule list. New
// scheduled jobs are added here and given a handler in the worker — no change to
// the scheduling engine is required (plan 17.4 NFR maintainability, FR-1).
type ScheduledJob struct {
	// Name is the stable identifier used in history, locks, the admin API and
	// the jobs.queue unique_key. It must be unique.
	Name string
	// Spec is the cron expression (UTC) controlling when the job fires.
	Spec string
	// JobType is the jobs.queue job_type enqueued when the schedule is due.
	JobType string
	// Description is a human-readable summary for the admin UI.
	Description string
	// DefaultEnabled is the built-in enabled state; an admin override in
	// jobs.schedule_overrides takes precedence (plan 17.4 FR-6).
	DefaultEnabled bool

	schedule Schedule
}

// Schedule returns the compiled cron schedule for this job.
func (j ScheduledJob) Schedule() Schedule { return j.schedule }

// BuiltinJobs returns the scheduled jobs shipped with the platform (plan 17.4
// FR-4). Cron expressions are in UTC. The list is compiled once and validated by
// MustParse so a malformed expression panics at startup rather than silently
// never firing.
func BuiltinJobs() []ScheduledJob {
	jobs := []ScheduledJob{
		{
			Name:           "late_submission_sweep",
			Spec:           "5 0 * * *", // daily 00:05 UTC
			JobType:        JobTypeLateSubmissionSweep,
			Description:    "Mark overdue assignment submissions as late.",
			DefaultEnabled: true,
		},
		{
			Name:           "expired_token_cleanup",
			Spec:           "0 * * * *", // hourly on the hour
			JobType:        JobTypeExpiredTokenCleanup,
			Description:    "Delete expired and revoked API tokens.",
			DefaultEnabled: true,
		},
		{
			Name:           "request_log_retention",
			Spec:           "0 3 * * *", // daily 03:00 UTC
			JobType:        JobTypeRequestLogRetention,
			Description:    "Delete API request-log rows older than 90 days (GDPR retention).",
			DefaultEnabled: true,
		},
		{
			Name:           "due_date_reminder",
			Spec:           "0 8 * * *", // daily 08:00 UTC
			JobType:        JobTypeDueDateReminder,
			Description:    "Enqueue reminder notifications for assignments due soon.",
			DefaultEnabled: true,
		},
		{
			Name:           "inactive_integration_alert",
			Spec:           "0 6 * * *", // daily 06:00 UTC
			JobType:        JobTypeInactiveIntegration,
			Description:    "Flag webhook subscriptions with no delivery activity in over 12 hours.",
			DefaultEnabled: true,
		},
		{
			Name:           "tutor_session_retention",
			Spec:           "30 4 * * *", // daily 04:30 UTC
			JobType:        JobTypeTutorSessionRetention,
			Description:    "Purge tutor sessions older than each org's retention policy (plan 19.1).",
			DefaultEnabled: true,
		},
	}
	for i := range jobs {
		jobs[i].schedule = MustParse(jobs[i].Spec)
	}
	return jobs
}
