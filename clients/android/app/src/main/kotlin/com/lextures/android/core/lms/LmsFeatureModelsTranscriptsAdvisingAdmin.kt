package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// Transcripts & Advising admin models (M14.9)

@Serializable
data class AdminTranscriptsConfig(
    val webhookUrl: String = "",
    val webhookSecret: String? = null,
    val hasWebhookSecret: Boolean = false,
    val pickupInstructions: String? = null,
)

@Serializable
data class PutAdminTranscriptsConfigRequest(
    val webhookUrl: String,
    val webhookSecret: String? = null,
    val pickupInstructions: String? = null,
)

@Serializable
data class AdminTranscriptRequestRow(
    val id: String,
    val status: String? = null,
    val deliveryType: String? = null,
    val requestedAt: String = "",
    val submittedAt: String? = null,
    val errorMessage: String? = null,
    val webhookResponseCode: Int? = null,
)

@Serializable
data class AdminTranscriptRequestsResponse(
    val requests: List<AdminTranscriptRequestRow>? = null,
)

@Serializable
data class AdminAdvisingConfig(
    val appointmentUrl: String = "",
    val degreeAuditProvider: String = "none",
    val degreeAuditBaseUrl: String = "",
    val apiCredentialsRef: String = "",
    val atRiskBannerEnabled: Boolean = false,
)

@Serializable
data class PutAdminAdvisingConfigRequest(
    val appointmentUrl: String = "",
    val degreeAuditProvider: String = "none",
    val degreeAuditBaseUrl: String = "",
    val apiCredentialsRef: String = "",
    val atRiskBannerEnabled: Boolean = false,
)
