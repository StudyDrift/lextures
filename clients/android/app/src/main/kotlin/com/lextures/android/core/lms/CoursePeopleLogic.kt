package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

enum class CoursePeopleRoleFilter {
    All,
    Staff,
    Students,
}

enum class CoursePeopleGroupKind {
    Teachers,
    Tas,
    Students,
    Other,
}

data class CoursePeopleGroup(
    val kind: CoursePeopleGroupKind,
    val enrollments: List<CourseEnrollment>,
)

data class CoursePeopleAssignableRole(
    val value: String,
    val labelKey: String,
)

data class CoursePeopleAddResultSummary(
    val added: List<String>,
    val alreadyEnrolled: List<String>,
    val notFound: List<String>,
) {
    val hasConflicts: Boolean get() = alreadyEnrolled.isNotEmpty() || notFound.isNotEmpty()
    val didAdd: Boolean get() = added.isNotEmpty()
}

object CoursePeopleLogic {
    val assignableRoles: List<CoursePeopleAssignableRole> = listOf(
        CoursePeopleAssignableRole("student", "mobile.people.role.student"),
        CoursePeopleAssignableRole("instructor", "mobile.people.role.teacher"),
        CoursePeopleAssignableRole("ta", "mobile.people.role.ta"),
        CoursePeopleAssignableRole("designer", "mobile.people.add.role.designer"),
        CoursePeopleAssignableRole("observer", "mobile.people.add.role.observer"),
        CoursePeopleAssignableRole("auditor", "mobile.people.add.role.auditor"),
        CoursePeopleAssignableRole("librarian", "mobile.people.add.role.librarian"),
    )

    val managedEnrollmentStates: List<String> = listOf(
        "active", "dropped", "withdrawn", "waitlist", "audit", "no_credit", "incomplete",
    )
    fun normalizedRole(role: String): String = role.trim().lowercase()

    fun enrollmentRoleRank(role: String): Int =
        when (normalizedRole(role)) {
            "owner", "teacher" -> 0
            "instructor" -> 1
            "ta" -> 2
            "designer" -> 3
            "observer" -> 4
            "auditor" -> 5
            "librarian" -> 6
            "student" -> 7
            else -> 8
        }

    fun isStaffRole(role: String): Boolean = enrollmentRoleRank(role) < 7

    fun isStudentRole(role: String): Boolean = normalizedRole(role) == "student"

    fun groupKind(role: String): CoursePeopleGroupKind =
        when (normalizedRole(role)) {
            "owner", "teacher", "instructor" -> CoursePeopleGroupKind.Teachers
            "ta" -> CoursePeopleGroupKind.Tas
            "student" -> CoursePeopleGroupKind.Students
            else -> CoursePeopleGroupKind.Other
        }

    fun displayName(enrollment: CourseEnrollment): String =
        enrollment.displayName?.trim()?.takeIf { it.isNotEmpty() } ?: "Unnamed"

    fun initials(enrollment: CourseEnrollment): String {
        val source = displayName(enrollment)
        val parts = source.split(" ").filter { it.isNotEmpty() }
        return if (parts.size >= 2) {
            "${parts.first().first()}${parts.last().first()}".uppercase()
        } else {
            source.take(2).uppercase()
        }
    }

    fun roleLabel(enrollment: CourseEnrollment): String {
        val custom = enrollment.roleDisplay?.trim().orEmpty()
        if (custom.isNotEmpty()) return custom
        return when (normalizedRole(enrollment.role)) {
            "owner", "teacher", "instructor" -> "Teacher"
            "ta" -> "Teaching assistant"
            "student" -> "Student"
            else -> enrollment.role
        }
    }

    fun sectionLabel(enrollment: CourseEnrollment): String? =
        enrollment.sectionName?.trim()?.takeIf { it.isNotEmpty() }
            ?: enrollment.sectionCode?.trim()?.takeIf { it.isNotEmpty() }

    fun matchesSearch(enrollment: CourseEnrollment, query: String): Boolean {
        val needle = query.trim().lowercase()
        if (needle.isEmpty()) return true
        val haystack = listOfNotNull(
            displayName(enrollment),
            roleLabel(enrollment),
            sectionLabel(enrollment),
            enrollment.role,
        ).joinToString(" ").lowercase()
        return haystack.contains(needle)
    }

    fun filter(
        enrollments: List<CourseEnrollment>,
        search: String,
        roleFilter: CoursePeopleRoleFilter,
        sectionId: String?,
    ): List<CourseEnrollment> =
        enrollments.filter { enrollment ->
            if (!matchesSearch(enrollment, search)) return@filter false
            when (roleFilter) {
                CoursePeopleRoleFilter.All -> Unit
                CoursePeopleRoleFilter.Staff -> if (!isStaffRole(enrollment.role)) return@filter false
                CoursePeopleRoleFilter.Students -> if (!isStudentRole(enrollment.role)) return@filter false
            }
            if (!sectionId.isNullOrEmpty() && enrollment.sectionId != sectionId) return@filter false
            true
        }

