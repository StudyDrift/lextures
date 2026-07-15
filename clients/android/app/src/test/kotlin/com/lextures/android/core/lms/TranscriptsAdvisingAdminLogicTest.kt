package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class TranscriptsAdvisingAdminLogicTest {
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

    @Test
    fun entryRequiresFlagPermissionAndFeature() {
        val offAdmin = MobilePlatformFeatures(ffMobileAdminSettings = false, ffTranscripts = true)
        assertFalse(
            TranscriptsAdvisingAdminLogic.shouldShowEntry(
                offAdmin,
                listOf(TranscriptsAdvisingAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )

        var on = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertFalse(
            TranscriptsAdvisingAdminLogic.shouldShowEntry(
                on,
                listOf(TranscriptsAdvisingAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )

        on = on.copy(ffTranscripts = true)
        assertFalse(TranscriptsAdvisingAdminLogic.shouldShowEntry(on, emptyList()))
        assertTrue(
            TranscriptsAdvisingAdminLogic.shouldShowEntry(
                on,
                listOf(TranscriptsAdvisingAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun sectionVisibilityByFlags() {
        var features = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertTrue(TranscriptsAdvisingAdminLogic.visibleSections(features).isEmpty())

        features = features.copy(ffTranscripts = true)
        assertEquals(
            listOf(TranscriptsAdvisingAdminLogic.Section.TRANSCRIPTS),
            TranscriptsAdvisingAdminLogic.visibleSections(features),
        )

        features = features.copy(ffAdvisingIntegration = true)
        assertEquals(
            TranscriptsAdvisingAdminLogic.Section.entries.toSet(),
            TranscriptsAdvisingAdminLogic.visibleSections(features).toSet(),
        )
    }

    @Test
    fun subViewGating() {
        val perms = listOf(TranscriptsAdvisingAdminLogic.RBAC_MANAGE_PERMISSION)
        val transcriptsOnly = MobilePlatformFeatures(
            ffMobileAdminSettings = true,
            ffTranscripts = true,
            ffAdvisingIntegration = false,
        )
        assertTrue(TranscriptsAdvisingAdminLogic.canViewTranscripts(transcriptsOnly, perms))
        assertFalse(TranscriptsAdvisingAdminLogic.canViewAdvising(transcriptsOnly, perms))

        val advisingOnly = MobilePlatformFeatures(
            ffMobileAdminSettings = true,
            ffTranscripts = false,
            ffAdvisingIntegration = true,
        )
        assertFalse(TranscriptsAdvisingAdminLogic.canViewTranscripts(advisingOnly, perms))
        assertTrue(TranscriptsAdvisingAdminLogic.canViewAdvising(advisingOnly, perms))
    }

    @Test
    fun transcriptsSavePayloadOmitsPlaceholderSecret() {
        val keep = TranscriptsAdvisingAdminLogic.buildTranscriptsSaveRequest(
            webhookUrl = " https://sis.example.edu/hook ",
            webhookSecret = TranscriptsAdvisingAdminLogic.SECRET_PLACEHOLDER,
            pickupInstructions = " Room 101 ",
        )
        assertEquals("https://sis.example.edu/hook", keep.webhookUrl)
        assertNull(keep.webhookSecret)
        assertEquals("Room 101", keep.pickupInstructions)

        val update = TranscriptsAdvisingAdminLogic.buildTranscriptsSaveRequest(
            webhookUrl = "https://sis.example.edu/hook",
            webhookSecret = " new-secret ",
            pickupInstructions = "",
        )
        assertEquals("new-secret", update.webhookSecret)

        val encoded = json.encodeToJsonElement(PutAdminTranscriptsConfigRequest.serializer(), keep).jsonObject
        assertFalse(encoded.containsKey("webhookSecret") && encoded["webhookSecret"]?.jsonPrimitive?.isString == true && encoded["webhookSecret"]?.toString()?.contains("••••") == true)
        assertEquals("https://sis.example.edu/hook", encoded["webhookUrl"]?.jsonPrimitive?.content)
    }

    @Test
    fun webhookUrlValidation() {
        assertTrue(TranscriptsAdvisingAdminLogic.isValidHttpUrl("https://example.edu/hook"))
        assertTrue(TranscriptsAdvisingAdminLogic.isValidHttpUrl("http://localhost:8080/hook"))
        assertFalse(TranscriptsAdvisingAdminLogic.isValidHttpUrl(""))
        assertFalse(TranscriptsAdvisingAdminLogic.isValidHttpUrl("ftp://example.edu"))
        assertFalse(TranscriptsAdvisingAdminLogic.isValidHttpUrl("not-a-url"))
        assertTrue(TranscriptsAdvisingAdminLogic.isTranscriptsSaveDisabled(false, ""))
        assertFalse(TranscriptsAdvisingAdminLogic.isTranscriptsSaveDisabled(false, "https://example.edu"))
    }

    @Test
    fun advisingSavePayload() {
        val none = TranscriptsAdvisingAdminLogic.buildAdvisingSaveRequest(
            appointmentUrl = " https://navigate.example.edu ",
            provider = TranscriptsAdvisingAdminLogic.DegreeAuditProvider.NONE,
            baseUrl = "https://should-clear.example",
            credentialsRef = "secret-ref",
            atRiskBannerEnabled = true,
        )
        assertEquals("https://navigate.example.edu", none.appointmentUrl)
        assertEquals("none", none.degreeAuditProvider)
        assertEquals("", none.degreeAuditBaseUrl)
        assertEquals("", none.apiCredentialsRef)
        assertFalse(none.atRiskBannerEnabled)

        val full = TranscriptsAdvisingAdminLogic.buildAdvisingSaveRequest(
            appointmentUrl = "",
            provider = TranscriptsAdvisingAdminLogic.DegreeAuditProvider.DEGREEWORKS,
            baseUrl = " https://degreeworks.example.edu/api ",
            credentialsRef = " cred-1 ",
            atRiskBannerEnabled = true,
        )
        assertEquals("degreeworks", full.degreeAuditProvider)
        assertEquals("https://degreeworks.example.edu/api", full.degreeAuditBaseUrl)
        assertEquals("cred-1", full.apiCredentialsRef)
        assertTrue(full.atRiskBannerEnabled)
    }

    @Test
    fun decodeModels() {
        val cfg = json.decodeFromString<AdminTranscriptsConfig>(
            """{"webhookUrl":"https://sis.example.edu/hook","webhookSecret":"••••••••••••","hasWebhookSecret":true,"pickupInstructions":"Room 101"}""",
        )
        assertEquals("https://sis.example.edu/hook", cfg.webhookUrl)
        assertTrue(cfg.hasWebhookSecret)
        assertEquals(
            TranscriptsAdvisingAdminLogic.SECRET_PLACEHOLDER,
            TranscriptsAdvisingAdminLogic.webhookSecretField(cfg),
        )

        val advising = json.decodeFromString<AdminAdvisingConfig>(
            """{"appointmentUrl":"https://navigate.example.edu","degreeAuditProvider":"stellic","degreeAuditBaseUrl":"https://stellic.example.edu","apiCredentialsRef":"ref-1","atRiskBannerEnabled":true}""",
        )
        assertEquals(
            TranscriptsAdvisingAdminLogic.DegreeAuditProvider.STELLIC,
            TranscriptsAdvisingAdminLogic.DegreeAuditProvider.normalized(advising.degreeAuditProvider),
        )
        assertTrue(advising.atRiskBannerEnabled)
    }

    @Test
    fun webPaths() {
        assertEquals("/settings/transcripts", TranscriptsAdvisingAdminLogic.Section.TRANSCRIPTS.webPath)
        assertEquals("/settings/advising", TranscriptsAdvisingAdminLogic.Section.ADVISING.webPath)
        assertEquals("/settings/transcripts", TranscriptsAdvisingAdminLogic.webHubPath())
    }
}
