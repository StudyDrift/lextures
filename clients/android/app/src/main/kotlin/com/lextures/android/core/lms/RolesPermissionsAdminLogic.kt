package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError

/** Roles & permissions admin helpers (M14.2). */
object RolesPermissionsAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings

    fun canManageRoles(permissions: List<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION)

    fun shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean =
        adminSettingsEnabled(features) && canManageRoles(permissions)

    fun canView(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean = shouldShowEntry(features, permissions)

    fun webSettingsPath(): String = "/settings/roles"

    fun filterRoles(roles: List<RoleWithPermissions>, query: String): List<RoleWithPermissions> {
        val trimmed = query.trim()
        if (trimmed.isEmpty()) return roles
        val needle = trimmed.lowercase()
        return roles.filter { role ->
            role.name.lowercase().contains(needle) ||
                role.description?.lowercase()?.contains(needle) == true ||
                role.permissions.any { permission ->
                    permission.permissionString.lowercase().contains(needle) ||
                        permission.description.lowercase().contains(needle)
                }
        }
    }

    fun filterPermissions(permissions: List<RbacPermission>, query: String): List<RbacPermission> {
        val trimmed = query.trim()
        if (trimmed.isEmpty()) return permissions
        val needle = trimmed.lowercase()
        return permissions.filter { permission ->
            permission.permissionString.lowercase().contains(needle) ||
                permission.description.lowercase().contains(needle)
        }
    }

    fun filterUsers(users: List<RbacUserBrief>, query: String): List<RbacUserBrief> {
        val trimmed = query.trim()
        if (trimmed.isEmpty()) return users
        val needle = trimmed.lowercase()
        return users.filter { user ->
            userDisplayLabel(user).lowercase().contains(needle) ||
                user.email.lowercase().contains(needle) ||
                user.sid?.lowercase()?.contains(needle) == true
        }
    }

    fun userDisplayLabel(user: RbacUserBrief): String {
        val name = user.displayName?.trim().orEmpty()
        if (name.isNotEmpty()) return name
        return user.email
    }

    fun roleGrantsRbacManage(role: RoleWithPermissions): Boolean =
        role.permissions.any { it.permissionString == RBAC_MANAGE_PERMISSION }

    fun blocksSelfElevation(
        role: RoleWithPermissions,
        targetUserId: String,
        currentUserId: String?,
    ): Boolean {
        if (currentUserId.isNullOrEmpty()) return false
        return targetUserId == currentUserId && roleGrantsRbacManage(role)
    }

    fun addRoleUserRequest(userId: String): AddRoleUserRequest = AddRoleUserRequest(userId = userId)

    fun userFacingError(error: Throwable, genericMessage: String): String =
        (error as? ApiError.HttpStatus)?.message?.takeIf { it.isNotEmpty() } ?: genericMessage
}
