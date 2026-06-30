package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class NotificationModelsTest {
    @Test
    fun categoryMapping() {
        assertEquals(NotificationCategory.Grades, NotificationLogic.category("grade_posted"))
        assertEquals(NotificationCategory.Discussions, NotificationLogic.category("discussion_reply"))
        assertEquals(NotificationCategory.Messages, NotificationLogic.category("inbox_message"))
        assertEquals(NotificationCategory.Other, NotificationLogic.category("unknown_event"))
    }

    @Test
    fun filterUnreadAndGrades() {
        val notifications = listOf(
            notification("1", "grade_posted", isRead = false),
            notification("2", "discussion_reply", isRead = true),
            notification("3", "grade_posted", isRead = true),
        )
        assertEquals(listOf("1"), NotificationLogic.filter(notifications, NotificationFilter.Unread).map { it.id })
        assertEquals(listOf("1", "3"), NotificationLogic.filter(notifications, NotificationFilter.Grades).map { it.id })
        assertEquals(listOf("2"), NotificationLogic.filter(notifications, NotificationFilter.Discussions).map { it.id })
    }

    @Test
    fun pushPreferenceGating() {
        val preferences = listOf(
            NotificationPreference(eventType = "discussion_reply", pushEnabled = false),
            NotificationPreference(eventType = "grade_posted", pushEnabled = true),
        )
        assertFalse(NotificationLogic.isPushEnabled("discussion_reply", preferences))
        assertTrue(NotificationLogic.isPushEnabled("grade_posted", preferences))
        assertTrue(NotificationLogic.isPushEnabled("missing", preferences))
    }

    @Test
    fun groupedPreferencesSortsWithinCategory() {
        val preferences = listOf(
            NotificationPreference(eventType = "discussion_reply"),
            NotificationPreference(eventType = "grade_posted"),
            NotificationPreference(eventType = "assignment_created"),
        )
        val grouped = NotificationLogic.groupedPreferences(preferences)
        assertEquals(
            listOf(NotificationCategory.Grades, NotificationCategory.Assignments, NotificationCategory.Discussions),
            grouped.map { it.first },
        )
        assertEquals(listOf("grade_posted"), grouped.first().second.map { it.eventType })
    }

    private fun notification(id: String, eventType: String, isRead: Boolean) = AppNotification(
        id = id,
        eventType = eventType,
        title = "Title",
        body = "Body",
        isRead = isRead,
        createdAt = "2026-06-30T12:00:00Z",
    )
}
