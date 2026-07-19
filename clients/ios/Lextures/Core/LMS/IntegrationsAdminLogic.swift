import Foundation

/// Integrations & provisioning admin helpers (M14.8) — LTI, SCIM, cloud, LRS, OER.
enum IntegrationsAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"

    enum Section: String, CaseIterable, Identifiable {
        case lti
        case scim
        case cloud
        case lrs
        case oer

        var id: String { rawValue }

        var titleKey: String.LocalizationValue {
            switch self {
            case .lti: "mobile.admin.integrations.lti.title"
            case .scim: "mobile.admin.integrations.scim.title"
            case .cloud: "mobile.admin.integrations.cloud.title"
            case .lrs: "mobile.admin.integrations.lrs.title"
            case .oer: "mobile.admin.integrations.oer.title"
            }
        }

        var subtitleKey: String.LocalizationValue {
            switch self {
            case .lti: "mobile.admin.integrations.lti.entry.subtitle"
            case .scim: "mobile.admin.integrations.scim.entry.subtitle"
            case .cloud: "mobile.admin.integrations.cloud.entry.subtitle"
            case .lrs: "mobile.admin.integrations.lrs.entry.subtitle"
            case .oer: "mobile.admin.integrations.oer.entry.subtitle"
            }
        }

        var systemImage: String {
            switch self {
            case .lti: "puzzlepiece.extension"
            case .scim: "person.2.badge.gearshape"
            case .cloud: "cloud"
            case .lrs: "arrow.triangle.branch"
            case .oer: "books.vertical"
            }
        }

        var webPath: String {
            switch self {
            case .lti: "/settings/lti-tools"
            case .scim: "/settings/scim-provisioning"
            case .cloud: "/settings/cloud-providers"
            case .lrs: "/settings/lrs-integrations"
            case .oer: "/settings/oer-providers"
            }
        }
    }

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings || features.ffMobileAdminConsole
    }

    static func canManage(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings && canManage(permissions: permissions)
    }

    static func canView(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features) && canManage(permissions: permissions)
    }

    /// LTI and cloud always available when admin entry is shown; SCIM/LRS/OER need extra flags.
    static func isSectionVisible(
        _ section: Section,
        features: MobilePlatformFeatures,
        scimEnabled: Bool
    ) -> Bool {
        switch section {
        case .lti, .cloud:
            return true
        case .scim:
            return scimEnabled
        case .lrs:
            return features.xapiEmissionEnabled
        case .oer:
            return features.oerLibraryEnabled
        }
    }

    static func visibleSections(
        features: MobilePlatformFeatures,
        scimEnabled: Bool
    ) -> [Section] {
        Section.allCases.filter { isSectionVisible($0, features: features, scimEnabled: scimEnabled) }
    }

    static func cloudProviderLabel(_ provider: String) -> String {
        switch provider {
        case "google_drive": return L.text("mobile.admin.integrations.cloud.googleDrive")
        case "onedrive": return L.text("mobile.admin.integrations.cloud.onedrive")
        case "dropbox": return L.text("mobile.admin.integrations.cloud.dropbox")
        default: return provider
        }
    }

    static func oerProviderLabel(_ provider: String) -> String {
        switch provider {
        case "oer_commons": return L.text("mobile.admin.integrations.oer.oerCommons")
        case "merlot": return L.text("mobile.admin.integrations.oer.merlot")
        case "openstax": return L.text("mobile.admin.integrations.oer.openstax")
        default: return provider
        }
    }

    static func activeTokenCount(_ tokens: [ScimTokenRow]) -> Int {
        tokens.filter { $0.revokedAt == nil || $0.revokedAt?.isEmpty == true }.count
    }

    static func lastEventAt(_ events: [ScimEventRow]) -> String? {
        events.first?.createdAt
    }

    static func formatTimestamp(_ raw: String?) -> String {
        let formatted = DateFormatting.formatDateTime(raw)
        return formatted.isEmpty ? L.text("mobile.emDash") : formatted
    }

    static func ltiActiveCount(platforms: [LtiParentPlatform], tools: [LtiExternalTool]) -> Int {
        platforms.filter(\.active).count + tools.filter(\.active).count
    }

    static func enabledCount(_ items: [(Bool)]) -> Int {
        items.filter { $0 }.count
    }

    static func enabledProviderCount(_ providers: [CloudProviderStatus]) -> Int {
        providers.filter(\.enabled).count
    }

    static func enabledLrsCount(_ endpoints: [LrsEndpointStatus]) -> Int {
        endpoints.filter(\.enabled).count
    }

    static func enabledOerCount(_ providers: [OerProviderStatus]) -> Int {
        providers.filter(\.enabled).count
    }

    /// Ensures mobile never treats secret-looking fields as present on status models.
    static func cloudStatusExcludesSecrets(_ jsonKeys: Set<String>) -> Bool {
        let forbidden = ["clientId", "apiKey", "appKey", "client_id", "api_key", "app_key"]
        return forbidden.allSatisfy { !jsonKeys.contains($0) }
    }

    static func applyingLtiPlatformActive(
        _ platforms: [LtiParentPlatform],
        id: String,
        active: Bool
    ) -> [LtiParentPlatform] {
        platforms.map { row in
            guard row.id == id else { return row }
            var copy = row
            copy.active = active
            return copy
        }
    }

    static func applyingLtiToolActive(
        _ tools: [LtiExternalTool],
        id: String,
        active: Bool
    ) -> [LtiExternalTool] {
        tools.map { row in
            guard row.id == id else { return row }
            var copy = row
            copy.active = active
            return copy
        }
    }

    static func applyingCloudEnabled(
        _ providers: [CloudProviderStatus],
        provider: String,
        enabled: Bool
    ) -> [CloudProviderStatus] {
        providers.map { row in
            guard row.provider == provider else { return row }
            var copy = row
            copy.enabled = enabled
            return copy
        }
    }

    static func applyingLrsEnabled(
        _ endpoints: [LrsEndpointStatus],
        id: String,
        enabled: Bool
    ) -> [LrsEndpointStatus] {
        endpoints.map { row in
            guard row.id == id else { return row }
            var copy = row
            copy.enabled = enabled
            return copy
        }
    }

    static func applyingOerEnabled(
        _ providers: [OerProviderStatus],
        provider: String,
        enabled: Bool
    ) -> [OerProviderStatus] {
        providers.map { row in
            guard row.provider == provider else { return row }
            var copy = row
            copy.enabled = enabled
            return copy
        }
    }

    static func userFacingError(_ error: Error, fallbackKey: String.LocalizationValue) -> String {
        if let api = error as? APIError, case let .httpStatus(_, message) = api, let message, !message.isEmpty {
            return message
        }
        return L.text(fallbackKey)
    }

    static func webHubPath() -> String { "/settings/lti-tools" }
}
