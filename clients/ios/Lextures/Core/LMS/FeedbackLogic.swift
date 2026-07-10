import Foundation

/// Product feedback helpers (FB3) — gating, validation, and payload building.
enum FeedbackLogic {
    static let maxMessageLength = 5000
    static let source = "ios"
    static let categories = ["bug", "idea", "question", "praise", "other"]

    static func feedbackEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffFeedback
    }

    static func messageValid(_ message: String) -> Bool {
        !message.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    static func trimmedMessageLength(_ message: String) -> Int {
        message.trimmingCharacters(in: .whitespacesAndNewlines).count
    }

    static func appVersion() -> String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "1"
        return "\(version) (\(build))"
    }

    static func buildSubmitRequest(
        message: String,
        category: String,
        route: String,
        locale: String?,
        viewport: String?
    ) -> SubmitFeedbackRequest {
        SubmitFeedbackRequest(
            message: message.trimmingCharacters(in: .whitespacesAndNewlines),
            source: source,
            appVersion: appVersion(),
            context: FeedbackContextPayload(route: route, locale: locale, viewport: viewport),
            category: category.trimmingCharacters(in: .whitespacesAndNewlines)
        )
    }

    enum SubmitOutcome: Equatable {
        case success
        case rateLimited
        case offline
        case error
    }

    static func mapSubmitError(_ error: Error, isOnline: Bool) -> SubmitOutcome {
        if !isOnline {
            return .offline
        }
        if case APIError.httpStatus(429, _) = error {
            return .rateLimited
        }
        if case APIError.transport = error {
            return .offline
        }
        return .error
    }

    static func errorMessageKey(for outcome: SubmitOutcome) -> String {
        switch outcome {
        case .success:
            return "mobile.feedback.success"
        case .rateLimited:
            return "mobile.feedback.rateLimited"
        case .offline:
            return "mobile.feedback.offline"
        case .error:
            return "mobile.feedback.error"
        }
    }

    static func categoryLabelKey(_ category: String) -> String {
        let trimmed = category.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "mobile.feedback.category.none" }
        switch trimmed {
        case "bug", "idea", "question", "praise", "other":
            return "mobile.feedback.category.\(trimmed)"
        default:
            return "mobile.feedback.category.other"
        }
    }
}
