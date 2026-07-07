package com.lextures.android.core.lms

/** District blueprint course helpers (M13.11). */
object CourseBlueprintLogic {
    const val GLOBAL_ADMIN_PERMISSION = "global:app:rbac:manage"
    const val ORG_UNITS_ADMIN_PERMISSION = "tenant:org:units:admin"

    enum class BlueprintRole {
        Master,
        Child,
        None,
    }

    data class BlueprintRoleState(val role: BlueprintRole, val parentCode: String? = null)

    fun canManageBlueprint(course: CourseSummary, permissions: List<String>): Boolean {
        if (course.orgId.isNullOrBlank()) return false
        return permissions.contains(GLOBAL_ADMIN_PERMISSION) || permissions.contains(ORG_UNITS_ADMIN_PERMISSION)
    }

    fun blueprintRole(course: CourseSummary): BlueprintRoleState {
        if (course.isBlueprint == true) return BlueprintRoleState(BlueprintRole.Master)
        val parent = course.blueprintParentCourseCode?.trim().orEmpty()
        if (parent.isNotEmpty()) return BlueprintRoleState(BlueprintRole.Child, parent)
        return BlueprintRoleState(BlueprintRole.None)
    }

    fun shouldLoadBlueprintDetails(course: CourseSummary, canManage: Boolean): Boolean =
        canManage && course.isBlueprint == true

    fun cacheKeyBlueprintData(courseCode: String): String = "course:$courseCode:blueprint"

    fun formatSyncAt(raw: String?): String {
        val formatted = LmsDates.shortDateTime(raw)
        return formatted.ifEmpty { "—" }
    }

    fun pushDisabledReason(isOnline: Boolean, childCount: Int): String? = when {
        !isOnline -> "offline-push"
        childCount == 0 -> "no-children"
        else -> null
    }

    fun mutationsDisabledReason(isOnline: Boolean): String? =
        if (!isOnline) "offline-mutations" else null

    fun userFacingError(error: Throwable): String = error.message ?: "generic"
}
