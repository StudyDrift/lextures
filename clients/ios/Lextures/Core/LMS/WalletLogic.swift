import Foundation

/// Credentials wallet helpers (M12.2).
enum WalletLogic {
    static func walletEnabled(_ features: MobilePlatformFeatures) -> Bool {
        credentialsSectionEnabled(features)
            || ccrEnabled(features)
            || ceTranscriptEnabled(features)
            || officialTranscriptsEnabled(features)
    }

    static func credentialsSectionEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffCompletionCredentials
    }

    static func ccrEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffCoCurricularTranscript
    }

    static func ceTranscriptEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffCeuTracking
    }

    static func officialTranscriptsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffTranscripts
    }

    static func cacheKeySummary() -> String { "wallet:summary" }
    static func cacheKeyCCR() -> String { "wallet:ccr" }
    static func cacheKeyCETranscript() -> String { "wallet:ce-transcript" }
    static func cacheKeyTranscriptRequests() -> String { "wallet:transcript-requests" }

    static func officialTranscriptWebURL() -> URL {
        AppConfiguration.webURL(path: "/transcripts")
    }

    static func dateLabel(iso: String) -> String {
        CredentialsLogic.issuedDateLabel(iso: iso)
    }

    static func achievementTypeLabel(_ type: String) -> String {
        type.replacingOccurrences(of: "_", with: " ").capitalized
    }

    static func transcriptStatusLabel(_ status: String) -> String {
        switch status {
        case "queued":
            return L.text("mobile.wallet.requestStatus.queued")
        case "submitted":
            return L.text("mobile.wallet.requestStatus.submitted")
        case "failed":
            return L.text("mobile.wallet.requestStatus.failed")
        default:
            return status
        }
    }

    static func deliveryTypeLabel(_ type: String) -> String {
        switch type.lowercased() {
        case "email":
            return L.text("mobile.wallet.delivery.email")
        case "mail":
            return L.text("mobile.wallet.delivery.mail")
        case "pickup":
            return L.text("mobile.wallet.delivery.pickup")
        default:
            return type
        }
    }

    static func ccrPdfPreviewTarget(documentId: String) -> FilePreviewTarget {
        FilePreviewTarget(
            courseCode: "_wallet",
            displayName: "ccr.pdf",
            mimeType: "application/pdf",
            byteSize: nil,
            source: .directPath(LMSAPI.ccrDownloadPath(documentId: documentId, format: "pdf"))
        )
    }

    static func ceTranscriptPdfPreviewTarget() -> FilePreviewTarget {
        FilePreviewTarget(
            courseCode: "_wallet",
            displayName: "ce-transcript.pdf",
            mimeType: "application/pdf",
            byteSize: nil,
            source: .directPath(LMSAPI.ceTranscriptPdfPath())
        )
    }
}
