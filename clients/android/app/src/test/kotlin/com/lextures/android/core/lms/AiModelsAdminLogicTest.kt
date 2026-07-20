package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant

class AiModelsAdminLogicTest {
    @Test
    fun entryRequiresFlagAndRbacPermission() {
        val off = MobilePlatformFeatures(ffMobileAdminConsole = false, ffMobileAdminSettings = false)
        assertFalse(
            AiModelsAdminLogic.shouldShowEntry(
                off,
                listOf(AiModelsAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
        val on = MobilePlatformFeatures(ffMobileAdminConsole = false, ffMobileAdminSettings = true)
        assertFalse(AiModelsAdminLogic.shouldShowEntry(on, emptyList()))
        assertTrue(
            AiModelsAdminLogic.shouldShowEntry(
                on,
                listOf(AiModelsAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun buildSaveRequestOmitsUnchangedPlaceholderKey() {
        val request = AiModelsAdminLogic.buildAiSettingsSaveRequest(
            imageModelId = "img-1",
            courseSetupModelId = "text-1",
            notebookFlashcardsModelId = "text-2",
            vibeActivityModelId = "text-3",
            graderAgentModelId = "text-4",
            openRouterApiKey = AiModelsAdminLogic.PLATFORM_SECRET_PLACEHOLDER,
            openRouterApiKeyBaseline = AiModelsAdminLogic.PLATFORM_SECRET_PLACEHOLDER,
        )
        assertNull(request.openRouterApiKey)
        assertNull(request.clearOpenRouterApiKey)
        assertEquals("img-1", request.imageModelId)
    }

    @Test
    fun buildSaveRequestSendsNewKey() {
        val request = AiModelsAdminLogic.buildAiSettingsSaveRequest(
            imageModelId = "img-1",
            courseSetupModelId = "text-1",
            notebookFlashcardsModelId = "text-2",
            vibeActivityModelId = "text-3",
            graderAgentModelId = "text-4",
            openRouterApiKey = "sk-new-secret",
            openRouterApiKeyBaseline = AiModelsAdminLogic.PLATFORM_SECRET_PLACEHOLDER,
        )
        assertEquals("sk-new-secret", request.openRouterApiKey)
        assertNull(request.clearOpenRouterApiKey)
        assertTrue(AiModelsAdminLogic.shouldSendOpenRouterKey("sk-new-secret"))
        assertFalse(AiModelsAdminLogic.shouldSendOpenRouterKey(AiModelsAdminLogic.PLATFORM_SECRET_PLACEHOLDER))
    }

    @Test
    fun buildSaveRequestClearsKeyWhenEmptiedFromPlaceholder() {
        val request = AiModelsAdminLogic.buildAiSettingsSaveRequest(
            imageModelId = "img-1",
            courseSetupModelId = "text-1",
            notebookFlashcardsModelId = "text-2",
            vibeActivityModelId = "text-3",
            graderAgentModelId = "text-4",
            openRouterApiKey = "",
            openRouterApiKeyBaseline = AiModelsAdminLogic.PLATFORM_SECRET_PLACEHOLDER,
        )
        assertNull(request.openRouterApiKey)
        assertEquals(true, request.clearOpenRouterApiKey)
        assertTrue(
            AiModelsAdminLogic.shouldClearOpenRouterKey(
                draft = "",
                baseline = AiModelsAdminLogic.PLATFORM_SECRET_PLACEHOLDER,
            ),
        )
    }

    @Test
    fun saveDisabledWhenRequiredModelsMissing() {
        assertTrue(
            AiModelsAdminLogic.isSaveDisabled(
                saving = false,
                imageModelId = "",
                courseSetupModelId = "a",
                notebookFlashcardsModelId = "b",
                vibeActivityModelId = "c",
            ),
        )
        assertFalse(
            AiModelsAdminLogic.isSaveDisabled(
                saving = false,
                imageModelId = "img",
                courseSetupModelId = "a",
                notebookFlashcardsModelId = "b",
                vibeActivityModelId = "c",
            ),
        )
    }

    @Test
    fun modelsWithSelectionInjectsMissingId() {
        val models = listOf(AiModelOption(id = "a", name = "A"))
        val merged = AiModelsAdminLogic.modelsWithSelection(models, "legacy")
        assertEquals("legacy", merged.first().id)
        assertEquals(2, merged.size)
        assertEquals(1, AiModelsAdminLogic.modelsWithSelection(models, "a").size)
    }

    @Test
    fun reportRangeAndFormatting() {
        val now = Instant.parse("2024-01-02T00:00:00Z")
        val range = AiModelsAdminLogic.utcRange(AiModelsAdminLogic.ReportPreset.HOURS_24, now)
        assertTrue(range.first.isNotEmpty())
        assertTrue(range.second.isNotEmpty())
        assertEquals("$0.00", AiModelsAdminLogic.formatUsd(0.0))
        assertEquals("$0.0012", AiModelsAdminLogic.formatUsd(0.0012))
        assertEquals("$1.50", AiModelsAdminLogic.formatUsd(1.5))
        assertEquals("AI Tutor", AiModelsAdminLogic.featureLabel("ai_tutor"))
        assertEquals("custom thing", AiModelsAdminLogic.featureLabel("custom_thing"))
        assertTrue(AiModelsAdminLogic.promptContentChanged("a", "b"))
        assertFalse(AiModelsAdminLogic.promptContentChanged("a", "a"))
    }

    @Test
    fun decodeAiSettingsAndReports() {
        val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }
        val settings = json.decodeFromString<AiSettingsResponse>(
            """{"imageModelId":"img","courseSetupModelId":"cs","notebookFlashcardsModelId":"fc",
                "vibeActivityModelId":"va","graderAgentModelId":"ga","openRouterApiKey":"••••••••••••"}""",
        )
        assertEquals("img", settings.imageModelId)
        assertEquals(AiModelsAdminLogic.PLATFORM_SECRET_PLACEHOLDER, settings.openRouterApiKey)

        val report = json.decodeFromString<AiReportsPayload>(
            """{"range":{"from":"2024-01-01T00:00:00Z","to":"2024-01-02T00:00:00Z"},
               "cost":{"summary":{"totalCostUsd":1.25,"totalCalls":3,"totalTokens":100},
               "byDay":[{"day":"2024-01-01","costUsd":1.25,"calls":3,"tokens":100}],
               "byFeature":[{"feature":"ai_tutor","costUsd":1.25,"calls":3,"tokens":100}]},
               "byUser":[{"userId":"u1","email":"a@b.c","displayName":"Ada","calls":1,
               "promptTokens":1,"completionTokens":1,"totalTokens":2,"costUsd":0.5}],
               "byCourse":[{"courseId":"c1","courseCode":"CS101","title":"Intro","calls":1,
               "totalTokens":2,"costUsd":0.5}]}""",
        )
        assertEquals(3, report.cost.summary.totalCalls)
        assertEquals("Ada", report.byUser.first().displayName)
        assertEquals("CS101", report.byCourse.first().courseCode)
    }
}
