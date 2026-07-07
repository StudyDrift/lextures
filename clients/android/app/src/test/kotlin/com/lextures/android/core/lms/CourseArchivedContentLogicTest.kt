package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseArchivedContentLogicTest {
    @Test
    fun canViewRequiresItemCreatePermission() {
        assertFalse(CourseArchivedContentLogic.canViewArchivedContent("C-1", emptyList()))
        assertTrue(
            CourseArchivedContentLogic.canViewArchivedContent(
                "C-1",
                listOf(CourseSettingsLogic.courseItemCreatePermission("C-1")),
            ),
        )
    }

    @Test
    fun archivedRowsFiltersRestorableItems() {
        val items = listOf(
            CourseStructureItem(
                id = "m1",
                sortOrder = 0,
                kind = "module",
                title = "Week 1",
                parentId = null,
                published = true,
                archived = false,
            ),
            CourseStructureItem(
                id = "p1",
                sortOrder = 1,
                kind = "content_page",
                title = "Reading",
                parentId = "m1",
                published = false,
                archived = true,
                updatedAt = "2026-01-15T12:00:00Z",
            ),
            CourseStructureItem(
                id = "q1",
                sortOrder = 2,
                kind = "quiz",
                title = "Quiz",
                parentId = "m1",
                published = true,
                archived = true,
            ),
            CourseStructureItem(
                id = "x1",
                sortOrder = 3,
                kind = "scorm",
                title = "Package",
                parentId = "m1",
                published = true,
                archived = true,
            ),
        )

        val rows = CourseArchivedContentLogic.archivedRows(items)
        assertEquals(listOf("p1", "q1"), rows.map { it.id })
        assertEquals("Week 1", rows.first().moduleTitle)
        assertEquals("2026-01-15T12:00:00Z", rows.first().archivedAt)
    }

    @Test
    fun itemsAfterRestoreRemovesRow() {
        val items = listOf(
            CourseStructureItem(
                id = "a",
                sortOrder = 0,
                kind = "content_page",
                title = "A",
                parentId = "m1",
                published = true,
                archived = true,
            ),
            CourseStructureItem(
                id = "b",
                sortOrder = 1,
                kind = "assignment",
                title = "B",
                parentId = "m1",
                published = true,
                archived = true,
            ),
        )
        val updated = CourseArchivedContentLogic.itemsAfterRestore(items, "a")
        assertEquals(listOf("b"), updated.map { it.id })
    }

    @Test
    fun kindLabelKey() {
        assertEquals(
            "mobile.courseSettings.archivedContent.kind.contentPage",
            CourseArchivedContentLogic.kindLabelKey("content_page"),
        )
        assertEquals(
            "mobile.courseSettings.archivedContent.kind.other",
            CourseArchivedContentLogic.kindLabelKey("unknown"),
        )
    }

    @Test
    fun cacheKey() {
        assertEquals(
            "course:C-1:archived-structure",
            CourseArchivedContentLogic.cacheKeyArchivedStructure("C-1"),
        )
    }
}
