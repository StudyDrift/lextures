package com.lextures.android.core.lms

import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

/** Conference slot formatting and filtering (M10.2). */
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

    fun todayDateString(): String {
        val formatter = DateTimeFormatter.ISO_LOCAL_DATE.withZone(ZoneId.systemDefault())
        return formatter.format(Instant.now())
    }
}
