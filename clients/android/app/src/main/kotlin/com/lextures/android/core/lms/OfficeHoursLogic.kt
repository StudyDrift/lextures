package com.lextures.android.core.lms

import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

/** Office-hours slot formatting, filtering, and planner calendar mapping (M7.3). */
object OfficeHoursLogic {
    const val CALENDAR_EVENT_TITLE = "Office Hours"
    const val VIRTUAL_LABEL = "Virtual"

    fun isMyBooking(slot: AppointmentSlot): Boolean =
        slot.status == "booked" && !slot.studentId.isNullOrBlank()

    fun upcomingAvailableSlots(slots: List<AppointmentSlot>, now: Instant = Instant.now()): List<AppointmentSlot> =
        slots
            .filter { it.status == "available" }
            .filter { slot ->
                val start = LmsDates.parse(slot.slotStart) ?: return@filter false
                !start.isBefore(now)
            }
            .sortedBy { it.slotStart }

    fun myBookedSlots(slots: List<AppointmentSlot>, now: Instant = Instant.now()): List<AppointmentSlot> =
        slots
            .filter(::isMyBooking)
            .filter { slot ->
                val start = LmsDates.parse(slot.slotStart) ?: return@filter false
                !start.isBefore(now)
            }
            .sortedBy { it.slotStart }

    fun formatSlotTime(slot: AppointmentSlot): String {
        val start = LmsDates.parse(slot.slotStart) ?: return slot.slotStart
        val end = LmsDates.parse(slot.slotEnd)
        val zone = ZoneId.systemDefault()
        val startText = DateTimeFormatter.ofLocalizedDateTime(FormatStyle.MEDIUM, FormatStyle.SHORT)
            .withZone(zone)
            .format(start)
        if (end == null) return startText
        val endText = DateTimeFormatter.ofLocalizedTime(FormatStyle.SHORT)
            .withZone(zone)
            .format(end)
        return "$startText – $endText"
    }

    fun locationLabel(window: AvailabilityWindow?): String? {
        if (window == null) return null
        val location = window.location?.trim().orEmpty()
        if (location.isEmpty()) {
            return if (window.isVirtual) VIRTUAL_LABEL else null
        }
        return if (window.isVirtual) {
            "$VIRTUAL_LABEL · $location"
        } else {
            location
        }
    }

    fun windowMap(windows: List<AvailabilityWindow>): Map<String, AvailabilityWindow> =
        windows.associateBy { it.id }

    fun calendarEvents(
        courseCode: String,
        courseTitle: String,
        slots: List<AppointmentSlot>,
        windows: List<AvailabilityWindow>,
    ): List<PlannerCalendarEvent> =
        myBookedSlots(slots).mapNotNull { slot ->
            val start = LmsDates.parse(slot.slotStart) ?: return@mapNotNull null
            val end = LmsDates.parse(slot.slotEnd)
            PlannerCalendarEvent(
                id = "office-hours:${slot.id}",
                title = CALENDAR_EVENT_TITLE,
                courseCode = courseCode,
                courseTitle = courseTitle,
                startsAt = start,
                endsAt = end,
                allDay = false,
                kind = PlannerCalendarEventKind.OfficeHours,
                officeHoursSlotId = slot.id,
                meetingId = slot.meetingId,
            )
        }

    fun collectCalendarEvents(
        studentCourses: List<CourseSummary>,
        availabilityByCourseCode: Map<String, OfficeHoursAvailability>,
    ): List<PlannerCalendarEvent> =
        studentCourses.flatMap { course ->
            if (!course.isOfficeHoursEnabled) return@flatMap emptyList()
            val availability = availabilityByCourseCode[course.courseCode] ?: return@flatMap emptyList()
            calendarEvents(course.courseCode, course.displayTitle, availability.slots, availability.windows)
        }
}

val CourseSummary.isOfficeHoursEnabled: Boolean
    get() = officeHoursEnabled == true
