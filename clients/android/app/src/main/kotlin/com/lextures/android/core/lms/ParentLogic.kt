package com.lextures.android.core.lms

import android.content.Context
import com.lextures.android.R

/** Parent portal display helpers and summaries (M10.1). */
object ParentLogic {
    fun childLabel(child: ParentChildSummary): String {
        val name = child.displayName?.trim().orEmpty()
        if (name.isNotEmpty()) return name
        return child.email
    }

    fun teacherLabel(context: Context, teacher: ConferenceTeacher): String {
        val name = teacher.displayName?.trim().orEmpty()
        if (name.isNotEmpty()) return name
        return context.getString(R.string.mobile_parent_conferences_teacherFallback)
    }

    fun resolveSelectedChildId(
        children: List<ParentChildSummary>,
        storedId: String?,
    ): String? {
        if (children.isEmpty()) return null
        if (storedId != null && children.any { it.studentUserId == storedId }) return storedId
        return children.firstOrNull()?.studentUserId
    }

    data class AttendanceSummary(val present: Int, val absent: Int, val tardy: Int)

    fun attendanceSummary(records: List<ParentAttendanceRecord>): AttendanceSummary {
        var present = 0
        var absent = 0
        var tardy = 0
        for (record in records) {
            val category = (record.category ?: record.code ?: "").lowercase()
            when {
                category.contains("absent") || category == "a" -> absent++
                category.contains("tardy") || category == "t" -> tardy++
                else -> present++
            }
        }
        return AttendanceSummary(present, absent, tardy)
    }

    fun recentAttendance(records: List<ParentAttendanceRecord>, limit: Int = 5): List<ParentAttendanceRecord> =
        records.sortedWith(compareByDescending<ParentAttendanceRecord> { it.date }.thenByDescending { it.recordedAt })
            .take(limit)

    data class GradePreview(val course: ParentCourseGradesRow, val itemId: String, val score: String)

    fun recentGrades(courses: List<ParentCourseGradesRow>, limit: Int = 6): List<GradePreview> {
        val rows = mutableListOf<GradePreview>()
        for (course in courses) {
            for ((itemId, score) in course.grades) {
                rows += GradePreview(course, itemId, score)
            }
        }
        return rows.take(limit)
    }

    fun upcomingAssignments(assignments: List<ParentAssignmentRow>, limit: Int = 8): List<ParentAssignmentRow> =
        assignments.sortedBy { it.dueAt.orEmpty() }.take(limit)

    fun weeklyItemsForChild(items: List<ParentWeeklySummaryItem>, childName: String): List<ParentWeeklySummaryItem> =
        items.filter { it.childName == childName }

    fun attendanceLabel(context: Context, record: ParentAttendanceRecord): String {
        val label = record.codeLabel?.trim().orEmpty()
        if (label.isNotEmpty()) return label
        val code = record.code?.trim().orEmpty()
        if (code.isNotEmpty()) return code
        return record.category ?: context.getString(R.string.mobile_parent_attendance_unknown)
    }
}
