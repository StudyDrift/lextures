package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.Instant

class OfficeHoursLogicTest {
    @Test
    fun upcomingAvailableSlotsFiltersPastAndBooked() {
        val now = Instant.parse("2023-01-01T12:00:00Z")
        val slots = listOf(
            slot("past", "2022-01-01T15:00:00Z", "available"),
            slot("open", "2099-06-01T15:00:00Z", "available"),
            slot("taken", "2099-06-02T15:00:00Z", "booked"),
        )
        assertEquals(listOf("open"), OfficeHoursLogic.upcomingAvailableSlots(slots, now).map { it.id })
    }

    @Test
    fun myBookedSlotsRequiresStudentId() {
        val now = Instant.parse("2023-01-01T12:00:00Z")
        val slots = listOf(
            slot("mine", "2099-06-01T15:00:00Z", "booked", studentId = "u1"),
            slot("other", "2099-06-02T15:00:00Z", "booked"),
        )
        assertEquals(listOf("mine"), OfficeHoursLogic.myBookedSlots(slots, now).map { it.id })
    }

    @Test
    fun calendarEventsFromBookings() {
        val course = CourseSummary(
            id = "1",
            courseCode = "BIO101",
            title = "Biology",
            officeHoursEnabled = true,
            viewerEnrollmentRoles = listOf("student"),
        )
        val slot = slot("s1", "2099-06-01T15:00:00Z", "booked", studentId = "u1")
        val events = OfficeHoursLogic.collectCalendarEvents(
            studentCourses = listOf(course),
            availabilityByCourseCode = mapOf(
                course.courseCode to OfficeHoursAvailability(
                    windows = listOf(
                        AvailabilityWindow(
                            id = "w1",
                            instructorId = "i1",
                            startTime = "15:00",
                            endTime = "16:00",
                            status = "active",
                        ),
                    ),
                    slots = listOf(slot),
                ),
            ),
        )
        assertEquals(1, events.size)
        assertEquals(PlannerCalendarEventKind.OfficeHours, events[0].kind)
        assertEquals("s1", events[0].officeHoursSlotId)
    }

    private fun slot(
        id: String,
        start: String,
        status: String,
        studentId: String? = null,
    ) = AppointmentSlot(
        id = id,
        windowId = "w1",
        slotStart = start,
        slotEnd = "2099-06-01T15:15:00Z",
        studentId = studentId,
        status = status,
    )
}
