package com.lextures.android.core.lms

enum class AttendanceMarkStatus(val raw: String) {
    Present("present"),
    Absent("absent"),
    Tardy("tardy"),
    Excused("excused"),
    ;

    companion object {
        val markable: List<AttendanceMarkStatus> = entries
    }
}

data class AttendanceSummaryCounts(
    val present: Int,
    val absent: Int,
    val tardy: Int,
    val excused: Int,
    val notRecorded: Int,
)

object TakeAttendanceLogic {
    fun todayDateString(): String =
        java.time.LocalDate.now(java.time.ZoneOffset.UTC).toString()

    fun studentLabel(record: AttendanceRecord): String =
        record.displayName?.trim()?.takeIf { it.isNotEmpty() } ?: "Student"

    fun buildDraft(records: List<AttendanceRecord>): Map<String, String> =
        records.associate { it.studentUserId to it.status }

    fun markAllPresent(records: List<AttendanceRecord>): Map<String, String> =
        records.associate { it.studentUserId to AttendanceMarkStatus.Present.raw }

    fun summaryCounts(records: List<AttendanceRecord>, draft: Map<String, String>): AttendanceSummaryCounts {
        var present = 0
        var absent = 0
        var tardy = 0
        var excused = 0
        var notRecorded = 0
        for (record in records) {
            when (draft[record.studentUserId] ?: record.status) {
                AttendanceMarkStatus.Present.raw -> present++
                AttendanceMarkStatus.Absent.raw -> absent++
                AttendanceMarkStatus.Tardy.raw -> tardy++
                AttendanceMarkStatus.Excused.raw -> excused++
                else -> notRecorded++
            }
        }
        return AttendanceSummaryCounts(present, absent, tardy, excused, notRecorded)
    }

    fun recordsPayload(
        records: List<AttendanceRecord>,
        draft: Map<String, String>,
    ): List<AttendanceRecordUpsert> =
        records.map { record ->
            AttendanceRecordUpsert(
                studentUserId = record.studentUserId,
                status = draft[record.studentUserId] ?: record.status,
            )
        }

    fun findTodaysOpenRollCallSession(
        sessions: List<AttendanceSession>,
        date: String = todayDateString(),
    ): AttendanceSession? =
        sessions.firstOrNull { session ->
            session.collectionMethod == "roll_call" &&
                session.status == "open" &&
                session.sessionDate == date
        }

    fun shouldTakeSession(session: AttendanceSession, isStaff: Boolean): Boolean =
        isStaff && session.collectionMethod == "roll_call"
}