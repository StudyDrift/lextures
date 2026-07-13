package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class RbacPermission(
    val id: String,
    val permissionString: String,
    val description: String = "",
    val createdAt: String? = null,
)

@Serializable
data class RoleWithPermissions(
    val id: String,
    val name: String,
    val description: String? = null,
    val scope: String? = null,
    val createdAt: String? = null,
    val permissions: List<RbacPermission> = emptyList(),
)

@Serializable
data class RbacUserBrief(
    val id: String,
    val email: String,
    val displayName: String? = null,
    val sid: String? = null,
)

@Serializable
data class RolesListResponse(
    val roles: List<RoleWithPermissions> = emptyList(),
)

@Serializable
data class RoleUsersResponse(
    val users: List<RbacUserBrief> = emptyList(),
)

@Serializable
data class AddRoleUserRequest(
    val userId: String,
)
