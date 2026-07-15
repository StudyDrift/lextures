package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// Status-only models for M14.8 — secrets are intentionally not modeled.

@Serializable
data class LtiRegistrationsResponse(
    val parentPlatforms: List<LtiParentPlatform> = emptyList(),
    val externalTools: List<LtiExternalTool> = emptyList(),
)

@Serializable
data class LtiParentPlatform(
    val id: String,
    val name: String = "",
    val clientId: String = "",
    val platformIss: String = "",
    val active: Boolean = false,
)

@Serializable
data class LtiExternalTool(
    val id: String,
    val name: String = "",
    val clientId: String = "",
    val toolIssuer: String = "",
    val active: Boolean = false,
)

@Serializable
data class LtiActiveBody(val active: Boolean)

@Serializable
data class ScimTokensResponse(val tokens: List<ScimTokenRow>? = null)

@Serializable
data class ScimTokenRow(
    val id: String,
    val institutionId: String = "",
    val label: String = "",
    val createdAt: String = "",
    val revokedAt: String? = null,
)

@Serializable
data class ScimEventsResponse(val events: List<ScimEventRow>? = null)

@Serializable
data class ScimEventRow(
    val id: String,
    val operation: String = "",
    val scimResource: String = "",
    val userEmail: String? = null,
    val createdAt: String = "",
)

@Serializable
data class PlatformScimFlag(val scimEnabled: Boolean? = null)

/** Secret-free cloud provider status for mobile admin. */
@Serializable
data class CloudProviderStatus(
    val provider: String,
    val enabled: Boolean = false,
    val updatedAt: String? = null,
)

@Serializable
data class CloudProviderEnabledBody(val enabled: Boolean)

@Serializable
data class LrsEndpointStatus(
    val id: String,
    val label: String = "",
    val endpointUrl: String = "",
    val authType: String = "",
    val username: String? = null,
    val enabled: Boolean = false,
    val hasPassword: Boolean? = null,
    val hasOauthSecret: Boolean? = null,
    val updatedAt: String? = null,
)

@Serializable
data class LrsEnabledBody(val enabled: Boolean)

@Serializable
data class OerProviderStatus(
    val provider: String,
    val enabled: Boolean = false,
    val updatedAt: String? = null,
)

@Serializable
data class OerProviderEnabledBody(val enabled: Boolean)
