package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.Instant

class ConferenceLogicTest {
    @Test
    fun upcomingAvailableSlotsFiltersBookedAndPast() {
        val future = Instant.now().plusSeconds(3600).toString()
        val past = Instant.now().minusSeconds(3600).toString()
        val slots = listOf(
            ConferenceSlot(id = "1", availabilityId = "a", startAt = future, endAt = future, status = "open"),
            ConferenceSlot(id = "2", availabilityId = "a", startAt = past, endAt = past, status = "open"),
            ConferenceSlot(id = "3", availabilityId = "a", startAt = future, endAt = future, status = "booked"),
        )
        assertEquals(listOf("1"), ConferenceLogic.upcomingAvailableSlots(slots).map { it.id })
    }

    @Test
    fun myBookedSlotsMatchesChild() {
        val future = Instant.now().plusSeconds(3600).toString()
        val slots = listOf(
            ConferenceSlot(
                id = "1",
                availabilityId = "a",
                startAt = future,
                endAt = future,
                status = "booked",
                bookedByParent = "p1",
                bookedForChild = "child1",
            ),
            ConferenceSlot(
                id = "2",
                availabilityId = "a",
                startAt = future,
                endAt = future,
                status = "booked",
                bookedByParent = "p1",
                bookedForChild = "child2",
            ),
        )
        assertEquals(listOf("1"), ConferenceLogic.myBookedSlots(slots, "p1", "child1").map { it.id })
    }
}
