package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import java.text.NumberFormat
import java.time.Instant
import java.time.temporal.ChronoUnit
import java.util.Locale

/** AI models, system prompts, and usage reports admin helpers (M14.7). */
object AiModelsAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"
    const val PLATFORM_SECRET_PLACEHOLDER = "••••••••••••"

    enum class ReportPreset(val id: String, val hours: Long) {
        HOURS_24("24h", 24),
        DAYS_7("7d", 7 * 24),
        DAYS_30("30d", 30 * 24),
        DAYS_90("90d", 90 * 24),
    }

    val FALLBACK_TEXT_MODELS = listOf(
        AiModelOption(id = "google/gemini-2.0-flash-001", name = "Gemini 2.0 Flash"),
        AiModelOption(id = "google/gemini-2.5-flash", name = "Gemini 2.5 Flash"),
        AiModelOption(id = "openai/gpt-4o-mini", name = "GPT-4o mini"),
        AiModelOption(id = "anthropic/claude-3.5-sonnet", name = "Claude 3.5 Sonnet"),
        AiModelOption(id = "meta-llama/llama-3.3-70b-instruct", name = "Llama 3.3 70B Instruct"),
    )

    val FALLBACK_IMAGE_MODELS = listOf(
        AiModelOption(id = "google/gemini-2.5-flash-image", name = "Gemini 2.5 Flash (image)"),
        AiModelOption(id = "google/gemini-3.1-flash-image-preview", name = "Gemini 3.1 Flash Image (preview)"),
        AiModelOption(id = "black-forest-labs/flux.2-pro", name = "FLUX.2 Pro"),
        AiModelOption(id = "black-forest-labs/flux.2-flex", name = "FLUX.2 Flex"),
        AiModelOption(id = "sourceful/riverflow-v2-fast", name = "Riverflow v2 Fast"),
        AiModelOption(id = "sourceful/riverflow-v2-pro", name = "Riverflow v2 Pro"),
    )

    private val featureLabels = mapOf(
        "ai_tutor" to "AI Tutor",
        "rag_notebook" to "Notebook AI",
        "syllabus_generation" to "Syllabus generation",
        "translation" to "Translation",
        "quiz_generation" to "Quiz generation",
        "reading_level_simplification" to "Reading level",
        "content_translation" to "Content translation",
        "alt_text_suggestion" to "Alt text",
        "vibe_generation" to "Vibe activities",
        "unknown" to "Unknown",
    )

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings || features.ffMobileAdminConsole

    fun canManage(permissions: Collection<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION)

    fun shouldShowEntry(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings && canManage(permissions)

    fun canView(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        adminSettingsEnabled(features) && canManage(permissions)

    /**
     * Builds the PUT body. The OpenRouter key is write-only: only send a new key when the
     * draft differs from the masked baseline, and only clear when emptied from a configured placeholder.
     */
    fun buildAiSettingsSaveRequest(
        imageModelId: String,
        courseSetupModelId: String,
        notebookFlashcardsModelId: String,
        vibeActivityModelId: String,
        graderAgentModelId: String,
        openRouterApiKey: String,
        openRouterApiKeyBaseline: String,
    ): PutAiSettingsRequest {
        val keyTrimmed = openRouterApiKey.trim()
        val baselineTrimmed = openRouterApiKeyBaseline.trim()
        var openRouter: String? = null
        var clear: Boolean? = null
        if (keyTrimmed != baselineTrimmed) {
            if (keyTrimmed.isNotEmpty() && keyTrimmed != PLATFORM_SECRET_PLACEHOLDER) {
                openRouter = keyTrimmed
            }
            if (baselineTrimmed == PLATFORM_SECRET_PLACEHOLDER &&
                keyTrimmed.isEmpty() &&
                openRouterApiKey != openRouterApiKeyBaseline
            ) {
                clear = true
            }
        }
        return PutAiSettingsRequest(
            imageModelId = imageModelId.trim(),
            courseSetupModelId = courseSetupModelId.trim(),
            notebookFlashcardsModelId = notebookFlashcardsModelId.trim(),
            vibeActivityModelId = vibeActivityModelId.trim(),
            graderAgentModelId = graderAgentModelId.trim(),
            openRouterApiKey = openRouter,
            clearOpenRouterApiKey = clear,
        )
    }

    fun isSaveDisabled(
        saving: Boolean,
        imageModelId: String,
        courseSetupModelId: String,
        notebookFlashcardsModelId: String,
        vibeActivityModelId: String,
    ): Boolean =
        saving ||
            imageModelId.trim().isEmpty() ||
            courseSetupModelId.trim().isEmpty() ||
            notebookFlashcardsModelId.trim().isEmpty() ||
            vibeActivityModelId.trim().isEmpty()

    fun modelsWithSelection(models: List<AiModelOption>, selectedId: String): List<AiModelOption> {
        val trimmed = selectedId.trim()
        if (trimmed.isEmpty() || models.any { it.id == trimmed }) return models
        return listOf(AiModelOption(id = trimmed, name = trimmed)) + models
    }

    fun modelDisplayLabel(model: AiModelOption): String {
        val name = model.name?.trim().orEmpty().ifEmpty { model.id }
        val parts = mutableListOf(name)
        if (name != model.id) parts += model.id
        model.modalitiesSummary?.trim()?.takeIf { it.isNotEmpty() }?.let { parts += it }
        model.contextLength?.let { parts += "ctx ${formatCount(it)}" }
        if (model.inputPricePerMillionUsd != null || model.outputPricePerMillionUsd != null) {
            val inStr = formatUsd(model.inputPricePerMillionUsd ?: 0.0)
            val outStr = formatUsd(model.outputPricePerMillionUsd ?: 0.0)
            parts += "$inStr/$outStr per 1M"
        }
        return parts.joinToString(" · ")
    }

    fun shouldSendOpenRouterKey(value: String): Boolean {
        val trimmed = value.trim()
        return trimmed.isNotEmpty() && trimmed != PLATFORM_SECRET_PLACEHOLDER
    }

    fun shouldClearOpenRouterKey(draft: String, baseline: String): Boolean {
        val keyTrimmed = draft.trim()
        val baselineTrimmed = baseline.trim()
        return baselineTrimmed == PLATFORM_SECRET_PLACEHOLDER &&
            keyTrimmed.isEmpty() &&
            draft != baseline
    }

    fun utcRange(preset: ReportPreset, now: Instant = Instant.now()): Pair<String, String> {
        val to = now
        val from = now.minus(preset.hours, ChronoUnit.HOURS)
        return from.toString() to to.toString()
    }

    fun featureLabel(feature: String): String =
        featureLabels[feature] ?: feature.replace('_', ' ')

    fun formatUsd(value: Double): String {
        if (!value.isFinite() || value == 0.0) return "$0.00"
        return if (value < 0.01) {
            String.format(Locale.US, "$%.4f", value)
        } else {
            String.format(Locale.US, "$%.2f", value)
        }
    }

    fun formatCount(value: Long): String =
        NumberFormat.getNumberInstance(Locale.getDefault()).format(value)

    fun promptContentChanged(original: String, draft: String): Boolean = original != draft
}
