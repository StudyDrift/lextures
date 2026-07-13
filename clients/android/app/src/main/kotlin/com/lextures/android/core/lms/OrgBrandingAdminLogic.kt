package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError
import kotlin.math.pow

/** Org branding, AI governance, and AI provider admin helpers (M14.5). */
object OrgBrandingAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"
    const val ORG_UNITS_ADMIN_PERMISSION = "tenant:org:units:admin"

    /** Matches web `PLATFORM_SECRET_PLACEHOLDER` and server `placeholderSecretResponse`. */
    const val SECRET_PLACEHOLDER = "••••••••••••"
    const val DEFAULT_PRIMARY_COLOR = "#4F46E5"
    const val DEFAULT_SECONDARY_COLOR = "#7C3AED"
    const val DEFAULT_PROVIDER = "openrouter"
    const val DEFAULT_MODEL_ALIAS = "claude-3-5-sonnet"
    const val CONTRAST_AA_THRESHOLD = 4.5

    data class FeatureKey(val key: String, val labelResName: String)

    val FEATURE_KEYS: List<FeatureKey> = listOf(
        FeatureKey("ai_tutor", "mobile_admin_orgBranding_ai_feature_aiTutor"),
        FeatureKey("rag_notebook", "mobile_admin_orgBranding_ai_feature_notebook"),
        FeatureKey("syllabus_generation", "mobile_admin_orgBranding_ai_feature_syllabus"),
        FeatureKey("translation", "mobile_admin_orgBranding_ai_feature_translation"),
        FeatureKey("quiz_generation", "mobile_admin_orgBranding_ai_feature_quiz"),
        FeatureKey("lesson_generation", "mobile_admin_orgBranding_ai_feature_lesson"),
    )

    private val providerLabelKeys: Map<String, String> = mapOf(
        "openrouter" to "mobile_admin_orgBranding_provider_openrouter",
        "anthropic" to "mobile_admin_orgBranding_provider_anthropic",
        "openai" to "mobile_admin_orgBranding_provider_openai",
        "azure_openai" to "mobile_admin_orgBranding_provider_azureOpenai",
        "bedrock" to "mobile_admin_orgBranding_provider_bedrock",
        "vertex" to "mobile_admin_orgBranding_provider_vertex",
    )

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings

    fun canManage(permissions: List<String>): Boolean =
        permissions.contains(RBAC_MANAGE_PERMISSION) ||
            permissions.contains(ORG_UNITS_ADMIN_PERMISSION)

    fun shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean = adminSettingsEnabled(features) && canManage(permissions)

    fun canView(
        features: MobilePlatformFeatures,
        permissions: List<String>,
    ): Boolean = shouldShowEntry(features, permissions)

    fun webBrandingPath(): String = "/settings/org-branding"

    fun resolveOrgId(accessToken: String?, courses: List<CourseSummary>): String? =
        CourseCreateLogic.resolveOrgId(accessToken, courses)

    fun isValidHexColor(value: String): Boolean {
        val trimmed = value.trim()
        if (!trimmed.startsWith("#")) return false
        val hex = trimmed.drop(1)
        if (hex.length != 3 && hex.length != 6) return false
        return hex.all { it.isDigit() || it.lowercaseChar() in 'a'..'f' }
    }

    fun normalizeHexColor(value: String, fallback: String): String {
        val trimmed = value.trim()
        if (!isValidHexColor(trimmed)) return fallback
        val hex = trimmed.drop(1)
        return if (hex.length == 3) {
            "#" + hex.map { "$it$it" }.joinToString("").uppercase()
        } else {
            "#${hex.uppercase()}"
        }
    }

    /** Relative luminance contrast ratio of a hex color against white (WCAG). */
    fun contrastRatioAgainstWhite(hex: String): Double? {
        val normalized = normalizeHexColor(hex, "")
        if (!normalized.startsWith("#") || normalized.length != 7) return null
        val raw = normalized.drop(1).toLongOrNull(16) ?: return null
        fun channel(value: Long): Double {
            val c = value / 255.0
            return if (c <= 0.03928) c / 12.92 else ((c + 0.055) / 1.055).pow(2.4)
        }
        val r = channel((raw shr 16) and 0xFF)
        val g = channel((raw shr 8) and 0xFF)
        val b = channel(raw and 0xFF)
        val l1 = 1.0
        val l2 = 0.2126 * r + 0.7152 * g + 0.0722 * b
        val lighter = maxOf(l1, l2)
        val darker = minOf(l1, l2)
        return (lighter + 0.05) / (darker + 0.05)
    }

    fun hasContrastWarning(
        primaryColor: String,
        serverWarning: Boolean,
        serverRatio: Double?,
    ): Boolean {
        if (serverWarning) return true
        if (serverRatio != null && serverRatio < CONTRAST_AA_THRESHOLD) return true
        val local = contrastRatioAgainstWhite(primaryColor)
        return local != null && local < CONTRAST_AA_THRESHOLD
    }

    fun brandingPutBody(
        logoUrl: String?,
        faviconUrl: String?,
        primaryColor: String,
        secondaryColor: String,
        customEmailDisplayName: String?,
    ): OrgBrandingPutRequest {
        val email = customEmailDisplayName?.trim()
        return OrgBrandingPutRequest(
            logoUrl = logoUrl,
            faviconUrl = faviconUrl,
            primaryColor = normalizeHexColor(primaryColor, DEFAULT_PRIMARY_COLOR),
            secondaryColor = normalizeHexColor(secondaryColor, DEFAULT_SECONDARY_COLOR),
            customDomain = null,
            customEmailDisplayName = email?.takeIf { it.isNotEmpty() },
        )
    }

    fun isFeatureEnabled(map: Map<String, Boolean>, key: String): Boolean =
        map[key] != false

    fun featuresEnabledPayload(enabled: Map<String, Boolean>): Map<String, Boolean> =
        FEATURE_KEYS.associate { it.key to isFeatureEnabled(enabled, it.key) }

    fun parseAllowedModels(text: String): List<String>? {
        val models = text
            .split('\n', ',')
            .map { it.trim() }
            .filter { it.isNotEmpty() }
        return models.takeIf { it.isNotEmpty() }
    }

    fun allowedModelsText(models: List<String>?): String =
        models.orEmpty().joinToString("\n")

    fun aiConfigPutBody(
        enabled: Map<String, Boolean>,
        allowedModelsText: String,
    ): AIGovernancePutRequest =
        AIGovernancePutRequest(
            featuresEnabled = featuresEnabledPayload(enabled),
            allowedModels = parseAllowedModels(allowedModelsText),
        )

    /** Returns the BYOK key to send, or null when empty / still the mask. */
    fun byokKeyForSave(entered: String): String? {
        val trimmed = entered.trim()
        if (trimmed.isEmpty() || trimmed == SECRET_PLACEHOLDER) return null
        return trimmed
    }

    fun displaySecretField(byokConfigured: Boolean): String =
        if (byokConfigured) SECRET_PLACEHOLDER else ""

    fun isSecretPlaceholder(value: String): Boolean =
        value.trim() == SECRET_PLACEHOLDER

    fun providerLabelKey(provider: String): String? = providerLabelKeys[provider]

    fun providerOptions(settings: AIProviderSettings?): List<String> {
        val list = settings?.providers ?: providerLabelKeys.keys.sorted()
        return list.ifEmpty { listOf(DEFAULT_PROVIDER) }
    }

    fun modelAliasOptions(settings: AIProviderSettings?): List<String> {
        val list = settings?.modelAliases
            ?: listOf(DEFAULT_MODEL_ALIAS, "gpt-4o", "gemini-1.5-pro")
        return list.ifEmpty { listOf(DEFAULT_MODEL_ALIAS) }
    }

    fun aiProviderPutBody(
        provider: String,
        modelAlias: String,
        fallbackProvider: String,
        byokKey: String,
    ): AIProviderSettingsPutRequest {
        val fallback = fallbackProvider.trim()
        return AIProviderSettingsPutRequest(
            provider = provider.ifBlank { DEFAULT_PROVIDER },
            modelAlias = modelAlias.ifBlank { DEFAULT_MODEL_ALIAS },
            fallbackProvider = fallback.takeIf { it.isNotEmpty() },
            byokApiKey = byokKeyForSave(byokKey),
        )
    }

    fun userFacingError(error: Throwable, genericMessage: String): String =
        (error as? ApiError.HttpStatus)?.message?.takeIf { it.isNotEmpty() } ?: genericMessage
}
