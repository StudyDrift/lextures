package notificationevents

// Event types for notification preferences and email jobs (plan 6.2).
const (
	GradePosted            = "grade_posted"
	AssignmentCreated      = "assignment_created"
	DiscussionReply        = "discussion_reply"
	CourseAnnouncement     = "course_announcement"
	SubmissionReceived     = "submission_received"
	AssignmentDueReminder  = "assignment_due_reminder"
	PasswordReset          = "password_reset"
	WelcomeInvite          = "welcome_invite"
	MeetingReminder        = "meeting_reminder"
	ConferenceConfirmed      = "conference_confirmed"
	ConferenceReminder       = "conference_reminder"
	CoachingTipWeekly      = "coaching_tip_weekly"
	CanvasCourseImported   = "canvas_course_imported"
	CourseCopyImported     = "course_copy_imported"
	CourseCopyImportFailed = "course_copy_import_failed"
	InboxMessage           = "inbox_message"
	IncompleteGranted      = "incomplete_granted"
	IncompleteReminder     = "incomplete_reminder"
	CEUAwarded             = "ceu_awarded"
	CertificateIssued      = "certificate_issued"
	PaymentFailed          = "payment_failed"
	StudyReminderDaily         = "study_reminder_daily"
	StudyReminderStreakAtRisk  = "study_reminder_streak_at_risk"
	StudyReminderWeeklySummary = "study_reminder_weekly_summary"
	SeatUtilizationAlert       = "seat_utilization_alert"
	IntroCourseCompleted       = "intro_course_completed"
	TranscriptOrderSubmitted   = "transcript_order_submitted"
	TranscriptOrderOnHold      = "transcript_order_on_hold"
	TranscriptOrderConsent     = "transcript_order_consent_needed"
	TranscriptOrderPayment     = "transcript_order_payment_needed"
	TranscriptOrderApproved    = "transcript_order_approved"
	TranscriptOrderRejected    = "transcript_order_rejected"
	TranscriptOrderSent        = "transcript_order_sent"
	TranscriptOrderDelivered    = "transcript_order_delivered"
	TranscriptOrderOpened      = "transcript_order_opened"
	TranscriptOrderFailed      = "transcript_order_failed"
	TranscriptOrderCanceled    = "transcript_order_canceled"
	TranscriptOrderException   = "transcript_order_exception"
)

// All is the canonical list for defaults and UI.
var All = []string{
	GradePosted,
	AssignmentCreated,
	DiscussionReply,
	CourseAnnouncement,
	SubmissionReceived,
	AssignmentDueReminder,
	PasswordReset,
	WelcomeInvite,
	MeetingReminder,
	ConferenceConfirmed,
	ConferenceReminder,
	CoachingTipWeekly,
	CanvasCourseImported,
	CourseCopyImported,
	CourseCopyImportFailed,
	InboxMessage,
	IncompleteGranted,
	IncompleteReminder,
	CEUAwarded,
	CertificateIssued,
	PaymentFailed,
	StudyReminderDaily,
	StudyReminderStreakAtRisk,
	StudyReminderWeeklySummary,
	SeatUtilizationAlert,
	IntroCourseCompleted,
	TranscriptOrderSubmitted,
	TranscriptOrderOnHold,
	TranscriptOrderConsent,
	TranscriptOrderPayment,
	TranscriptOrderApproved,
	TranscriptOrderRejected,
	TranscriptOrderSent,
	TranscriptOrderDelivered,
	TranscriptOrderOpened,
	TranscriptOrderFailed,
	TranscriptOrderCanceled,
	TranscriptOrderException,
}
