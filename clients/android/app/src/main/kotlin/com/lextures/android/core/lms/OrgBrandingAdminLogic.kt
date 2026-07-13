package com.lextures.android.core.lms

import androidx.compose.ui.graphics.Color
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError
import kotlin.math.pow

/** Org branding, AI governance, and AI provider admin helpers (M14.5). */
object OrgBrandingAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"
    const val ORG_UNITS_ADMIN_PERMISSION = "tenant:org:units:admin"
    const val PLATFORM_SECRET_PLACEHOLDER = "••••••••••••"
    const val DEFAULT_PRIMARY_COLOR = "#4F46E5"
    const val DEFAULT_SECONDARY_COLOR = "#7C3AED"
    const val WCAG_CONTRAST_MINIMUM = 4.5
    const val MAX_LOGO_UPLOAD_BYTES = 4 * 1024 * 1024

    data class AiFeatureKey(val key: String, val labelResSuffix: String)

    val AI_FEATURE_KEYS: List<AiFeatureKey> = listOf(
        AiFeatureKey("ai_tutor", "aiTutor"),
        AiFeatureKey("rag_notebook", "notebook"),
        AiFeatureKey("syllabus_generation", "syllabus"),
        AiFeatureKey("translation", "translation"),
        AiFeatureKey("quiz_generation", "quiz"),
        AiFeatureKey("lesson_generation", "lesson"),
    )

    val PROVIDER_LABELS: Map<String, String> = mapOf(
        "openrouter" to "OpenRouter",
        "anthropic" to "Anthropic",
        "openai" to "OpenAI",
        "azure_openai" to "Azure OpenAI",
        "bedrock" to "AWS Bedrock",
        "vertex" to "Google Vertex AI",
    )

    enum class SaveStatus {
        Idle,
        Saving,
        Saved,
        Error,
    }

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings

    fun canManageOrgBranding(permissions: List<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION) ||
            permissions.contains(ORG_UNITS_ADMIN_PERMISSION)

    fun shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean = adminSettingsEnabled(features) && canManageOrgBranding(permissions)

    fun canView(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean = shouldShowEntry(features, permissions)

    fun webOrgBrandingPath(): String = "/settings/org-branding"

    fun resolveOrgId(accessToken: String?, courses: List<CourseSummary>): String? =
        CourseCreateLogic.resolveOrgId(accessToken, courses)

    fun resolveBrandAssetUrl(pathOrUrl: String?): String? {
        val raw = pathOrUrl?.trim().orEmpty()
        if (raw.isEmpty()) return null
        if (raw.startsWith("http://") || raw.startsWith("https://")) return raw
        val path = if (raw.startsWith("/")) raw else "/$raw"
        return AppConfiguration.apiUrl(path).toString()
    }

    fun normalizedHexColor(value: String): String? {
        val trimmed = value.trim()
        if (!Regex("^#([0-9a-fA-F]{6})$").matches(trimmed)) return null
        return "#" + trimmed.drop(1).uppercase()
    }

    fun isValidHexColor(value: String): Boolean = normalizedHexColor(value) != null

    fun contrastRatioAgainstWhite(hex: String): Double? {
        val normalized = normalizedHexColor(hex) ?: return null
        val rgb = normalized.drop(1).toIntOrNull(16) ?: return null
        val r = ((rgb shr 16) and 0xFF) / 255.0
        val g = ((rgb shr 8) and 0xFF) / 255.0
        val b = (rgb and 0xFF) / 255.0
        fun channel(value: Double): Double =
            if (value <= 0.03928) value / 12.92 else ((value + 0.055) / 1.055).pow(2.4)
        val luminance = 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
        return (1.0 + 0.05) / (luminance + 0.05)
    }

    fun showsContrastWarning(
        primaryColor: String,
        serverWarning: Boolean,
        serverRatio: Double?,
    ): Boolean {
        if (serverWarning) return true
        if (serverRatio != null && serverRatio < WCAG_CONTRAST_MINIMUM) return true
        val ratio = contrastRatioAgainstWhite(primaryColor)
        return ratio != null && ratio < WCAG_CONTRAST_MINIMUM
    }

    fun parseAllowedModels(text: String): List<String> =
        text.split('\n', ',')
            .map { it.trim() }
            .filter { it.isNotEmpty() }

    fun allowedModelsText(models: List<String>?): String = models.orEmpty().joinToString("\n")

    fun buildAiConfigSaveRequest(
        enabled: Map<String, Boolean>,
        allowedModelsText: String,
    ): PutAiConfigRequest {
        val models = parseAllowedModels(allowedModelsText)
        val featuresEnabled = AI_FEATURE_KEYS.associate { feature ->
            feature.key to (enabled[feature.key] != false)
        }
        return PutAiConfigRequest(
            featuresEnabled = featuresEnabled,
            allowedModels = models.takeIf { it.isNotEmpty() },
        )
    }

    fun buildAiProviderSaveRequest(
        provider: String,
        modelAlias: String,
        fallbackProvider: String,
        byokKey: String,
    ): PutAiProviderSettingsRequest {
        val trimmedFallback = fallbackProvider.trim()
        val trimmedKey = byokKey.trim()
        val keyToSend =
            trimmedKey.takeIf { it.isNotEmpty() && it != PLATFORM_SECRET_PLACEHOLDER }
        return PutAiProviderSettingsRequest(
            provider = provider,
            modelAlias = modelAlias,
            fallbackProvider = trimmedFallback.takeIf { it.isNotEmpty() },
            byokApiKey = keyToSend,
        )
    }

    fun byokFieldValue(configured: Boolean, draft: String): String =
        draft.ifEmpty { if (configured) PLATFORM_SECRET_PLACEHOLDER else "" }

    fun shouldSendByokKey(value: String): Boolean {
        val trimmed = value.trim()
        return trimmed.isNotEmpty() && trimmed != PLATFORM_SECRET_PLACEHOLDER
    }

    fun brandingPutRequest(branding: OrgBrandingResponse): PutOrgBrandingRequest =
        PutOrgBrandingRequest(
            logoUrl = branding.logoUrl,
            faviconUrl = branding.faviconUrl,
            primaryColor = branding.primaryColor,
            secondaryColor = branding.secondaryColor,
            customDomain = branding.customDomain,
            customEmailDisplayName = branding.customEmailDisplayName,
        )

    fun providerLabel(provider: String): String = PROVIDER_LABELS[provider] ?: provider

    fun colorFromHex(hex: String): Color {
        val normalized = normalizedHexColor(hex) ?: return Color(0xFF4F46E5)
        val rgb = normalized.drop(1).toIntOrNull(16) ?: return Color(0xFF4F46E5)
        return Color(
            red = ((rgb shr 16) and 0xFF) / 255f,
            green = ((rgb shr 8) and 0xFF) / 255f,
            blue = (rgb and 0xFF) / 255f,
        )
    }

    fun userFacingError(error: Throwable, genericMessage: String): String =
        (error as? ApiError.HttpStatus)?.message?.takeIf { it.isNotEmpty() } ?: genericMessage
}
