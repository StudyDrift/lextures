package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// MARK: - Org branding / AI governance / AI provider admin (M14.5)

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
data class OrgBrandingPutRequest(
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
data class AIGovernanceConfig(
    val orgId: String? = null,
    val featuresEnabled: Map<String, Boolean>? = null,
    val allowedModels: List<String>? = null,
    val updatedAt: String? = null,
    val updatedBy: String? = null,
)

@Serializable
data class AIGovernancePutRequest(
    val featuresEnabled: Map<String, Boolean>,
    val allowedModels: List<String>? = null,
)

@Serializable
data class AIProviderSettings(
    val orgId: String? = null,
    val provider: String? = null,
    val modelAlias: String? = null,
    val fallbackProvider: String? = null,
    val byokConfigured: Boolean? = null,
    val providers: List<String>? = null,
    val modelAliases: List<String>? = null,
    val updatedAt: String? = null,
    val updatedBy: String? = null,
)

@Serializable
data class AIProviderSettingsPutRequest(
    val provider: String,
    val modelAlias: String,
    val fallbackProvider: String? = null,
    val byokApiKey: String? = null,
)

@Serializable
data class AIProviderTestResponse(
    val provider: String? = null,
    val latencyMs: Double? = null,
    val responsePreview: String? = null,
)
