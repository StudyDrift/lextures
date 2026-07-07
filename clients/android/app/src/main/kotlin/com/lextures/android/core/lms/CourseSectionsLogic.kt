package com.lextures.android.core.lms

/** Course sections, roster assignment, overrides, and cross-listing helpers (M13.3). */
object CourseSectionsLogic {
    const val GLOBAL_ADMIN_PERMISSION = "global:app:rbac:manage"
    const val ORG_UNITS_ADMIN_PERMISSION = "tenant:org:units:admin"

    fun canManageCrossListing(permissions: List<String>): Boolean =
        permissions.contains(GLOBAL_ADMIN_PERMISSION) || permissions.contains(ORG_UNITS_ADMIN_PERMISSION)

    fun canAssignStudents(courseCode: String, permissions: List<String>): Boolean =
        permissions.contains("course:$courseCode:enrollments:update")

    fun shouldShowEditors(sectionsEnabled: Boolean): Boolean = sectionsEnabled

    fun activeSections(sections: List<CourseSection>): List<CourseSection> =
        sections.filter { it.isActive }

    fun rosterCount(sectionId: String, enrollments: List<CourseEnrollment>): Int =
        enrollments.count { it.sectionId == sectionId && CoursePeopleLogic.isStudentRole(it.role) }

    fun assignmentItems(structure: List<CourseStructureItem>): List<CourseStructureItem> =
        structure.filter { it.kind == "assignment" }

    fun crossListGroup(courseId: String, groups: List<CrossListGroup>): CrossListGroup? =
        groups.firstOrNull { it.courseId == courseId }

    fun crossListAddCandidates(
        activeSections: List<CourseSection>,
        group: CrossListGroup?,
    ): List<CourseSection> {
        val memberIds = (group?.members ?: emptyList()).map { it.sectionId }.toSet()
        return activeSections.filter { it.id !in memberIds }
    }

    fun cacheKeySections(courseCode: String): String = "course:$courseCode:sections-settings"

    fun createSectionIdempotencyKey(courseCode: String, sectionCode: String): String =
        "course-sections:$courseCode:create:${sectionCode.lowercase()}"

    fun patchSectionIdempotencyKey(courseCode: String, sectionId: String): String =
        "course-sections:$courseCode:patch:$sectionId"

    fun archiveSectionIdempotencyKey(courseCode: String, sectionId: String): String =
        "course-sections:$courseCode:archive:$sectionId"

    fun overrideIdempotencyKey(sectionId: String, itemId: String): String =
        "section-override:$sectionId:$itemId"

    fun enrollmentSectionIdempotencyKey(enrollmentId: String, sectionId: String): String =
        "enrollment-section:$enrollmentId:$sectionId"

    fun crossListCreateIdempotencyKey(orgId: String, courseCode: String): String =
        "cross-list:$orgId:$courseCode:create"

    fun crossListAddMemberIdempotencyKey(orgId: String, groupId: String, sectionId: String): String =
        "cross-list:$orgId:$groupId:add:$sectionId"

    fun crossListRemoveMemberIdempotencyKey(orgId: String, groupId: String, sectionId: String): String =
        "cross-list:$orgId:$groupId:remove:$sectionId"

    fun buildOverrideBody(dueAtLocal: String): SectionAssignmentOverrideBody? {
        val trimmed = dueAtLocal.trim()
        if (trimmed.isEmpty()) {
            return SectionAssignmentOverrideBody(dueAt = null, availableFrom = null, availableUntil = null)
        }
        val iso = CourseSettingsLogic.localDateStringToIso(trimmed) ?: return null
        return SectionAssignmentOverrideBody(dueAt = iso, availableFrom = null, availableUntil = null)
    }

    fun validateCreateSection(sectionCode: String): String? =
        if (sectionCode.trim().isEmpty()) "section-code-required" else null

    fun mutationsDisabledReason(isOnline: Boolean): String? =
        if (!isOnline) "offline-mutations-disabled" else null
}

@kotlinx.serialization.Serializable
data class CrossListMember(
    val sectionId: String,
    val isPrimary: Boolean = false,
    val sectionCode: String,
    val sectionName: String? = null,
) {
    val displayLabel: String
        get() {
            val name = sectionName?.trim().orEmpty()
            return if (name.isNotEmpty()) "$sectionCode — $name" else sectionCode
        }
}

@kotlinx.serialization.Serializable
data class CrossListGroup(
    val id: String,
    val courseId: String,
    val name: String? = null,
    val createdAt: String? = null,
    val primarySectionId: String? = null,
    val members: List<CrossListMember> = emptyList(),
)

@kotlinx.serialization.Serializable
data class CrossListGroupsResponse(
    val groups: List<CrossListGroup> = emptyList(),
)

@kotlinx.serialization.Serializable
data class CreateCourseSectionBody(
    val sectionCode: String,
    val name: String? = null,
)

@kotlinx.serialization.Serializable
data class PatchCourseSectionBody(
    val sectionCode: String? = null,
    val name: String? = null,
    val status: String? = null,
)

@kotlinx.serialization.Serializable
data class SectionAssignmentOverrideBody(
    val dueAt: String? = null,
    val availableFrom: String? = null,
    val availableUntil: String? = null,
)

@kotlinx.serialization.Serializable
data class CreateCrossListGroupBody(
    val courseCode: String,
    val primarySectionId: String,
    val name: String? = null,
)

@kotlinx.serialization.Serializable
data class AddCrossListMemberBody(
    val sectionId: String,
)

@kotlinx.serialization.Serializable
data class EnrollmentSectionPatchBody(
    val sectionId: String,
)

@kotlinx.serialization.Serializable
data class CourseSectionsCachedPayload(
    val sections: List<CourseSection> = emptyList(),
    val enrollments: List<CourseEnrollment> = emptyList(),
    val assignments: List<CourseStructureItem> = emptyList(),
)
