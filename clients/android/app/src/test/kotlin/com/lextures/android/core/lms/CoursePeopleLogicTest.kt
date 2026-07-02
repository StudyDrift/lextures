package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CoursePeopleLogicTest {
    private fun enrollment(
        id: String,
        name: String? = null,
        role: String,
        sectionId: String? = null,
    ) = CourseEnrollment(
        id = id,
        userId = "user-$id",
        displayName = name,
        role = role,
        sectionId = sectionId,
    )

    @Test
    fun enrollmentRoleRankOrdersStaffBeforeStudents() {
        assertTrue(CoursePeopleLogic.enrollmentRoleRank("teacher") < CoursePeopleLogic.enrollmentRoleRank("student"))
        assertTrue(CoursePeopleLogic.enrollmentRoleRank("ta") < CoursePeopleLogic.enrollmentRoleRank("student"))
    }

    @Test
    fun filterByRoleAndSection() {
        val rows = listOf(
            enrollment(id = "1", name = "Alex", role = "student", sectionId = "sec-a"),
            enrollment(id = "2", name = "Blair", role = "teacher"),
            enrollment(id = "3", name = "Casey", role = "student", sectionId = "sec-b"),
        )
        val staffOnly = CoursePeopleLogic.filter(rows, "", CoursePeopleRoleFilter.Staff, null)
        assertEquals(listOf("2"), staffOnly.map { it.id })

        val sectionA = CoursePeopleLogic.filter(rows, "", CoursePeopleRoleFilter.All, "sec-a")
        assertEquals(listOf("1"), sectionA.map { it.id })
    }

    @Test
    fun searchMatchesDisplayName() {
        val rows = listOf(
            enrollment(id = "1", name = "Alex Rivera", role = "student"),
            enrollment(id = "2", name = "Blair", role = "ta"),
        )
        val matches = CoursePeopleLogic.filter(rows, "rivera", CoursePeopleRoleFilter.All, null)
        assertEquals(listOf("1"), matches.map { it.id })
    }

    @Test
    fun groupedSectionsOrdersTeachersThenStudents() {
        val rows = listOf(
            enrollment(id = "1", name = "Zoe", role = "student"),
            enrollment(id = "2", name = "Alex", role = "teacher"),
            enrollment(id = "3", name = "Blair", role = "ta"),
        )
        val groups = CoursePeopleLogic.groupedSections(rows)
        assertEquals(
            listOf(
                CoursePeopleGroupKind.Teachers,
                CoursePeopleGroupKind.Tas,
                CoursePeopleGroupKind.Students,
            ),
            groups.map { it.kind },
        )
        assertEquals(listOf("2"), groups[0].enrollments.map { it.id })
        assertEquals(listOf("1"), groups[2].enrollments.map { it.id })
    }

    @Test
    fun canUpdateEnrollmentsPermission() {
        assertTrue(
            CoursePeopleLogic.canUpdateEnrollments(
                courseCode = "BIO101",
                permissions = listOf("course:BIO101:enrollments:update"),
            ),
        )
        assertFalse(
            CoursePeopleLogic.canUpdateEnrollments(
                courseCode = "BIO101",
                permissions = listOf("course:BIO101:enrollments:read"),
            ),
        )
    }
}