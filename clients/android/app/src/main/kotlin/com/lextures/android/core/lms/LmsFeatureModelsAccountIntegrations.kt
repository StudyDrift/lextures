package com.lextures.android.core.lms

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

@Serializable
data class AccessKeyScopeDef(
    val id: String,
    val label: String,
    val description: String,
    val group: String,
)

@Serializable
data class AccessKeyScopesResponse(val scopes: List<AccessKeyScopeDef> = emptyList())

@Serializable
data class AccessKeyCourseSummary(
    val id: String,
    val courseCode: String,
    val title: String,
)

@Serializable
data class AccessKeySummary(
    val id: String,
    val label: String,
    val tokenMask: String,
    val scopes: List<String> = emptyList(),
    val courseIds: List<String>? = null,
    val courses: List<AccessKeyCourseSummary>? = null,
    val allCourses: Boolean? = null,
    val isServiceToken: Boolean? = null,
    val serviceAccountName: String? = null,
    val expiresAt: String? = null,
    val lastUsedAt: String? = null,
    val revokedAt: String? = null,
    val createdAt: String,
    val unusedDays: Int? = null,
)

@Serializable
data class AccessKeysListResponse(val tokens: List<AccessKeySummary> = emptyList())

@Serializable
data class CreateAccessKeyRequest(
    val label: String,
    val scopes: List<String>,
    val courseIds: List<String> = emptyList(),
)

@Serializable
data class CreateAccessKeyResponse(
    val token: String? = null,
    val label: String? = null,
)

@Serializable
data class RotateAccessKeyRequest(val overlapHours: Int = 24)

@Serializable
data class RotateAccessKeyResponse(
    val token: String? = null,
    val label: String? = null,
)

@Serializable
data class MCPConfigResponse(
    val apiBaseUrl: String,
    val cursorConfig: JsonObject,
    val claudeDesktopConfig: JsonObject,
    val instructions: List<String> = emptyList(),
)

@Serializable
data class CreateServiceTokenRequest(
    val serviceAccountName: String,
    val label: String,
    val scopes: List<String>,
)

@Serializable
data class CreateServiceTokenResponse(
    val token: String? = null,
    val label: String? = null,
)

data class OneTimeSecretReveal(
    val token: String,
    val label: String,
)