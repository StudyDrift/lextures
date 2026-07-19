package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AuditLogAdminLogicTest {
    @Test
    fun gateRequiresConsoleFlagsAndRbac() {
        var features = MobilePlatformFeatures(
            ffMobileAdminConsole = true,
            adminConsoleEnabled = true,
            adminAuditLogEnabled = true,
        )
        assertFalse(AuditLogAdminLogic.shouldShowInMenu(features, emptyList()))
        assertTrue(
            AuditLogAdminLogic.shouldShowInMenu(features, listOf(AuditLogAdminLogic.RBAC_MANAGE_PERMISSION)),
        )
        features = features.copy(adminAuditLogEnabled = false)
        assertFalse(
            AuditLogAdminLogic.shouldShowInMenu(features, listOf(AuditLogAdminLogic.RBAC_MANAGE_PERMISSION)),
        )
    }

    @Test
    fun normalizedActionFilter() {
        assertNull(AuditLogAdminLogic.normalizedActionFilter("  "))
        assertEquals("user_deactivate", AuditLogAdminLogic.normalizedActionFilter(" user_deactivate "))
    }

    @Test
    fun targetLabel() {
        assertEquals("—", AuditLogAdminLogic.targetLabel(null, null))
        assertEquals("user", AuditLogAdminLogic.targetLabel("user", null))
        assertEquals("abc", AuditLogAdminLogic.targetLabel(null, "abc"))
        assertEquals("user / abc", AuditLogAdminLogic.targetLabel("user", "abc"))
    }

    @Test
    fun webPath() {
        assertEquals("/org-admin/audit-log", AuditLogAdminLogic.webPath())
    }
}
