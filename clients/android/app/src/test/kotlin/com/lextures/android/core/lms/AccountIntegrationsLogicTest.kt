package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AccountIntegrationsLogicTest {
    @Test
    fun integrationsEnabledRequiresFlag() {
        val off = MobilePlatformFeatures()
        assertFalse(AccountIntegrationsLogic.integrationsEnabled(off))
        val on = MobilePlatformFeatures(ffMobileSettingsIntegrations = true)
        assertTrue(AccountIntegrationsLogic.integrationsEnabled(on))
    }

    @Test
    fun canManageServiceTokensRequiresRbacManage() {
        assertFalse(AccountIntegrationsLogic.canManageServiceTokens(emptyList()))
        assertTrue(AccountIntegrationsLogic.canManageServiceTokens(listOf(AccountIntegrationsLogic.RBAC_MANAGE_PERMISSION)))
    }

    @Test
    fun shouldHideServiceTokensForNonAdmin() {
        assertFalse(
            AccountIntegrationsLogic.shouldShowServiceTokensSection(
                permissions = emptyList(),
                adminApiForbidden = true,
            ),
        )
        assertTrue(
            AccountIntegrationsLogic.shouldShowServiceTokensSection(
                permissions = listOf(AccountIntegrationsLogic.RBAC_MANAGE_PERMISSION),
                adminApiForbidden = false,
            ),
        )
    }

    @Test
    fun resolveCalendarFeedURLSubstitutesToken() {
        val url = AccountIntegrationsLogic.resolveCalendarFeedURL(
            "https://example.com/feed?token=<token>",
            "abc+def",
        )
        assertFalse(url.contains("<token>"))
    }

    @Test
    fun activeAccessKeysExcludesRevokedAndServiceTokens() {
        val tokens = listOf(
            AccessKeySummary(
                id = "1",
                label = "Active",
                tokenMask = "ltk_***",
                scopes = listOf("mcp:connect"),
                isServiceToken = false,
                createdAt = "2026-01-01T00:00:00Z",
            ),
            AccessKeySummary(
                id = "2",
                label = "Revoked",
                tokenMask = "ltk_***",
                scopes = listOf("courses:read"),
                revokedAt = "2026-01-02T00:00:00Z",
                createdAt = "2026-01-01T00:00:00Z",
            ),
            AccessKeySummary(
                id = "3",
                label = "Service",
                tokenMask = "ltk_***",
                scopes = listOf("enrollments:read"),
                isServiceToken = true,
                createdAt = "2026-01-01T00:00:00Z",
            ),
        )
        assertEquals(listOf("1"), AccountIntegrationsLogic.activeAccessKeys(tokens).map { it.id })
    }
}