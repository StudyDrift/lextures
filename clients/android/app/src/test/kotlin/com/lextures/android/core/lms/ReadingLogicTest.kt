package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.LocalDate

class ReadingLogicTest {
    @Test
    fun weeklyPagesCountsLastSevenDays() {
        val entries = listOf(
            ReadingLogEntry(id = "1", logDate = "2026-07-01", pagesRead = 10),
            ReadingLogEntry(id = "2", logDate = "2025-01-01", pagesRead = 99),
            ReadingLogEntry(id = "3", logDate = "2026-07-02", pagesRead = 5),
        )
        assertEquals(15, ReadingLogic.weeklyPages(entries, LocalDate.parse("2026-07-02")))
    }

    @Test
    fun readingStreakCountsConsecutiveDays() {
        val entries = listOf(
            ReadingLogEntry(id = "1", logDate = "2026-07-02", pagesRead = 1),
            ReadingLogEntry(id = "2", logDate = "2026-07-01", pagesRead = 1),
            ReadingLogEntry(id = "3", logDate = "2026-06-29", pagesRead = 1),
        )
        assertEquals(2, ReadingLogic.readingStreakDays(entries, LocalDate.parse("2026-07-02")))
    }

    @Test
    fun logEntryValidRequiresTitleOrBookId() {
        assertFalse(ReadingLogic.logEntryValid("", null, "2026-07-02"))
        assertTrue(ReadingLogic.logEntryValid("Frog and Toad", null, "2026-07-02"))
        assertTrue(ReadingLogic.logEntryValid(null, "book-1", "2026-07-02"))
    }

    @Test
    fun bookClubCoursesFiltersStudentGroups() {
        val courses = listOf(
            CourseSummary(
                id = "1",
                courseCode = "a",
                title = "Alpha",
                description = "",
                groupSpacesEnabled = true,
                viewerEnrollmentRoles = listOf("student"),
            ),
            CourseSummary(
                id = "2",
                courseCode = "b",
                title = "Beta",
                description = "",
                groupSpacesEnabled = false,
                viewerEnrollmentRoles = listOf("student"),
            ),
        )
        assertEquals(1, ReadingLogic.bookClubCourses(courses).size)
        assertEquals("a", ReadingLogic.bookClubCourses(courses).first().courseCode)
    }
}