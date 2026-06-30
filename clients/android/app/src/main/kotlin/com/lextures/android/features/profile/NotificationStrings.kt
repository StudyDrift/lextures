package com.lextures.android.features.profile

import androidx.compose.runtime.Composable
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.NotificationCategory
import com.lextures.android.core.lms.NotificationFilter

@Composable
fun notificationsTitle(): String = L.text(R.string.mobile_notifications_title)

@Composable
fun notificationsStaleOfflineLabel(): String = L.text(R.string.mobile_notifications_stale_offline)

@Composable
fun notificationsPreferencesTitle(): String = L.text(R.string.mobile_notifications_preferences_title)

@Composable
fun notificationsMarkAllReadLabel(): String = L.text(R.string.mobile_notifications_markAllRead)

@Composable
fun notificationsMarkAllReadMutationLabel(): String = L.text(R.string.mobile_notifications_markAllReadLabel)

@Composable
fun notificationsMarkReadMutationLabel(): String = L.text(R.string.mobile_notifications_markReadLabel)

@Composable
fun notificationsEmptyUnreadTitle(): String = L.text(R.string.mobile_notifications_empty_unread)

@Composable
fun notificationsEmptyAllTitle(): String = L.text(R.string.mobile_notifications_empty_all)

@Composable
fun notificationsEmptyMessage(): String = L.text(R.string.mobile_notifications_empty_message)

@Composable
fun notificationsAccessibilityReadLabel(): String = L.text(R.string.mobile_notifications_accessibility_read)

@Composable
fun notificationsAccessibilityUnreadLabel(): String = L.text(R.string.mobile_notifications_accessibility_unread)

@Composable
fun notificationsPreferencesDescription(): String = L.text(R.string.mobile_notifications_preferences_description)

@Composable
fun notificationsPreferencesSavedMessage(): String = L.text(R.string.mobile_notifications_preferences_saved)

@Composable
fun notificationsPreferencesPushLabel(): String = L.text(R.string.mobile_notifications_preferences_push)

@Composable
fun notificationsPreferencesEmailLabel(): String = L.text(R.string.mobile_notifications_preferences_email)

@Composable
fun notificationsPreferencesSaveMutationLabel(): String = L.text(R.string.mobile_notifications_preferences_saveLabel)

@Composable
fun filterLabel(filter: NotificationFilter): String = when (filter) {
    NotificationFilter.All -> L.text(R.string.mobile_notifications_filter_all)
    NotificationFilter.Unread -> L.text(R.string.mobile_notifications_filter_unread)
    NotificationFilter.Grades -> L.text(R.string.mobile_notifications_category_grades)
    NotificationFilter.Assignments -> L.text(R.string.mobile_notifications_category_assignments)
    NotificationFilter.Discussions -> L.text(R.string.mobile_notifications_category_discussions)
    NotificationFilter.Announcements -> L.text(R.string.mobile_notifications_category_announcements)
    NotificationFilter.Messages -> L.text(R.string.mobile_notifications_category_messages)
    NotificationFilter.Reminders -> L.text(R.string.mobile_notifications_category_reminders)
}

@Composable
fun categoryLabel(category: NotificationCategory): String = when (category) {
    NotificationCategory.Grades -> L.text(R.string.mobile_notifications_category_grades)
    NotificationCategory.Assignments -> L.text(R.string.mobile_notifications_category_assignments)
    NotificationCategory.Discussions -> L.text(R.string.mobile_notifications_category_discussions)
    NotificationCategory.Announcements -> L.text(R.string.mobile_notifications_category_announcements)
    NotificationCategory.Messages -> L.text(R.string.mobile_notifications_category_messages)
    NotificationCategory.Reminders -> L.text(R.string.mobile_notifications_category_reminders)
    NotificationCategory.Account -> L.text(R.string.mobile_notifications_category_account)
    NotificationCategory.Courses -> L.text(R.string.mobile_notifications_category_courses)
    NotificationCategory.Other -> L.text(R.string.mobile_notifications_category_other)
}

@Composable
fun eventTypeLabel(eventType: String): String = when (eventType) {
    "assignment_created" -> L.text(R.string.mobile_notifications_event_assignment_created)
    "assignment_due_reminder" -> L.text(R.string.mobile_notifications_event_assignment_due_reminder)
    "canvas_course_imported" -> L.text(R.string.mobile_notifications_event_canvas_course_imported)
    "coaching_tip_weekly" -> L.text(R.string.mobile_notifications_event_coaching_tip_weekly)
    "conference_confirmed" -> L.text(R.string.mobile_notifications_event_conference_confirmed)
    "conference_reminder" -> L.text(R.string.mobile_notifications_event_conference_reminder)
    "course_announcement" -> L.text(R.string.mobile_notifications_event_course_announcement)
    "course_copy_import_failed" -> L.text(R.string.mobile_notifications_event_course_copy_import_failed)
    "course_copy_imported" -> L.text(R.string.mobile_notifications_event_course_copy_imported)
    "discussion_reply" -> L.text(R.string.mobile_notifications_event_discussion_reply)
    "grade_posted" -> L.text(R.string.mobile_notifications_event_grade_posted)
    "inbox_message" -> L.text(R.string.mobile_notifications_event_inbox_message)
    "meeting_reminder" -> L.text(R.string.mobile_notifications_event_meeting_reminder)
    "password_reset" -> L.text(R.string.mobile_notifications_event_password_reset)
    "study_reminder_daily" -> L.text(R.string.mobile_notifications_event_study_reminder_daily)
    "study_reminder_streak_at_risk" -> L.text(R.string.mobile_notifications_event_study_reminder_streak_at_risk)
    "study_reminder_weekly_summary" -> L.text(R.string.mobile_notifications_event_study_reminder_weekly_summary)
    "submission_received" -> L.text(R.string.mobile_notifications_event_submission_received)
    "welcome_invite" -> L.text(R.string.mobile_notifications_event_welcome_invite)
    else -> eventType.replace('_', ' ')
}
