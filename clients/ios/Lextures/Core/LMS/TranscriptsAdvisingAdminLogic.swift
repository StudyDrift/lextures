import Foundation

/// Transcripts & advising configuration admin helpers (M14.9).
enum TranscriptsAdvisingAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"
    static let secretPlaceholder = "••••••••••••"

    enum Section: String, CaseIterable, Identifiable {
        case transcripts
        case advising

        var id: String { rawValue }

        var titleKey: String.LocalizationValue {
            switch self {
            case .transcripts: "mobile.admin.transcripts.title"
            case .advising: "mobile.admin.advising.title"
            }
        }

        var subtitleKey: String.LocalizationValue {
            switch self {
            case .transcripts: "mobile.admin.transcripts.entry.subtitle"
            case .advising: "mobile.admin.advising.entry.subtitle"
            }
        }

        var systemImage: String {
            switch self {
            case .transcripts: "doc.text"
            case .advising: "person.2.wave.2"
            }
        }

        var webPath: String {
            switch self {
            case .transcripts: "/settings/transcripts"
            case .advising: "/settings/advising"
            }
        }
    }

    enum DegreeAuditProvider: String, CaseIterable, Identifiable {
        case none
        case degreeworks
        case stellic

        var id: String { rawValue }

        var labelKey: String.LocalizationValue {
            switch self {
            case .none: "mobile.admin.advising.provider.none"
            case .degreeworks: "mobile.admin.advising.provider.degreeworks"
            case .stellic: "mobile.admin.advising.provider.stellic"
            }
        }

        static func normalized(_ raw: String) -> DegreeAuditProvider {
            DegreeAuditProvider(rawValue: raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()) ?? .none
        }
    }

    enum SaveStatus: Equatable {
        case idle
        case saving
        case saved
        case error(String)
    }

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings
    }

    static func canManage(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func isTranscriptsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffTranscripts
    }

    static func isAdvisingEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffAdvisingIntegration
    }

    static func shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features)
            && canManage(permissions: permissions)
            && (isTranscriptsEnabled(features) || isAdvisingEnabled(features))
    }

    static func canView(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        shouldShowEntry(features: features, permissions: permissions)
    }

    static func canViewTranscripts(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features)
            && canManage(permissions: permissions)
            && isTranscriptsEnabled(features)
    }

    static func canViewAdvising(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features)
            && canManage(permissions: permissions)
            && isAdvisingEnabled(features)
    }

    static func isSectionVisible(
        _ section: Section,
        features: MobilePlatformFeatures
    ) -> Bool {
        switch section {
        case .transcripts: return isTranscriptsEnabled(features)
        case .advising: return isAdvisingEnabled(features)
        }
    }

    static func visibleSections(features: MobilePlatformFeatures) -> [Section] {
        Section.allCases.filter { isSectionVisible($0, features: features) }
    }

    static func webhookSecretField(from config: AdminTranscriptsConfig) -> String {
        config.hasWebhookSecret ? secretPlaceholder : ""
    }

    /// Builds the PUT body. Only send a new webhook secret when the draft differs from the
    /// masked placeholder (leave blank / placeholder to keep the existing secret).
    static func buildTranscriptsSaveRequest(
        webhookUrl: String,
        webhookSecret: String,
        pickupInstructions: String
    ) -> PutAdminTranscriptsConfigRequest {
        let url = webhookUrl.trimmingCharacters(in: .whitespacesAndNewlines)
        let secret = webhookSecret.trimmingCharacters(in: .whitespacesAndNewlines)
        let pickup = pickupInstructions.trimmingCharacters(in: .whitespacesAndNewlines)
        var request = PutAdminTranscriptsConfigRequest(
            webhookUrl: url,
            webhookSecret: nil,
            pickupInstructions: pickup
        )
        if !secret.isEmpty, secret != secretPlaceholder {
            request.webhookSecret = secret
        }
        return request
    }

    static func isTranscriptsSaveDisabled(saving: Bool, webhookUrl: String) -> Bool {
        saving || !isValidHttpUrl(webhookUrl)
    }

    static func isValidHttpUrl(_ raw: String) -> Bool {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }
        guard let url = URL(string: trimmed), let scheme = url.scheme?.lowercased() else { return false }
        return scheme == "http" || scheme == "https"
    }

    static func buildAdvisingSaveRequest(
        appointmentUrl: String,
        provider: DegreeAuditProvider,
        baseUrl: String,
        credentialsRef: String,
        atRiskBannerEnabled: Bool
    ) -> PutAdminAdvisingConfigRequest {
        PutAdminAdvisingConfigRequest(
            appointmentUrl: appointmentUrl.trimmingCharacters(in: .whitespacesAndNewlines),
            degreeAuditProvider: provider.rawValue,
            degreeAuditBaseUrl: provider == .none
                ? ""
                : baseUrl.trimmingCharacters(in: .whitespacesAndNewlines),
            apiCredentialsRef: provider == .none
                ? ""
                : credentialsRef.trimmingCharacters(in: .whitespacesAndNewlines),
            atRiskBannerEnabled: provider == .none ? false : atRiskBannerEnabled
        )
    }

    static func isAdvisingSaveDisabled(saving: Bool, appointmentUrl: String) -> Bool {
        if saving { return true }
        let trimmed = appointmentUrl.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return false }
        return !isValidHttpUrl(trimmed)
    }

    static func formatTimestamp(_ raw: String?) -> String {
        let formatted = DateFormatting.formatDateTime(raw)
        return formatted.isEmpty ? L.text("mobile.emDash") : formatted
    }

    static func httpStatusLabel(_ code: Int?) -> String {
        guard let code else { return L.text("mobile.emDash") }
        return "\(code)"
    }

    static func userFacingError(_ error: Error, fallbackKey: String.LocalizationValue) -> String {
        if let api = error as? APIError, case let .httpStatus(_, message) = api, let message, !message.isEmpty {
            return message
        }
        return L.text(fallbackKey)
    }

    static func webHubPath() -> String { "/settings/transcripts" }
}
