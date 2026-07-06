package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.config.AppConfiguration

/** Credentials wallet helpers (M12.2). */
object WalletLogic {
    fun walletEnabled(features: MobilePlatformFeatures): Boolean =
        credentialsSectionEnabled(features) ||
            ccrEnabled(features) ||
            ceTranscriptEnabled(features) ||
            officialTranscriptsEnabled(features)

    fun credentialsSectionEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffCompletionCredentials

    fun ccrEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffCoCurricularTranscript

    fun ceTranscriptEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffCeuTracking

    fun officialTranscriptsEnabled(features: MobilePlatformFeatures): Boolean =
        features.ffTranscripts

    fun cacheKeyCcr(): String = "wallet:ccr"
    fun cacheKeyCeTranscript(): String = "wallet:ce-transcript"
    fun cacheKeyTranscriptRequests(): String = "wallet:transcript-requests"

    fun officialTranscriptWebUrl(): String = AppConfiguration.webUrl("/transcripts")

    fun dateLabel(iso: String): String = CredentialsLogic.issuedDateLabel(iso)

    fun achievementTypeLabel(type: String): String =
        type.replace('_', ' ').replaceFirstChar { if (it.isLowerCase()) it.titlecase() else it.toString() }

    fun transcriptStatusLabel(status: String): String = when (status) {
        "queued" -> "Queued"
        "submitted" -> "Submitted"
        "failed" -> "Failed"
        else -> status
    }

    fun deliveryTypeLabel(type: String): String = when (type.lowercase()) {
        "email" -> "Email"
        "mail" -> "Mail"
        "pickup" -> "Pickup"
        else -> type
    }

    fun ccrPdfPreviewTarget(documentId: String): FilePreviewTarget =
        FilePreviewTarget.submissionContentPath(
            courseCode = "wallet",
            contentPath = LmsApi.ccrDownloadPath(documentId, "pdf"),
            fileName = "ccr.pdf",
            mimeType = "application/pdf",
        )

    fun ceTranscriptPdfPreviewTarget(): FilePreviewTarget =
        FilePreviewTarget.submissionContentPath(
            courseCode = "wallet",
            contentPath = LmsApi.ceTranscriptPdfPath(),
            fileName = "ce-transcript.pdf",
            mimeType = "application/pdf",
        )
}
