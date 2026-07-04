package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant

class LiveMeetingsLogicTest {
    @Test
    fun groupMeetingsPartitionsByStatus() {
        val meetings = listOf(
            meeting("1", "live"),
            meeting("2", "scheduled"),
            meeting("3", "ended"),
        )
        val grouped = LiveMeetingsLogic.groupMeetings(meetings)
        assertEquals(listOf("1"), grouped.live.map { it.id })
        assertEquals(listOf("2"), grouped.upcoming.map { it.id })
        assertEquals(listOf("3"), grouped.past.map { it.id })
    }

    @Test
    fun isLiveOrSoonWithinThirtyMinutes() {
        val now = Instant.parse("2024-01-01T15:00:00Z")
        val soon = meeting("soon", "scheduled", start = "2024-01-01T15:10:00Z")
        val later = meeting("later", "scheduled", start = "2024-01-01T17:00:00Z")
        assertTrue(LiveMeetingsLogic.isLiveOrSoon(soon, now))
        assertFalse(LiveMeetingsLogic.isLiveOrSoon(later, now))
    }

    @Test
    fun calendarEventsFromScheduledMeetings() {
        val course = CourseSummary(
            id = "1",
            courseCode = "BIO101",
            title = "Biology",
            liveSessionsEnabled = true,
            viewerEnrollmentRoles = listOf("student"),
        )
        val meeting = meeting("m1", "scheduled")
        val events = LiveMeetingsLogic.collectCalendarEvents(
            studentCourses = listOf(course),
            meetingsByCourseCode = mapOf(course.courseCode to listOf(meeting)),
        )
        assertEquals(1, events.size)
        assertEquals(PlannerCalendarEventKind.LiveMeeting, events[0].kind)
        assertEquals("m1", events[0].meetingId)
    }

    private fun meeting(
        id: String,
        status: String,
        start: String = "2099-06-01T15:00:00Z",
        end: String = "2099-06-01T16:00:00Z",
    ) = VirtualMeeting(
        id = id,
        courseId = "c1",
        provider = "jitsi",
        title = "Class",
        scheduledStart = start,
        scheduledEnd = end,
        joinUrl = "https://example.com/join",
        status = status,
        createdBy = "u1",
        createdAt = "2099-01-01T00:00:00Z",
    )
}