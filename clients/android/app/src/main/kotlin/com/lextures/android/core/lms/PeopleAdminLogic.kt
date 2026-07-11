package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError

/** People / user management admin helpers (M14.3). */
object PeopleAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"
    const val ERASED_EMAIL_SUFFIX = "@erased.invalid"
    const val DEFAULT_PER_PAGE = 25

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings

    fun canManagePeople(permissions: List<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION)

    fun shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean =
        adminSettingsEnabled(features) && canManagePeople(permissions)

    fun canView(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean = shouldShowEntry(features, permissions)

    fun webSettingsPath(): String = "/settings/people"

    fun personDisplayName(row: PersonRow): String =
        personDisplayName(row.displayName, row.firstName, row.lastName, row.email)

    fun personDisplayName(report: PersonReport): String =
        personDisplayName(report.displayName, report.firstName, report.lastName, report.email)

    fun personDisplayName(
        displayName: String?,
        firstName: String?,
        lastName: String?,
        email: String,
    ): String {
        val dn = displayName?.trim().orEmpty()
        if (dn.isNotEmpty()) return dn
        val full = listOfNotNull(firstName?.trim(), lastName?.trim())
            .filter { it.isNotEmpty() }
            .joinToString(" ")
        if (full.isNotEmpty()) return full
        return email
    }

    fun statusLabel(active: Boolean, activeLabel: String, suspendedLabel: String): String =
        if (active) activeLabel else suspendedLabel

    fun isErased(email: String): Boolean =
        email.lowercase().endsWith(ERASED_EMAIL_SUFFIX)

    fun blocksSelfSuspend(targetUserId: String, currentUserId: String?): Boolean {
        if (currentUserId.isNullOrEmpty()) return false
        return targetUserId == currentUserId
    }

    fun normalizedSearchQuery(query: String): String = query.trim()

    fun shouldSearch(query: String): Boolean = normalizedSearchQuery(query).isNotEmpty()

    fun invitePersonRequest(
        email: String,
        firstName: String?,
        lastName: String?,
    ): InvitePersonRequest {
        val trimmedEmail = email.trim()
        val first = firstName?.trim()?.takeIf { it.isNotEmpty() }
        val last = lastName?.trim()?.takeIf { it.isNotEmpty() }
        return InvitePersonRequest(email = trimmedEmail, firstName = first, lastName = last)
    }

    fun patchPersonRequest(active: Boolean): PatchPersonRequest = PatchPersonRequest(active = active)

    fun resendInviteRequest(email: String): ForgotPasswordRequest =
        ForgotPasswordRequest(email = email.trim())

    fun roleMatchesReport(role: RoleWithPermissions, report: PersonReport): Boolean {
        val reportRole = report.role.trim().lowercase()
        if (reportRole.isEmpty()) return false
        return role.name.trim().lowercase() == reportRole
    }

    fun userFacingError(error: Throwable, genericMessage: String): String =
        (error as? ApiError.HttpStatus)?.message?.takeIf { it.isNotEmpty() } ?: genericMessage
}
