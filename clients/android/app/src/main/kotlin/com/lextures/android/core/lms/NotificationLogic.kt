package com.lextures.android.core.lms

enum class NotificationCategory {
    Grades,
    Assignments,
    Discussions,
    Announcements,
    Messages,
    Reminders,
    Account,
    Courses,
    Other,
}

enum class NotificationFilter(val id: String, val labelKey: String) {
    All("all", "mobile.notifications.filter.all"),
    Unread("unread", "mobile.notifications.filter.unread"),
    Grades("grades", "mobile.notifications.category.grades"),
    Assignments("assignments", "mobile.notifications.category.assignments"),
    Discussions("discussions", "mobile.notifications.category.discussions"),
    Announcements("announcements", "mobile.notifications.category.announcements"),
    Messages("messages", "mobile.notifications.category.messages"),
    Reminders("reminders", "mobile.notifications.category.reminders"),
}

object NotificationLogic {
    fun category(eventType: String): NotificationCategory = when (eventType) {
        "grade_posted" -> NotificationCategory.Grades
        "assignment_created", "assignment_due_reminder", "submission_received",
        "incomplete_granted", "incomplete_reminder" -> NotificationCategory.Assignments
        "discussion_reply" -> NotificationCategory.Discussions
        "course_announcement", "meeting_reminder", "conference_confirmed",
        "conference_reminder", "coaching_tip_weekly" -> NotificationCategory.Announcements
        "inbox_message" -> NotificationCategory.Messages
        "study_reminder_daily", "study_reminder_streak_at_risk", "study_reminder_weekly_summary" ->
            NotificationCategory.Reminders
        "password_reset", "welcome_invite", "payment_failed", "ceu_awarded", "certificate_issued" ->
            NotificationCategory.Account
        "canvas_course_imported", "course_copy_imported", "course_copy_import_failed" ->
            NotificationCategory.Courses
        else -> NotificationCategory.Other
    }

    fun matchesFilter(notification: AppNotification, filter: NotificationFilter): Boolean = when (filter) {
        NotificationFilter.All -> true
        NotificationFilter.Unread -> !notification.isRead
        NotificationFilter.Grades -> category(notification.eventType) == NotificationCategory.Grades
        NotificationFilter.Assignments -> category(notification.eventType) == NotificationCategory.Assignments
        NotificationFilter.Discussions -> category(notification.eventType) == NotificationCategory.Discussions
        NotificationFilter.Announcements -> category(notification.eventType) == NotificationCategory.Announcements
        NotificationFilter.Messages -> category(notification.eventType) == NotificationCategory.Messages
        NotificationFilter.Reminders -> category(notification.eventType) == NotificationCategory.Reminders
    }

    fun filter(notifications: List<AppNotification>, filter: NotificationFilter): List<AppNotification> =
        notifications.filter { matchesFilter(it, filter) }

    fun eventLabelKey(eventType: String): String = "mobile.notifications.event.$eventType"

    fun groupedPreferences(preferences: List<NotificationPreference>): List<Pair<NotificationCategory, List<NotificationPreference>>> {
        val grouped = preferences.groupBy { category(it.eventType) }
        return NotificationCategory.entries.mapNotNull { category ->
            val rows = grouped[category]?.sortedBy { it.eventType }.orEmpty()
            if (rows.isEmpty()) null else category to rows
        }
    }

    fun isPushEnabled(eventType: String, preferences: List<NotificationPreference>): Boolean =
        preferences.firstOrNull { it.eventType == eventType }?.pushEnabled ?: true

    fun isEmailEnabled(eventType: String, preferences: List<NotificationPreference>): Boolean =
        preferences.firstOrNull { it.eventType == eventType }?.emailEnabled ?: true
}
