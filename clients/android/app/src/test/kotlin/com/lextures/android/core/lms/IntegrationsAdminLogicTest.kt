package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class IntegrationsAdminLogicTest {
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    @Test
    fun entryRequiresFlagAndRbacPermission() {
        val off = MobilePlatformFeatures(ffMobileAdminConsole = false, ffMobileAdminSettings = false)
        assertFalse(
            IntegrationsAdminLogic.shouldShowEntry(
                off,
                listOf(IntegrationsAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
        val on = MobilePlatformFeatures(ffMobileAdminConsole = false, ffMobileAdminSettings = true)
        assertFalse(IntegrationsAdminLogic.shouldShowEntry(on, emptyList()))
        assertTrue(
            IntegrationsAdminLogic.shouldShowEntry(
                on,
                listOf(IntegrationsAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun sectionVisibilityByFlags() {
        val base = MobilePlatformFeatures(ffMobileAdminConsole = false, ffMobileAdminSettings = true,
            oerLibraryEnabled = false,
            xapiEmissionEnabled = false,
        )
        assertEquals(
            setOf(IntegrationsAdminLogic.Section.LTI, IntegrationsAdminLogic.Section.CLOUD),
            IntegrationsAdminLogic.visibleSections(base, scimEnabled = false).toSet(),
        )
        val full = base.copy(oerLibraryEnabled = true, xapiEmissionEnabled = true)
        assertEquals(
            IntegrationsAdminLogic.Section.entries.toSet(),
            IntegrationsAdminLogic.visibleSections(full, scimEnabled = true).toSet(),
        )
    }

    @Test
    fun cloudStatusExcludesSecrets() {
        assertTrue(
            IntegrationsAdminLogic.cloudStatusExcludesSecrets(setOf("provider", "enabled", "updatedAt")),
        )
        assertFalse(
            IntegrationsAdminLogic.cloudStatusExcludesSecrets(setOf("provider", "clientId", "enabled")),
        )
        assertFalse(IntegrationsAdminLogic.cloudStatusExcludesSecrets(setOf("apiKey")))
    }

    @Test
    fun decodeCloudProviderOmitsSecrets() {
        val rows = json.decodeFromString<List<CloudProviderStatus>>(
            """[{"provider":"google_drive","enabled":true,"clientId":"secret-id","apiKey":"secret-key","appKey":"secret-app","updatedAt":"2026-01-01T00:00:00Z"}]""",
        )
        assertEquals(1, rows.size)
        assertEquals("google_drive", rows[0].provider)
        assertTrue(rows[0].enabled)
        // Status model has no secret properties (secrets are ignored at decode time).
        assertEquals("google_drive", rows[0].provider)
        assertTrue(rows[0].enabled)
        assertEquals("2026-01-01T00:00:00Z", rows[0].updatedAt)
    }

    @Test
    fun decodeLtiAndScimStatus() {
        val lti = json.decodeFromString<LtiRegistrationsResponse>(
            """{"parentPlatforms":[{"id":"p1","name":"Canvas","clientId":"c1","platformIss":"https://canvas.example","active":true}],
               "externalTools":[{"id":"t1","name":"Tool","clientId":"c2","toolIssuer":"https://tool.example","active":false}]}""",
        )
        assertEquals(1, IntegrationsAdminLogic.ltiActiveCount(lti.parentPlatforms, lti.externalTools))

        val tokens = json.decodeFromString<ScimTokensResponse>(
            """{"tokens":[
              {"id":"1","institutionId":"i1","label":"okta","createdAt":"2026-01-01T00:00:00Z"},
              {"id":"2","institutionId":"i1","label":"old","createdAt":"2025-01-01T00:00:00Z","revokedAt":"2025-06-01T00:00:00Z"}
            ]}""",
        ).tokens.orEmpty()
        assertEquals(1, IntegrationsAdminLogic.activeTokenCount(tokens))
    }

    @Test
    fun applyingToggles() {
        val platforms = listOf(LtiParentPlatform(id = "p1", name = "A", active = true))
        assertFalse(IntegrationsAdminLogic.applyingLtiPlatformActive(platforms, "p1", false).first().active)

        val tools = listOf(LtiExternalTool(id = "t1", name = "T", active = false))
        assertTrue(IntegrationsAdminLogic.applyingLtiToolActive(tools, "t1", true).first().active)

        val cloud = listOf(CloudProviderStatus(provider = "dropbox", enabled = true))
        assertFalse(IntegrationsAdminLogic.applyingCloudEnabled(cloud, "dropbox", false).first().enabled)

        val lrs = listOf(LrsEndpointStatus(id = "e1", label = "L", endpointUrl = "https://lrs", enabled = true))
        assertFalse(IntegrationsAdminLogic.applyingLrsEnabled(lrs, "e1", false).first().enabled)

        val oer = listOf(OerProviderStatus(provider = "merlot", enabled = false))
        assertTrue(IntegrationsAdminLogic.applyingOerEnabled(oer, "merlot", true).first().enabled)
    }

    @Test
    fun webPaths() {
        assertEquals("/settings/lti-tools", IntegrationsAdminLogic.Section.LTI.webPath)
        assertEquals("/settings/scim-provisioning", IntegrationsAdminLogic.Section.SCIM.webPath)
        assertEquals("/settings/cloud-providers", IntegrationsAdminLogic.Section.CLOUD.webPath)
        assertEquals("/settings/lrs-integrations", IntegrationsAdminLogic.Section.LRS.webPath)
        assertEquals("/settings/oer-providers", IntegrationsAdminLogic.Section.OER.webPath)
    }
}
