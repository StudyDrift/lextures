package notifications

import "github.com/lextures/lextures/server/internal/notificationevents"

const (
	EventGradePosted           = notificationevents.GradePosted
	EventAssignmentCreated     = notificationevents.AssignmentCreated
	EventDiscussionReply       = notificationevents.DiscussionReply
	EventCourseAnnouncement    = notificationevents.CourseAnnouncement
	EventSubmissionReceived    = notificationevents.SubmissionReceived
	EventAssignmentDueReminder = notificationevents.AssignmentDueReminder
	EventPasswordReset         = notificationevents.PasswordReset
	EventWelcomeInvite         = notificationevents.WelcomeInvite
	EventMeetingReminder       = notificationevents.MeetingReminder
	EventConferenceConfirmed   = notificationevents.ConferenceConfirmed
	EventConferenceReminder    = notificationevents.ConferenceReminder
	EventCoachingTipWeekly     = notificationevents.CoachingTipWeekly
	EventCanvasCourseImported  = notificationevents.CanvasCourseImported
	EventCourseCopyImported    = notificationevents.CourseCopyImported
	EventCourseCopyImportFailed = notificationevents.CourseCopyImportFailed
	EventInboxMessage          = notificationevents.InboxMessage
	EventIncompleteGranted     = notificationevents.IncompleteGranted
	EventIncompleteReminder    = notificationevents.IncompleteReminder
	EventCEUAwarded            = notificationevents.CEUAwarded
	EventCertificateIssued     = notificationevents.CertificateIssued
	EventStudyReminderDaily         = notificationevents.StudyReminderDaily
	EventStudyReminderStreakAtRisk  = notificationevents.StudyReminderStreakAtRisk
	EventStudyReminderWeeklySummary = notificationevents.StudyReminderWeeklySummary
)

// AllEventTypes re-exports the canonical event list for callers outside this package.
var AllEventTypes = notificationevents.All
