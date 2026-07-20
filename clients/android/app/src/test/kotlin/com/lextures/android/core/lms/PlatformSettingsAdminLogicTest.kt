package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.json.Json
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PlatformSettingsAdminLogicTest {
    @Test
    fun entryRequiresFlagAndRbacPermission() {
        // Legacy settings entry only shows when admin console is off and admin settings is on.
        val consoleOn = MobilePlatformFeatures(ffMobileAdminConsole = true, ffMobileAdminSettings = true)
        assertFalse(
            PlatformSettingsAdminLogic.shouldShowEntry(
                consoleOn,
                listOf(PlatformSettingsAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
        val legacy = MobilePlatformFeatures(ffMobileAdminConsole = false, ffMobileAdminSettings = true)
        assertFalse(PlatformSettingsAdminLogic.shouldShowEntry(legacy, emptyList()))
        assertTrue(
            PlatformSettingsAdminLogic.shouldShowEntry(
                legacy,
                listOf(PlatformSettingsAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun allowlistExcludesLockoutAndInfrastructureFlags() {
        val keys = PlatformSettingsAdminLogic.FEATURE_DEFINITIONS.map { it.key }.toSet()
        assertFalse("ffMobileAdminSettings" in keys)
        assertFalse("samlSsoEnabled" in keys)
        assertFalse("ffPaymentsEnabled" in keys)
        assertFalse("ffFeedback" in keys)
        assertTrue("ffPublicCatalog" in keys)
    }

    @Test
    fun secretFieldsAreIgnoredWhenDecodingSnapshot() {
        val snapshot = Json { ignoreUnknownKeys = true }.decodeFromString<PlatformSettingsSnapshot>(
            """{"ffFeedback":true,"ffPublicCatalog":false,"smtpPassword":"••••••••••••","samlSpPrivateKeyPem":"••••••••••••"}""",
        )
        assertTrue(snapshot.ffFeedback)
        assertFalse(PlatformSettingsAdminLogic.value("ffFeedback", snapshot))
        assertFalse(PlatformSettingsAdminLogic.value("ffPublicCatalog", snapshot))
        val merged = PlatformSettingsAdminLogic.applyingEffectiveFeatures(
            PlatformFeatureStates(ffPublicCatalog = true),
            snapshot,
        )
        assertTrue(merged.ffPublicCatalog)
    }
}
