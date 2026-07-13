package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class OrgBrandingResponse(
    val logoUrl: String? = null,
    val faviconUrl: String? = null,
    val primaryColor: String = OrgBrandingAdminLogic.DEFAULT_PRIMARY_COLOR,
    val secondaryColor: String = OrgBrandingAdminLogic.DEFAULT_SECONDARY_COLOR,
    val customDomain: String? = null,
    val customEmailDisplayName: String? = null,
    val contrastWarningPrimary: Boolean? = null,
    val contrastRatioPrimary: Double? = null,
)

@Serializable
data class PutOrgBrandingRequest(
    val logoUrl: String? = null,
    val faviconUrl: String? = null,
    val primaryColor: String,
    val secondaryColor: String,
    val customDomain: String? = null,
    val customEmailDisplayName: String? = null,
)

@Serializable
data class OrgBrandingUploadResponse(
    val url: String? = null,
)

@Serializable
data class AiConfigResponse(
    val orgId: String? = null,
    val featuresEnabled: Map<String, Boolean>? = null,
    val allowedModels: List<String>? = null,
)

@Serializable
data class PutAiConfigRequest(
    val featuresEnabled: Map<String, Boolean>,
    val allowedModels: List<String>? = null,
)

@Serializable
data class AiProviderSettingsResponse(
    val orgId: String? = null,
    val provider: String? = null,
    val modelAlias: String? = null,
    val fallbackProvider: String? = null,
    val byokConfigured: Boolean? = null,
    val settings: Map<String, JsonElement>? = null,
    val providers: List<String>? = null,
    val modelAliases: List<String>? = null,
)

@Serializable
data class PutAiProviderSettingsRequest(
    val provider: String,
    val modelAlias: String,
    val fallbackProvider: String? = null,
    @SerialName("byokApiKey") val byokApiKey: String? = null,
)

@Serializable
data class AiProviderTestResponse(
    val provider: String? = null,
    val latencyMs: Int? = null,
    val responsePreview: String? = null,
)
