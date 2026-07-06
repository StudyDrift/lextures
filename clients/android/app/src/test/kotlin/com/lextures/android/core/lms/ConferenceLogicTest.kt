package com.lextures.android.core.lms

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Test
import java.time.Instant

class ConferenceLogicTest {
    private val context: Context = ApplicationProvider.getApplicationContext()

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

    @Test
    fun calendarEventsFromBookings() {
        val future = Instant.now().plusSeconds(3600).toString()
        val end = Instant.now().plusSeconds(5400).toString()
        val booking = ParentConferenceBooking(
            slot = ConferenceSlot(
                id = "s1",
                availabilityId = "a",
                startAt = future,
                endAt = end,
                status = "booked",
                bookedForChild = "child1",
            ),
            teacher = ConferenceTeacher(teacherId = "t1", displayName = "Ms. Lee"),
            studentId = "child1",
            childName = "Alex",
            availability = ConferenceAvailability(id = "a", location = "Room 12", videoLink = "https://meet.example.com/abc"),
        )
        val events = ConferenceLogic.calendarEvents(context, listOf(booking))
        assertEquals(1, events.size)
        assertEquals(PlannerCalendarEventKind.Conference, events[0].kind)
        assertEquals("s1", events[0].conferenceSlotId)
        assertEquals("https://meet.example.com/abc", events[0].videoLink)
        assertEquals("Alex", events[0].courseTitle)
    }

    @Test
    fun locationLabelVirtual() {
        val availability = ConferenceAvailability(id = "a", location = "Room 5", videoLink = "https://meet.example.com")
        assertNotNull(ConferenceLogic.locationLabel(context, availability))
    }
}
