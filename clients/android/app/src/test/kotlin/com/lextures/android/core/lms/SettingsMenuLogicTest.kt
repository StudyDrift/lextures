package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class SettingsMenuLogicTest {
    private val rbac = SettingsMenuLogic.RBAC_MANAGE_PERMISSION

    @Test
    fun hubHiddenWhenFlagOff() {
        val features = MobilePlatformFeatures(ffMobileAdminConsole = false, adminConsoleEnabled = true)
        assertFalse(SettingsMenuLogic.shouldShowHubEntry(features, listOf(rbac)))
    }

    @Test
    fun hubRequiresRbac() {
        val features = MobilePlatformFeatures(
            ffMobileAdminConsole = true,
            adminConsoleEnabled = true,
            adminAuditLogEnabled = true,
        )
        assertFalse(SettingsMenuLogic.shouldShowHubEntry(features, emptyList()))
        assertTrue(SettingsMenuLogic.shouldShowHubEntry(features, listOf(rbac)))
    }

    @Test
    fun visibleGroupsHideAdminItemsWithoutRbac() {
        val features = MobilePlatformFeatures(
            ffMobileAdminConsole = true,
            ffMobileAdminSettings = true,
            adminConsoleEnabled = true,
            adminAuditLogEnabled = true,
        )
        assertTrue(SettingsMenuLogic.visibleGroups(features, emptyList()).isEmpty())
    }

    @Test
    fun visibleGroupsIncludeShippedAndAuditLog() {
        val features = MobilePlatformFeatures(
            ffMobileAdminConsole = true,
            ffMobileAdminSettings = true,
            adminConsoleEnabled = true,
            adminAuditLogEnabled = true,
        )
        val groups = SettingsMenuLogic.visibleGroups(features, listOf(rbac))
        val ids = groups.flatMap { it.items.map { item -> item.id } }.toSet()
        assertTrue(SettingsMenuLogic.ItemId.PlatformSettings in ids)
        assertTrue(SettingsMenuLogic.ItemId.People in ids)
        assertTrue(SettingsMenuLogic.ItemId.AuditLog in ids)
        assertTrue(groups.any { it.id == SettingsMenuLogic.GroupId.Platform })
        assertTrue(groups.any { it.id == SettingsMenuLogic.GroupId.Compliance })
    }

    @Test
    fun auditLogHiddenWhenAdminConsoleFlagOff() {
        val features = MobilePlatformFeatures(
            ffMobileAdminConsole = true,
            adminConsoleEnabled = false,
            adminAuditLogEnabled = true,
        )
        assertFalse(SettingsMenuLogic.isItemVisible(SettingsMenuLogic.ItemId.AuditLog, features, listOf(rbac)))
    }

    @Test
    fun phase1RegistryGroupsMatchInventory() {
        val labels = SettingsMenuLogic.phase1Registry.toMap()
        assertTrue(SettingsMenuLogic.ItemId.PlatformSettings in labels.getValue(SettingsMenuLogic.GroupId.Platform))
        assertTrue(SettingsMenuLogic.ItemId.BoardsGovernance in labels.getValue(SettingsMenuLogic.GroupId.Platform))
        assertTrue(SettingsMenuLogic.ItemId.AuditLog in labels.getValue(SettingsMenuLogic.GroupId.Compliance))
        assertTrue(SettingsMenuLogic.ItemId.Integrations in labels.getValue(SettingsMenuLogic.GroupId.Integrations))
        assertTrue(
            SettingsMenuLogic.ItemId.TranscriptsAdvising in
                labels.getValue(SettingsMenuLogic.GroupId.StudentRecords),
        )
        assertTrue(labels.getValue(SettingsMenuLogic.GroupId.SchoolOperations).isEmpty())
    }

    @Test
    fun searchFiltersItems() {
        val features = MobilePlatformFeatures(
            ffMobileAdminConsole = true,
            ffMobileAdminSettings = true,
            adminConsoleEnabled = true,
            adminAuditLogEnabled = true,
        )
        val groups = SettingsMenuLogic.visibleGroups(features, listOf(rbac), query = "audit")
        assertEquals(listOf(SettingsMenuLogic.ItemId.AuditLog), groups.flatMap { it.items.map { i -> i.id } })
    }
}
