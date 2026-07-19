package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
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
        state: String? = null,
        invited: Boolean = false,
    ) = CourseEnrollment(
        id = id,
        userId = "user-$id",
        displayName = name,
        role = role,
        sectionId = sectionId,
        state = state,
        invitationPending = invited,
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

    @Test
    fun canAddEnrollmentsRequiresFlagPermissionAndOnline() {
        val perms = listOf("course:BIO101:enrollments:update")
        assertFalse(
            CoursePeopleLogic.canAddEnrollments(
                courseCode = "BIO101",
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileEnrollmentAdd = false),
                isOnline = true,
            ),
        )
        assertTrue(
            CoursePeopleLogic.canAddEnrollments(
                courseCode = "BIO101",
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileEnrollmentAdd = true),
                isOnline = true,
            ),
        )
        assertFalse(
            CoursePeopleLogic.canAddEnrollments(
                courseCode = "BIO101",
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileEnrollmentAdd = true),
                isOnline = false,
            ),
        )
        assertFalse(
            CoursePeopleLogic.canAddEnrollments(
                courseCode = "BIO101",
                permissions = listOf("course:BIO101:enrollments:read"),
                features = MobilePlatformFeatures(ffMobileEnrollmentAdd = true),
                isOnline = true,
            ),
        )
    }

    @Test
    fun parseAndValidateEmails() {
        assertEquals(
            listOf("alex@school.edu", "blair@school.edu", "casey@school.edu"),
            CoursePeopleLogic.parseEmails("Alex@School.edu, blair@school.edu; casey@school.edu"),
        )
        assertTrue(CoursePeopleLogic.validateEmailsForAdd("").isFailure)
        assertTrue(CoursePeopleLogic.validateEmailsForAdd("not-an-email").isFailure)
        assertEquals(
            listOf("ok@school.edu"),
            CoursePeopleLogic.validateEmailsForAdd("ok@school.edu").getOrThrow(),
        )
    }

    @Test
    fun buildAddRequestAndSummarize() {
        val request = CoursePeopleLogic.buildAddRequest(
            emails = listOf("a@school.edu", "b@school.edu"),
            courseRole = "Teacher",
        )
        assertEquals("a@school.edu\nb@school.edu", request.emails)
        assertEquals("instructor", request.courseRole)
        assertTrue(CoursePeopleLogic.isAssignableRole("ta"))

        val summary = CoursePeopleLogic.summarizeAddResponse(
            AddCourseEnrollmentsResponse(
                added = listOf("a@school.edu"),
                alreadyEnrolled = listOf("b@school.edu"),
                notFound = listOf("c@school.edu"),
            ),
        )
        assertTrue(summary.didAdd)
        assertTrue(summary.hasConflicts)
        assertEquals(listOf("b@school.edu"), summary.alreadyEnrolled)
    }

    @Test
    fun stateHelpersAndChangeGate() {
        assertTrue(CoursePeopleLogic.isInactiveState("dropped"))
        assertFalse(CoursePeopleLogic.isInactiveState("active"))
        assertEquals("dropped", CoursePeopleLogic.deactivateState("active"))
        assertEquals("active", CoursePeopleLogic.deactivateState("dropped"))
        assertEquals("mobile.people.state.waitlist", CoursePeopleLogic.stateLabelKey("waitlist"))

        val student = enrollment(id = "1", role = "student", state = "active")
        val teacher = enrollment(id = "2", role = "teacher")
        val perms = listOf("course:BIO101:enrollments:update")
        assertTrue(
            CoursePeopleLogic.canChangeEnrollmentState(
                enrollment = student,
                courseCode = "BIO101",
                permissions = perms,
                features = MobilePlatformFeatures(ffEnrollmentStateMachine = true),
                isOnline = true,
            ),
        )
        assertFalse(
            CoursePeopleLogic.canChangeEnrollmentState(
                enrollment = teacher,
                courseCode = "BIO101",
                permissions = perms,
                features = MobilePlatformFeatures(ffEnrollmentStateMachine = true),
                isOnline = true,
            ),
        )
        assertFalse(
            CoursePeopleLogic.canChangeEnrollmentState(
                enrollment = student,
                courseCode = "BIO101",
                permissions = perms,
                features = MobilePlatformFeatures(ffEnrollmentStateMachine = false),
                isOnline = true,
            ),
        )
    }
}
