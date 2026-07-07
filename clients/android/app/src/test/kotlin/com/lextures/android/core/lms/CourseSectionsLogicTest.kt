package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseSectionsLogicTest {
    @Test
    fun shouldShowEditorsWhenSectionsEnabled() {
        assertTrue(CourseSectionsLogic.shouldShowEditors(true))
        assertFalse(CourseSectionsLogic.shouldShowEditors(false))
    }

    @Test
    fun activeSectionsFiltersArchived() {
        val sections = listOf(
            CourseSection(id = "1", sectionCode = "A", status = "active"),
            CourseSection(id = "2", sectionCode = "B", status = "archived"),
        )
        assertEquals(listOf("1"), CourseSectionsLogic.activeSections(sections).map { it.id })
    }

    @Test
    fun rosterCountStudentsOnly() {
        val enrollments = listOf(
            enrollment("e1", "student", "sec-a"),
            enrollment("e2", "teacher", "sec-a"),
            enrollment("e3", "student", "sec-b"),
        )
        assertEquals(1, CourseSectionsLogic.rosterCount("sec-a", enrollments))
    }

    @Test
    fun assignmentItemsFromStructure() {
        val items = listOf(
            CourseStructureItem(id = "1", sortOrder = 0, kind = "assignment", title = "Essay", published = true),
            CourseStructureItem(id = "2", sortOrder = 1, kind = "module", title = "Week 1", published = true),
        )
        assertEquals(listOf("1"), CourseSectionsLogic.assignmentItems(items).map { it.id })
    }

    @Test
    fun buildOverrideBodyRejectsInvalidDate() {
        assertNull(CourseSectionsLogic.buildOverrideBody("not-a-date"))
    }

    @Test
    fun crossListAddCandidates() {
        val sections = listOf(
            CourseSection(id = "s1", sectionCode = "001", status = "active"),
            CourseSection(id = "s2", sectionCode = "002", status = "active"),
        )
        val group = CrossListGroup(
            id = "g1",
            courseId = "c1",
            members = listOf(CrossListMember(sectionId = "s1", isPrimary = true, sectionCode = "001")),
        )
        assertEquals(listOf("s2"), CourseSectionsLogic.crossListAddCandidates(sections, group).map { it.id })
    }

    private fun enrollment(id: String, role: String, sectionId: String) = CourseEnrollment(
        id = id,
        userId = "u-$id",
        displayName = "User $id",
        role = role,
        sectionId = sectionId,
    )
}
