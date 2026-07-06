package com.lextures.android.core.lms

import android.content.Context
import com.lextures.android.R
import com.lextures.android.core.config.AppConfiguration
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

/** A parent's booked conference slot with teacher and child context (M10.2). */
data class ParentConferenceBooking(
    val slot: ConferenceSlot,
    val teacher: ConferenceTeacher,
    val studentId: String,
    val childName: String,
    val availability: ConferenceAvailability? = null,
)

/** Conference slot formatting, filtering, and planner mapping (M10.2). */
object ConferenceLogic {
    fun isMyBooking(slot: ConferenceSlot, parentId: String?, studentId: String): Boolean =
        slot.status == "booked" &&
            slot.bookedForChild == studentId &&
            (parentId == null || slot.bookedByParent == null || slot.bookedByParent == parentId)

    fun upcomingAvailableSlots(slots: List<ConferenceSlot>, now: Instant = Instant.now()): List<ConferenceSlot> =
        slots.filter { it.status == "open" }
            .filter { slot ->
                val start = LmsDates.parse(slot.startAt) ?: return@filter false
                !start.isBefore(now)
            }
            .sortedBy { it.startAt }

    fun myBookedSlots(
        slots: List<ConferenceSlot>,
        parentId: String?,
        studentId: String,
        now: Instant = Instant.now(),
    ): List<ConferenceSlot> =
        slots.filter { isMyBooking(it, parentId, studentId) }
            .filter { slot ->
                val start = LmsDates.parse(slot.startAt) ?: return@filter false
                !start.isBefore(now)
            }
            .sortedBy { it.startAt }

    fun formatSlotTime(slot: ConferenceSlot): String {
        val start = LmsDates.parse(slot.startAt) ?: return slot.startAt
        val end = LmsDates.parse(slot.endAt)
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

    fun locationLabel(context: Context, availability: ConferenceAvailability?): String? {
        if (availability == null) return null
        val location = availability.location?.trim().orEmpty()
        val videoLink = availability.videoLink?.trim().orEmpty()
        val isVirtual = videoLink.isNotEmpty()
        if (location.isEmpty()) {
            return if (isVirtual) context.getString(R.string.mobile_parent_conferences_virtual) else null
        }
        return if (isVirtual) {
            "${context.getString(R.string.mobile_parent_conferences_virtual)} · $location"
        } else {
            location
        }
    }

    fun isJoinWindow(slot: ConferenceSlot, availability: ConferenceAvailability?, now: Instant = Instant.now()): Boolean {
        val videoLink = availability?.videoLink?.trim().orEmpty()
        if (videoLink.isEmpty()) return false
        val start = LmsDates.parse(slot.startAt) ?: return false
        val end = LmsDates.parse(slot.endAt) ?: start.plusSeconds(15 * 60)
        val openFrom = start.minusSeconds(10 * 60)
        return !now.isBefore(openFrom) && !now.isAfter(end)
    }

    fun icalUrl(slotId: String): String =
        AppConfiguration.apiUrl("/api/v1/conference-slots/${slotId}/ical").toString()

    fun todayDateString(): String {
        val formatter = DateTimeFormatter.ISO_LOCAL_DATE.withZone(ZoneId.systemDefault())
        return formatter.format(Instant.now())
    }

    fun scanDates(dayCount: Int = 21, from: Instant = Instant.now()): List<String> {
        val formatter = DateTimeFormatter.ISO_LOCAL_DATE.withZone(ZoneId.systemDefault())
        val zone = ZoneId.systemDefault()
        return (0 until dayCount).mapNotNull { offset ->
            formatter.format(from.atZone(zone).toLocalDate().plusDays(offset.toLong()))
        }
    }

    suspend fun loadParentBookings(
        children: List<Pair<String, String>>,
        accessToken: String,
        dates: List<String>? = null,
    ): List<ParentConferenceBooking> {
        val scan = dates ?: scanDates()
        val results = mutableListOf<ParentConferenceBooking>()
        for ((studentId, childName) in children) {
            val teachers = runCatching {
                LmsApi.fetchParentConferenceTeachers(studentId, accessToken)
            }.getOrDefault(emptyList())
            for (teacher in teachers) {
                for (date in scan) {
                    val response = runCatching {
                        LmsApi.fetchConferenceSlots(teacher.teacherId, date, accessToken)
                    }.getOrNull() ?: continue
                    val booked = myBookedSlots(response.slots, null, studentId)
                    for (slot in booked) {
                        results += ParentConferenceBooking(
                            slot = slot,
                            teacher = teacher,
                            studentId = studentId,
                            childName = childName,
                            availability = response.availability,
                        )
                    }
                }
            }
        }
        return results.sortedBy { it.slot.startAt }
    }

    fun calendarEvents(context: Context, bookings: List<ParentConferenceBooking>): List<PlannerCalendarEvent> =
        bookings.mapNotNull { booking ->
            val start = LmsDates.parse(booking.slot.startAt) ?: return@mapNotNull null
            val end = LmsDates.parse(booking.slot.endAt)
            val teacherName = ParentLogic.teacherLabel(context, booking.teacher)
            PlannerCalendarEvent(
                id = "conference:${booking.slot.id}",
                title = context.getString(R.string.mobile_parent_conferences_calendarTitle, teacherName),
                courseTitle = booking.childName,
                startsAt = start,
                endsAt = end,
                kind = PlannerCalendarEventKind.Conference,
                conferenceSlotId = booking.slot.id,
                videoLink = booking.availability?.videoLink,
            )
        }
}
