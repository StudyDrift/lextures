package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class OrgBrandingAdminLogicTest {
    @Test
    fun adminSettingsEnabled() {
        val off = MobilePlatformFeatures()
        assertFalse(OrgBrandingAdminLogic.adminSettingsEnabled(off))
        val on = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertTrue(OrgBrandingAdminLogic.adminSettingsEnabled(on))
    }

    @Test
    fun canManage() {
        assertFalse(OrgBrandingAdminLogic.canManage(emptyList()))
        assertTrue(
            OrgBrandingAdminLogic.canManage(listOf(OrgBrandingAdminLogic.RBAC_MANAGE_PERMISSION)),
        )
        assertTrue(
            OrgBrandingAdminLogic.canManage(listOf(OrgBrandingAdminLogic.ORG_UNITS_ADMIN_PERMISSION)),
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
        assertTrue(OrgBrandingAdminLogic.isValidHexColor("#abc"))
        assertFalse(OrgBrandingAdminLogic.isValidHexColor("4F46E5"))
        assertFalse(OrgBrandingAdminLogic.isValidHexColor("#GG0000"))
        assertFalse(OrgBrandingAdminLogic.isValidHexColor(""))
    }

    @Test
    fun normalizeHexColor() {
        assertEquals("#AABBCC", OrgBrandingAdminLogic.normalizeHexColor("#abc", "#000000"))
        assertEquals("#4F46E5", OrgBrandingAdminLogic.normalizeHexColor("bad", "#4F46E5"))
    }

    @Test
    fun secretPlaceholderHandling() {
        assertNull(OrgBrandingAdminLogic.byokKeyForSave(""))
        assertNull(OrgBrandingAdminLogic.byokKeyForSave(OrgBrandingAdminLogic.SECRET_PLACEHOLDER))
        assertEquals("sk-live-xyz", OrgBrandingAdminLogic.byokKeyForSave(" sk-live-xyz "))
        assertEquals(
            OrgBrandingAdminLogic.SECRET_PLACEHOLDER,
            OrgBrandingAdminLogic.displaySecretField(true),
        )
        assertEquals("", OrgBrandingAdminLogic.displaySecretField(false))
        assertTrue(OrgBrandingAdminLogic.isSecretPlaceholder(OrgBrandingAdminLogic.SECRET_PLACEHOLDER))
    }

    @Test
    fun allowedModelsParsing() {
        assertNull(OrgBrandingAdminLogic.parseAllowedModels("  \n  "))
        assertEquals(
            listOf("claude-3-5-sonnet", "gpt-4o", "gemini-1.5-pro"),
            OrgBrandingAdminLogic.parseAllowedModels("claude-3-5-sonnet\ngpt-4o, gemini-1.5-pro"),
        )
        assertEquals("a\nb", OrgBrandingAdminLogic.allowedModelsText(listOf("a", "b")))
    }

    @Test
    fun featuresEnabledPayloadDefaultsTrue() {
        val payload = OrgBrandingAdminLogic.featuresEnabledPayload(emptyMap())
        for (item in OrgBrandingAdminLogic.FEATURE_KEYS) {
            assertEquals(true, payload[item.key])
        }
        val disabled = OrgBrandingAdminLogic.featuresEnabledPayload(mapOf("ai_tutor" to false))
        assertEquals(false, disabled["ai_tutor"])
        assertEquals(true, disabled["translation"])
    }

    @Test
    fun aiProviderPutBodyOmitsPlaceholderSecret() {
        val body = OrgBrandingAdminLogic.aiProviderPutBody(
            provider = "openai",
            modelAlias = "gpt-4o",
            fallbackProvider = "",
            byokKey = OrgBrandingAdminLogic.SECRET_PLACEHOLDER,
        )
        assertEquals("openai", body.provider)
        assertEquals("gpt-4o", body.modelAlias)
        assertNull(body.fallbackProvider)
        assertNull(body.byokApiKey)
    }

    @Test
    fun contrastAgainstWhite() {
        val white = OrgBrandingAdminLogic.contrastRatioAgainstWhite("#FFFFFF")
        assertTrue(white != null && kotlin.math.abs(white - 1.0) < 0.01)
        val black = OrgBrandingAdminLogic.contrastRatioAgainstWhite("#000000")
        assertTrue(black != null && black > 20)
        assertTrue(
            OrgBrandingAdminLogic.hasContrastWarning(
                primaryColor = "#EEEEEE",
                serverWarning = false,
                serverRatio = null,
            ),
        )
        assertFalse(
            OrgBrandingAdminLogic.hasContrastWarning(
                primaryColor = "#111111",
                serverWarning = false,
                serverRatio = 12.0,
            ),
        )
    }

    @Test
    fun brandingPutBodyTrimsEmail() {
        val body = OrgBrandingAdminLogic.brandingPutBody(
            logoUrl = null,
            faviconUrl = null,
            primaryColor = "#4f46e5",
            secondaryColor = "#7c3aed",
            customEmailDisplayName = "  District  ",
        )
        assertEquals("#4F46E5", body.primaryColor)
        assertEquals("District", body.customEmailDisplayName)
        assertNull(body.customDomain)
    }

    @Test
    fun webPath() {
        assertEquals("/settings/org-branding", OrgBrandingAdminLogic.webBrandingPath())
    }
}
