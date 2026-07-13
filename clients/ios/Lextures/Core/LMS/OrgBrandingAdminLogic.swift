import Foundation
import SwiftUI

/// Org branding, AI governance, and AI provider admin helpers (M14.5).
enum OrgBrandingAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"
    static let orgUnitsAdminPermission = "tenant:org:units:admin"
    static let platformSecretPlaceholder = "••••••••••••"
    static let defaultPrimaryColor = "#4F46E5"
    static let defaultSecondaryColor = "#7C3AED"
    static let wcagContrastMinimum = 4.5
    static let maxLogoUploadBytes = 4 * 1024 * 1024

    static let aiFeatureKeys: [(key: String, labelKey: String)] = [
        ("ai_tutor", "mobile.admin.orgBranding.aiGovernance.feature.aiTutor"),
        ("rag_notebook", "mobile.admin.orgBranding.aiGovernance.feature.notebook"),
        ("syllabus_generation", "mobile.admin.orgBranding.aiGovernance.feature.syllabus"),
        ("translation", "mobile.admin.orgBranding.aiGovernance.feature.translation"),
        ("quiz_generation", "mobile.admin.orgBranding.aiGovernance.feature.quiz"),
        ("lesson_generation", "mobile.admin.orgBranding.aiGovernance.feature.lesson"),
    ]

    static let providerLabels: [String: String] = [
        "openrouter": "OpenRouter",
        "anthropic": "Anthropic",
        "openai": "OpenAI",
        "azure_openai": "Azure OpenAI",
        "bedrock": "AWS Bedrock",
        "vertex": "Google Vertex AI",
    ]

    enum SaveStatus: Equatable {
        case idle
        case saving
        case saved
        case error(String)
    }

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings
    }

    static func canManageOrgBranding(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission) || permissions.contains(orgUnitsAdminPermission)
    }

    static func shouldShowEntry(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        adminSettingsEnabled(features) && canManageOrgBranding(permissions: permissions)
    }

    static func canView(
        features: MobilePlatformFeatures,
        permissions: [String]
    ) -> Bool {
        shouldShowEntry(features: features, permissions: permissions)
    }

    static func webOrgBrandingPath() -> String { "/settings/org-branding" }

    static func resolveOrgId(accessToken: String?, courses: [CourseSummary]) -> String? {
        CourseCreateLogic.resolveOrgId(accessToken: accessToken, courses: courses)
    }

    static func resolveBrandAssetUrl(_ pathOrUrl: String?) -> URL? {
        guard let raw = pathOrUrl?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
            return nil
        }
        if raw.hasPrefix("http://") || raw.hasPrefix("https://") {
            return URL(string: raw)
        }
        let path = raw.hasPrefix("/") ? raw : "/\(raw)"
        return AppConfiguration.apiURL(path: path)
    }

    static func normalizedHexColor(_ value: String) -> String? {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }
        let pattern = #"^#([0-9a-fA-F]{6})$"#
        guard let regex = try? NSRegularExpression(pattern: pattern),
              regex.firstMatch(in: trimmed, range: NSRange(trimmed.startIndex..., in: trimmed)) != nil else {
            return nil
        }
        return "#" + trimmed.dropFirst().uppercased()
    }

    static func isValidHexColor(_ value: String) -> Bool {
        normalizedHexColor(value) != nil
    }

    static func contrastRatioAgainstWhite(hex: String) -> Double? {
        guard let normalized = normalizedHexColor(hex) else { return nil }
        let hexValue = String(normalized.dropFirst())
        guard let rgb = UInt32(hexValue, radix: 16) else { return nil }
        let r = Double((rgb >> 16) & 0xFF) / 255.0
        let g = Double((rgb >> 8) & 0xFF) / 255.0
        let b = Double(rgb & 0xFF) / 255.0
        func channel(_ value: Double) -> Double {
            value <= 0.03928 ? value / 12.92 : pow((value + 0.055) / 1.055, 2.4)
        }
        let luminance = 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
        return (1.0 + 0.05) / (luminance + 0.05)
    }

    static func showsContrastWarning(primaryColor: String, serverWarning: Bool, serverRatio: Double?) -> Bool {
        if serverWarning { return true }
        if let serverRatio, serverRatio < wcagContrastMinimum { return true }
        if let ratio = contrastRatioAgainstWhite(hex: primaryColor), ratio < wcagContrastMinimum {
            return true
        }
        return false
    }

    static func parseAllowedModels(_ text: String) -> [String] {
        text
            .split { $0 == "\n" || $0 == "," }
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
    }

    static func allowedModelsText(_ models: [String]?) -> String {
        (models ?? []).joined(separator: "\n")
    }

    static func buildAiConfigSaveRequest(
        enabled: [String: Bool],
        allowedModelsText: String
    ) -> PutAiConfigRequest {
        let models = parseAllowedModels(allowedModelsText)
        var featuresEnabled: [String: Bool] = [:]
        for feature in aiFeatureKeys {
            featuresEnabled[feature.key] = enabled[feature.key] != false
        }
        return PutAiConfigRequest(
            featuresEnabled: featuresEnabled,
            allowedModels: models.isEmpty ? nil : models
        )
    }

    static func buildAiProviderSaveRequest(
        provider: String,
        modelAlias: String,
        fallbackProvider: String,
        byokKey: String
    ) -> PutAiProviderSettingsRequest {
        let trimmedFallback = fallbackProvider.trimmingCharacters(in: .whitespacesAndNewlines)
        var request = PutAiProviderSettingsRequest(
            provider: provider,
            modelAlias: modelAlias,
            fallbackProvider: trimmedFallback.isEmpty ? nil : trimmedFallback,
            byokApiKey: nil
        )
        let trimmedKey = byokKey.trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmedKey.isEmpty, trimmedKey != platformSecretPlaceholder {
            request.byokApiKey = trimmedKey
        }
        return request
    }

    static func byokFieldValue(configured: Bool, draft: String) -> String {
        if !draft.isEmpty { return draft }
        return configured ? platformSecretPlaceholder : ""
    }

    static func shouldSendByokKey(_ value: String) -> Bool {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return !trimmed.isEmpty && trimmed != platformSecretPlaceholder
    }

    static func brandingPutRequest(from branding: OrgBrandingResponse) -> PutOrgBrandingRequest {
        PutOrgBrandingRequest(
            logoUrl: branding.logoUrl,
            faviconUrl: branding.faviconUrl,
            primaryColor: branding.primaryColor,
            secondaryColor: branding.secondaryColor,
            customDomain: branding.customDomain,
            customEmailDisplayName: branding.customEmailDisplayName
        )
    }

    static func providerLabel(_ provider: String) -> String {
        providerLabels[provider] ?? provider
    }

    static func color(from hex: String) -> Color {
        guard let normalized = normalizedHexColor(hex),
              let rgb = UInt32(normalized.dropFirst(), radix: 16) else {
            return Color.indigo
        }
        return Color(
            red: Double((rgb >> 16) & 0xFF) / 255.0,
            green: Double((rgb >> 8) & 0xFF) / 255.0,
            blue: Double(rgb & 0xFF) / 255.0
        )
    }

    static func userFacingError(_ error: Error, fallbackKey: String) -> String {
        if let apiError = error as? APIError, case let .httpStatus(_, message) = apiError,
           let message, !message.isEmpty {
            return message
        }
        return L.dynamicText(fallbackKey)
    }
}
