package com.lextures.android.core.lms

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import com.lextures.android.R
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class ParentLogicTest {
    private val context: Context = ApplicationProvider.getApplicationContext()

    @Test
    fun childLabelPrefersDisplayName() {
        val child = ParentChildSummary(
            studentUserId = "s1",
            displayName = "Alex",
            email = "alex@school.edu",
        )
        assertEquals("Alex", ParentLogic.childLabel(child))
    }

    @Test
    fun resolveSelectedChildId() {
        val children = listOf(
            ParentChildSummary(studentUserId = "a", email = "a@x"),
            ParentChildSummary(studentUserId = "b", email = "b@x"),
        )
        assertEquals("b", ParentLogic.resolveSelectedChildId(children, "b"))
        assertEquals("a", ParentLogic.resolveSelectedChildId(children, "missing"))
        assertNull(ParentLogic.resolveSelectedChildId(emptyList(), "a"))
    }

    @Test
    fun attendanceSummary() {
        val records = listOf(
            ParentAttendanceRecord(id = "1", date = "2026-01-01", category = "present"),
            ParentAttendanceRecord(id = "2", date = "2026-01-02", category = "absent"),
            ParentAttendanceRecord(id = "3", date = "2026-01-03", code = "T"),
        )
        val summary = ParentLogic.attendanceSummary(records)
        assertEquals(1, summary.present)
        assertEquals(1, summary.absent)
        assertEquals(1, summary.tardy)
    }

    @Test
    fun attendanceLabelUsesCodeLabel() {
        val record = ParentAttendanceRecord(id = "1", date = "2026-01-01", codeLabel = "Present")
        assertEquals("Present", ParentLogic.attendanceLabel(context, record))
    }
}
