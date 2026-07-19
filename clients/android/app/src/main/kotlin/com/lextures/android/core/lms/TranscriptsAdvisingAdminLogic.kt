package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError
import java.net.URI

/** Transcripts & advising configuration admin helpers (M14.9). */
object TranscriptsAdvisingAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"
    const val SECRET_PLACEHOLDER = "••••••••••••"

    enum class Section {
        TRANSCRIPTS,
        ADVISING,
        ;

        val webPath: String
            get() = when (this) {
                TRANSCRIPTS -> "/settings/transcripts"
                ADVISING -> "/settings/advising"
            }
    }

    enum class DegreeAuditProvider(val id: String) {
        NONE("none"),
        DEGREEWORKS("degreeworks"),
        STELLIC("stellic"),
        ;

        companion object {
            fun normalized(raw: String): DegreeAuditProvider {
                val key = raw.trim().lowercase()
                return entries.firstOrNull { it.id == key } ?: NONE
            }
        }
    }

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings || features.ffMobileAdminConsole

    fun canManage(permissions: Collection<String>): Boolean =
        RBAC_MANAGE_PERMISSION in permissions

    fun isTranscriptsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffTranscripts

    fun isAdvisingEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffAdvisingIntegration

    fun shouldShowEntry(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings &&
            canManage(permissions) &&
            (isTranscriptsEnabled(features) || isAdvisingEnabled(features))

    fun canView(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        adminSettingsEnabled(features) && canManage(permissions)

    fun canViewTranscripts(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        adminSettingsEnabled(features) && canManage(permissions) && isTranscriptsEnabled(features)

    fun canViewAdvising(features: MobilePlatformFeatures, permissions: Collection<String>): Boolean =
        adminSettingsEnabled(features) && canManage(permissions) && isAdvisingEnabled(features)

    fun isSectionVisible(section: Section, features: MobilePlatformFeatures): Boolean = when (section) {
        Section.TRANSCRIPTS -> isTranscriptsEnabled(features)
        Section.ADVISING -> isAdvisingEnabled(features)
    }

    fun visibleSections(features: MobilePlatformFeatures): List<Section> =
        Section.entries.filter { isSectionVisible(it, features) }

    fun webhookSecretField(config: AdminTranscriptsConfig): String =
        if (config.hasWebhookSecret) SECRET_PLACEHOLDER else ""

    /**
     * Builds the PUT body. Only send a new webhook secret when the draft differs from the
     * masked placeholder (leave blank / placeholder to keep the existing secret).
     */
    fun buildTranscriptsSaveRequest(
        webhookUrl: String,
        webhookSecret: String,
        pickupInstructions: String,
    ): PutAdminTranscriptsConfigRequest {
        val secret = webhookSecret.trim()
        return PutAdminTranscriptsConfigRequest(
            webhookUrl = webhookUrl.trim(),
            webhookSecret = if (secret.isNotEmpty() && secret != SECRET_PLACEHOLDER) secret else null,
            pickupInstructions = pickupInstructions.trim(),
        )
    }

    fun isTranscriptsSaveDisabled(saving: Boolean, webhookUrl: String): Boolean =
        saving || !isValidHttpUrl(webhookUrl)

    fun isValidHttpUrl(raw: String): Boolean {
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return false
        return try {
            val uri = URI(trimmed)
            val scheme = uri.scheme?.lowercase()
            (scheme == "http" || scheme == "https") && !uri.host.isNullOrBlank()
        } catch (_: Exception) {
            false
        }
    }

    fun buildAdvisingSaveRequest(
        appointmentUrl: String,
        provider: DegreeAuditProvider,
        baseUrl: String,
        credentialsRef: String,
        atRiskBannerEnabled: Boolean,
    ): PutAdminAdvisingConfigRequest {
        val none = provider == DegreeAuditProvider.NONE
        return PutAdminAdvisingConfigRequest(
            appointmentUrl = appointmentUrl.trim(),
            degreeAuditProvider = provider.id,
            degreeAuditBaseUrl = if (none) "" else baseUrl.trim(),
            apiCredentialsRef = if (none) "" else credentialsRef.trim(),
            atRiskBannerEnabled = if (none) false else atRiskBannerEnabled,
        )
    }

    fun isAdvisingSaveDisabled(saving: Boolean, appointmentUrl: String): Boolean {
        if (saving) return true
        val trimmed = appointmentUrl.trim()
        if (trimmed.isEmpty()) return false
        return !isValidHttpUrl(trimmed)
    }

    fun httpStatusLabel(code: Int?): String = code?.toString() ?: "—"

    fun userFacingError(error: Throwable, fallback: String): String {
        val api = error as? ApiError.HttpStatus
        val message = api?.message
        return if (!message.isNullOrBlank()) message else fallback
    }

    fun webHubPath(): String = "/settings/transcripts"
}
