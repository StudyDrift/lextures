package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class IssuedCredentialSummary(
    val id: String,
    val title: String,
    val sourceType: String,
    val sourceId: String,
    val issuedAt: String,
    val verificationUrl: String,
    val revoked: Boolean = false,
)

@Serializable
data class CredentialsListResponse(
    val credentials: List<IssuedCredentialSummary>? = null,
)

@Serializable
data class CredentialLinkedInParams(
    val name: String,
    val organizationName: String,
    val issueYear: Int,
    val issueMonth: Int,
    val certUrl: String,
    val certId: String,
    val url: String,
)

@Serializable
data class CredentialBadgeExportResponse(
    val downloadUrl: String,
    val expiresAt: String,
)

@Serializable
data class CredentialShareRequest(
    val channel: String,
)