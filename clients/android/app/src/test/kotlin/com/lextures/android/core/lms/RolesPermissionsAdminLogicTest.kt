package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import com.lextures.android.core.navigation.MobilePlatformFeatures

class RolesPermissionsAdminLogicTest {
    @Test
    fun adminSettingsEnabledRequiresFlag() {
        val off = MobilePlatformFeatures()
        assertFalse(RolesPermissionsAdminLogic.adminSettingsEnabled(off))
        val on = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertTrue(RolesPermissionsAdminLogic.adminSettingsEnabled(on))
    }

    @Test
    fun canManageRolesRequiresRbacManage() {
        assertFalse(RolesPermissionsAdminLogic.canManageRoles(emptyList()))
        assertTrue(
            RolesPermissionsAdminLogic.canManageRoles(
                listOf(RolesPermissionsAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun shouldShowEntryRequiresFlagAndPermission() {
        val features = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertFalse(
            RolesPermissionsAdminLogic.shouldShowEntry(features, emptyList()),
        )
        assertTrue(
            RolesPermissionsAdminLogic.shouldShowEntry(
                features,
                listOf(RolesPermissionsAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun filterRolesMatchesNameAndPermission() {
        val roles = listOf(
            RoleWithPermissions(
                id = "1",
                name = "Teacher",
                permissions = listOf(
                    RbacPermission(id = "p1", permissionString = "course:foo:item:read", description = "Read"),
                ),
            ),
            RoleWithPermissions(
                id = "2",
                name = "Admin",
                permissions = listOf(
                    RbacPermission(
                        id = "p2",
                        permissionString = RolesPermissionsAdminLogic.RBAC_MANAGE_PERMISSION,
                        description = "Manage",
                    ),
                ),
            ),
        )
        assertEquals(
            listOf("1"),
            RolesPermissionsAdminLogic.filterRoles(roles, "teacher").map { it.id },
        )
        assertEquals(
            listOf("2"),
            RolesPermissionsAdminLogic.filterRoles(roles, "rbac").map { it.id },
        )
    }

    @Test
    fun blocksSelfElevationForRbacManageRole() {
        val role = RoleWithPermissions(
            id = "admin",
            name = "Global Admin",
            permissions = listOf(
                RbacPermission(
                    id = "p1",
                    permissionString = RolesPermissionsAdminLogic.RBAC_MANAGE_PERMISSION,
                    description = "",
                ),
            ),
        )
        assertTrue(
            RolesPermissionsAdminLogic.blocksSelfElevation(role, "me", "me"),
        )
        assertFalse(
            RolesPermissionsAdminLogic.blocksSelfElevation(role, "other", "me"),
        )
    }

    @Test
    fun addRoleUserRequestPayload() {
        assertEquals("user-123", RolesPermissionsAdminLogic.addRoleUserRequest("user-123").userId)
    }

    @Test
    fun userDisplayLabelPrefersName() {
        val user = RbacUserBrief(id = "1", email = "a@example.com", displayName = "Alex Admin")
        assertEquals("Alex Admin", RolesPermissionsAdminLogic.userDisplayLabel(user))
    }
}
