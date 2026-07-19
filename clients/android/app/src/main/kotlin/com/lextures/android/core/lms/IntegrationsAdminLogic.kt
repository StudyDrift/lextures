package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError

/** Integrations & provisioning admin helpers (M14.8) — LTI, SCIM, cloud, LRS, OER. */
object IntegrationsAdminLogic {
    const val RBAC_MANAGE_PERMISSION = "global:app:rbac:manage"

    enum class Section {
        LTI,
        SCIM,
        CLOUD,
        LRS,
        OER,
        ;

        val webPath: String
            get() = when (this) {
                LTI -> "/settings/lti-tools"
                SCIM -> "/settings/scim-provisioning"
                CLOUD -> "/settings/cloud-providers"
                LRS -> "/settings/lrs-integrations"
                OER -> "/settings/oer-providers"
            }
    }

    fun adminSettingsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffMobileAdminSettings || features.ffMobileAdminConsole

    fun canManage(permissions: List<String>): Boolean =
        RBAC_MANAGE_PERMISSION in permissions

    fun shouldShowEntry(features: MobilePlatformFeatures, permissions: List<String>): Boolean =
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings && canManage(permissions)

    fun canView(features: MobilePlatformFeatures, permissions: List<String>): Boolean =
        adminSettingsEnabled(features) && canManage(permissions)

    fun isSectionVisible(
        section: Section,
        features: MobilePlatformFeatures,
        scimEnabled: Boolean,
    ): Boolean = when (section) {
        Section.LTI, Section.CLOUD -> true
        Section.SCIM -> scimEnabled
        Section.LRS -> features.xapiEmissionEnabled
        Section.OER -> features.oerLibraryEnabled
    }

    fun visibleSections(
        features: MobilePlatformFeatures,
        scimEnabled: Boolean,
    ): List<Section> = Section.entries.filter { isSectionVisible(it, features, scimEnabled) }

    fun cloudProviderLabelKey(provider: String): String = when (provider) {
        "google_drive" -> "mobile_admin_integrations_cloud_googleDrive"
        "onedrive" -> "mobile_admin_integrations_cloud_onedrive"
        "dropbox" -> "mobile_admin_integrations_cloud_dropbox"
        else -> provider
    }

    fun oerProviderLabelKey(provider: String): String = when (provider) {
        "oer_commons" -> "mobile_admin_integrations_oer_oerCommons"
        "merlot" -> "mobile_admin_integrations_oer_merlot"
        "openstax" -> "mobile_admin_integrations_oer_openstax"
        else -> provider
    }

    fun activeTokenCount(tokens: List<ScimTokenRow>): Int =
        tokens.count { it.revokedAt.isNullOrEmpty() }

    fun lastEventAt(events: List<ScimEventRow>): String? =
        events.firstOrNull()?.createdAt

    fun ltiActiveCount(platforms: List<LtiParentPlatform>, tools: List<LtiExternalTool>): Int =
        platforms.count { it.active } + tools.count { it.active }

    fun enabledProviderCount(providers: List<CloudProviderStatus>): Int =
        providers.count { it.enabled }

    fun enabledLrsCount(endpoints: List<LrsEndpointStatus>): Int =
        endpoints.count { it.enabled }

    fun enabledOerCount(providers: List<OerProviderStatus>): Int =
        providers.count { it.enabled }

    /** Ensures mobile never models secret fields on status payloads. */
    fun cloudStatusExcludesSecrets(jsonKeys: Set<String>): Boolean {
        val forbidden = setOf("clientId", "apiKey", "appKey", "client_id", "api_key", "app_key")
        return forbidden.none { it in jsonKeys }
    }

    fun applyingLtiPlatformActive(
        platforms: List<LtiParentPlatform>,
        id: String,
        active: Boolean,
    ): List<LtiParentPlatform> = platforms.map {
        if (it.id == id) it.copy(active = active) else it
    }

    fun applyingLtiToolActive(
        tools: List<LtiExternalTool>,
        id: String,
        active: Boolean,
    ): List<LtiExternalTool> = tools.map {
        if (it.id == id) it.copy(active = active) else it
    }

    fun applyingCloudEnabled(
        providers: List<CloudProviderStatus>,
        provider: String,
        enabled: Boolean,
    ): List<CloudProviderStatus> = providers.map {
        if (it.provider == provider) it.copy(enabled = enabled) else it
    }

    fun applyingLrsEnabled(
        endpoints: List<LrsEndpointStatus>,
        id: String,
        enabled: Boolean,
    ): List<LrsEndpointStatus> = endpoints.map {
        if (it.id == id) it.copy(enabled = enabled) else it
    }

    fun applyingOerEnabled(
        providers: List<OerProviderStatus>,
        provider: String,
        enabled: Boolean,
    ): List<OerProviderStatus> = providers.map {
        if (it.provider == provider) it.copy(enabled = enabled) else it
    }

    fun userFacingError(error: Throwable, fallback: String): String {
        val api = error as? ApiError.HttpStatus
        val message = api?.message
        return if (!message.isNullOrBlank()) message else fallback
    }

    fun webHubPath(): String = "/settings/lti-tools"
}
