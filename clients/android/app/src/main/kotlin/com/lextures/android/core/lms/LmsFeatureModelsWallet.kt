package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

@Serializable
data class CCRAchievement(
    val id: String,
    val type: String,
    val title: String,
    val description: String? = null,
    val issuedAt: String,
    val evidenceUrl: String? = null,
    val outcomeTags: List<String>? = null,
)

@Serializable
data class CCRDocument(
    val id: String,
    val generatedAt: String,
    val shareable: Boolean = false,
    val verificationUrl: String? = null,
)

@Serializable
data class CCRSummaryResponse(
    val achievements: List<CCRAchievement>? = null,
    val documents: List<CCRDocument>? = null,
)

@Serializable
data class CCRGenerateRequest(
    val sharePublicly: Boolean,
)

@Serializable
data class CCRGenerateResponse(
    val document: CCRDocument,
    val achievements: List<CCRAchievement>? = null,
    val verificationUrl: String? = null,
)

@Serializable
data class CETranscriptAward(
    val courseTitle: String,
    val ceuCredit: Double,
    val contactHours: Double,
    val completedAt: String,
)

@Serializable
data class CETranscriptResponse(
    val awards: List<CETranscriptAward>? = null,
)

@Serializable
data class TranscriptRequestSummary(
    val id: String,
    val status: String,
    val deliveryType: String,
    val deliveryEmail: String? = null,
    val deliveryAddress: String? = null,
    val urgencyDays: Int? = null,
    val urgencyDaysMin: Int? = null,
    val urgencyUnit: String? = null,
    val requestedAt: String,
    val submittedAt: String? = null,
    val errorMessage: String? = null,
    val webhookResponseCode: Int? = null,
)

@Serializable
data class TranscriptRequestsResponse(
    val requests: List<TranscriptRequestSummary>? = null,
)

@Serializable
data class TranscriptsStudentConfig(
    val pickupInstructions: String? = null,
    val pickupAvailable: Boolean = false,
)
