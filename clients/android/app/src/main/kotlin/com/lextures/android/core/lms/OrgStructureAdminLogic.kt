package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

/** Organizations, org units, and academic terms admin helpers (M14.4). */
object OrgStructureAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"
    const val ORG_UNITS_ADMIN_PERMISSION = "tenant:org:units:admin"
    const val DEFAULT_TERM_TYPE = "semester"
    const val ORG_LIST_LIMIT = 200

    private val isoDateFormat: SimpleDateFormat by lazy {
        SimpleDateFormat("yyyy-MM-dd", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }
    }

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings

    fun canManageOrganizations(permissions: List<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION)

    fun canManageOrgUnitsAndTerms(permissions: List<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION) ||
            permissions.contains(ORG_UNITS_ADMIN_PERMISSION)

    fun shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean =
        adminSettingsEnabled(features) &&
            (canManageOrganizations(permissions) || canManageOrgUnitsAndTerms(permissions))

    fun canView(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean = shouldShowEntry(features, permissions)

    fun webOrganizationsPath(): String = "/settings/organizations"
    fun webOrgUnitsPath(): String = "/settings/org-units"
    fun webTermsPath(): String = "/settings/terms"

    fun resolveOrgId(accessToken: String?, courses: List<CourseSummary>): String? =
        CourseCreateLogic.resolveOrgId(accessToken, courses)

    fun normalizedName(value: String): String = value.trim()

    fun isValidTermName(value: String): Boolean = normalizedName(value).isNotEmpty()

    fun isoDateString(date: Date): String = isoDateFormat.format(date)

    fun dateFromIso(value: String?): Date? {
        val trimmed = value?.trim().orEmpty()
        if (trimmed.isEmpty()) return null
        return runCatching { isoDateFormat.parse(trimmed) }.getOrNull()
    }

    fun formatDateRange(start: String?, end: String?): String {
        val startText = start?.takeIf { it.isNotBlank() } ?: "—"
        val endText = end?.takeIf { it.isNotBlank() } ?: "—"
        return "$startText — $endText"
    }

    fun isValidDateRange(start: String, end: String): Boolean {
        val startDate = dateFromIso(start) ?: return false
        val endDate = dateFromIso(end) ?: return false
        return !endDate.before(startDate)
    }

    fun createTermRequest(
        name: String,
        termType: String,
        startDate: String,
        endDate: String,
    ): CreateAcademicTermRequest =
        CreateAcademicTermRequest(
            name = normalizedName(name),
            termType = termType.ifBlank { DEFAULT_TERM_TYPE },
            startDate = startDate,
            endDate = endDate,
        )

    fun patchTermDatesRequest(startDate: String, endDate: String): PatchAcademicTermRequest =
        PatchAcademicTermRequest(startDate = startDate, endDate = endDate)

    fun patchOrgUnitNameRequest(name: String): PatchOrgUnitRequest =
        PatchOrgUnitRequest(name = normalizedName(name))

    fun flattenTree(nodes: List<OrgUnitTreeNode>): List<OrgUnitTreeNode> =
        nodes.flatMap { node -> listOf(node) + flattenTree(node.children.orEmpty()) }

    fun userFacingError(error: Throwable, genericMessage: String): String =
        (error as? ApiError.HttpStatus)?.message?.takeIf { it.isNotEmpty() } ?: genericMessage
}
