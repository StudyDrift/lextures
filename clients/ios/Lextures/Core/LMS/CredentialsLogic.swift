import Foundation

/// Completion credentials helpers (M9.3).
enum CredentialsLogic {
    static func credentialsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffCompletionCredentials
    }

    static func cacheKey() -> String { "credentials:list" }

    static func credentialDetailCacheKey(id: String) -> String { "credentials:\(id)" }

    static func sourceTypeLabel(_ sourceType: String) -> String {
        switch sourceType {
        case "course":
            return L.text("mobile.credentials.source.course")
        case "path":
            return L.text("mobile.credentials.source.path")
        case "ceu":
            return L.text("mobile.credentials.source.ceu")
        default:
            return sourceType
        }
    }

    static func issuedDateLabel(iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        var date = formatter.date(from: iso)
        if date == nil {
            formatter.formatOptions = [.withInternetDateTime]
            date = formatter.date(from: iso)
        }
        guard let date else { return iso }
        return date.formatted(date: .abbreviated, time: .omitted)
    }

    static func shareItems(verificationUrl: String, title: String) -> [Any] {
        [L.format("mobile.credentials.shareText", title, verificationUrl)]
    }
}