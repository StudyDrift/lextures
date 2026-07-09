package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError

/** Global archived-course admin helpers (M14.10). */
object ArchivedCoursesAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings

    fun canManageArchivedCourses(permissions: List<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION)

    fun shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean =
        adminSettingsEnabled(features) && canManageArchivedCourses(permissions)

    fun canView(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean = shouldShowEntry(features, permissions)

    fun filterRows(rows: List<ArchivedCourseRow>, query: String): List<ArchivedCourseRow> {
        val trimmed = query.trim()
        if (trimmed.isEmpty()) return rows
        val needle = trimmed.lowercase()
        return rows.filter { row ->
            row.title.lowercase().contains(needle) ||
                row.courseCode.lowercase().contains(needle) ||
                archivedByLabel(row).lowercase().contains(needle)
        }
    }

    fun archivedByLabel(row: ArchivedCourseRow): String {
        val name = row.archivedByName?.trim().orEmpty()
        if (name.isNotEmpty()) return name
        val email = row.archivedByEmail?.trim().orEmpty()
        if (email.isNotEmpty()) return email
        return "—"
    }

    fun formatArchivedAt(raw: String?): String {
        val formatted = LmsDates.shortDateTime(raw)
        return formatted.ifEmpty { "—" }
    }

    fun deleteConfirmPhrase(row: ArchivedCourseRow): String =
        row.courseCode.trim()

    fun deleteConfirmMatches(typed: String, row: ArchivedCourseRow): Boolean =
        typed.trim().equals(deleteConfirmPhrase(row), ignoreCase = true)

    fun rowsAfterRestore(rows: List<ArchivedCourseRow>, courseCode: String): List<ArchivedCourseRow> =
        rows.filter { it.courseCode != courseCode }

    fun rowsAfterDelete(rows: List<ArchivedCourseRow>, courseCode: String): List<ArchivedCourseRow> =
        rows.filter { it.courseCode != courseCode }

    fun userFacingError(error: Throwable): String =
        (error as? ApiError.HttpStatus)?.message?.takeIf { it.isNotEmpty() }
            ?: "generic"
}