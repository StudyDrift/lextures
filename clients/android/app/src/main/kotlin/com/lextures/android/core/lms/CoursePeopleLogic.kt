package com.lextures.android.core.lms

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

object CoursePeopleLogic {
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
}