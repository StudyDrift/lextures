package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TakeAttendanceLogicTest {
    private fun record(id: String, name: String, status: String) =
        AttendanceRecord(studentUserId = id, displayName = name, status = status)

    @Test
    fun markAllPresent() {
        val records = listOf(
            record("a", "Alex", "not_recorded"),
            record("b", "Blair", "absent"),
        )
        val draft = TakeAttendanceLogic.markAllPresent(records)
        assertEquals("present", draft["a"])
        assertEquals("present", draft["b"])
    }

    @Test
    fun summaryCounts() {
        val records = listOf(
            record("a", "Alex", "present"),
            record("b", "Blair", "absent"),
            record("c", "Casey", "tardy"),
            record("d", "Dana", "excused"),
            record("e", "Eden", "not_recorded"),
        )
        val counts = TakeAttendanceLogic.summaryCounts(records, mapOf("e" to "present"))
        assertEquals(2, counts.present)
        assertEquals(1, counts.absent)
        assertEquals(1, counts.tardy)
        assertEquals(1, counts.excused)
        assertEquals(0, counts.notRecorded)
    }

    @Test
    fun findTodaysOpenRollCallSession() {
        val today = TakeAttendanceLogic.todayDateString()
        val sessions = listOf(
            AttendanceSession(id = "1", title = "Old", collectionMethod = "roll_call", sessionDate = "2020-01-01", status = "open"),
            AttendanceSession(id = "2", title = "Today", collectionMethod = "roll_call", sessionDate = today, status = "open"),
            AttendanceSession(id = "3", title = "Self", collectionMethod = "self_report", sessionDate = today, status = "open"),
        )
        assertEquals("2", TakeAttendanceLogic.findTodaysOpenRollCallSession(sessions)?.id)
    }

    @Test
    fun shouldTakeSession() {
        val rollCall = AttendanceSession(id = "1", collectionMethod = "roll_call", status = "open")
        val selfReport = AttendanceSession(id = "2", collectionMethod = "self_report", status = "open")
        assertTrue(TakeAttendanceLogic.shouldTakeSession(rollCall, isStaff = true))
        assertFalse(TakeAttendanceLogic.shouldTakeSession(rollCall, isStaff = false))
        assertFalse(TakeAttendanceLogic.shouldTakeSession(selfReport, isStaff = true))
    }

    @Test
    fun recordsPayload() {
        val records = listOf(record("a", "Alex", "not_recorded"))
        val payload = TakeAttendanceLogic.recordsPayload(records, mapOf("a" to "absent"))
        assertEquals(1, payload.size)
        assertEquals("a", payload[0].studentUserId)
        assertEquals("absent", payload[0].status)
        assertEquals("instructor", payload[0].source)
    }
}