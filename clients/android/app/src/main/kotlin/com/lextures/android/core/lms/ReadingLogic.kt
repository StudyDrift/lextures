package com.lextures.android.core.lms

import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter

object ReadingLogic {
    const val REFLECTION_MAX_LENGTH = 500
    val GRADE_BANDS = listOf("", "K-2", "3-5", "6-8", "9-12")

    private val isoDate = DateTimeFormatter.ISO_LOCAL_DATE

    fun todayIso(): String = LocalDate.now(ZoneId.systemDefault()).format(isoDate)

    fun logEntryValid(bookTitle: String?, bookId: String?, logDate: String): Boolean {
        val title = bookTitle?.trim().orEmpty()
        val book = bookId?.trim().orEmpty()
        val date = logDate.trim()
        return (title.isNotEmpty() || book.isNotEmpty()) && date.isNotEmpty()
    }

    fun weeklyPages(entries: List<ReadingLogEntry>, asOf: LocalDate = LocalDate.now()): Int {
        val cutoff = asOf.minusDays(6)
        return entries.sumOf { entry ->
            val date = runCatching { LocalDate.parse(entry.logDate, isoDate) }.getOrNull() ?: return@sumOf 0
            if (date.isBefore(cutoff)) 0 else entry.pagesRead ?: 0
        }
    }

    fun readingStreakDays(entries: List<ReadingLogEntry>, asOf: LocalDate = LocalDate.now()): Int {
        val dates = entries.mapNotNull { runCatching { LocalDate.parse(it.logDate, isoDate) }.getOrNull() }.toSet()
        if (dates.isEmpty()) return 0
        var streak = 0
        var cursor = asOf
        while (dates.contains(cursor)) {
            streak += 1
            cursor = cursor.minusDays(1)
        }
        return streak
    }

    fun resolveOrgId(courses: List<CourseSummary>): String? =
        courses.mapNotNull { it.orgId?.trim()?.takeIf { id -> id.isNotEmpty() } }.firstOrNull()

    fun bookClubCourses(courses: List<CourseSummary>): List<CourseSummary> =
        courses
            .filter { it.viewerIsStudent && it.isGroupSpacesEnabled }
            .sortedBy { it.displayTitle.lowercase() }

    fun formatLexile(level: Int?): String? = level?.let { "Lexile ${it}L" }

    fun bookSubtitle(book: LibraryBook): String? {
        book.author?.trim()?.takeIf { it.isNotEmpty() }?.let { return it }
        return formatLexile(book.lexileLevel) ?: book.fpBand ?: book.gradeBand
    }
}