    fun groupedSections(enrollments: List<CourseEnrollment>): List<CoursePeopleGroup> {
        val buckets = enrollments.groupBy { groupKind(it.role) }
        return CoursePeopleGroupKind.entries.mapNotNull { kind ->
            val rows = buckets[kind].orEmpty()
            if (rows.isEmpty()) return@mapNotNull null
            val sorted = rows.sortedWith(
                compareBy<CourseEnrollment> { displayName(it).lowercase() }
                    .thenBy { enrollmentRoleRank(it.role) },
            )
            CoursePeopleGroup(kind = kind, enrollments = sorted)
        }
    }

    fun canUpdateEnrollments(courseCode: String, permissions: List<String>): Boolean =
        permissions.contains("course:$courseCode:enrollments:update")

    fun enrollmentAddEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileEnrollmentAdd

    fun canAddEnrollments(
        courseCode: String,
        permissions: List<String>,
        features: MobilePlatformFeatures,
        isOnline: Boolean,
    ): Boolean {
        if (!enrollmentAddEnabled(features)) return false
        if (!isOnline) return false
        return canUpdateEnrollments(courseCode, permissions)
    }

    fun canChangeEnrollmentState(
        enrollment: CourseEnrollment,
        courseCode: String,
        permissions: List<String>,
        features: MobilePlatformFeatures,
        isOnline: Boolean,
    ): Boolean {
        if (!features.ffEnrollmentStateMachine) return false
        if (!isOnline) return false
        if (!canUpdateEnrollments(courseCode, permissions)) return false
        return isStudentRole(enrollment.role)
    }

    fun parseEmails(raw: String): List<String> {
        val seen = linkedSetOf<String>()
        raw.split(',', ';', '\n', '\r', '\t', ' ')
            .map { it.trim().lowercase() }
            .filter { it.isNotEmpty() }
            .forEach { seen.add(it) }
        return seen.toList()
    }

    fun isValidEmail(email: String): Boolean {
        val trimmed = email.trim()
        if (trimmed.length !in 3..254) return false
        val at = trimmed.indexOf('@')
        if (at <= 0 || at >= trimmed.lastIndex) return false
        val local = trimmed.substring(0, at)
        val domain = trimmed.substring(at + 1)
        if (local.isEmpty() || domain.isEmpty()) return false
        if (!domain.contains('.')) return false
        if ('@' in local || '@' in domain) return false
        return true
    }

    fun validateEmailsForAdd(raw: String): Result<List<String>> {
        val emails = parseEmails(raw)
        if (emails.isEmpty()) return Result.failure(IllegalArgumentException("mobile.people.add.error.emailsRequired"))
        if (emails.any { !isValidEmail(it) }) {
            return Result.failure(IllegalArgumentException("mobile.people.add.error.invalidEmail"))
        }
        return Result.success(emails)
    }

    fun normalizeCourseRole(role: String): String {
        val value = role.trim().lowercase()
        return if (value == "teacher" || value == "owner") "instructor" else value
    }

    fun isAssignableRole(role: String): Boolean {
        val normalized = normalizeCourseRole(role)
        return assignableRoles.any { it.value == normalized }
    }

    fun buildAddRequest(emails: List<String>, courseRole: String): AddCourseEnrollmentsRequest =
        AddCourseEnrollmentsRequest(
            emails = emails.joinToString("\n"),
            courseRole = normalizeCourseRole(courseRole),
        )

    fun summarizeAddResponse(response: AddCourseEnrollmentsResponse): CoursePeopleAddResultSummary =
        CoursePeopleAddResultSummary(
            added = response.added,
            alreadyEnrolled = response.alreadyEnrolled,
            notFound = response.notFound,
        )

    fun normalizedState(state: String?): String {
        val value = state?.trim()?.lowercase().orEmpty()
        return value.ifEmpty { "active" }
    }

    fun isInactiveState(state: String?): Boolean =
        when (normalizedState(state)) {
            "dropped", "withdrawn", "no_credit" -> true
            else -> false
        }

    fun stateLabelKey(state: String?): String =
        when (normalizedState(state)) {
            "active" -> "mobile.people.state.active"
            "waitlist" -> "mobile.people.state.waitlist"
            "dropped" -> "mobile.people.state.dropped"
            "withdrawn" -> "mobile.people.state.withdrawn"
            "audit" -> "mobile.people.state.audit"
            "no_credit" -> "mobile.people.state.noCredit"
            "incomplete" -> "mobile.people.state.incomplete"
            else -> "mobile.people.state.active"
        }

    fun deactivateState(forCurrent: String?): String =
        if (isInactiveState(forCurrent)) "active" else "dropped"
}