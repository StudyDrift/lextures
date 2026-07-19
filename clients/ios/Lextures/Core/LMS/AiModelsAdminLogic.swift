import Foundation

/// AI models, system prompts, and usage reports admin helpers (M14.7).
enum AiModelsAdminLogic {
    static let rbacManagePermission = "global:app:rbac:manage"
    static let platformSecretPlaceholder = "••••••••••••"

    enum SaveStatus: Equatable {
        case idle
        case saving
        case saved
        case error(String)
    }

    enum ReportPreset: String, CaseIterable, Identifiable {
        case hours24 = "24h"
        case days7 = "7d"
        case days30 = "30d"
        case days90 = "90d"

        var id: String { rawValue }

        var labelKey: String.LocalizationValue {
            switch self {
            case .hours24: "mobile.admin.ai.reports.preset.24h"
            case .days7: "mobile.admin.ai.reports.preset.7d"
            case .days30: "mobile.admin.ai.reports.preset.30d"
            case .days90: "mobile.admin.ai.reports.preset.90d"
            }
        }

        var hours: Int {
            switch self {
            case .hours24: 24
            case .days7: 7 * 24
            case .days30: 30 * 24
            case .days90: 90 * 24
            }
        }
    }

    static let fallbackTextModels: [AiModelOption] = [
        AiModelOption(id: "google/gemini-2.0-flash-001", name: "Gemini 2.0 Flash"),
        AiModelOption(id: "google/gemini-2.5-flash", name: "Gemini 2.5 Flash"),
        AiModelOption(id: "openai/gpt-4o-mini", name: "GPT-4o mini"),
        AiModelOption(id: "anthropic/claude-3.5-sonnet", name: "Claude 3.5 Sonnet"),
        AiModelOption(id: "meta-llama/llama-3.3-70b-instruct", name: "Llama 3.3 70B Instruct"),
    ]

    static let fallbackImageModels: [AiModelOption] = [
        AiModelOption(id: "google/gemini-2.5-flash-image", name: "Gemini 2.5 Flash (image)"),
        AiModelOption(id: "google/gemini-3.1-flash-image-preview", name: "Gemini 3.1 Flash Image (preview)"),
        AiModelOption(id: "black-forest-labs/flux.2-pro", name: "FLUX.2 Pro"),
        AiModelOption(id: "black-forest-labs/flux.2-flex", name: "FLUX.2 Flex"),
        AiModelOption(id: "sourceful/riverflow-v2-fast", name: "Riverflow v2 Fast"),
        AiModelOption(id: "sourceful/riverflow-v2-pro", name: "Riverflow v2 Pro"),
    ]

    private static let featureLabels: [String: String] = [
        "ai_tutor": "AI Tutor",
        "rag_notebook": "Notebook AI",
        "syllabus_generation": "Syllabus generation",
        "translation": "Translation",
        "quiz_generation": "Quiz generation",
        "reading_level_simplification": "Reading level",
        "content_translation": "Content translation",
        "alt_text_suggestion": "Alt text",
        "vibe_generation": "Vibe activities",
        "unknown": "Unknown",
    ]

