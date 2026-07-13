package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class OrgBrandingAdminLogicTest {
    @Test
    fun adminSettingsFlag() {
        val off = MobilePlatformFeatures()
        val on = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertFalse(OrgBrandingAdminLogic.adminSettingsEnabled(off))
        assertTrue(OrgBrandingAdminLogic.adminSettingsEnabled(on))
    }

    @Test
    fun permissions() {
        assertFalse(OrgBrandingAdminLogic.canManageOrgBranding(emptyList()))
        assertTrue(
            OrgBrandingAdminLogic.canManageOrgBranding(
                listOf(OrgBrandingAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
        assertTrue(
            OrgBrandingAdminLogic.canManageOrgBranding(
                listOf(OrgBrandingAdminLogic.ORG_UNITS_ADMIN_PERMISSION),
            ),
        )
    }

    @Test
    fun shouldShowEntry() {
        val features = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertFalse(OrgBrandingAdminLogic.shouldShowEntry(features, emptyList()))
        assertTrue(
            OrgBrandingAdminLogic.shouldShowEntry(
                features,
                listOf(OrgBrandingAdminLogic.ORG_UNITS_ADMIN_PERMISSION),
            ),
        )
    }

    @Test
    fun hexColorValidation() {
        assertTrue(OrgBrandingAdminLogic.isValidHexColor("#4F46E5"))
        assertEquals("#4F46E5", OrgBrandingAdminLogic.normalizedHexColor("#4f46e5"))
        assertFalse(OrgBrandingAdminLogic.isValidHexColor("blue"))
        assertFalse(OrgBrandingAdminLogic.isValidHexColor("#FFF"))
    }

    @Test
    fun contrastWarning() {
        assertTrue(
            OrgBrandingAdminLogic.showsContrastWarning(
                primaryColor = "#FFFF00",
                serverWarning = false,
                serverRatio = null,
            ),
        )
        assertFalse(
            OrgBrandingAdminLogic.showsContrastWarning(
                primaryColor = "#111827",
                serverWarning = false,
                serverRatio = null,
            ),
        )
    }

    @Test
    fun secretPlaceholderHandling() {
        assertFalse(OrgBrandingAdminLogic.shouldSendByokKey(OrgBrandingAdminLogic.PLATFORM_SECRET_PLACEHOLDER))
        assertTrue(OrgBrandingAdminLogic.shouldSendByokKey("sk-live-secret"))
        val placeholderRequest = OrgBrandingAdminLogic.buildAiProviderSaveRequest(
            provider = "openrouter",
            modelAlias = "claude-3-5-sonnet",
            fallbackProvider = "",
            byokKey = OrgBrandingAdminLogic.PLATFORM_SECRET_PLACEHOLDER,
        )
        assertNull(placeholderRequest.byokApiKey)
        val withKey = OrgBrandingAdminLogic.buildAiProviderSaveRequest(
            provider = "openrouter",
            modelAlias = "claude-3-5-sonnet",
            fallbackProvider = "",
            byokKey = "sk-live-secret",
        )
        assertEquals("sk-live-secret", withKey.byokApiKey)
    }

    @Test
    fun parseAllowedModels() {
        assertEquals(
            listOf("gpt-4o", "claude-3-5-sonnet"),
            OrgBrandingAdminLogic.parseAllowedModels("gpt-4o\nclaude-3-5-sonnet"),
        )
        assertEquals(
            listOf("gpt-4o", "claude-3-5-sonnet"),
            OrgBrandingAdminLogic.parseAllowedModels("gpt-4o, claude-3-5-sonnet"),
        )
    }
}
