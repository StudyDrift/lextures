package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant

class BehaviorLogicTest {
    @Test
    fun studentRosterFiltersLearners() {
        val enrollments = listOf(
            CourseEnrollment(id = "1", userId = "s1", displayName = "Alex", role = "student"),
            CourseEnrollment(id = "2", userId = "t1", displayName = "Teacher", role = "teacher"),
            CourseEnrollment(id = "3", userId = "s2", displayName = null, role = "learner"),
        )
        assertEquals(listOf("s1", "s2"), BehaviorLogic.studentRoster(enrollments).map { it.userId })
    }

    @Test
    fun positiveAndNegativeCategories() {
        val categories = listOf(
            BehaviorCategory(id = "1", orgId = "o", name = "Respect", type = "positive"),
            BehaviorCategory(id = "2", orgId = "o", name = "Tardy", type = "negative"),
            BehaviorCategory(id = "3", orgId = "o", name = "Old", type = "positive", active = false),
        )
        assertEquals(listOf("1"), BehaviorLogic.positiveCategories(categories).map { it.id })
        assertEquals(listOf("2"), BehaviorLogic.negativeCategories(categories).map { it.id })
    }

    @Test
    fun awardPayloadBuildsOnePerStudent() {
        val payload = BehaviorLogic.awardPayload(setOf("a", "b"), "cat", " nice ")
        assertEquals(2, payload.size)
        assertEquals(1, payload[0].points)
        assertEquals("nice", payload[0].note)
    }

    @Test
    fun hallPassCountdownUsesApprovedAt() {
        val approved = "2026-07-06T12:00:00.000Z"
        val pass = HallPass(
            id = "p1",
            sectionId = "s1",
            studentId = "u1",
            destination = "bathroom",
            status = "approved",
            estimatedMins = 5,
            requestedAt = approved,
            approvedAt = approved,
        )
        val countdown = BehaviorLogic.hallPassCountdown(
            pass,
            now = Instant.parse("2026-07-06T12:02:00.000Z"),
        )
        assertNotNull(countdown)
        assertEquals(180, countdown?.remainingSeconds)
    }

    @Test
    fun isActiveHallPass() {
        val requested = HallPass(id = "1", sectionId = "s", destination = "office", status = "requested", requestedAt = "")
        val returned = HallPass(id = "2", sectionId = "s", destination = "office", status = "returned", requestedAt = "")
        assertTrue(BehaviorLogic.isActiveHallPass(requested))
        assertFalse(BehaviorLogic.isActiveHallPass(returned))
    }
}
