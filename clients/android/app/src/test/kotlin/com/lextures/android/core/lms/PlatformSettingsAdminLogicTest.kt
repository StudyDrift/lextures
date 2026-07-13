package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.json.Json
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PlatformSettingsAdminLogicTest {
    @Test
    fun entryRequiresFlagAndRbacPermission() {
        val off = MobilePlatformFeatures(ffMobileAdminSettings = false)
        assertFalse(PlatformSettingsAdminLogic.shouldShowEntry(off, listOf(PlatformSettingsAdminLogic.RBAC_MANAGE_PERMISSION)))
        val on = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertFalse(PlatformSettingsAdminLogic.shouldShowEntry(on, emptyList()))
        assertTrue(PlatformSettingsAdminLogic.shouldShowEntry(on, listOf(PlatformSettingsAdminLogic.RBAC_MANAGE_PERMISSION)))
    }

    @Test
    fun allowlistExcludesLockoutAndInfrastructureFlags() {
        val keys = PlatformSettingsAdminLogic.FEATURE_DEFINITIONS.map { it.key }.toSet()
        assertFalse("ffMobileAdminSettings" in keys)
        assertFalse("samlSsoEnabled" in keys)
        assertFalse("ffPaymentsEnabled" in keys)
        assertTrue("ffFeedback" in keys)
    }

    @Test
    fun secretFieldsAreIgnoredWhenDecodingSnapshot() {
        val snapshot = Json { ignoreUnknownKeys = true }.decodeFromString<PlatformSettingsSnapshot>(
            """{"ffFeedback":true,"smtpPassword":"••••••••••••","samlSpPrivateKeyPem":"••••••••••••"}""",
        )
        assertTrue(snapshot.ffFeedback)
        assertTrue(PlatformSettingsAdminLogic.value("ffFeedback", snapshot))
        val merged = PlatformSettingsAdminLogic.applyingEffectiveFeatures(
            PlatformFeatureStates(ffFeedback = false),
            snapshot,
        )
        assertFalse(merged.ffFeedback)
    }
}