    static func adminSettingsEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffMobileAdminSettings || features.ffMobileAdminConsole
    }

    static func canManage(permissions: [String]) -> Bool {
        permissions.contains(rbacManagePermission)
    }

    static func shouldShowEntry(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        !features.ffMobileAdminConsole && features.ffMobileAdminSettings && canManage(permissions: permissions)
    }

    static func canView(features: MobilePlatformFeatures, permissions: [String]) -> Bool {
        adminSettingsEnabled(features) && canManage(permissions: permissions)
    }

    /// Builds the PUT body. The OpenRouter key is write-only: only send a new key when the
    /// draft differs from the masked baseline, and only clear when the field was emptied from a
    /// configured placeholder.
    static func buildAiSettingsSaveRequest(
        imageModelId: String,
        courseSetupModelId: String,
        notebookFlashcardsModelId: String,
        vibeActivityModelId: String,
        graderAgentModelId: String,
        openRouterApiKey: String,
        openRouterApiKeyBaseline: String
    ) -> PutAiSettingsRequest {
        var request = PutAiSettingsRequest(
            imageModelId: imageModelId.trimmingCharacters(in: .whitespacesAndNewlines),
            courseSetupModelId: courseSetupModelId.trimmingCharacters(in: .whitespacesAndNewlines),
            notebookFlashcardsModelId: notebookFlashcardsModelId.trimmingCharacters(in: .whitespacesAndNewlines),
            vibeActivityModelId: vibeActivityModelId.trimmingCharacters(in: .whitespacesAndNewlines),
            graderAgentModelId: graderAgentModelId.trimmingCharacters(in: .whitespacesAndNewlines),
            openRouterApiKey: nil,
            clearOpenRouterApiKey: nil
        )

        let keyTrimmed = openRouterApiKey.trimmingCharacters(in: .whitespacesAndNewlines)
        let baselineTrimmed = openRouterApiKeyBaseline.trimmingCharacters(in: .whitespacesAndNewlines)
        guard keyTrimmed != baselineTrimmed else { return request }

        if !keyTrimmed.isEmpty, keyTrimmed != platformSecretPlaceholder {
            request.openRouterApiKey = keyTrimmed
        }
        if baselineTrimmed == platformSecretPlaceholder,
           keyTrimmed.isEmpty,
           openRouterApiKey != openRouterApiKeyBaseline {
            request.clearOpenRouterApiKey = true
        }
        return request
    }

    static func isSaveDisabled(
        saving: Bool,
        imageModelId: String,
        courseSetupModelId: String,
        notebookFlashcardsModelId: String,
        vibeActivityModelId: String
    ) -> Bool {
        saving
            || imageModelId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            || courseSetupModelId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            || notebookFlashcardsModelId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            || vibeActivityModelId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    /// Ensures the currently selected model id remains pickable even when absent from the list.
    static func modelsWithSelection(_ models: [AiModelOption], selectedId: String) -> [AiModelOption] {
        let trimmed = selectedId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return models }
        if models.contains(where: { $0.id == trimmed }) { return models }
        return [AiModelOption(id: trimmed, name: trimmed)] + models
    }

    static func modelDisplayLabel(_ model: AiModelOption) -> String {
        let name = (model.name?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? model.id
        var parts: [String] = [name]
        if name != model.id {
            parts.append(model.id)
        }
        if let modalities = model.modalitiesSummary?.trimmingCharacters(in: .whitespacesAndNewlines), !modalities.isEmpty {
            parts.append(modalities)
        }
        if let ctx = model.contextLength {
            parts.append("ctx \(formatCount(Int64(ctx)))")
        }
        let inPrice = model.inputPricePerMillionUsd
        let outPrice = model.outputPricePerMillionUsd
        if inPrice != nil || outPrice != nil {
            let inStr = formatUsd(inPrice ?? 0)
            let outStr = formatUsd(outPrice ?? 0)
            parts.append("\(inStr)/\(outStr) per 1M")
        }
        return parts.joined(separator: " · ")
    }

    static func shouldSendOpenRouterKey(_ value: String) -> Bool {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return !trimmed.isEmpty && trimmed != platformSecretPlaceholder
    }

    static func shouldClearOpenRouterKey(draft: String, baseline: String) -> Bool {
        let keyTrimmed = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        let baselineTrimmed = baseline.trimmingCharacters(in: .whitespacesAndNewlines)
        return baselineTrimmed == platformSecretPlaceholder
            && keyTrimmed.isEmpty
            && draft != baseline
    }

    static func utcRange(for preset: ReportPreset, now: Date = Date()) -> (from: String, to: String) {
        let to = now
        let from = to.addingTimeInterval(TimeInterval(-preset.hours * 3600))
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        // Fall back without fractional seconds if needed.
        let toStr = formatter.string(from: to)
        let fromStr = formatter.string(from: from)
        if !toStr.isEmpty, !fromStr.isEmpty {
            return (fromStr, toStr)
        }
        formatter.formatOptions = [.withInternetDateTime]
        return (formatter.string(from: from), formatter.string(from: to))
    }

    static func featureLabel(_ feature: String) -> String {
        if let known = featureLabels[feature] { return known }
        return feature.replacingOccurrences(of: "_", with: " ")
    }

    static func formatUsd(_ value: Double) -> String {
        if !value.isFinite || value == 0 { return "$0.00" }
        if value < 0.01 {
            return String(format: "$%.4f", value)
        }
        return String(format: "$%.2f", value)
    }

    static func formatCount(_ value: Int64) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        return formatter.string(from: NSNumber(value: value)) ?? "\(value)"
    }

    static func promptContentChanged(original: String, draft: String) -> Bool {
        original != draft
    }

    static func userFacingError(_ error: Error, fallbackKey: String.LocalizationValue) -> String {
        if let api = error as? APIError, case let .httpStatus(_, message) = api, let message, !message.isEmpty {
            return message
        }
        return L.text(fallbackKey)
    }
}
