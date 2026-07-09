import Foundation

/// Course plagiarism / AI-authorship settings helpers (M13.7).
enum CoursePlagiarismLogic {
    struct ProviderOption: Identifiable, Hashable {
        var id: String { value }
        var value: String
        var labelKey: String
    }

    struct FormDraft: Equatable, Hashable {
        var checksEnabled: Bool
        var provider: String
        var thresholdPct: String
    }

    enum ValidationError: Equatable {
        case thresholdInvalid
    }

    static let defaultThresholdPct = 40.0

    static let providerOptions: [ProviderOption] = [
        ProviderOption(value: "", labelKey: "mobile.courseSettings.plagiarism.provider.default"),
        ProviderOption(value: "none", labelKey: "mobile.courseSettings.plagiarism.provider.none"),
        ProviderOption(value: "turnitin", labelKey: "mobile.courseSettings.plagiarism.provider.turnitin"),
        ProviderOption(value: "copyleaks", labelKey: "mobile.courseSettings.plagiarism.provider.copyleaks"),
        ProviderOption(value: "gptzero", labelKey: "mobile.courseSettings.plagiarism.provider.gptzero"),
    ]

    static func cacheKey(courseCode: String) -> String {
        "course:\(courseCode):plagiarism-settings"
    }

    static func saveIdempotencyKey(courseCode: String) -> String {
        "course-plagiarism:\(courseCode):save"
    }

    static func patchPath(courseCode: String) -> String {
        "/api/v1/courses/\(courseCode)/plagiarism-settings"
    }

    static func draft(from settings: CoursePlagiarismSettings?) -> FormDraft {
        FormDraft(
            checksEnabled: settings?.plagiarismChecksEnabled ?? true,
            provider: normalizedProvider(settings?.plagiarismProvider),
            thresholdPct: formatThreshold(settings?.plagiarismAlertThresholdPct)
        )
    }

    static func isDirty(current: FormDraft, baseline: FormDraft) -> Bool {
        current != baseline
    }

    static func validateDraft(_ draft: FormDraft) -> ValidationError? {
        parsedThreshold(draft.thresholdPct) == nil ? .thresholdInvalid : nil
    }

    static func buildPatchBody(current: FormDraft) -> PatchCoursePlagiarismBody {
        PatchCoursePlagiarismBody(
            plagiarismChecksEnabled: current.checksEnabled,
            plagiarismProvider: current.provider.isEmpty ? nil : current.provider,
            plagiarismAlertThresholdPct: parsedThreshold(current.thresholdPct) ?? defaultThresholdPct
        )
    }

    static func normalizedProvider(_ provider: String?) -> String {
        let trimmed = (provider ?? "").trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard providerOptions.contains(where: { $0.value == trimmed && !$0.value.isEmpty }) else {
            return ""
        }
        return trimmed
    }

    static func formatThreshold(_ value: Double?) -> String {
        let resolved = value.flatMap { $0.isFinite ? $0 : nil } ?? defaultThresholdPct
        if resolved.rounded() == resolved {
            return String(Int(resolved))
        }
        return String(resolved)
    }

    static func parsedThreshold(_ text: String) -> Double? {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let value = Double(trimmed), value.isFinite, value >= 0, value <= 100 else {
            return nil
        }
        return value
    }
}
