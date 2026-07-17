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
	EventSeatUtilizationAlert       = notificationevents.SeatUtilizationAlert
	EventIntroCourseCompleted       = notificationevents.IntroCourseCompleted
	EventTranscriptOrderSubmitted   = notificationevents.TranscriptOrderSubmitted
	EventTranscriptOrderOnHold      = notificationevents.TranscriptOrderOnHold
	EventTranscriptOrderConsent     = notificationevents.TranscriptOrderConsent
	EventTranscriptOrderPayment     = notificationevents.TranscriptOrderPayment
	EventTranscriptOrderApproved    = notificationevents.TranscriptOrderApproved
	EventTranscriptOrderRejected    = notificationevents.TranscriptOrderRejected
	EventTranscriptOrderSent        = notificationevents.TranscriptOrderSent
	EventTranscriptOrderDelivered    = notificationevents.TranscriptOrderDelivered
	EventTranscriptOrderOpened      = notificationevents.TranscriptOrderOpened
	EventTranscriptOrderFailed      = notificationevents.TranscriptOrderFailed
	EventTranscriptOrderCanceled    = notificationevents.TranscriptOrderCanceled
	EventTranscriptOrderException   = notificationevents.TranscriptOrderException
)

// AllEventTypes re-exports the canonical event list for callers outside this package.
var AllEventTypes = notificationevents.All
