import Foundation
import SwiftUI

/// Org branding, AI governance, and AI provider admin helpers (M14.5).
enum OrgBrandingAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"
    static let orgUnitsAdminPermission = "tenant:org:units:admin"
    /// Matches web `PLATFORM_SECRET_PLACEHOLDER` and server `placeholderSecretResponse`.
    static let secretPlaceholder = "••••••••••••"
    static let defaultPrimaryColor = "#4F46E5"
    static let defaultSecondaryColor = "#7C3AED"
    static let defaultProvider = "openrouter"
    static let defaultModelAlias = "claude-3-5-sonnet"
    static let contrastAAThreshold = 4.5

    static let featureKeys: [(key: String, labelKey: String)] = [
        ("ai_tutor", "mobile.admin.orgBranding.ai.feature.aiTutor"),
        ("rag_notebook", "mobile.admin.orgBranding.ai.feature.notebook"),
        ("syllabus_generation", "mobile.admin.orgBranding.ai.feature.syllabus"),
        ("translation", "mobile.admin.orgBranding.ai.feature.translation"),
        ("quiz_generation", "mobile.admin.orgBranding.ai.feature.quiz"),
        ("lesson_generation", "mobile.admin.orgBranding.ai.feature.lesson"),
    ]

    private static let providerLabelKeys: [String: String] = [
        "openrouter": "mobile.admin.orgBranding.provider.openrouter",
        "anthropic": "mobile.admin.orgBranding.provider.anthropic",
        "openai": "mobile.admin.orgBranding.provider.openai",
        "azure_openai": "mobile.admin.orgBranding.provider.azureOpenai",
        "bedrock": "mobile.admin.orgBranding.provider.bedrock",
        "vertex": "mobile.admin.orgBranding.provider.vertex",
    ]

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings
    }

    static func canManage(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission) || permissions.contains(orgUnitsAdminPermission)
    }

    static func shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features) && canManage(permissions: permissions)
    }

    static func canView(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        shouldShowEntry(features: features, permissions: permissions)
    }

    static func webBrandingPath() -> String { "/settings/org-branding" }

    static func resolveOrgId(accessToken: String?, courses: [CourseSummary]) -> String? {
        CourseCreateLogic.resolveOrgId(accessToken: accessToken, courses: courses)
    }

    static func resolveAssetURL(_ pathOrURL: String?) -> URL? {
        guard let raw = pathOrURL?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
            return nil
        }
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
            return URL(string: raw)
        }
        let path = raw.hasPrefix("/") ? raw : "/\(raw)"
        return AppConfiguration.apiURL(path: path)
    }

    static func isValidHexColor(_ value: String) -> Bool {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.hasPrefix("#") else { return false }
        let hex = String(trimmed.dropFirst())
        guard hex.count == 3 || hex.count == 6 else { return false }
        return hex.allSatisfy { $0.isHexDigit }
    }

    static func normalizeHexColor(_ value: String, fallback: String) -> String {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard isValidHexColor(trimmed) else { return fallback }
        let hex = String(trimmed.dropFirst())
        if hex.count == 3 {
            let expanded = hex.map { "\($0)\($0)" }.joined()
            return "#\(expanded.uppercased())"
        }
        return "#\(hex.uppercased())"
    }

    static func color(fromHex value: String) -> Color? {
        let normalized = normalizeHexColor(value, fallback: "")
        guard normalized.hasPrefix("#"), normalized.count == 7 else { return nil }
        let hex = String(normalized.dropFirst())
        guard let int = UInt64(hex, radix: 16) else { return nil }
        let red = Double((int >> 16) & 0xFF) / 255
        let green = Double((int >> 8) & 0xFF) / 255
        let blue = Double(int & 0xFF) / 255
        return Color(red: red, green: green, blue: blue)
    }

    /// Relative luminance contrast ratio of a hex color against white (WCAG).
    static func contrastRatioAgainstWhite(_ hex: String) -> Double? {
        let normalized = normalizeHexColor(hex, fallback: "")
        guard normalized.hasPrefix("#"), normalized.count == 7 else { return nil }
        let raw = String(normalized.dropFirst())
        guard let int = UInt64(raw, radix: 16) else { return nil }
        func channel(_ value: UInt64) -> Double {
            let component = Double(value) / 255
            return component <= 0.03928
                ? component / 12.92
                : pow((component + 0.055) / 1.055, 2.4)
        }
        let red = channel((int >> 16) & 0xFF)
        let green = channel((int >> 8) & 0xFF)
        let blue = channel(int & 0xFF)
        let whiteLuminance = 1.0
        let colorLuminance = 0.2126 * red + 0.7152 * green + 0.0722 * blue
        let lighter = max(whiteLuminance, colorLuminance)
        let darker = min(whiteLuminance, colorLuminance)
        return (lighter + 0.05) / (darker + 0.05)
    }

    static func hasContrastWarning(
        primaryColor: String,
        serverWarning: Bool,
        serverRatio: Double?
    ) -> Bool {
        if serverWarning { return true }
        if let serverRatio, serverRatio < contrastAAThreshold { return true }
        if let local = contrastRatioAgainstWhite(primaryColor), local < contrastAAThreshold {
            return true
        }
        return false
    }

    static func brandingPutBody(
        logoUrl: String?,
        faviconUrl: String?,
        primaryColor: String,
        secondaryColor: String,
        customEmailDisplayName: String?
    ) -> OrgBrandingPutRequest {
        let email = customEmailDisplayName?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        return OrgBrandingPutRequest(
            logoUrl: logoUrl,
            faviconUrl: faviconUrl,
            primaryColor: normalizeHexColor(primaryColor, fallback: defaultPrimaryColor),
            secondaryColor: normalizeHexColor(secondaryColor, fallback: defaultSecondaryColor),
            customDomain: nil,
            customEmailDisplayName: (email?.isEmpty == false) ? email : nil
        )
    }

    static func isFeatureEnabled(_ map: [String: Bool], key: String) -> Bool {
        map[key] != false
    }

    static func featuresEnabledPayload(_ enabled: [String: Bool]) -> [String: Bool] {
        var out: [String: Bool] = [:]
        for item in featureKeys {
            out[item.key] = isFeatureEnabled(enabled, key: item.key)
        }
        return out
    }

    static func parseAllowedModels(_ text: String) -> [String]? {
        let models = text
            .split { $0 == "\n" || $0 == "," }
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        return models.isEmpty ? nil : models
    }

    static func allowedModelsText(_ models: [String]?) -> String {
        (models ?? []).joined(separator: "\n")
    }

    static func aiConfigPutBody(
        enabled: [String: Bool],
        allowedModelsText: String
    ) -> AIGovernancePutRequest {
        AIGovernancePutRequest(
            featuresEnabled: featuresEnabledPayload(enabled),
            allowedModels: parseAllowedModels(allowedModelsText)
        )
    }

    /// Returns the BYOK key to send, or nil when the field is empty / still the mask.
    static func byokKeyForSave(_ entered: String) -> String? {
        let trimmed = entered.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty || trimmed == secretPlaceholder {
            return nil
        }
        return trimmed
    }

    static func displaySecretField(byokConfigured: Bool) -> String {
        byokConfigured ? secretPlaceholder : ""
    }

    static func isSecretPlaceholder(_ value: String) -> Bool {
        value.trimmingCharacters(in: .whitespacesAndNewlines) == secretPlaceholder
    }

    static func providerLabel(for provider: String) -> String {
        if let key = providerLabelKeys[provider] {
            return L.dynamicText(key)
        }
        return provider
    }

    static func providerOptions(from settings: AIProviderSettings?) -> [String] {
        let list = settings?.providers ?? Array(providerLabelKeys.keys).sorted()
        return list.isEmpty ? [defaultProvider] : list
    }

    static func modelAliasOptions(from settings: AIProviderSettings?) -> [String] {
        let list = settings?.modelAliases ?? [defaultModelAlias, "gpt-4o", "gemini-1.5-pro"]
        return list.isEmpty ? [defaultModelAlias] : list
    }

    static func aiProviderPutBody(
        provider: String,
        modelAlias: String,
        fallbackProvider: String,
        byokKey: String
    ) -> AIProviderSettingsPutRequest {
        let fallback = fallbackProvider.trimmingCharacters(in: .whitespacesAndNewlines)
        return AIProviderSettingsPutRequest(
            provider: provider.isEmpty ? defaultProvider : provider,
            modelAlias: modelAlias.isEmpty ? defaultModelAlias : modelAlias,
            fallbackProvider: fallback.isEmpty ? nil : fallback,
            byokApiKey: byokKeyForSave(byokKey)
        )
    }

    static func userFacingError(_ error: Error) -> String {
        if let apiError = error as? APIError, case let .httpStatus(_, message) = apiError,
           let message, !message.isEmpty {
            return message
        }
        return L.text("mobile.admin.orgBranding.error")
    }
}
